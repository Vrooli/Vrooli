package capacity

import "fmt"

const (
	recommendPercentScale = 100
)

// Recommendation is one advisory right-sizing suggestion (§Phase 4, contract C7).
// It is NEVER auto-applied — it is a signal a human/operator acts on, comparing a
// claim's declared reservation against what it actually peaked at.
type Recommendation struct {
	Class             string `json:"class"`
	ClaimID           string `json:"claim_id"`
	OwnerKind         string `json:"owner_kind"`
	OwnerID           string `json:"owner_id"`
	PriorityTier      string `json:"priority_tier"`
	PreferredBytes    int64  `json:"preferred_bytes"`
	ObservedPeakBytes int64  `json:"observed_peak_bytes"`
	FloorBytes        int64  `json:"floor_bytes"`
	SuggestedBytes    int64  `json:"suggested_bytes"`
	SavingsBytes      int64  `json:"savings_bytes"`
	ExcessBytes       int64  `json:"excess_bytes"`
	Message           string `json:"message"`
}

// Recommend compares each active VRAM claim's declared preferred_bytes against
// its decaying observed peak and emits a right-sizing suggestion when the peak
// plus headroom is materially below the reservation. Guarantees (contract C7):
//
//   - SILENT without data: a claim with no recorded sample (observed_at unset, or
//     a zero peak) is skipped — a missing or single idle reading must never shrink
//     a reservation.
//   - NEVER below peak+headroom: suggested = max(observed_peak * (1 + headroom%),
//     floor_bytes), so the suggestion always carries the safety margin.
//   - advisory only: this returns suggestions; nothing is applied.
func Recommend(claims []CapacityClaim, policy Policy) []Recommendation {
	pct := policy.RecommendHeadroomPct
	if pct < 0 {
		pct = DefaultRecommendHeadroomPct
	}
	var out []Recommendation
	for _, c := range claims {
		if !IsActiveClaimStatus(c.Status) || c.ResourceKind != ResourceKindVRAM {
			continue
		}
		if c.ObservedAt == nil || c.ObservedPeakBytes <= 0 || c.PreferredBytes <= 0 {
			continue // no usable sample yet — stay silent
		}
		suggested := c.ObservedPeakBytes + c.ObservedPeakBytes*int64(pct)/recommendPercentScale
		if suggested < c.FloorBytes {
			suggested = c.FloorBytes
		}
		// Only flag a genuine, material over-reservation: the suggestion (with its
		// headroom) must sit below the declared preferred amount.
		if suggested >= c.PreferredBytes {
			continue
		}
		out = append(out, Recommendation{
			Class:             "over_reservation",
			ClaimID:           c.ClaimID,
			OwnerKind:         c.OwnerKind,
			OwnerID:           c.OwnerID,
			PriorityTier:      PriorityTierName(c.Priority),
			PreferredBytes:    c.PreferredBytes,
			ObservedPeakBytes: c.ObservedPeakBytes,
			FloorBytes:        c.FloorBytes,
			SuggestedBytes:    suggested,
			SavingsBytes:      c.PreferredBytes - suggested,
			Message: fmt.Sprintf("%q reserves %s but peaked at %s; consider ~%s (peak + %d%% headroom) to free %s",
				c.OwnerID, humanBytes(c.PreferredBytes), humanBytes(c.ObservedPeakBytes), humanBytes(suggested), pct, humanBytes(c.PreferredBytes-suggested)),
		})
	}
	return out
}

// RecommendWithFootprints adds the dangerous opposite direction to Recommend:
// a durable measured high-water mark above the declared reservation. The
// footprint identity is resolved from the same rung, tunables, and GPU key used
// when sampling, so observations from another configuration cannot bleed in.
func RecommendWithFootprints(claims []CapacityClaim, footprints []Footprint, policy Policy, resolve FootprintResolver) []Recommendation {
	out := Recommend(claims, policy)
	if resolve == nil {
		return out
	}
	pct := policy.RecommendHeadroomPct
	if pct < 0 {
		pct = DefaultRecommendHeadroomPct
	}
	for _, claim := range claims {
		if !IsActiveClaimStatus(claim.Status) || claim.ResourceKind != ResourceKindVRAM || claim.PreferredBytes <= 0 {
			continue
		}
		identity, err := resolve(claim)
		if err != nil {
			continue
		}
		footprint, ok := measuredFootprintForIdentity(footprints, identity)
		if !ok || footprint.PeakBytes <= claim.PreferredBytes {
			continue
		}
		suggested := footprint.PeakBytes + footprint.PeakBytes*int64(pct)/recommendPercentScale
		out = append(out, Recommendation{
			Class:             "over_consumption",
			ClaimID:           claim.ClaimID,
			OwnerKind:         claim.OwnerKind,
			OwnerID:           claim.OwnerID,
			PriorityTier:      PriorityTierName(claim.Priority),
			PreferredBytes:    claim.PreferredBytes,
			ObservedPeakBytes: footprint.PeakBytes,
			FloorBytes:        claim.FloorBytes,
			SuggestedBytes:    suggested,
			ExcessBytes:       footprint.PeakBytes - claim.PreferredBytes,
			Message: fmt.Sprintf("%q is granted %s but its durable peak is %s; consider at least ~%s (peak + %d%% headroom)",
				claim.OwnerID, humanBytes(claim.PreferredBytes), humanBytes(footprint.PeakBytes), humanBytes(suggested), pct),
		})
	}
	return out
}

func measuredFootprintForIdentity(footprints []Footprint, identity FootprintIdentity) (Footprint, bool) {
	for _, footprint := range footprints {
		if footprint.Resource == identity.Resource && footprint.Rung == identity.Rung &&
			footprint.TunablesKey == identity.TunablesKey && footprint.GPUIndex == identity.GPUIndex &&
			footprint.Source == FootprintSourceMeasured && footprint.Samples > 0 {
			return footprint, true
		}
	}
	return Footprint{}, false
}
