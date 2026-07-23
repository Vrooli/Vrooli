package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// TestNewServerBuildsAndShutsDownTheRealCompositionRoot is a lightweight
// composition-root proof: it uses a disposable SQLite store and a bounded
// empty project root, but otherwise exercises the same graph construction,
// recovery ordering, middleware, and cleanup path as production startup.
func TestNewServerBuildsAndShutsDownTheRealCompositionRoot(t *testing.T) {
	t.Setenv("AM_SQLITE_PATH", filepath.Join(t.TempDir(), "agent-manager.db"))
	t.Setenv("UPLOAD_DIR", t.TempDir())
	t.Setenv("PROJECT_ROOT", t.TempDir())
	server, err := NewServer()
	if err != nil {
		t.Fatalf("build server: %v", err)
	}
	if server.Router() == nil || server.orchestrator == nil || server.reconciler == nil || server.awaitRegistry == nil || server.workflowNudger == nil {
		t.Fatalf("incomplete service graph: %+v", server)
	}
	t.Cleanup(func() {
		if err := server.Cleanup(); err != nil {
			t.Errorf("cleanup server: %v", err)
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	server.Router().ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestEnvOrEmptyReflectsConfiguredAndAbsentValues(t *testing.T) {
	t.Setenv("AM_TEST_ENV", "configured")
	if got := envOrEmpty("AM_TEST_ENV"); got != "configured" {
		t.Fatalf("configured env=%q", got)
	}
	if got := envOrEmpty("AM_UNSET_TEST_ENV"); got != "" {
		t.Fatalf("unset env=%q", got)
	}
}
