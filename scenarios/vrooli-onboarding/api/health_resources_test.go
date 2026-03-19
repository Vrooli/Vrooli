package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestResourceHealthEndpoint verifies GET /api/v1/resources/health returns health statuses.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthEndpoint(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "redis", "status": "stopped", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources/health", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	resources, ok := resp["resources"].([]any)
	if !ok || len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %v", resp["resources"])
	}

	healthyCount, ok := resp["healthy_count"].(float64)
	if !ok {
		t.Fatal("missing healthy_count field")
	}
	if healthyCount != 1 {
		t.Errorf("expected healthy_count=1, got %v", healthyCount)
	}
}

// TestResourceHealthAvailability verifies running resources are marked available.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthAvailability(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources/health", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	resources := resp["resources"].([]any)
	first := resources[0].(map[string]any)
	if first["available"] != true {
		t.Errorf("expected running resource to be available=true, got %v", first["available"])
	}
	if first["last_checked"] == nil || first["last_checked"] == "" {
		t.Error("expected last_checked to be set")
	}
}

// TestResourceHealthUnreachable verifies stopped resources are marked unavailable.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthUnreachable(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "redis", "status": "stopped", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources/health", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	resources := resp["resources"].([]any)
	first := resources[0].(map[string]any)
	if first["available"] != false {
		t.Errorf("expected stopped resource to be available=false, got %v", first["available"])
	}
}
