package opsrunner

import (
	"context"
	"testing"
	"time"

	"swarm-manager/internal/agentops"
)

// TestSchedulerFiresIntentAfterRestart proves scheduled intents are durable: an
// intent persisted before a "restart" (a fresh scheduler over the same store)
// still fires, and firing consumes it exactly once.
func TestSchedulerFiresIntentAfterRestart(t *testing.T) {
	root := t.TempDir()
	repo := NewWorkflowRepo(memLocator{root: root})
	kind, id := agentops.TargetInitiative, "init-a"

	// Persist an intent with a scheduler instance, then discard it.
	pre := NewScheduler(repo, nil, nil)
	if err := pre.ScheduleIntent(kind, id, agentops.ScheduledIntent{
		Intent: "auto-advance", Action: agentops.ActionOpenReview,
	}); err != nil {
		t.Fatalf("schedule: %v", err)
	}

	// A fresh scheduler over the same store (a restart) reloads and fires it.
	var fired int
	fresh := NewScheduler(NewWorkflowRepo(memLocator{root: root}), func(context.Context, agentops.WorkflowInstance, agentops.ScheduledIntent) error {
		fired++
		return nil
	}, nil)
	if err := fresh.Tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}
	if fired != 1 {
		t.Fatalf("intent fired %d times, want 1", fired)
	}
	// A second tick does not re-fire (the intent was consumed).
	if err := fresh.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fired != 1 {
		t.Fatalf("intent re-fired after consumption: %d", fired)
	}
	w, _, _ := repo.Load(kind, id)
	if len(w.Timers) != 0 {
		t.Fatalf("consumed intent still present: %+v", w.Timers)
	}
}

// TestSchedulerDeduplicatesIntent proves scheduling the same intent name twice is
// a no-op.
func TestSchedulerDeduplicatesIntent(t *testing.T) {
	repo := NewWorkflowRepo(memLocator{root: t.TempDir()})
	kind, id := agentops.TargetInitiative, "init-a"
	s := NewScheduler(repo, nil, nil)
	intent := agentops.ScheduledIntent{Intent: "auto-advance", Action: agentops.ActionOpenReview}
	if err := s.ScheduleIntent(kind, id, intent); err != nil {
		t.Fatal(err)
	}
	if err := s.ScheduleIntent(kind, id, intent); err != nil {
		t.Fatal(err)
	}
	w, _, _ := repo.Load(kind, id)
	if len(w.Timers) != 1 {
		t.Fatalf("intent scheduled %d times, want dedup to 1", len(w.Timers))
	}
}

// TestSchedulerNotBeforeGate proves a future intent does not fire early and does
// fire once its time passes.
func TestSchedulerNotBeforeGate(t *testing.T) {
	repo := NewWorkflowRepo(memLocator{root: t.TempDir()})
	kind, id := agentops.TargetInitiative, "init-a"
	future := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := now
	s := NewScheduler(repo, func(context.Context, agentops.WorkflowInstance, agentops.ScheduledIntent) error {
		return nil
	}, func() time.Time { return clock })
	if err := s.ScheduleIntent(kind, id, agentops.ScheduledIntent{
		Intent: "later", Action: agentops.ActionOpenReview, NotBefore: future.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	w, _, _ := repo.Load(kind, id)
	if len(w.Timers) != 1 {
		t.Fatalf("future intent fired early")
	}
	// Advance the clock past not_before.
	clock = future.Add(time.Hour)
	if err := s.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	w, _, _ = repo.Load(kind, id)
	if len(w.Timers) != 0 {
		t.Fatalf("due intent did not fire")
	}
}

// TestSchedulerCancelIntent proves a pending intent can be canceled.
func TestSchedulerCancelIntent(t *testing.T) {
	repo := NewWorkflowRepo(memLocator{root: t.TempDir()})
	kind, id := agentops.TargetInitiative, "init-a"
	s := NewScheduler(repo, nil, nil)
	if err := s.ScheduleIntent(kind, id, agentops.ScheduledIntent{Intent: "x", Action: agentops.ActionOpenReview}); err != nil {
		t.Fatal(err)
	}
	if err := s.CancelIntent(kind, id, "x"); err != nil {
		t.Fatal(err)
	}
	w, _, _ := repo.Load(kind, id)
	if len(w.Timers) != 0 {
		t.Fatalf("canceled intent still present")
	}
}
