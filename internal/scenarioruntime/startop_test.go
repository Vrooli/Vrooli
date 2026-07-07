package scenarioruntime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStartOperationLifecycleAndSupersede(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	pid := 4242
	first, err := store.BeginStartOperation(ctx, StartOperation{
		Scenario:     "alpha",
		Operation:    "start",
		InitiatorPID: &pid,
	})
	if err != nil {
		t.Fatalf("BeginStartOperation() error = %v", err)
	}
	if first.OperationID == "" || first.Status != StartOperationStatusRunning {
		t.Fatalf("first = %+v, want running with generated id", first)
	}
	if first.Variant != DefaultVariant {
		t.Fatalf("variant = %q, want %q", first.Variant, DefaultVariant)
	}

	// Update: current step + steps json round-trip.
	first.CurrentStep = "develop"
	first.DependencyCurrent = "beta"
	first.DependencyIndex = 2
	first.DependencyTotal = 3
	ended := clk.Now()
	first.WithSteps([]StartOperationStep{
		{Name: "setup", Status: StartStepDone, StartedAt: clk.Now(), EndedAt: &ended},
		{Name: "develop", Status: StartStepRunning, StartedAt: clk.Now()},
	})
	if _, err := store.UpdateStartOperation(ctx, first); err != nil {
		t.Fatalf("UpdateStartOperation() error = %v", err)
	}
	got, err := store.GetLatestStartOperation(ctx, "alpha", "")
	if err != nil {
		t.Fatalf("GetLatestStartOperation() error = %v", err)
	}
	if got.CurrentStep != "develop" || got.DependencyCurrent != "beta" || got.DependencyIndex != 2 || got.DependencyTotal != 3 {
		t.Fatalf("got = %+v, want develop step with dependency 2/3", got)
	}
	steps := got.Steps()
	if len(steps) != 2 || steps[0].Name != "setup" || steps[0].Status != StartStepDone || steps[1].Status != StartStepRunning {
		t.Fatalf("steps = %+v, want [setup done, develop running]", steps)
	}
	if got.InitiatorPID == nil || *got.InitiatorPID != pid {
		t.Fatalf("initiator pid = %v, want %d", got.InitiatorPID, pid)
	}

	// A second begin supersedes the still-running first record.
	clk.Advance(time.Minute)
	second, err := store.BeginStartOperation(ctx, StartOperation{Scenario: "alpha", Operation: "restart"})
	if err != nil {
		t.Fatalf("BeginStartOperation(second) error = %v", err)
	}
	latest, err := store.GetLatestStartOperation(ctx, "alpha", "live")
	if err != nil {
		t.Fatalf("GetLatestStartOperation(after second) error = %v", err)
	}
	if latest.OperationID != second.OperationID {
		t.Fatalf("latest = %s, want the new operation %s", latest.OperationID, second.OperationID)
	}
	old, err := store.getStartOperation(ctx, first.OperationID)
	if err != nil {
		t.Fatalf("getStartOperation(first) error = %v", err)
	}
	if old.Status != StartOperationStatusAbandoned || old.FinishedAt == nil {
		t.Fatalf("superseded record = %+v, want abandoned with finished_at", old)
	}
}

