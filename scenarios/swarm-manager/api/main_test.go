package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/promptmanager"
)

// newTestServer creates a Server backed by an isolated temp directory so that
// tests never read from or write to the production scenario root.
//
// VROOLI_STORAGE_ROOT is the canonical substrate for routing data/cache/state
// under the test's tempdir (see runtimepaths.paths_test.go). The legacy
// XDG_* envs are not used by the storage substrate.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	root := t.TempDir()
	// Disable auto-workshop flags so item creation doesn't fire background
	// agent spawns under the test. The production settings handler reads
	// from `<scenarioRoot>/config/settings.json` (see main.go), so the
	// file must land there.
	configDir := filepath.Join(root, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := map[string]any{
		"auto_initialize_workshop": false,
		"auto_advance_workshop":    false,
		"auto_cascade_workshop":    false,
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	if err := os.WriteFile(filepath.Join(configDir, "settings.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCENARIO_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "storage"))
	t.Setenv("AGENT_MANAGER_ENABLED", "false")
	return newServerWithRoot(root, &promptmanager.MockClient{Result: "test prompt"})
}

// testDataRoot returns the test's data-class root for swarm-manager. The
// VROOLI_STORAGE_ROOT env (set by newTestServer) must be in scope.
func testDataRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv("VROOLI_STORAGE_ROOT")
	if root == "" {
		t.Fatal("VROOLI_STORAGE_ROOT not set; call newTestServer before testDataRoot")
	}
	return filepath.Join(root, "data", "vrooli", "swarm-manager")
}

func TestNewServerHealthRoutes(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.Handler()

	for _, path := range []string{"/health", "/api/v1/health"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %s to return 200, got %d", path, rec.Code)
		}
	}
}

func TestLoggingMiddlewarePassesThrough(t *testing.T) {
	wrapped := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestHandlerRecovery(t *testing.T) {
	srv := newTestServer(t)
	srv.router.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}
