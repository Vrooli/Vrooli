package checks

import (
	"context"
	"testing"
	"time"
)

// [REQ:AUTOHEAL-P0-012]
func TestHealInterlockRefusesCrossCheckDestructionInsideWindow(t *testing.T) {
	registry, clock, starter, destroyer := newHealInterlockFixture(t)

	started := registry.ExecuteAction(context.Background(), starter.ID(), "start")
	if !started.Success {
		t.Fatalf("start result = %#v, want success", started)
	}

	refused := registry.ExecuteAction(context.Background(), destroyer.ID(), "kill")
	if refused.Success || refused.Refusal == nil {
		t.Fatalf("destructive result = %#v, want typed refusal", refused)
	}
	if refused.Refusal.SourceCheckID != starter.ID() || refused.Refusal.AttemptingCheckID != destroyer.ID() {
		t.Fatalf("refusal check IDs = %q/%q, want %q/%q", refused.Refusal.SourceCheckID, refused.Refusal.AttemptingCheckID, starter.ID(), destroyer.ID())
	}
	if refused.Refusal.Target != (HealTarget{Kind: "resource", Name: "qdrant"}) {
		t.Fatalf("refusal target = %#v, want resource:qdrant", refused.Refusal.Target)
	}
	if got := len(destroyer.executedActions); got != 0 {
		t.Fatalf("destructive executions = %d, want 0", got)
	}
	_ = clock
}

// [REQ:AUTOHEAL-P0-012]
func TestHealInterlockAllowsDestructionOutsideWindow(t *testing.T) {
	registry, clock, starter, destroyer := newHealInterlockFixture(t)
	if result := registry.ExecuteAction(context.Background(), starter.ID(), "start"); !result.Success {
		t.Fatalf("start result = %#v, want success", result)
	}
	clock.current = clock.current.Add(DefaultHealInterlockWindow)

	result := registry.ExecuteAction(context.Background(), destroyer.ID(), "kill")
	if !result.Success || result.Refusal != nil {
		t.Fatalf("destructive result = %#v, want permitted success", result)
	}
	if got := len(destroyer.executedActions); got != 1 {
		t.Fatalf("destructive executions = %d, want 1", got)
	}
}

// [REQ:AUTOHEAL-P0-012]
func TestHealInterlockAllowsSameCheckToHealItself(t *testing.T) {
	registry := newTestRegistry()
	check := &mockHealableCheck{
		id: "scenario-vrooli-autoheal",
		actions: []RecoveryAction{
			{ID: "start", Available: true},
			{ID: "restart", Available: true, Dangerous: true},
		},
		executeResult: ActionResult{Success: true},
	}
	registry.Register(check)

	if result := registry.ExecuteAction(context.Background(), check.ID(), "start"); !result.Success {
		t.Fatalf("self start result = %#v, want success", result)
	}
	if result := registry.ExecuteAction(context.Background(), check.ID(), "restart"); !result.Success || result.Refusal != nil {
		t.Fatalf("self restart result = %#v, want permitted success", result)
	}
}

func newHealInterlockFixture(t *testing.T) (*Registry, *fixedClock, *mockHealableCheck, *mockHealableCheck) {
	t.Helper()
	registry := newTestRegistry()
	clock := &fixedClock{current: time.Date(2026, 8, 29, 1, 0, 0, 0, time.UTC)}
	registry.SetClock(clock)
	if err := registry.SetHealInterlockWindow(DefaultHealInterlockWindow); err != nil {
		t.Fatalf("SetHealInterlockWindow: %v", err)
	}
	starter := &mockHealableCheck{
		id:            "resource-qdrant",
		actions:       []RecoveryAction{{ID: "start", Available: true}},
		executeResult: ActionResult{Success: true},
	}
	destroyer := &mockHealableCheck{
		id:            "resource-qdrant-mode-drift",
		actions:       []RecoveryAction{{ID: "kill", Available: true, Dangerous: true}},
		executeResult: ActionResult{Success: true},
	}
	registry.Register(starter)
	registry.Register(destroyer)
	return registry, clock, starter, destroyer
}
