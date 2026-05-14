package main

import (
	"sync"
	"testing"
	"time"
)

func TestConversationFanout_SendDelivered(t *testing.T) {
	f := NewConversationFanout("s1")
	ch := f.Subscribe()
	defer f.Unsubscribe(ch)

	event := ConversationEvent{ID: "evt-1", Source: "test", SessionID: "s1", Text: "speak this"}
	f.Send(event)

	select {
	case msg := <-ch:
		if msg.ID != event.ID || msg.Text != event.Text || msg.Source != event.Source {
			t.Fatalf("expected %+v, got %+v", event, msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for conversation event")
	}
}

func TestConversationFanout_UnsubscribeClosesChannel(t *testing.T) {
	f := NewConversationFanout("s1")
	ch := f.Subscribe()
	f.Unsubscribe(ch)

	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after Unsubscribe")
	}
}

func TestConversationFanout_ConcurrentUnsubscribe(t *testing.T) {
	t.Parallel()
	f := NewConversationFanout("s1")

	const numGoroutines = 100
	const numSends = 1000

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < numSends; i++ {
			f.Send(ConversationEvent{ID: "evt", Source: "test", SessionID: "s1", Text: "msg"})
		}
	}()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			ch := f.Subscribe()
			select {
			case <-ch:
			default:
			}
			f.Unsubscribe(ch)
		}()
	}

	wg.Wait()
	<-done
}

func TestConversationFanout_DropsCountedAndLogged(t *testing.T) {
	f := NewConversationFanout("drop-test")
	ch := f.Subscribe()
	defer f.Unsubscribe(ch)

	// Fill exactly the per-subscriber buffer; no drop yet.
	for i := 0; i < conversationChannelBuffer; i++ {
		f.Send(ConversationEvent{ID: "fill", Source: "test", SessionID: "s1", Sequence: int64(i + 1), Text: "fill"})
	}

	if f.DropLogged() {
		t.Fatal("DropLogged should be false before overflow")
	}
	if f.DropCount() != 0 {
		t.Fatalf("expected drop count 0 before overflow, got %d", f.DropCount())
	}

	// Each subsequent send must overflow and be counted (every drop is logged
	// with its sequence so missing events can be correlated to their causes).
	const overflowCount = 3
	for i := 0; i < overflowCount; i++ {
		f.Send(ConversationEvent{
			ID:        "drop",
			Source:    "test",
			SessionID: "s1",
			Sequence:  int64(conversationChannelBuffer + 1 + i),
			Text:      "drop",
		})
	}

	if !f.DropLogged() {
		t.Error("expected DropLogged to be true after channel overflow")
	}
	if f.DropCount() != overflowCount {
		t.Errorf("expected drop count %d after overflow, got %d", overflowCount, f.DropCount())
	}

	// Resync signal should have been pulsed (at least once, capacity 1).
	resync := f.ResyncSignal(ch)
	if resync == nil {
		t.Fatal("expected non-nil resync signal for active subscription")
	}
	select {
	case <-resync:
	default:
		t.Error("expected resync signal to be pulsed after overflow")
	}
}

func TestConversationFanout_NoSubscribers(t *testing.T) {
	f := NewConversationFanout("s1")
	f.Send(ConversationEvent{ID: "evt", Source: "test", SessionID: "s1", Text: "nobody listening"})
}
