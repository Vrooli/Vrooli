package capacity

import (
	"testing"
	"time"
)

func qualifiedTestClaim(id, role string, now time.Time) (CapacityClaim, PlacementQualification) {
	idleSince := now.Add(-10 * time.Minute)
	model := ModelIdentity{Name: role + "-model", Digest: "sha256:" + role}
	claim := CapacityClaim{
		ClaimID: id, OwnerKind: OwnerKindResource, OwnerID: role, ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 4 * gib, PreferredBytes: 4 * gib, FloorBytes: 1 * gib,
		Priority: PriorityService, Status: StatusGranted, ActivityState: ActivityIdle,
		Generation: 7, LastActiveAt: &idleSince,
		DegradeProfile: &DegradeProfile{Role: role, Model: model, Steps: []DegradeStep{
			{Label: "preferred", AmountBytes: 4 * gib},
			{Label: "compact", AmountBytes: 2 * gib},
			{Label: "cpu", AmountBytes: 0},
		}},
	}
	proof := PlacementQualification{
		ClaimID: id, ClaimGeneration: 7, Role: role, Model: model, Rung: "compact", Device: ResidencyBackendCUDA,
		Suite: role + "-quality", SuiteDigest: "sha256:suite-" + role, Cases: 24,
		Passed: true, P95LatencyMS: 120, MaxLatencyMS: 200, ObservedVRAM: gib,
		SampledAt: now.Add(-time.Hour),
	}
	return claim, proof
}

func TestPlanQualifiedDownsizeCoordinatesThreeQualifiedRoles(t *testing.T) {
	now := time.Date(2026, 11, 16, 3, 10, 0, 0, time.UTC)
	policy := DefaultPolicy()
	claims := make([]CapacityClaim, 0, 3)
	proofs := make([]PlacementQualification, 0, 3)
	for i, role := range []string{"rerank", "speech", "chat"} {
		claim, proof := qualifiedTestClaim(string(rune('a'+i)), role, now)
		claims = append(claims, claim)
		proofs = append(proofs, proof)
	}
	plan := PlanQualifiedDownsize(QualifiedPlacementRequest{
		RequesterPriority: PriorityInteractive, DeficitBytes: 6 * gib,
		Claims: claims, Evidence: proofs, Policy: policy, Now: now,
	})
	if !plan.Satisfied || plan.FreedBytes != 6*gib || len(plan.Actions) != 3 {
		t.Fatalf("expected three qualified 2 GiB moves to cover 6 GiB, got %+v", plan)
	}
	seen := map[string]bool{}
	for _, action := range plan.Actions {
		if action.ToStep != "compact" || action.Action != ActionRequestDegrade || action.Generation != 7 || action.FreesBytes != 2*gib {
			t.Errorf("unexpected action: %+v", action)
		}
		seen[action.OwnerID] = true
	}
	for _, role := range []string{"rerank", "speech", "chat"} {
		if !seen[role] {
			t.Errorf("role %q missing from coordinated plan", role)
		}
	}
}

func TestPlanQualifiedDownsizeFailsClosedForUnsafeOrUnqualifiedClaims(t *testing.T) {
	now := time.Date(2026, 11, 16, 3, 10, 0, 0, time.UTC)
	policy := DefaultPolicy()
	claim, proof := qualifiedTestClaim("a", "rerank", now)
	tests := []struct {
		name  string
		claim CapacityClaim
		proof PlacementQualification
	}{
		{"active", func() CapacityClaim { c := claim; c.ActivityState = ActivityActive; return c }(), proof},
		{"protected", func() CapacityClaim { c := claim; c.Protected = true; return c }(), proof},
		{"wrong model digest", claim, func() PlacementQualification { p := proof; p.Model.Digest = "other"; return p }()},
		{"stale", claim, func() PlacementQualification { p := proof; p.SampledAt = now.Add(-8 * 24 * time.Hour); return p }()},
		{"failed suite", claim, func() PlacementQualification { p := proof; p.Passed = false; return p }()},
		{"insufficient cases", claim, func() PlacementQualification { p := proof; p.Cases = 19; return p }()},
		{"latency objective missed", claim, func() PlacementQualification { p := proof; p.P95LatencyMS = 201; return p }()},
		{"wrong claim", claim, func() PlacementQualification { p := proof; p.ClaimID = "other"; return p }()},
		{"wrong claim generation", claim, func() PlacementQualification { p := proof; p.ClaimGeneration++; return p }()},
		{"wrong role", claim, func() PlacementQualification { p := proof; p.Role = "different-role"; return p }()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan := PlanQualifiedDownsize(QualifiedPlacementRequest{
				RequesterPriority: PriorityInteractive, DeficitBytes: gib,
				Claims: []CapacityClaim{tc.claim}, Evidence: []PlacementQualification{tc.proof},
				Policy: policy, Now: now,
			})
			if len(plan.Actions) != 0 || plan.Satisfied {
				t.Fatalf("unsafe or weak evidence must not authorize placement change: %+v", plan)
			}
		})
	}
}

func TestQualifiedCPUPlacementRequiresZeroMeasuredVRAM(t *testing.T) {
	now := time.Date(2026, 11, 16, 3, 10, 0, 0, time.UTC)
	claim, proof := qualifiedTestClaim("a", "embedding", now)
	claim.FloorBytes = 0
	claim.DegradeProfile.Steps = append(claim.DegradeProfile.Steps, DegradeStep{Label: "cpu", AmountBytes: 0})
	proof.Rung = "cpu"
	proof.Device = ResidencyBackendCPU
	proof.ObservedVRAM = 1
	if _, ok := qualifiedDegradeAction(claim, gib, []PlacementQualification{proof}, now, 7*24*time.Hour); ok {
		t.Fatal("CPU placement with measured VRAM must be rejected")
	}
	proof.ObservedVRAM = 0
	if _, ok := qualifiedDegradeAction(claim, gib, []PlacementQualification{proof}, now, 7*24*time.Hour); !ok {
		t.Fatal("passing CPU suite with zero VRAM should qualify")
	}
}

func TestQualifiedUpshiftReturnsOnlyRolesWithFreshPassingTargetEvidence(t *testing.T) {
	now := time.Date(2026, 11, 16, 3, 10, 0, 0, time.UTC)
	claim, proof := qualifiedTestClaim("a", "rerank", now)
	claim.AmountBytes = 2 * gib
	claim.Status = StatusDegraded
	claim.DegradeProfile.Upshift = true
	proof.Rung = "preferred"
	proof.ObservedVRAM = 3 * gib
	snapshot := snapshotWithGPU0(16, 2)
	plan := PlanQualifiedUpshiftAll([]CapacityClaim{claim}, snapshot, DefaultPolicy(), []PlacementQualification{proof}, now, 24*time.Hour)
	if len(plan.Actions) != 1 || plan.Actions[0].ToStep != "preferred" {
		t.Fatalf("fresh qualified preferred rung should be restored: %+v", plan)
	}
	proof.SampledAt = now.Add(-25 * time.Hour)
	plan = PlanQualifiedUpshiftAll([]CapacityClaim{claim}, snapshot, DefaultPolicy(), []PlacementQualification{proof}, now, 24*time.Hour)
	if len(plan.Actions) != 0 {
		t.Fatalf("stale role evidence must not authorize a return rung: %+v", plan)
	}
}
