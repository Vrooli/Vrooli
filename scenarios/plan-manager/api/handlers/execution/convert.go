package execution

import (
	internalexecution "plan-manager/internal/execution"
	planmodel "plan-manager/internal/planmodel"
	"plan-manager/internal/planproto"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
)

// orderToInt32 is a bounds-safe int to int32 conversion for a phase order.
func orderToInt32(n int) int32 {
	return planproto.OrderToInt32(n)
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

func phaseToProto(ph planmodel.Phase) *sharedv1.Phase {
	return planproto.PhaseToProto(ph)
}

func referencesToProto(refs []planmodel.Reference) []*sharedv1.Reference {
	return planproto.ReferencesToProto(refs)
}

func planToProto(p planmodel.Plan) *sharedv1.Plan {
	return planproto.PlanToProto(p)
}

func phasesToProto(phases []planmodel.Phase) []*sharedv1.Phase {
	return planproto.PhasesToProto(phases)
}

func anchorToProto(a planmodel.RegressionAnchor) *sharedv1.RegressionAnchor {
	return planproto.AnchorToProto(a)
}

// --- enum converters ---

func phaseStatusToProto(s planmodel.PhaseStatus) sharedv1.PhaseStatus {
	return planproto.PhaseStatusToProto(s)
}

func phaseStatusFromProto(s sharedv1.PhaseStatus) planmodel.PhaseStatus {
	return planproto.PhaseStatusFromProto(s)
}

func planStatusToProto(s planmodel.PlanStatus) sharedv1.PlanStatus {
	return planproto.PlanStatusToProto(s)
}

func refKindToProto(k planmodel.ReferenceKind) sharedv1.ReferenceKind {
	return planproto.RefKindToProto(k)
}

func refResolutionToProto(r planmodel.ReferenceResolution) sharedv1.ReferenceResolution {
	return planproto.RefResolutionToProto(r)
}

func stalenessToProto(s planmodel.StalenessTier) sharedv1.StalenessTier {
	return planproto.StalenessToProto(s)
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
