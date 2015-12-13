package pakpos

import (
	"fmt"
	"github.com/eaciit/knot/knot.v1"
	"github.com/eaciit/toolkit"
)

type Subscriber struct {
	knot.Server
	Protocol    string
	Broadcaster string
	Secret      string
}

func (s *Subscriber) BaseUrl() string {
	if s.Protocol == "" {
		s.Protocol = "http"
	}
	return fmt.Sprintf("%s://%s/", s.Protocol, s.Address)
}

func (s *Subscriber) Start(address, broadcaster, broadcastServerSecret string) error {
	if string(broadcaster[len(broadcaster)-1]) != "/" {
		broadcaster += "/broadcaster/"
	}
	s.Broadcaster = broadcaster
	s.Address = address

	//--- get confirmation from Broadcaster first
	r, e := CallResult(broadcaster+"addnode", "POST",
		toolkit.M{}.Set("subscriber", s.Address).Set("secret", broadcastServerSecret).ToBytes("json", nil))
	if e != nil {
		return e
	} else {
		//fmt.Printf("R: %v \n", r)
		s.Secret = r.Data.(string)
	}

	app := knot.NewApp("pakpos")
	app.DefaultOutputType = knot.OutputJson
	app.Register(s)
	go func() {
		knot.StartApp(app, address)
	}()
	return nil
}

func (s *Subscriber) ReceiveMessage(messageType string, key string, message interface{}) error {
	if messageType == "Que" {
	}
	s.Server.Log().Info(fmt.Sprintf("Message %s has been received by %s", key, s.Address))
	return nil
}

func (s *Subscriber) PushMessage(k *knot.WebContext) interface{} {
	result := toolkit.NewResult()
	var model struct {
		Secret string
		Key    string
		Data   interface{}
	}
	k.GetPayload(&model)
	if model.Secret != s.Secret {
		result.SetErrorTxt("Broadcaster not authorised")
		return result
	}
	e := s.ReceiveMessage("Broadcast", model.Key, model.Data)
	if e != nil {
		result.SetErrorTxt("Unable to accept message: " + e.Error())
		return result
	}
	return result
}
