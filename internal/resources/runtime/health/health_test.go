package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

func TestRunChecksHTTPHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	result, err := RunChecks(context.Background(), []manifestpkg.ResourceHealthCheck{{
		Type:   "http",
		Target: srv.URL,
	}}, Config{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("RunChecks: %v", err)
	}
	if !result.Healthy {
		t.Fatalf("expected healthy result: %+v", result)
	}
}

func TestRunCheckCommandFailureReturnsUnhealthy(t *testing.T) {
	result, err := RunCheck(context.Background(), manifestpkg.ResourceHealthCheck{
		Type:    "command",
		Command: []string{"fake", "cmd"},
	}, Config{
		Runner: func(context.Context, *exec.Cmd) ([]byte, error) {
			return nil, errors.New("boom")
		},
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Healthy {
		t.Fatalf("expected unhealthy result: %+v", result)
	}
}

