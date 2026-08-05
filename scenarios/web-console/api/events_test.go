package main

import (
	"context"
	"testing"
	"time"

	"web-console/internal/events"

	"connectrpc.com/connect"

	eventsH "web-console/handlers/events"

	eventsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events"
)

// callEventsList invokes the EventsService.List Connect RPC directly
// against the supplied event logger. Lives here in the test file so we
// keep production code clean — the handler is the public boundary, the
// test is just an adapter.
func callEventsList(el *events.Logger, limit int32) (*eventsv1.ListResponse, error) {
	h := eventsH.NewConnectHandler(eventsH.Deps{Service: testEventsService{el: el}})
	resp, err := h.List(context.Background(), connect.NewRequest(&eventsv1.ListRequest{Limit: limit}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

type testEventsService struct{ el *events.Logger }

func (s testEventsService) Recent(_ context.Context, limit int) []eventsH.Event {
	in := s.el.Recent(limit)
	out := make([]eventsH.Event, len(in))
	for i, e := range in {
		out[i] = eventsH.Event{
			Type:      e.Type,
			SessionID: e.SessionID,
			Timestamp: e.Timestamp.UTC().Format(time.RFC3339Nano),
			Details:   e.Details,
		}
	}
	return out
}

func (s testEventsService) Count(_ context.Context) int { return s.el.Count() }

// [REQ:P1-004a] Structured Event Logging - event logger tests

func TestEventLogger_EmitAndRecent(t *testing.T) {
	el := events.NewLogger(100)

	el.Emit(events.SessionCreated, "sess-1", map[string]string{"shell": "/bin/bash"})
	el.Emit(events.SessionConnected, "sess-1", nil)
	el.Emit(events.SessionDisconnected, "sess-1", nil)

	evts := el.Recent(10)
	if len(evts) != 3 {
		t.Fatalf("expected 3 events, got %d", len(evts))
	}

	if evts[0].Type != events.SessionCreated {
		t.Errorf("expected first event type %q, got %q", events.SessionCreated, evts[0].Type)
	}
	if evts[0].SessionID != "sess-1" {
		t.Errorf("expected session ID %q, got %q", "sess-1", evts[0].SessionID)
	}
	if evts[0].Details["shell"] != "/bin/bash" {
		t.Errorf("expected detail shell=/bin/bash, got %q", evts[0].Details["shell"])
	}
	if evts[1].Type != events.SessionConnected {
		t.Errorf("expected second event type %q, got %q", events.SessionConnected, evts[1].Type)
	}
	if evts[2].Type != events.SessionDisconnected {
		t.Errorf("expected third event type %q, got %q", events.SessionDisconnected, evts[2].Type)
	}
}

func TestEventLogger_RecentLimitsOutput(t *testing.T) {
	el := events.NewLogger(100)

	for i := 0; i < 5; i++ {
		el.Emit(events.SessionCreated, "sess", nil)
	}

	evts := el.Recent(3)
	if len(evts) != 3 {
		t.Fatalf("expected 3 events, got %d", len(evts))
	}
}

func TestEventLogger_BoundedHistory(t *testing.T) {
	el := events.NewLogger(3) // Small buffer

	for i := 0; i < 10; i++ {
		el.Emit(events.SessionCreated, "sess", nil)
	}

	evts := el.Recent(0)
	if len(evts) != 3 {
		t.Fatalf("expected history capped at 3, got %d", len(evts))
	}
}

func TestEventLogger_Count(t *testing.T) {
	el := events.NewLogger(100)

	if el.Count() != 0 {
		t.Fatalf("expected 0 events initially, got %d", el.Count())
	}

	el.Emit(events.SessionCreated, "s1", nil)
	el.Emit(events.SessionDeleted, "s1", nil)

	if el.Count() != 2 {
		t.Fatalf("expected 2 events, got %d", el.Count())
	}
}

func TestEventLogger_AllEventTypes(t *testing.T) {
	el := events.NewLogger(100)

	types := []string{
		events.SessionCreated,
		events.SessionConnected,
		events.SessionDisconnected,
		events.SessionTerminated,
		events.SessionDeleted,
		events.PaneResized,
	}
	for _, et := range types {
		el.Emit(et, "sess", nil)
	}

	evts := el.Recent(0)
	if len(evts) != len(types) {
		t.Fatalf("expected %d events, got %d", len(types), len(evts))
	}
	for i, evt := range evts {
		if evt.Type != types[i] {
			t.Errorf("event[%d]: expected type %q, got %q", i, types[i], evt.Type)
		}
		if evt.Timestamp.IsZero() {
			t.Errorf("event[%d]: timestamp should not be zero", i)
		}
	}
}

func TestEventLogger_NilDetails(t *testing.T) {
	el := events.NewLogger(100)
	el.Emit(events.SessionCreated, "s1", nil)

	evts := el.Recent(1)
	if len(evts) != 1 {
		t.Fatal("expected 1 event")
	}
	if evts[0].Details != nil {
		t.Error("expected nil details for event emitted with nil details")
	}
}

// [REQ:P1-004a] events.SessionPolicyUpdate constant is used instead of inline string
func TestEventLogger_PolicyUpdateConstant(t *testing.T) {
	if events.SessionPolicyUpdate != "session.policy_updated" {
		t.Errorf("expected %q, got %q", "session.policy_updated", events.SessionPolicyUpdate)
	}

	el := events.NewLogger(100)
	el.Emit(events.SessionPolicyUpdate, "s1", map[string]string{"mode": "preset", "duration": "1h"})

	evts := el.Recent(1)
	if len(evts) != 1 {
		t.Fatal("expected 1 event")
	}
	if evts[0].Type != events.SessionPolicyUpdate {
		t.Errorf("expected type %q, got %q", events.SessionPolicyUpdate, evts[0].Type)
	}
}

// [REQ:P1-004a] EventsService.List returns recent events
func TestEventsService_List_ReturnsRecentEvents(t *testing.T) {
	el := events.NewLogger(100)
	el.Emit(events.SessionCreated, "s1", map[string]string{"shell": "/bin/bash"})
	el.Emit(events.SessionConnected, "s1", nil)
	el.Emit(events.SessionDeleted, "s1", nil)

	resp, err := callEventsList(el, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(resp.Events))
	}
	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
}

// [REQ:P1-004a] EventsService.List with limit<=0 falls back to default 50
func TestEventsService_List_DefaultLimit(t *testing.T) {
	el := events.NewLogger(100)
	for i := 0; i < 5; i++ {
		el.Emit(events.SessionCreated, "s", nil)
	}

	resp, err := callEventsList(el, -1)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Events) != 5 {
		t.Fatalf("expected 5 events with negative limit, got %d", len(resp.Events))
	}

	resp2, err := callEventsList(el, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp2.Events) != 5 {
		t.Fatalf("expected 5 events with zero limit, got %d", len(resp2.Events))
	}
}

// [REQ:P1-004a] Subscriber receives emitted events in real time
func TestEventLogger_SubscriberNotification(t *testing.T) {
	el := events.NewLogger(100)

	ch := make(chan events.Event, 10)
	unsub := el.Subscribe(ch)
	defer unsub()

	el.Emit(events.SessionCreated, "sub-test", map[string]string{"source": "test"})

	select {
	case evt := <-ch:
		if evt.Type != events.SessionCreated {
			t.Errorf("expected type %q, got %q", events.SessionCreated, evt.Type)
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

// [REQ:P1-004a] EventsService.List caps limit at 1000
func TestEventsService_List_LimitCappedAt1000(t *testing.T) {
	el := events.NewLogger(2000)
	for i := 0; i < 1500; i++ {
		el.Emit(events.SessionCreated, "s", nil)
	}

	resp, err := callEventsList(el, 9999)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Events) != 1000 {
		t.Errorf("expected events capped at 1000, got %d", len(resp.Events))
	}
	if resp.Total != 1500 {
		t.Errorf("expected total=1500, got %d", resp.Total)
	}
}

// [REQ:P1-004a] EventsService.List respects caller-supplied limit
func TestEventsService_List_RespectsLimit(t *testing.T) {
	el := events.NewLogger(100)
	for i := 0; i < 10; i++ {
		el.Emit(events.SessionCreated, "s", nil)
	}

	resp, err := callEventsList(el, 3)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(resp.Events))
	}
	if resp.Total != 10 {
		t.Errorf("expected total=10, got %d", resp.Total)
	}
}
