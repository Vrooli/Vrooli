package opsrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentops"
)

// TestReconcileOrphanSnapshots reaps aged, unreferenced execution snapshots while
// preserving referenced snapshots and skipping orphans younger than the grace
// period (which a concurrent in-flight Invoke may still be committing).
func TestReconcileOrphanSnapshots(t *testing.T) {
	root := t.TempDir()
	loc := memLocator{root: root}
	repo := NewWorkflowRepo(loc)

	kind, id := agentops.TargetPlanExecution, "plan-handle"
	w, err := repo.CreateOrLoad(kind, id)
	if err != nil {
		t.Fatal(err)
	}
	next := cloneWorkflow(w)
	next.State = agentops.WorkflowRunning
	next.Operations = append(next.Operations, agentops.OperationExecutionRecord{
		Operation: "execution-run", ExecutionID: "keep-1", IdempotencyKey: "k1",
		ProvenanceDigest: "sha256:" + strings.Repeat("a", 64), State: "running", RunID: "run-1",
	})
	next.IdempotencyKeys = []string{"k1"}
	next.Version = 1
	if err := repo.Commit(0, next); err != nil {
		t.Fatalf("commit workflow: %v", err)
	}

	dir, err := loc.AgentOpsDir(kind, id)
	if err != nil {
		t.Fatal(err)
	}
	execDir := filepath.Join(dir, executionsSubdir)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	writeSnap := func(name string, recordedAt time.Time) {
		body := []byte(`{"recorded_at":"` + recordedAt.UTC().Format(time.RFC3339Nano) + `"}`)
		if err := os.WriteFile(filepath.Join(execDir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeSnap("keep-1.json", now)                          // referenced -> always kept
	writeSnap("orphan-old.json", now.Add(-time.Hour))      // unreferenced + aged -> reaped
	writeSnap("orphan-recent.json", now.Add(-time.Minute)) // unreferenced but too new -> skipped

	report, err := ReconcileOrphanSnapshots(loc, now, 10*time.Minute)
	if err != nil {
		t.Fatalf("ReconcileOrphanSnapshots: %v", err)
	}
	if report.SnapshotsSeen != 3 {
		t.Fatalf("expected 3 snapshots seen, got %d", report.SnapshotsSeen)
	}
	if len(report.Reaped) != 1 || filepath.Base(report.Reaped[0]) != "orphan-old.json" {
		t.Fatalf("expected only orphan-old.json reaped, got %v", report.Reaped)
	}
	if report.SkippedTooRecent != 1 {
		t.Fatalf("expected 1 skipped-too-recent, got %d", report.SkippedTooRecent)
	}
	if _, err := os.Stat(filepath.Join(execDir, "keep-1.json")); err != nil {
		t.Fatalf("referenced snapshot must survive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(execDir, "orphan-old.json")); !os.IsNotExist(err) {
		t.Fatalf("aged orphan must be removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(execDir, "orphan-recent.json")); err != nil {
		t.Fatalf("recent orphan must be preserved: %v", err)
	}
}

// TestReconcileOrphanSnapshotsSkipsCorruptWorkflow leaves every snapshot in place
// when the workflow.json cannot be decoded (conservative: never risk deleting a
// referenced snapshot on a corruption signal).
func TestReconcileOrphanSnapshotsSkipsCorruptWorkflow(t *testing.T) {
	root := t.TempDir()
	loc := memLocator{root: root}
	dir := filepath.Join(root, string(agentops.TargetPlanExecution), "h", agentOpsSubdir)
	execDir := filepath.Join(dir, executionsSubdir)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, workflowFile), []byte("{ not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(execDir, "orphan.json"), []byte(`{"recorded_at":"`+time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ReconcileOrphanSnapshots(loc, time.Now(), time.Minute)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(report.Reaped) != 0 {
		t.Fatalf("corrupt-workflow dir must reap nothing, got %v", report.Reaped)
	}
	if _, err := os.Stat(filepath.Join(execDir, "orphan.json")); err != nil {
		t.Fatalf("snapshot must be preserved under corrupt workflow: %v", err)
	}
}
