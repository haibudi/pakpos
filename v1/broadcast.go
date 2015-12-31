package pakpos

import (
	"fmt"
	"github.com/eaciit/knot/knot.v1"
	"github.com/eaciit/toolkit"
	_ "net/http"
	"net/url"
	"os"
	//"os/exec"
	"strings"
	//"time"
)

var (
	pro string
)

type SubscriberInfo struct {
	Address, Protocol string
	Secret            string
}

func (si *SubscriberInfo) Url(s string) string {
	return fmt.Sprintf("%s://%s/%s", si.Protocol, si.Address, s)
}

type Broadcaster struct {
	knot.Server
	Subscibers map[string]*SubscriberInfo

	secret             string // this secret will be used to validate addnode request
	userTokens         map[string]*Token
	channelSubscribers map[string][]string
	messages           map[string]*MessageMonitor
	Certificate        string
	PrivateKey         string
}

func (b *Broadcaster) Start(address, secret string) {
	var u, er = url.Parse(address)
	if er != nil {
		fmt.Println(er.Error())
	}

	var scheme = u.Scheme
	pro = scheme

	if scheme == "https" {
		basepath, _ := os.Getwd()
		b.Address = address
		b.secret = secret
		app := knot.NewApp("pakpos")
		app.Register(b)
		app.UseSSL = true
		app.CertificatePath = basepath + b.Certificate
		app.PrivateKeyPath = basepath + b.PrivateKey
		app.DefaultOutputType = knot.OutputJson
		knot.RegisterApp(app)
		go func() {
			knot.StartApp(app, address)
		}()

	} else {
		b.Address = address
		b.secret = secret
		app := knot.NewApp("pakpos")
		app.Register(b)
		app.DefaultOutputType = knot.OutputJson
		knot.RegisterApp(app)
		go func() {
			knot.StartApp(app, address)
		}()
	}
}

func (b *Broadcaster) initToken() {
	if b.userTokens == nil {
		b.userTokens = map[string]*Token{}
	}

	if b.channelSubscribers == nil {
		b.channelSubscribers = map[string][]string{}
	}

	if b.messages == nil {
		b.messages = map[string]*MessageMonitor{}
	}
}

func (b *Broadcaster) validateToken(tokentype, secret, reference string) bool {
	b.initToken()
	ret := false
	if tokentype == "user" {
		if t, e := b.userTokens[reference]; !e {
			return false
		} else {
			if t.Secret != secret {
				return false
			} else {
				return true
			}
		}
	} else if tokentype == "node" {
		if s, e := b.Subscibers[reference]; !e {
			return false
		} else {
			if s.Secret != secret {
				return false
			} else {
				return true
			}
		}
	}
	return ret
}

func (b *Broadcaster) AddNode(k *knot.WebContext) interface{} {
	var nodeModel struct {
		Subscriber string
		Secret     string
	}
	k.GetPayload(&nodeModel)

	result := toolkit.NewResult()
	if nodeModel.Subscriber == "" {
		result.SetErrorTxt("Invalid subscriber info " + nodeModel.Subscriber)
		return result
	}

	if nodeModel.Secret != b.secret {
		result.SetErrorTxt("Not authorised")
		return result
	}

	if b.Subscibers == nil {
		b.Subscibers = map[string]*SubscriberInfo{}
	} else {
		if _, exist := b.Subscibers[nodeModel.Subscriber]; exist {
			result.SetErrorTxt(fmt.Sprintf("%s has been registered as subsriber", nodeModel.Subscriber))
			return result
		}
	}
	si := new(SubscriberInfo)
	si.Address = nodeModel.Subscriber
	si.Protocol = pro
	si.Secret = toolkit.RandomString(32)
	b.Subscibers[nodeModel.Subscriber] = si
	result.Data = si.Secret
	k.Server.Log().Info(fmt.Sprintf("%s register as new subscriber. Current active subscibers: %d", nodeModel.Subscriber, len(b.Subscibers)))
	return result
}

func (b *Broadcaster) RemoveNode(k *knot.WebContext) interface{} {
	var nodeModel struct {
		Subscriber string
		Secret     string
	}
	k.GetPayload(&nodeModel)

	result := toolkit.NewResult()
	if nodeModel.Subscriber == "" {
		result.SetErrorTxt("Invalid subscriber info " + nodeModel.Subscriber)
		return result
	}

	if b.validateToken("node", nodeModel.Secret, nodeModel.Subscriber) == false {
		result.SetErrorTxt("Not authorised")
		return result
	}

	sub := b.Subscibers[nodeModel.Subscriber]
	_, e := toolkit.CallResult(sub.Url("subscriber/stop"), "POST", toolkit.M{}.Set("secret", sub.Secret).ToBytes("json", nil))
	if e != nil {
		result.SetErrorTxt(e.Error())
		return result
	}

	delete(b.Subscibers, nodeModel.Subscriber)
	k.Server.Log().Info(fmt.Sprintf("%s has been removed as subscriber. Current active subscibers: %d", nodeModel.Subscriber, len(b.Subscibers)))
	return result
}

func (b *Broadcaster) Broadcast(k *knot.WebContext) interface{} {
	result := toolkit.NewResult()
	var model struct {
		UserID, Secret, Key string
		Data                interface{}
	}
	k.GetPayload(&model)
	if b.validateToken("user", model.Secret, model.UserID) == false {
		result.SetErrorTxt("User " + model.UserID + " is not authorised")
		return result
	}

	targets := b.getChannelSubscribers(model.Key)
	if len(targets) == 0 {
		result.SetErrorTxt("No subscriber can receive this message")
		return result
	}

	mm := NewMessageMonitor(b, model.Key, model.Data, DefaultExpiry())
	mm.DistributionType = DistributeAsBroadcast
	result.Data = mm.Message
	go func() {
		mm.Wait()
	}()
	return result
}

