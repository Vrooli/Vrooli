package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:P1-004a] Structured Event Logging - event logger tests

func TestEventLogger_EmitAndRecent(t *testing.T) {
	el := NewEventLogger(100)

	el.Emit(EventSessionCreated, "sess-1", map[string]string{"shell": "/bin/bash"})
	el.Emit(EventSessionConnected, "sess-1", nil)
	el.Emit(EventSessionDisconnected, "sess-1", nil)

	events := el.Recent(10)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	if events[0].Type != EventSessionCreated {
		t.Errorf("expected first event type %q, got %q", EventSessionCreated, events[0].Type)
	}
	if events[0].SessionID != "sess-1" {
		t.Errorf("expected session ID %q, got %q", "sess-1", events[0].SessionID)
	}
	if events[0].Details["shell"] != "/bin/bash" {
		t.Errorf("expected detail shell=/bin/bash, got %q", events[0].Details["shell"])
	}
	if events[1].Type != EventSessionConnected {
		t.Errorf("expected second event type %q, got %q", EventSessionConnected, events[1].Type)
	}
	if events[2].Type != EventSessionDisconnected {
		t.Errorf("expected third event type %q, got %q", EventSessionDisconnected, events[2].Type)
	}
}

func TestEventLogger_RecentLimitsOutput(t *testing.T) {
	el := NewEventLogger(100)

	for i := 0; i < 5; i++ {
		el.Emit(EventSessionCreated, "sess", nil)
	}

	events := el.Recent(3)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestEventLogger_BoundedHistory(t *testing.T) {
	el := NewEventLogger(3) // Small buffer

	for i := 0; i < 10; i++ {
		el.Emit(EventSessionCreated, "sess", nil)
	}

	events := el.Recent(0)
	if len(events) != 3 {
		t.Fatalf("expected history capped at 3, got %d", len(events))
	}
}

func TestEventLogger_Count(t *testing.T) {
	el := NewEventLogger(100)

	if el.Count() != 0 {
		t.Fatalf("expected 0 events initially, got %d", el.Count())
	}

	el.Emit(EventSessionCreated, "s1", nil)
	el.Emit(EventSessionDeleted, "s1", nil)

	if el.Count() != 2 {
		t.Fatalf("expected 2 events, got %d", el.Count())
	}
}

func TestEventLogger_AllEventTypes(t *testing.T) {
	el := NewEventLogger(100)

	types := []string{
		EventSessionCreated,
		EventSessionConnected,
		EventSessionDisconnected,
		EventSessionTerminated,
		EventSessionDeleted,
		EventPaneResized,
	}
	for _, et := range types {
		el.Emit(et, "sess", nil)
	}

	events := el.Recent(0)
	if len(events) != len(types) {
		t.Fatalf("expected %d events, got %d", len(types), len(events))
	}
	for i, evt := range events {
		if evt.Type != types[i] {
			t.Errorf("event[%d]: expected type %q, got %q", i, types[i], evt.Type)
		}
		if evt.Timestamp.IsZero() {
			t.Errorf("event[%d]: timestamp should not be zero", i)
		}
	}
}

func TestEventLogger_NilDetails(t *testing.T) {
	el := NewEventLogger(100)
	el.Emit(EventSessionCreated, "s1", nil)

	events := el.Recent(1)
	if len(events) != 1 {
		t.Fatal("expected 1 event")
	}
	if events[0].Details != nil {
		t.Error("expected nil details for event emitted with nil details")
	}
}

// [REQ:P1-004a] EventSessionPolicyUpdate constant is used instead of inline string
func TestEventLogger_PolicyUpdateConstant(t *testing.T) {
	if EventSessionPolicyUpdate != "session.policy_updated" {
		t.Errorf("expected %q, got %q", "session.policy_updated", EventSessionPolicyUpdate)
	}

	el := NewEventLogger(100)
	el.Emit(EventSessionPolicyUpdate, "s1", map[string]string{"mode": "preset", "duration": "1h"})

	events := el.Recent(1)
	if len(events) != 1 {
		t.Fatal("expected 1 event")
	}
	if events[0].Type != EventSessionPolicyUpdate {
		t.Errorf("expected type %q, got %q", EventSessionPolicyUpdate, events[0].Type)
	}
}

// [REQ:P1-004a] GET /api/v1/events returns recent events
func TestHandleEvents_ReturnsRecentEvents(t *testing.T) {
	events := NewEventLogger(100)
	events.Emit(EventSessionCreated, "s1", map[string]string{"shell": "/bin/bash"})
	events.Emit(EventSessionConnected, "s1", nil)
	events.Emit(EventSessionDeleted, "s1", nil)

	srv := &Server{events: events}
	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	w := httptest.NewRecorder()

	srv.handleEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp EventsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(resp.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(resp.Events))
	}
	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
}

