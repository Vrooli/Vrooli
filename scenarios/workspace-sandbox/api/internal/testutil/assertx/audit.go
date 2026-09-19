package assertx

import (
	"reflect"
	"testing"

	"workspace-sandbox/internal/types"
)

// ExpectedEvent is one expected entry in an audit-event sequence.
// EventType is matched exactly. Actor and ActorType match exactly when
// non-empty. Details — when non-nil — must be a subset of the actual
// event's Details map (extra keys in the actual map are allowed).
type ExpectedEvent struct {
	EventType string
	Actor     string
	ActorType string
	Details   map[string]interface{}
}

// AssertAuditEvents verifies that `actual` contains the expected
// events in order. Extra trailing actual events fail the assertion.
//
// Uses a partial-match policy on Details so callers don't have to
// repeat unrelated detail keys (e.g., the audit emitter may inject
// timestamps the test doesn't care about).
func AssertAuditEvents(t *testing.T, actual []*types.AuditEvent, want []ExpectedEvent) {
	t.Helper()
	if len(actual) != len(want) {
		t.Errorf("AssertAuditEvents: got %d events, want %d", len(actual), len(want))
		dumpEvents(t, actual)
		return
	}
	for i, exp := range want {
		got := actual[i]
		if got == nil {
			t.Errorf("event[%d]: nil event", i)
			continue
		}
		if got.EventType != exp.EventType {
			t.Errorf("event[%d]: type = %q, want %q", i, got.EventType, exp.EventType)
		}
		if exp.Actor != "" && got.Actor != exp.Actor {
			t.Errorf("event[%d]: actor = %q, want %q", i, got.Actor, exp.Actor)
		}
		if exp.ActorType != "" && got.ActorType != exp.ActorType {
			t.Errorf("event[%d]: actorType = %q, want %q", i, got.ActorType, exp.ActorType)
		}
		for k, v := range exp.Details {
			actualV, ok := got.Details[k]
			if !ok {
				t.Errorf("event[%d]: details missing key %q", i, k)
				continue
			}
			if !reflect.DeepEqual(actualV, v) {
				t.Errorf("event[%d]: details[%q] = %v, want %v", i, k, actualV, v)
			}
		}
	}
}

// AssertEventCount checks that the number of events of the given
// EventType equals `want`. Useful when order doesn't matter.
func AssertEventCount(t *testing.T, actual []*types.AuditEvent, eventType string, want int) {
	t.Helper()
	got := 0
	for _, e := range actual {
		if e != nil && e.EventType == eventType {
			got++
		}
	}
	if got != want {
		t.Errorf("AssertEventCount[%s]: got %d, want %d", eventType, got, want)
	}
}

func dumpEvents(t *testing.T, events []*types.AuditEvent) {
	t.Helper()
	for i, e := range events {
		if e == nil {
			t.Logf("  event[%d] = nil", i)
			continue
		}
		t.Logf("  event[%d] = type=%s actor=%s actorType=%s details=%v", i, e.EventType, e.Actor, e.ActorType, e.Details)
	}
}
