package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	// The operating-mode registry loads mode data from `<scenarioRoot>/modes`
	// (see operatingmode.ValidateRegistry in main.go). Symlink the real
	// committed mode folder into the isolated test root so the server starts
	// with the production modes without copying files or reaching outside the
	// tempdir. The api package's working directory is `scenarios/swarm-manager/api`,
	// so the committed modes live at `../modes`.
	if realModes, err := filepath.Abs("../modes"); err == nil {
		if err := os.Symlink(realModes, filepath.Join(root, "modes")); err != nil {
			t.Fatalf("link operating-mode data into test root: %v", err)
		}
	}
	// Transition declarations are production-owned scenario configuration. The
	// integration server must load them too; otherwise workflow-backed routes
	// fail with "transition ... is not registered" before exercising their
	// actual behavior.
	if realTransitions, err := filepath.Abs("../.vrooli/swarm-transitions"); err == nil {
		if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(realTransitions, filepath.Join(root, ".vrooli", "swarm-transitions")); err != nil {
			t.Fatalf("link transition declarations into test root: %v", err)
		}
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

func TestNewServerComposesMigratedAdaptersOnSharedTransitionRunner(t *testing.T) {
	srv := newTestServer(t)
	if srv.transitionRunner == nil || srv.capturesHandler == nil {
		t.Fatal("transition runner and capture handler must be composed")
	}
	applies, inputs := srv.transitionRunner.Counts()
	if applies != 12 || inputs != 14 {
		t.Fatalf("shared transition runner registrations = applies:%d inputs:%d, want applies:12 inputs:14", applies, inputs)
	}
}

func TestLoadTransitionRegistryFailsLoudlyWhenDeclarationsAreMissing(t *testing.T) {
	root := t.TempDir()
	defer func() {
		panicValue := recover()
		if panicValue == nil {
			t.Fatal("loadTransitionRegistry did not fail for a missing registry")
		}
		if !strings.Contains(panicValue.(error).Error(), "load workflow transition registry") {
			t.Fatalf("panic = %v", panicValue)
		}
	}()
	_ = loadTransitionRegistry(root)
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
