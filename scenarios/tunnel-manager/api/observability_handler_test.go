package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// [REQ:OBS-001] Metrics handler tests

func TestHandlerMetricsHistory_DefaultHours(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewMetricsStore(db)
	if err := store.Store(&TunnelMetrics{HAConnections: 4, ActiveStreams: 10}); err != nil {
		t.Fatalf("store: %v", err)
	}

	handler := handleMetricsHistory(store)
	req := httptest.NewRequest("GET", "/api/v1/metrics/history", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var records []MetricsRecord
	if err := json.Unmarshal(w.Body.Bytes(), &records); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

func TestHandlerMetricsHistory_CustomHours(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewMetricsStore(db)
	if err := store.Store(&TunnelMetrics{HAConnections: 2}); err != nil {
		t.Fatalf("store: %v", err)
	}

	handler := handleMetricsHistory(store)
	req := httptest.NewRequest("GET", "/api/v1/metrics/history?hours=1", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var records []MetricsRecord
	if err := json.Unmarshal(w.Body.Bytes(), &records); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

func TestHandlerMetricsHistory_EmptyResult(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewMetricsStore(db)
	handler := handleMetricsHistory(store)
	req := httptest.NewRequest("GET", "/api/v1/metrics/history", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var records []MetricsRecord
	if err := json.Unmarshal(w.Body.Bytes(), &records); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestHandlerMetricsHistory_InvalidHoursParam(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewMetricsStore(db)
	if err := store.Store(&TunnelMetrics{HAConnections: 1}); err != nil {
		t.Fatalf("store: %v", err)
	}

	handler := handleMetricsHistory(store)
	// Invalid hours param should fall back to default 24
	req := httptest.NewRequest("GET", "/api/v1/metrics/history?hours=abc", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var records []MetricsRecord
	if err := json.Unmarshal(w.Body.Bytes(), &records); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record (default 24h window), got %d", len(records))
	}
}

// [REQ:OBS-001] Metrics latest handler tests

func TestHandlerMetricsLatest_WithData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewMetricsStore(db)
	if err := store.Store(&TunnelMetrics{HAConnections: 4, ActiveStreams: 8}); err != nil {
		t.Fatalf("store: %v", err)
	}

	handler := handleMetricsLatest(store)
	req := httptest.NewRequest("GET", "/api/v1/metrics/latest", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var record MetricsRecord
	if err := json.Unmarshal(w.Body.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if record.HAConnections != 4 {
		t.Errorf("HAConnections = %d, want 4", record.HAConnections)
	}
}

func TestHandlerMetricsLatest_NoData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewMetricsStore(db)
	handler := handleMetricsLatest(store)
	req := httptest.NewRequest("GET", "/api/v1/metrics/latest", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["status"] != "no_data" {
		t.Errorf("expected status=no_data, got %q", resp["status"])
	}
}

// [REQ:OBS-002] Probe history handler tests

func TestHandlerProbeHistory_DefaultLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewProbeStore(db)
	// Seed a route so we have a valid route_id for probe results
	route := seedTestRoute(t, db, "probe-hist", "test-scenario", 3000)

	// Insert probe results
	for i := 0; i < 3; i++ {
		_, err := db.Exec(
			`INSERT INTO probe_results (route_id, probe_type, status, latency_ms) VALUES ($1, 'internal', 'up', $2)`,
			route.ID, 10+i,
		)
		if err != nil {
			t.Fatalf("insert probe %d: %v", i, err)
		}
	}

	handler := handleProbeHistory(store)
	req := httptest.NewRequest("GET", "/api/v1/probes/history", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var results []StoredProbeResult
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestHandlerProbeHistory_CustomLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewProbeStore(db)
	route := seedTestRoute(t, db, "probe-limit", "test-scenario", 3000)

	for i := 0; i < 5; i++ {
		_, err := db.Exec(
			`INSERT INTO probe_results (route_id, probe_type, status, latency_ms) VALUES ($1, 'internal', 'up', $2)`,
			route.ID, 10+i,
		)
		if err != nil {
			t.Fatalf("insert probe %d: %v", i, err)
		}
	}

	handler := handleProbeHistory(store)
	req := httptest.NewRequest("GET", "/api/v1/probes/history?limit=2", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var results []StoredProbeResult
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestHandlerProbeHistory_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewProbeStore(db)
	handler := handleProbeHistory(store)
	req := httptest.NewRequest("GET", "/api/v1/probes/history", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var results []StoredProbeResult
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// [REQ:OBS-002] Probe store query edge cases

func TestProbeStore_QueryByRoute_EmptyResult(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewProbeStore(db)
	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)
	results, err := store.QueryByRoute(99999, from, to)
	if err != nil {
		t.Fatalf("QueryByRoute: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestProbeStore_QueryRecent_DefaultsNegativeLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewProbeStore(db)
	route := seedTestRoute(t, db, "default-limit", "test-scenario", 3000)
	_, err := db.Exec(
		`INSERT INTO probe_results (route_id, probe_type, status, latency_ms) VALUES ($1, 'internal', 'up', 10)`,
		route.ID,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Negative limit should default to 100
	results, err := store.QueryRecent(-1)
	if err != nil {
		t.Fatalf("QueryRecent: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with default limit, got %d", len(results))
	}
}
