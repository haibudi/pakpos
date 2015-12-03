package pakpos

import (
	"fmt"
	"github.com/eaciit/toolkit"
	"testing"
	"time"
)

var b *Broadcaster
var subs []*Subscriber

func init() {
	b = new(Broadcaster)
	b.Start(":19000")
	for i := 0; i < 5; i++ {
		s = new(Subscriber)
		s.BroadcasterAddress = ":19000"
		port := 19001 + i
		s.Start(fmt.Sprintf(":%d", port))
		subs = append(subs, s)
	}
}

func TestBroadcast(t *testing.T) {
	mm := b.Broadcast("Message01", time.Now())
	mm.Wait()
}

func TestSubscribe(t *testing.T) {
	subs[0].SubcribtionType = SubcribeChannel
	subs[1].SubcribtionType = SubcribeChannel
	subs[0].AddChannel("Group1")
	subs[0].AddChannel("Group1")
	mm := b.Broadcast("Group1:OK")
	mm.Wait()
}

func TestSubcribeWithAction(t *testing.T) {
	for _, s := range subs {
		s.AddFn("DoThis", func(kv toolkit.KvString) interface{} {

		})
	}
}

func TestStop(t *testing.T) {
	mm := b.Broadcast("Stop")
	mm.Wait()
	if mm.Success == len(b.Subscibers) {
		b.Stop()
	} else {
		t.Errorf("Fail to stop. Fail nodes: %d", mm.Fail)
	}
}
