package pakpos

import (
	"github.com/eaciit/knot"
	"github.com/eaciit/toolkit"
)

type ServerRole int
type SubcribtionType int

const (
	ServerBroadcaster ServerRole = 1
	ServerSubscriber  ServerRole = 10

	SubcribeAll     SubcribtionType = 0
	SubcribeChannel SubcribtionType = 1
)

type FnMessage func(toolkit.KvString) interface{}

type Server struct {
	knot.Server
	Role ServerRole
}

type Broadcaster struct {
	Server     *Server
	Subscibers []*Server
}

type Subscriber struct {
	Server             *Server
	BroadcasterAddress *Server
	SubcribtionType    SubcribtionType
	Channels           []*Channel
}

func (s *Server) Start(address string) {
}

func (s *Server) Stop() {
}

func (b *Broadcaster) Broadcast(k string, v interface{}) (*MessageMonitor, error) {
	mm := new(MessageMonitor)
	return mm, nil
}

func (s *Subscriber) Receive(k string) (interface{}, error) {
	return nil, nil
}

func (s *Subscriber) AddChannel(c string) {
}

func (s *Subscriber) AddFn(c string, fn FnMessage) {

}
