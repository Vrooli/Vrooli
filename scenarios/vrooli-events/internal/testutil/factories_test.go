package testutil

import (
	"testing"
)

func TestMakeEvent(t *testing.T) {
	ev := MakeEvent("evt-1", "test.created", "src-scenario")
	if ev.EventID != "evt-1" {
		t.Errorf("EventID = %q, want %q", ev.EventID, "evt-1")
	}
	if ev.EventType != "test.created" {
		t.Errorf("EventType = %q, want %q", ev.EventType, "test.created")
	}
	if ev.SourceScenario != "src-scenario" {
		t.Errorf("SourceScenario = %q, want %q", ev.SourceScenario, "src-scenario")
	}
	if ev.TargetScenario != "target-1" {
		t.Errorf("TargetScenario = %q, want default %q", ev.TargetScenario, "target-1")
	}
	if ev.CorrelationID != "corr-1" {
		t.Errorf("CorrelationID = %q, want default %q", ev.CorrelationID, "corr-1")
	}
}

func TestNewTestStore(t *testing.T) {
	s := NewTestStore(t)
	if s == nil {
		t.Fatal("NewTestStore returned nil")
	}
}

func TestNewTestPolicyStore(t *testing.T) {
	ps := NewTestPolicyStore(t)
	if ps == nil {
		t.Fatal("NewTestPolicyStore returned nil")
	}
}

func TestNewTestSubscriptionStore(t *testing.T) {
	ss := NewTestSubscriptionStore(t)
	if ss == nil {
		t.Fatal("NewTestSubscriptionStore returned nil")
	}
}
