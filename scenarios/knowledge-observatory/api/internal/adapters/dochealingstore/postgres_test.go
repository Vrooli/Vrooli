package dochealingstore

import (
	"context"
	"testing"

	"knowledge-observatory/internal/services/dochealing"
)

func TestPostgresCreateJobRequiresDB(t *testing.T) {
	t.Parallel()

	store := &Postgres{}
	_, err := store.CreateJob(context.Background(), dochealing.HealRequest{
		ScenarioName: "alpha",
	}, nil)
	if err == nil {
		t.Fatalf("expected error when DB is nil")
	}
}
