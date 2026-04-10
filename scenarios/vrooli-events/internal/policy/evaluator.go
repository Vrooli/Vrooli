package policy

import (
	"context"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/match"
)

// EvalRequest contains the parameters for a policy evaluation.
type EvalRequest struct {
	Source   string
	Target   string
	Endpoint string
}

// Evaluator checks incoming requests against stored policy rules.
type Evaluator struct {
	store Store
}

// NewEvaluator creates a new policy evaluator.
func NewEvaluator(s Store) *Evaluator {
	return &Evaluator{store: s}
}

// Evaluate checks all enabled rules and returns a Decision.
// Evaluation order: access control (priority-ordered rule matching).
// Default policy is allow-all when no rules match.
func (e *Evaluator) Evaluate(ctx context.Context, req EvalRequest) Decision {
	rules, err := e.store.ListRules(ctx, ListFilters{
		RuleType: RuleTypeAccess,
	})
	if err != nil {
		return Decision{Allowed: true, Reason: "policy store unavailable, fail-open"}
	}

	// Rules are already sorted by priority DESC from the store.
	// Find the first matching enabled rule.
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		if !matchesScenario(r.SourceScenario, req.Source) {
			continue
		}
		if !matchesScenario(r.TargetScenario, req.Target) {
			continue
		}
		if r.EndpointPattern != "" && !matchEndpoint(r.EndpointPattern, req.Endpoint) {
			continue
		}

		allowed := r.Effect == EffectAllow
		decision := Decision{
			Allowed:  allowed,
			RuleID:   r.ID,
			RuleType: r.RuleType,
			Reason:   "matched rule",
		}
		if !allowed {
			decision.Reason = "denied by access control rule"
		}
		return decision
	}

	return Decision{Allowed: true, Reason: "no matching rule, default allow"}
}

// matchesScenario checks whether a pattern matches a scenario name.
// Supports glob patterns via the match package.
func matchesScenario(pattern, name string) bool {
	if pattern == "*" || pattern == "**" {
		return true
	}
	return match.Glob(pattern, name)
}

// matchEndpoint checks whether an endpoint pattern matches a path.
// Endpoint patterns use "/" as separator, but the match package uses ".".
// We convert both to "." for matching.
func matchEndpoint(pattern, endpoint string) bool {
	if pattern == "" {
		return true
	}
	// Convert / separators to . for segment-aware matching
	patDot := strings.ReplaceAll(strings.TrimPrefix(pattern, "/"), "/", ".")
	endDot := strings.ReplaceAll(strings.TrimPrefix(endpoint, "/"), "/", ".")
	return match.Glob(patDot, endDot)
}
