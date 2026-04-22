package execution

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// capturingEventLogger records every emit call so tests can assert on the
// event stream without wiring a real event-log repo. It is goroutine-safe.
type capturingEventLogger struct {
	mu               sync.Mutex
	created          []string
	statusChanges    []statusChange
	completed        []completedEmit
	failed           []failedEmit
	canceled         []string
	manuallyAccepted []manuallyAcceptedEmit
	viewed           []string
}

type statusChange struct {
	ExecID, From, To string
}

type completedEmit struct {
	ExecID          string
	DurationSeconds float64
	HadFixups       bool
}

type failedEmit struct {
	ExecID, Reason  string
	DurationSeconds float64
}

type manuallyAcceptedEmit struct {
	ExecID, AcceptedBy, Reason, PreviousStatus string
}

func (l *capturingEventLogger) EmitExecutionCreated(execID, _, _, _ string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.created = append(l.created, execID)
}

func (l *capturingEventLogger) EmitExecutionStatusChanged(execID, from, to string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.statusChanges = append(l.statusChanges, statusChange{ExecID: execID, From: from, To: to})
}

func (l *capturingEventLogger) EmitExecutionCompleted(execID string, dur float64, hadFixups bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.completed = append(l.completed, completedEmit{ExecID: execID, DurationSeconds: dur, HadFixups: hadFixups})
}

func (l *capturingEventLogger) EmitExecutionFailed(execID, reason string, dur float64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failed = append(l.failed, failedEmit{ExecID: execID, Reason: reason, DurationSeconds: dur})
}

func (l *capturingEventLogger) EmitExecutionCanceled(execID, _ string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.canceled = append(l.canceled, execID)
}

func (l *capturingEventLogger) EmitExecutionManuallyAccepted(execID, acceptor, reason, prev string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.manuallyAccepted = append(l.manuallyAccepted, manuallyAcceptedEmit{
		ExecID: execID, AcceptedBy: acceptor, Reason: reason, PreviousStatus: prev,
	})
}

func (l *capturingEventLogger) EmitExecutionViewed(execID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.viewed = append(l.viewed, execID)
}

// newServiceForStatsEventTests constructs an execution.Service with a
// disk-backed store and a capturing event logger. Tests then seed the store
// directly and exercise one transition at a time.
func newServiceForStatsEventTests(t *testing.T) (*Service, *capturingEventLogger) {
	t.Helper()
	root := t.TempDir()
	storePath := filepath.Join(root, "execution-runs.json")
	svc := NewService(ServiceConfig{
		RootDir:   root,
		StorePath: storePath,
	})
	logger := &capturingEventLogger{}
	svc.SetEventLogger(logger)
	return svc, logger
}

