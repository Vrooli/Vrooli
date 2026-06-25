package plans

import (
	"math"

	internalplans "plan-manager/internal/plans"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// This file is the only translation point between the proto wire types
// (vrooli.plan_manager.v1.shared) and the plans domain vocabulary
// (internal/plans). The domain layer never imports proto (api-steer §7).

// Aliases keep the connect_handler response-construction terse without it
// importing the shared proto package directly.
type (
	sharedPlan     = sharedv1.Plan
	sharedPlanEdge = sharedv1.PlanEdge
)

// orderToInt32 is a bounds-safe int→int32 conversion for a phase order (always
// small and non-negative in practice). The explicit clamp satisfies gosec G115
// without a //nosec — a phase order can never legitimately overflow int32.
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

func planFromProto(p *sharedv1.Plan) internalplans.Plan {
	if p == nil {
		return internalplans.Plan{}
	}
	return internalplans.Plan{
		ID:               p.GetId(),
		Slug:             p.GetSlug(),
		Title:            p.GetTitle(),
		Status:           planStatusFromProto(p.GetStatus()),
		ContentHash:      p.GetContentHash(),
		CreatedAt:        p.GetCreatedAt(),
		UpdatedAt:        p.GetUpdatedAt(),
		Purpose:          p.GetPurpose(),
		Scope:            p.GetScope(),
		Constraints:      p.GetConstraints(),
		NonGoals:         p.GetNonGoals(),
		References:       referencesFromProto(p.GetReferences()),
		RegressionAnchor: anchorFromProto(p.GetRegressionAnchor()),
		DefinitionOfDone: p.GetDefinitionOfDone(),
		Phases:           phasesFromProto(p.GetPhases()),
		Supersedes:       p.GetSupersedes(),
		SupersededBy:     p.GetSupersededBy(),
	}
}

func phasesToProto(phases []internalplans.Phase) []*sharedv1.Phase {
	out := make([]*sharedv1.Phase, 0, len(phases))
	for _, ph := range phases {
		out = append(out, phaseToProto(ph))
	}
	return out
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

func phasesFromProto(phases []*sharedv1.Phase) []internalplans.Phase {
	out := make([]internalplans.Phase, 0, len(phases))
	for _, ph := range phases {
		out = append(out, phaseFromProto(ph))
	}
	return out
}

func phaseFromProto(ph *sharedv1.Phase) internalplans.Phase {
	if ph == nil {
		return internalplans.Phase{}
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
	return &sharedv1.PlanEdge{
		FromPlanId: e.FromPlanID,
		ToPlanId:   e.ToPlanID,
		Kind:       e.Kind,
	}
}

// --- enum converters ---

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

func planStatusFromProto(s sharedv1.PlanStatus) internalplans.PlanStatus {
	switch s {
	case sharedv1.PlanStatus_PLAN_STATUS_DRAFT:
		return internalplans.PlanStatusDraft
	case sharedv1.PlanStatus_PLAN_STATUS_ACTIVE:
		return internalplans.PlanStatusActive
	case sharedv1.PlanStatus_PLAN_STATUS_COMPLETE:
		return internalplans.PlanStatusComplete
	case sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED:
		return internalplans.PlanStatusArchived
	default:
		return ""
	}
}

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

func refKindFromProto(k sharedv1.ReferenceKind) internalplans.ReferenceKind {
	switch k {
	case sharedv1.ReferenceKind_REFERENCE_KIND_REQ:
		return internalplans.ReferenceReq
	case sharedv1.ReferenceKind_REFERENCE_KIND_DOC:
		return internalplans.ReferenceDoc
	case sharedv1.ReferenceKind_REFERENCE_KIND_CODE:
		return internalplans.ReferenceCode
	default:
		return internalplans.ReferenceCode
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

func refResolutionFromProto(r sharedv1.ReferenceResolution) internalplans.ReferenceResolution {
	switch r {
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED:
		return internalplans.ResolutionResolved
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED:
		return internalplans.ResolutionUnresolved
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE:
		return internalplans.ResolutionFuture
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING:
		return internalplans.ResolutionMissing
	default:
		return internalplans.ResolutionUnspecified
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

func stalenessFromProto(s sharedv1.StalenessTier) internalplans.StalenessTier {
	switch s {
	case sharedv1.StalenessTier_STALENESS_TIER_FRESH:
		return internalplans.StalenessFresh
	case sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE:
		return internalplans.StalenessLightlyStale
	case sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE:
		return internalplans.StalenessDefinitelyStale
	default:
		return internalplans.StalenessUnknown
	}
}
