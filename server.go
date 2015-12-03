package pakpos

import (
	"github.com/eaciit/knot"
)

type ServerRole int

const (
	ServerBroadcaster ServerRole = 1
	ServerSubscriber  ServerRole = 10
)

type Server struct {
	knot.Server
	Role ServerRole
}

type Broadcaster struct {
	Server     *Server
	Subscibers []*Server
}

type Subscriber struct {
	Server   *Server
	Master   *Server
	Channels []*Channel
}
