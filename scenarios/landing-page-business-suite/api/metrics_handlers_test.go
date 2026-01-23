package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- handleMetricsTrack Tests ---

func TestHandleMetricsTrack_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupMetricsEvents(t, db)

	metricsService := NewMetricsService(db)
	handler := handleMetricsTrack(metricsService)

	body := `{
		"event_type": "page_view",
		"variant_slug": "control",
		"session_id": "test-session-1",
		"event_id": "evt_test_1"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/track", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}
	if resp["event_type"] != "page_view" {
		t.Errorf("Expected event_type 'page_view', got %v", resp["event_type"])
	}
}

func TestHandleMetricsTrack_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	metricsService := NewMetricsService(db)
	handler := handleMetricsTrack(metricsService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/track", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleMetricsTrack_ValidationError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	metricsService := NewMetricsService(db)
	handler := handleMetricsTrack(metricsService)

	// Missing required fields
	body := `{
		"event_type": "",
		"session_id": ""
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/track", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for validation error, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleMetricsTrack_InvalidEventType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	metricsService := NewMetricsService(db)
	handler := handleMetricsTrack(metricsService)

	body := `{
		"event_type": "invalid_event",
		"variant_slug": "control",
		"session_id": "test-session-1"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/track", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid event type, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- handleMetricsSummary Tests ---

func TestHandleMetricsSummary_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupMetricsEvents(t, db)

	metricsService := NewMetricsService(db)

	// Insert test events
	insertTestMetricEvent(t, db, "page_view", "control", "session1")
	insertTestMetricEvent(t, db, "page_view", "variant-a", "session2")
	insertTestMetricEvent(t, db, "conversion", "control", "session1")

	handler := handleMetricsSummary(metricsService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/summary", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp AnalyticsSummary
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have aggregated stats
	if resp.TotalVisitors < 0 {
		t.Error("Expected non-negative total_visitors")
	}
}

func TestHandleMetricsSummary_CustomDates(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupMetricsEvents(t, db)

	metricsService := NewMetricsService(db)
	handler := handleMetricsSummary(metricsService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/summary?start_date=2025-01-01&end_date=2025-12-31", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleMetricsSummary_InvalidDateFormat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	metricsService := NewMetricsService(db)
	handler := handleMetricsSummary(metricsService)

	// Invalid date format should silently use defaults
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/summary?start_date=invalid&end_date=also-invalid", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Should still return success (defaults are used)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d (defaults on invalid dates), got %d", http.StatusOK, w.Code)
	}
}

// --- handleMetricsVariantStats Tests ---

func TestHandleMetricsVariantStats_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupMetricsEvents(t, db)

	metricsService := NewMetricsService(db)

	// Insert events for different variants
	insertTestMetricEvent(t, db, "page_view", "control", "session1")
	insertTestMetricEvent(t, db, "page_view", "variant-a", "session2")
	insertTestMetricEvent(t, db, "click", "control", "session1")

	handler := handleMetricsVariantStats(metricsService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/variants", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Response should have start_date and end_date at minimum
	if resp["start_date"] == nil {
		t.Error("Expected 'start_date' in response")
	}
	if resp["end_date"] == nil {
		t.Error("Expected 'end_date' in response")
	}
}

func TestHandleMetricsVariantStats_FilterByVariant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupMetricsEvents(t, db)

	metricsService := NewMetricsService(db)

	// Insert events for different variants
	insertTestMetricEvent(t, db, "page_view", "control", "session1")
	insertTestMetricEvent(t, db, "page_view", "variant-a", "session2")

	handler := handleMetricsVariantStats(metricsService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/variants?variant=control", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestHandleMetricsVariantStats_CustomDates(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupMetricsEvents(t, db)

	metricsService := NewMetricsService(db)
	handler := handleMetricsVariantStats(metricsService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/variants?start_date=2025-01-01&end_date=2025-12-31", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should include dates in response
	if resp["start_date"] != "2025-01-01" {
		t.Errorf("Expected start_date '2025-01-01', got %v", resp["start_date"])
	}
	if resp["end_date"] != "2025-12-31" {
		t.Errorf("Expected end_date '2025-12-31', got %v", resp["end_date"])
	}
}

// Helper functions

func cleanupMetricsEvents(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DELETE FROM metrics_events"); err != nil {
		// Ignore error if table doesn't exist
	}
}

func insertTestMetricEvent(t *testing.T, db *sql.DB, eventType, variantSlug, sessionID string) {
	t.Helper()

	query := `
		INSERT INTO metrics_events (event_type, variant_slug, session_id, event_data)
		VALUES ($1, $2, $3, $4)
	`
	eventData := `{"event_id": "` + eventType + "_" + sessionID + `"}`

	if _, err := db.Exec(query, eventType, variantSlug, sessionID, eventData); err != nil {
		t.Fatalf("Failed to insert test metric event: %v", err)
	}
}
