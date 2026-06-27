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
		RelevantContext:  RelevantContextItemsToProto(p.RelevantContext),
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
		RelevantContext: RelevantContextItemsToProto(ph.RelevantContext),
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

func RelevantContextItemsToProto(items []planmodel.RelevantContextItem) []*sharedv1.RelevantContextItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]*sharedv1.RelevantContextItem, 0, len(items))
	for _, item := range items {
		out = append(out, &sharedv1.RelevantContextItem{
			Id:           item.ID,
			Kind:         RelevantContextKindToProto(item.Kind),
			Scope:        RelevantContextScopeToProto(item.Scope),
			PhaseId:      item.PhaseID,
			Label:        item.Label,
			Reason:       item.Reason,
			Instruction:  item.Instruction,
			Command:      item.Command,
			Argv:         item.Argv,
			Target:       item.Target,
			Required:     item.Required,
			RepeatPolicy: RelevantContextRepeatPolicyToProto(item.RepeatPolicy),
			Source:       RelevantContextSourceToProto(item.Source),
			Status:       RelevantContextStatusToProto(item.Status),
			StatusDetail: item.StatusDetail,
		})
	}
	return out
}

func RelevantContextItemsFromProto(items []*sharedv1.RelevantContextItem) []planmodel.RelevantContextItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]planmodel.RelevantContextItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, planmodel.RelevantContextItem{
			ID:           item.GetId(),
			Kind:         RelevantContextKindFromProto(item.GetKind()),
			Scope:        RelevantContextScopeFromProto(item.GetScope()),
			PhaseID:      item.GetPhaseId(),
			Label:        item.GetLabel(),
			Reason:       item.GetReason(),
			Instruction:  item.GetInstruction(),
			Command:      item.GetCommand(),
			Argv:         item.GetArgv(),
			Target:       item.GetTarget(),
			Required:     item.GetRequired(),
			RepeatPolicy: RelevantContextRepeatPolicyFromProto(item.GetRepeatPolicy()),
			Source:       RelevantContextSourceFromProto(item.GetSource()),
			Status:       RelevantContextStatusFromProto(item.GetStatus()),
			StatusDetail: item.GetStatusDetail(),
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

func RelevantContextKindToProto(k planmodel.RelevantContextKind) sharedv1.RelevantContextKind {
	switch k {
	case planmodel.RelevantContextSkill:
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SKILL
	case planmodel.RelevantContextDoc:
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_DOC
	case planmodel.RelevantContextCommand:
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND
	case planmodel.RelevantContextSearch:
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SEARCH
	case planmodel.RelevantContextCodeRef:
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_CODE_REF
	case planmodel.RelevantContextReqRef:
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_REQ_REF
	case planmodel.RelevantContextNote:
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_NOTE
	default:
		return sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_UNSPECIFIED
	}
}

func RelevantContextKindFromProto(k sharedv1.RelevantContextKind) planmodel.RelevantContextKind {
	switch k {
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SKILL:
		return planmodel.RelevantContextSkill
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_DOC:
		return planmodel.RelevantContextDoc
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND:
		return planmodel.RelevantContextCommand
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SEARCH:
		return planmodel.RelevantContextSearch
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_CODE_REF:
		return planmodel.RelevantContextCodeRef
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_REQ_REF:
		return planmodel.RelevantContextReqRef
	case sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_NOTE:
		return planmodel.RelevantContextNote
	default:
		return ""
	}
}

func RelevantContextScopeToProto(s planmodel.RelevantContextScope) sharedv1.RelevantContextScope {
	switch s {
	case planmodel.RelevantContextScopeGlobal:
		return sharedv1.RelevantContextScope_RELEVANT_CONTEXT_SCOPE_GLOBAL
	case planmodel.RelevantContextScopePhase:
		return sharedv1.RelevantContextScope_RELEVANT_CONTEXT_SCOPE_PHASE
	default:
		return sharedv1.RelevantContextScope_RELEVANT_CONTEXT_SCOPE_UNSPECIFIED
	}
}

func RelevantContextScopeFromProto(s sharedv1.RelevantContextScope) planmodel.RelevantContextScope {
	switch s {
	case sharedv1.RelevantContextScope_RELEVANT_CONTEXT_SCOPE_GLOBAL:
		return planmodel.RelevantContextScopeGlobal
	case sharedv1.RelevantContextScope_RELEVANT_CONTEXT_SCOPE_PHASE:
		return planmodel.RelevantContextScopePhase
	default:
		return ""
	}
}

func RelevantContextRepeatPolicyToProto(p planmodel.RelevantContextRepeatPolicy) sharedv1.RelevantContextRepeatPolicy {
	switch p {
	case planmodel.RelevantContextOncePerExecution:
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_ONCE_PER_EXECUTION
	case planmodel.RelevantContextOnResume:
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_ON_RESUME
	case planmodel.RelevantContextEveryPhase:
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_EVERY_PHASE
	case planmodel.RelevantContextPhaseEntry:
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_PHASE_ENTRY
	case planmodel.RelevantContextAsNeeded:
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_AS_NEEDED
	default:
		return sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_UNSPECIFIED
	}
}

func RelevantContextRepeatPolicyFromProto(p sharedv1.RelevantContextRepeatPolicy) planmodel.RelevantContextRepeatPolicy {
	switch p {
	case sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_ONCE_PER_EXECUTION:
		return planmodel.RelevantContextOncePerExecution
	case sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_ON_RESUME:
		return planmodel.RelevantContextOnResume
	case sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_EVERY_PHASE:
		return planmodel.RelevantContextEveryPhase
	case sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_PHASE_ENTRY:
		return planmodel.RelevantContextPhaseEntry
	case sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_AS_NEEDED:
		return planmodel.RelevantContextAsNeeded
	default:
		return ""
	}
}

func RelevantContextSourceToProto(s planmodel.RelevantContextSource) sharedv1.RelevantContextSource {
	switch s {
	case planmodel.RelevantContextSourceAuthored:
		return sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_AUTHORED
	case planmodel.RelevantContextSourceDiscovered:
		return sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_DISCOVERED
	case planmodel.RelevantContextSourceMigrated:
		return sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_MIGRATED
	case planmodel.RelevantContextSourceAutofilled:
		return sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_AUTOFILLED
	default:
		return sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_UNSPECIFIED
	}
}

func RelevantContextSourceFromProto(s sharedv1.RelevantContextSource) planmodel.RelevantContextSource {
	switch s {
	case sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_AUTHORED:
		return planmodel.RelevantContextSourceAuthored
	case sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_DISCOVERED:
		return planmodel.RelevantContextSourceDiscovered
	case sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_MIGRATED:
		return planmodel.RelevantContextSourceMigrated
	case sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_AUTOFILLED:
		return planmodel.RelevantContextSourceAutofilled
	default:
		return ""
	}
}

func RelevantContextStatusToProto(s planmodel.RelevantContextStatus) sharedv1.RelevantContextStatus {
	switch s {
	case planmodel.RelevantContextStatusReady:
		return sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_READY
	case planmodel.RelevantContextStatusDegraded:
		return sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_DEGRADED
	case planmodel.RelevantContextStatusUnresolved:
		return sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_UNRESOLVED
	default:
		return sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_UNSPECIFIED
	}
}

func RelevantContextStatusFromProto(s sharedv1.RelevantContextStatus) planmodel.RelevantContextStatus {
	switch s {
	case sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_READY:
		return planmodel.RelevantContextStatusReady
	case sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_DEGRADED:
		return planmodel.RelevantContextStatusDegraded
	case sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_UNRESOLVED:
		return planmodel.RelevantContextStatusUnresolved
	default:
		return ""
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
