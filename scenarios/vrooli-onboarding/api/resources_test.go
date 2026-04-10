package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

// Common test resource fixtures to avoid repeating verbose map literals.
var (
	testResPostgres = map[string]string{"name": "postgres", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"}
	testResRedis    = map[string]string{"name": "redis", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"}
	testResOllama   = map[string]string{"name": "ollama", "status": "installed", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"}
	testResPostgis  = map[string]string{"name": "postgis", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"}
	testResJudge0   = map[string]string{"name": "judge0", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"}
	testResStopped  = map[string]string{"name": "redis", "status": "stopped", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"}
	testResMystery  = map[string]string{"name": "mystery", "status": "WEIRD_STATUS", "installed": "false", "last_updated": "2026-01-01T00:00:00Z"}
)

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

// doRequest performs an HTTP request against the test server and returns the recorder.
// For GET requests, body should be empty string. For POST/PUT, pass JSON body string.
func doRequest(t *testing.T, srv *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	return w
}

// doGet is a convenience wrapper for GET requests.
func doGet(t *testing.T, srv *Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	return doRequest(t, srv, http.MethodGet, path, "")
}

// doPost is a convenience wrapper for POST requests with a JSON body.
func doPost(t *testing.T, srv *Server, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doRequest(t, srv, http.MethodPost, path, body)
}

// requireStatus asserts the response status code and fatals with the body on mismatch.
func requireStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, want, w.Body.String())
	}
}

// decodeJSON unmarshals the response body into dst.
func decodeJSON(t *testing.T, w *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), dst); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

// TestLoadResources verifies loading from a temp running-resources.json.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestLoadResources(t *testing.T) {
	dir := t.TempDir()
	writeResourcesFile(t, dir, []map[string]string{
		testResPostgres,
		testResOllama,
		testResMystery,
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
	srv := newTestServer(t, []map[string]string{testResPostgres, testResRedis})

	w := doGet(t, srv, "/api/v1/resources")
	requireStatus(t, w, http.StatusOK)

	var body map[string]any
	decodeJSON(t, w, &body)

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
	srv := newTestServer(t, []map[string]string{testResPostgres, testResOllama})

	w := doGet(t, srv, "/api/v1/resources/postgres")
	requireStatus(t, w, http.StatusOK)

	var res Resource
	decodeJSON(t, w, &res)

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
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/resources/nonexistent")
	requireStatus(t, w, http.StatusNotFound)

	var body map[string]string
	decodeJSON(t, w, &body)
	if body["error"] == "" {
		t.Error("expected error message in response")
	}
}

// TestHandleGetResourceCaseInsensitive verifies case-insensitive matching.
// [REQ:REQ-P0-002] - Resource Detail View
func TestHandleGetResourceCaseInsensitive(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/resources/Postgres")
	requireStatus(t, w, http.StatusOK)
}

// TestResolveResourcesPathWithVrooliRoot verifies VROOLI_ROOT env var path.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestResolveResourcesPathWithVrooliRoot(t *testing.T) {
	dir := t.TempDir()
	writeResourcesFile(t, dir, []map[string]string{testResPostgres})
	t.Setenv("VROOLI_ROOT", dir)

	got := resolveResourcesPath()
	want := filepath.Join(dir, ".vrooli", "running-resources.json")
	if got != want {
		t.Errorf("resolveResourcesPath() = %q, want %q", got, want)
	}
}

// TestResolveResourcesPathWalkUp verifies walk-up directory traversal.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestResolveResourcesPathWalkUp(t *testing.T) {
	dir := t.TempDir()
	writeResourcesFile(t, dir, []map[string]string{testResPostgres})
	t.Setenv("VROOLI_ROOT", "")

	// Create a nested directory to walk up from
	nested := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	// Change to nested dir so walk-up finds .vrooli/running-resources.json
	origDir, _ := os.Getwd()
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("cleanup chdir: %v", err)
		}
	})

	got := resolveResourcesPath()
	want := filepath.Join(dir, ".vrooli", "running-resources.json")
	if got != want {
		t.Errorf("resolveResourcesPath() = %q, want %q", got, want)
	}
}

// TestResolveResourcesPathNotFound verifies empty return when no resources file exists.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestResolveResourcesPathNotFound(t *testing.T) {
	t.Setenv("VROOLI_ROOT", "")

	// Use a temp dir with no .vrooli directory
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("cleanup chdir: %v", err)
		}
	})

	got := resolveResourcesPath()
	if got != "" {
		t.Errorf("resolveResourcesPath() = %q, want empty string", got)
	}
}

