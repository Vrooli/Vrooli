package policy

import (
	"testing"
	"time"
)

// [REQ:POL-005] Subscribe returns a channel that receives broadcast events.
func TestBroadcaster_SubscribeReceives(t *testing.T) {
	b := NewPolicyBroadcaster()

	id, ch := b.Subscribe()
	defer b.Unsubscribe(id)

	evt := PolicyEvent{Action: "created", RuleID: 1}
	b.Broadcast(evt)

	select {
	case got := <-ch:
		if got.Action != "created" || got.RuleID != 1 {
			t.Fatalf("unexpected event: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast event")
	}
}

// [REQ:POL-005] Unsubscribe closes the channel and removes the subscriber.
func TestBroadcaster_Unsubscribe(t *testing.T) {
	b := NewPolicyBroadcaster()

	id, ch := b.Subscribe()
	b.Unsubscribe(id)

	// Channel should be closed.
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}

	// Broadcasting after unsubscribe should not panic.
	b.Broadcast(PolicyEvent{Action: "deleted", RuleID: 99})
}

// [REQ:POL-005] Multiple subscribers each receive the broadcast.
func TestBroadcaster_MultipleSubscribers(t *testing.T) {
	b := NewPolicyBroadcaster()

	id1, ch1 := b.Subscribe()
	defer b.Unsubscribe(id1)
	id2, ch2 := b.Subscribe()
	defer b.Unsubscribe(id2)

	evt := PolicyEvent{Action: "updated", RuleID: 42, Rule: &Rule{ID: 42}}
	b.Broadcast(evt)

	for i, ch := range []<-chan PolicyEvent{ch1, ch2} {
		select {
		case got := <-ch:
			if got.RuleID != 42 || got.Action != "updated" {
				t.Fatalf("subscriber %d: unexpected event: %+v", i, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timeout", i)
		}
	}
}

// [REQ:POL-005] Non-blocking send skips slow subscribers without blocking.
func TestBroadcaster_NonBlockingSend(t *testing.T) {
	b := NewPolicyBroadcaster()

	id, ch := b.Subscribe()
	defer b.Unsubscribe(id)

	// Fill the buffer (capacity 64).
	for i := 0; i < 64; i++ {
		b.Broadcast(PolicyEvent{Action: "created", RuleID: int64(i)})
	}

	// The 65th broadcast should not block.
	done := make(chan struct{})
	go func() {
		b.Broadcast(PolicyEvent{Action: "created", RuleID: 999})
		close(done)
	}()

	select {
	case <-done:
		// Success: broadcast returned without blocking.
	case <-time.After(time.Second):
		t.Fatal("broadcast blocked on full subscriber channel")
	}

	// Drain and verify the channel has the first 64 events.
	count := 0
	for range ch {
		count++
		if count == 64 {
			break
		}
	}
	if count != 64 {
		t.Fatalf("expected 64 buffered events, got %d", count)
	}
}

// [REQ:POL-005] Double unsubscribe does not panic.
func TestBroadcaster_DoubleUnsubscribe(t *testing.T) {
	b := NewPolicyBroadcaster()
	id, _ := b.Subscribe()
	b.Unsubscribe(id)
	b.Unsubscribe(id) // Should not panic.
}
