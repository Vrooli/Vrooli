package testutil

import (
	"io"
	"net/http"
	"os"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})
}

func TestNewHTTPServer_ServesAndCleansUp(t *testing.T) {
	server := NewHTTPServer(t, okHandler())
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET %s: %v", server.URL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Fatalf("body = %q, want ok", string(body))
	}
}

func TestNewAPIServer_SetsAPIBaseURL(t *testing.T) {
	server := NewAPIServer(t, okHandler())
	if got, configured := os.LookupEnv("API_BASE_URL"); !configured || got != server.URL {
		t.Fatalf("API_BASE_URL = %q, want %q", got, server.URL)
	}
}

// TestNewAPIServer_RestoresEnvOnCleanup proves t.Setenv's auto-restore is
// what's actually cleaning up the env var. The outer test sets a sentinel;
// the inner subtest invokes NewAPIServer which clobbers it; when the inner
// exits the sentinel must be back. If a future refactor switches to bare
// os.Setenv, this test fails.
func TestNewAPIServer_RestoresEnvOnCleanup(t *testing.T) {
	const sentinel = "outer-sentinel-value"
	t.Setenv("API_BASE_URL", sentinel)

	t.Run("inner", func(tt *testing.T) {
		_ = NewAPIServer(tt, okHandler())
		if got, _ := os.LookupEnv("API_BASE_URL"); got == sentinel {
			tt.Fatal("inner subtest did not override API_BASE_URL")
		}
	})

	if got, _ := os.LookupEnv("API_BASE_URL"); got != sentinel {
		t.Fatalf("API_BASE_URL after inner cleanup = %q, want %q", got, sentinel)
	}
}
