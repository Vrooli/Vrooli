package ingest_test

import (
	"context"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/dbtest"
	"knowledge-observatory/internal/ingest"
)

func newRepo(t *testing.T) *ingest.SQLite {
	t.Helper()
	return ingest.NewSQLite(dbtest.New(t, apidb.SchemaProviderFunc(ingest.Schema)))
}

func TestHistoryRoundTripCoversEveryColumn(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	in := ingest.HistoryEntry{
		RecordID:       "rec-1",
		Namespace:      "vrooli",
		CollectionName: "vrooli_knowledge",
		ContentHash:    "hash-1",
		Visibility:     "shared",
		Source:         "docs/README.md",
		SourceType:     "markdown",
		Status:         "failure",
		ErrorMessage:   "embedding timed out",
		TookMS:         2431,
	}
	id, err := repo.InsertHistory(ctx, in)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, ok, err := repo.GetHistory(ctx, id)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.RecordID != in.RecordID || got.Namespace != in.Namespace || got.CollectionName != in.CollectionName {
		t.Errorf("identity = %+v", got)
	}
	if got.ContentHash != in.ContentHash || got.Visibility != in.Visibility {
		t.Errorf("content = %q/%q", got.ContentHash, got.Visibility)
	}
	if got.Source != in.Source || got.SourceType != in.SourceType {
		t.Errorf("source = %q/%q", got.Source, got.SourceType)
	}
	if got.Status != in.Status || got.ErrorMessage != in.ErrorMessage || got.TookMS != in.TookMS {
		t.Errorf("outcome = %q/%q/%d", got.Status, got.ErrorMessage, got.TookMS)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at was not defaulted")
	}
}

func TestConstrainedColumnsAreEnforced(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	if _, err := repo.InsertHistory(ctx, ingest.HistoryEntry{
		RecordID: "r", Namespace: "n", CollectionName: "c",
		Visibility: "public", Status: "success", // "public" is not a valid visibility
	}); err == nil {
		t.Error("expected the visibility CHECK constraint to reject 'public'")
	}
	if _, err := repo.InsertHistory(ctx, ingest.HistoryEntry{
		RecordID: "r", Namespace: "n", CollectionName: "c",
		Visibility: "shared", Status: "pending", // not a valid terminal status
	}); err == nil {
		t.Error("expected the status CHECK constraint to reject 'pending'")
	}
}

func TestProvenanceForCollection(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	for _, e := range []ingest.HistoryEntry{
		{RecordID: "1", Namespace: "a", CollectionName: "c", Visibility: "shared", Status: "success"},
		{RecordID: "2", Namespace: "a", CollectionName: "c", Visibility: "shared", Status: "success"},
		{RecordID: "3", Namespace: "b", CollectionName: "c", Visibility: "shared", Status: "failure"},
		{RecordID: "4", Namespace: "z", CollectionName: "other", Visibility: "shared", Status: "success"},
	} {
		if _, err := repo.InsertHistory(ctx, e); err != nil {
			t.Fatal(err)
		}
	}

	got, err := repo.ProvenanceForCollection(ctx, "c")
	if err != nil {
		t.Fatalf("provenance: %v", err)
	}
	if got.IngestAttempts != 3 {
		t.Errorf("ingest_attempts = %d, want 3", got.IngestAttempts)
	}
	if got.DistinctNamespaces != 2 {
		t.Errorf("distinct_namespaces = %d, want 2", got.DistinctNamespaces)
	}
	if got.LastIngestAt == nil {
		t.Error("last_ingest_at was not returned")
	}

	// An unknown collection must answer zero, not error.
	empty, err := repo.ProvenanceForCollection(ctx, "nope")
	if err != nil {
		t.Fatalf("unknown collection: %v", err)
	}
	if empty.IngestAttempts != 0 || empty.LastIngestAt != nil {
		t.Errorf("unknown collection = %+v, want zero", empty)
	}
}

// TestHealthForCollection covers the aggregate FILTER clauses and the 24-hour
// window, which Postgres wrote as NOW() - INTERVAL '24 hours'.
func TestHealthForCollection(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	for _, status := range []string{"success", "success", "failure", "failure", "failure"} {
		if _, err := repo.InsertHistory(ctx, ingest.HistoryEntry{
			RecordID: "r", Namespace: "n", CollectionName: "c",
			Visibility: "shared", Status: status,
		}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := repo.HealthForCollection(ctx, "c")
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if got.TotalAttempts != 5 || got.SuccessCount != 2 || got.FailureCount != 3 {
		t.Errorf("tallies = %d/%d/%d, want 5/2/3",
			got.TotalAttempts, got.SuccessCount, got.FailureCount)
	}
	// Every row was just written, so all failures fall inside the window.
	if got.FailureCountLast24H != 3 {
		t.Errorf("failures in last 24h = %d, want 3", got.FailureCountLast24H)
	}
	if got.LastFailureAt == nil {
		t.Fatal("last_failure_at was not returned")
	}
	if time.Since(*got.LastFailureAt) > time.Hour {
		t.Errorf("last_failure_at = %v, want a moment ago", got.LastFailureAt)
	}
}

func TestDeleteHistoryByCollection(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	for _, c := range []string{"a", "a", "b"} {
		if _, err := repo.InsertHistory(ctx, ingest.HistoryEntry{
			RecordID: "r", Namespace: "n", CollectionName: c,
			Visibility: "shared", Status: "success",
		}); err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := repo.DeleteHistoryByCollection(ctx, "a")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d, want 2", deleted)
	}
}

func TestJobRoundTripCoversEveryColumn(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	started := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	finished := started.Add(90 * time.Second)
	in := ingest.Job{
		RequestJSON:     `{"collection":"c"}`,
		Status:          "running",
		TotalChunks:     10,
		CompletedChunks: 4,
		StartedAt:       &started,
	}
	id, err := repo.UpsertJob(ctx, in)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, ok, err := repo.GetJob(ctx, id)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.RequestJSON != in.RequestJSON {
		t.Errorf("request_json = %q, want %q", got.RequestJSON, in.RequestJSON)
	}
	if got.Status != "running" || got.TotalChunks != 10 || got.CompletedChunks != 4 {
		t.Errorf("progress = %q %d/%d", got.Status, got.CompletedChunks, got.TotalChunks)
	}
	if got.StartedAt == nil || !got.StartedAt.Equal(started) {
		t.Errorf("started_at = %v, want %v", got.StartedAt, started)
	}
	if got.FinishedAt != nil {
		t.Errorf("finished_at = %v, want nil", got.FinishedAt)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at was not defaulted")
	}

	// Completing the job must update in place and keep started_at.
	in.ID = id
	in.Status = "success"
	in.CompletedChunks = 10
	in.StartedAt = nil
	in.FinishedAt = &finished
	if _, err := repo.UpsertJob(ctx, in); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	got, _, err = repo.GetJob(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "success" || got.CompletedChunks != 10 {
		t.Errorf("after completion = %q %d", got.Status, got.CompletedChunks)
	}
	if got.StartedAt == nil || !got.StartedAt.Equal(started) {
		t.Errorf("started_at = %v, want it preserved at %v", got.StartedAt, started)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(finished) {
		t.Errorf("finished_at = %v, want %v", got.FinishedAt, finished)
	}
}
