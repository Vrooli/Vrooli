package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestCategorize verifies category lookup for known and unknown resources.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestCategorize(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		want     string
	}{
		{"postgres is database", "postgres", "database"},
		{"ollama is ai", "ollama", "ai"},
		{"redis is database", "redis", "database"},
		{"minio is storage", "minio", "storage"},
		{"browserless is browser", "browserless", "browser"},
		{"unknown falls back to general", "some-unknown-resource", "general"},
		{"empty string falls back to general", "", "general"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := categorize(tc.resource)
			if got != tc.want {
				t.Errorf("categorize(%q) = %q, want %q", tc.resource, got, tc.want)
			}
		})
	}
}

// writeResourcesFile creates a running-resources.json in the given dir under .vrooli/.
func writeResourcesFile(t *testing.T, dir string, resources []map[string]string) {
	t.Helper()
	vrooliDir := filepath.Join(dir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	rawResources := make([]map[string]string, len(resources))
	copy(rawResources, resources)

	data := map[string]any{
		"resources":    rawResources,
		"last_updated": "2026-01-01T00:00:00Z",
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vrooliDir, "running-resources.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLoadResources verifies loading from a temp running-resources.json.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestLoadResources(t *testing.T) {
	dir := t.TempDir()
	writeResourcesFile(t, dir, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "ollama", "status": "installed", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "mystery", "status": "WEIRD_STATUS", "installed": "false", "last_updated": "2026-01-01T00:00:00Z"},
	})

	t.Setenv("VROOLI_ROOT", dir)

	resources, err := loadResources()
	if err != nil {
		t.Fatalf("loadResources() error: %v", err)
	}

	if len(resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(resources))
	}

	// Check postgres
	if resources[0].Name != "postgres" {
		t.Errorf("resources[0].Name = %q, want %q", resources[0].Name, "postgres")
	}
	if resources[0].Category != "database" {
		t.Errorf("resources[0].Category = %q, want %q", resources[0].Category, "database")
	}
	if resources[0].Status != "running" {
		t.Errorf("resources[0].Status = %q, want %q", resources[0].Status, "running")
	}

	// Check ollama
	if resources[1].Category != "ai" {
		t.Errorf("resources[1].Category = %q, want %q", resources[1].Category, "ai")
	}
	if resources[1].Status != "installed" {
		t.Errorf("resources[1].Status = %q, want %q", resources[1].Status, "installed")
	}

	// Check unknown status normalization
	if resources[2].Status != "stopped" {
		t.Errorf("resources[2].Status = %q, want %q (unknown status should normalize to stopped)", resources[2].Status, "stopped")
	}
	if resources[2].Category != "general" {
		t.Errorf("resources[2].Category = %q, want %q", resources[2].Category, "general")
	}
}

// TestLoadResourcesMissingFile verifies error when no resources file exists.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestLoadResourcesMissingFile(t *testing.T) {
	t.Setenv("VROOLI_ROOT", t.TempDir())

	_, err := loadResources()
	if err == nil {
		t.Fatal("expected error when resources file is missing, got nil")
	}
}

// newTestServer creates a Server with nil db (sufficient for resource/config tests)
// and sets up VROOLI_ROOT to a temp directory with resources.
func newTestServer(t *testing.T, resources []map[string]string) *Server {
	t.Helper()
	dir := t.TempDir()
	writeResourcesFile(t, dir, resources)
	t.Setenv("VROOLI_ROOT", dir)
	return NewServer(nil)
}

// TestHandleListResources verifies the GET /api/v1/resources endpoint.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestHandleListResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "redis", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	resources, ok := body["resources"].([]any)
	if !ok {
		t.Fatal("response missing 'resources' array")
	}
	if len(resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(resources))
	}

	count, ok := body["count"].(float64)
	if !ok {
		t.Fatal("response missing 'count'")
	}
	if int(count) != 2 {
		t.Errorf("count = %v, want 2", count)
	}

	if _, ok := body["loaded_at"]; !ok {
		t.Error("response missing 'loaded_at'")
	}
}

// TestHandleGetResource verifies GET /api/v1/resources/{name} for an existing resource.
// [REQ:REQ-P0-002] - Resource Detail View
func TestHandleGetResource(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "ollama", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources/postgres", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var res Resource
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Name != "postgres" {
		t.Errorf("Name = %q, want %q", res.Name, "postgres")
	}
	if res.Category != "database" {
		t.Errorf("Category = %q, want %q", res.Category, "database")
	}
}

// TestHandleGetResourceNotFound verifies 404 for unknown resource names.
// [REQ:REQ-P0-002] - Resource Detail View
func TestHandleGetResourceNotFound(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] == "" {
		t.Error("expected error message in response")
	}
}

// TestHandleGetResourceCaseInsensitive verifies case-insensitive matching.
// [REQ:REQ-P0-002] - Resource Detail View
func TestHandleGetResourceCaseInsensitive(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/resources/Postgres", nil)
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