// [REQ:P1-004a] GET /api/v1/events?limit=invalid falls back to default 50
func TestHandleEvents_InvalidLimitFallback(t *testing.T) {
	events := NewEventLogger(100)
	for i := 0; i < 5; i++ {
		events.Emit(EventSessionCreated, "s", nil)
	}

	srv := &Server{events: events}

	// Negative limit should be ignored, returning default (50, capped to actual count)
	req := httptest.NewRequest("GET", "/api/v1/events?limit=-1", nil)
	w := httptest.NewRecorder()
	srv.handleEvents(w, req)

	var resp EventsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	// Default 50, but only 5 events exist
	if len(resp.Events) != 5 {
		t.Fatalf("expected 5 events with invalid limit, got %d", len(resp.Events))
	}

	// Non-numeric limit should also fall back
	req2 := httptest.NewRequest("GET", "/api/v1/events?limit=abc", nil)
	w2 := httptest.NewRecorder()
	srv.handleEvents(w2, req2)

	var resp2 EventsResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(resp2.Events) != 5 {
		t.Fatalf("expected 5 events with non-numeric limit, got %d", len(resp2.Events))
	}
}

// [REQ:P1-004a] Subscriber receives emitted events in real time
func TestEventLogger_SubscriberNotification(t *testing.T) {
	el := NewEventLogger(100)

	ch := make(chan Event, 10)
	el.mu.Lock()
	el.subscribers = append(el.subscribers, ch)
	el.mu.Unlock()

	el.Emit(EventSessionCreated, "sub-test", map[string]string{"source": "test"})

	select {
	case evt := <-ch:
		if evt.Type != EventSessionCreated {
			t.Errorf("expected type %q, got %q", EventSessionCreated, evt.Type)
		}
		if evt.SessionID != "sub-test" {
			t.Errorf("expected session ID %q, got %q", "sub-test", evt.SessionID)
		}
		if evt.Details["source"] != "test" {
			t.Errorf("expected detail source=test, got %q", evt.Details["source"])
		}
	default:
		t.Error("subscriber channel should have received the event")
	}
}

// [REQ:P1-004a] GET /api/v1/events?limit=9999 is capped at 1000
func TestHandleEvents_LimitCappedAt1000(t *testing.T) {
	events := NewEventLogger(2000)
	for i := 0; i < 1500; i++ {
		events.Emit(EventSessionCreated, "s", nil)
	}

	srv := &Server{events: events}
	req := httptest.NewRequest("GET", "/api/v1/events?limit=9999", nil)
	w := httptest.NewRecorder()

	srv.handleEvents(w, req)

	var resp EventsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(resp.Events) != 1000 {
		t.Errorf("expected events capped at 1000, got %d", len(resp.Events))
	}
	if resp.Total != 1500 {
		t.Errorf("expected total=1500, got %d", resp.Total)
	}
}

// [REQ:P1-004a] GET /api/v1/events?limit=N respects limit
func TestHandleEvents_RespectsLimit(t *testing.T) {
	events := NewEventLogger(100)
	for i := 0; i < 10; i++ {
		events.Emit(EventSessionCreated, "s", nil)
	}

	srv := &Server{events: events}
	req := httptest.NewRequest("GET", "/api/v1/events?limit=3", nil)
	w := httptest.NewRecorder()

	srv.handleEvents(w, req)

	var resp EventsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(resp.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(resp.Events))
	}
	if resp.Total != 10 {
		t.Errorf("expected total=10, got %d", resp.Total)
	}
}