// TestHandleListResourcesLoadError verifies 500 when resources file is missing.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestHandleListResourcesLoadError(t *testing.T) {
	t.Setenv("VROOLI_ROOT", t.TempDir()) // no .vrooli/running-resources.json
	srv := NewServer(nil)

	w := doGet(t, srv, "/api/v1/resources")
	requireStatus(t, w, http.StatusInternalServerError)

	var body map[string]string
	decodeJSON(t, w, &body)
	if !strings.Contains(body["error"], "failed to load resources") {
		t.Errorf("expected resource load error, got %q", body["error"])
	}
}

// TestHandleGetResourceLoadError verifies 500 when resources file is missing.
// [REQ:REQ-P0-002] - Resource Detail View
func TestHandleGetResourceLoadError(t *testing.T) {
	t.Setenv("VROOLI_ROOT", t.TempDir())
	srv := NewServer(nil)

	w := doGet(t, srv, "/api/v1/resources/postgres")
	requireStatus(t, w, http.StatusInternalServerError)
}

// TestLoadResourcesInvalidJSON verifies error for malformed resources file.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestLoadResourcesInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	vrooliDir := filepath.Join(dir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vrooliDir, "running-resources.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_ROOT", dir)

	_, err := loadResources()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestLoadResourcesEmptyList verifies loading an empty resources list.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestLoadResourcesEmptyList(t *testing.T) {
	dir := t.TempDir()
	writeResourcesFile(t, dir, []map[string]string{})
	t.Setenv("VROOLI_ROOT", dir)

	resources, err := loadResources()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(resources))
	}
}

// TestHandleListResourcesEmpty verifies GET /api/v1/resources with no resources.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestHandleListResourcesEmpty(t *testing.T) {
	srv := newTestServer(t, []map[string]string{})

	w := doGet(t, srv, "/api/v1/resources")
	requireStatus(t, w, http.StatusOK)

	var body map[string]any
	decodeJSON(t, w, &body)

	count := body["count"].(float64)
	if int(count) != 0 {
		t.Errorf("expected count=0, got %v", count)
	}
}

// TestHandleListResourcesResponseHasLoadedAt verifies loaded_at timestamp format.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestHandleListResourcesResponseHasLoadedAt(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/resources")
	requireStatus(t, w, http.StatusOK)

	var body map[string]any
	decodeJSON(t, w, &body)

	loadedAt, ok := body["loaded_at"].(string)
	if !ok || loadedAt == "" {
		t.Error("expected non-empty loaded_at timestamp string")
	}
}

// TestCategorizeSpecificResources verifies categorization for more resource types.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestCategorizeSpecificResources(t *testing.T) {
	tests := []struct {
		resource string
		want     string
	}{
		{"vault", "security"},
		{"postgis", "database"},
		{"qdrant", "database"},
		{"judge0", "devops"},
		{"n8n", "devops"},
	}

	for _, tc := range tests {
		got := categorize(tc.resource)
		if got != tc.want {
			t.Errorf("categorize(%q) = %q, want %q", tc.resource, got, tc.want)
		}
	}
}

// TestLoadResourcesStatusNormalization verifies all status variants are normalized.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestLoadResourcesStatusNormalization(t *testing.T) {
	dir := t.TempDir()
	writeResourcesFile(t, dir, []map[string]string{
		{"name": "a", "status": "running", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "b", "status": "installed", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "c", "status": "stopped", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "d", "status": "RUNNING", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "e", "status": "Starting", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
		{"name": "f", "status": "", "installed": "true", "last_updated": "2026-01-01T00:00:00Z"},
	})
	t.Setenv("VROOLI_ROOT", dir)

	resources, err := loadResources()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"running", "installed", "stopped", "running", "stopped", "stopped"}
	for i, want := range expected {
		if resources[i].Status != want {
			t.Errorf("resources[%d].Status = %q, want %q (input was %q)",
				i, resources[i].Status, want, resources[i].Name)
		}
	}
}

// TestHandleGetResourceAllFields verifies the full response structure for a single resource.
// [REQ:REQ-P0-002] - Resource Detail View
func TestHandleGetResourceAllFields(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/resources/postgres")
	requireStatus(t, w, http.StatusOK)

	var res Resource
	decodeJSON(t, w, &res)

	if res.Name != "postgres" {
		t.Errorf("Name = %q, want %q", res.Name, "postgres")
	}
	if res.Category != "database" {
		t.Errorf("Category = %q, want %q", res.Category, "database")
	}
	if res.Status != "running" {
		t.Errorf("Status = %q, want %q", res.Status, "running")
	}
	if res.Installed != "true" {
		t.Errorf("Installed = %q, want %q", res.Installed, "true")
	}
	if res.LastUpdated == "" {
		t.Error("LastUpdated should not be empty")
	}
}

// TestHandleUnknownRoute verifies 405 for wrong method on known endpoint.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestHandleDeleteResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doRequest(t, srv, http.MethodDelete, "/api/v1/resources", "")
	if w.Code == http.StatusOK {
		t.Error("DELETE /api/v1/resources should not return 200")
	}
}
