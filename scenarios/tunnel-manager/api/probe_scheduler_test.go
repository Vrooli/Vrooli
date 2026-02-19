package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// [REQ:PROBE-005] Periodic probe scheduling - starts and runs at interval
func TestProbeSchedulerRunsAtInterval(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	port := extractPort(t, ts.URL)
	seedTestRoute(t, db, "sched-test", "scenario-a", port)

	probeSvc := NewProbeService(db, svc)
	scheduler := NewProbeScheduler(probeSvc, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scheduler.Start(ctx)
	defer scheduler.Stop()

	if !scheduler.IsRunning() {
		t.Error("scheduler should be running after Start()")
	}

	// Wait for at least 2 cycles
	time.Sleep(350 * time.Millisecond)

	lastRun := scheduler.LastRun()
	if lastRun.IsZero() {
		t.Error("expected lastRun to be non-zero after running")
	}

	if scheduler.LastError() != nil {
		t.Errorf("unexpected scheduler error: %v", scheduler.LastError())
	}

	// Verify probe results were persisted
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM probe_results").Scan(&count)
	if err != nil {
		t.Fatalf("query probe_results: %v", err)
	}
	// At least 2 cycles × 2 probes (internal + external) = 4 results minimum
	if count < 4 {
		t.Errorf("expected at least 4 persisted probe results, got %d", count)
	}
}

// [REQ:PROBE-005] Scheduler stops cleanly
func TestProbeSchedulerStops(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	port := extractPort(t, ts.URL)
	seedTestRoute(t, db, "sched-stop", "scenario-b", port)

	probeSvc := NewProbeService(db, svc)
	scheduler := NewProbeScheduler(probeSvc, 50*time.Millisecond)

	ctx := context.Background()
	scheduler.Start(ctx)

	// Let it run briefly
	time.Sleep(100 * time.Millisecond)

	scheduler.Stop()

	if scheduler.IsRunning() {
		t.Error("scheduler should not be running after Stop()")
	}

	// Verify no more probes run after stop
	var countBefore int
	if err := db.QueryRow("SELECT COUNT(*) FROM probe_results").Scan(&countBefore); err != nil {
		t.Fatalf("count before: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	var countAfter int
	if err := db.QueryRow("SELECT COUNT(*) FROM probe_results").Scan(&countAfter); err != nil {
		t.Fatalf("count after: %v", err)
	}

	if countAfter > countBefore {
		t.Error("probes continued running after Stop()")
	}
}

// [REQ:PROBE-005] Scheduler is idempotent (double start is no-op)
func TestProbeSchedulerDoubleStart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	probeSvc := NewProbeService(db, svc)
	scheduler := NewProbeScheduler(probeSvc, 1*time.Second)

	ctx := context.Background()
	scheduler.Start(ctx)
	scheduler.Start(ctx) // second call should be no-op

	defer scheduler.Stop()

	if !scheduler.IsRunning() {
		t.Error("scheduler should be running")
	}
}
