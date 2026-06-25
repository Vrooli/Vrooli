package authoring

import (
	internalauthoring "plan-manager/internal/authoring"
	internalplans "plan-manager/internal/plans"

	authoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// This file is the only translation point between the proto wire types
// (vrooli.plan_manager.v1.authoring + .shared) and the authoring/plans domain
// vocabulary. The domain layer never imports proto (api-steer §7).

func sessionToProto(s internalauthoring.Session) *authoringv1.AuthoringSession {
	return &authoringv1.AuthoringSession{
		Id:                s.ID,
		Title:             s.Title,
		PlanSlug:          s.Slug,
		Sections:          sectionsToProto(s.Sections),
		CurrentSectionKey: string(s.CurrentSectionKey),
		Finalized:         s.Finalized,
		PlanId:            s.PlanID,
	}
}

func sectionsToProto(sections []internalauthoring.Section) []*authoringv1.Section {
	out := make([]*authoringv1.Section, 0, len(sections))
	for _, sec := range sections {
		out = append(out, sectionToProto(sec))
	}
	return out
}

func sectionToProto(sec internalauthoring.Section) *authoringv1.Section {
	return &authoringv1.Section{
		Key:        string(sec.Key),
		Label:      sec.Label,
		Content:    sec.Content,
		Mandatory:  sec.Mandatory,
		Filled:     sec.Filled,
		Autofilled: sec.Autofilled,
	}
}

func violationsToProto(violations []internalauthoring.StructureViolation) []*authoringv1.StructureViolation {
	out := make([]*authoringv1.StructureViolation, 0, len(violations))
	for _, v := range violations {
		out = append(out, &authoringv1.StructureViolation{
			SectionKey: string(v.SectionKey),
			Message:    v.Message,
		})
	}
	return out
}

func autofillResultsToProto(results []internalauthoring.AutofillResult) []*authoringv1.AutofillResult {
	out := make([]*authoringv1.AutofillResult, 0, len(results))
	for _, r := range results {
		out = append(out, &authoringv1.AutofillResult{
			Source:     string(r.Source),
			SectionKey: string(r.SectionKey),
			Filled:     r.Filled,
			Degraded:   r.Degraded,
			Detail:     r.Detail,
		})
	}
	return out
}

func autofillSourcesFromProto(sources []string) []internalauthoring.AutofillSource {
	if len(sources) == 0 {
		return nil
	}
	out := make([]internalauthoring.AutofillSource, 0, len(sources))
	for _, s := range sources {
		out = append(out, internalauthoring.AutofillSource(s))
	}
	return out
}

// planToProto translates a persisted plans domain Plan into its shared proto wire
// form. Finalize returns the persisted plan; the shape mirrors the plans handler
// convert (the domain Plan is the SSOT and is mapped field-for-field here).
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
		out = append(out, &sharedv1.Phase{
			Id:              ph.ID,
			Order:           int32Of(ph.Order),
			Title:           ph.Title,
			Intent:          ph.Intent,
			RequiredReading: ph.RequiredReading,
			Reminders:       ph.Reminders,
			BaselineScope:   ph.BaselineScope,
			Acceptance:      ph.Acceptance,
			Status:          phaseStatusToProto(ph.Status),
			References:      referencesToProto(ph.References),
		})
	}
	return out
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

// int32Of narrows a phase order to int32 for the wire. Phase orders are small
// positive counts; a negative or overflowing value (impossible from the domain,
// which assigns 1..N) clamps to 0 rather than wrapping.
func int32Of(v int) int32 {
	if v < 0 || v > 1<<31-1 {
		return 0
	}
	return int32(v)
}

// --- enum converters (mirror handlers/plans/convert.go) ---

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
