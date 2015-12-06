package pakpos

import (
	"fmt"
	"github.com/eaciit/toolkit"
	"strings"
	"testing"
	"time"
)

var b *Broadcaster
var subs []*Subscriber

func init() {
	b = new(Broadcaster)

	b.Start(":19000")
	SetDefaultExpiry(5 * time.Second)
}

func TestConnect(t *testing.T) {
	var es []error
	ips, _ := toolkit.GetIP()
	for i := 0; i < 5; i++ {
		s := new(Subscriber)
		s.BroadcasterAddress = "http://localhost:19000"
		port := 19001 + i
		subs = append(subs, s)
		eStart := s.Start(fmt.Sprintf("%s:%d", ips[0], port))
		if eStart != nil {
			es = append(es, eStart)
		}
	}
	if len(es) > 0 {
		t.Errorf("Unable to connect \n%s",
			strings.Join(func() []string {
				var s []string
				for _, e := range es {
					s = append(s, e.Error())
				}
				return s
			}(), "\n"))
	}
}

func TestBroadcast(t *testing.T) {
	mm, _ := b.Broadcast("Message01", time.Now())
	mm.Wait()
	if mm.Success != len(mm.Targets) {
		t.Errorf("Unable to send Message01. Success %d of %d\n%s", mm.Success, len(mm.Targets), strings.Join(mm.Status, "\n"))
	}
}

/*
func TestSubscribe(t *testing.T) {
	subs[0].SubcribtionType = SubcribeChannel
	subs[1].SubcribtionType = SubcribeChannel
	subs[0].AddChannel("Group1")
	subs[0].AddChannel("Group1")
	mm, _ := b.Broadcast("Group1:OK", "OK Data")
	mm.Wait()
}

func TestSubscribeQue(t *testing.T) {
	mm, _ := b.Que("Que01", "Queue 01")
	mm.Wait()

	var es []string
	for _, s := range subs {
		s.Log().Info(fmt.Sprintf("Receive que on node %s", s.Address))
		_, e := s.Receive("Que01")
		if e != nil {
			es = append(es, e.Error())
		}
		time.Sleep(time.Duration(toolkit.RandInt(5000)) * time.Millisecond)
	}

	if len(es) == 0 {
		t.Errorf("Fail to receive que because: %s", strings.Join(es, "\n"))
	}
}
*/

func TestStop(t *testing.T) {
	time.Sleep(1 * time.Millisecond) //-- just to make sure time package used
	mm, _ := b.Broadcast("Stop", true)
	mm.Wait()
	if mm.Success != len(b.Subscibers) {
		t.Errorf("Fail to stop. Success %d of %d \n%s", mm.Success, len(b.Subscibers), strings.Join(mm.Status, "\n"))
	}
}
