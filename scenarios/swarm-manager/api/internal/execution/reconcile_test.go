package execution

import (
	"path/filepath"
	"testing"

	"swarm-manager/internal/promptmanager"
)

// TestReconcileStrandedRecords sweeps run-id-less inspectable records to failed
// while leaving healthy records (with a run id, or already terminal) untouched.
func TestReconcileStrandedRecords(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "chore", "stranded-item", map[string]any{
		"name":     "stranded-item",
		"title":    "Stranded",
		"status":   "queued",
		"priority": 3,
		"tags":     []string{},
	})

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})

	seed := []Record{
		// Stranded: starting with no run id (the 542467 class).
		{ExecutionID: "stranded-1", BacklogKind: "chore", BacklogName: "stranded-item", Status: StatusStarting, Mode: ModeManual, PreviousStatus: "ready"},
		// Healthy running record with a run id — untouched.
		{ExecutionID: "healthy-1", BacklogKind: "chore", BacklogName: "stranded-item", Status: StatusRunning, RunID: "run-xyz", Mode: ModeManual},
		// Already terminal — untouched.
		{ExecutionID: "done-1", BacklogKind: "chore", BacklogName: "stranded-item", Status: StatusCompleted, RunID: "run-old", Mode: ModeManual},
		// Pending with no run id is NOT stranded (it has not started; the drain owns it).
		{ExecutionID: "pending-1", BacklogKind: "chore", BacklogName: "stranded-item", Status: StatusPending, Mode: ModeManual},
	}
	if err := service.store.Save(seed); err != nil {
		t.Fatalf("save seed: %v", err)
	}

	report, err := service.ReconcileStrandedRecords()
	if err != nil {
		t.Fatalf("ReconcileStrandedRecords error: %v", err)
	}
	if len(report.Stranded) != 1 || report.Stranded[0] != "stranded-1" {
		t.Fatalf("expected exactly stranded-1 swept, got %v", report.Stranded)
	}

	after, err := service.store.Load()
	if err != nil {
		t.Fatalf("load after reconcile: %v", err)
	}
	byID := map[string]Record{}
	for _, r := range after {
		byID[r.ExecutionID] = r
	}
	if byID["stranded-1"].Status != StatusFailed {
		t.Fatalf("stranded-1 should be failed, got %q", byID["stranded-1"].Status)
	}
	if byID["stranded-1"].FailureReason == "" {
		t.Fatalf("stranded-1 should carry a failure reason")
	}
	if byID["healthy-1"].Status != StatusRunning {
		t.Fatalf("healthy-1 (has run id) must be untouched, got %q", byID["healthy-1"].Status)
	}
	if byID["done-1"].Status != StatusCompleted {
		t.Fatalf("done-1 must be untouched, got %q", byID["done-1"].Status)
	}
	if byID["pending-1"].Status != StatusPending {
		t.Fatalf("pending-1 must be untouched, got %q", byID["pending-1"].Status)
	}

	// Idempotent: a second sweep finds nothing.
	report2, err := service.ReconcileStrandedRecords()
	if err != nil {
		t.Fatalf("second reconcile error: %v", err)
	}
	if len(report2.Stranded) != 0 {
		t.Fatalf("second sweep must find nothing, got %v", report2.Stranded)
	}
}

// TestReconcileStrandedRecordsReapsMissedOperations proves the sweep delivers a
// missed operation reap: a terminal (failed/canceled) record still correlated to
// a durable operation execution gets its operation offered a cancel through the
// OperationStarter seam (the 542467c6 divergence class — the stranded sweep
// marked the legacy record failed without reaping its agentops operation).
// Completed records and records without an operation correlation are untouched.
func TestReconcileStrandedRecordsReapsMissedOperations(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "chore", "diverged-item", map[string]any{
		"name":     "diverged-item",
		"title":    "Diverged",
		"status":   "backlog",
		"priority": 3,
		"tags":     []string{},
	})

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	starter := &stubOperationStarter{}
	service.SetOperationStarter(starter)

	seed := []Record{
		// Failed record with an op correlation whose reap was missed — must be reaped.
		{
			ExecutionID: "diverged-1", BacklogKind: "chore", BacklogName: "diverged-item", Status: StatusFailed,
			Mode: ModeManual, OpWorkflowID: "wf-plan-execution-x", OpExecutionID: "opx-stale",
		},
		// Completed record with an op correlation — must NOT be reaped (a running
		// op behind a completed record is a divergence for operator attention).
		{
			ExecutionID: "done-1", BacklogKind: "chore", BacklogName: "diverged-item", Status: StatusCompleted,
			Mode: ModeManual, OpWorkflowID: "wf-plan-execution-x", OpExecutionID: "opx-done",
		},
		// Failed record without any op correlation — nothing to reap.
		{ExecutionID: "legacy-1", BacklogKind: "chore", BacklogName: "diverged-item", Status: StatusFailed, Mode: ModeManual},
	}
	if err := service.store.Save(seed); err != nil {
		t.Fatalf("save seed: %v", err)
	}

	report, err := service.ReconcileStrandedRecords()
	if err != nil {
		t.Fatalf("ReconcileStrandedRecords error: %v", err)
	}
	if len(report.Stranded) != 0 {
		t.Fatalf("no stranded records expected, got %v", report.Stranded)
	}
	if len(report.OpReapsAttempted) != 1 || report.OpReapsAttempted[0] != "diverged-1" {
		t.Fatalf("expected exactly diverged-1 offered a reap, got %v", report.OpReapsAttempted)
	}
	if starter.cancelCalls != 1 {
		t.Fatalf("expected exactly one CancelOperation call, got %d", starter.cancelCalls)
	}
	if starter.cancelReq.ExecutionID != "opx-stale" {
		t.Fatalf("expected reap of opx-stale, got %q", starter.cancelReq.ExecutionID)
	}
	if starter.cancelReq.TargetKind != "plan-execution" || starter.cancelReq.TargetID != "test-plan-diverged-item" {
		t.Fatalf("unexpected reap target %s/%s", starter.cancelReq.TargetKind, starter.cancelReq.TargetID)
	}
}