// seedRecord writes a single execution record to the service's store with
// the given initial status and finalization state.
func seedRecord(t *testing.T, svc *Service, execID string, status Status, withFinalization bool) {
	t.Helper()
	rec := Record{
		ExecutionID: execID,
		BacklogKind: "idea",
		BacklogName: "item-" + execID,
		Status:      status,
		Mode:        ModeYOLO,
		StartedAt:   time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
		CreatedAt:   time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if withFinalization {
		rec.Finalization = &Finalization{
			Eligible:                true,
			Status:                  FinalizationStatusRunning,
			Phase:                   FinalizationPhaseScopeDetection,
			StartedAt:               time.Now().UTC().Format(time.RFC3339),
			Scenarios:               []ScenarioFinalization{},
			AffectedScenarios:       []string{},
			Warnings:                []FinalizationWarning{},
			AggregateClassification: "",
			AggregateSummary:        "",
		}
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatalf("seed record: %v", err)
	}
}

// TestCompleteFinalizationSkippedEmitsCompleted verifies Phase 1: the
// completeFinalizationSkipped path in finalization_store.go now calls
// logExecutionEvent, so a terminal execution.completed event reaches the
// event log. Before Phase 1, this site emitted only a dispatch update.
func TestCompleteFinalizationSkippedEmitsCompleted(t *testing.T) {
	svc, logger := newServiceForStatsEventTests(t)
	seedRecord(t, svc, "exec-skip-1", StatusValidating, true)

	if err := svc.completeFinalizationSkipped("exec-skip-1", "no changed files"); err != nil {
		t.Fatalf("completeFinalizationSkipped: %v", err)
	}

	if len(logger.completed) != 1 || logger.completed[0].ExecID != "exec-skip-1" {
		t.Fatalf("expected 1 completed emit for exec-skip-1, got %+v", logger.completed)
	}
	if len(logger.statusChanges) != 1 || logger.statusChanges[0].To != string(StatusCompleted) {
		t.Fatalf("expected status change ending in completed, got %+v", logger.statusChanges)
	}
}

// TestFailFinalizationEmitsFailed is the Phase 1 regression guard for
// finalization_store.failFinalization — it emitted only a dispatch update
// before the fix, losing the failure event.
func TestFailFinalizationEmitsFailed(t *testing.T) {
	svc, logger := newServiceForStatsEventTests(t)
	seedRecord(t, svc, "exec-fail-1", StatusValidating, true)

	if err := svc.failFinalization("exec-fail-1", "svc-a", "health check timed out"); err != nil {
		t.Fatalf("failFinalization: %v", err)
	}

	if len(logger.failed) != 1 || logger.failed[0].ExecID != "exec-fail-1" {
		t.Fatalf("expected 1 failed emit for exec-fail-1, got %+v", logger.failed)
	}
	if logger.failed[0].Reason != "health check timed out" {
		t.Fatalf("expected failure reason to reach emit, got %q", logger.failed[0].Reason)
	}
}

// TestMarkFinalizationPhaseEmitsStatusChanged verifies the validating
// transition now produces a status-changed event on the log.
func TestMarkFinalizationPhaseEmitsStatusChanged(t *testing.T) {
	svc, logger := newServiceForStatsEventTests(t)
	seedRecord(t, svc, "exec-running-1", StatusRunning, true)

	if err := svc.markFinalizationPhase("exec-running-1", string(FinalizationPhaseScopeDetection)); err != nil {
		t.Fatalf("markFinalizationPhase: %v", err)
	}

	if len(logger.statusChanges) != 1 || logger.statusChanges[0].To != string(StatusValidating) {
		t.Fatalf("expected one status change to validating, got %+v", logger.statusChanges)
	}
}

// TestManuallyAcceptLatestFlipsFailedToCompleted verifies Phase 2. A failed
// execution is flipped to Completed, and BOTH execution.manually_accepted
// and execution.completed are emitted so stats can count the run as success
// while surfacing the manual-accept subset.
func TestManuallyAcceptLatestFlipsFailedToCompleted(t *testing.T) {
	svc, logger := newServiceForStatsEventTests(t)

	// Seed two executions for the same backlog item: an older completed run
	// and a more recent failed run. Only the recent one should be accepted.
	now := time.Now().UTC()
	recs := []Record{
		{
			ExecutionID: "exec-older", BacklogKind: "idea", BacklogName: "item-1",
			Status: StatusCompleted, Mode: ModeYOLO,
			StartedAt:  now.Add(-2 * time.Hour).Format(time.RFC3339),
			FinishedAt: now.Add(-90 * time.Minute).Format(time.RFC3339),
			CreatedAt:  now.Add(-2 * time.Hour).Format(time.RFC3339),
			UpdatedAt:  now.Add(-90 * time.Minute).Format(time.RFC3339),
		},
		{
			ExecutionID: "exec-newer", BacklogKind: "idea", BacklogName: "item-1",
			Status: StatusFailed, Mode: ModeYOLO,
			StartedAt:     now.Add(-30 * time.Minute).Format(time.RFC3339),
			FinishedAt:    now.Add(-15 * time.Minute).Format(time.RFC3339),
			FailureReason: "agent self-reported failure",
			CreatedAt:     now.Add(-30 * time.Minute).Format(time.RFC3339),
			UpdatedAt:     now.Add(-15 * time.Minute).Format(time.RFC3339),
		},
	}
	if err := svc.store.Save(recs); err != nil {
		t.Fatalf("seed: %v", err)
	}

	execID, accepted, err := svc.ManuallyAcceptLatestForBacklog(context.Background(), "idea", "item-1", "user", "good enough")
	if err != nil {
		t.Fatalf("ManuallyAcceptLatestForBacklog: %v", err)
	}
	if !accepted || execID != "exec-newer" {
		t.Fatalf("expected exec-newer accepted, got %q accepted=%v", execID, accepted)
	}
	if len(logger.manuallyAccepted) != 1 || logger.manuallyAccepted[0].ExecID != "exec-newer" {
		t.Fatalf("expected 1 manually_accepted emit for exec-newer, got %+v", logger.manuallyAccepted)
	}
	if logger.manuallyAccepted[0].PreviousStatus != string(StatusFailed) {
		t.Fatalf("expected previous_status=failed, got %q", logger.manuallyAccepted[0].PreviousStatus)
	}
	if len(logger.completed) != 1 || logger.completed[0].ExecID != "exec-newer" {
		t.Fatalf("expected completed emit for exec-newer, got %+v", logger.completed)
	}
}

// TestManuallyAcceptNoopWhenNoEligible confirms the accept is silent when
// there is no failed/needs_fixup execution for the backlog item — e.g. a
// research item that never ran an agent at all.
func TestManuallyAcceptNoopWhenNoEligible(t *testing.T) {
	svc, logger := newServiceForStatsEventTests(t)

	// Only a completed execution exists — nothing to flip.
	seedRecord(t, svc, "exec-only-ok", StatusCompleted, false)

	_, accepted, err := svc.ManuallyAcceptLatestForBacklog(context.Background(), "idea", "item-exec-only-ok", "user", "")
	if err != nil {
		t.Fatalf("ManuallyAcceptLatestForBacklog: %v", err)
	}
	if accepted {
		t.Fatalf("expected accepted=false, got true")
	}
	if len(logger.manuallyAccepted) != 0 || len(logger.completed) != 0 {
		t.Fatalf("expected no emits, got ma=%+v completed=%+v", logger.manuallyAccepted, logger.completed)
	}
}

// TestBackfillEmitsMissingCompletedForValidatingRecords is the Phase 6
// regression guard: records stuck at validating with a completed
// finalization must be promoted and emitted as completed. Records that
// already have a terminal event in the log must be skipped.
func TestBackfillEmitsMissingCompletedForValidatingRecords(t *testing.T) {
	svc, logger := newServiceForStatsEventTests(t)

	now := time.Now().UTC()
	recs := []Record{
		{
			ExecutionID: "stuck-1", BacklogKind: "idea", BacklogName: "item-stuck-1",
			Status: StatusValidating, Mode: ModeYOLO,
			StartedAt: now.Add(-30 * time.Minute).Format(time.RFC3339),
			CreatedAt: now.Add(-30 * time.Minute).Format(time.RFC3339),
			UpdatedAt: now.Add(-25 * time.Minute).Format(time.RFC3339),
			Finalization: &Finalization{
				Eligible: true, Status: FinalizationStatusCompleted,
				Phase: FinalizationPhaseCompleted, StartedAt: now.Add(-25 * time.Minute).Format(time.RFC3339),
				CompletedAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
			},
		},
		{
			ExecutionID: "already-complete", BacklogKind: "idea", BacklogName: "item-already-complete",
			Status: StatusCompleted, Mode: ModeYOLO,
			StartedAt:  now.Add(-20 * time.Minute).Format(time.RFC3339),
			FinishedAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
			CreatedAt:  now.Add(-20 * time.Minute).Format(time.RFC3339),
			UpdatedAt:  now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
		{
			ExecutionID: "failed-unemitted", BacklogKind: "idea", BacklogName: "item-failed-unemitted",
			Status: StatusFailed, Mode: ModeYOLO,
			StartedAt:     now.Add(-15 * time.Minute).Format(time.RFC3339),
			FinishedAt:    now.Add(-10 * time.Minute).Format(time.RFC3339),
			FailureReason: "agent died",
			CreatedAt:     now.Add(-15 * time.Minute).Format(time.RFC3339),
			UpdatedAt:     now.Add(-10 * time.Minute).Format(time.RFC3339),
		},
	}
	if err := svc.store.Save(recs); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Pretend "already-complete" already has its completed event in the log.
	alreadyEmitted := map[string]struct{}{"already-complete": {}}
	report, err := svc.BackfillStuckTerminalEvents(context.Background(), alreadyEmitted)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}

	if report.StuckValidating != 1 {
		t.Errorf("expected 1 stuck-validating, got %d", report.StuckValidating)
	}
	if report.MissingTerminal != 2 {
		t.Errorf("expected 2 missing-terminal emits, got %d", report.MissingTerminal)
	}
	if report.Affected != 2 {
		t.Errorf("expected 2 affected records, got %d", report.Affected)
	}

	// Should have emitted: 1 status_changed for stuck-1, 1 completed for
	// stuck-1, 1 failed for failed-unemitted. NOT a second completed for
	// already-complete.
	if len(logger.completed) != 1 || logger.completed[0].ExecID != "stuck-1" {
		t.Errorf("expected 1 completed for stuck-1, got %+v", logger.completed)
	}
	if len(logger.failed) != 1 || logger.failed[0].ExecID != "failed-unemitted" {
		t.Errorf("expected 1 failed for failed-unemitted, got %+v", logger.failed)
	}

	// Re-load records — stuck-1 should now be Completed on disk.
	loaded, err := svc.store.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, r := range loaded {
		if r.ExecutionID == "stuck-1" && r.Status != StatusCompleted {
			t.Errorf("stuck-1 should be promoted to completed, got %q", r.Status)
		}
	}
}

// TestBackfillIdempotentWhenAllEventsAlreadyEmitted ensures no emits happen
// when every terminal record has an existing event.
func TestBackfillIdempotentWhenAllEventsAlreadyEmitted(t *testing.T) {
	svc, logger := newServiceForStatsEventTests(t)

	now := time.Now().UTC()
	recs := []Record{
		{
			ExecutionID: "exec-done-1", BacklogKind: "idea", BacklogName: "item-1",
			Status: StatusCompleted, Mode: ModeYOLO,
			StartedAt:  now.Add(-20 * time.Minute).Format(time.RFC3339),
			FinishedAt: now.Add(-10 * time.Minute).Format(time.RFC3339),
			CreatedAt:  now.Add(-20 * time.Minute).Format(time.RFC3339),
			UpdatedAt:  now.Add(-10 * time.Minute).Format(time.RFC3339),
		},
		{
			ExecutionID: "exec-done-2", BacklogKind: "idea", BacklogName: "item-2",
			Status: StatusFailed, Mode: ModeYOLO,
			StartedAt:  now.Add(-15 * time.Minute).Format(time.RFC3339),
			FinishedAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
			CreatedAt:  now.Add(-15 * time.Minute).Format(time.RFC3339),
			UpdatedAt:  now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
	}
	if err := svc.store.Save(recs); err != nil {
		t.Fatalf("seed: %v", err)
	}

	alreadyEmitted := map[string]struct{}{"exec-done-1": {}, "exec-done-2": {}}
	report, err := svc.BackfillStuckTerminalEvents(context.Background(), alreadyEmitted)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}

	if report.Affected != 0 {
		t.Errorf("expected 0 affected, got %d", report.Affected)
	}
	if len(logger.completed) != 0 || len(logger.failed) != 0 {
		t.Errorf("expected no emits, got completed=%+v failed=%+v", logger.completed, logger.failed)
	}
}
