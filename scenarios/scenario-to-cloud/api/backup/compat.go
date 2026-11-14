package backup

import (
	"fmt"
	"sort"
	"strings"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// Schema strategies a scenario declares in deployment.recovery.schema_strategy.
const (
	SchemaStrategyExpandContract  = "expand_contract"
	SchemaStrategyExplicitRestore = "explicit_restore"
	SchemaStrategyNone            = "none"
)

// Rollback reason codes.
const (
	ReasonSchemaAhead        = "predecessor_cannot_read_schema"
	ReasonNoRollback         = "code_rollback_not_declared"
	ReasonRestoreRequired    = "explicit_restore_required"
	ReasonSchemaUnobserved   = "schema_version_unobserved"
	ReasonCompatiblePeriod   = "expand_phase_still_readable"
	ReasonSameSchema         = "same_schema"
	ReasonNoSchema           = "no_schema"
	ReasonPredecessorAllowed = "any_predecessor_declared"
)

// RollbackFacts are the observed facts the admission decision binds to.
type RollbackFacts struct {
	// CurrentSchema is the schema version the live data is at.
	CurrentSchema string
	// TargetSchema is the schema version the rollback candidate (the
	// predecessor code) was built against.
	TargetSchema string
	// SchemaStrategy and CodeRollback are the scenario's declared recovery
	// contract (closure component recovery).
	SchemaStrategy string
	CodeRollback   string
	// ReadableBy lists the schema versions whose code can still read
	// CurrentSchema (the expand phase keeps the predecessor readable until
	// the contract phase runs). Empty means only CurrentSchema itself.
	ReadableBy []string
	// RecoveryPoints are the deployment's stored points; the newest point at
	// TargetSchema anchors a restore plan.
	RecoveryPoints []domain.RecoveryPoint
}

// EvaluateRollback decides whether the predecessor code can run on the
// current data. A compatible verdict keeps the data in place. An
// incompatible verdict never leaves the operator with a bare refusal: it
// carries a typed forward-repair plan (fix forward on the current schema) or
// a restore plan anchored on a recovery point at the target schema, with the
// preconditions each requires.
func EvaluateRollback(f RollbackFacts) domain.RollbackVerdict {
	verdict := domain.RollbackVerdict{SchemaStrategy: f.SchemaStrategy, CurrentSchema: f.CurrentSchema, TargetSchema: f.TargetSchema}
	if f.CodeRollback == "none" {
		return refuseRollback(verdict, ReasonNoRollback, "the scenario declares code_rollback none", forwardRepair(f))
	}
	if f.CurrentSchema == "" && f.TargetSchema == "" {
		verdict.Compatible, verdict.ReasonCode = true, ReasonNoSchema
		return verdict
	}
	if f.CurrentSchema == "" || f.TargetSchema == "" {
		return refuseRollback(verdict, ReasonSchemaUnobserved, "the current or target schema version is unobserved; admission fails closed", forwardRepair(f))
	}
	if f.CurrentSchema == f.TargetSchema {
		verdict.Compatible, verdict.ReasonCode = true, ReasonSameSchema
		return verdict
	}
	switch f.SchemaStrategy {
	case SchemaStrategyNone:
		verdict.Compatible, verdict.ReasonCode = true, ReasonNoSchema
		return verdict
	case SchemaStrategyExpandContract:
		for _, readable := range f.ReadableBy {
			if readable == f.TargetSchema {
				verdict.Compatible, verdict.ReasonCode = true, ReasonCompatiblePeriod
				return verdict
			}
		}
		if f.CodeRollback == "any_predecessor" {
			verdict.Compatible, verdict.ReasonCode = true, ReasonPredecessorAllowed
			return verdict
		}
		return refuseRollback(verdict, ReasonSchemaAhead, fmt.Sprintf("schema %s has contracted past what code at %s can read", f.CurrentSchema, f.TargetSchema), restoreOrRepair(f))
	case SchemaStrategyExplicitRestore:
		return refuseRollback(verdict, ReasonRestoreRequired, "the scenario declares explicit_restore: a code rollback across schema versions requires restoring a recovery point", restoreOrRepair(f))
	default:
		return refuseRollback(verdict, ReasonSchemaUnobserved, fmt.Sprintf("schema strategy %q is not declared", f.SchemaStrategy), forwardRepair(f))
	}
}

func refuseRollback(v domain.RollbackVerdict, code, reason string, plan *domain.RecoveryPlan) domain.RollbackVerdict {
	v.Compatible, v.ReasonCode, v.Reason, v.Plan = false, code, reason, plan
	return v
}

func forwardRepair(f RollbackFacts) *domain.RecoveryPlan {
	return &domain.RecoveryPlan{
		Kind:          domain.RecoveryPlanForwardRepair,
		Preconditions: []string{"recovery_point_required:" + f.CurrentSchema, "candidate_release_reads_schema:" + f.CurrentSchema},
		Steps: []string{
			"cloud-target:data.backup (capture a recovery point at schema " + f.CurrentSchema + " before any change)",
			"release.verify + release.stage (a candidate built to read schema " + f.CurrentSchema + ")",
			"release.activate (the candidate replaces the failed revision; data stays in place)",
			"verify.readiness + application invariants",
		},
	}
}

func restoreOrRepair(f RollbackFacts) *domain.RecoveryPlan {
	anchor := NewestAtSchema(f.RecoveryPoints, f.TargetSchema)
	if anchor == nil {
		plan := forwardRepair(f)
		plan.Preconditions = append(plan.Preconditions, "no_recovery_point_at_schema:"+f.TargetSchema)
		return plan
	}
	return &domain.RecoveryPlan{
		Kind:            domain.RecoveryPlanRestore,
		RecoveryPointID: anchor.ID,
		Preconditions: []string{
			"recovery_point_required:" + f.CurrentSchema,
			"restore_target_clean",
			"recovery_key_resolvable:" + anchor.RecoveryKeyRef,
			"acknowledged_writes_since:" + anchor.CapturedAt.UTC().Format("2006-01-02T15:04:05Z07:00") + " are lost unless replayed",
		},
		Steps: []string{
			"cloud-target:data.backup (capture the current data at schema " + f.CurrentSchema + " first)",
			"workload.stop (maintenance: no writes during the switch)",
			"cloud-target:data.restore --recovery-point " + anchor.ID + " into clean bindings",
			"cloud-target:release.rollback --to <predecessor release>",
			"cloud-target:data.verify + application invariants",
		},
	}
}

// NewestAtSchema returns the newest recovery point captured at schema, or
// nil when none exists.
func NewestAtSchema(points []domain.RecoveryPoint, schema string) *domain.RecoveryPoint {
	var newest *domain.RecoveryPoint
	for i := range points {
		p := &points[i]
		if p.SchemaVersion != schema {
			continue
		}
		if newest == nil || p.CapturedAt.After(newest.CapturedAt) {
			newest = p
		}
	}
	return newest
}

// RollbackError renders a refused verdict as the typed rollback_incompatible
// error carrying the plan.
func RollbackError(v domain.RollbackVerdict) *apierrors.Error {
	if v.Compatible {
		return nil
	}
	err := apierrors.New(apierrors.CodeRollbackIncompatible, "code rollback refused: "+v.Reason).
		WithDetail("reason_code", v.ReasonCode).WithDetail("current_schema", v.CurrentSchema).WithDetail("target_schema", v.TargetSchema).WithDetail("schema_strategy", v.SchemaStrategy)
	if v.Plan != nil {
		err = err.WithDetail("plan", v.Plan).WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: v.Plan.Kind, Reference: v.Plan.RecoveryPointID, Label: strings.Join(v.Plan.Steps, "; ")})
	}
	return err
}

