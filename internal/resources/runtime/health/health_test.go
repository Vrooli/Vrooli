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

func TestRunCheckCommandLeavesCombinedOutputUnset(t *testing.T) {
	var captured *exec.Cmd

	result, err := RunCheck(context.Background(), manifestpkg.ResourceHealthCheck{
		Type:    "command",
		Command: []string{"claude", "--version"},
	}, Config{
		Runner: func(_ context.Context, cmd *exec.Cmd) ([]byte, error) {
			captured = cmd
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if !result.Healthy {
		t.Fatalf("expected healthy result: %+v", result)
	}
	if captured == nil {
		t.Fatal("expected runner to receive command")
	}
	if captured.Stdout != nil || captured.Stderr != nil {
		t.Fatalf("expected command health check to leave stdout/stderr unset, got stdout=%v stderr=%v", captured.Stdout, captured.Stderr)
	}
}
