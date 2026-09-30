// Package middleware provides importable HTTP middleware for receiver-side
// policy enforcement using locally-cached vrooli-events policies.
//
// [REQ:DI-004] Receiver-side policy middleware
// [REQ:DI-005] Graceful degradation
//
// DOC: docs/guides/integrating-a-scenario.md
// DOC: docs/guides/managing-policies.md
package middleware

import (
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/fallback"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/headers"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// Config configures the PolicyMiddleware.
type Config struct {
	EventsURL    string        // vrooli-events base URL for fetching policies
	ScenarioName string        // this scenario's identity (the receiver)
	DefaultMode  fallback.Mode // fail-open or fail-closed (default: fail-open)
	InitialRules []policy.Rule // initial policy snapshot (for testing, may be nil)
}

// PolicyMiddleware enforces access control, rate limiting, and circuit breaking
// on incoming requests using locally-cached policy rules.
type PolicyMiddleware struct {
	scenarioName string
	defaultMode  fallback.Mode

	access       *AccessMiddleware
	rateLimit    *RateLimitMiddleware
	circuitBreak *CircuitBreakerMiddleware

	cacheUpdatedAt atomic.Value // time.Time
	sseConnected   atomic.Bool
	ruleCount      atomic.Int64
}

// PolicyDeniedResponse is the JSON body returned on 403.
type PolicyDeniedResponse struct {
	Error  string `json:"error"`
	RuleID int64  `json:"rule_id"`
	Reason string `json:"reason"`
}

// NewPolicyMiddleware creates a receiver-side policy middleware.
// It starts with initialRules (or empty cache if nil).
func NewPolicyMiddleware(cfg Config) *PolicyMiddleware {
	mode := cfg.DefaultMode
	if mode == "" {
		mode = fallback.ModeFailOpen
	}
	rules := cfg.InitialRules
	if rules == nil {
		rules = []policy.Rule{}
	}

	pm := &PolicyMiddleware{
		scenarioName: cfg.ScenarioName,
		defaultMode:  mode,
		access:       NewAccessMiddleware(cfg.ScenarioName),
		rateLimit:    NewRateLimitMiddleware(cfg.ScenarioName),
		circuitBreak: NewCircuitBreakerMiddleware(cfg.ScenarioName),
	}
	pm.cacheUpdatedAt.Store(time.Now())
	pm.sseConnected.Store(true)
	pm.distributeRules(rules)
	return pm
}

// Handler returns an http.Handler middleware that enforces the full policy chain:
// access control -> rate limiting -> circuit breaking.
func (pm *PolicyMiddleware) Handler(next http.Handler) http.Handler {
	return pm.access.Wrap(pm.rateLimit.Wrap(pm.circuitBreak.Wrap(next)))
}

// UpdateRules replaces all cached rules. Thread-safe.
func (pm *PolicyMiddleware) UpdateRules(rules []policy.Rule) {
	pm.distributeRules(rules)
	pm.cacheUpdatedAt.Store(time.Now())
}

// ApplyEvent applies a single policy change event from SSE.
func (pm *PolicyMiddleware) ApplyEvent(evt policy.PolicyEvent) {
	// Reconstruct a full snapshot from access middleware's current rules.
	// This is a legacy path; snapshot push replaces it.
	pm.access.mu.Lock()
	current := make([]policy.Rule, len(pm.access.rules))
	copy(current, pm.access.rules)
	pm.access.mu.Unlock()

	switch evt.Action {
	case "deleted":
		for i, rule := range current {
			if rule.ID == evt.RuleID {
				current = append(current[:i], current[i+1:]...)
				break
			}
		}
	case "created":
		if evt.Rule != nil {
			current = append(current, *evt.Rule)
		}
	case "updated":
		if evt.Rule != nil {
			found := false
			for i, rule := range current {
				if rule.ID == evt.RuleID {
					current[i] = *evt.Rule
					found = true
					break
				}
			}
			if !found {
				current = append(current, *evt.Rule)
			}
		}
	}

	pm.distributeRules(current)
	pm.cacheUpdatedAt.Store(time.Now())
}

// SetSSEConnected updates the SSE connection status.
func (pm *PolicyMiddleware) SetSSEConnected(connected bool) {
	pm.sseConnected.Store(connected)
}

// HealthInfo returns cache health information.
type HealthInfo struct {
	CacheAge     time.Duration `json:"policy_cache_age_ms"`
	CacheStale   bool          `json:"policy_cache_stale"`
	RuleCount    int           `json:"rule_count"`
	SSEConnected bool          `json:"sse_connected"`
}

// Health returns cache health info for the health endpoint.
func (pm *PolicyMiddleware) Health() HealthInfo {
	updatedAt, _ := pm.cacheUpdatedAt.Load().(time.Time)
	age := time.Since(updatedAt)
	connected := pm.sseConnected.Load()
	stale := !connected && age > 60*time.Second

	return HealthInfo{
		CacheAge:     age,
		CacheStale:   stale,
		RuleCount:    int(pm.ruleCount.Load()),
		SSEConnected: connected,
	}
}

// distributeRules fans out rules to each middleware layer and prunes orphaned state.
func (pm *PolicyMiddleware) distributeRules(rules []policy.Rule) {
	pm.ruleCount.Store(int64(len(rules)))

	var accessRules []policy.Rule
	var rateLimitRules []policy.Rule
	var circuitRules []policy.Rule
	for _, r := range rules {
		switch r.RuleType {
		case policy.RuleTypeAccess:
			accessRules = append(accessRules, r)
		case policy.RuleTypeRateLimit:
			rateLimitRules = append(rateLimitRules, r)
		case policy.RuleTypeCircuitBreaker:
			circuitRules = append(circuitRules, r)
		}
	}
	pm.access.UpdateRules(accessRules)
	pm.rateLimit.UpdateRules(rateLimitRules)
	pm.circuitBreak.UpdateRules(circuitRules)
}

// PolicyMiddlewareFunc is the convenience constructor matching the spec:
// func PolicyMiddleware(eventsURL, scenarioName string) func(http.Handler) http.Handler
func PolicyMiddlewareFunc(eventsURL, scenarioName string) func(http.Handler) http.Handler {
	if eventsURL == "" {
		log.Printf("[policy-middleware] no events URL, using noop middleware")
		return fallback.NoopMiddleware()
	}

	pm := NewPolicyMiddleware(Config{
		EventsURL:    eventsURL,
		ScenarioName: scenarioName,
		DefaultMode:  fallback.ModeFailOpen,
	})
	return pm.Handler
}

// extractSource reads X-Source-Scenario from the request, defaulting to "external".
func extractSource(r *http.Request) string {
	source := headers.ExtractSource(r)
	if source == "" {
		return "external"
	}
	return source
}
