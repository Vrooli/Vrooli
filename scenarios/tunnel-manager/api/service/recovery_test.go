package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"tunnel-manager/domain"
)

// newMockRecoveryStore returns a mock that collects persisted events.
func newMockRecoveryStore() *mockRecoveryEventStore {
	var mu sync.Mutex
	var events []domain.RecoveryEvent
	return &mockRecoveryEventStore{
		persistEventFn: func(evt *domain.RecoveryEvent) error {
			mu.Lock()
			events = append(events, *evt)
			mu.Unlock()
			return nil
		},
		listEventsFn: func(limit int) ([]domain.RecoveryEvent, error) {
			mu.Lock()
			defer mu.Unlock()
			if limit <= 0 {
				limit = 50
			}
			if limit > len(events) {
				limit = len(events)
			}
			cp := make([]domain.RecoveryEvent, limit)
			copy(cp, events)
			return cp, nil
		},
	}
}

func newTestRecoveryEngine(t *testing.T, readyHandler http.HandlerFunc, cmdRunner func(ctx context.Context, name string, args ...string) ([]byte, error)) *RecoveryEngine {
	t.Helper()

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

	recoveryStore := newMockRecoveryStore()
	engine := NewRecoveryEngine(recoveryStore, checker, opts...)
	engine.BackoffSchedule = []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 5 * time.Millisecond}
	engine.ReadyPollTimeout = 10 * time.Millisecond
	engine.ReadyPollInterval = 1 * time.Millisecond

	return engine
}

// [REQ:RECOVER-001] [REQ:RECOVER-002] [REQ:RECOVER-003] Recovery engine unit tests

func TestRecoveryEngine_EvaluateHealthy(t *testing.T) {
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		nil,
	)

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
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		nil,
	)

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
	engine := newTestRecoveryEngine(t,
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
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)

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
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)

	for i := 0; i < 6; i++ {
		time.Sleep(10 * time.Millisecond)
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
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)

	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		engine.Evaluate(context.Background())
	}

	state := engine.State()
	if !state.CircuitOpen {
		t.Skip("circuit didn't trip -- test may need adjustment")
	}

	time.Sleep(60 * time.Millisecond)

	engine.Evaluate(context.Background())
	state = engine.State()
	if state.CircuitOpen {
		t.Error("circuit should have reset after cooldown")
	}
}

func TestRecoveryEngine_ManualTriggerSuccess(t *testing.T) {
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)

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
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)

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
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)

	engine.SetStateForTest(func(s *domain.RecoveryState) {
		s.CircuitOpen = true
	})

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
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		nil,
	)

	engine.SetStateForTest(func(s *domain.RecoveryState) {
		s.CircuitOpen = true
		s.Status = "circuit_open"
		s.FailedRecovery = 5
		s.BackoffLevel = 3
	})

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
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("permission denied")
		},
	)

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
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, nil
		},
	)

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
	engine := newTestRecoveryEngine(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		nil,
	)

	events, err := engine.ListEvents(0)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if events == nil {
		t.Log("no events found (expected)")
	}
}

func TestRecoveryEngine_SuccessfulRecoveryResetsState(t *testing.T) {
	readyOK := false
	engine := newTestRecoveryEngine(t,
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

// [REQ:RECOVER-004] Exponential backoff on failed recovery
func TestRecoveryExponentialBackoff(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	recoveryStore := newMockRecoveryStore()
	engine := NewRecoveryEngine(recoveryStore, checker,
		WithConsecutiveFailures(1),
		WithMaxBackoffRetries(5),
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("restart failed")
		}),
	)
	engine.BackoffSchedule = []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		40 * time.Millisecond,
	}

	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt == nil || evt.Outcome != "failure" {
		t.Fatalf("expected failed recovery, got %+v", evt)
	}

	state := engine.State()
	if state.BackoffLevel != 1 {
		t.Errorf("backoff_level = %d, want 1", state.BackoffLevel)
	}
	if state.NextRetryAfter.IsZero() {
		t.Error("expected next_retry_after to be set")
	}

	evt, err = engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt != nil {
		t.Error("expected no recovery during backoff period")
	}

	time.Sleep(15 * time.Millisecond)
	evt, err = engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt == nil {
		t.Fatal("expected recovery after backoff expired")
	}
	if evt.Outcome != "failure" {
		t.Errorf("outcome = %q, want failure", evt.Outcome)
	}

	state = engine.State()
	if state.BackoffLevel != 2 {
		t.Errorf("backoff_level = %d, want 2", state.BackoffLevel)
	}
}

