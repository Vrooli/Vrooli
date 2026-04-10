package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/match"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
)

// AccessMiddleware enforces access control rules on incoming requests.
// It checks cached access rules (sorted by priority/specificity) and returns
// 403 for denied requests.
type AccessMiddleware struct {
	scenarioName string
	mu           sync.RWMutex
	rules        []policy.Rule // only access rules, sorted by priority DESC
}

// NewAccessMiddleware creates an access control middleware layer.
func NewAccessMiddleware(scenarioName string) *AccessMiddleware {
	return &AccessMiddleware{
		scenarioName: scenarioName,
		rules:        []policy.Rule{},
	}
}

// UpdateRules replaces the cached access rules. Thread-safe.
func (am *AccessMiddleware) UpdateRules(rules []policy.Rule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules = rules
}

// Wrap returns an http.Handler middleware that enforces access control.
func (am *AccessMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source := extractSource(r)
		decision := am.evaluate(source, r.URL.Path)
		if !decision.Allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(PolicyDeniedResponse{
				Error:  "policy_denied",
				RuleID: decision.RuleID,
				Reason: decision.Reason,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// evaluate checks cached access rules against the source and endpoint.
func (am *AccessMiddleware) evaluate(source, endpoint string) policy.Decision {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if len(am.rules) == 0 {
		return policy.Decision{Allowed: true, Reason: "no rules, default allow"}
	}

	for _, rule := range am.rules {
		if !rule.Enabled || rule.RuleType != policy.RuleTypeAccess {
			continue
		}
		if !matchesScenario(rule.SourceScenario, source) {
			continue
		}
		if !matchesScenario(rule.TargetScenario, am.scenarioName) {
			continue
		}
		if rule.EndpointPattern != "" && !matchEndpoint(rule.EndpointPattern, endpoint) {
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

// matchesScenario checks whether a glob pattern matches a scenario name.
func matchesScenario(pattern, name string) bool {
	if pattern == "*" || pattern == "**" {
		return true
	}
	return match.Glob(pattern, name)
}

// matchEndpoint checks whether an endpoint pattern matches a path.
// Endpoint patterns use "/" as separator, but the match package uses ".".
func matchEndpoint(pattern, endpoint string) bool {
	if pattern == "" {
		return true
	}
	patDot := strings.ReplaceAll(strings.TrimPrefix(pattern, "/"), "/", ".")
	endDot := strings.ReplaceAll(strings.TrimPrefix(endpoint, "/"), "/", ".")
	return match.Glob(patDot, endDot)
}
