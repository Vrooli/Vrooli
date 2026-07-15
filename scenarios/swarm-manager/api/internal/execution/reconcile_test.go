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
