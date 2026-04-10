//go:build integration

package store_test

import (
	"testing"
	"time"

	"tunnel-manager/domain"
	"tunnel-manager/store"
	"tunnel-manager/testutil"
)

// [REQ:OBS-001] Metrics time-series storage tests

func TestMetricsStore_StoreAndQuery(t *testing.T) {
	db := testutil.SetupTestDB(t)
	metricsStore := store.NewMetricsStore(db)

	// Store a metrics snapshot
	m := &domain.TunnelMetrics{
		HAConnections: 4,
		RequestErrors: 1.5,
		ActiveStreams: 10,
		SmoothedRTT:   42.3,
		ScrapedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	if err := metricsStore.Store(m); err != nil {
		t.Fatalf("Store: %v", err)
	}

	// Query last 24 hours
	to := time.Now().Add(time.Minute)
	from := to.Add(-24 * time.Hour)
	records, err := metricsStore.Query(from, to)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].HAConnections != 4 {
		t.Errorf("HAConnections: want 4, got %d", records[0].HAConnections)
	}
	if records[0].ActiveStreams != 10 {
		t.Errorf("ActiveStreams: want 10, got %d", records[0].ActiveStreams)
	}
}

func TestMetricsStore_Latest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	metricsStore := store.NewMetricsStore(db)

	// No data yet
	r, err := metricsStore.Latest()
	if err != nil {
		t.Fatalf("Latest (empty): %v", err)
	}
	if r != nil {
		t.Fatal("expected nil for empty store")
	}

	// Store two snapshots
	if err := metricsStore.Store(&domain.TunnelMetrics{HAConnections: 2}); err != nil {
		t.Fatalf("Store first: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := metricsStore.Store(&domain.TunnelMetrics{HAConnections: 5}); err != nil {
		t.Fatalf("Store second: %v", err)
	}

	r, err = metricsStore.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil record")
	}
	if r.HAConnections != 5 {
		t.Errorf("Latest HAConnections: want 5, got %d", r.HAConnections)
	}
}

func TestMetricsStore_PurgeOld(t *testing.T) {
	db := testutil.SetupTestDB(t)
	metricsStore := store.NewMetricsStore(db)
	metricsStore.SetRetention(1 * time.Millisecond) // very short for testing

	if err := metricsStore.Store(&domain.TunnelMetrics{HAConnections: 1}); err != nil {
		t.Fatalf("Store: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // ensure record is old enough

	deleted, err := metricsStore.PurgeOld()
	if err != nil {
		t.Fatalf("PurgeOld: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	r, _ := metricsStore.Latest()
	if r != nil {
		t.Error("expected no records after purge")
	}
}
