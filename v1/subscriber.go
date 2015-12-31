package pakpos

import (
	"fmt"
	"github.com/eaciit/knot/knot.v1"
	"github.com/eaciit/toolkit"
	//"net/url"
	"time"
)

type Subscriber struct {
	knot.Server
	Protocol    string
	Broadcaster string
	Secret      string

	messages map[string]*Message
}

func (s *Subscriber) BaseUrl() string {
	if s.Protocol == "" {
		s.Protocol = "https"
	}
	return fmt.Sprintf("%s://%s/", s.Protocol, s.Address)
}

func (s *Subscriber) Start(address, broadcaster, broadcastServerSecret string) error {
	if string(broadcaster[len(broadcaster)-1]) != "/" {
		broadcaster += "/broadcaster/"
	}
	s.Broadcaster = broadcaster
	s.Address = address
	s.messages = map[string]*Message{}

	//--- get confirmation from Broadcaster first
	r, e := toolkit.CallResult(broadcaster+"addnode", "POST",
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
		r, e := toolkit.CallResult(s.Broadcaster+"getmessage", "POST",
			toolkit.M{}.Set("subscriber", s.Address).Set("secret", s.Secret).Set("key", key).ToBytes("", nil))
		if e != nil {
			return e
		}
		message = r.Data
		delete(s.messages, key)
		s.Server.Log().Info(fmt.Sprintf("Message %s has been received by %s: %v", key, s.Address, message))
		return nil
	} else {
		s.Server.Log().Info(fmt.Sprintf("Message %s has been received by %s: %v", key, s.Address, message))
		return nil
	}
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

func (s *Subscriber) NewKey(k *knot.WebContext) interface{} {
	result := toolkit.NewResult()
	var model struct {
		Secret string
		Key    string
		Expiry time.Time
	}
	e := k.GetPayload(&model)
	if e != nil {
		result.SetErrorTxt("Decode error: %s" + e.Error())
		return result
	}
	if model.Secret != s.Secret {
		result.SetErrorTxt("Broadcaster is not authorised")
		return result
	}
	msg := new(Message)
	msg.Key = model.Key
	msg.Expiry = model.Expiry
	s.messages[model.Key] = msg
	s.Server.Log().Info(fmt.Sprintf("Receive new key %s. Current keys available in que: %d", model.Key, len(s.messages)))
	return result
}

func (s *Subscriber) CollectMessage(k *knot.WebContext) interface{} {
	result := toolkit.NewResult()
	var model struct {
		Secret string
		Key    string
	}
	k.GetPayload(&model)
	if model.Secret != s.Secret {
		result.SetErrorTxt("Call is not authorised")
		return result
	}
	if model.Key == "" {
		for k, _ := range s.messages {
			model.Key = k
			break
		}
		if model.Key == "" {
			result.SetErrorTxt("No key has been provided to collect message")
			return result
		}
	}

	var data interface{}
	e := s.ReceiveMessage("Que", model.Key, &data)
	if e != nil {
		result.SetErrorTxt("Unable to get message " + model.Key + " from broadcaster. " + e.Error())
		return result
	}
	result.Data = data
	return result
}
