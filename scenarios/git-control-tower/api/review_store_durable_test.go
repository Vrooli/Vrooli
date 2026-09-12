package main

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestReviewJobStoreHydratesTerminalResultAfterRestart(t *testing.T) {
	db, err := sql.Open("sqlite", "file:gct-review-restart?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	first, err := NewReviewJobStoreWithDB(db)
	if err != nil {
		t.Fatal(err)
	}
	first.CreateWithIdempotency("job-restart", "request-restart", []string{"tests"}, "scenario", 2, DefaultReadinessThresholds())
	first.Complete("job-restart", &ReviewSummaryResponse{ScenarioName: "scenario", Readiness: ReadinessYellow, Timestamp: "2026-09-06T00:00:00Z"})

	second, err := NewReviewJobStoreWithDB(db)
	if err != nil {
		t.Fatal(err)
	}
	status, ok := second.Get("job-restart")
	if !ok || status.Status != "completed" || status.Summary == nil || status.Summary.Readiness != ReadinessYellow {
		t.Fatalf("hydrated status = %#v ok=%v", status, ok)
	}
	if id, ok := second.FindByIdempotency("request-restart"); !ok || id != "job-restart" {
		t.Fatalf("hydrated idempotency = %q ok=%v", id, ok)
	}
}
