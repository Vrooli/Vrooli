package authoring

import (
	internalauthoring "plan-manager/internal/authoring"
	planmodel "plan-manager/internal/planmodel"
	"plan-manager/internal/planproto"

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
		PhaseDrafts:       phaseDraftsToProto(s.PhaseDrafts),
		CurrentPhaseId:    s.CurrentPhaseID,
		RelevantContext:   planproto.RelevantContextItemsToProto(s.RelevantContext),
		ContextCandidates: contextCandidatesToProto(s.ContextCandidates),
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

func contextCandidateToProto(candidate internalauthoring.ContextCandidate) *authoringv1.ContextCandidate {
	return &authoringv1.ContextCandidate{
		Id:              candidate.ID,
		Item:            relevantContextItemToProto(candidate.Item),
		Concept:         candidate.Concept,
		Source:          candidate.Source,
		Degraded:        candidate.Degraded,
		Detail:          candidate.Detail,
		Status:          string(candidate.Status),
		RejectionReason: candidate.RejectionReason,
	}
}

func contextCandidatesToProto(candidates []internalauthoring.ContextCandidate) []*authoringv1.ContextCandidate {
	out := make([]*authoringv1.ContextCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, contextCandidateToProto(candidate))
	}
	return out
}

func guidedStepToProto(g internalauthoring.GuidedStep) *sharedv1.GuidedStep {
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

func nextActionsToProto(actions []internalauthoring.NextAction) []*sharedv1.NextAction {
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

func nextActionKindToProto(kind internalauthoring.NextActionKind) sharedv1.NextActionKind {
	switch kind {
	case internalauthoring.NextActionRecommended:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOMMENDED
	case internalauthoring.NextActionAlternative:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_ALTERNATIVE
	case internalauthoring.NextActionOptional:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_OPTIONAL
	case internalauthoring.NextActionRecovery:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOVERY
	default:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_UNSPECIFIED
	}
}

func phaseDraftsToProto(phases []internalauthoring.PhaseDraft) []*authoringv1.PhaseDraft {
	out := make([]*authoringv1.PhaseDraft, 0, len(phases))
	for _, phase := range phases {
		out = append(out, phaseDraftToProto(phase))
	}
	return out
}

func phaseDraftToProto(phase internalauthoring.PhaseDraft) *authoringv1.PhaseDraft {
	return &authoringv1.PhaseDraft{
		Id:               phase.ID,
		Order:            int32Of(phase.Order),
		Title:            phase.Title,
		Intent:           phase.Intent,
		References:       referencesToProto(phase.References),
		RequiredReading:  append([]string(nil), phase.RequiredReading...),
		Reminders:        append([]string(nil), phase.Reminders...),
		Acceptance:       phase.Acceptance,
		NoCodeRefsReason: phase.NoCodeRefsReason,
		RelevantContext:  planproto.RelevantContextItemsToProto(phase.RelevantContext),
	}
}

func relevantContextItemFromProto(item *sharedv1.RelevantContextItem) planmodel.RelevantContextItem {
	items := planproto.RelevantContextItemsFromProto([]*sharedv1.RelevantContextItem{item})
	if len(items) == 0 {
		return planmodel.RelevantContextItem{}
	}
	return items[0]
}

func relevantContextItemToProto(item planmodel.RelevantContextItem) *sharedv1.RelevantContextItem {
	items := planproto.RelevantContextItemsToProto([]planmodel.RelevantContextItem{item})
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func relevantContextItemsToProto(items []planmodel.RelevantContextItem) []*sharedv1.RelevantContextItem {
	return planproto.RelevantContextItemsToProto(items)
}

// planToProto translates a persisted plans domain Plan into its shared proto wire
// form. Finalize returns the persisted plan; the shape mirrors the plans handler
// convert (the domain Plan is the SSOT and is mapped field-for-field here).
func planToProto(p planmodel.Plan) *sharedv1.Plan {
	return planproto.PlanToProto(p)
}

func phasesToProto(phases []planmodel.Phase) []*sharedv1.Phase {
	return planproto.PhasesToProto(phases)
}

func referencesToProto(refs []planmodel.Reference) []*sharedv1.Reference {
	return planproto.ReferencesToProto(refs)
}

func anchorToProto(a planmodel.RegressionAnchor) *sharedv1.RegressionAnchor {
	return planproto.AnchorToProto(a)
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

func planStatusToProto(s planmodel.PlanStatus) sharedv1.PlanStatus {
	return planproto.PlanStatusToProto(s)
}

func phaseStatusToProto(s planmodel.PhaseStatus) sharedv1.PhaseStatus {
	return planproto.PhaseStatusToProto(s)
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
