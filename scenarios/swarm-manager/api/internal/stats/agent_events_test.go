package stats

import (
	"context"
	"testing"
	"time"

	"swarm-manager/internal/eventlog"
)

// TestManualAcceptDoesNotDoubleCount is the key regression guard for the
// Agent tab: a run that went failed → manually_accepted → completed must
// count as ONE successful run (not one success + one failure). Before the
// execOutcome-map refactor, the stats engine tracked independent
// increment counters which double-counted the terminal sequence.
func TestManualAcceptDoesNotDoubleCount(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	base := time.Now().UTC().Add(-time.Hour)

	// Exec A: agent-succeeded, no manual accept.
	appendEvent(t, repo, base, eventlog.EntityExecution, "exec-a",
		eventlog.EventExecutionCreated, eventlog.ExecutionCreatedPayload{BacklogKind: "idea", BacklogName: "a", Mode: "yolo"})
	appendEvent(t, repo, base.Add(5*time.Minute), eventlog.EntityExecution, "exec-a",
		eventlog.EventExecutionCompleted, eventlog.ExecutionCompletedPayload{DurationSeconds: 300})

	// Exec B: agent-failed, then manually accepted. Sequence matches what
	// ManuallyAcceptLatestForBacklog produces at runtime.
	appendEvent(t, repo, base.Add(10*time.Minute), eventlog.EntityExecution, "exec-b",
		eventlog.EventExecutionCreated, eventlog.ExecutionCreatedPayload{BacklogKind: "idea", BacklogName: "b", Mode: "yolo"})
	appendEvent(t, repo, base.Add(15*time.Minute), eventlog.EntityExecution, "exec-b",
		eventlog.EventExecutionFailed, eventlog.ExecutionFailedPayload{Reason: "agent flaked", DurationSeconds: 300})
	// User intervention arrives later:
	appendEvent(t, repo, base.Add(20*time.Minute), eventlog.EntityExecution, "exec-b",
		eventlog.EventExecutionManuallyAccepted, eventlog.ExecutionManuallyAcceptedPayload{
			AcceptedBy: "user", PreviousExecStatus: "failed",
		})
	appendEvent(t, repo, base.Add(20*time.Minute), eventlog.EntityExecution, "exec-b",
		eventlog.EventExecutionCompleted, eventlog.ExecutionCompletedPayload{DurationSeconds: 300})

	// Exec C: agent-failed, no accept.
	appendEvent(t, repo, base.Add(25*time.Minute), eventlog.EntityExecution, "exec-c",
		eventlog.EventExecutionCreated, eventlog.ExecutionCreatedPayload{BacklogKind: "idea", BacklogName: "c", Mode: "yolo"})
	appendEvent(t, repo, base.Add(30*time.Minute), eventlog.EntityExecution, "exec-c",
		eventlog.EventExecutionFailed, eventlog.ExecutionFailedPayload{Reason: "agent flaked", DurationSeconds: 600})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	stats := engine.GetStats()

	// Accounting: 2 completed (a + manually-accepted b), 1 failed (c), 3 total.
	if got := stats.Agent.TotalExecutions; got != 3 {
		t.Errorf("TotalExecutions: got %d, want 3", got)
	}
	if got := stats.Agent.CompletedCount; got != 2 {
		t.Errorf("CompletedCount: got %d, want 2 (includes manually-accepted)", got)
	}
	if got := stats.Agent.FailedCount; got != 1 {
		t.Errorf("FailedCount: got %d, want 1", got)
	}
	if got := stats.Agent.ManuallyAcceptedCount; got != 1 {
		t.Errorf("ManuallyAcceptedCount: got %d, want 1", got)
	}

	// Success rate is 2/3, not 3/4 — the manually-accepted run must not be
	// double-counted as both success and failure.
	wantSuccess := 2.0 / 3.0
	if diff := stats.Agent.SuccessRate - wantSuccess; diff > 1e-6 || diff < -1e-6 {
		t.Errorf("SuccessRate: got %.4f, want %.4f", stats.Agent.SuccessRate, wantSuccess)
	}

	// ManualAcceptRate: 1 manually-accepted out of 3 finished = 1/3.
	wantManual := 1.0 / 3.0
	if diff := stats.Agent.ManualAcceptRate - wantManual; diff > 1e-6 || diff < -1e-6 {
		t.Errorf("ManualAcceptRate: got %.4f, want %.4f", stats.Agent.ManualAcceptRate, wantManual)
	}

	// SuccessRateSampleSize is finished = completed + failed = 3.
	if got := stats.Agent.SuccessRateSampleSize; got != 3 {
		t.Errorf("SuccessRateSampleSize: got %d, want 3", got)
	}
}

