package deepsearch_test

import (
	"context"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/dbtest"
	deepsearchdomain "knowledge-observatory/internal/deepsearch"
	"knowledge-observatory/internal/services/deepsearch"
)

func newStore(t *testing.T) *deepsearchdomain.SQLite {
	t.Helper()
	return deepsearchdomain.NewSQLite(dbtest.New(t, apidb.SchemaProviderFunc(deepsearchdomain.Schema)))
}

func TestJobLifecycleCoversEveryColumn(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	id, err := store.CreateJob(ctx, deepsearch.DeepSearchRequest{
		Query:      "where is the rdp check",
		Scope:      "scenario",
		Scenario:   "vrooli-autoheal",
		BasePath:   "api/internal/checks",
		MaxResults: 25,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == "" {
		t.Fatal("create returned an empty id; SQLite has no gen_random_uuid() so the id must be minted in Go")
	}

	job, ok, err := store.GetJob(ctx, id)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if job.Status != "pending" {
		t.Errorf("status = %q, want pending", job.Status)
	}
	if job.MaxResults != 25 {
		t.Errorf("max_results = %d, want 25", job.MaxResults)
	}
	if len(job.Results) != 0 {
		t.Errorf("results = %v, want empty on a pending job", job.Results)
	}
	if job.StartedAt != nil || job.CompletedAt != nil {
		t.Error("timestamps should be nil on a pending job")
	}

	started := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	if err := store.MarkRunning(ctx, id, "run-77", started); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if err := store.UpdateProgress(ctx, id, "scanning 12 files"); err != nil {
		t.Fatalf("progress: %v", err)
	}

	job, _, err = store.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "running" || job.AgentRunID != "run-77" {
		t.Errorf("running state = %q/%q", job.Status, job.AgentRunID)
	}
	if job.Progress != "scanning 12 files" {
		t.Errorf("progress = %q", job.Progress)
	}
	if job.StartedAt == nil || !job.StartedAt.Equal(started) {
		t.Errorf("started_at = %v, want %v", job.StartedAt, started)
	}

	completed := started.Add(time.Minute)
	results := []deepsearch.DeepSearchResult{
		{Path: "api/internal/checks/infra/rdp.go", Snippet: "func (c *RDPCheck) Run"},
	}
	if err := store.CompleteJob(ctx, id, results, completed); err != nil {
		t.Fatalf("complete: %v", err)
	}

	job, _, err = store.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "completed" {
		t.Errorf("status = %q, want completed", job.Status)
	}
	// results was JSONB on Postgres and is TEXT here; the round trip must be lossless.
	if len(job.Results) != 1 || job.Results[0].Path != results[0].Path {
		t.Errorf("results = %+v, want %+v", job.Results, results)
	}
	if job.CompletedAt == nil || !job.CompletedAt.Equal(completed) {
		t.Errorf("completed_at = %v, want %v", job.CompletedAt, completed)
	}
	if job.Error != "" {
		t.Errorf("error = %q, want cleared on completion", job.Error)
	}
}

func TestFailJobRecordsTheMessage(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	id, err := store.CreateJob(ctx, deepsearch.DeepSearchRequest{Query: "q", Scope: "global"})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 5, 5, 5, 5, 5, 0, time.UTC)
	if err := store.FailJob(ctx, id, "agent unavailable", at); err != nil {
		t.Fatalf("fail: %v", err)
	}

	job, _, err := store.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "failed" || job.Error != "agent unavailable" {
		t.Errorf("failed state = %q/%q", job.Status, job.Error)
	}
	if job.CompletedAt == nil || !job.CompletedAt.Equal(at) {
		t.Errorf("completed_at = %v, want %v", job.CompletedAt, at)
	}
}

func TestInvalidScopeIsRejected(t *testing.T) {
	store := newStore(t)
	if _, err := store.CreateJob(context.Background(), deepsearch.DeepSearchRequest{
		Query: "q", Scope: "universe",
	}); err == nil {
		t.Fatal("expected the scope CHECK constraint to reject 'universe'")
	}
}

func TestGetMissingJob(t *testing.T) {
	store := newStore(t)
	if _, ok, err := store.GetJob(context.Background(), "no-such-job"); err != nil || ok {
		t.Fatalf("missing job: ok=%v err=%v", ok, err)
	}
}
