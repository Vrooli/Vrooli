// Package planproto converts between the neutral planmodel kernel and the
// shared plan-manager proto messages.
package planproto

import (
	"math"

	"plan-manager/internal/planmodel"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// OrderToInt32 is a bounds-safe int to int32 conversion for phase orders.
func OrderToInt32(n int) int32 {
	switch {
	case n < 0:
		return 0
	case n > math.MaxInt32:
		return math.MaxInt32
	default:
		return int32(n)
	}
}

func PlanToProto(p planmodel.Plan) *sharedv1.Plan {
	return &sharedv1.Plan{
		Id:               p.ID,
		Slug:             p.Slug,
		Title:            p.Title,
		Status:           PlanStatusToProto(p.Status),
		ContentHash:      p.ContentHash,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
		Purpose:          p.Purpose,
		Scope:            p.Scope,
		Constraints:      p.Constraints,
		NonGoals:         p.NonGoals,
		References:       ReferencesToProto(p.References),
		RegressionAnchor: AnchorToProto(p.RegressionAnchor),
		DefinitionOfDone: p.DefinitionOfDone,
		Phases:           PhasesToProto(p.Phases),
		Supersedes:       p.Supersedes,
		SupersededBy:     p.SupersededBy,
	}
}

func PhaseToProto(ph planmodel.Phase) *sharedv1.Phase {
	return &sharedv1.Phase{
		Id:              ph.ID,
		Order:           OrderToInt32(ph.Order),
		Title:           ph.Title,
		Intent:          ph.Intent,
		RequiredReading: ph.RequiredReading,
		Reminders:       ph.Reminders,
		BaselineScope:   ph.BaselineScope,
		Acceptance:      ph.Acceptance,
		Status:          PhaseStatusToProto(ph.Status),
		References:      ReferencesToProto(ph.References),
	}
}

func PhasesToProto(phases []planmodel.Phase) []*sharedv1.Phase {
	out := make([]*sharedv1.Phase, 0, len(phases))
	for _, ph := range phases {
		out = append(out, PhaseToProto(ph))
	}
	return out
}

func ReferencesToProto(refs []planmodel.Reference) []*sharedv1.Reference {
	out := make([]*sharedv1.Reference, 0, len(refs))
	for _, r := range refs {
		out = append(out, &sharedv1.Reference{
			Id:           r.ID,
			Kind:         RefKindToProto(r.Kind),
			Target:       r.Target,
			Future:       r.Future,
			Resolution:   RefResolutionToProto(r.Resolution),
			Staleness:    StalenessToProto(r.Staleness),
			ChangeFactor: r.ChangeFactor,
			Note:         r.Note,
		})
	}
	return out
}

func AnchorToProto(a planmodel.RegressionAnchor) *sharedv1.RegressionAnchor {
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

func EdgeToProto(e planmodel.PlanEdge) *sharedv1.PlanEdge {
	return &sharedv1.PlanEdge{
		FromPlanId: e.FromPlanID,
		ToPlanId:   e.ToPlanID,
		Kind:       e.Kind,
	}
}

func PlanStatusToProto(s planmodel.PlanStatus) sharedv1.PlanStatus {
	switch s {
	case planmodel.PlanStatusDraft:
		return sharedv1.PlanStatus_PLAN_STATUS_DRAFT
	case planmodel.PlanStatusActive:
		return sharedv1.PlanStatus_PLAN_STATUS_ACTIVE
	case planmodel.PlanStatusComplete:
		return sharedv1.PlanStatus_PLAN_STATUS_COMPLETE
	case planmodel.PlanStatusArchived:
		return sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED
	default:
		return sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED
	}
}

func PlanStatusFromProto(s sharedv1.PlanStatus) planmodel.PlanStatus {
	switch s {
	case sharedv1.PlanStatus_PLAN_STATUS_DRAFT:
		return planmodel.PlanStatusDraft
	case sharedv1.PlanStatus_PLAN_STATUS_ACTIVE:
		return planmodel.PlanStatusActive
	case sharedv1.PlanStatus_PLAN_STATUS_COMPLETE:
		return planmodel.PlanStatusComplete
	case sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED:
		return planmodel.PlanStatusArchived
	default:
		return ""
	}
}

func PhaseStatusToProto(s planmodel.PhaseStatus) sharedv1.PhaseStatus {
	switch s {
	case planmodel.PhaseStatusTodo:
		return sharedv1.PhaseStatus_PHASE_STATUS_TODO
	case planmodel.PhaseStatusActive:
		return sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE
	case planmodel.PhaseStatusDone:
		return sharedv1.PhaseStatus_PHASE_STATUS_DONE
	case planmodel.PhaseStatusBlocked:
		return sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED
	default:
		return sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED
	}
}

func PhaseStatusFromProto(s sharedv1.PhaseStatus) planmodel.PhaseStatus {
	switch s {
	case sharedv1.PhaseStatus_PHASE_STATUS_TODO:
		return planmodel.PhaseStatusTodo
	case sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE:
		return planmodel.PhaseStatusActive
	case sharedv1.PhaseStatus_PHASE_STATUS_DONE:
		return planmodel.PhaseStatusDone
	case sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED:
		return planmodel.PhaseStatusBlocked
	default:
		return ""
	}
}

func RefKindToProto(k planmodel.ReferenceKind) sharedv1.ReferenceKind {
	switch k {
	case planmodel.ReferenceCode:
		return sharedv1.ReferenceKind_REFERENCE_KIND_CODE
	case planmodel.ReferenceReq:
		return sharedv1.ReferenceKind_REFERENCE_KIND_REQ
	case planmodel.ReferenceDoc:
		return sharedv1.ReferenceKind_REFERENCE_KIND_DOC
	default:
		return sharedv1.ReferenceKind_REFERENCE_KIND_UNSPECIFIED
	}
}

func RefKindFromProto(k sharedv1.ReferenceKind) planmodel.ReferenceKind {
	switch k {
	case sharedv1.ReferenceKind_REFERENCE_KIND_REQ:
		return planmodel.ReferenceReq
	case sharedv1.ReferenceKind_REFERENCE_KIND_DOC:
		return planmodel.ReferenceDoc
	case sharedv1.ReferenceKind_REFERENCE_KIND_CODE:
		return planmodel.ReferenceCode
	default:
		return planmodel.ReferenceCode
	}
}

func RefResolutionToProto(r planmodel.ReferenceResolution) sharedv1.ReferenceResolution {
	switch r {
	case planmodel.ResolutionResolved:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED
	case planmodel.ResolutionUnresolved:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED
	case planmodel.ResolutionFuture:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE
	case planmodel.ResolutionMissing:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING
	default:
		return sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNSPECIFIED
	}
}

func RefResolutionFromProto(r sharedv1.ReferenceResolution) planmodel.ReferenceResolution {
	switch r {
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED:
		return planmodel.ResolutionResolved
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED:
		return planmodel.ResolutionUnresolved
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE:
		return planmodel.ResolutionFuture
	case sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING:
		return planmodel.ResolutionMissing
	default:
		return planmodel.ResolutionUnspecified
	}
}

func StalenessToProto(s planmodel.StalenessTier) sharedv1.StalenessTier {
	switch s {
	case planmodel.StalenessFresh:
		return sharedv1.StalenessTier_STALENESS_TIER_FRESH
	case planmodel.StalenessLightlyStale:
		return sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE
	case planmodel.StalenessDefinitelyStale:
		return sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE
	default:
		return sharedv1.StalenessTier_STALENESS_TIER_UNSPECIFIED
	}
}

func StalenessFromProto(s sharedv1.StalenessTier) planmodel.StalenessTier {
	switch s {
	case sharedv1.StalenessTier_STALENESS_TIER_FRESH:
		return planmodel.StalenessFresh
	case sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE:
		return planmodel.StalenessLightlyStale
	case sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE:
		return planmodel.StalenessDefinitelyStale
	default:
		return planmodel.StalenessUnknown
	}
}