func TestStartOperationAbandonAndTerminalPruning(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	op, err := store.BeginStartOperation(ctx, StartOperation{Scenario: "alpha"})
	if err != nil {
		t.Fatalf("BeginStartOperation() error = %v", err)
	}
	abandoned, err := store.MarkStartOperationAbandoned(ctx, op.OperationID, "initiator interrupted")
	if err != nil {
		t.Fatalf("MarkStartOperationAbandoned() error = %v", err)
	}
	if abandoned.Status != StartOperationStatusAbandoned || abandoned.Error != "initiator interrupted" {
		t.Fatalf("abandoned = %+v", abandoned)
	}
	if !abandoned.IsTerminal() {
		t.Fatal("abandoned must be terminal")
	}
	// Abandoning a non-running record is a no-op error.
	if _, err := store.MarkStartOperationAbandoned(ctx, op.OperationID, "again"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second abandon error = %v, want ErrNotFound", err)
	}

	// Terminal history is pruned to StartOperationKeepTerminal.
	for i := 0; i < StartOperationKeepTerminal+3; i++ {
		clk.Advance(time.Minute)
		next, err := store.BeginStartOperation(ctx, StartOperation{Scenario: "alpha"})
		if err != nil {
			t.Fatalf("BeginStartOperation(%d) error = %v", i, err)
		}
		next.Status = StartOperationStatusSucceeded
		now := clk.Now()
		next.FinishedAt = &now
		if _, err := store.UpdateStartOperation(ctx, next); err != nil {
			t.Fatalf("UpdateStartOperation(%d) error = %v", i, err)
		}
	}
	clk.Advance(time.Minute)
	if _, err := store.BeginStartOperation(ctx, StartOperation{Scenario: "alpha"}); err != nil {
		t.Fatalf("BeginStartOperation(final) error = %v", err)
	}
	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM runtime_start_operations WHERE scenario = 'alpha' AND status != ?`, StartOperationStatusRunning).Scan(&count); err != nil {
		t.Fatalf("count terminal: %v", err)
	}
	if count > StartOperationKeepTerminal {
		t.Fatalf("terminal records = %d, want ≤ %d", count, StartOperationKeepTerminal)
	}
}

// TestUpdateStartOperationDoesNotResurrectTerminalRecord proves terminal
// records are immutable: a flush racing the signal-handler abandon (or a
// takeover's supersede) gets ErrNotFound instead of rewriting the record
// back to running.
func TestUpdateStartOperationDoesNotResurrectTerminalRecord(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	op, err := store.BeginStartOperation(ctx, StartOperation{Scenario: "alpha"})
	if err != nil {
		t.Fatalf("BeginStartOperation() error = %v", err)
	}
	if _, err := store.MarkStartOperationAbandoned(ctx, op.OperationID, "start interrupted (signal)"); err != nil {
		t.Fatalf("MarkStartOperationAbandoned() error = %v", err)
	}

	// The orchestration goroutine's late flush still carries status=running.
	op.Status = StartOperationStatusRunning
	op.CurrentStep = "develop"
	if _, err := store.UpdateStartOperation(ctx, op); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateStartOperation(after abandon) error = %v, want ErrNotFound", err)
	}
	got, err := store.GetLatestStartOperation(ctx, "alpha", "")
	if err != nil {
		t.Fatalf("GetLatestStartOperation() error = %v", err)
	}
	if got.Status != StartOperationStatusAbandoned || got.Error != "start interrupted (signal)" {
		t.Fatalf("got = %+v, want abandoned with signal reason preserved", got)
	}
	if got.CurrentStep == "develop" {
		t.Fatal("late flush must not modify a terminal record")
	}
}

func TestPhaseDurationEstimatesSmoothedAndPruned(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	// No history → empty map, never a fabricated number.
	estimates, err := store.PhaseDurationEstimates(ctx, "alpha", "")
	if err != nil {
		t.Fatalf("PhaseDurationEstimates() error = %v", err)
	}
	if len(estimates) != 0 {
		t.Fatalf("estimates = %v, want empty with no history", estimates)
	}

	for _, d := range []time.Duration{2 * time.Second, 4 * time.Second} {
		clk.Advance(time.Second)
		if err := store.RecordPhaseDuration(ctx, "alpha", "", "setup", d); err != nil {
			t.Fatalf("RecordPhaseDuration() error = %v", err)
		}
	}
	estimates, err = store.PhaseDurationEstimates(ctx, "alpha", "live")
	if err != nil {
		t.Fatalf("PhaseDurationEstimates() error = %v", err)
	}
	if got := estimates["setup"]; got != 3*time.Second {
		t.Fatalf("setup estimate = %v, want 3s mean", got)
	}
	if _, ok := estimates["develop"]; ok {
		t.Fatal("develop must be absent without history")
	}

	// History is pruned to the last PhaseDurationKeep entries.
	for i := 0; i < PhaseDurationKeep+5; i++ {
		clk.Advance(time.Second)
		if err := store.RecordPhaseDuration(ctx, "alpha", "", "develop", time.Duration(i+1)*time.Second); err != nil {
			t.Fatalf("RecordPhaseDuration(develop %d) error = %v", i, err)
		}
	}
	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM runtime_phase_durations WHERE scenario = 'alpha' AND phase = 'develop'`).Scan(&count); err != nil {
		t.Fatalf("count durations: %v", err)
	}
	if count != PhaseDurationKeep {
		t.Fatalf("duration rows = %d, want %d", count, PhaseDurationKeep)
	}

	if err := store.RecordPhaseDuration(ctx, "", "", "setup", time.Second); err == nil {
		t.Fatal("expected error for missing scenario")
	}
	if err := store.RecordPhaseDuration(ctx, "alpha", "", "setup", -time.Second); err == nil {
		t.Fatal("expected error for negative duration")
	}
}

func TestGetLatestStartOperationNotFound(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newFixedClock(time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)))
	if _, err := store.GetLatestStartOperation(ctx, "ghost", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
	if _, err := store.BeginStartOperation(ctx, StartOperation{}); err == nil {
		t.Fatal("expected error for missing scenario")
	}
	if _, err := store.UpdateStartOperation(ctx, StartOperation{OperationID: "missing"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update missing error = %v, want ErrNotFound", err)
	}
}
