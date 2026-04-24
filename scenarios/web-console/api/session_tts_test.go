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

func TestSendConversation_DropsLoggedOnce(t *testing.T) {
	sess := &Session{
		ID:                  "drop-test",
		clients:             make(map[chan []byte]*ClientInfo),
		conversationClients: make(map[chan ConversationEvent]*conversationSubscriber),
	}

	ch := sess.SubscribeConversation()
	defer sess.UnsubscribeConversation(ch)

	event := ConversationEvent{ID: "evt", Source: "test", SessionID: "s1", Text: "fill"}
	// Fill exactly the per-subscriber buffer; no drop should be logged yet.
	for i := 0; i < conversationChannelBuffer; i++ {
		sess.SendConversation(event)
	}

	if sess.conversationDropLogged {
		t.Fatal("conversationDropLogged should be false before overflow")
	}

	// One more send must overflow and trigger the drop log + resync pulse.
	sess.SendConversation(event)
	sess.SendConversation(event)

	if !sess.conversationDropLogged {
		t.Error("expected conversationDropLogged to be true after channel overflow")
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
