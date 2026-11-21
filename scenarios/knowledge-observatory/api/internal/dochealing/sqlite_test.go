package dochealing_test

import (
	"context"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/dbtest"
	dochealingdomain "knowledge-observatory/internal/dochealing"
	"knowledge-observatory/internal/services/dochealing"
)

func newStore(t *testing.T) *dochealingdomain.SQLite {
	t.Helper()
	return dochealingdomain.NewSQLite(dbtest.New(t, apidb.SchemaProviderFunc(dochealingdomain.Schema)))
}

func ptr(v float64) *float64 { return &v }

func TestJobLifecycleCoversEveryColumn(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	id, err := store.CreateJob(ctx, dochealing.HealRequest{
		ScenarioName: "vrooli-autoheal",
		AutoApprove:  true,
		DryRun:       true,
	}, ptr(0.42))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == "" {
		t.Fatal("create returned an empty id")
	}

	job, ok, err := store.GetJob(ctx, id)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if job.ScenarioName != "vrooli-autoheal" || job.Status != "pending" {
		t.Errorf("identity = %q/%q", job.ScenarioName, job.Status)
	}
	if !job.AutoApprove || !job.DryRun {
		t.Errorf("flags = %v/%v, want true/true", job.AutoApprove, job.DryRun)
	}
	if job.HealthBefore == nil || *job.HealthBefore != 0.42 {
		t.Errorf("health_before = %v, want 0.42", job.HealthBefore)
	}
	if job.HealthAfter != nil {
		t.Errorf("health_after = %v, want nil", job.HealthAfter)
	}

	started := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	if err := store.MarkRunning(ctx, id, "run-9", started); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if err := store.UpdateProgress(ctx, id, "drafting"); err != nil {
		t.Fatalf("progress: %v", err)
	}

	reviewed := started.Add(time.Minute)
	diff := &dochealing.DiffPreview{}
	if err := store.UpdateReview(ctx, id, diff, ptr(0.88), "needs_review", reviewed); err != nil {
		t.Fatalf("update review: %v", err)
	}

	job, _, err = store.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "needs_review" {
		t.Errorf("status = %q, want needs_review", job.Status)
	}
	if job.AgentRunID != "run-9" || job.Progress != "drafting" {
		t.Errorf("run state = %q/%q", job.AgentRunID, job.Progress)
	}
	if job.HealthAfter == nil || *job.HealthAfter != 0.88 {
		t.Errorf("health_after = %v, want 0.88", job.HealthAfter)
	}
	if job.StartedAt == nil || !job.StartedAt.Equal(started) {
		t.Errorf("started_at = %v, want %v", job.StartedAt, started)
	}
	if job.CompletedAt == nil || !job.CompletedAt.Equal(reviewed) {
		t.Errorf("completed_at = %v, want %v", job.CompletedAt, reviewed)
	}
}

func TestApprovalAndRejectionPaths(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	at := time.Date(2026, 7, 8, 9, 10, 11, 0, time.UTC)

	approved, err := store.CreateJob(ctx, dochealing.HealRequest{ScenarioName: "a"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkApproved(ctx, approved, "matt", at, ptr(0.95)); err != nil {
		t.Fatalf("approve: %v", err)
	}
	job, _, err := store.GetJob(ctx, approved)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "approved" {
		t.Errorf("status = %q, want approved", job.Status)
	}
	if job.HealthAfter == nil || *job.HealthAfter != 0.95 {
		t.Errorf("health_after = %v, want 0.95", job.HealthAfter)
	}

	rejected, err := store.CreateJob(ctx, dochealing.HealRequest{ScenarioName: "b"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkRejected(ctx, rejected, "matt", "changes the meaning", at); err != nil {
		t.Fatalf("reject: %v", err)
	}
	job, _, err = store.GetJob(ctx, rejected)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "rejected" {
		t.Errorf("status = %q, want rejected", job.Status)
	}
}

func TestFailAndErrorPaths(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	id, err := store.CreateJob(ctx, dochealing.HealRequest{ScenarioName: "a"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateError(ctx, id, "agent timed out"); err != nil {
		t.Fatalf("update error: %v", err)
	}
	job, _, err := store.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if job.Error != "agent timed out" {
		t.Errorf("error = %q", job.Error)
	}

	at := time.Date(2026, 8, 8, 8, 8, 8, 0, time.UTC)
	if err := store.FailJob(ctx, id, "gave up", at); err != nil {
		t.Fatalf("fail: %v", err)
	}
	job, _, err = store.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "failed" || job.Error != "gave up" {
		t.Errorf("failed state = %q/%q", job.Status, job.Error)
	}
	if job.CompletedAt == nil || !job.CompletedAt.Equal(at) {
		t.Errorf("completed_at = %v, want %v", job.CompletedAt, at)
	}
}

func TestInvalidStatusIsRejected(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	id, err := store.CreateJob(ctx, dochealing.HealRequest{ScenarioName: "a"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = store.UpdateReview(ctx, id, nil, nil, "half_done", time.Now().UTC())
	if err == nil {
		t.Fatal("expected the status CHECK constraint to reject 'half_done'")
	}
}
