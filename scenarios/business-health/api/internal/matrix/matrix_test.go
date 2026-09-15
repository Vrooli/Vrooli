package matrix

import (
	"testing"
	"time"

	"business-health/internal/evidence"
	"business-health/internal/extraction"

	intent "intent-go"

	"github.com/stretchr/testify/require"
)

func now() time.Time { return time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC) }

func contractFixture() extraction.Contract {
	return extraction.Contract{
		Scenario:        "fixture",
		PRDPresent:      true,
		RegistryPresent: true,
		PRDDoc: intent.PRDDocument{
			Path:    "PRD.md",
			Present: true,
			Targets: []intent.OperationalTarget{
				{RawID: "OT-P0-001", ID: "OT-P0-001", Tier: "P0", Checked: true, Title: "Done thing", Line: 10},
				{RawID: "OT-P1-001", ID: "OT-P1-001", Tier: "P1", Checked: false, Title: "Open thing", Line: 14},
				{RawID: "OT-P2-001", ID: "OT-P2-001", Tier: "P2", Checked: true, Title: "Orphan checked", Line: 18},
			},
		},
		Registry: intent.Registry{
			Present: true,
			Modules: []intent.RegistryModule{{
				Path: "requirements/01-core/module.json",
				Requirements: []intent.RegistryRequirement{
					{
						ID: "R-001", Title: "Done requirement", Status: "complete", Criticality: "P0", PRDRef: "OT-P0-001", Module: "requirements/01-core/module.json",
						Validations: []intent.RegistryValidation{{Type: "manual", Status: "implemented"}},
					},
					{
						ID: "R-002", Title: "Open requirement", Status: "planned", Criticality: "P1", PRDRef: "OT-P1-001", Module: "requirements/01-core/module.json",
						Validations: []intent.RegistryValidation{{Type: "test", Ref: "api/x_test.go", Status: "planned", Phase: "unit"}},
					},
				},
			}},
		},
	}
}

func snapshotFixture() evidence.SyncSnapshot {
	return evidence.SyncSnapshot{
		Version:     "1.0.0",
		GeneratedAt: now().Add(-time.Hour),
		OperationalTargets: []evidence.SyncTarget{
			{ID: "OT-P0-001", Status: "complete", RequirementIDs: []string{"R-001"}, CompletionRate: 100},
		},
	}
}

// [REQ:BH-EVD-002] The join answers rows for requirements, joins the OT
// side via prd_ref, and appends orphaned OTs.
func TestJoinShape(t *testing.T) {
	attested := evidence.Attestation{RequirementID: "R-001", AttestedBy: "op", AttestedAt: now().Add(-24 * time.Hour), ExpiresAt: now().Add(24 * time.Hour)}
	res := Join(Inputs{
		Contract:     contractFixture(),
		Snapshot:     snapshotFixture(),
		HasSnapshot:  true,
		Attestations: map[string]evidence.Attestation{"R-001": attested},
		Now:          now(),
	})
	require.Len(t, res.Rows, 3, "2 requirement rows + 1 orphan OT row (OT-P2-001; OT-P1-001 is covered by R-002)")

	byReq := map[string]Row{}
	var orphans []Row
	for _, r := range res.Rows {
		if r.RequirementID != "" {
			byReq[r.RequirementID] = r
		} else {
			orphans = append(orphans, r)
		}
	}
	r1 := byReq["R-001"]
	require.Equal(t, "OT-P0-001", r1.OTID)
	require.True(t, r1.OTChecked)
	require.Equal(t, "complete", r1.Evidence.OTStatus)
	require.NotNil(t, r1.Evidence.Manual)
	require.False(t, r1.Unproven, "complete with unexpired attestation is proven")

	require.Len(t, orphans, 1)
	require.Equal(t, "OT-P2-001", orphans[0].OTID)
	require.True(t, orphans[0].Unproven, "checked orphan OT is an unproven claim")

	require.Equal(t, 2, res.Registry.RequirementCount)
	require.Equal(t, 3, res.Registry.OperationalTargetCount)
}

// [REQ:BH-EVD-004] Unproven derivations: complete-without-snapshot,
// complete-with-expired-manual, checked-OT-with-incomplete-requirement.
func TestUnprovenClaims(t *testing.T) {
	t.Run("complete status with no snapshot is unearned", func(t *testing.T) {
		res := Join(Inputs{
			Contract:  contractFixture(),
			Staleness: evidence.Staleness{Stale: true, Detail: "suite runs exist but no snapshot"},
			Now:       now(),
		})
		kinds := driftKinds(res.Drift)
		require.Contains(t, kinds, "status_unearned")
		require.Contains(t, kinds, "stale_snapshot")
	})
	t.Run("valid manual evidence earns a manual-only claim without snapshot", func(t *testing.T) {
		attested := evidence.Attestation{RequirementID: "R-001", AttestedBy: "op", AttestedAt: now().Add(-time.Hour), ExpiresAt: now().Add(time.Hour)}
		res := Join(Inputs{
			Contract:     contractFixture(),
			Attestations: map[string]evidence.Attestation{"R-001": attested},
			Now:          now(),
		})
		for _, row := range res.Rows {
			if row.RequirementID == "R-001" {
				require.False(t, row.Unproven, "valid manual evidence should earn a manual-only claim")
				return
			}
		}
		t.Fatal("R-001 row not found")
	})
	t.Run("complete with only expired manual evidence", func(t *testing.T) {
		expired := evidence.Attestation{RequirementID: "R-001", AttestedBy: "op", AttestedAt: now().Add(-100 * 24 * time.Hour), ExpiresAt: now().Add(-10 * 24 * time.Hour)}
		res := Join(Inputs{
			Contract:     contractFixture(),
			Snapshot:     snapshotFixture(),
			HasSnapshot:  true,
			Attestations: map[string]evidence.Attestation{"R-001": expired},
			Now:          now(),
		})
		kinds := driftKinds(res.Drift)
		require.Contains(t, kinds, "expired_manual")
		require.Contains(t, kinds, "unproven_claim")
	})
	t.Run("checked OT with incomplete linked requirement", func(t *testing.T) {
		c := contractFixture()
		c.PRDDoc.Targets[1].Checked = true // OT-P1-001 checked, R-002 planned
		res := Join(Inputs{Contract: c, Snapshot: snapshotFixture(), HasSnapshot: true, Now: now()})
		var found bool
		for _, d := range res.Drift {
			if d.Kind == "unproven_claim" && d.SubjectID == "OT-P1-001" {
				found = true
			}
		}
		require.True(t, found, "expected unproven claim for the prematurely checked OT, got %+v", res.Drift)
	})
}

// [REQ:BH-EVD-002] Zero-requirement scenario still renders (orphan OTs
// only) and a missing snapshot degrades, never errors.
func TestJoinDegradedAndEmpty(t *testing.T) {
	c := contractFixture()
	c.Registry = intent.Registry{Present: true}
	res := Join(Inputs{
		Contract:  c,
		Staleness: evidence.Staleness{Detail: "no evidence artifacts yet (no suite runs, no snapshot)"},
		Now:       now(),
	})
	require.Len(t, res.Rows, 3, "every OT becomes an orphan row")
	require.NotEmpty(t, res.DegradedReason)
	require.Equal(t, 0, res.Registry.RequirementCount)
}

func driftKinds(entries []DriftEntry) []string {
	out := make([]string, 0, len(entries))
	for _, d := range entries {
		out = append(out, d.Kind)
	}
	return out
}
