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

func TestRecommendWithFootprintsFlagsKokoroOverConsumption(t *testing.T) {
	policy := DefaultPolicy()
	claim := CapacityClaim{
		ClaimID: "clm-k", OwnerID: "kokoro", OwnerKind: OwnerKindResource,
		ResourceKind: ResourceKindVRAM, Status: StatusGranted, Priority: PriorityInteractive,
		PreferredBytes: 2 * gib, FloorBytes: gib,
	}
	identity := FootprintIdentity{Resource: "kokoro", Rung: "gpu", GPUIndex: 0}
	footprints := []Footprint{{
		Resource: "kokoro", Rung: "gpu", GPUIndex: 0, PeakBytes: 216 * gib / 100,
		Samples: 3, Source: FootprintSourceMeasured,
	}}
	recs := RecommendWithFootprints([]CapacityClaim{claim}, footprints, policy, func(CapacityClaim) (FootprintIdentity, error) {
		return identity, nil
	})
	if len(recs) != 1 || recs[0].Class != "over_consumption" {
		t.Fatalf("recommendations = %+v, want one over-consumption warning", recs)
	}
	if recs[0].ObservedPeakBytes != footprints[0].PeakBytes || recs[0].SuggestedBytes <= recs[0].ObservedPeakBytes {
		t.Fatalf("recommendation does not use durable peak plus headroom: %+v", recs[0])
	}
}

func TestRecommendWithFootprintsSilentWithoutMeasuredSample(t *testing.T) {
	claim := CapacityClaim{ClaimID: "clm", OwnerID: "kokoro", ResourceKind: ResourceKindVRAM, Status: StatusGranted, PreferredBytes: 2 * gib}
	identity := FootprintIdentity{Resource: "kokoro", Rung: "gpu"}
	resolve := func(CapacityClaim) (FootprintIdentity, error) { return identity, nil }
	manifestOnly := []Footprint{{Resource: "kokoro", Rung: "gpu", PeakBytes: 3 * gib, Source: FootprintSourceManifestDefault}}
	if recs := RecommendWithFootprints([]CapacityClaim{claim}, manifestOnly, DefaultPolicy(), resolve); len(recs) != 0 {
		t.Fatalf("must stay silent without a measured footprint, got %+v", recs)
	}
}
