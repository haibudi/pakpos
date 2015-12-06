package pakpos

import (
	"fmt"
	"github.com/eaciit/knot/knot.v1"
	"github.com/eaciit/toolkit"
	"io/ioutil"
)

type Subscriber struct {
	PosServer
	Protocol           string
	BroadcasterAddress string
	SubcribtionType    SubcribtionType
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
		//var payload toolkit.M
		//ePayLoad := k.GetPayload(&payload)

		//--- get payload
		defer k.Request.Body.Close()
		bs, ePayload := ioutil.ReadAll(k.Request.Body)
		if ePayload != nil {
			result.Status = toolkit.Status_NOK
			result.Message = "SUBS PAYLOAD ERROR " + ePayload.Error()
		}

		var payload Message
		eDecode := toolkit.Unjson(bs, &payload)
		if eDecode != nil {
			result.Status = toolkit.Status_NOK
			result.Message = fmt.Sprintf("SUBS DECODE ERROR %v ERR %s", bs, eDecode.Error())
		} else {
			result.Data = payload
		}
		//k.Server.Log().Info(fmt.Sprintf("Data received is %v", payload.Data))
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

func (s *Subscriber) AddChannel(c string) {
}

func (s *Subscriber) AddFn(c string, fn FnMessage) {

}
