package pakpos

import (
	"fmt"
	"github.com/eaciit/knot/knot.v1"
	"github.com/eaciit/toolkit"
	"strings"
)

func (b *Broadcaster) Broadcast(k string, v interface{}) (*MessageMonitor, error) {
	var mm *MessageMonitor
	lk := strings.ToLower(k)
	if lk == "stop" {
		mm = NewMessageMonitor(b, "stop", k, v, DefaultExpiry())
	} else {
		mm = NewMessageMonitor(b, "", k, v, DefaultExpiry())
	}
	mm.DistributionType = DistributeAsBroadcast
	b.Server.Log().Info(fmt.Sprintf("Broadcasting %s to %d server(s)", k, len(mm.Targets)))
	return mm, nil
}

func (b *Broadcaster) Que(k string, v interface{}) (*MessageMonitor, error) {
	mm := NewMessageMonitor(b, "", k, v, DefaultExpiry())
	mm.DistributionType = DistributeAsQue
	b.Server.Log().Info(fmt.Sprintf("Create new que  %s to %d server(s)", k, len(mm.Targets)))
	return mm, nil
}

func (b *Broadcaster) initRoute() {
	//-- add node
	b.Route("/nodeadd", func(kr *knot.WebContext) interface{} {
		url := kr.Query("node")
		b.Subscibers = append(b.Subscibers, url)
		kr.Server.Log().Info(fmt.Sprintf("Add node %s to %s", url, b.Address))
		return "OK"
	})

	//-- add channel subscribtion
	b.Route("/channelregister", func(k *knot.WebContext) interface{} {
		k.Config.OutputType = knot.OutputJson
		result := toolkit.NewResult()
		cr := &ChannelRegister{}
		k.GetPayload(cr)
		b.addChannelSubcriber(cr.Channel, cr.Subscriber)
		return result
	})

	//-- get the message (used for DstributeAsQue type)
	b.Route("/getmsg", func(k *knot.WebContext) interface{} {
		k.Config.OutputType = knot.OutputJson
		result := toolkit.NewResult()
		tm := toolkit.M{}
		e := k.GetPayload(&tm)
		if e != nil {
			fmt.Println(e.Error())
			result.SetErrorTxt(fmt.Sprintf("Broadcaster GetMsg Payload Error: %s", e.Error()))
		} else {
			key := tm.Get("Key", "").(string)
			if key == "" {
				result.SetErrorTxt("Broadcaste GetMsg Error: No key is provided")
			} else {
				m, exist := b.messages[key]
				//fmt.Println(m.Key + " : " + m.Data.(string))
				if exist == false {
					result.SetErrorTxt("Message " + key + " is not exist")
				} else {
					result.Data = m.Data
					//fmt.Printf("Sent data: %v \n", m.Data)
				}
			}
		}
		return result
	})

	//-- invoked to identify if a msg has been received (used for DistAsQue)
	b.Route("/msgreceived", func(k *knot.WebContext) interface{} {
		k.Config.OutputType = knot.OutputJson
		result := toolkit.NewResult()

		tm := toolkit.M{}
		e := k.GetPayload(&tm)
		if e != nil {
			result.SetErrorTxt("Broadcaster MsgReceived Payload Error: " + e.Error())
		} else {
			key := tm.Get("key", "").(string)
			subscriber := tm.Get("subscriber", "").(string)

			mm, exist := b.messages[key]
			if exist == false {
				result.SetErrorTxt("Broadcaster MsgReceived Error: " + key + " is not exist")
			} else {
				for tIndex, t := range mm.Targets {
					if t == subscriber {
						mm.Status[tIndex] = "OK"
						break
					}
				}
			}
		}

		return result
	})

	//-- gracefully stop the server
	b.Route("/stop", func(k *knot.WebContext) interface{} {
		result := toolkit.NewResult()
		defer func() {
			if result.Status == toolkit.Status_OK {
				k.Server.Stop()
			}
		}()
		for _, s := range b.Subscibers {
			url := "http://" + s + "/stop"
			bs, e := webcall(url, "GET", nil)
			if e != nil {
				result.SetErrorTxt("Unable to stop " + s + ": " + e.Error())
			}
			sresult := toolkit.NewResult()
			toolkit.Unjson(bs, sresult)
			if sresult.Status == toolkit.Status_NOK {
				result.SetErrorTxt("Unable to stop " + s + ": " + sresult.Message)
			}
		}

		k.Config.OutputType = knot.OutputJson
		return result
	})
}

func (b *Broadcaster) Stop() {
	b.Broadcast("stop", "")
}

func webcall(url, method string, data []byte) ([]byte, error) {
	r, e := toolkit.HttpCall(url, method, data, nil)
	if e != nil {
		return nil, e
	}
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("Invalid Code: " + r.Status)
	}
	return toolkit.HttpContent(r), nil
}
