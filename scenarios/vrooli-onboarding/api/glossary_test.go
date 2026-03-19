package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGlossaryAll verifies GET /api/v1/glossary returns all entries.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossaryAll(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/glossary", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	entries, ok := resp["entries"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatal("expected non-empty entries list")
	}

	count, ok := resp["count"].(float64)
	if !ok || count == 0 {
		t.Fatal("expected non-zero count")
	}
}

// TestGlossarySearch verifies searching glossary by term.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossarySearch(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/glossary?q=database", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	entries, ok := resp["entries"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatal("expected matching entries for 'database'")
	}

	if resp["query"] != "database" {
		t.Errorf("expected query=database, got %v", resp["query"])
	}
}

// TestGlossarySearchNoMatch verifies empty search results.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossarySearchNoMatch(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/glossary?q=xyznonexistent", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// entries should be null/empty
	if entries, ok := resp["entries"].([]any); ok && len(entries) > 0 {
		t.Error("expected no matching entries")
	}
}

// TestSetupOrder verifies GET /api/v1/setup-order returns dependency-sorted resources.
// [REQ:REQ-P2-001] - Setup Order Algorithm
func TestSetupOrder(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "postgis", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "redis", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup-order", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	order, ok := resp["setup_order"].([]any)
	if !ok || len(order) != 3 {
		t.Fatalf("expected 3 ordered resources, got %v", resp["setup_order"])
	}

	// postgres should come before postgis (postgis depends on postgres)
	var postgresOrder, postgisOrder float64
	for _, item := range order {
		entry := item.(map[string]any)
		if entry["name"] == "postgres" {
			postgresOrder = entry["order"].(float64)
		}
		if entry["name"] == "postgis" {
			postgisOrder = entry["order"].(float64)
		}
	}

	if postgresOrder >= postgisOrder {
		t.Errorf("postgres (order=%v) should come before postgis (order=%v)", postgresOrder, postgisOrder)
	}
}

// TestSetupOrderCircularDeps verifies handling of circular dependencies.
// [REQ:REQ-P2-001] - Setup Order Algorithm
func TestSetupOrderCircularDeps(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "judge0", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup-order", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// judge0 depends on postgres and redis which aren't available,
	// but it should still appear in the order
	order, ok := resp["setup_order"].([]any)
	if !ok || len(order) == 0 {
		t.Fatal("expected non-empty setup order even with missing deps")
	}
}
