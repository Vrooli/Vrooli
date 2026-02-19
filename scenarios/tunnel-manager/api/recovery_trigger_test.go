package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:RECOVER-001] Trigger on ready endpoint failure
func TestRecoveryTriggersOnReadyFailure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	readyOK := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if readyOK {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	engine := NewRecoveryEngine(db, checker,
		WithConsecutiveFailures(3),
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			// Simulate successful restart — next health check will see ready
			readyOK = true
			return []byte(""), nil
		}),
	)

	// First two failures should NOT trigger recovery
	for i := 0; i < 2; i++ {
		evt, err := engine.Evaluate(context.Background())
		if err != nil {
			t.Fatalf("evaluate %d: %v", i, err)
		}
		if evt != nil {
			t.Fatalf("expected no recovery on failure %d, got event: %+v", i+1, evt)
		}
	}

	// Third consecutive failure SHOULD trigger recovery
	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatalf("evaluate 3: %v", err)
	}
	if evt == nil {
		t.Fatal("expected recovery event on 3rd consecutive failure")
	}
	if evt.TriggerType != "ready_failure" {
		t.Errorf("trigger_type = %q, want ready_failure", evt.TriggerType)
	}
	if evt.Outcome != "success" {
		t.Errorf("outcome = %q, want success", evt.Outcome)
	}
}

// [REQ:RECOVER-001] Single transient failure should not trigger recovery
func TestRecoverySingleFailureNoTrigger(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusServiceUnavailable) // one transient failure
		} else {
			w.WriteHeader(http.StatusOK) // then recovers
		}
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	engine := NewRecoveryEngine(db, checker, WithConsecutiveFailures(3))

	// One failure
	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt != nil {
		t.Fatal("should not trigger on single failure")
	}

	// Then healthy — should reset counter
	evt, err = engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt != nil {
		t.Fatal("should not trigger when healthy")
	}

	state := engine.State()
	if state.ConsecFailures != 0 {
		t.Errorf("consec_failures = %d, want 0 after healthy check", state.ConsecFailures)
	}
}

// [REQ:RECOVER-003] Recovery action: systemctl restart
func TestRecoveryActionSystemctlRestart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var restartCalled bool
	readyOK := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if readyOK {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	engine := NewRecoveryEngine(db, checker,
		WithConsecutiveFailures(1), // trigger immediately
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if name == "sudo" && len(args) >= 3 && args[0] == "systemctl" && args[1] == "restart" && args[2] == "cloudflared" {
				restartCalled = true
				readyOK = true
				return nil, nil
			}
			return nil, fmt.Errorf("unexpected command: %s %v", name, args)
		}),
	)

	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !restartCalled {
		t.Error("expected sudo systemctl restart cloudflared to be called")
	}
	if evt == nil {
		t.Fatal("expected recovery event")
	}
	if evt.Action != "systemctl_restart" {
		t.Errorf("action = %q, want systemctl_restart", evt.Action)
	}
}
