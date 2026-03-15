package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// =============================================================================
// ParseServiceJSON unit tests
// =============================================================================

func TestParseServiceJSON_FullData(t *testing.T) {
	raw := `{
		"service": {
			"name": "my-scenario",
			"displayName": "My Scenario",
			"description": "A test scenario",
			"tags": ["web", "api"]
		},
		"dependencies": {
			"scenarios": {
				"test-genie": {"description": "Test runner"},
				"tidiness-manager": {"description": "Quality checker"}
			},
			"resources": {
				"postgres": {"description": "Database"}
			}
		},
		"lifecycle": {
			"test": {
				"steps": [
					{"name": "install-deps", "run": "npm install"},
					{"name": "run-tests", "run": "test-genie execute my-scenario --preset comprehensive"}
				]
			},
			"setup": {
				"steps": [
					{"name": "install-deps", "run": "npm install"},
					{"name": "build-api", "run": "go build -o api ."},
					{"name": "show-urls", "run": "echo done"}
				]
			}
		}
	}`

	env, err := ParseServiceJSON([]byte(raw), "my-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if env.Name != "my-scenario" {
		t.Errorf("Name = %q, want %q", env.Name, "my-scenario")
	}
	if env.DisplayName != "My Scenario" {
		t.Errorf("DisplayName = %q, want %q", env.DisplayName, "My Scenario")
	}
	if env.Description != "A test scenario" {
		t.Errorf("Description = %q, want %q", env.Description, "A test scenario")
	}
	if env.Path != "scenarios/my-scenario" {
		t.Errorf("Path = %q, want %q", env.Path, "scenarios/my-scenario")
	}
	if len(env.Tags) != 2 || env.Tags[0] != "web" || env.Tags[1] != "api" {
		t.Errorf("Tags = %v, want [web api]", env.Tags)
	}

	// Dependencies.
	if len(env.Dependencies.Scenarios) != 2 {
		t.Errorf("got %d scenario deps, want 2", len(env.Dependencies.Scenarios))
	}
	if env.Dependencies.Scenarios["test-genie"] != "Test runner" {
		t.Errorf("test-genie dep = %q, want %q", env.Dependencies.Scenarios["test-genie"], "Test runner")
	}
	if len(env.Dependencies.Resources) != 1 {
		t.Errorf("got %d resource deps, want 1", len(env.Dependencies.Resources))
	}
	if env.Dependencies.Resources["postgres"] != "Database" {
		t.Errorf("postgres dep = %q, want %q", env.Dependencies.Resources["postgres"], "Database")
	}

	// Lifecycle — test command is last step in lifecycle.test.
	if env.Lifecycle.TestCommand != "test-genie execute my-scenario --preset comprehensive" {
		t.Errorf("TestCommand = %q, want test-genie command", env.Lifecycle.TestCommand)
	}
	// Build command is first setup step whose name contains "build".
	if env.Lifecycle.BuildCommand != "go build -o api ." {
		t.Errorf("BuildCommand = %q, want %q", env.Lifecycle.BuildCommand, "go build -o api .")
	}
}

