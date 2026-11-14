// DOC: docs/guides/managing-policies.md
// DOC: docs/internal/TEMPORAL-FLOWS.md
package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// CircuitBreakerMiddleware enforces circuit breaker rules.
// Each matching rule tracks failure counts and transitions between
// closed -> open -> half-open states.
type CircuitBreakerMiddleware struct {
	scenarioName string
	mu           sync.RWMutex
	rules        []policy.Rule // only circuit_breaker rules
	statesMu     sync.Mutex
	states       map[string]*circuitState
	nowFunc      func() time.Time
}

type circuitState struct {
	state        policy.CircuitState
	failures     int
	lastFailure  time.Time
	openedAt     time.Time
	probeAllowed bool
	ruleID       int64
	threshold    int
	cooldown     time.Duration
}

// CircuitBreakerResponse is the JSON body returned on 503.
type CircuitBreakerResponse struct {
	Error      string `json:"error"`
	RuleID     int64  `json:"rule_id"`
	Reason     string `json:"reason"`
	RetryAfter int    `json:"retry_after"`
}

// NewCircuitBreakerMiddleware creates a circuit breaker middleware layer.
func NewCircuitBreakerMiddleware(scenarioName string) *CircuitBreakerMiddleware {
	return &CircuitBreakerMiddleware{
		scenarioName: scenarioName,
		rules:        []policy.Rule{},
		states:       make(map[string]*circuitState),
		nowFunc:      time.Now,
	}
}

// UpdateRules replaces the cached circuit breaker rules and prunes orphaned state.
func (cb *CircuitBreakerMiddleware) UpdateRules(rules []policy.Rule) {
	cb.mu.Lock()
	cb.rules = rules
	cb.mu.Unlock()

	activeKeys := make(map[string]bool, len(rules))
	for _, r := range rules {
		activeKeys[routeKey(r.SourceScenario, r.TargetScenario, r.EndpointPattern)] = true
	}
	cb.pruneState(activeKeys)
}

// Wrap returns an http.Handler middleware that enforces circuit breaking.
func (cb *CircuitBreakerMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source := extractSource(r)

		rule, key := cb.findMatchingRule(source, r.URL.Path)
		if rule == nil {
			next.ServeHTTP(w, r)
			return
		}

		cs := cb.getOrCreateState(key, *rule)

		cb.statesMu.Lock()
		rejected, reason, retryAfter := cs.checkEntry(cb.nowFunc())
		cb.statesMu.Unlock()

		if rejected {
			writeCircuitOpenResponse(w, rule.ID, key, reason, retryAfter)
			return
		}

		// Wrap response to observe status code
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)

		cb.statesMu.Lock()
		cs.recordResult(sw.status >= 500, cb.nowFunc())
		cb.statesMu.Unlock()
	})
}

// checkEntry evaluates whether a request should be rejected based on the current
// circuit state. Must be called with statesMu held. Returns (rejected, reason, retryAfter).
func (cs *circuitState) checkEntry(now time.Time) (bool, string, int) {
	// Transition open -> half-open when cooldown expires
	if cs.state == policy.CircuitOpen && now.Sub(cs.openedAt) >= cs.cooldown {
		cs.state = policy.CircuitHalfOpen
		cs.probeAllowed = true
	}

	switch cs.state {
	case policy.CircuitOpen:
		cooldownLeft := int(cs.cooldown.Seconds()) - int(now.Sub(cs.openedAt).Seconds())
		return true, "open", cooldownLeft
	case policy.CircuitHalfOpen:
		if !cs.probeAllowed {
			return true, "half-open (probe in progress)", int(cs.cooldown.Seconds())
		}
		cs.probeAllowed = false // only one probe at a time
	}
	return false, "", 0
}

// recordResult updates the circuit state after a request completes.
// Must be called with statesMu held.
func (cs *circuitState) recordResult(failed bool, now time.Time) {
	if failed {
		cs.failures++
		cs.lastFailure = now
		if cs.state == policy.CircuitHalfOpen {
			cs.state = policy.CircuitOpen
			cs.openedAt = now
			cs.failures = 0
		} else if cs.failures >= cs.threshold {
			cs.state = policy.CircuitOpen
			cs.openedAt = now
			cs.failures = 0
		}
	} else if cs.state == policy.CircuitHalfOpen {
		cs.state = policy.CircuitClosed
		cs.failures = 0
	}
}

// findMatchingRule returns the first matching circuit breaker rule and its route key.
func (cb *CircuitBreakerMiddleware) findMatchingRule(source, endpoint string) (*policy.Rule, string) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	for i := range cb.rules {
		r := &cb.rules[i]
		if !r.Enabled || r.RuleType != policy.RuleTypeCircuitBreaker {
			continue
		}
		if !matchesScenario(r.SourceScenario, source) {
			continue
		}
		if !matchesScenario(r.TargetScenario, cb.scenarioName) {
			continue
		}
		if r.EndpointPattern != "" && !matchEndpoint(r.EndpointPattern, endpoint) {
			continue
		}
		return r, routeKey(r.SourceScenario, r.TargetScenario, r.EndpointPattern)
	}
	return nil, ""
}

func (cb *CircuitBreakerMiddleware) getOrCreateState(key string, rule policy.Rule) *circuitState {
	cb.statesMu.Lock()
	defer cb.statesMu.Unlock()

	cs, exists := cb.states[key]
	if !exists {
		cooldown := time.Duration(rule.CooldownSeconds) * time.Second
		if cooldown == 0 {
			cooldown = 30 * time.Second
		}
		cs = &circuitState{
			state:     policy.CircuitClosed,
			ruleID:    rule.ID,
			threshold: rule.FailureThreshold,
			cooldown:  cooldown,
		}
		cb.states[key] = cs
	}
	return cs
}

// pruneState removes circuit states for route keys not in activeKeys.
func (cb *CircuitBreakerMiddleware) pruneState(activeKeys map[string]bool) {
	cb.statesMu.Lock()
	defer cb.statesMu.Unlock()

	for key := range cb.states {
		if !activeKeys[key] {
			delete(cb.states, key)
		}
	}
}

// routeKey builds a unique key for a rule's source.target.endpoint combination.
func routeKey(source, target, endpoint string) string {
	if endpoint == "" {
		return source + "." + target
	}
	return source + "." + target + "." + endpoint
}

// writeCircuitOpenResponse sends a 503 with Retry-After and a JSON body
// describing the circuit breaker rejection.
func writeCircuitOpenResponse(w http.ResponseWriter, ruleID int64, key, stateDesc string, retryAfter int) {
	if retryAfter < 1 {
		retryAfter = 1
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(CircuitBreakerResponse{
		Error:      "circuit_open",
		RuleID:     ruleID,
		Reason:     fmt.Sprintf("circuit breaker %s for %s", stateDesc, key),
		RetryAfter: retryAfter,
	})
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.status = code
		sw.wroteHeader = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if !sw.wroteHeader {
		sw.status = http.StatusOK
		sw.wroteHeader = true
	}
	return sw.ResponseWriter.Write(b)
}
