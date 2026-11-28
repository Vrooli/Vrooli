package capacity

import (
	"testing"
	"time"
)

func TestBuildPlacementTableSeparatesSuiteMeasurementFromCurrentClaim(t *testing.T) {
	now := time.Date(2026, 11, 16, 3, 14, 0, 0, time.UTC)
	claim, proof := qualifiedTestClaim("a", "rerank", now)
	claim.AmountBytes = 2 * gib
	claim.Status = StatusDegraded
	observedAt := now.Add(-time.Second)
	claim.ObservedAt = &observedAt
	claim.ObservedBytes = gib
	proof.Rung = "compact"
	rows := BuildPlacementTable([]CapacityClaim{claim}, []PlacementQualification{proof}, now, 24*time.Hour)
	if len(rows) != 1 {
		t.Fatalf("expected one measured role/rung row, got %+v", rows)
	}
	row := rows[0]
	if row.EvidenceStatus != PlacementEvidenceQualified || !row.CurrentClaimRung || row.Device != ResidencyBackendCUDA || row.P95LatencyMS != 120 || row.MeasuredVRAMBytes != gib || row.CurrentClaimObservedVRAM != gib {
		t.Fatalf("table row lost measured or current claim facts: %+v", row)
	}
}

func TestBuildPlacementTableDoesNotGuessCPUVersusUnloadedOrTrustOldIdentity(t *testing.T) {
	now := time.Date(2026, 11, 16, 3, 14, 0, 0, time.UTC)
	claim, proof := qualifiedTestClaim("a", "embedding", now)
	claim.AmountBytes = 0
	claim.Status = StatusDegraded
	claim.DegradeProfile.Steps = append(claim.DegradeProfile.Steps, DegradeStep{Label: "unloaded", AmountBytes: 0})
	proof.Rung = "cpu"
	proof.Device = ResidencyBackendCPU
	proof.ObservedVRAM = 0
	rows := BuildPlacementTable([]CapacityClaim{claim}, []PlacementQualification{proof}, now, 24*time.Hour)
	if len(rows) != 2 || rows[0].EvidenceStatus != PlacementEvidenceQualified || rows[0].CurrentClaimRung || rows[1].EvidenceStatus != PlacementEvidenceMissing {
		t.Fatalf("same zero-byte CPU/unloaded rungs must not be collapsed into current state: %+v", rows)
	}
	proof.ClaimGeneration++
	rows = BuildPlacementTable([]CapacityClaim{claim}, []PlacementQualification{proof}, now, 24*time.Hour)
	if len(rows) != 2 || rows[0].EvidenceStatus != PlacementEvidenceMismatched || rows[1].EvidenceStatus != PlacementEvidenceMissing {
		t.Fatalf("stale claim incarnation should be reported and current role remain missing: %+v", rows)
	}
}
