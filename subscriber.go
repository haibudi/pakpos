package pakpos

import (
	"fmt"
	"github.com/eaciit/knot/knot.v1"
	"github.com/eaciit/toolkit"
	//"io/ioutil"
)

type MessageQue struct {
	Key       string
	Data      interface{}
	Collected bool
}

type Subscriber struct {
	PosServer
	Protocol           string
	BroadcasterAddress string

	messageKeys []string
	MessageQues map[string]*MessageQue
}

func (s *Subscriber) Validate() error {
	e := s.PosServer.Validate()
	if e != nil {
		return e
	}
	if s.BroadcasterAddress == "" {
		return fmt.Errorf("BroadcasterAddress is not configured properly")
	}
	return nil
}

func (s *Subscriber) Start(address string) error {
	s.Address = address
	s.MessageQues = map[string]*MessageQue{}
	broadcasterUrl := s.BroadcasterAddress + "/nodeadd?node=" + s.Address
	r, e := toolkit.HttpCall(broadcasterUrl, "GET", nil, nil)
	if e != nil {
		return fmt.Errorf("Unable to contact Broadcaster Server. %s", e.Error())
	}
	if sOk := toolkit.HttpContentString(r); sOk != "OK" {
		return fmt.Errorf("Unable to add %s as subsriber to %s. Error: %s", s.Address, s.BroadcasterAddress, sOk)
	}

	//-- inform subscriber if newkey is available to be collected
	s.Route("/newkey", func(k *knot.WebContext) interface{} {
		result := toolkit.NewResult()
		k.Config.OutputType = knot.OutputJson
		msg := new(MessageQue)
		e := k.GetPayload(&msg)
		if e != nil {
			result.Message = e.Error()
			result.Status = toolkit.Status_NOK
		} else {
			msg.Data = nil
			msg.Collected = false
			_, exist := s.MessageQues[msg.Key]
			if !exist {
				s.messageKeys = append(s.messageKeys, msg.Key)
			}
			s.messageKeys = append(s.messageKeys, msg.Key)
		}
		return result
	})

	s.Route("/getmsg", func(k *knot.WebContext) interface{} {
		k.Config.OutputType = knot.OutputJson
		key := k.Query("key")
		result := s.getMsgAsResult(key)
		return result
	})

	//-- replicate message to subscribe
	s.Route("/msg", func(k *knot.WebContext) interface{} {
		k.Config.OutputType = knot.OutputJson
		result := toolkit.NewResult()
		var payload Message
		eDecode := k.GetPayload(&payload)
		if eDecode != nil {
			result.Status = toolkit.Status_NOK
			result.Message = fmt.Sprintf("Subscriber Message Decode Error %s", eDecode.Error())
		} else {
			result.Data = payload
		}
		return result
	})

	//-- stop the server
	s.Route("/stop", func(k *knot.WebContext) interface{} {
		defer k.Server.Stop()
		k.Config.OutputType = knot.OutputJson
		result := new(toolkit.Result)
		result.Status = "OK"
		return result
	})

	go func() {
		s.Server.Listen()
	}()
	return nil
}

func (s *Subscriber) GetMsg(key string) (interface{}, error) {
	result := s.getMsgAsResult(key)
	return result.Data, result.Error()
}

func (s *Subscriber) getMsgAsResult(key string) *toolkit.Result {
	result := toolkit.NewResult()
	if key == "" {
		if len(s.messageKeys) > 0 {
			key = s.messageKeys[0]
		}
	}

	// if no key is provided, check from the latest
	if key == "" {
		result.Status = toolkit.Status_NOK
		result.Message = "No key has been provided to receive the message"
	} else {
		url := fmt.Sprintf("%s/getmsg", s.BroadcasterAddress)
		msgQue, exist := s.MessageQues[key]
		if !exist {
			result.Status = toolkit.Status_NOK
			result.Message = "Key " + key + " is not exist on message que or it has been collected"
		} else {
			r, e := toolkit.HttpCall(url, "POST",
				toolkit.Jsonify(msgQue), nil)
			if e != nil {
				result.SetErrorTxt("Subscriber ReceiveMsg Call Error: " + e.Error())
			} else if r.StatusCode != 200 {
				result.SetErrorTxt("Subsciber ReceiveMsg Call Error: " + r.Status)
			} else {
				var msg Message
				e := toolkit.Unjson(toolkit.HttpContent(r), &msg)
				if e != nil {
					result.SetErrorTxt(fmt.Sprintf("Subsciber ReceiveMsg Decode Error: ", e.Error()))
				} else {
					result.Data = msg
				}
			}
		}
	}

	if result.Status == "OK" {
		url := fmt.Sprintf("%s/msgreceived")
		toolkit.HttpCall(url, "POST", toolkit.Jsonify(struct {
			Key        string
			Subscriber string
		}{key, s.Address}), nil)
	}
	return result
}

type ChannelRegister struct {
	Channel    string
	Subscriber string
}

func (s *Subscriber) AddChannel(c string) error {
	url := fmt.Sprintf("%s/channelregister", s.BroadcasterAddress)
	r, e := toolkit.HttpCall(url, "POST", toolkit.Jsonify(ChannelRegister{c, s.Address}), nil)
	if e != nil {
		return fmt.Errorf("Channel Register Call Error: %s", e.Error())
	}
	if r.StatusCode != 200 {
		return fmt.Errorf("Channel Register Call Error: %s", r.Status)
	}
	result := new(toolkit.Result)
	e = toolkit.Unjson(toolkit.HttpContent(r), &result)
	if e != nil {
		return fmt.Errorf("Channel Register Decode error: %s", e.Error())
	}
	if result.Status != toolkit.Status_OK {
		return fmt.Errorf("Channel Register error: %s", result.Message)
	}
	return nil
}

func (s *Subscriber) AddFn(c string, fn FnMessage) {

}
