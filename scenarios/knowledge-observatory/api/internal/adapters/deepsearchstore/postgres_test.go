package deepsearchstore

import (
	"context"
	"testing"

	"knowledge-observatory/internal/services/deepsearch"
)

func TestPostgresCreateJobRequiresDB(t *testing.T) {
	t.Parallel()

	store := &Postgres{}
	_, err := store.CreateJob(context.Background(), deepsearch.DeepSearchRequest{
		Query:      "docs",
		Scope:      deepsearch.ScopeGlobal,
		MaxResults: 5,
	})
	if err == nil {
		t.Fatalf("expected error when DB is nil")
	}
}
