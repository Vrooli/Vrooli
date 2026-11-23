package safety

import (
	"context"
	"strings"
)

// Gate bundles the resolved policy, the consent audit log, and the abuse rate
// limiter into the single object the AI submit edge enforces. It is the seam the
// handler holds; constructing it once at boot (from the resolved tier) keeps the
// edge code a thin "evaluate → act" sequence.
type Gate struct {
	policy  Policy
	log     *ConsentLog
	limiter *RateLimiter
	minor   MinorReviewer
	faces   FaceCounter
}

// NewGateFromPolicy builds the gate from an already-resolved policy (which may
// carry per-control overrides). Prefer this over NewGate so the IMAGE_TOOLS_*
// control overrides actually take effect.
func NewGateFromPolicy(p Policy, log *ConsentLog) *Gate {
	return &Gate{policy: p, log: log, limiter: NewRateLimiter(p.RateLimitPerMin)}
}

// NewGate builds the gate for a deployment tier using the tier preset only
// (no per-control overrides). Kept for tests and callers that want the preset.
func NewGate(tier Tier, log *ConsentLog) *Gate {
	return NewGateFromPolicy(PolicyFor(tier), log)
}

// Policy returns the resolved policy (for discovery / telemetry).
func (g *Gate) Policy() Policy { return g.policy }

// SetMinorReviewer attaches the deployment's minor-safety reviewer. The resolved
// policy is updated in the same call so discovery and enforcement cannot
// disagree about whether the required control is actually present.
func (g *Gate) SetMinorReviewer(r MinorReviewer) {
	g.minor = r
	g.policy.MinorSafetyDetector = r != nil
}

// ReviewMinor enforces the non-configurable refusal on the images a request
// carries. It runs only when the policy requires it; a required control with no
// reviewer, a reviewer error, or a positive minor+sexual verdict all refuse. An
// unconfigured reviewer on a local deployment is not consulted at all.
func (g *Gate) ReviewMinor(ctx context.Context, op string, images [][]byte) Decision {
	if !g.policy.RequiresMinorReview(op) {
		return Decision{Allowed: true}
	}
	if g.minor == nil {
		return Decision{
			Allowed:            false,
			RequireMinorReview: true,
			Reason:             "this deployment requires a minor-safety reviewer for image operations, but none is configured",
			RecoveryHint:       "set " + MinorSafetyEnv + " to a program implementing the documented JSON review contract",
		}
	}
	verdict, err := g.minor.Review(ctx, images)
	if err != nil {
		return Decision{
			Allowed:            false,
			RequireMinorReview: true,
			Reason:             "the minor-safety review could not be completed: " + err.Error(),
			RecoveryHint:       "fix the configured reviewer; a deployment that serves other people fails closed when its required review is unavailable",
		}
	}
	if verdict.Minor && verdict.Sexual {
		detail := strings.TrimSpace(verdict.Detail)
		if detail == "" {
			detail = "sexual content involving a minor"
		}
		return Decision{
			Allowed:            false,
			RequireMinorReview: true,
			Reason:             "refused: " + detail,
		}
	}
	return Decision{Allowed: true, RequireMinorReview: true}
}

// Evaluate decides whether a submit for op may proceed (see Policy.Evaluate).
func (g *Gate) Evaluate(op string, consentAffirmed bool) Decision {
	return g.policy.Evaluate(op, consentAffirmed)
}

// EvaluateWeight decides whether a submit may proceed under an explicitly
// supplied consent weight — the conditioning weight-elevation seam (C4). See
// Policy.EvaluateWeight.
func (g *Gate) EvaluateWeight(weight Weight, op string, consentAffirmed bool) Decision {
	return g.policy.EvaluateWeight(weight, op, consentAffirmed)
}

// EvaluateRoute decides a submit once its route tier is resolved (see
// Policy.EvaluateRoute).
func (g *Gate) EvaluateRoute(weight Weight, op, tier string, consentAffirmed bool) Decision {
	return g.policy.EvaluateRoute(weight, op, tier, consentAffirmed)
}

// ConsentFor reports whether a request resolved to tier needs a consent
// affirmation (see Policy.ConsentFor).
func (g *Gate) ConsentFor(weight Weight, op, tier string) ConsentRequirement {
	return g.policy.ConsentFor(weight, op, tier)
}

// AllowRate reports whether the tier's abuse throttle permits one more submit
// (always true on the local tier / when unlimited).
func (g *Gate) AllowRate() bool { return g.limiter.Allow() }

// RecordConsent appends a consent-affirmation audit row for an allowed
// high-weight op. route names where the work was sent (tier and model, "" when
// the affirmation was taken before a route existed). Best-effort: the caller
// logs but does not fail on an error.
func (g *Gate) RecordConsent(ctx context.Context, op string, weight Weight, route string) error {
	return g.log.Record(ctx, op, weight, g.policy.Tier, route)
}
