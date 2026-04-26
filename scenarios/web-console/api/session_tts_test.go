package main

import (
	"sync"
	"testing"
	"time"
)

func TestSubscribeConversation_SendConversation_Delivered(t *testing.T) {
	sess := &Session{
		clients:             make(map[chan []byte]*ClientInfo),
		conversationClients: make(map[chan ConversationEvent]*conversationSubscriber),
	}

	ch := sess.SubscribeConversation()
	defer sess.UnsubscribeConversation(ch)

	event := ConversationEvent{ID: "evt-1", Source: "test", SessionID: "s1", Text: "speak this"}
	sess.SendConversation(event)

	select {
	case msg := <-ch:
		if msg.ID != event.ID || msg.Text != event.Text || msg.Source != event.Source {
			t.Fatalf("expected %+v, got %+v", event, msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for conversation event")
	}
}

func TestUnsubscribeConversation_ChannelClosed(t *testing.T) {
	sess := &Session{
		clients:             make(map[chan []byte]*ClientInfo),
		conversationClients: make(map[chan ConversationEvent]*conversationSubscriber),
	}

	ch := sess.SubscribeConversation()
	sess.UnsubscribeConversation(ch)

	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after UnsubscribeConversation")
	}
}

func TestSendConversation_ConcurrentUnsubscribe(t *testing.T) {
	t.Parallel()
	sess := &Session{
		clients:             make(map[chan []byte]*ClientInfo),
		conversationClients: make(map[chan ConversationEvent]*conversationSubscriber),
	}

	const numGoroutines = 100
	const numSends = 1000

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < numSends; i++ {
			sess.SendConversation(ConversationEvent{ID: "evt", Source: "test", SessionID: "s1", Text: "msg"})
		}
	}()

	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			ch := sess.SubscribeConversation()
			select {
			case <-ch:
			default:
			}
			sess.UnsubscribeConversation(ch)
		}()
	}

	wg.Wait()
	<-done
}

func TestSendConversation_DropsCountedAndLogged(t *testing.T) {
	sess := &Session{
		ID:                  "drop-test",
		clients:             make(map[chan []byte]*ClientInfo),
		conversationClients: make(map[chan ConversationEvent]*conversationSubscriber),
	}

	ch := sess.SubscribeConversation()
	defer sess.UnsubscribeConversation(ch)

	// Fill exactly the per-subscriber buffer; no drop yet.
	for i := 0; i < conversationChannelBuffer; i++ {
		sess.SendConversation(ConversationEvent{ID: "fill", Source: "test", SessionID: "s1", Sequence: int64(i + 1), Text: "fill"})
	}

	if sess.conversationDropLogged {
		t.Fatal("conversationDropLogged should be false before overflow")
	}
	if sess.conversationDropCount != 0 {
		t.Fatalf("expected drop count 0 before overflow, got %d", sess.conversationDropCount)
	}

	// Each subsequent send must overflow and be counted (every drop is logged
	// with its sequence so missing events can be correlated to their causes).
	const overflowCount = 3
	for i := 0; i < overflowCount; i++ {
		sess.SendConversation(ConversationEvent{
			ID:        "drop",
			Source:    "test",
			SessionID: "s1",
			Sequence:  int64(conversationChannelBuffer + 1 + i),
			Text:      "drop",
		})
	}

	if !sess.conversationDropLogged {
		t.Error("expected conversationDropLogged to be true after channel overflow")
	}
	if sess.conversationDropCount != overflowCount {
		t.Errorf("expected drop count %d after overflow, got %d", overflowCount, sess.conversationDropCount)
	}

	// Resync signal should have been pulsed (at least once, capacity 1).
	resync := sess.ConversationResyncSignal(ch)
	if resync == nil {
		t.Fatal("expected non-nil resync signal for active subscription")
	}
	select {
	case <-resync:
	default:
		t.Error("expected resync signal to be pulsed after overflow")
	}
}

func TestSendConversation_NoSubscribers(t *testing.T) {
	sess := &Session{
		clients:             make(map[chan []byte]*ClientInfo),
		conversationClients: make(map[chan ConversationEvent]*conversationSubscriber),
	}
	sess.SendConversation(ConversationEvent{ID: "evt", Source: "test", SessionID: "s1", Text: "nobody listening"})
}
