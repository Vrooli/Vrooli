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

// decayHalfLife is the effective half-life the read path uses. It defaults to
// DecayHalfLife and may be overridden ONCE at boot (before serving traffic)
// via SetDecayHalfLife — it is not safe to mutate concurrently with reads.
var decayHalfLife = DecayHalfLife

// SetDecayHalfLife overrides the effective confidence half-life. It is a
// boot-time lever (WEB_SEARCH_DECAY_HALF_LIFE in main.go): non-positive values
// are ignored. It returns the effective half-life after the call.
func SetDecayHalfLife(d time.Duration) time.Duration {
	if d > 0 {
		decayHalfLife = d
	}
	return decayHalfLife
}

// EffectiveDecayHalfLife returns the half-life currently applied by
// EffectiveConfidence (and from which the GC derives its default min-age).
func EffectiveDecayHalfLife() time.Duration { return decayHalfLife }

const (
	// UsageGracePeriod is how long a never-surfaced finding is left unpenalized:
	// a fresh finding simply has not had a chance to be surfaced yet, so its
	// usage factor stays 1.0 until it is older than this.
	UsageGracePeriod = 30 * 24 * time.Hour // ~1 month
	// UsageHalfLife is the half-life of the usage penalty applied to a
	// never-surfaced finding once it is past the grace period.
	UsageHalfLife = 90 * 24 * time.Hour // ~3 months
	// UsageFloor is the lowest the usage factor can drive a finding: usage
	// telemetry only DOWN-WEIGHTS a never-surfaced claim, it never zeroes it out
	// on its own (curation still goes through the confidence + supersede gate).
	UsageFloor = 0.5
)

// UsageFactor is the OT-P2-001 usage signal in [UsageFloor, 1]: a finding that
// has ever been surfaced (or explicitly used) is "proven" and keeps factor 1.0;
// a never-surfaced finding keeps 1.0 until it is older than UsageGracePeriod,
// then decays on UsageHalfLife toward UsageFloor. It is pure (no storage
// mutation), mirroring the age-decay design.
func UsageFactor(u Usage, f Finding, now time.Time) float64 {
	if u.SurfacedCount > 0 || u.UsedCount > 0 {
		return 1.0
	}
	age := Age(f, now)
	if age <= UsageGracePeriod {
		return 1.0
	}
	over := age - UsageGracePeriod
	halfLives := over.Seconds() / UsageHalfLife.Seconds()
	factor := math.Pow(0.5, halfLives)
	if factor < UsageFloor {
		return UsageFloor
	}
	return factor
}

// EffectiveScore blends age-decayed confidence with the usage factor — the
// trust signal the effectiveness surface displays and the GC eligibility check
// reads: effective = EffectiveConfidence × UsageFactor. A never-surfaced,
// fully-decayed finding scores lowest; a recently-surfaced, fresh one scores
// highest. Pure; storage is never mutated.
func EffectiveScore(f Finding, u Usage, now time.Time) float64 {
	return EffectiveConfidence(f, now) * UsageFactor(u, f, now)
}

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
	halfLives := age.Seconds() / decayHalfLife.Seconds()
	factor := math.Pow(0.5, halfLives)
	return clampConfidence(f.Confidence * factor)
}
