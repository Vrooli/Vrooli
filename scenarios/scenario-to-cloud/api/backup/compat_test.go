package backup

import (
	"testing"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

func point(id, schema, release string, at time.Time) domain.RecoveryPoint {
	return domain.RecoveryPoint{ID: id, DeploymentID: "dep", SchemaVersion: schema, ReleaseDigest: release, CapturedAt: at, Encrypted: true, RecoveryKeyRef: "fixture/recovery:key"}
}

// TestEvaluateRollbackIsSchemaStrategyAware [REQ:STC-P0-031] proves
// P12-A02: the predecessor is admitted only when it can read the current
// schema, and every refusal carries a typed forward-repair or restore plan.
func TestEvaluateRollbackIsSchemaStrategyAware(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	points := []domain.RecoveryPoint{point("rp-old", "v1", "r1", now.Add(-2*time.Hour)), point("rp-new", "v1", "r1", now.Add(-time.Hour)), point("rp-v2", "v2", "r2", now)}
	cases := []struct {
		name       string
		facts      RollbackFacts
		compatible bool
		reason     string
		planKind   string
		anchor     string
	}{
		{"same schema", RollbackFacts{CurrentSchema: "v2", TargetSchema: "v2", SchemaStrategy: SchemaStrategyExpandContract, CodeRollback: "compatible_predecessor_only"}, true, ReasonSameSchema, "", ""},
		{"no schema at all", RollbackFacts{SchemaStrategy: SchemaStrategyNone, CodeRollback: "any_predecessor"}, true, ReasonNoSchema, "", ""},
		{"expand phase still readable", RollbackFacts{CurrentSchema: "v2", TargetSchema: "v1", SchemaStrategy: SchemaStrategyExpandContract, CodeRollback: "compatible_predecessor_only", ReadableBy: []string{"v1"}}, true, ReasonCompatiblePeriod, "", ""},
		{"contracted past predecessor", RollbackFacts{CurrentSchema: "v2", TargetSchema: "v1", SchemaStrategy: SchemaStrategyExpandContract, CodeRollback: "compatible_predecessor_only", RecoveryPoints: points}, false, ReasonSchemaAhead, domain.RecoveryPlanRestore, "rp-new"},
		{"contracted without a point", RollbackFacts{CurrentSchema: "v2", TargetSchema: "v0", SchemaStrategy: SchemaStrategyExpandContract, CodeRollback: "compatible_predecessor_only", RecoveryPoints: points}, false, ReasonSchemaAhead, domain.RecoveryPlanForwardRepair, ""},
		{"explicit restore", RollbackFacts{CurrentSchema: "v2", TargetSchema: "v1", SchemaStrategy: SchemaStrategyExplicitRestore, CodeRollback: "any_predecessor", RecoveryPoints: points}, false, ReasonRestoreRequired, domain.RecoveryPlanRestore, "rp-new"},
		{"rollback not declared", RollbackFacts{CurrentSchema: "v2", TargetSchema: "v1", SchemaStrategy: SchemaStrategyExpandContract, CodeRollback: "none"}, false, ReasonNoRollback, domain.RecoveryPlanForwardRepair, ""},
		{"unobserved schema fails closed", RollbackFacts{CurrentSchema: "", TargetSchema: "v1", SchemaStrategy: SchemaStrategyExpandContract, CodeRollback: "any_predecessor"}, false, ReasonSchemaUnobserved, domain.RecoveryPlanForwardRepair, ""},
		{"undeclared strategy fails closed", RollbackFacts{CurrentSchema: "v2", TargetSchema: "v1", SchemaStrategy: "", CodeRollback: "any_predecessor"}, false, ReasonSchemaUnobserved, domain.RecoveryPlanForwardRepair, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := EvaluateRollback(tc.facts)
			if v.Compatible != tc.compatible || v.ReasonCode != tc.reason {
				t.Fatalf("verdict = %+v", v)
			}
			if tc.compatible {
				if v.Plan != nil || RollbackError(v) != nil {
					t.Fatalf("compatible verdict must carry no plan: %+v", v)
				}
				return
			}
			if v.Plan == nil || v.Plan.Kind != tc.planKind || v.Plan.RecoveryPointID != tc.anchor || len(v.Plan.Steps) == 0 || len(v.Plan.Preconditions) == 0 {
				t.Fatalf("plan = %+v", v.Plan)
			}
			err := RollbackError(v)
			if err.Code != apierrors.CodeRollbackIncompatible || err.Status() != 409 || apierrors.ExitCodeFor(err.Code) != apierrors.ExitRefused || err.NextAction == nil || err.NextAction.Kind != tc.planKind {
				t.Fatalf("error = %+v", err)
			}
		})
	}
}

func TestRequireRecoveryPointPrefersNewestForRelease(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	points := []domain.RecoveryPoint{point("rp-old", "v1", "r1", now.Add(-2*time.Hour)), point("rp-new", "v1", "r1", now.Add(-time.Hour)), point("rp-other", "v1", "r9", now)}
	got, err := RequireRecoveryPoint(points, "r1", "v1")
	if err != nil || got.ID != "rp-new" {
		t.Fatalf("got %+v err %+v", got, err)
	}
	unencrypted := point("rp-plain", "v3", "r1", now)
	unencrypted.Encrypted = false
	if _, err := RequireRecoveryPoint(append(points, unencrypted), "r1", "v3"); err == nil || err.Code != apierrors.CodeRecoveryPointRequired || err.NextAction == nil || err.NextAction.Kind != "data.backup" {
		t.Fatalf("an unencrypted point must not satisfy the precondition: %+v", err)
	}
}

func TestSelectPostureFromPredecessorState(t *testing.T) {
	cases := map[string]struct {
		state PredecessorState
		want  string
	}{
		"first deployment":     {PredecessorState{}, domain.MigrationPostureGreenfield},
		"predecessor, no data": {PredecessorState{Exists: true}, domain.MigrationPostureGreenfield},
		"unversioned data":     {PredecessorState{Exists: true, HasData: true}, domain.MigrationPostureGreenfieldWithData},
		"versioned data":       {PredecessorState{Exists: true, HasData: true, SchemaVersion: "v4"}, domain.MigrationPostureProductionEvolution},
		"declared versioned":   {PredecessorState{Exists: true, HasData: true, Versioned: true}, domain.MigrationPostureProductionEvolution},
	}
	for name, tc := range cases {
		if got := SelectPosture(tc.state); got != tc.want {
			t.Errorf("%s: posture %q, want %q", name, got, tc.want)
		}
	}
}
