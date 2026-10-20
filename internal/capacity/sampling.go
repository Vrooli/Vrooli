package capacity

import (
	"context"
	"math"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

const (
	halfLifeDecayBase = 0.5
)

// DecayedPeak folds a new observed sample into a decaying high-water mark
// (contract C2 — VRAM is non-compressible, so we size to the peak, not an
// average). The prior peak decays geometrically — losing half its value every
// halflife — but the result is NEVER below the new observed sample (a real
// working-set reading always wins). A first sample (zero prior / unknown prevAt)
// yields exactly the observed value.
func DecayedPeak(prevPeak, observed int64, prevAt, now time.Time, halflife time.Duration) int64 {
	decayed := prevPeak
	if halflife > 0 && !prevAt.IsZero() && now.After(prevAt) {
		factor := math.Pow(halfLifeDecayBase, float64(now.Sub(prevAt))/float64(halflife))
		decayed = int64(float64(prevPeak) * factor)
	}
	if decayed < 0 {
		decayed = 0
	}
	if observed > decayed {
		return observed
	}
	return decayed
}

// SampleObservedUsage records per-claim observed GPU usage + the decaying peak for
// every active VRAM claim in `claims`, using the live snapshot and attributor
// (§Phase 2). It is best-effort telemetry: a RecordObserved error on one claim is
// skipped (the next sweep retries), never fatal. The sample NEVER feeds Decide
// (contract C1). Returns the number of claims sampled.
func SampleObservedUsage(ctx context.Context, store ClaimRepository, claims []CapacityClaim, snapshot hostinventory.Snapshot, attr Attributor, policy Policy, now time.Time) int {
	return SampleObservedUsageWithFootprints(ctx, store, claims, snapshot, attr, policy, now, nil)
}

type FootprintIdentity struct {
	Resource    string
	Rung        string
	TunablesKey string
	GPUIndex    int
	SeedBytes   int64
}

type FootprintResolver func(CapacityClaim) (FootprintIdentity, error)

func SampleObservedUsageWithFootprints(ctx context.Context, store ClaimRepository, claims []CapacityClaim, snapshot hostinventory.Snapshot, attr Attributor, policy Policy, now time.Time, resolve FootprintResolver) int {
	if attr == nil {
		attr = unknownAttributor{}
	}
	return sampleClaimUsage(ctx, store, claims, snapshot, observeClaimUsage(ctx, claims, snapshot, attr), policy, now, resolve)
}

func sampleClaimUsage(ctx context.Context, store ClaimRepository, claims []CapacityClaim, snapshot hostinventory.Snapshot, usage claimUsage, policy Policy, now time.Time, resolve FootprintResolver) int {
	halflife := policy.ObservedPeakHalflife
	if halflife <= 0 {
		halflife = DefaultObservedPeakHalflife
	}
	sampled := 0
	for _, claim := range claims {
		if !IsActiveClaimStatus(claim.Status) || claim.ResourceKind != ResourceKindVRAM {
			continue
		}
		workers := usage.assigned[claim.ClaimID]
		if usage.uncertain.Contains(claim.ClaimID) || (len(workers) == 0 && !gpuProcessAbsenceVerified(snapshot, claim, now)) {
			continue
		}
		var observed int64
		for _, proc := range workers {
			observed += int64(proc.UsedBytes)
		}
		var prevAt time.Time
		if claim.ObservedAt != nil {
			prevAt = *claim.ObservedAt
		}
		peak := DecayedPeak(claim.ObservedPeakBytes, observed, prevAt, now, halflife)
		var absence *AbsenceProof
		if observed == 0 {
			absence = &AbsenceProof{Generation: claim.Generation, ObservedAt: snapshot.FieldProvenance["gpu_processes"].ObservedAt}
		}
		if updated, err := store.RecordObserved(ctx, claim.ClaimID, observed, peak, now, absence); err != nil || !IsActiveClaimStatus(updated.Status) {
			continue
		}
		if footprints, ok := store.(FootprintRepository); ok && resolve != nil {
			identity, err := resolve(claim)
			if err == nil {
				_, _ = footprints.RecordFootprint(ctx, FootprintObservation{
					Resource: identity.Resource, Rung: identity.Rung, TunablesKey: identity.TunablesKey,
					GPUIndex: identity.GPUIndex, Bytes: identity.SeedBytes,
					Source: FootprintSourceManifestDefault, ObservedAt: now,
				})
				if observed > 0 {
					_, _ = footprints.RecordFootprint(ctx, FootprintObservation{
						Resource: identity.Resource, Rung: identity.Rung, TunablesKey: identity.TunablesKey,
						GPUIndex: identity.GPUIndex, Bytes: observed,
						Source: FootprintSourceMeasured, ObservedAt: now,
					})
				}
			}
		}
		sampled++
	}
	return sampled
}