func (b *Broadcaster) Que(k *knot.WebContext) interface{} {
	result := toolkit.NewResult()
	var model struct {
		UserID, Secret, Key string
		Data                interface{}
	}
	k.GetPayload(&model)
	if b.validateToken("user", model.Secret, model.UserID) == false {
		result.SetErrorTxt("User " + model.UserID + " is not authorised")
		return result
	}

	targets := b.getChannelSubscribers(model.Key)
	if len(targets) == 0 {
		result.SetErrorTxt("No subscriber can receive this message")
		return result
	}

	mm := NewMessageMonitor(b, model.Key, model.Data, DefaultExpiry())
	mm.DistributionType = DistributeAsQue
	result.Data = mm.Message
	go func() {
		mm.Wait()
	}()
	return result
}

func (b *Broadcaster) GetMessage(k *knot.WebContext) interface{} {
	result := toolkit.NewResult()
	var model struct {
		Subscriber, Secret, Key string
	}
	k.GetPayload(&model)
	if model.Key == "" {
		return result.SetErrorTxt("Key is empty")
	}
	if !b.validateToken("node", model.Secret, model.Subscriber) {
		return result.SetErrorTxt("Subscriber is not authorised")
	}
	msg, exist := b.messages[model.Key]
	if !exist {
		return result.SetErrorTxt(fmt.Sprintf("Message %s is not exist. Either it has been fully collected or message is never exist", model.Key))
	}
	targets := b.getChannelSubscribers(model.Key)
	found := false
	foundIndex := 0
	for idx, t := range targets {
		if t == model.Subscriber {
			found = true
			foundIndex = idx
			break
		}
	}
	if !found {
		return result.SetErrorTxt(fmt.Sprintf("Subscibers %s is not valid subscriber of message %s", model.Subscriber, model.Key))
	}
	targets = append(targets[0:foundIndex], targets[foundIndex+1:]...)
	if len(targets) == 0 {
		delete(b.channelSubscribers, model.Key)
	} else {
		b.channelSubscribers[model.Key] = targets
	}
	result.Data = msg.Data
	return result
}

func (b *Broadcaster) MsgStatus(k *knot.WebContext) interface{} {
	k.Config.NoLog = true
	result := toolkit.NewResult()
	var model struct {
		UserID, Secret, Key string
	}
	k.GetPayload(&model)
	if b.validateToken("user", model.Secret, model.UserID) == false {
		result.SetErrorTxt("Not authorised")
		return result
	}

	m, e := b.messages[model.Key]
	if e == false {
		result.SetErrorTxt("Message " + model.Key + " is not exist")
		return result
	}

	result.Data = fmt.Sprintf("Targets: %d, Success: %d, Fail: %d", len(m.Targets), m.Success, m.Fail)
	return result
}

func (b *Broadcaster) getChannelSubscribers(key string) []string {
	channel, key := ParseKey(key)
	//fmt.Println("Get subsriber for " + channel + ":" + key)
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

func (b *Broadcaster) SubscribeChannel(k *knot.WebContext) interface{} {
	result := toolkit.NewResult()
	var model struct {
		Subscriber, Secret, Channel string
	}
	k.GetPayload(&model)
	if b.validateToken("node", model.Secret, model.Subscriber) == false {
		result.SetErrorTxt("Subscriber not authorised")
		return result
	}
	model.Channel = strings.ToLower(model.Channel)
	subs, exist := b.channelSubscribers[model.Channel]
	if !exist {
		subs = []string{}
	}
	for _, s := range subs {
		if s == model.Subscriber {
			result.SetErrorTxt(fmt.Sprintf("%s has been subscribe to channel %s", model.Subscriber, model.Channel))
			return result
		}
	}
	subs = append(subs, model.Subscriber)
	b.channelSubscribers[model.Channel] = subs
	b.Server.Log().Info(fmt.Sprintf("Node %s successfully subscibed to channel %s. Current subscribers count: %d", model.Subscriber, model.Channel, len(subs)))
	return result
}

func (b *Broadcaster) Stop(k *knot.WebContext) interface{} {
	m := toolkit.M{}
	result := toolkit.NewResult()
	k.GetPayload(&m)
	defer func() {
		if result.Status == toolkit.Status_OK {
			k.Server.Stop()
		}
	}()
	valid := b.validateToken("user", m.GetString("secret"),
		m.GetString("userid"))
	if !valid {
		result.SetErrorTxt("Not authorised")
		return result
	}

	return result
}

func (b *Broadcaster) Login(k *knot.WebContext) interface{} {
	r := toolkit.NewResult()
	var loginInfo struct{ UserID, Password string }
	k.GetPayload(&loginInfo)
	if loginInfo.UserID != "arief" && loginInfo.Password != "darmawan" {
		r.SetErrorTxt("Invalid Login")
		return r
	}

	b.initToken()
	if t, te := b.userTokens[loginInfo.UserID]; te {
		if !t.IsExpired() {
			r.SetErrorTxt("User " + loginInfo.UserID + " has logged-in before and already has a valid session created")
		}
	}

	t := NewToken("user", loginInfo.UserID)
	b.userTokens[loginInfo.UserID] = t
	r.Data = t.Secret
	return r
}

func (b *Broadcaster) Logout(k *knot.WebContext) interface{} {
	r := toolkit.NewResult()
	m := toolkit.M{}
	k.GetPayload(&m)
	if !b.validateToken("user", m.GetString("secret"), m.GetString("userid")) {
		r.SetErrorTxt("Not authorised")
		return r
	}
	delete(b.userTokens, m.GetString("userid"))
	return r
}
