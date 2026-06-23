package capacity

import (
	"testing"
	"time"
)

func TestRecommendFlagsOverReservation(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	policy := DefaultPolicy() // 20% headroom

	// whisper reserves 8 GiB but peaked at 1.4 GiB → suggest peak+20%, floored.
	c := CapacityClaim{
		ClaimID: "clm-w", OwnerID: "whisper", OwnerKind: OwnerKindResource,
		ResourceKind: ResourceKindVRAM, Status: StatusGranted, Priority: PriorityInteractive,
		PreferredBytes: 8 * gib, FloorBytes: 2 * gib,
		ObservedPeakBytes: 7 * gib / 5, ObservedAt: &now, // ~1.4 GiB
	}
	recs := Recommend([]CapacityClaim{c}, policy)
	if len(recs) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(recs))
	}
	r := recs[0]
	// Suggested must NEVER be below peak + headroom.
	minSuggest := c.ObservedPeakBytes + c.ObservedPeakBytes*20/100
	if r.SuggestedBytes < minSuggest {
		t.Errorf("suggested %d below peak+headroom %d", r.SuggestedBytes, minSuggest)
	}
	// ...and not below the floor, and below the reservation.
	if r.SuggestedBytes < c.FloorBytes {
		t.Errorf("suggested %d below floor %d", r.SuggestedBytes, c.FloorBytes)
	}
	if r.SuggestedBytes >= c.PreferredBytes {
		t.Errorf("suggested %d not below reservation %d", r.SuggestedBytes, c.PreferredBytes)
	}
}

func TestRecommendSilentWithoutSamples(t *testing.T) {
	policy := DefaultPolicy()
	// No ObservedAt / zero peak → no recommendation regardless of over-reservation.
	c := CapacityClaim{
		ClaimID: "clm", OwnerID: "whisper", ResourceKind: ResourceKindVRAM,
		Status: StatusGranted, PreferredBytes: 8 * gib, FloorBytes: 2 * gib,
	}
	if recs := Recommend([]CapacityClaim{c}, policy); len(recs) != 0 {
		t.Fatalf("must stay silent without observed-peak data, got %+v", recs)
	}
}

func TestRecommendSilentWhenRightSized(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	policy := DefaultPolicy()
	// Peak ~= reservation → peak+headroom exceeds preferred → no recommendation.
	c := CapacityClaim{
		ClaimID: "clm", OwnerID: "reranker", ResourceKind: ResourceKindVRAM,
		Status: StatusGranted, PreferredBytes: 2 * gib, FloorBytes: 1 * gib,
		ObservedPeakBytes: 2 * gib, ObservedAt: &now,
	}
	if recs := Recommend([]CapacityClaim{c}, policy); len(recs) != 0 {
		t.Fatalf("a right-sized claim must not be flagged, got %+v", recs)
	}
}
