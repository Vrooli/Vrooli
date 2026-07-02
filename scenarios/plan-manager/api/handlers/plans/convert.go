package plans

import (
	"fmt"
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

func planFromProto(p *sharedv1.Plan) internalplans.Plan {
	out, _ := planFromProtoChecked(p)
	return out
}

func planFromProtoChecked(p *sharedv1.Plan) (internalplans.Plan, error) {
	if p == nil {
		return internalplans.Plan{}, nil
	}
	phases, err := phasesFromProtoChecked(p.GetPhases())
	if err != nil {
		return internalplans.Plan{}, err
	}
	return internalplans.Plan{
		ID:                      p.GetId(),
		Slug:                    p.GetSlug(),
		Title:                   p.GetTitle(),
		Status:                  planStatusFromProto(p.GetStatus()),
		ContentHash:             p.GetContentHash(),
		CreatedAt:               p.GetCreatedAt(),
		UpdatedAt:               p.GetUpdatedAt(),
		WorkspaceID:             p.GetWorkspaceId(),
		WorkspaceRoot:           p.GetWorkspaceRoot(),
		Purpose:                 p.GetPurpose(),
		Scope:                   p.GetScope(),
		Constraints:             p.GetConstraints(),
		NonGoals:                p.GetNonGoals(),
		References:              referencesFromProto(p.GetReferences()),
		RegressionAnchor:        anchorFromProto(p.GetRegressionAnchor()),
		DefinitionOfDone:        p.GetDefinitionOfDone(),
		Phases:                  phases,
		Supersedes:              p.GetSupersedes(),
		SupersededBy:            p.GetSupersededBy(),
		RelevantContext:         planproto.RelevantContextItemsFromProto(p.GetRelevantContext()),
		ProblemStatement:        p.GetProblemStatement(),
		TargetOutcome:           p.GetTargetOutcome(),
		Assumptions:             p.GetAssumptions(),
		TechnicalApproach:       p.GetTechnicalApproach(),
		ValidationStrategy:      p.GetValidationStrategy(),
		FinalValidationCommands: p.GetFinalValidationCommands(),
		RisksHazards:            p.GetRisksHazards(),
		ProhibitedApproaches:    p.GetProhibitedApproaches(),
		WorkPosture:             planproto.WorkPostureFromProto(p.GetWorkPosture()),
		WorkPostureSource:       planproto.WorkPostureSourceFromProto(p.GetWorkPostureSource()),
		WorkPostureDetail:       p.GetWorkPostureDetail(),
		ImportProvenance:        planproto.ImportProvenanceFromProto(p.GetImportProvenance()),
		PreservedLegacySections: planproto.LegacySectionsFromProto(p.GetPreservedLegacySections()),
		Mirror:                  planproto.MirrorFromProto(p.GetMirror()),
	}, nil
}

func phasesFromProtoChecked(phases []*sharedv1.Phase) ([]internalplans.Phase, error) {
	out := make([]internalplans.Phase, 0, len(phases))
	for i, ph := range phases {
		converted, err := phaseFromProtoChecked(ph)
		if err != nil {
			return nil, fmt.Errorf("phase %d: %w", i, err)
		}
		out = append(out, converted)
	}
	return out, nil
}

func phaseFromProto(ph *sharedv1.Phase) internalplans.Phase {
	out, _ := phaseFromProtoChecked(ph)
	return out
}

func phaseFromProtoChecked(ph *sharedv1.Phase) (internalplans.Phase, error) {
	if ph == nil {
		return internalplans.Phase{}, nil
	}
	if err := rejectUnsupportedPhaseFields(ph); err != nil {
		return internalplans.Phase{}, err
	}
	return internalplans.Phase{
		ID:              ph.GetId(),
		Order:           int(ph.GetOrder()),
		Title:           ph.GetTitle(),
		Intent:          ph.GetIntent(),
		RequiredReading: ph.GetRequiredReading(),
		Reminders:       ph.GetReminders(),
		BaselineScope:   ph.GetBaselineScope(),
		Acceptance:      ph.GetAcceptance(),
		Status:          phaseStatusFromProto(ph.GetStatus()),
		References:      referencesFromProto(ph.GetReferences()),
		RelevantContext: planproto.RelevantContextItemsFromProto(ph.GetRelevantContext()),
		AffectedAreas:   ph.GetAffectedAreas(),
		Steps:           ph.GetSteps(),
		ExpectedOutputs: ph.GetExpectedOutputs(),
		Validation:      ph.GetValidation(),
		HandoffNotes:    ph.GetHandoffNotes(),
		RisksHazards:    ph.GetRisksHazards(),
	}, nil
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
		AdoptLegacy:            req.GetAdoptLegacy(),
		CleanupAdoptedSources:  req.GetCleanupAdoptedSources(),
		IncludeArchived:        req.GetIncludeArchived(),
		IncludeArchivedLegacy:  req.GetIncludeArchivedLegacy(),
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

func reconcileResultToProto(result internalplans.ReconcileResult) *plansv1.ReconcilePlansResponse {
	resp := &plansv1.ReconcilePlansResponse{
		DryRun: result.DryRun,
		Items:  make([]*plansv1.ReconcilePlanItem, 0, len(result.Items)),
	}
	for _, item := range result.Items {
		resp.Items = append(resp.Items, &plansv1.ReconcilePlanItem{
			Action:               reconcileActionToProto(item.Action),
			PlanId:               item.PlanID,
			Slug:                 item.Slug,
			Title:                item.Title,
			SourcePath:           item.SourcePath,
			Mirror:               planproto.MirrorToProto(item.Mirror),
			SourceUntouched:      item.SourceUntouched,
			Error:                item.Error,
			SourceCleanupPlanned: item.SourceCleanupPlanned,
			SourceRemoved:        item.SourceRemoved,
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
