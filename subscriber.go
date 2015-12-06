package pakpos

import (
	"fmt"
	"github.com/eaciit/knot/knot.v1"
	"github.com/eaciit/toolkit"
	//"io/ioutil"
)

type Subscriber struct {
	PosServer
	Protocol           string
	BroadcasterAddress string
	Channels           []*Channel
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
		k.Config.OutputType = knot.OutputJson
		result := toolkit.NewResult()
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

func (s *Subscriber) Receive(k string) (interface{}, error) {
	return nil, nil
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
