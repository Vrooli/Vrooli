package orchestration

import (
	"context"
	"testing"

	"agent-manager/internal/fallback"
	"agent-manager/internal/health"
	"agent-manager/internal/testutil"
)

func TestHealthMarkerAdapterRecordsHealthyAndUnavailableObservations(t *testing.T) {
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	store := health.NewStore(db)
	adapter := newHealthMarkerAdapter(store, "run-123")
	adapter.MarkModelHealthy("codex", "gpt-test")
	adapter.MarkModelUnavailable("codex", "gpt-test", "capacity exhausted")
	entry, err := store.LatestModelStatus(context.Background(), "codex", "gpt-test")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Status != health.StatusFailed || entry.Reason != string(fallback.ReasonUnknown) || entry.Message != "capacity exhausted" {
		t.Fatalf("latest health entry = %+v", entry)
	}
}

func TestHealthMarkerAdapterNilStoreAndTriggeredByAreSafe(t *testing.T) {
	var nilAdapter *healthMarkerAdapter
	nilAdapter.MarkModelHealthy("codex", "gpt-test")
	nilAdapter.MarkModelUnavailable("codex", "gpt-test", "ignored")
	if got := nilAdapter.triggeredBy(); got != "runtime" {
		t.Fatalf("nil adapter trigger = %q", got)
	}
	if got := newHealthMarkerAdapter(nil, "").triggeredBy(); got != "runtime" {
		t.Fatalf("empty run trigger = %q", got)
	}
}
