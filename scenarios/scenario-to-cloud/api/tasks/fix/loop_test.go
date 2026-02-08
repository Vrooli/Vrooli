package fix

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/tasks/shared"
)

func TestDefaultLoopConfig(t *testing.T) {
	cfg := DefaultLoopConfig(0, "https://vrooli.com/health")
	if cfg.MaxIterations != 5 {
		t.Fatalf("expected default max iterations 5, got %d", cfg.MaxIterations)
	}
	if cfg.IterationTimeout != 15*time.Minute {
		t.Fatalf("unexpected iteration timeout: %v", cfg.IterationTimeout)
	}
	if cfg.DeployTimeout != 10*time.Minute {
		t.Fatalf("unexpected deploy timeout: %v", cfg.DeployTimeout)
	}
}

func TestLoopState_ShouldContinue(t *testing.T) {
	state := NewLoopState(DefaultLoopConfig(2, ""))
	if !state.ShouldContinue() {
		t.Fatal("expected continue with no iterations")
	}

	state.StartIteration()
	state.RecordIteration(domain.FixIterationRecord{Outcome: "continue"})
	if !state.ShouldContinue() {
		t.Fatal("expected continue after first continue outcome")
	}

	state.StartIteration()
	state.RecordIteration(domain.FixIterationRecord{Outcome: "continue"})
	if state.ShouldContinue() {
		t.Fatal("expected stop after max iterations reached")
	}
}

func TestRunHealthCheck(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okSrv.Close()

	passed, err := RunHealthCheck(context.Background(), okSrv.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected health check error: %v", err)
	}
	if !passed {
		t.Fatal("expected successful health check")
	}

	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer failSrv.Close()
	passed, err = RunHealthCheck(context.Background(), failSrv.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected health check error for 502 response: %v", err)
	}
	if passed {
		t.Fatal("expected failed health check for non-2xx response")
	}

	_, err = RunHealthCheck(context.Background(), "://bad-url", 2*time.Second)
	if err == nil {
		t.Fatal("expected request creation error for malformed URL")
	}
}

func TestDetermineOutcome(t *testing.T) {
	state := NewLoopState(DefaultLoopConfig(3, ""))
	state.CurrentIteration = 2

	if got := DetermineOutcome(state, &shared.AgentResult{Output: `{"iteration_report":{"outcome":"gave_up"}}`}, true); got != shared.FixStatusSuccess {
		t.Fatalf("health pass should win, got %q", got)
	}

	if got := DetermineOutcome(state, &shared.AgentResult{Output: `{"iteration_report":{"outcome":"gave_up"}}`}, false); got != shared.FixStatusAgentGaveUp {
		t.Fatalf("expected agent gave up, got %q", got)
	}

	state.CurrentIteration = 3
	if got := DetermineOutcome(state, &shared.AgentResult{Output: `{"iteration_report":{"outcome":"continue"}}`}, false); got != shared.FixStatusMaxIterations {
		t.Fatalf("expected max iterations, got %q", got)
	}
}