func TestParseServiceJSON_MissingLifecycle(t *testing.T) {
	raw := `{
		"service": {
			"name": "bare-scenario",
			"displayName": "Bare",
			"description": "No lifecycle"
		},
		"dependencies": {}
	}`

	env, err := ParseServiceJSON([]byte(raw), "bare-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fall back to vrooli scenario test.
	want := "vrooli scenario test bare-scenario"
	if env.Lifecycle.TestCommand != want {
		t.Errorf("TestCommand = %q, want %q", env.Lifecycle.TestCommand, want)
	}
	if env.Lifecycle.BuildCommand != "" {
		t.Errorf("BuildCommand = %q, want empty", env.Lifecycle.BuildCommand)
	}
}

func TestParseServiceJSON_NilTags(t *testing.T) {
	raw := `{
		"service": {"name": "x", "displayName": "X", "description": "d"},
		"dependencies": {}
	}`

	env, err := ParseServiceJSON([]byte(raw), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if env.Tags == nil {
		t.Error("Tags should be non-nil empty slice, got nil")
	}
	if len(env.Tags) != 0 {
		t.Errorf("Tags = %v, want empty", env.Tags)
	}
}

func TestParseServiceJSON_EmptyDependencies(t *testing.T) {
	raw := `{
		"service": {"name": "x", "displayName": "X", "description": "d"},
		"dependencies": {"scenarios": {}, "resources": {}}
	}`

	env, err := ParseServiceJSON([]byte(raw), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(env.Dependencies.Scenarios) != 0 {
		t.Errorf("expected 0 scenario deps, got %d", len(env.Dependencies.Scenarios))
	}
	if len(env.Dependencies.Resources) != 0 {
		t.Errorf("expected 0 resource deps, got %d", len(env.Dependencies.Resources))
	}
}

func TestParseServiceJSON_InvalidJSON(t *testing.T) {
	_, err := ParseServiceJSON([]byte("not json"), "x")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// =============================================================================
// containsBuild unit tests
// =============================================================================

func TestContainsBuild(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"build-api", true},
		{"Build-UI", true},
		{"BUILD", true},
		{"rebuild-all", true},
		{"install-deps", false},
		{"show-urls", false},
		{"buil", false}, // too short
		{"", false},
	}
	for _, tt := range tests {
		if got := containsBuild(tt.name); got != tt.want {
			t.Errorf("containsBuild(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// =============================================================================
// HTTP handler integration tests
// =============================================================================

// setupEnvelopeTestServer creates a minimal Server with a temp filesystem for service.json fixtures.
func setupEnvelopeTestServer(t *testing.T) (*Server, *mux.Router, string) {
	t.Helper()

	tmpDir := t.TempDir()
	router := mux.NewRouter()

	srv := &Server{
		router:        router,
		git:           &FakeGitRunner{RepoRoot: tmpDir, IsRepository: true},
		envelopeCache: NewEnvelopeCache(60 * time.Second),
	}

	router.HandleFunc("/api/v1/scenarios/{slug}/envelope", srv.handleScenarioEnvelope).Methods("GET")

	return srv, router, tmpDir
}

// writeServiceJSON writes a service.json fixture for the given scenario slug.
func writeServiceJSON(t *testing.T, repoRoot, slug, content string) {
	t.Helper()
	dir := filepath.Join(repoRoot, "scenarios", slug, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create fixture dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
}

func TestHandleScenarioEnvelope_Success(t *testing.T) {
	_, router, tmpDir := setupEnvelopeTestServer(t)

	writeServiceJSON(t, tmpDir, "test-scenario", `{
		"service": {"name": "test-scenario", "displayName": "Test Scenario", "description": "A test", "tags": ["test"]},
		"dependencies": {"scenarios": {"test-genie": {"description": "Tests"}}},
		"lifecycle": {"test": {"steps": [{"name": "run", "run": "make test"}]}}
	}`)

	req := httptest.NewRequest("GET", "/api/v1/scenarios/test-scenario/envelope", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp ScenarioEnvelopeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "test-scenario" {
		t.Errorf("Name = %q, want %q", resp.Name, "test-scenario")
	}
	if resp.Lifecycle.TestCommand != "make test" {
		t.Errorf("TestCommand = %q, want %q", resp.Lifecycle.TestCommand, "make test")
	}
	if resp.Dependencies.Scenarios["test-genie"] != "Tests" {
		t.Errorf("test-genie dep = %q, want %q", resp.Dependencies.Scenarios["test-genie"], "Tests")
	}
}

func TestHandleScenarioEnvelope_NotFound(t *testing.T) {
	_, router, _ := setupEnvelopeTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/scenarios/nonexistent/envelope", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleScenarioEnvelope_MalformedJSON(t *testing.T) {
	_, router, tmpDir := setupEnvelopeTestServer(t)

	writeServiceJSON(t, tmpDir, "broken", "not valid json {{{")

	req := httptest.NewRequest("GET", "/api/v1/scenarios/broken/envelope", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleScenarioEnvelope_CachesResult(t *testing.T) {
	srv, router, tmpDir := setupEnvelopeTestServer(t)

	writeServiceJSON(t, tmpDir, "cached", `{
		"service": {"name": "cached", "displayName": "Cached", "description": "d"},
		"dependencies": {}
	}`)

	// First request — populates cache.
	req1 := httptest.NewRequest("GET", "/api/v1/scenarios/cached/envelope", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first request: status = %d, want 200", w1.Code)
	}

	// Delete the file — second request should still succeed from cache.
	os.RemoveAll(filepath.Join(tmpDir, "scenarios", "cached"))

	// Verify cache is populated.
	if _, ok := srv.envelopeCache.Get("cached"); !ok {
		t.Fatal("expected cache entry after first request")
	}

	req2 := httptest.NewRequest("GET", "/api/v1/scenarios/cached/envelope", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("cached request: status = %d, want 200", w2.Code)
	}
}
