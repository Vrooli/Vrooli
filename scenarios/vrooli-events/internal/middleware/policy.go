// Package middleware provides importable HTTP middleware for receiver-side
// policy enforcement using locally-cached vrooli-events policies.
//
// [REQ:DI-004] Receiver-side policy middleware
// [REQ:DI-005] Graceful degradation
package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
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

// PolicyMiddleware enforces access control on incoming requests using
// locally-cached policy rules. The X-Source-Scenario header identifies
// the caller; requests without it are classified as "external".
type PolicyMiddleware struct {
	scenarioName string
	defaultMode  fallback.Mode

	mu    sync.RWMutex
	rules []policy.Rule

	cacheUpdatedAt atomic.Value // time.Time
	sseConnected   atomic.Bool
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
		rules:        rules,
	}
	pm.cacheUpdatedAt.Store(time.Now())
	pm.sseConnected.Store(true) // assume connected initially
	return pm
}

// Handler returns an http.Handler middleware that enforces policy.
func (pm *PolicyMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source := headers.ExtractSource(r)
		if source == "" {
			source = "external"
		}

		decision := pm.evaluate(source, r.URL.Path)
		if !decision.Allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(PolicyDeniedResponse{
				Error:  "policy_denied",
				RuleID: decision.RuleID,
				Reason: decision.Reason,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// UpdateRules replaces the cached rules. Thread-safe.
func (pm *PolicyMiddleware) UpdateRules(rules []policy.Rule) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.rules = rules
	pm.cacheUpdatedAt.Store(time.Now())
}

// ApplyEvent applies a single policy change event from SSE.
func (pm *PolicyMiddleware) ApplyEvent(evt policy.PolicyEvent) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cacheUpdatedAt.Store(time.Now())

	switch evt.Action {
	case "deleted":
		for i, rule := range pm.rules {
			if rule.ID == evt.RuleID {
				pm.rules = append(pm.rules[:i], pm.rules[i+1:]...)
				return
			}
		}
	case "created":
		if evt.Rule != nil {
			pm.rules = append(pm.rules, *evt.Rule)
		}
	case "updated":
		if evt.Rule != nil {
			for i, rule := range pm.rules {
				if rule.ID == evt.RuleID {
					pm.rules[i] = *evt.Rule
					return
				}
			}
			pm.rules = append(pm.rules, *evt.Rule)
		}
	}
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
	pm.mu.RLock()
	ruleCount := len(pm.rules)
	pm.mu.RUnlock()

	updatedAt, _ := pm.cacheUpdatedAt.Load().(time.Time)
	age := time.Since(updatedAt)
	connected := pm.sseConnected.Load()
	stale := !connected && age > 60*time.Second

	return HealthInfo{
		CacheAge:     age,
		CacheStale:   stale,
		RuleCount:    ruleCount,
		SSEConnected: connected,
	}
}

// evaluate checks cached rules against the source and endpoint.
func (pm *PolicyMiddleware) evaluate(source, endpoint string) policy.Decision {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.rules) == 0 {
		return policy.Decision{Allowed: true, Reason: "no rules, default allow"}
	}

	for _, rule := range pm.rules {
		if !rule.Enabled || rule.RuleType != policy.RuleTypeAccess {
			continue
		}
		// Match source scenario
		if rule.SourceScenario != "*" && rule.SourceScenario != source {
			continue
		}
		// Match target (this scenario)
		if rule.TargetScenario != "*" && rule.TargetScenario != pm.scenarioName {
			continue
		}
		// Match endpoint if specified
		if rule.EndpointPattern != "" && rule.EndpointPattern != endpoint {
			continue
		}

		allowed := rule.Effect == policy.EffectAllow
		reason := "matched rule"
		if !allowed {
			reason = "denied by access control rule"
		}
		return policy.Decision{
			Allowed:  allowed,
			RuleID:   rule.ID,
			RuleType: rule.RuleType,
			Reason:   reason,
		}
	}

	return policy.Decision{Allowed: true, Reason: "no matching rule, default allow"}
}

// PolicyMiddlewareFunc is the convenience constructor matching the spec:
// func PolicyMiddleware(eventsURL, scenarioName string) func(http.Handler) http.Handler
//
// It creates a middleware with default settings. For production use with
// custom config, use NewPolicyMiddleware directly.
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
