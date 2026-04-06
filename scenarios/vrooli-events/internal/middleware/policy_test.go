package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/headers"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// [REQ:DI-004] Receiver-side policy middleware tests
// [REQ:DI-005] Graceful degradation tests

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestPolicyMiddleware_NoRules_AllowsAll(t *testing.T) {
	pm := NewPolicyMiddleware(Config{
		ScenarioName: "my-svc",
	})

	handler := pm.Handler(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestPolicyMiddleware_DenyRule_Returns403(t *testing.T) {
	pm := NewPolicyMiddleware(Config{
		ScenarioName: "my-svc",
		InitialRules: []policy.Rule{
			{
				ID: 1, RuleType: policy.RuleTypeAccess,
				SourceScenario: "bad-actor", TargetScenario: "my-svc",
				Effect: policy.EffectDeny, Enabled: true,
			},
		},
	})

	handler := pm.Handler(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	headers.InjectSource(req, "bad-actor")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	var resp PolicyDeniedResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.RuleID != 1 {
		t.Fatalf("expected rule ID 1, got %d", resp.RuleID)
	}
}

func TestPolicyMiddleware_AllowRule_Passes(t *testing.T) {
	pm := NewPolicyMiddleware(Config{
		ScenarioName: "my-svc",
		InitialRules: []policy.Rule{
			{
				ID: 1, RuleType: policy.RuleTypeAccess,
				SourceScenario: "good-svc", TargetScenario: "my-svc",
				Effect: policy.EffectAllow, Enabled: true,
			},
		},
	})

	handler := pm.Handler(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	headers.InjectSource(req, "good-svc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestPolicyMiddleware_NoSourceHeader_TreatedAsExternal(t *testing.T) {
	pm := NewPolicyMiddleware(Config{
		ScenarioName: "my-svc",
		InitialRules: []policy.Rule{
			{
				ID: 1, RuleType: policy.RuleTypeAccess,
				SourceScenario: "external", TargetScenario: "my-svc",
				Effect: policy.EffectDeny, Enabled: true,
			},
		},
	})

	handler := pm.Handler(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	// No X-Source-Scenario header
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for external request, got %d", rec.Code)
	}
}

func TestPolicyMiddleware_WildcardSource_MatchesAll(t *testing.T) {
	pm := NewPolicyMiddleware(Config{
		ScenarioName: "my-svc",
		InitialRules: []policy.Rule{
			{
				ID: 1, RuleType: policy.RuleTypeAccess,
				SourceScenario: "*", TargetScenario: "*",
				Effect: policy.EffectDeny, Enabled: true,
			},
		},
	})

	handler := pm.Handler(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	headers.InjectSource(req, "any-service")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 with wildcard deny, got %d", rec.Code)
	}
}

func TestPolicyMiddleware_DisabledRule_Skipped(t *testing.T) {
	pm := NewPolicyMiddleware(Config{
		ScenarioName: "my-svc",
		InitialRules: []policy.Rule{
			{
				ID: 1, RuleType: policy.RuleTypeAccess,
				SourceScenario: "bad-actor", TargetScenario: "my-svc",
				Effect: policy.EffectDeny, Enabled: false,
			},
		},
	})

	handler := pm.Handler(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	headers.InjectSource(req, "bad-actor")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with disabled rule, got %d", rec.Code)
	}
}

func TestPolicyMiddleware_UpdateRules_Applied(t *testing.T) {
	pm := NewPolicyMiddleware(Config{
		ScenarioName: "my-svc",
	})

	handler := pm.Handler(okHandler())

	// First request — no rules → allow
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	headers.InjectSource(req, "test-svc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 initially, got %d", rec.Code)
	}

	// Add deny rule
	pm.UpdateRules([]policy.Rule{
		{
			ID: 2, RuleType: policy.RuleTypeAccess,
			SourceScenario: "test-svc", TargetScenario: "my-svc",
			Effect: policy.EffectDeny, Enabled: true,
		},
	})

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	headers.InjectSource(req, "test-svc")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 after update, got %d", rec.Code)
	}
}

func TestPolicyMiddleware_ApplyEvent_CRUD(t *testing.T) {
	pm := NewPolicyMiddleware(Config{
		ScenarioName: "my-svc",
	})

	// Create
	r := policy.Rule{
		ID: 1, RuleType: policy.RuleTypeAccess,
		SourceScenario: "test", TargetScenario: "my-svc",
		Effect: policy.EffectDeny, Enabled: true,
	}
	pm.ApplyEvent(policy.PolicyEvent{Action: "created", RuleID: 1, Rule: &r})
	h := pm.Health()
	if h.RuleCount != 1 {
		t.Fatalf("expected 1 rule, got %d", h.RuleCount)
	}

	// Update to allow
	r2 := r
	r2.Effect = policy.EffectAllow
	pm.ApplyEvent(policy.PolicyEvent{Action: "updated", RuleID: 1, Rule: &r2})

	// Delete
	pm.ApplyEvent(policy.PolicyEvent{Action: "deleted", RuleID: 1})
	h = pm.Health()
	if h.RuleCount != 0 {
		t.Fatalf("expected 0 rules, got %d", h.RuleCount)
	}
}

func TestPolicyMiddleware_Health_StaleDetection(t *testing.T) {
	pm := NewPolicyMiddleware(Config{
		ScenarioName: "my-svc",
	})

	// Initially connected
	h := pm.Health()
	if h.CacheStale {
		t.Fatal("expected not stale initially")
	}
	if !h.SSEConnected {
		t.Fatal("expected connected initially")
	}

	// Disconnect SSE but recent update — not stale yet
	pm.SetSSEConnected(false)
	h = pm.Health()
	if h.CacheStale {
		t.Fatal("expected not stale with recent cache update")
	}

	// Simulate old cache
	pm.cacheUpdatedAt.Store(time.Now().Add(-2 * time.Minute))
	h = pm.Health()
	if !h.CacheStale {
		t.Fatal("expected stale with old cache and disconnected SSE")
	}
}

func TestPolicyMiddlewareFunc_NoURL_Noop(t *testing.T) {
	mw := PolicyMiddlewareFunc("", "my-svc")
	handler := mw(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with noop middleware, got %d", rec.Code)
	}
}

func TestPolicyMiddlewareFunc_WithURL_CreatesMiddleware(t *testing.T) {
	mw := PolicyMiddlewareFunc("http://localhost:17654", "my-svc")
	handler := mw(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// No rules → should allow
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestPolicyMiddleware_EndpointPattern(t *testing.T) {
	pm := NewPolicyMiddleware(Config{
		ScenarioName: "my-svc",
		InitialRules: []policy.Rule{
			{
				ID: 1, RuleType: policy.RuleTypeAccess,
				SourceScenario: "caller", TargetScenario: "my-svc",
				EndpointPattern: "/api/v1/secret",
				Effect:          policy.EffectDeny, Enabled: true,
			},
		},
	})

	handler := pm.Handler(okHandler())

	// Request to denied endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/v1/secret", nil)
	headers.InjectSource(req, "caller")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for denied endpoint, got %d", rec.Code)
	}

	// Request to different endpoint — no matching rule → allow
	req = httptest.NewRequest(http.MethodGet, "/api/v1/public", nil)
	headers.InjectSource(req, "caller")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for non-matching endpoint, got %d", rec.Code)
	}
}

func TestPolicyMiddleware_RateLimitRule_Ignored(t *testing.T) {
	pm := NewPolicyMiddleware(Config{
		ScenarioName: "my-svc",
		InitialRules: []policy.Rule{
			{
				ID: 1, RuleType: policy.RuleTypeRateLimit,
				SourceScenario: "*", TargetScenario: "*",
				Effect: policy.EffectDeny, Enabled: true,
			},
		},
	})

	handler := pm.Handler(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// rate_limit rules are not access rules — should be skipped
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (rate limit rule ignored), got %d", rec.Code)
	}
}
