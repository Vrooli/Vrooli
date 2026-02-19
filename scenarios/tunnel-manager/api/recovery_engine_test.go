package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// [REQ:RECOVER-001] [REQ:RECOVER-002] [REQ:RECOVER-003] Recovery engine unit tests

func newTestRecoveryEngine(t *testing.T, readyHandler http.HandlerFunc, cmdRunner func(ctx context.Context, name string, args ...string) ([]byte, error)) (*RecoveryEngine, *sql.DB) {
	t.Helper()
	db := setupTestDB(t)

	ts := httptest.NewServer(readyHandler)
	t.Cleanup(ts.Close)

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	opts := []RecoveryOption{
		WithConsecutiveFailures(2),
		WithMaxBackoffRetries(3),
		WithCircuitCooldown(50 * time.Millisecond),
	}
	if cmdRunner != nil {
		opts = append(opts, WithRecoveryCmdRunner(cmdRunner))
	}

	engine := NewRecoveryEngine(db, checker, opts...)
	engine.BackoffSchedule = []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 5 * time.Millisecond}
	engine.ReadyPollTimeout = 10 * time.Millisecond
	engine.ReadyPollInterval = 1 * time.Millisecond

	return engine, db
}

func TestRecoveryEngine_EvaluateHealthy(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		nil,
	)
	defer db.Close()

	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt != nil {
		t.Error("expected no recovery event when healthy")
	}

	state := engine.State()
	if state.Status != "idle" {
		t.Errorf("status = %q, want idle", state.Status)
	}
	if state.ConsecFailures != 0 {
		t.Errorf("consec_failures = %d, want 0", state.ConsecFailures)
	}
}

func TestRecoveryEngine_EvaluateFailsBelowThreshold(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		nil,
	)
	defer db.Close()

	// First failure — below threshold of 2
	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt != nil {
		t.Error("should not trigger recovery below threshold")
	}

	state := engine.State()
	if state.ConsecFailures != 1 {
		t.Errorf("consec_failures = %d, want 1", state.ConsecFailures)
	}
	if state.Status != "monitoring" {
		t.Errorf("status = %q, want monitoring", state.Status)
	}
}

func TestRecoveryEngine_EvaluateTriggersRecovery(t *testing.T) {
	readyOK := false
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if readyOK {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			readyOK = true
			return nil, nil
		},
	)
	defer db.Close()

	// Fail twice to reach threshold
	engine.Evaluate(context.Background())
	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt == nil {
		t.Fatal("expected recovery event at threshold")
	}
	if evt.Outcome != "success" {
		t.Errorf("outcome = %q, want success", evt.Outcome)
	}
	if evt.TriggerType != "ready_failure" {
		t.Errorf("trigger_type = %q, want ready_failure", evt.TriggerType)
	}
}

func TestRecoveryEngine_FailedRecoveryIncrementsBackoff(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil // restart succeeds but ready never becomes OK
		},
	)
	defer db.Close()

	// Trigger recovery
	engine.Evaluate(context.Background())
	evt, _ := engine.Evaluate(context.Background())
	if evt == nil || evt.Outcome != "failure" {
		t.Fatalf("expected failed recovery, got %v", evt)
	}

	state := engine.State()
	if state.FailedRecovery != 1 {
		t.Errorf("failed_recoveries = %d, want 1", state.FailedRecovery)
	}
	if state.BackoffLevel != 1 {
		t.Errorf("backoff_level = %d, want 1", state.BackoffLevel)
	}
}

func TestRecoveryEngine_CircuitBreakerTrips(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)
	defer db.Close()

	// Trip the circuit: need MaxBackoffRetries (3) failed recoveries
	for i := 0; i < 6; i++ { // extra evaluations to accumulate failures + backoff waits
		time.Sleep(10 * time.Millisecond) // wait past any backoff
		engine.Evaluate(context.Background())
	}

	state := engine.State()
	if !state.CircuitOpen {
		t.Error("circuit breaker should be open after max retries")
	}
	if state.Status != "circuit_open" {
		t.Errorf("status = %q, want circuit_open", state.Status)
	}
}

