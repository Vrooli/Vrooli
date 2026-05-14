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
func newTestServer(t *testing.T) *Server {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"ideas", "research", "fix", "execute", "chore"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Disable auto-workshop flags so item creation doesn't fire background
	// agent spawns under the test. The production settings handler reads
	// from `<scenarioRoot>/config/settings.json` (see main.go:238), so the
	// file must land there — writing to `.vrooli/settings.json` (an older
	// location) is silently ignored.
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
	// Isolate XDG state/data/cache roots so agent-activity and other
	// runtimepaths consumers write into this test's tempdir instead of
	// the user's real XDG dirs. Without this, stale state from prior
	// tests (e.g., old agent-activity "initialize" records) leaks into
	// new servers and breaks any test that asserts a clean slate.
	xdg := filepath.Join(root, "xdg")
	t.Setenv("XDG_STATE_HOME", filepath.Join(xdg, "state"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(xdg, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(xdg, "cache"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(xdg, "config"))
	t.Setenv("AGENT_MANAGER_ENABLED", "false")
	return newServerWithRoot(root, &promptmanager.MockClient{Result: "test prompt"})
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
