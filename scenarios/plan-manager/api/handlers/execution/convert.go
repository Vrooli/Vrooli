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
		Id:              e.ID,
		PlanId:          e.PlanID,
		RunId:           e.RunID,
		CurrentPhaseId:  e.CurrentPhaseID,
		Complete:        e.Complete,
		StartedAt:       e.StartedAt,
		UpdatedAt:       e.UpdatedAt,
		BaselineSet:     baselineSetToProto(e.BaselineSet),
		ScopeAmendments: scopeAmendmentsToProto(e.ScopeAmendments),
		DegradedReason:  e.DegradedReason,
	}
}

func scopeAmendmentsToProto(items []internalexecution.ScopeAmendment) []*executionv1.ScopeAmendment {
	out := make([]*executionv1.ScopeAmendment, 0, len(items))
	for _, item := range items {
		out = append(out, &executionv1.ScopeAmendment{Id: item.ID, PhaseId: item.PhaseID, Author: item.Author, Reason: item.Reason, OldMinimum: append([]string(nil), item.OldMinimum...), NewMinimum: append([]string(nil), item.NewMinimum...), InvalidatedAt: item.InvalidatedAt, CreatedAt: item.CreatedAt, InvalidatedTicketIds: append([]string(nil), item.InvalidatedTicketIDs...)})
	}
	return out
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
		BaselineSet:     baselineSetToProto(c.BaselineSet),
		ScopeGeneration: int32(c.ScopeGeneration),
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

func baselineSetToProto(state internalexecution.BaselineSetState) *executionv1.BaselineSetState {
	if state.Name == "" {
		return nil
	}
	return &executionv1.BaselineSetState{
		Version: int32(state.Version), Name: state.Name,
		CollectionBranch: state.CollectionBranch,
		ScenarioTargets:  append([]string(nil), state.ScenarioTargets...), RepoPaths: append([]string(nil), state.RepoPaths...),
		CapturedAt: state.CapturedAt, Status: string(state.Status), Required: int32(state.Required), Ready: int32(state.Ready),
		Pending: int32(state.Pending), Failed: int32(state.Failed), Skipped: int32(state.Skipped), Stale: int32(state.Stale), Detail: state.Detail,
		Members: baselineSetMembersToProto(state.Members), PathSnapshots: baselineSetPathSnapshotsToProto(state.PathSnapshots),
		CaptureArgv: append([]string(nil), state.CaptureArgv...), WaitArgv: append([]string(nil), state.WaitArgv...), SyncArgv: append([]string(nil), state.SyncArgv...), LastSyncedAt: state.LastSyncedAt,
		SourcePreflight: sourcePreflightToProto(state.SourcePreflight), PreflightUnavailable: state.PreflightUnavailable,
	}
}

func sourcePreflightToProto(preflight internalexecution.SourceEvidencePreflight) *executionv1.SourceEvidencePreflight {
	if preflight.PolicyVersion == 0 && preflight.EligibleFiles == 0 && preflight.EligibleBytes == 0 && !preflight.RepairRequired && len(preflight.Issues) == 0 && len(preflight.Recommendations) == 0 {
		return nil
	}
	out := &executionv1.SourceEvidencePreflight{
		PolicyVersion: int32(preflight.PolicyVersion), IncludeIgnored: preflight.IncludeIgnored, RetainContent: preflight.RetainContent,
		EligibleFiles: int32(preflight.EligibleFiles), EligibleBytes: preflight.EligibleBytes,
		ExcludedIgnoredFiles: int32(preflight.ExcludedIgnoredFiles), ExcludedIgnoredBytes: preflight.ExcludedIgnoredBytes,
		ExcludedSensitiveFiles: int32(preflight.ExcludedSensitiveFiles), ExcludedBinaryFiles: int32(preflight.ExcludedBinaryFiles), OversizedFiles: int32(preflight.OversizedFiles),
		RetainedContentBytes: preflight.RetainedContentBytes, RepairRequired: preflight.RepairRequired,
	}
	for _, contributor := range preflight.TopContributors {
		out.TopContributors = append(out.TopContributors, &executionv1.SourceEvidenceContributor{Path: contributor.Path, Files: int32(contributor.Files), Bytes: contributor.Bytes})
	}
	for _, issue := range preflight.Issues {
		out.Issues = append(out.Issues, &executionv1.SourceEvidenceIssue{Code: issue.Code, Severity: issue.Severity, Detail: issue.Detail})
	}
	for _, recommendation := range preflight.Recommendations {
		out.Recommendations = append(out.Recommendations, &executionv1.SourceEvidenceRecommendation{Selection: recommendation.Selection, Reason: recommendation.Reason})
	}
	return out
}

func baselineSetMembersToProto(members []internalexecution.BaselineSetMember) []*executionv1.BaselineSetMember {
	out := make([]*executionv1.BaselineSetMember, 0, len(members))
	for _, member := range members {
		out = append(out, &executionv1.BaselineSetMember{Scenario: member.Scenario, BaselineName: member.BaselineName, Required: member.Required, Status: member.Status, RunId: member.RunID, GitSha: member.GitSHA, Error: member.Error})
	}
	return out
}

func baselineSetPathSnapshotsToProto(snapshots []internalexecution.BaselineSetPathSnapshot) []*executionv1.BaselineSetPathSnapshot {
	out := make([]*executionv1.BaselineSetPathSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		out = append(out, &executionv1.BaselineSetPathSnapshot{Name: snapshot.Name, Branch: snapshot.Branch, CreatedAt: snapshot.CreatedAt})
	}
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
	return planproto.GuidedStepToProto(g)
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
		Id:              r.ID,
		PlanId:          r.PlanID,
		PhaseId:         r.PhaseID,
		Verdict:         verdictToProto(r.Verdict),
		Staleness:       stalenessToProto(r.Staleness),
		CommandsRun:     r.CommandsRun,
		Detail:          r.Detail,
		RanAt:           r.RanAt,
		ExecutionId:     r.ExecutionID,
		OperationId:     r.OperationID,
		ScopeGeneration: int32(r.ScopeGeneration),
		FullInventory:   r.FullInventory,
		RequiredMembers: append([]string(nil), r.RequiredMembers...),
		SelectedMembers: append([]string(nil), r.SelectedMembers...),
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
