package jobstore

import (
	"context"
	"strings"
	"testing"

	"knowledge-observatory/internal/ports"
)

func TestPostgresJobStoreRequiresDB(t *testing.T) {
	store := &Postgres{}

	if _, err := store.EnqueueDocumentIngest(context.Background(), ports.DocumentIngestJobRequest{}); err == nil {
		t.Fatalf("expected enqueue error without db")
	}
	if _, _, err := store.GetJob(context.Background(), "job"); err == nil {
		t.Fatalf("expected get job error without db")
	}
	if _, _, err := store.ClaimNextPendingJob(context.Background()); err == nil {
		t.Fatalf("expected claim error without db")
	}
	if err := store.UpdateJobProgress(context.Background(), "job", 1, 2); err == nil {
		t.Fatalf("expected update error without db")
	}
	if err := store.CompleteJob(context.Background(), "job", "success", ""); err == nil {
		t.Fatalf("expected complete error without db")
	}
}

func TestNewUUIDv4ForJobFormat(t *testing.T) {
	id := newUUIDv4ForJob()
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Fatalf("expected uuid format, got %q", id)
	}
	if len(id) < 32 {
		t.Fatalf("expected uuid length, got %q", id)
	}
}
