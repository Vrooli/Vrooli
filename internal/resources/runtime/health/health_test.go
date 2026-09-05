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

// Scenario: a failing liveness check degrades the resource without downing it.
//
// The control plane executes liveness checks. A failing one means the resource
// answers requests but is not meeting its contract — running below its declared
// accelerator backend is the canonical case — so serving stays true while
// healthy becomes false.
func TestRunChecksDegradeOnFailingLiveness(t *testing.T) {
	// Given a resource whose readiness passes and whose liveness fails
	result, err := RunChecks(context.Background(), []manifestpkg.ResourceHealthCheck{
		{Kind: "readiness", Type: "command", Command: []string{"true"}},
		{Kind: "liveness", Type: "command", Command: []string{"not-a-real-command"}},
	}, Config{})
	// When the combined verdict is read
	if err != nil {
		t.Fatalf("RunChecks: %v", err)
	}

	// Then it is not healthy, because the contract is not met
	if result.Healthy {
		t.Fatalf("a failing liveness check must make the result unhealthy: %+v", result)
	}
	// And it is still serving, because the resource answers requests
	if !result.Serving {
		t.Fatalf("a failing liveness check must not make a serving resource look stopped: %+v", result)
	}
	// And the message names the check that failed
	if result.LivenessFailed == "" {
		t.Fatalf("LivenessFailed is empty; the operator must be told which contract was not met: %+v", result)
	}
}

func TestRunCheckHTTPRendersEnvironmentTarget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	result, err := RunCheck(context.Background(), manifestpkg.ResourceHealthCheck{Type: "http", Target: "${HEALTH_ENDPOINT}", ExpectedStatus: []int{http.StatusNoContent}}, Config{Env: []string{"HEALTH_ENDPOINT=" + srv.URL}})
	if err != nil || !result.Healthy {
		t.Fatalf("rendered HTTP health = %#v, %v", result, err)
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
