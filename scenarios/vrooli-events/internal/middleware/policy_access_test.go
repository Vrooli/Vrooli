package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/headers"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

func TestAccessMiddleware_NoRules_AllowsAll(t *testing.T) {
	am := NewAccessMiddleware("my-svc")
	handler := am.Wrap(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAccessMiddleware_DenyRule_Returns403(t *testing.T) {
	am := NewAccessMiddleware("my-svc")
	am.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeAccess,
			SourceScenario: "bad-actor", TargetScenario: "my-svc",
			Effect: policy.EffectDeny, Enabled: true, Priority: 6,
		},
	})
	handler := am.Wrap(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	headers.InjectSource(req, "bad-actor")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestAccessMiddleware_GlobPatternMatching(t *testing.T) {
	am := NewAccessMiddleware("my-svc")
	// Segment-aware glob: "swarm.*" matches "swarm.manager" (dot-separated)
	// For scenario names (single segments), use "*" or exact match.
	// Test with a dot-segmented source name:
	am.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeAccess,
			SourceScenario: "swarm.*", TargetScenario: "my-svc",
			Effect: policy.EffectDeny, Enabled: true, Priority: 5,
		},
	})
	handler := am.Wrap(okHandler())

	// Should match glob (swarm.manager matches swarm.*)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	headers.InjectSource(req, "swarm.manager")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for glob match, got %d", rec.Code)
	}

	// Should NOT match glob (other.service doesn't match swarm.*)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	headers.InjectSource(req, "other.service")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for non-matching glob, got %d", rec.Code)
	}
}

func TestAccessMiddleware_EndpointGlobPattern(t *testing.T) {
	am := NewAccessMiddleware("my-svc")
	am.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeAccess,
			SourceScenario: "*", TargetScenario: "*",
			EndpointPattern: "/api/v1/*",
			Effect:          policy.EffectDeny, Enabled: true, Priority: 4,
		},
	})
	handler := am.Wrap(okHandler())

	// Endpoint matches glob
	req := httptest.NewRequest(http.MethodGet, "/api/v1/secret", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for endpoint glob match, got %d", rec.Code)
	}

	// Endpoint does not match (more segments)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/a/b", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for non-matching endpoint glob, got %d", rec.Code)
	}
}

func TestAccessMiddleware_SpecificityOrdering(t *testing.T) {
	am := NewAccessMiddleware("my-svc")
	// Higher priority (more specific) rule should match first.
	// Rules are expected pre-sorted by priority DESC.
	am.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeAccess,
			SourceScenario: "caller", TargetScenario: "my-svc",
			EndpointPattern: "/api/v1/specific",
			Effect:          policy.EffectAllow, Enabled: true,
			Priority: 9, // exact+exact+exact
		},
		{
			ID: 2, RuleType: policy.RuleTypeAccess,
			SourceScenario: "*", TargetScenario: "*",
			Effect: policy.EffectDeny, Enabled: true,
			Priority: 3, // wildcard+wildcard+wildcard(empty endpoint = 1pt per patternScore)
		},
	})
	handler := am.Wrap(okHandler())

	// Specific rule allows
	req := httptest.NewRequest(http.MethodGet, "/api/v1/specific", nil)
	headers.InjectSource(req, "caller")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for specific allow, got %d", rec.Code)
	}

	// Wildcard deny catches everything else
	req = httptest.NewRequest(http.MethodGet, "/api/v1/other", nil)
	headers.InjectSource(req, "caller")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for wildcard deny, got %d", rec.Code)
	}
}

func TestAccessMiddleware_SkipsNonAccessRules(t *testing.T) {
	am := NewAccessMiddleware("my-svc")
	am.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeRateLimit,
			SourceScenario: "*", TargetScenario: "*",
			Effect: policy.EffectDeny, Enabled: true,
		},
	})
	handler := am.Wrap(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (rate limit rule ignored), got %d", rec.Code)
	}
}
