package execution

import (
	"math"

	internalexecution "plan-manager/internal/execution"
	internalplans "plan-manager/internal/plans"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
)

// orderToInt32 is a bounds-safe int→int32 conversion for a phase order (always
// small and non-negative in practice). The explicit clamp satisfies gosec G115
// without a //nosec.
func orderToInt32(n int) int32 {
	switch {
	case n < 0:
		return 0
	case n > math.MaxInt32:
		return math.MaxInt32
	default:
		return int32(n)
	}
}

// This file is the only translation point between the proto wire types
// (vrooli.plan_manager.v1.execution + .shared) and the execution/plans domain
// vocabulary. The domain layer never imports proto (api-steer §7).

func executionToProto(e internalexecution.Execution) *executionv1.Execution {
	return &executionv1.Execution{
		Id:             e.ID,
		PlanId:         e.PlanID,
		RunId:          e.RunID,
		CurrentPhaseId: e.CurrentPhaseID,
		Complete:       e.Complete,
		StartedAt:      e.StartedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func phaseContextToProto(c internalexecution.PhaseContext) *executionv1.PhaseContext {
	out := &executionv1.PhaseContext{
		RequiredReading: c.RequiredReading,
		Reminders:       c.Reminders,
		Staleness:       stalenessToProto(c.Staleness),
		ResumePhaseId:   c.ResumePhaseID,
		Completeness:    completenessToProto(c.Completeness),
	}
	if c.HasCurrent {
		out.CurrentPhase = phaseToProto(c.CurrentPhase)
	}
	if c.HasNext {
		out.NextPhase = phaseToProto(c.NextPhase)
	}
	if c.HasValidation {
		out.LastValidation = validationResultToProto(c.LastValidation)
	}
	return out
}

func nudgesToProto(nudges []internalexecution.CompletionNudge) []*executionv1.CompletionNudge {
	out := make([]*executionv1.CompletionNudge, 0, len(nudges))
	for _, n := range nudges {
		out = append(out, &executionv1.CompletionNudge{
			Kind:      n.Kind,
			Message:   n.Message,
			Satisfied: n.Satisfied,
		})
	}
	return out
}

func decisionToProto(d internalexecution.Decision) *sharedv1.Decision {
	return &sharedv1.Decision{
		Id:         d.ID,
		Summary:    d.Summary,
		Detail:     d.Detail,
		PhaseId:    d.PhaseID,
		RecordedAt: d.RecordedAt,
	}
}

func decisionsToProto(ds []internalexecution.Decision) []*sharedv1.Decision {
	out := make([]*sharedv1.Decision, 0, len(ds))
	for _, d := range ds {
		out = append(out, decisionToProto(d))
	}
	return out
}

func findingToProto(f internalexecution.Finding) *sharedv1.Finding {
	return &sharedv1.Finding{
		Id:               f.ID,
		Title:            f.Title,
		Detail:           f.Detail,
		Triage:           triageToProto(f.Triage),
		PhaseId:          f.PhaseID,
		RecordedAt:       f.RecordedAt,
		AttributionRunId: f.AttributionRunID,
	}
}

func findingsToProto(fs []internalexecution.Finding) []*sharedv1.Finding {
	out := make([]*sharedv1.Finding, 0, len(fs))
	for _, f := range fs {
		out = append(out, findingToProto(f))
	}
	return out
}

func handoffToProto(h internalexecution.Handoff) *sharedv1.Handoff {
	out := &sharedv1.Handoff{
		Id:                h.ID,
		ExecutionId:       h.ExecutionID,
		PlanId:            h.PlanID,
		Completeness:      completenessToProto(h.Completeness),
		ResumePhaseId:     h.ResumePhaseID,
		Decisions:         decisionsToProto(h.Decisions),
		CandidateFindings: findingsToProto(h.CandidateFindings),
		Staleness:         stalenessToProto(h.Staleness),
		ProseHandoffRef:   h.ProseHandoffRef,
		AssembledAt:       h.AssembledAt,
	}
	if h.HasValidation {
		out.LastValidation = validationResultToProto(h.LastValidation)
	}
	return out
}

func velocityToProto(v internalexecution.VelocityPoint) *sharedv1.VelocityPoint {
	return &sharedv1.VelocityPoint{
		Id:              v.ID,
		PlanId:          v.PlanID,
		RunId:           v.RunID,
		WallTimeSeconds: v.WallTimeSeconds,
		Tokens:          v.Tokens,
		Iterations:      v.Iterations,
		Completeness:    completenessToProto(v.Completeness),
		RecordedAt:      v.RecordedAt,
	}
}

func velocitiesToProto(vs []internalexecution.VelocityPoint) []*sharedv1.VelocityPoint {
	out := make([]*sharedv1.VelocityPoint, 0, len(vs))
	for _, v := range vs {
		out = append(out, velocityToProto(v))
	}
	return out
}

func validationResultToProto(r internalexecution.ValidationResult) *sharedv1.ValidationResult {
	return &sharedv1.ValidationResult{
		Id:          r.ID,
		PlanId:      r.PlanID,
		PhaseId:     r.PhaseID,
		Verdict:     verdictToProto(r.Verdict),
		Staleness:   stalenessToProto(r.Staleness),
		CommandsRun: r.CommandsRun,
		Detail:      r.Detail,
		RanAt:       r.RanAt,
	}
}

func phaseToProto(ph internalplans.Phase) *sharedv1.Phase {
	return &sharedv1.Phase{
		Id:              ph.ID,
		Order:           orderToInt32(ph.Order),
		Title:           ph.Title,
		Intent:          ph.Intent,
		RequiredReading: ph.RequiredReading,
		Reminders:       ph.Reminders,
		BaselineScope:   ph.BaselineScope,
		Acceptance:      ph.Acceptance,
		Status:          phaseStatusToProto(ph.Status),
		References:      referencesToProto(ph.References),
	}
}

func referencesToProto(refs []internalplans.Reference) []*sharedv1.Reference {
	out := make([]*sharedv1.Reference, 0, len(refs))
	for _, r := range refs {
		out = append(out, &sharedv1.Reference{
			Id:           r.ID,
			Kind:         refKindToProto(r.Kind),
			Target:       r.Target,
			Future:       r.Future,
			Resolution:   refResolutionToProto(r.Resolution),
			Staleness:    stalenessToProto(r.Staleness),
			ChangeFactor: r.ChangeFactor,
			Note:         r.Note,
		})
	}
	return out
}

func planToProto(p internalplans.Plan) *sharedv1.Plan {
	return &sharedv1.Plan{
		Id:               p.ID,
		Slug:             p.Slug,
		Title:            p.Title,
		Status:           planStatusToProto(p.Status),
		ContentHash:      p.ContentHash,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
		Purpose:          p.Purpose,
		Scope:            p.Scope,
		Constraints:      p.Constraints,
		NonGoals:         p.NonGoals,
		References:       referencesToProto(p.References),
		RegressionAnchor: anchorToProto(p.RegressionAnchor),
		DefinitionOfDone: p.DefinitionOfDone,
		Phases:           phasesToProto(p.Phases),
		Supersedes:       p.Supersedes,
		SupersededBy:     p.SupersededBy,
	}
}

func phasesToProto(phases []internalplans.Phase) []*sharedv1.Phase {
	out := make([]*sharedv1.Phase, 0, len(phases))
	for _, ph := range phases {
		out = append(out, phaseToProto(ph))
	}
	return out
}

func anchorToProto(a internalplans.RegressionAnchor) *sharedv1.RegressionAnchor {
	return &sharedv1.RegressionAnchor{
		Strategy:       a.Strategy,
		Scenario:       a.Scenario,
		BaselineName:   a.BaselineName,
		HeadSha:        a.HeadSha,
		AllowlistPaths: a.AllowlistPaths,
		Commands:       a.Commands,
		CapturedAt:     a.CapturedAt,
		Unavailable:    a.Unavailable,
	}
}

// --- enum converters ---

func phaseStatusToProto(s internalplans.PhaseStatus) sharedv1.PhaseStatus {
	switch s {
	case internalplans.PhaseStatusTodo:
		return sharedv1.PhaseStatus_PHASE_STATUS_TODO
	case internalplans.PhaseStatusActive:
		return sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE
	case internalplans.PhaseStatusDone:
		return sharedv1.PhaseStatus_PHASE_STATUS_DONE
	case internalplans.PhaseStatusBlocked:
		return sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED
	default:
		return sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED
	}
}

func phaseStatusFromProto(s sharedv1.PhaseStatus) internalplans.PhaseStatus {
	switch s {
	case sharedv1.PhaseStatus_PHASE_STATUS_TODO:
		return internalplans.PhaseStatusTodo
	case sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE:
		return internalplans.PhaseStatusActive
	case sharedv1.PhaseStatus_PHASE_STATUS_DONE:
		return internalplans.PhaseStatusDone
	case sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED:
		return internalplans.PhaseStatusBlocked
	default:
		return ""
	}
}

func planStatusToProto(s internalplans.PlanStatus) sharedv1.PlanStatus {
	switch s {
	case internalplans.PlanStatusDraft:
		return sharedv1.PlanStatus_PLAN_STATUS_DRAFT
	case internalplans.PlanStatusActive:
		return sharedv1.PlanStatus_PLAN_STATUS_ACTIVE
	case internalplans.PlanStatusComplete:
		return sharedv1.PlanStatus_PLAN_STATUS_COMPLETE
	case internalplans.PlanStatusArchived:
		return sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED
	default:
		return sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED
	}
}

func refKindToProto(k internalplans.ReferenceKind) sharedv1.ReferenceKind {
	switch k {
	case internalplans.ReferenceCode:
		return sharedv1.ReferenceKind_REFERENCE_KIND_CODE
	case internalplans.ReferenceReq:
		return sharedv1.ReferenceKind_REFERENCE_KIND_REQ
	case internalplans.ReferenceDoc:
		return sharedv1.ReferenceKind_REFERENCE_KIND_DOC
	default:
		return sharedv1.ReferenceKind_REFERENCE_KIND_UNSPECIFIED
	}
}

func refResolutionToProto(r internalplans.ReferenceResolution) sharedv1.ReferenceResolution {
	switch r {
	case internalplans.ResolutionResolved:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED
	case internalplans.ResolutionUnresolved:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED
	case internalplans.ResolutionFuture:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE
	case internalplans.ResolutionMissing:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING
	default:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNSPECIFIED
	}
}

func stalenessToProto(s internalplans.StalenessTier) sharedv1.StalenessTier {
	switch s {
	case internalplans.StalenessFresh:
		return sharedv1.StalenessTier_STALENESS_TIER_FRESH
	case internalplans.StalenessLightlyStale:
		return sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE
	case internalplans.StalenessDefinitelyStale:
		return sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE
	default:
		return sharedv1.StalenessTier_STALENESS_TIER_UNSPECIFIED
	}
}

func completenessToProto(c internalexecution.Completeness) sharedv1.Completeness {
	switch c {
	case internalexecution.CompletenessFull:
		return sharedv1.Completeness_COMPLETENESS_FULL
	case internalexecution.CompletenessPartial:
		return sharedv1.Completeness_COMPLETENESS_PARTIAL
	default:
		return sharedv1.Completeness_COMPLETENESS_UNSPECIFIED
	}
}

func triageToProto(t internalexecution.FindingTriage) sharedv1.FindingTriage {
	switch t {
	case internalexecution.TriageCandidate:
		return sharedv1.FindingTriage_FINDING_TRIAGE_CANDIDATE
	case internalexecution.TriagePromoted:
		return sharedv1.FindingTriage_FINDING_TRIAGE_PROMOTED
	case internalexecution.TriageDismissed:
		return sharedv1.FindingTriage_FINDING_TRIAGE_DISMISSED
	default:
		return sharedv1.FindingTriage_FINDING_TRIAGE_UNSPECIFIED
	}
}

func triageFromProto(t sharedv1.FindingTriage) internalexecution.FindingTriage {
	switch t {
	case sharedv1.FindingTriage_FINDING_TRIAGE_CANDIDATE:
		return internalexecution.TriageCandidate
	case sharedv1.FindingTriage_FINDING_TRIAGE_PROMOTED:
		return internalexecution.TriagePromoted
	case sharedv1.FindingTriage_FINDING_TRIAGE_DISMISSED:
		return internalexecution.TriageDismissed
	default:
		return internalexecution.TriageUnspecified
	}
}

// verdictToProto maps the validation verdict string carried on a
// ValidationResult into its proto enum. The strings mirror
// internal/validation.Verdict.
func verdictToProto(v string) sharedv1.ValidationVerdict {
	switch v {
	case "pass":
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS
	case "fail":
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_FAIL
	case "unknown":
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNKNOWN
	default:
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNSPECIFIED
	}
}