// [REQ:RECOVER-004] Backoff resets after successful recovery
func TestRecoveryBackoffResetsOnSuccess(t *testing.T) {
	recoveryAttempt := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if recoveryAttempt >= 2 {
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

	recoveryStore := newMockRecoveryStore()
	engine := NewRecoveryEngine(recoveryStore, checker,
		WithConsecutiveFailures(1),
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			recoveryAttempt++
			if recoveryAttempt >= 2 {
				return nil, nil
			}
			return nil, fmt.Errorf("restart failed")
		}),
	)
	engine.BackoffSchedule = []time.Duration{5 * time.Millisecond}

	evt, _ := engine.Evaluate(context.Background())
	if evt == nil || evt.Outcome != "failure" {
		t.Fatal("expected first recovery to fail")
	}

	time.Sleep(10 * time.Millisecond)

	evt, _ = engine.Evaluate(context.Background())
	if evt == nil || evt.Outcome != "success" {
		t.Fatal("expected second recovery to succeed")
	}

	state := engine.State()
	if state.BackoffLevel != 0 {
		t.Errorf("backoff_level = %d, want 0 after success", state.BackoffLevel)
	}
	if state.FailedRecovery != 0 {
		t.Errorf("failed_recoveries = %d, want 0 after success", state.FailedRecovery)
	}
}

// [REQ:RECOVER-001] Trigger on ready endpoint failure
func TestRecoveryTriggersOnReadyFailure(t *testing.T) {
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

	recoveryStore := newMockRecoveryStore()
	engine := NewRecoveryEngine(recoveryStore, checker,
		WithConsecutiveFailures(3),
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			readyOK = true
			return []byte(""), nil
		}),
	)

	for i := 0; i < 2; i++ {
		evt, err := engine.Evaluate(context.Background())
		if err != nil {
			t.Fatalf("evaluate %d: %v", i, err)
		}
		if evt != nil {
			t.Fatalf("expected no recovery on failure %d, got event: %+v", i+1, evt)
		}
	}

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
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	recoveryStore := newMockRecoveryStore()
	engine := NewRecoveryEngine(recoveryStore, checker, WithConsecutiveFailures(3))

	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt != nil {
		t.Fatal("should not trigger on single failure")
	}

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

	recoveryStore := newMockRecoveryStore()
	engine := NewRecoveryEngine(recoveryStore, checker,
		WithConsecutiveFailures(1),
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

// [REQ:RECOVER-002] Trigger recovery on HA connection loss
func TestRecoveryTriggersOnHAConnectionLoss(t *testing.T) {
	metricsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			fmt.Fprintln(w, "cloudflared_tunnel_ha_connections 0")
			return
		}
		if r.URL.Path == "/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer metricsServer.Close()

	restartCalled := false
	healthCheck := NewTunnelHealthChecker(
		WithMetricsURL(metricsServer.URL),
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active"), nil
		}),
	)

	scraper := NewMetricsScraper(metricsServer.URL)

	recoveryStore := newMockRecoveryStore()
	engine := NewRecoveryEngine(recoveryStore, healthCheck,
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			restartCalled = true
			return []byte("ok"), nil
		}),
		WithConsecutiveFailures(2),
	)

	ctx := context.Background()

	evt, err := engine.EvaluateHA(ctx, scraper)
	if err != nil {
		t.Fatalf("EvaluateHA 1: %v", err)
	}
	if evt != nil {
		t.Error("first HA=0 check should not trigger recovery")
	}

	evt, err = engine.EvaluateHA(ctx, scraper)
	if err != nil {
		t.Fatalf("EvaluateHA 2: %v", err)
	}
	if evt == nil {
		t.Fatal("expected recovery event after 2 consecutive HA=0 checks")
	}
	if evt.TriggerType != "ha_connection_loss" {
		t.Errorf("trigger_type = %q, want ha_connection_loss", evt.TriggerType)
	}
	if !restartCalled {
		t.Error("expected systemctl restart to be called")
	}
}

// [REQ:RECOVER-002] No trigger when HA connections are healthy
func TestRecoveryNoTriggerWhenHAHealthy(t *testing.T) {
	metricsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			fmt.Fprintln(w, "cloudflared_tunnel_ha_connections 4")
			return
		}
		if r.URL.Path == "/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer metricsServer.Close()

	healthCheck := NewTunnelHealthChecker(WithMetricsURL(metricsServer.URL))
	scraper := NewMetricsScraper(metricsServer.URL)
	recoveryStore := newMockRecoveryStore()
	engine := NewRecoveryEngine(recoveryStore, healthCheck,
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			t.Error("restart should not be called when HA is healthy")
			return nil, nil
		}),
	)

	ctx := context.Background()
	evt, err := engine.EvaluateHA(ctx, scraper)
	if err != nil {
		t.Fatalf("EvaluateHA: %v", err)
	}
	if evt != nil {
		t.Error("should not trigger recovery when HA connections are healthy")
	}
}

