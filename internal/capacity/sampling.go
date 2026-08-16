package capacity

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
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
		factor := math.Pow(0.5, float64(now.Sub(prevAt))/float64(halflife))
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

// observedBytesForOwner sums the GPU memory of every observed process that
// attributes to the claim's owner on the claim's GPU (mirrors the reconcile /
// sweep owner-matching). Zero when nothing attributes to the owner — an honest
// "idle / not resident right now" reading.
func observedBytesForOwner(ctx context.Context, claim CapacityClaim, snapshot hostinventory.Snapshot, attr Attributor) int64 {
	owner := strings.TrimSpace(claim.OwnerID)
	if owner == "" {
		return 0
	}
	var sum int64
	for _, proc := range snapshot.GPUProcesses {
		idx := proc.GPUIndex
		if !sameGPU(claim.GPUIndex, &idx) {
			continue
		}
		a := attr.Attribute(ctx, proc.PID)
		for _, cand := range []string{a.OwnerID, NormalizeOwnerName(a.ContainerName), strings.TrimPrefix(a.ContainerName, "/"), NormalizeProcessOwner(proc.ProcessName)} {
			cand = strings.TrimSpace(cand)
			if cand == "" || cand == OwnerUnknown {
				continue
			}
			if ownerMatches(owner, cand) {
				sum += int64(proc.UsedBytes)
				break
			}
		}
	}
	return sum
}

// SampleObservedUsage records per-claim observed GPU usage + the decaying peak for
// every active VRAM claim in `claims`, using the live snapshot and attributor
// (§Phase 2). It is best-effort telemetry: a RecordObserved error on one claim is
// skipped (the next sweep retries), never fatal. The sample NEVER feeds Decide
// (contract C1). Returns the number of claims sampled.
func SampleObservedUsage(ctx context.Context, store ClaimRepository, claims []CapacityClaim, snapshot hostinventory.Snapshot, attr Attributor, policy Policy, now time.Time) int {
	if attr == nil {
		attr = unknownAttributor{}
	}
	halflife := policy.ObservedPeakHalflife
	if halflife <= 0 {
		halflife = DefaultObservedPeakHalflife
	}
	sampled := 0
	for _, claim := range claims {
		if !IsActiveClaimStatus(claim.Status) || claim.ResourceKind != ResourceKindVRAM {
			continue
		}
		observed := observedBytesForOwner(ctx, claim, snapshot, attr)
		var prevAt time.Time
		if claim.ObservedAt != nil {
			prevAt = *claim.ObservedAt
		}
		peak := DecayedPeak(claim.ObservedPeakBytes, observed, prevAt, now, halflife)
		if _, err := store.RecordObserved(ctx, claim.ClaimID, observed, peak, now); err != nil {
			continue
		}
		sampled++
	}
	return sampled
}