func TestAgentStatsSeparateAbstainedAndBudgetExhausted(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	base := time.Now().UTC().Add(-time.Hour)

	for i, reason := range []string{"abstained", "budget_exhausted", "agent flaked"} {
		id := "exec-outcome-" + string(rune('a'+i))
		at := base.Add(time.Duration(i) * time.Minute)
		appendEvent(t, repo, at, eventlog.EntityExecution, id,
			eventlog.EventExecutionCreated, eventlog.ExecutionCreatedPayload{BacklogKind: "idea", BacklogName: id, Mode: "yolo"})
		appendEvent(t, repo, at.Add(time.Minute), eventlog.EntityExecution, id,
			eventlog.EventExecutionFailed, eventlog.ExecutionFailedPayload{Reason: reason, DurationSeconds: 60})
	}

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	stats := engine.GetStats().Agent
	if stats.AbstainedCount != 1 || stats.BudgetExhaustedCount != 1 || stats.FailedCount != 1 {
		t.Fatalf("outcome counts = abstained:%d budget:%d failed:%d", stats.AbstainedCount, stats.BudgetExhaustedCount, stats.FailedCount)
	}
	if stats.SuccessRateSampleSize != 3 {
		t.Fatalf("sample size = %d, want 3", stats.SuccessRateSampleSize)
	}
}

// TestHistoryWindowReportsEarliestEvent verifies Phase 5: the top-level
// response now reports the span of history observed so the UI can decide
// whether a "30d" aggregate is misleading against a shorter real history.
func TestHistoryWindowReportsEarliestEvent(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	// Seed an event 3 days ago — that becomes the earliest.
	threeDaysAgo := time.Now().UTC().Add(-72 * time.Hour)
	appendEvent(t, repo, threeDaysAgo, eventlog.EntityExecution, "exec-early",
		eventlog.EventExecutionCreated, eventlog.ExecutionCreatedPayload{BacklogKind: "idea", BacklogName: "x", Mode: "yolo"})
	appendEvent(t, repo, time.Now().UTC(), eventlog.EntityExecution, "exec-recent",
		eventlog.EventExecutionCreated, eventlog.ExecutionCreatedPayload{BacklogKind: "idea", BacklogName: "y", Mode: "yolo"})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	stats := engine.GetStats()

	if !stats.History.HasHistory {
		t.Fatalf("expected HasHistory=true")
	}
	if stats.History.HistoryDays < 2.9 || stats.History.HistoryDays > 3.1 {
		t.Errorf("HistoryDays: got %.2f, want ~3", stats.History.HistoryDays)
	}
	if stats.History.EarliestEventAt.IsZero() {
		t.Errorf("EarliestEventAt should be populated")
	}
	if stats.History.MinSampleMeaningful != MinSampleMeaningful {
		t.Errorf("MinSampleMeaningful: got %d, want %d", stats.History.MinSampleMeaningful, MinSampleMeaningful)
	}
}

// TestTimingHasExecutionDurationInsteadOfCycleTime verifies Phase 4: the
// removed cycle-time metric has been replaced by execution-duration derived
// from the same samples the Agent tab already uses.
func TestTimingHasExecutionDurationInsteadOfCycleTime(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	base := time.Now().UTC().Add(-time.Hour)

	// Seed two completed executions with known durations: 60s and 180s.
	appendEvent(t, repo, base, eventlog.EntityExecution, "exec-a",
		eventlog.EventExecutionCreated, eventlog.ExecutionCreatedPayload{BacklogKind: "idea", BacklogName: "a", Mode: "yolo"})
	appendEvent(t, repo, base.Add(1*time.Minute), eventlog.EntityExecution, "exec-a",
		eventlog.EventExecutionCompleted, eventlog.ExecutionCompletedPayload{DurationSeconds: 60})
	appendEvent(t, repo, base.Add(2*time.Minute), eventlog.EntityExecution, "exec-b",
		eventlog.EventExecutionCreated, eventlog.ExecutionCreatedPayload{BacklogKind: "idea", BacklogName: "b", Mode: "yolo"})
	appendEvent(t, repo, base.Add(5*time.Minute), eventlog.EntityExecution, "exec-b",
		eventlog.EventExecutionCompleted, eventlog.ExecutionCompletedPayload{DurationSeconds: 180})

	if err := engine.Rebuild(ctx); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	stats := engine.GetStats()

	// Avg (60+180)/2 seconds = 120s = 2 minutes.
	if stats.Timing.AvgExecutionMinutes != 2.0 {
		t.Errorf("AvgExecutionMinutes: got %.2f, want 2.0", stats.Timing.AvgExecutionMinutes)
	}
	// Median of [1, 3] = 2 minutes.
	if stats.Timing.MedianExecutionMinutes != 2.0 {
		t.Errorf("MedianExecutionMinutes: got %.2f, want 2.0", stats.Timing.MedianExecutionMinutes)
	}
	if stats.Timing.ExecutionDurationSamples != 2 {
		t.Errorf("ExecutionDurationSamples: got %d, want 2", stats.Timing.ExecutionDurationSamples)
	}
}
