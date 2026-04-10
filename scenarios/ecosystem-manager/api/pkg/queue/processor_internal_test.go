package queue

import (
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
)

func TestReserveExecutionCreatesEntry(t *testing.T) {
	qp := &Processor{registry: NewExecutionRegistry()}

	startedAt := time.Unix(1_700_000_000, 0)
	qp.reserveExecution("task-1", "ecosystem-task-1", startedAt)

	execState, ok := qp.registry.GetExecution("task-1")

	if !ok {
		t.Fatalf("expected reservation to exist")
	}
	if execState.agentTag != "ecosystem-task-1" {
		t.Fatalf("unexpected agent tag: %s", execState.agentTag)
	}
	if !execState.started.Equal(startedAt) {
		t.Fatalf("expected start time %s, got %s", startedAt, execState.started)
	}
}

func TestRegisterRunIDUpgradesReservation(t *testing.T) {
	qp := &Processor{registry: NewExecutionRegistry()}

	startedAt := time.Unix(1_700_000_000, 0)
	qp.reserveExecution("task-2", "ecosystem-task-2", startedAt)

	// Modern approach: register runID instead of cmd
	qp.registry.RegisterRunID("task-2", "run-12345")

	execState, ok := qp.registry.GetExecution("task-2")

	if !ok {
		t.Fatalf("expected execution to exist")
	}
	if execState.runID != "run-12345" {
		t.Fatalf("expected runID 'run-12345', got %s", execState.runID)
	}
	if !execState.started.Equal(startedAt) {
		t.Fatalf("expected original start time to be preserved")
	}
}

func TestUnregisterExecutionRemovesEntry(t *testing.T) {
	qp := &Processor{registry: NewExecutionRegistry()}
	qp.reserveExecution("task-3", "ecosystem-task-3", time.Now())

	qp.unregisterExecution("task-3")

	_, ok := qp.registry.GetExecution("task-3")

	if ok {
		t.Fatalf("expected execution to be removed")
	}
}

func TestComputeSlotSnapshot(t *testing.T) {
	qp := &Processor{}
	internal := map[string]struct{}{
		"a": {}, "b": {},
	}
	external := map[string]struct{}{
		"b": {}, // overlap with internal
		"c": {},
	}

	t.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	// default settings slots fallback is 1; ensure higher by setting env via settings helper
	prev := settings.GetSettings()
	updated := prev
	updated.Slots = 3
	settings.UpdateSettings(updated)
	t.Cleanup(func() { settings.UpdateSettings(prev) })

	snap := qp.computeSlotSnapshot(internal, external)
	if snap.Running != 3 { // a,b,c (b counted once)
		t.Fatalf("expected running=3, got %d", snap.Running)
	}
	if snap.Slots != 3 {
		t.Fatalf("expected slots=3, got %d", snap.Slots)
	}
	if snap.Available != 0 {
		t.Fatalf("expected available=0, got %d", snap.Available)
	}
}
