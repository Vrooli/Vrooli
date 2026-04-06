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

func TestRateLimitMiddleware_NoRules_AllowsAll(t *testing.T) {
	rl := NewRateLimitMiddleware("my-svc")
	handler := rl.Wrap(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_ExhaustsBucket(t *testing.T) {
	rl := NewRateLimitMiddleware("my-svc")

	// Freeze time
	now := time.Now()
	rl.nowFunc = func() time.Time { return now }

	rl.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeRateLimit,
			SourceScenario: "*", TargetScenario: "*",
			MaxRequests: 2, WindowSeconds: 10, BurstAllowance: 0,
			Enabled: true,
		},
	})
	handler := rl.Wrap(okHandler())

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	var resp RateLimitResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.RuleID != 1 {
		t.Fatalf("expected rule ID 1, got %d", resp.RuleID)
	}
	if resp.RetryAfter < 1 {
		t.Fatalf("expected retry_after >= 1, got %d", resp.RetryAfter)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestRateLimitMiddleware_RefillsOverTime(t *testing.T) {
	rl := NewRateLimitMiddleware("my-svc")

	now := time.Now()
	rl.nowFunc = func() time.Time { return now }

	rl.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeRateLimit,
			SourceScenario: "*", TargetScenario: "*",
			MaxRequests: 1, WindowSeconds: 1, BurstAllowance: 0,
			Enabled: true,
		},
	})
	handler := rl.Wrap(okHandler())

	// Consume the single token
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Second request at same time should fail
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	// Advance time by 1 second (full refill)
	now = now.Add(time.Second)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 after refill, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_BurstAllowance(t *testing.T) {
	rl := NewRateLimitMiddleware("my-svc")

	now := time.Now()
	rl.nowFunc = func() time.Time { return now }

	rl.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeRateLimit,
			SourceScenario: "*", TargetScenario: "*",
			MaxRequests: 2, WindowSeconds: 10, BurstAllowance: 3,
			Enabled: true,
		},
	})
	handler := rl.Wrap(okHandler())

	// Should allow 5 requests (2 base + 3 burst)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// 6th should fail
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after burst exhaustion, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_SourcePatternMatching(t *testing.T) {
	rl := NewRateLimitMiddleware("my-svc")

	now := time.Now()
	rl.nowFunc = func() time.Time { return now }

	rl.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeRateLimit,
			SourceScenario: "bad-caller", TargetScenario: "my-svc",
			MaxRequests: 1, WindowSeconds: 10, BurstAllowance: 0,
			Enabled: true,
		},
	})
	handler := rl.Wrap(okHandler())

	// bad-caller: limited
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	headers.InjectSource(req, "bad-caller")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for first request, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	headers.InjectSource(req, "bad-caller")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for bad-caller, got %d", rec.Code)
	}

	// good-caller: not matched, should pass
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	headers.InjectSource(req, "good-caller")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for good-caller, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_PruneState(t *testing.T) {
	rl := NewRateLimitMiddleware("my-svc")

	now := time.Now()
	rl.nowFunc = func() time.Time { return now }

	rl.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeRateLimit,
			SourceScenario: "*", TargetScenario: "*",
			MaxRequests: 1, WindowSeconds: 10,
			Enabled: true,
		},
	})

	// Consume a token to create bucket state
	handler := rl.Wrap(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Verify bucket exists
	rl.bucketsMu.Lock()
	if _, exists := rl.buckets[1]; !exists {
		t.Fatal("expected bucket for rule 1")
	}
	rl.bucketsMu.Unlock()

	// Update rules without rule 1
	rl.UpdateRules([]policy.Rule{})

	// Verify bucket was pruned
	rl.bucketsMu.Lock()
	if _, exists := rl.buckets[1]; exists {
		t.Fatal("expected bucket for rule 1 to be pruned")
	}
	rl.bucketsMu.Unlock()
}

func TestRateLimitMiddleware_DisabledRule_Skipped(t *testing.T) {
	rl := NewRateLimitMiddleware("my-svc")
	rl.UpdateRules([]policy.Rule{
		{
			ID: 1, RuleType: policy.RuleTypeRateLimit,
			SourceScenario: "*", TargetScenario: "*",
			MaxRequests: 0, WindowSeconds: 1,
			Enabled: false,
		},
	})
	handler := rl.Wrap(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
