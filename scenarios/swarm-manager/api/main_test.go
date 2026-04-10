package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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
	// Disable auto-workshop to prevent goroutine leaks in tests.
	vrooliDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := map[string]any{
		"auto_initialize_workshop": false,
		"auto_advance_workshop":    false,
		"auto_cascade_workshop":    false,
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	if err := os.WriteFile(filepath.Join(vrooliDir, "settings.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCENARIO_ROOT", root)
	return NewServerWithRoot(root)
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