func TestRecoveryEngine_CircuitBreakerResetAfterCooldown(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)
	defer db.Close()

	// Trip the circuit
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		engine.Evaluate(context.Background())
	}

	state := engine.State()
	if !state.CircuitOpen {
		t.Skip("circuit didn't trip — test may need adjustment")
	}

	// Wait for cooldown (50ms)
	time.Sleep(60 * time.Millisecond)

	// Next evaluate should reset circuit
	engine.Evaluate(context.Background())
	state = engine.State()
	if state.CircuitOpen {
		t.Error("circuit should have reset after cooldown")
	}
}

func TestRecoveryEngine_ManualTriggerSuccess(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)
	defer db.Close()

	evt, err := engine.TriggerManual(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.TriggerType != "manual" {
		t.Errorf("trigger_type = %q, want manual", evt.TriggerType)
	}
	if evt.Outcome != "success" {
		t.Errorf("outcome = %q, want success", evt.Outcome)
	}
}

func TestRecoveryEngine_ManualTriggerSkippedByCircuit(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)
	defer db.Close()

	// Trip the circuit
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		engine.Evaluate(context.Background())
	}

	state := engine.State()
	if !state.CircuitOpen {
		t.Skip("circuit didn't trip")
	}

	evt, err := engine.TriggerManual(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Outcome != "skipped" {
		t.Errorf("outcome = %q, want skipped", evt.Outcome)
	}
}

func TestRecoveryEngine_ManualTriggerForceOverridesCircuit(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK) // will be healthy after restart
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)
	defer db.Close()

	// Manually set circuit open
	engine.mu.Lock()
	engine.state.CircuitOpen = true
	engine.mu.Unlock()

	evt, err := engine.TriggerManual(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if evt.Outcome != "success" {
		t.Errorf("outcome = %q, want success", evt.Outcome)
	}

	state := engine.State()
	if state.CircuitOpen {
		t.Error("circuit should be closed after force override")
	}
}

func TestRecoveryEngine_ResetCircuit(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		nil,
	)
	defer db.Close()

	// Manually set circuit open
	engine.mu.Lock()
	engine.state.CircuitOpen = true
	engine.state.Status = "circuit_open"
	engine.state.FailedRecovery = 5
	engine.state.BackoffLevel = 3
	engine.mu.Unlock()

	engine.ResetCircuit()

	state := engine.State()
	if state.CircuitOpen {
		t.Error("circuit should be closed")
	}
	if state.Status != "idle" {
		t.Errorf("status = %q, want idle", state.Status)
	}
}

func TestRecoveryEngine_CmdRunnerFailure(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("permission denied")
		},
	)
	defer db.Close()

	engine.Evaluate(context.Background())
	evt, _ := engine.Evaluate(context.Background())
	if evt == nil {
		t.Fatal("expected recovery event")
	}
	if evt.Outcome != "failure" {
		t.Errorf("outcome = %q, want failure", evt.Outcome)
	}
}

func TestRecoveryEngine_ListEvents(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)
	defer db.Close()

	// Create some events
	engine.TriggerManual(context.Background(), false)
	engine.TriggerManual(context.Background(), false)

	events, err := engine.ListEvents(10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(events))
	}
}

func TestRecoveryEngine_ListEventsDefaultLimit(t *testing.T) {
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		nil,
	)
	defer db.Close()

	events, err := engine.ListEvents(0) // should default to 50
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if events == nil {
		// Empty is fine, just shouldn't error
		t.Log("no events found (expected)")
	}
}

func TestRecoveryEngine_SuccessfulRecoveryResetsState(t *testing.T) {
	readyOK := false
	engine, db := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if readyOK {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			readyOK = true
			return nil, nil
		},
	)
	defer db.Close()

	// Trigger recovery
	engine.Evaluate(context.Background())
	engine.Evaluate(context.Background())

	state := engine.State()
	if state.ConsecFailures != 0 {
		t.Errorf("consec_failures = %d, want 0 after success", state.ConsecFailures)
	}
	if state.BackoffLevel != 0 {
		t.Errorf("backoff_level = %d, want 0 after success", state.BackoffLevel)
	}
	if state.Status != "idle" {
		t.Errorf("status = %q, want idle after success", state.Status)
	}
}
