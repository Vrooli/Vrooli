package capacity

import "fmt"

const (
	recommendParameterA = 100
)

// Recommendation is one advisory right-sizing suggestion (§Phase 4, contract C7).
// It is NEVER auto-applied — it is a signal a human/operator acts on, comparing a
// claim's declared reservation against what it actually peaked at.
type Recommendation struct {
	ClaimID           string `json:"claim_id"`
	OwnerKind         string `json:"owner_kind"`
	OwnerID           string `json:"owner_id"`
	PriorityTier      string `json:"priority_tier"`
	PreferredBytes    int64  `json:"preferred_bytes"`
	ObservedPeakBytes int64  `json:"observed_peak_bytes"`
	FloorBytes        int64  `json:"floor_bytes"`
	SuggestedBytes    int64  `json:"suggested_bytes"`
	SavingsBytes      int64  `json:"savings_bytes"`
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
		suggested := c.ObservedPeakBytes + c.ObservedPeakBytes*int64(pct)/recommendParameterA
		if suggested < c.FloorBytes {
			suggested = c.FloorBytes
		}
		// Only flag a genuine, material over-reservation: the suggestion (with its
		// headroom) must sit below the declared preferred amount.
		if suggested >= c.PreferredBytes {
			continue
		}
		out = append(out, Recommendation{
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