// PreconditionRecoveryPointRequired is the executable-plan precondition kind
// a destructive schema change binds to; its value is the recovery point id
// that must exist before the change runs.
const PreconditionRecoveryPointRequired = "recovery_point_required"

// RequireRecoveryPoint is the admission check before a destructive schema
// change: a recovery point captured for this release at the current schema
// must exist and verify. It returns the satisfying point, or the typed
// recovery_point_required error naming the owner verb that produces one.
func RequireRecoveryPoint(points []domain.RecoveryPoint, releaseDigest, currentSchema string) (*domain.RecoveryPoint, *apierrors.Error) {
	candidates := make([]domain.RecoveryPoint, 0, len(points))
	for _, p := range points {
		if p.SchemaVersion == currentSchema && (releaseDigest == "" || p.ReleaseDigest == releaseDigest) && p.Encrypted && p.RecoveryKeyRef != "" {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return nil, apierrors.New(apierrors.CodeRecoveryPointRequired, "a recovery point at the current schema is required before a destructive schema change").
			WithDetail("release_digest", releaseDigest).WithDetail("schema_version", currentSchema).
			WithNextAction(apierrors.NextAction{Owner: "cloud-target", Kind: "data.backup", Reference: "vrooli cloud-target data backup", Label: "Capture a recovery point bound to this release and schema, then retry."})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].CapturedAt.After(candidates[j].CapturedAt) })
	return &candidates[0], nil
}
