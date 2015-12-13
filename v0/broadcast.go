package pakpos

import (
	"fmt"
	"github.com/eaciit/knot/knot.v1"
	"github.com/eaciit/toolkit"
	"strings"
)

type ServerRole int
type SubcribtionType int

const (
	ServerBroadcaster ServerRole = 1
	ServerSubscriber  ServerRole = 10
)

type SubscriberInfo struct {
	Address, Protocol string
	Token             string
}

func (si *SubscriberInfo) Url(s string) string {
	return fmt.Sprintf("%s://%s/%s", si.Protocol, si.Address, s)
}

type FnMessage func(toolkit.KvString) interface{}

type IPostServer interface {
	Validate() error
	Start(string) error
}

type PosServer struct {
	knot.Server
}

type Broadcaster struct {
	//knot.Server
	PosServer
	Subscibers []SubscriberInfo

	channelSubscribers map[string][]string
	messages           map[string]*MessageMonitor
}

func (p *PosServer) Validate() error {
	return nil
}

func (b *Broadcaster) Start(address string) error {
	b.messages = map[string]*MessageMonitor{}
	b.Address = address
	if e := b.Validate(); e != nil {
		return e
	}
	b.initRoute()
	go func() {
		b.Listen()
	}()
	return nil
}

func (b *Broadcaster) getChannelSubscribers(key string) []string {
	channel, key := ParseKey(key)
	channel = strings.ToLower(channel)
	if channel == "public" {
		return func() []string {
			s := []string{}
			for _, sub := range b.Subscibers {
				s = append(s, sub.Address)
			}
			return s
		}()
	} else {
		if b.channelSubscribers == nil {
			b.channelSubscribers = map[string][]string{}
		}
		if subs, exist := b.channelSubscribers[channel]; exist {
			return subs
		} else {
			return []string{}
		}
	}
}

func (b *Broadcaster) addChannelSubcriber(channel, subscriber string) {
	channel = strings.ToLower(channel)
	if b.channelSubscribers == nil {
		b.channelSubscribers = map[string][]string{}
	}
	if subs, exist := b.channelSubscribers[channel]; exist {
		b.channelSubscribers[channel] = append(subs, subscriber)
	} else {
		b.channelSubscribers[channel] = []string{subscriber}
	}
}

func (b *Broadcaster) removeChannelSubscriber(channel, subscriber string) {
	channel = strings.ToLower(channel)
	subscriber = strings.ToLower(subscriber)
	if b.channelSubscribers == nil {
		b.channelSubscribers = map[string][]string{}
	}
	if subs, exist := b.channelSubscribers[channel]; exist {
		i := 0
		found := false
		idx := 0
		for i < len(subs) && !found {
			if subs[i] == subscriber {
				idx = i
				found = true
			} else {
				i++
			}
		}
		if found {
			b.channelSubscribers[channel] = append(subs[:idx], subs[idx+1:]...)
		}
	}
}

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
