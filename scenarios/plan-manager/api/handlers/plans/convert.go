package plans

import (
	"strings"

	"plan-manager/internal/planproto"
	internalplans "plan-manager/internal/plans"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// This file is the only translation point between the proto wire types
// (vrooli.plan_manager.v1.shared) and the plans domain vocabulary
// (internal/plans). The domain layer never imports proto (api-steer §7).

// Aliases keep the connect_handler response-construction terse without it
// importing the shared proto package directly.
type (
	sharedPlan                = sharedv1.Plan
	sharedPlanEdge            = sharedv1.PlanEdge
	sharedReference           = sharedv1.Reference
	sharedRelevantContextItem = sharedv1.RelevantContextItem
)

// orderToInt32 is a bounds-safe int→int32 conversion for a phase order (always
// small and non-negative in practice). The explicit clamp satisfies gosec G115
// without a //nosec — a phase order can never legitimately overflow int32.
func orderToInt32(n int) int32 {
	return planproto.OrderToInt32(n)
}

func planToProto(p internalplans.Plan) *sharedv1.Plan {
	return planproto.PlanToProto(p)
}

func mutationImpactToProto(impact internalplans.MutationImpact) *plansv1.PlanMutationImpact {
	return &plansv1.PlanMutationImpact{
		BeforeGrade:              impact.BeforeGrade,
		AfterGrade:               impact.AfterGrade,
		AddedIssueCodes:          impact.AddedIssueCodes,
		ClearedIssueCodes:        impact.ClearedIssueCodes,
		ExecutionGradeRegression: impact.ExecutionGradeRegression,
		RegressionAcknowledged:   impact.RegressionAcknowledged,
	}
}

func candidateRevisionToProto(candidate internalplans.CandidateRevision) *plansv1.CandidateRevision {
	return &plansv1.CandidateRevision{
		Id:                      candidate.ID,
		PlanId:                  candidate.PlanID,
		ExpectedBaseContentHash: candidate.ExpectedBaseContentHash,
		ProposalProvenance:      candidate.ProposalProvenance,
		CandidatePlan:           planToProto(candidate.CandidatePlan),
		Workspace:               workspaceScopeToProto(candidate.Workspace),
		State:                   candidateStateToProto(candidate.State),
		CreatedAt:               candidate.CreatedAt,
		UpdatedAt:               candidate.UpdatedAt,
		ExpiresAt:               candidate.ExpiresAt,
		AppliedAt:               candidate.AppliedAt,
		AppliedContentHash:      candidate.AppliedContentHash,
		DiscardReason:           candidate.DiscardReason,
	}
}

func candidateRevisionFromProto(candidate *plansv1.CandidateRevision) (internalplans.CandidateRevision, error) {
	if candidate == nil {
		return internalplans.CandidateRevision{}, internalplans.ErrInvalidPlan{Reason: "candidate is required"}
	}
	plan, err := planFromProtoChecked(candidate.GetCandidatePlan())
	if err != nil {
		return internalplans.CandidateRevision{}, err
	}
	return internalplans.CandidateRevision{
		ID:                      candidate.GetId(),
		PlanID:                  candidate.GetPlanId(),
		ExpectedBaseContentHash: candidate.GetExpectedBaseContentHash(),
		ProposalProvenance:      candidate.GetProposalProvenance(),
		CandidatePlan:           plan,
		Workspace:               workspaceScopeFromProto(candidate.GetWorkspace()),
		ExpiresAt:               candidate.GetExpiresAt(),
	}, nil
}

func candidatePreviewToProto(preview internalplans.CandidateRevisionPreview) *plansv1.CandidateRevisionPreview {
	changes := make([]*plansv1.CandidateFieldChange, 0, len(preview.Diff.Changes))
	for _, change := range preview.Diff.Changes {
		changes = append(changes, &plansv1.CandidateFieldChange{Field: change.Field, BeforeJson: change.BeforeJSON, AfterJson: change.AfterJSON})
	}
	diagnostics := make([]*plansv1.CandidateValidationDiagnostic, 0, len(preview.Diagnostics))
	for _, diagnostic := range preview.Diagnostics {
		diagnostics = append(diagnostics, &plansv1.CandidateValidationDiagnostic{Severity: diagnostic.Severity, Code: diagnostic.Code, Location: diagnostic.Location, Message: diagnostic.Message, Guidance: diagnostic.Guidance})
	}
	return &plansv1.CandidateRevisionPreview{
		Candidate:        candidateRevisionToProto(preview.Candidate),
		BasePlan:         planToProto(preview.BasePlan),
		Diff:             &plansv1.CandidateRevisionDiff{Changes: changes},
		Impact:           mutationImpactToProto(preview.Impact),
		RenderedMarkdown: preview.Rendered,
		QualityStatus:    preview.QualityStatus,
		Diagnostics:      diagnostics,
	}
}

func planFromProto(p *sharedv1.Plan) internalplans.Plan {
	return planproto.PlanFromProto(p)
}

func planFromProtoChecked(p *sharedv1.Plan) (internalplans.Plan, error) {
	if p == nil {
		return internalplans.Plan{}, nil
	}
	if err := rejectUnsupportedPlanPhaseFields(p); err != nil {
		return internalplans.Plan{}, err
	}
	return planproto.PlanFromProto(p), nil
}

func rejectUnsupportedPlanPhaseFields(p *sharedv1.Plan) error {
	for _, ph := range p.GetPhases() {
		if err := rejectUnsupportedPhaseFields(ph); err != nil {
			return err
		}
	}
	return nil
}

func phaseFromProto(ph *sharedv1.Phase) internalplans.Phase {
	return planproto.PhaseFromProto(ph)
}

func phaseFromProtoChecked(ph *sharedv1.Phase) (internalplans.Phase, error) {
	if ph == nil {
		return internalplans.Phase{}, nil
	}
	if err := rejectUnsupportedPhaseFields(ph); err != nil {
		return internalplans.Phase{}, err
	}
	return planproto.PhaseFromProto(ph), nil
}

func rejectUnsupportedPhaseFields(ph *sharedv1.Phase) error {
	var dropped []string
	if ph.GetLastValidation() != nil {
		dropped = append(dropped, "last_validation")
	}
	if len(dropped) == 0 {
		return nil
	}
	return internalplans.ErrInvalidPlan{Reason: "plans write cannot accept computed/joined phase fields: " + strings.Join(dropped, ", ")}
}

func referencesFromProto(refs []*sharedv1.Reference) []internalplans.Reference {
	out := make([]internalplans.Reference, 0, len(refs))
	for _, r := range refs {
		if r == nil {
			continue
		}
		out = append(out, internalplans.Reference{
			ID:           r.GetId(),
			Kind:         refKindFromProto(r.GetKind()),
			Target:       r.GetTarget(),
			Future:       r.GetFuture(),
			Resolution:   refResolutionFromProto(r.GetResolution()),
			Staleness:    stalenessFromProto(r.GetStaleness()),
			ChangeFactor: r.GetChangeFactor(),
			Note:         r.GetNote(),
		})
	}
	return out
}

func anchorFromProto(a *sharedv1.RegressionAnchor) internalplans.RegressionAnchor {
	if a == nil {
		return internalplans.RegressionAnchor{}
	}
	return internalplans.RegressionAnchor{
		Strategy:       a.GetStrategy(),
		Scenario:       a.GetScenario(),
		BaselineName:   a.GetBaselineName(),
		HeadSha:        a.GetHeadSha(),
		AllowlistPaths: a.GetAllowlistPaths(),
		Commands:       a.GetCommands(),
		CapturedAt:     a.GetCapturedAt(),
		Unavailable:    a.GetUnavailable(),
	}
}

func edgeToProto(e internalplans.PlanEdge) *sharedv1.PlanEdge {
	return planproto.EdgeToProto(e)
}

func reconcileRequestFromProto(req *plansv1.ReconcilePlansRequest) internalplans.ReconcileRequest {
	if req == nil {
		return internalplans.ReconcileRequest{}
	}
	return internalplans.ReconcileRequest{
		DryRun:                 req.GetDryRun(),
		RepairMirrors:          req.GetRepairMirrors(),
		SourceIntake:           req.GetSourceIntake(),
		RetireSources:          req.GetRetireSources(),
		IncludeArchived:        req.GetIncludeArchived(),
		IncludeArchivedSources: req.GetIncludeArchivedSources(),
		ConflictPolicy:         reconcileConflictPolicyFromProto(req.GetConflictPolicy()),
		Workspace:              workspaceScopeFromProto(req.GetWorkspace()),
		SourceRuntimeHomePlans: req.GetSourceRuntimeHomePlans(),
		SourceDocsPlans:        req.GetSourceDocsPlans(),
		SourceRepoPlans:        req.GetSourceRepoPlans(),
	}
}

func workspaceScopeFromProto(scope *plansv1.WorkspaceScope) internalplans.WorkspaceScope {
	if scope == nil {
		return internalplans.WorkspaceScope{}
	}
	return internalplans.WorkspaceScope{
		ID:   strings.TrimSpace(scope.GetId()),
		Root: strings.TrimSpace(scope.GetRoot()),
	}
}

func workspaceScopeToProto(scope internalplans.WorkspaceScope) *plansv1.WorkspaceScope {
	if scope.ID == "" && scope.Root == "" {
		return nil
	}
	return &plansv1.WorkspaceScope{Id: scope.ID, Root: scope.Root}
}

func candidateStateToProto(state internalplans.CandidateRevisionState) plansv1.CandidateRevisionState {
	switch state {
	case internalplans.CandidateRevisionPending:
		return plansv1.CandidateRevisionState_CANDIDATE_REVISION_STATE_PENDING
	case internalplans.CandidateRevisionApplied:
		return plansv1.CandidateRevisionState_CANDIDATE_REVISION_STATE_APPLIED
	case internalplans.CandidateRevisionDiscarded:
		return plansv1.CandidateRevisionState_CANDIDATE_REVISION_STATE_DISCARDED
	case internalplans.CandidateRevisionExpired:
		return plansv1.CandidateRevisionState_CANDIDATE_REVISION_STATE_EXPIRED
	default:
		return plansv1.CandidateRevisionState_CANDIDATE_REVISION_STATE_UNSPECIFIED
	}
}

func reconcileResultToProto(result internalplans.ReconcileResult) *plansv1.ReconcilePlansResponse {
	resp := &plansv1.ReconcilePlansResponse{
		DryRun: result.DryRun,
		Items:  make([]*plansv1.ReconcilePlanItem, 0, len(result.Items)),
	}
	for _, item := range result.Items {
		resp.Items = append(resp.Items, &plansv1.ReconcilePlanItem{
			Action:                  reconcileActionToProto(item.Action),
			PlanId:                  item.PlanID,
			Slug:                    item.Slug,
			Title:                   item.Title,
			SourcePath:              item.SourcePath,
			Mirror:                  planproto.MirrorToProto(item.Mirror),
			SourceUntouched:         item.SourceUntouched,
			Error:                   item.Error,
			SourceRetirementPlanned: item.SourceRetirementPlanned,
			SourceRemoved:           item.SourceRemoved,
		})
	}
	return resp
}

// --- enum converters ---

func planStatusToProto(s internalplans.PlanStatus) sharedv1.PlanStatus {
	return planproto.PlanStatusToProto(s)
}

func planStatusFromProto(s sharedv1.PlanStatus) internalplans.PlanStatus {
	return planproto.PlanStatusFromProto(s)
}

func phaseStatusToProto(s internalplans.PhaseStatus) sharedv1.PhaseStatus {
	return planproto.PhaseStatusToProto(s)
}

func phaseStatusFromProto(s sharedv1.PhaseStatus) internalplans.PhaseStatus {
	return planproto.PhaseStatusFromProto(s)
}

func refKindToProto(k internalplans.ReferenceKind) sharedv1.ReferenceKind {
	return planproto.RefKindToProto(k)
}

func refKindFromProto(k sharedv1.ReferenceKind) internalplans.ReferenceKind {
	return planproto.RefKindFromProto(k)
}

func refResolutionToProto(r internalplans.ReferenceResolution) sharedv1.ReferenceResolution {
	return planproto.RefResolutionToProto(r)
}

func refResolutionFromProto(r sharedv1.ReferenceResolution) internalplans.ReferenceResolution {
	return planproto.RefResolutionFromProto(r)
}

func stalenessToProto(s internalplans.StalenessTier) sharedv1.StalenessTier {
	return planproto.StalenessToProto(s)
}

func stalenessFromProto(s sharedv1.StalenessTier) internalplans.StalenessTier {
	return planproto.StalenessFromProto(s)
}

func reconcileConflictPolicyFromProto(p plansv1.ReconcileConflictPolicy) internalplans.ReconcileConflictPolicy {
	switch p {
	case plansv1.ReconcileConflictPolicy_RECONCILE_CONFLICT_POLICY_REPORT_ONLY:
		return internalplans.ReconcileConflictReportOnly
	case plansv1.ReconcileConflictPolicy_RECONCILE_CONFLICT_POLICY_SKIP_EXISTING:
		return internalplans.ReconcileConflictSkipExisting
	default:
		return ""
	}
}

func reconcileActionToProto(a internalplans.ReconcileAction) plansv1.ReconcileAction {
	switch a {
	case internalplans.ReconcileActionAlreadyCanonical:
		return plansv1.ReconcileAction_RECONCILE_ACTION_ALREADY_CANONICAL
	case internalplans.ReconcileActionMirrorFresh:
		return plansv1.ReconcileAction_RECONCILE_ACTION_MIRROR_FRESH
	case internalplans.ReconcileActionMirrorRepairNeeded:
		return plansv1.ReconcileAction_RECONCILE_ACTION_MIRROR_REPAIR_NEEDED
	case internalplans.ReconcileActionMirrorRepaired:
		return plansv1.ReconcileAction_RECONCILE_ACTION_MIRROR_REPAIRED
	case internalplans.ReconcileActionImportPlanned:
		return plansv1.ReconcileAction_RECONCILE_ACTION_IMPORT_PLANNED
	case internalplans.ReconcileActionImported:
		return plansv1.ReconcileAction_RECONCILE_ACTION_IMPORTED
	case internalplans.ReconcileActionSkippedDuplicate:
		return plansv1.ReconcileAction_RECONCILE_ACTION_SKIPPED_DUPLICATE
	case internalplans.ReconcileActionParseFailed:
		return plansv1.ReconcileAction_RECONCILE_ACTION_PARSE_FAILED
	case internalplans.ReconcileActionConflict:
		return plansv1.ReconcileAction_RECONCILE_ACTION_CONFLICT
	default:
		return plansv1.ReconcileAction_RECONCILE_ACTION_UNSPECIFIED
	}
}
