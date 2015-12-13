package pakpos

import (
	"fmt"
	"github.com/eaciit/toolkit"
	"strings"
	"sync"
	"time"
)

type DistributionType int

const (
	DistributeAsBroadcast DistributionType = 0
	DistributeAsQue       DistributionType = 1
)

var (
	defaultExpiry        time.Duration
	defaultRetryWhenFail int
)

func SetRetryWhenFail(i int) {
	defaultRetryWhenFail = i
}

func RetryWhenFail() int {
	if defaultRetryWhenFail == 0 {
		defaultRetryWhenFail = 3
	}
	return defaultRetryWhenFail
}

func SetDefaultExpiry(d time.Duration) {
	defaultExpiry = d
}

func DefaultExpiry() time.Duration {
	if int(defaultExpiry) == 0 {
		defaultExpiry = 10 * time.Second
	}
	return defaultExpiry
}

type Message struct {
	Key string
	//Command string
	Data   interface{}
	Expiry time.Time
}

type MessageMonitor struct {
	sync.Mutex
	Message
	Success          int
	Fail             int
	Targets          []string
	Status           []string
	DistributionType DistributionType

	retry       []int
	broadcaster *Broadcaster
}

func ParseKey(key string) (string, string) {
	keys := strings.Split(key, ":")
	if len(keys) == 1 {
		return "Public", key
	}
	return keys[0], strings.Join(keys[1:], ":")
}

func NewMessageMonitor(broadcaster *Broadcaster, key string, data interface{}, expiryAfter time.Duration) *MessageMonitor {
	targets := broadcaster.getChannelSubscribers(key)

	m := new(MessageMonitor)
	m.broadcaster = broadcaster
	m.Targets = targets
	m.Key = key
	//im.Command = command
	m.Data = data
	m.Expiry = time.Now().Add(expiryAfter)
	for _, _ = range targets {
		m.Status = append(m.Status, "")
		m.retry = append(m.retry, 0)
	}
	m.broadcaster.Server.Log().Info(fmt.Sprintf("Message %s written expired on %v", m.Key, m.Expiry))
	return m
}

func (m *MessageMonitor) Wait() {
	//--- add m to broadcaster messages
	m.broadcaster.messages[m.Key] = m
	if m.DistributionType == DistributeAsBroadcast {
		m.distributeBroadcast()
	} else if m.DistributionType == DistributeAsQue {
		m.distributeQue()
	}

	if m.Success != len(m.Targets) {
		m.broadcaster.Log().Warning(fmt.Sprintf("Message %s is not fully received "+
			"(%d out of %d) "+
			"and will be disposed because already exceed its expiry", m.Key, m.Success, len(m.Targets)))
	}
	delete(m.broadcaster.messages, m.Key)
}

func (m *MessageMonitor) setSuccessFail(k int, status string) {
	m.Lock()
	if status == "OK" {
		if m.Status[k] == "" {
			m.Success++
		} else if m.Status[k] != "OK" {
			m.Success++
			m.Fail--
		}
	} else {
		m.retry[k]++
		if m.Status[k] == "" {
			m.Fail++
		} else if m.Status[k] == "OK" {
			m.Fail++
			m.Success--
		}
	}
	m.Status[k] = status
	m.Unlock()

	if m.Success+m.Fail != len(m.Targets) {
		//remaining := len(m.Targets) - m.Success - m.Fail
		for k, _ := range m.Targets {
			status := m.Status[k]
			if status == "" {
				m.Fail++
				m.Status[k] = "Unknown reason"
			}
		}
	}
}

func (m *MessageMonitor) distributeBroadcast() {
	for len(m.Targets) != m.Success && time.Now().After(m.Expiry) == false {
		wg := new(sync.WaitGroup)
		for k, t := range m.Targets {
			wg.Add(1)
			go func(wg *sync.WaitGroup, k int, t string) {
				defer wg.Done()
				if m.Status[k] != "OK" && m.retry[k] < RetryWhenFail() {
					var command, url string
					/*
						if m.Command != "" {
							command = m.Command
						} else {
							command = "msg"
						}
					*/
					sub := m.broadcaster.Subscibers[t]
					if sub != nil {
						command = "pushmessage"
						url = fmt.Sprintf("http://%s/subscriber/%s", t, command)
						r, ecall := toolkit.HttpCall(url, "POST",
							toolkit.M{}.Set("secret", sub.Secret).Set("Data", m.Data).Set("Key", m.Key).ToBytes("json", nil),
							nil)
						if ecall != nil {
							m.setSuccessFail(k, "CALL ERROR: "+url+" ERR:"+ecall.Error())
						} else if r.StatusCode != 200 {
							m.setSuccessFail(k, fmt.Sprintf("CALL STATUS ERROR: %s ERR: %s", url, r.Status))
						} else {
							var result toolkit.Result
							bs := toolkit.HttpContent(r)
							edecode := toolkit.Unjson(bs, &result)
							if edecode != nil {
								m.setSuccessFail(k, "DECODE ERROR: "+string(bs)+" ERR:"+edecode.Error())
							} else {
								m.setSuccessFail(k, toolkit.IfEq(result.Status, toolkit.Status_OK, "OK", result.Message).(string))
							}
						}

					}
				}
			}(wg, k, t)
		}
		wg.Wait()
		//time.Sleep(1 * time.Second)
		//fmt.Printf("%d = %d \n", len(m.Targets), m.Success+m.Fail)
	}
}

func (m *MessageMonitor) distributeQue() {
	//--- inform all targets that new message has been created
	msg := toolkit.Jsonify(Message{Key: m.Key, Data: m.Data, Expiry: m.Expiry})
	wg := new(sync.WaitGroup)

	targetCount := len(m.Targets)
	var newtargets []string
	var failtargets []string
	for _, t := range m.Targets {
		wg.Add(1)
		go func(wg *sync.WaitGroup, t string) {
			defer wg.Done()
			url := fmt.Sprintf("%s://%s/newkey", "http", t)
			r, e := toolkit.HttpCall(url, "POST", msg, nil)
			if e != nil {
				m.broadcaster.Log().Warning(fmt.Sprintf(
					"Unable to inform %s for new que %s. %s",
					url, m.Key, e.Error()))
				failtargets = append(failtargets, t)
			} else if r.StatusCode != 200 {
				m.broadcaster.Log().Warning(fmt.Sprintf(
					"Unable to inform %s for new que %s  %s",
					url, m.Key, r.Status))
				failtargets = append(failtargets, t)
			} else {
				newtargets = append(newtargets, t)
			}
		}(wg, t)
	}
	wg.Wait()
	m.Targets = newtargets
	m.broadcaster.Log().Info(fmt.Sprintf("Ping %d servers for new message %s. Succcess: %d Fail: %d", targetCount, m.Key, len(newtargets), len(failtargets)))

	//-- loop while not all target complete receival or expire
	for len(m.Targets) != m.Success && time.Now().After(m.Expiry) == false {
		time.Sleep(1 * time.Millisecond)
	}
}

func (m *MessageMonitor) State() string {
	targetCount := len(m.Targets)
	if m.Success+m.Fail < targetCount {
		return "Waiting"
	} else if m.Success == targetCount {
		return "OK"
	} else {
		return "Not OK"
	}
}