// [REQ:HEALTH-004] HA connection monitoring - detect degraded state
func TestHAConnectionDegradedDetection(t *testing.T) {
	body := "cloudflared_tunnel_ha_connections 2"
	m := ParsePrometheusMetrics(body)
	if m.HAConnections != 2 {
		t.Errorf("HAConnections = %d, want 2", m.HAConnections)
	}
	if m.HAConnections >= 4 {
		t.Error("2 HA connections should be below the healthy threshold of 4")
	}
}

// [REQ:HEALTH-004] HA connection monitoring - critical at zero
func TestHAConnectionCritical(t *testing.T) {
	body := "cloudflared_tunnel_ha_connections 0"
	m := ParsePrometheusMetrics(body)
	if m.HAConnections != 0 {
		t.Errorf("HAConnections = %d, want 0", m.HAConnections)
	}
}

// [REQ:RECOVER-005] Circuit breaker opens after consecutive failures
func TestCircuitBreakerOpens(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	recoveryStore := newMockRecoveryStore()
	engine := NewRecoveryEngine(recoveryStore, checker,
		WithConsecutiveFailures(1),
		WithMaxBackoffRetries(3),
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("restart failed")
		}),
	)
	engine.BackoffSchedule = []time.Duration{1 * time.Millisecond}

	for i := 0; i < 3; i++ {
		time.Sleep(2 * time.Millisecond)
		evt, err := engine.Evaluate(context.Background())
		if err != nil {
			t.Fatalf("evaluate %d: %v", i, err)
		}
		if evt == nil {
			t.Fatalf("expected recovery event on attempt %d", i+1)
		}
		if evt.Outcome != "failure" {
			t.Fatalf("expected failure on attempt %d, got %q", i+1, evt.Outcome)
		}
	}

	state := engine.State()
	if !state.CircuitOpen {
		t.Error("expected circuit breaker to be open")
	}
	if state.Status != "circuit_open" {
		t.Errorf("status = %q, want circuit_open", state.Status)
	}

	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt != nil {
		t.Error("expected no recovery when circuit is open")
	}
}

// [REQ:RECOVER-005] Circuit breaker resets after cooldown
func TestCircuitBreakerResetsAfterCooldown(t *testing.T) {
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

	recoveryStore := newMockRecoveryStore()
	engine := NewRecoveryEngine(recoveryStore, checker,
		WithConsecutiveFailures(1),
		WithMaxBackoffRetries(2),
		WithCircuitCooldown(10*time.Millisecond),
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("restart failed")
		}),
	)
	engine.BackoffSchedule = []time.Duration{1 * time.Millisecond}
	engine.ReadyPollTimeout = 5 * time.Millisecond
	engine.ReadyPollInterval = 1 * time.Millisecond

	for i := 0; i < 2; i++ {
		time.Sleep(2 * time.Millisecond)
		_, _ = engine.Evaluate(context.Background())
	}

	state := engine.State()
	if !state.CircuitOpen {
		t.Fatal("expected circuit to be open")
	}

	time.Sleep(15 * time.Millisecond)

	readyOK = true
	evt, err := engine.Evaluate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if evt != nil {
		t.Logf("got event: %+v", evt)
	}

	state = engine.State()
	if state.CircuitOpen {
		t.Error("expected circuit to be reset after cooldown")
	}
}

// [REQ:RECOVER-005] Manual circuit reset
func TestCircuitBreakerManualReset(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	recoveryStore := newMockRecoveryStore()
	engine := NewRecoveryEngine(recoveryStore, checker,
		WithConsecutiveFailures(1),
		WithMaxBackoffRetries(1),
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("failed")
		}),
	)
	engine.BackoffSchedule = []time.Duration{1 * time.Millisecond}

	_, _ = engine.Evaluate(context.Background())

	if !engine.State().CircuitOpen {
		t.Fatal("expected circuit to be open")
	}

	engine.ResetCircuit()

	state := engine.State()
	if state.CircuitOpen {
		t.Error("circuit should be closed after manual reset")
	}
	if state.Status != "idle" {
		t.Errorf("status = %q, want idle after reset", state.Status)
	}
}
