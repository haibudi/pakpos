package pakpos_test_1

import (
	"fmt"
	"github.com/eaciit/toolkit"
	. "pakpos/v1"
	//"strings"
	"testing"
	"time"
	//"net/http"
)

var (
	b               *Broadcaster
	broadcastSecret string = "uliyamahalin"
	userSecret      string
	userid          string = "arief"
	password        string = "darmawan"
	subs            []*Subscriber
)

func init() {
	b = new(Broadcaster)
	b.CertificatePath = "/cert.pem"
	b.PrivateKeyPath = "/key.pem"
	b.Start("https://localhost:12345", broadcastSecret)

	/*
		for i := 0; i < 3; i++ {
			subs = append(subs, new(Subscriber))
			address := fmt.Sprintf("localhost:%d", 2345+i)
			subs[i].BroadcasterAddress = "http://" + b.Address
			subs[i].Start(address)
		}
	*/
}

func TestLogin(t *testing.T) {
	r, e := call(b.Address+"/broadcaster/login", "POST",
		toolkit.M{}.Set("userid", userid).Set("password", password).ToBytes("json", nil), 200)
	if e != nil {
		t.Errorf("Fail to login. " + e.Error())
		return
	}

	userSecret = r.Data.(string)
	t.Logf("Login success. Secret token is " + userSecret)
}

func TestAddNodes(t *testing.T) {
	for i := 0; i < 3; i++ {
		sub := new(Subscriber)
		//sub.Broadcaster = "http://localhost:12345"
		e := sub.Start(fmt.Sprintf("localhost:%d", 54321+i), "http://localhost:12345", broadcastSecret)
		if e != nil {
			t.Errorf("Fail start node %d : %s", i, e.Error())
		} else {
			t.Logf("%s has been subscribed with secret: %s", sub.Address, sub.Secret)
		}
		subs = append(subs, sub)
	}
}

func TestBroadcast(t *testing.T) {
	_, e := call(b.Address+"/broadcaster/broadcast", "POST",
		toolkit.M{}.
			Set("userid", userid).
			Set("secret", userSecret).
			Set("key", "BC01").
			Set("data", "Ini adalah Broadcast Message 01").
			ToBytes("json", nil),
		200)
	if e != nil {
		t.Errorf("%s %s", b.Address+"/broadcaster/broadcast", e.Error())
		return
	}
	time.Sleep(2 * time.Second)

	/*
		var msg Message
		//toolkit.Unjson(toolkit.Jsonify(r.Data.(map[string]interface{})), &msg)
		e = r.Cast(&msg, "json")
		if e != nil {
			t.Errorf("Unable to serde result: %s", e.Error())
			return
		}
		expiry := msg.Expiry
		for {
			r, e := call(b.Address+"/broadcaster/msgstatus", "POST",
				toolkit.M{}.Set("userid", userid).
					Set("secret", userSecret).
					Set("key", "BC01").
					ToBytes("json", nil),
				200)
			if e == nil {
				t.Logf("%v %s", time.Now(), r.Data.(string))
			} else {
				str := e.Error()
				if strings.Contains(str, "not exist") {
					break
				} else {
					t.Errorf(str)
					break
				}
			}
			time.Sleep(1 * time.Second)
			if time.Now().After(expiry) {
				t.Errorf("Message expiry exceeded. Process will be exited")
				return
			}
		}
	*/
}

func TestSubcribeChannel(t *testing.T) {
	_, e := toolkit.CallResult(b.Address+"/broadcaster/subscribechannel", "POST",
		toolkit.M{}.Set("Subscriber", subs[1].Address).Set("Secret", subs[1].Secret).Set("Channel", "Ch01").ToBytes("json", nil))
	if e != nil {
		t.Error(e)
		return
	}

	_, e = call(b.Address+"/broadcaster/broadcast", "POST",
		toolkit.M{}.
			Set("userid", userid).
			Set("secret", userSecret).
			Set("key", "Ch01:Message01").
			Set("data", "Ini adalah Channel 01 - Broadcast Message 01").
			ToBytes("json", nil),
		200)
	if e != nil {
		t.Errorf("%s %s", b.Address+"/broadcaster/broadcast", e.Error())
		return
	}
	time.Sleep(2 * time.Second)

}

func TestQue(t *testing.T) {
	_, e := toolkit.CallResult(b.Address+"/broadcaster/que", "POST",
		toolkit.M{}.Set("userid", userid).Set("secret", userSecret).Set("key", "Ch01:QueMessage01").Set("data", "Ini adalah Channel 01 Que Message 01").ToBytes("json", nil))

	if e != nil {
		t.Errorf("Sending Que Error: " + e.Error())
		return
	}

	time.Sleep(3 * time.Second)
	found := 0
	for _, s := range subs {
		_, e = toolkit.CallResult("http://"+s.Address+"/subscriber/collectmessage", "POST",
			toolkit.M{}.Set("subscriber", s.Address).Set("secret", s.Secret).Set("key", "").ToBytes("json", nil))
		if e == nil {
			found++
		} else {
			t.Logf("Node %s, could not collect message: %s", s.Address, e.Error())
		}
	}

	if found != 1 {
		t.Errorf("Que not collected properly. Should only 1 subscriber collecting it, got %d", found)
	}
}
func TestQueInvalid(t *testing.T) {
	_, e := toolkit.CallResult(b.Address+"/broadcaster/que", "POST",
		toolkit.M{}.Set("userid", userid).Set("secret", userSecret).Set("key", "Ch01:QueMessage02").Set("data", "Ini adalah Channel 02 Que Message 02").ToBytes("json", nil))

	if e != nil {
		t.Errorf("Sending Que Error: " + e.Error())
		return
	}

	time.Sleep(15 * time.Second)
}

func TestClose0(t *testing.T) {
	_, e := call(b.Address+"/broadcaster/stop", "GET",
		toolkit.M{}.Set("userid", userid).Set("secret", userSecret).ToBytes("json", nil), 200)
	if e != nil {
		t.Errorf(e.Error())
	}
}

func call(url, call string, data []byte, expectedStatus int) (*toolkit.Result, error) {
	cfg := toolkit.M{}
	if expectedStatus != 0 {
		cfg.Set("expectedstatus", expectedStatus)
	}
	r, e := toolkit.HttpCall(url, call, data, cfg)
	if e != nil {
		return nil, fmt.Errorf(url + " Call Error: " + e.Error())
	}
	result := toolkit.NewResult()
	bs := toolkit.HttpContent(r)
	edecode := toolkit.Unjson(bs, result)
	if edecode != nil {
		return nil, fmt.Errorf(url + " Http Result Decode Error: " + edecode.Error() + "\nFollowing got: " + string(bs))
	}
	if result.Status == toolkit.Status_NOK {
		return nil, fmt.Errorf(url + " Http Result Error: " + result.Message)
	}
	return result, nil
}
