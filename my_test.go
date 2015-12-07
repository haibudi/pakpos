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

func TestSubscribe(t *testing.T) {
	//subs[0].SubcribtionType = SubcribeChannel
	//subs[1].SubcribtionType = SubcribeChannel
	e1 := subs[1].AddChannel("Group1")
	e2 := subs[3].AddChannel("Group1")
	if e1 != nil || e2 != nil {
		t.Errorf("Unable to add channel:\n1: %v\n2: %v", e1, e2)
		return
	}
	mm, _ := b.Broadcast("Group1:OK", "OK Data")
	mm.Wait()
	want := 2
	if len(mm.Targets) != want {
		t.Errorf("Invalid target, want %d got %d", want, len(mm.Targets))
	}
}

func TestSubscribeQue(t *testing.T) {
	mm, _ := b.Que("Que01", "Queue 01")
	go func() {
		mm.Wait()
	}()

	time.Sleep(2 * time.Millisecond)
	for _, s := range subs {
		d, e := s.GetMsg("Que01")
		if e == nil {
			t.Logf("Msg Que01 on %s receive %v", s.Address, d)
		} else {
			t.Logf("Msg Que01 on %s error %s", s.Address, e.Error())
		}
	}
}

func TestStop(t *testing.T) {
	time.Sleep(1 * time.Millisecond) //-- just to make sure time package used
	mm, _ := b.Broadcast("Stop", true)
	mm.Wait()
	if mm.Success != len(b.Subscibers) {
		t.Errorf("Fail to stop. Success %d of %d \n%s", mm.Success, len(b.Subscibers), strings.Join(mm.Status, "\n"))
	}
}
