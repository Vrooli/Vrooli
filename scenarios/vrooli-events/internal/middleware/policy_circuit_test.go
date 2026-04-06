package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

func statusHandler(code int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
	})
}

func TestCircuitBreaker_NoRules_AllowsAll(t *testing.T) {
	cb := NewCircuitBreakerMiddleware("my-svc")
	handler := cb.Wrap(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCircuitBreaker_OpensOnThreshold(t *testing.T) {
	cb := NewCircuitBreakerMiddleware("my-svc")
	now := time.Now()
	cb.nowFunc = func() time.Time { return now }

	cb.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeCircuitBreaker,
			SourceScenario: "*", TargetScenario: "*",
			FailureThreshold: 2, CooldownSeconds: 30,
			Enabled: true,
		},
	})

	// Backend returns 500
	handler := cb.Wrap(statusHandler(500))

	// First failure
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	// Second failure — should trip the circuit
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	// Third request — circuit should be open, return 503
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	var resp CircuitBreakerResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "circuit_open" {
		t.Fatalf("expected circuit_open, got %s", resp.Error)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestCircuitBreaker_HalfOpenProbeSuccess(t *testing.T) {
	cb := NewCircuitBreakerMiddleware("my-svc")
	now := time.Now()
	cb.nowFunc = func() time.Time { return now }

	cb.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeCircuitBreaker,
			SourceScenario: "*", TargetScenario: "*",
			FailureThreshold: 1, CooldownSeconds: 10,
			Enabled: true,
		},
	})

	// Trip the circuit with 500
	handler500 := cb.Wrap(statusHandler(500))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler500.ServeHTTP(rec, req)

	// Verify circuit is open
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler500.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	// Advance past cooldown
	now = now.Add(11 * time.Second)

	// Probe with successful backend
	handlerOK := cb.Wrap(okHandler())
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handlerOK.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for probe, got %d", rec.Code)
	}

	// Circuit should be closed now — subsequent requests pass
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handlerOK.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 after circuit close, got %d", rec.Code)
	}
}

func TestCircuitBreaker_HalfOpenProbeFailure(t *testing.T) {
	cb := NewCircuitBreakerMiddleware("my-svc")
	now := time.Now()
	cb.nowFunc = func() time.Time { return now }

	cb.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeCircuitBreaker,
			SourceScenario: "*", TargetScenario: "*",
			FailureThreshold: 1, CooldownSeconds: 10,
			Enabled: true,
		},
	})

	// Trip the circuit
	handler := cb.Wrap(statusHandler(500))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Advance past cooldown
	now = now.Add(11 * time.Second)

	// Probe fails (still 500)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for failed probe, got %d", rec.Code)
	}

	// Circuit should be open again
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 after failed probe, got %d", rec.Code)
	}
}

func TestCircuitBreaker_4xxDoesNotTrip(t *testing.T) {
	cb := NewCircuitBreakerMiddleware("my-svc")
	now := time.Now()
	cb.nowFunc = func() time.Time { return now }

	cb.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeCircuitBreaker,
			SourceScenario: "*", TargetScenario: "*",
			FailureThreshold: 1, CooldownSeconds: 10,
			Enabled: true,
		},
	})

	// 4xx should NOT trip the circuit
	handler := cb.Wrap(statusHandler(400))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("request %d: expected 400, got %d", i+1, rec.Code)
		}
	}
}

func TestCircuitBreaker_PruneState(t *testing.T) {
	cb := NewCircuitBreakerMiddleware("my-svc")

	cb.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeCircuitBreaker,
			SourceScenario: "*", TargetScenario: "*",
			FailureThreshold: 5, CooldownSeconds: 10,
			Enabled: true,
		},
	})

	// Create some state
	handler := cb.Wrap(statusHandler(500))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	cb.statesMu.Lock()
	stateCount := len(cb.states)
	cb.statesMu.Unlock()
	if stateCount == 0 {
		t.Fatal("expected state to exist")
	}

	// Remove all rules
	cb.UpdateRules([]policy.Rule{})

	cb.statesMu.Lock()
	stateCount = len(cb.states)
	cb.statesMu.Unlock()
	if stateCount != 0 {
		t.Fatalf("expected states to be pruned, got %d", stateCount)
	}
}

func TestCircuitBreaker_DisabledRule_Skipped(t *testing.T) {
	cb := NewCircuitBreakerMiddleware("my-svc")
	cb.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeCircuitBreaker,
			SourceScenario: "*", TargetScenario: "*",
			FailureThreshold: 1, CooldownSeconds: 10,
			Enabled: false,
		},
	})

	handler := cb.Wrap(statusHandler(500))

	// Even with 500s, disabled rule should not trip
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 passthrough, got %d", rec.Code)
		}
	}
}
