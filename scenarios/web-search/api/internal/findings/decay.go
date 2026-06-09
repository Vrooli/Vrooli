package findings

import (
	"math"
	"time"
)

// DecayHalfLife is the age at which a finding's stored confidence has decayed to
// half on the read path. It is a trust-freshness signal only — confidence is
// never mutated in storage and findings are never hard-deleted by age. A
// finding's persisted confidence is its provenance-time confidence; the decayed
// value is computed on demand so an old, unrefreshed claim is trusted less
// without rewriting history.
const DecayHalfLife = 180 * 24 * time.Hour // ~6 months

// Age returns how old a finding is relative to now, measured from its retrieval
// date (when the claim was last gathered) falling back to created_at. A
// zero/future stamp yields a zero age.
func Age(f Finding, now time.Time) time.Duration {
	anchor := f.RetrievalDate
	if anchor.IsZero() {
		anchor = f.CreatedAt
	}
	if anchor.IsZero() {
		return 0
	}
	age := now.Sub(anchor)
	if age < 0 {
		return 0
	}
	return age
}

// EffectiveConfidence applies deterministic exponential age decay to a finding's
// stored confidence and returns the freshness-adjusted value in [0,1]:
//
//	effective = stored * 0.5^(age / DecayHalfLife)
//
// At age 0 it equals the stored confidence; at one half-life, half; and so on.
// It is pure (no storage mutation) so the read path can surface a freshness-
// aware confidence while the audit trail and stored value stay intact.
func EffectiveConfidence(f Finding, now time.Time) float64 {
	age := Age(f, now)
	if age <= 0 {
		return clampConfidence(f.Confidence)
	}
	halfLives := age.Seconds() / DecayHalfLife.Seconds()
	factor := math.Pow(0.5, halfLives)
	return clampConfidence(f.Confidence * factor)
}
