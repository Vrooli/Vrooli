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
		RelevantContext: planproto.RelevantContextItemsToProto(c.RelevantContext),
		InputsFreshened: c.InputsFreshened,
		FreshenStatus:   c.FreshenStatus,
		FreshenDetail:   c.FreshenDetail,
		ChangeBoundary:  planproto.ChangeBoundaryToProto(c.ChangeBoundary),
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
	out.LogSummary = planproto.LogSummaryToProto(c.LogSummary)
	out.FeedbackCheckpoint = feedbackCheckpointToProto(c.FeedbackCheckpoint)
	return out
}

func feedbackCheckpointToProto(c internalexecution.PhaseFeedbackCheckpoint) *executionv1.PhaseFeedbackCheckpoint {
	return &executionv1.PhaseFeedbackCheckpoint{
		PhaseId:          c.PhaseID,
		Reviewed:         c.Reviewed,
		Satisfied:        c.Satisfied,
		Summary:          c.Summary,
		Decisions:        int32(c.Decisions),
		Findings:         int32(c.Findings),
		BugReports:       int32(c.BugReports),
		Records:          int32(c.Records),
		Notes:            int32(c.Notes),
		PendingSync:      int32(c.PendingSync),
		FailedSync:       int32(c.FailedSync),
		NoFeedbackTitle:  c.NoFeedbackTitle,
		NoFeedbackDetail: c.NoFeedbackDetail,
	}
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

func handoffToProto(h internalexecution.Handoff) *sharedv1.Handoff {
	out := &sharedv1.Handoff{
		Id:              h.ID,
		ExecutionId:     h.ExecutionID,
		PlanId:          h.PlanID,
		Completeness:    completenessToProto(h.Completeness),
		ResumePhaseId:   h.ResumePhaseID,
		LogSummary:      planproto.LogSummaryToProto(h.LogSummary),
		LogEntries:      planproto.LogEntriesToProto(h.LogEntries),
		Staleness:       stalenessToProto(h.Staleness),
		ProseHandoffRef: h.ProseHandoffRef,
		AssembledAt:     h.AssembledAt,
		ChangeBoundary:  planproto.ChangeBoundaryToProto(h.ChangeBoundary),
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

func guidedStepToProto(g internalexecution.GuidedStep) *sharedv1.GuidedStep {
	return &sharedv1.GuidedStep{
		StepKind:       g.StepKind,
		Title:          g.Title,
		Summary:        g.Summary,
		Instructions:   append([]string(nil), g.Instructions...),
		RequiredInputs: append([]string(nil), g.RequiredInputs...),
		Examples:       append([]string(nil), g.Examples...),
		CommonMistakes: append([]string(nil), g.CommonMistakes...),
		NextActions:    nextActionsToProto(g.NextActions),
	}
}

func nextActionsToProto(actions []internalexecution.NextAction) []*sharedv1.NextAction {
	out := make([]*sharedv1.NextAction, 0, len(actions))
	for _, action := range actions {
		out = append(out, &sharedv1.NextAction{
			Id:                 action.ID,
			Kind:               nextActionKindToProto(action.Kind),
			Label:              action.Label,
			Reason:             action.Reason,
			Argv:               append([]string(nil), action.Argv...),
			ContentPlaceholder: action.ContentPlaceholder,
			BlockedBy:          append([]string(nil), action.BlockedBy...),
		})
	}
	return out
}

func nextActionKindToProto(kind internalexecution.NextActionKind) sharedv1.NextActionKind {
	switch kind {
	case internalexecution.NextActionRecommended:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOMMENDED
	case internalexecution.NextActionAlternative:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_ALTERNATIVE
	case internalexecution.NextActionOptional:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_OPTIONAL
	case internalexecution.NextActionRecovery:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOVERY
	default:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_UNSPECIFIED
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

func planToProto(p planmodel.Plan) *sharedv1.Plan {
	return planproto.PlanToProto(p)
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
