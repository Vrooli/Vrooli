//go:build integration

package store_test

import (
	"testing"
	"time"

	"tunnel-manager/store"
	"tunnel-manager/testutil"
)

// [REQ:OBS-002] Probe history storage tests

func TestProbeStore_QueryByRoute(t *testing.T) {
	db := testutil.SetupTestDB(t)
	probeStore := store.NewProbeStore(db)

	// Seed a route and probe results
	route := testutil.SeedTestRoute(t, db, "test-app", "test-scenario", 8080)

	// Insert probe results directly
	if _, err := db.Exec(
		`INSERT INTO probe_results (route_id, probe_type, status, latency_ms) VALUES ($1, 'internal', 'up', 42)`,
		route.ID,
	); err != nil {
		t.Fatalf("insert internal probe: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO probe_results (route_id, probe_type, status, latency_ms) VALUES ($1, 'external', 'down', 1500)`,
		route.ID,
	); err != nil {
		t.Fatalf("insert external probe: %v", err)
	}

	// Query
	to := time.Now().Add(time.Minute)
	from := to.Add(-1 * time.Hour)
	results, err := probeStore.QueryByRoute(route.ID, from, to)
	if err != nil {
		t.Fatalf("QueryByRoute: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestProbeStore_QueryRecent(t *testing.T) {
	db := testutil.SetupTestDB(t)
	probeStore := store.NewProbeStore(db)

	route := testutil.SeedTestRoute(t, db, "recent-app", "recent-scenario", 9090)
	for i := 0; i < 5; i++ {
		if _, err := db.Exec(
			`INSERT INTO probe_results (route_id, probe_type, status, latency_ms) VALUES ($1, 'internal', 'up', $2)`,
			route.ID, i*10,
		); err != nil {
			t.Fatalf("insert probe %d: %v", i, err)
		}
	}

	results, err := probeStore.QueryRecent(3)
	if err != nil {
		t.Fatalf("QueryRecent: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestProbeStore_PurgeOld(t *testing.T) {
	db := testutil.SetupTestDB(t)
	probeStore := store.NewProbeStore(db)
	probeStore.SetRetention(1 * time.Millisecond)

	route := testutil.SeedTestRoute(t, db, "purge-app", "purge-scenario", 7070)
	if _, err := db.Exec(
		`INSERT INTO probe_results (route_id, probe_type, status, latency_ms) VALUES ($1, 'internal', 'up', 10)`,
		route.ID,
	); err != nil {
		t.Fatalf("insert probe: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	deleted, err := probeStore.PurgeOld()
	if err != nil {
		t.Fatalf("PurgeOld: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
}
