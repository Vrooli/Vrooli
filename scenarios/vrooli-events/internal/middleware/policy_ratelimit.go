// DOC: docs/guides/managing-policies.md
package middleware

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// RateLimitMiddleware enforces token bucket rate limiting per rule.
type RateLimitMiddleware struct {
	scenarioName string
	mu           sync.RWMutex
	rules        []policy.Rule // only rate_limit rules
	bucketsMu    sync.Mutex
	buckets      map[int64]*bucket
	nowFunc      func() time.Time // overridable for tests
}

type bucket struct {
	tokens   float64
	lastFill time.Time
	capacity float64
	fillRate float64 // tokens per second
}

// RateLimitResponse is the JSON body returned on 429.
type RateLimitResponse struct {
	Error      string `json:"error"`
	RuleID     int64  `json:"rule_id"`
	Reason     string `json:"reason"`
	RetryAfter int    `json:"retry_after"`
}

// NewRateLimitMiddleware creates a rate limiting middleware layer.
func NewRateLimitMiddleware(scenarioName string) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		scenarioName: scenarioName,
		rules:        []policy.Rule{},
		buckets:      make(map[int64]*bucket),
		nowFunc:      time.Now,
	}
}

// UpdateRules replaces the cached rate limit rules and prunes orphaned state.
func (rl *RateLimitMiddleware) UpdateRules(rules []policy.Rule) {
	rl.mu.Lock()
	rl.rules = rules
	rl.mu.Unlock()

	activeIDs := make(map[int64]bool, len(rules))
	for _, r := range rules {
		activeIDs[r.ID] = true
	}
	rl.pruneState(activeIDs)
}

// Wrap returns an http.Handler middleware that enforces rate limiting.
func (rl *RateLimitMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source := extractSource(r)
		if decision, ok := rl.check(source, r.URL.Path); !ok {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", strconv.Itoa(decision.RetryAfter))
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(RateLimitResponse{
				Error:      "rate_limited",
				RuleID:     decision.RuleID,
				Reason:     decision.Reason,
				RetryAfter: decision.RetryAfter,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// check evaluates rate limit rules. Returns (decision, allowed).
func (rl *RateLimitMiddleware) check(source, endpoint string) (policy.Decision, bool) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	for _, rule := range rl.rules {
		if !rule.Enabled || rule.RuleType != policy.RuleTypeRateLimit {
			continue
		}
		if rule.MaxRequests <= 0 || rule.WindowSeconds <= 0 {
			continue
		}
		if !matchesScenario(rule.SourceScenario, source) {
			continue
		}
		if !matchesScenario(rule.TargetScenario, rl.scenarioName) {
			continue
		}
		if rule.EndpointPattern != "" && !matchEndpoint(rule.EndpointPattern, endpoint) {
			continue
		}

		// Found a matching rate limit rule — check the bucket.
		allowed, retryAfter := rl.consume(rule)
		if !allowed {
			return policy.Decision{
				Allowed:    false,
				RuleID:     rule.ID,
				RuleType:   policy.RuleTypeRateLimit,
				Reason:     "token bucket exhausted",
				RetryAfter: retryAfter,
			}, false
		}
	}

	return policy.Decision{Allowed: true}, true
}

// consume tries to consume a token from the bucket for the given rule.
// Returns (allowed, retryAfterSeconds).
func (rl *RateLimitMiddleware) consume(rule policy.Rule) (bool, int) {
	rl.bucketsMu.Lock()
	defer rl.bucketsMu.Unlock()

	now := rl.nowFunc()

	b, exists := rl.buckets[rule.ID]
	if !exists {
		capacity := float64(rule.MaxRequests + rule.BurstAllowance)
		fillRate := float64(rule.MaxRequests) / float64(rule.WindowSeconds)
		b = &bucket{
			tokens:   capacity,
			lastFill: now,
			capacity: capacity,
			fillRate: fillRate,
		}
		rl.buckets[rule.ID] = b
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastFill).Seconds()
	if elapsed > 0 {
		b.tokens = math.Min(b.capacity, b.tokens+elapsed*b.fillRate)
		b.lastFill = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}

	// Calculate retry-after: seconds until 1 token is available
	deficit := 1 - b.tokens
	retryAfter := int(math.Ceil(deficit / b.fillRate))
	if retryAfter < 1 {
		retryAfter = 1
	}
	return false, retryAfter
}

// pruneState removes buckets for rule IDs not in activeIDs.
func (rl *RateLimitMiddleware) pruneState(activeIDs map[int64]bool) {
	rl.bucketsMu.Lock()
	defer rl.bucketsMu.Unlock()

	for id := range rl.buckets {
		if !activeIDs[id] {
			delete(rl.buckets, id)
		}
	}
}
