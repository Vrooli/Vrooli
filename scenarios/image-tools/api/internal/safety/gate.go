package safety

import "context"

// Gate bundles the resolved policy, the consent audit log, and the abuse rate
// limiter into the single object the AI submit edge enforces. It is the seam the
// handler holds; constructing it once at boot (from the resolved tier) keeps the
// edge code a thin "evaluate → act" sequence.
type Gate struct {
	policy  Policy
	log     *ConsentLog
	limiter *RateLimiter
}

// NewGate builds the gate for a deployment tier, backed by the consent log
// (nil-safe — a nil log just skips audit writes).
func NewGate(tier Tier, log *ConsentLog) *Gate {
	p := PolicyFor(tier)
	return &Gate{policy: p, log: log, limiter: NewRateLimiter(p.RateLimitPerMin)}
}

// Policy returns the resolved policy (for discovery / telemetry).
func (g *Gate) Policy() Policy { return g.policy }

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

// AllowRate reports whether the tier's abuse throttle permits one more submit
// (always true on the local tier / when unlimited).
func (g *Gate) AllowRate() bool { return g.limiter.Allow() }

// RecordConsent appends a consent-affirmation audit row for an allowed
// high-weight op. Best-effort: the caller logs but does not fail on an error.
func (g *Gate) RecordConsent(ctx context.Context, op string, weight Weight) error {
	return g.log.Record(ctx, op, weight, g.policy.Tier)
}
