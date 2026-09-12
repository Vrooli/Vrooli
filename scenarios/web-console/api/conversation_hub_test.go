package main

import (
	"sync"
	"testing"
	"time"
)

func testEnvelope(sessionID string) HubEnvelope {
	return HubEnvelope{
		SessionID: sessionID,
		Kind:      HubKindConversationEvent,
		Sequence:  1,
		Payload:   conversationEventPayload{ID: "evt", Text: "hi"},
	}
}

func TestConversationHub_PublishAssignsMonotonicIDs(t *testing.T) {
	h := NewConversationHub()
	const n = 50
	prev := int64(0)
	for i := 0; i < n; i++ {
		id := h.Publish(testEnvelope("s1"))
		if id != prev+1 {
			t.Fatalf("publish %d: expected id %d, got %d", i, prev+1, id)
		}
		prev = id
	}
}

func TestConversationHub_ConcurrentSubscribersReceiveFullStream(t *testing.T) {
	h := NewConversationHub()

	const numSubs = 8
	const numEvents = 100

	subs := make([]*hubSubscriber, numSubs)
	for i := range subs {
		subs[i], _, _ = h.Subscribe(0)
		defer h.Unsubscribe(subs[i])
	}

	var wg sync.WaitGroup
	got := make([][]int64, numSubs)
	for i := range subs {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < numEvents; j++ {
				select {
				case env := <-subs[idx].events:
					got[idx] = append(got[idx], env.ID)
				case <-time.After(2 * time.Second):
					return
				}
			}
		}(i)
	}

	// Give readers time to block on their channels, then publish.
	time.Sleep(20 * time.Millisecond)
	for j := 0; j < numEvents; j++ {
		h.Publish(testEnvelope("s1"))
	}

	wg.Wait()

	for i := range got {
		if len(got[i]) != numEvents {
			t.Fatalf("subscriber %d: expected %d events, got %d", i, numEvents, len(got[i]))
		}
		for k, id := range got[i] {
			if id != int64(k+1) {
				t.Fatalf("subscriber %d: event %d out of order: got id %d", i, k, id)
			}
		}
	}
}

func TestConversationHub_RingRetainsLastN(t *testing.T) {
	h := NewConversationHub()
	total := hubRingSize + 100
	for i := 0; i < total; i++ {
		h.Publish(testEnvelope("s1"))
	}
	// Resume from id 1: every retained entry has id > 1, so the full window
	// comes back (capped at hubRingSize).
	_, replay, _ := h.Subscribe(1)
	if len(replay) != hubRingSize {
		t.Fatalf("expected %d retained entries newer than id 1, got %d", hubRingSize, len(replay))
	}
	// Oldest retained id should be total-hubRingSize+1.
	wantOldest := int64(total - hubRingSize + 1)
	if replay[0].ID != wantOldest {
		t.Fatalf("expected oldest retained id %d, got %d", wantOldest, replay[0].ID)
	}
	if replay[len(replay)-1].ID != int64(total) {
		t.Fatalf("expected newest retained id %d, got %d", total, replay[len(replay)-1].ID)
	}
}

func TestConversationHub_ReplayReturnsOnlyNewerInOrder(t *testing.T) {
	h := NewConversationHub()
	for i := 0; i < 10; i++ {
		h.Publish(testEnvelope("s1"))
	}
	_, replay, gap := h.Subscribe(7)
	if gap {
		t.Fatal("did not expect a gap when cursor is within the retained window")
	}
	if len(replay) != 3 {
		t.Fatalf("expected 3 replayed entries (8,9,10), got %d", len(replay))
	}
	for i, env := range replay {
		want := int64(8 + i)
		if env.ID != want {
			t.Fatalf("replay[%d]: expected id %d, got %d", i, want, env.ID)
		}
	}
}

func TestConversationHub_GapYieldsOutOfSync(t *testing.T) {
	h := NewConversationHub()
	// Fill well past the ring so the oldest entries are evicted.
	total := hubRingSize + 50
	for i := 0; i < total; i++ {
		h.Publish(testEnvelope("s1"))
	}
	// Cursor pointing at id 1 is far older than the retained window → gap.
	_, _, gap := h.Subscribe(1)
	if !gap {
		t.Fatal("expected a gap when cursor predates the retained window")
	}
}

func TestConversationHub_NoGapAtWindowBoundary(t *testing.T) {
	h := NewConversationHub()
	total := hubRingSize + 50
	for i := 0; i < total; i++ {
		h.Publish(testEnvelope("s1"))
	}
	// oldest retained id = total-hubRingSize+1. A cursor exactly one below it
	// means the next event is still retained → no gap.
	oldest := int64(total - hubRingSize + 1)
	_, replay, gap := h.Subscribe(oldest - 1)
	if gap {
		t.Fatalf("did not expect a gap when cursor is oldest-1 (%d)", oldest-1)
	}
	if len(replay) != hubRingSize {
		t.Fatalf("expected full window replay (%d), got %d", hubRingSize, len(replay))
	}
}

func TestConversationHub_BackpressureDropsAndSignalsResync(t *testing.T) {
	h := NewConversationHub()
	sub, _, _ := h.Subscribe(0)
	defer h.Unsubscribe(sub)

	// Fill the subscriber buffer exactly; no drop yet.
	for i := 0; i < hubSubscriberBuffer; i++ {
		h.Publish(testEnvelope("sX"))
	}
	if h.DropCount() != 0 {
		t.Fatalf("expected no drops at buffer capacity, got %d", h.DropCount())
	}

	// Overflow — these must be dropped and resync pulsed with the session id.
	h.Publish(testEnvelope("sX"))
	if h.DropCount() == 0 {
		t.Fatal("expected at least one drop after overflow")
	}
	select {
	case sid := <-sub.resync:
		if sid != "sX" {
			t.Fatalf("expected resync session id sX, got %q", sid)
		}
	default:
		t.Fatal("expected resync signal after overflow")
	}
}

func TestConversationHub_RetainedSessionIDsDistinct(t *testing.T) {
	h := NewConversationHub()
	h.Publish(testEnvelope("a"))
	h.Publish(testEnvelope("b"))
	h.Publish(testEnvelope("a"))
	ids := h.RetainedSessionIDs()
	if len(ids) != 2 {
		t.Fatalf("expected 2 distinct session ids, got %d: %v", len(ids), ids)
	}
}

func TestPublishConversationEvent_MapsUpdateKind(t *testing.T) {
	srv := &Server{hub: NewConversationHub()}
	sub, _, _ := srv.hub.Subscribe(0)
	defer srv.hub.Unsubscribe(sub)

	srv.publishConversationEvent(ConversationEvent{
		SessionID: "s1",
		Sequence:  3,
		IsUpdate:  true,
		Text:      "summarized",
		Role:      ConversationRoleAssistant,
		CreatedAt: time.Now().UTC(),
	})

	select {
	case env := <-sub.events:
		if env.Kind != HubKindConversationEventUpdate {
			t.Fatalf("expected kind %q, got %q", HubKindConversationEventUpdate, env.Kind)
		}
		if env.SessionID != "s1" || env.Sequence != 3 {
			t.Fatalf("unexpected envelope fields: %+v", env)
		}
		if _, ok := env.Payload.(conversationEventPayload); !ok {
			t.Fatalf("expected conversationEventPayload payload, got %T", env.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for published update event")
	}
}
