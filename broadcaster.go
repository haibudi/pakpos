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
	Subscibers []string

	channelSubscribers map[string][]string
}

func (p *PosServer) Validate() error {
	return nil
}

func (b *Broadcaster) Start(address string) error {
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
		return b.Subscibers
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
	targets := b.getChannelSubscribers(k)
	if lk == "stop" {
		mm = NewMessageMonitor(targets, "stop", k, v, DefaultExpiry())
	} else {
		mm = NewMessageMonitor(targets, "", k, v, DefaultExpiry())
	}
	mm.DistributionType = DistributeAsBroadcast
	b.Server.Log().Info(fmt.Sprintf("Broadcasting %s to %d server(s)", k, len(mm.Targets)))
	return mm, nil
}

func (b *Broadcaster) Que(k string, v interface{}) (*MessageMonitor, error) {
	targets := b.getChannelSubscribers(k)
	mm := NewMessageMonitor(targets, "", k, v, DefaultExpiry())
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
	b.Route("/msgget", func(k *knot.WebContext) interface{} {
		k.Config.OutputType = knot.OutputJson
		result := toolkit.NewResult()
		return result
	})

	//-- invoked to identify if a msg has been received (used for DistAsQue)
	b.Route("/msgreceived", func(k *knot.WebContext) interface{} {
		k.Config.OutputType = knot.OutputJson
		result := toolkit.NewResult()
		return result
	})

	//-- gracefully stop the server
	b.Route("/stop", func(k *knot.WebContext) interface{} {
		defer k.Server.Stop()
		return "OK"
	})
}

func (b *Broadcaster) Stop() {
	b.Broadcast("stop", "")
}
