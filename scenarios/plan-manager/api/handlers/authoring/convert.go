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
		Id:                  s.ID,
		Title:               s.Title,
		PlanSlug:            s.Slug,
		Sections:            sectionsToProto(s.Sections),
		CurrentSectionKey:   string(s.CurrentSectionKey),
		Finalized:           s.Finalized,
		PlanId:              s.PlanID,
		PhaseDrafts:         phaseDraftsToProto(s.PhaseDrafts),
		CurrentPhaseId:      s.CurrentPhaseID,
		RelevantContext:     planproto.RelevantContextItemsToProto(s.RelevantContext),
		ContextCandidates:   contextCandidatesToProto(s.ContextCandidates),
		ReferenceCandidates: referenceCandidatesToProto(s.ReferenceCandidates),
		DiscoveryBatches:    discoveryBatchesToProto(s.DiscoveryBatches),
		ReferenceBatches:    discoveryBatchesToProto(s.ReferenceBatches),
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
		Score:           candidate.Score,
		Origin:          candidate.Origin,
		SizeChars:       int32(candidate.SizeChars),
		Tags:            append([]string(nil), candidate.Tags...),
		Title:           candidate.Title,
		Snippet:         candidate.Snippet,
		Corroboration:   probeHitsToProto(candidate.Corroboration),
		Handle:          candidate.Handle,
		BatchId:         candidate.BatchID,
		Tier:            candidate.Tier,
		HighConfidence:  candidate.HighConfidence,
		SetupLine:       candidate.SetupLine,
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

func referenceCandidateToProto(candidate internalauthoring.ReferenceCandidate) *authoringv1.ReferenceCandidate {
	return &authoringv1.ReferenceCandidate{
		Id:              candidate.ID,
		Reference:       referenceToProto(candidate.Reference),
		Source:          candidate.Source,
		Confidence:      candidate.Confidence,
		Corroboration:   probeHitsToProto(candidate.Corroboration),
		Handle:          candidate.Handle,
		BatchId:         candidate.BatchID,
		Tier:            candidate.Tier,
		HighConfidence:  candidate.HighConfidence,
		Status:          string(candidate.Status),
		Degraded:        candidate.Degraded,
		Detail:          candidate.Detail,
		RejectionReason: candidate.RejectionReason,
	}
}

func probeHitsToProto(hits []internalauthoring.ProbeHit) []*authoringv1.ProbeHit {
	out := make([]*authoringv1.ProbeHit, 0, len(hits))
	for _, hit := range hits {
		out = append(out, &authoringv1.ProbeHit{
			Probe:   hit.Probe,
			Concept: hit.Concept,
			Score:   hit.Score,
		})
	}
	return out
}

func probeNotesToProto(notes []internalauthoring.ProbeNote) []*authoringv1.ProbeNote {
	out := make([]*authoringv1.ProbeNote, 0, len(notes))
	for _, note := range notes {
		out = append(out, &authoringv1.ProbeNote{
			Probe:    note.Probe,
			Concept:  note.Concept,
			Degraded: note.Degraded,
			Detail:   note.Detail,
		})
	}
	return out
}

func curationStatsToProto(stats internalauthoring.CurationStats) *authoringv1.CurationStats {
	return &authoringv1.CurationStats{
		SuppressedDispositioned: int32(stats.SuppressedDispositioned),
		OmittedBelowThreshold:   int32(stats.OmittedBelowThreshold),
		OmittedTopicFiller:      int32(stats.OmittedTopicFiller),
		OmittedByCap:            int32(stats.OmittedByCap),
	}
}

func discoveryBatchToProto(batch internalauthoring.DiscoveryBatch) *authoringv1.DiscoveryBatch {
	if batch.ID == "" {
		return nil
	}
	return &authoringv1.DiscoveryBatch{
		Id:            batch.ID,
		Concepts:      append([]string(nil), batch.Concepts...),
		Complexity:    batch.Complexity,
		ProbeNotes:    probeNotesToProto(batch.ProbeNotes),
		CurationStats: curationStatsToProto(batch.CurationStats),
		Status:        string(batch.Status),
		AppliedNote:   batch.AppliedNote,
		CreatedSeq:    int32(batch.CreatedSeq),
		Source:        batch.Source,
	}
}

func discoveryBatchesToProto(batches []internalauthoring.DiscoveryBatch) []*authoringv1.DiscoveryBatch {
	out := make([]*authoringv1.DiscoveryBatch, 0, len(batches))
	for _, batch := range batches {
		if pb := discoveryBatchToProto(batch); pb != nil {
			out = append(out, pb)
		}
	}
	return out
}

func contextDispositionTakesFromProto(takes []*authoringv1.ContextDispositionTake) []internalauthoring.ContextDispositionTake {
	out := make([]internalauthoring.ContextDispositionTake, 0, len(takes))
	for _, take := range takes {
		out = append(out, internalauthoring.ContextDispositionTake{
			CandidateID: take.GetCandidate(),
			PhaseID:     take.GetPhaseId(),
			Reason:      take.GetReason(),
		})
	}
	return out
}

func contextDispositionDropsFromProto(drops []*authoringv1.ContextDispositionDrop) []internalauthoring.ContextDispositionDrop {
	out := make([]internalauthoring.ContextDispositionDrop, 0, len(drops))
	for _, drop := range drops {
		out = append(out, internalauthoring.ContextDispositionDrop{
			CandidateID: drop.GetCandidate(),
			Reason:      drop.GetReason(),
		})
	}
	return out
}

func contextDispositionResultToProto(result internalauthoring.ContextDispositionResult) *authoringv1.ContextDispositionResult {
	return &authoringv1.ContextDispositionResult{
		Candidate:  contextCandidateToProto(result.Candidate),
		Item:       relevantContextItemToProto(result.Item),
		Action:     result.Action,
		Accepted:   result.Accepted,
		Message:    result.Message,
		Violations: violationsToProto(result.Violations),
	}
}

func contextDispositionResultsToProto(results []internalauthoring.ContextDispositionResult) []*authoringv1.ContextDispositionResult {
	out := make([]*authoringv1.ContextDispositionResult, 0, len(results))
	for _, result := range results {
		out = append(out, contextDispositionResultToProto(result))
	}
	return out
}

func referenceDispositionTakesFromProto(takes []*authoringv1.ReferenceDispositionTake) []internalauthoring.ReferenceDispositionTake {
	out := make([]internalauthoring.ReferenceDispositionTake, 0, len(takes))
	for _, take := range takes {
		if take == nil {
			continue
		}
		out = append(out, internalauthoring.ReferenceDispositionTake{CandidateID: take.GetCandidate()})
	}
	return out
}

func referenceDispositionDropsFromProto(drops []*authoringv1.ReferenceDispositionDrop) []internalauthoring.ReferenceDispositionDrop {
	out := make([]internalauthoring.ReferenceDispositionDrop, 0, len(drops))
	for _, drop := range drops {
		if drop == nil {
			continue
		}
		out = append(out, internalauthoring.ReferenceDispositionDrop{
			CandidateID: drop.GetCandidate(),
			Reason:      drop.GetReason(),
		})
	}
	return out
}

func referenceDispositionResultsToProto(results []internalauthoring.ReferenceDispositionResult) []*authoringv1.ReferenceDispositionResult {
	out := make([]*authoringv1.ReferenceDispositionResult, 0, len(results))
	for _, result := range results {
		out = append(out, &authoringv1.ReferenceDispositionResult{
			Candidate:  referenceCandidateToProto(result.Candidate),
			Reference:  referenceToProto(result.Reference),
			Action:     result.Action,
			Accepted:   result.Accepted,
			Message:    result.Message,
			Violations: violationsToProto(result.Violations),
		})
	}
	return out
}

func referenceCandidatesToProto(candidates []internalauthoring.ReferenceCandidate) []*authoringv1.ReferenceCandidate {
	out := make([]*authoringv1.ReferenceCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, referenceCandidateToProto(candidate))
	}
	return out
}

// referenceEditFromProto maps an optional inline locator edit on
// AcceptReferenceCandidate. A nil/empty proto reference means "no edit".
func referenceEditFromProto(ref *sharedv1.Reference) *planmodel.Reference {
	if ref == nil {
		return nil
	}
	out := referenceFromProto(ref)
	if out.Kind == "" && out.Target == "" && !out.Future {
		return nil
	}
	return &out
}

func referenceFromProto(ref *sharedv1.Reference) planmodel.Reference {
	refs := planproto.ReferencesFromProto([]*sharedv1.Reference{ref})
	if len(refs) == 0 {
		return planmodel.Reference{}
	}
	return refs[0]
}

func referenceToProto(ref planmodel.Reference) *sharedv1.Reference {
	refs := planproto.ReferencesToProto([]planmodel.Reference{ref})
	if len(refs) == 0 {
		return nil
	}
	return refs[0]
}

func guidedStepToProto(g internalauthoring.GuidedStep) *sharedv1.GuidedStep {
	return planproto.GuidedStepToProto(g)
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
		AffectedAreas:    append([]string(nil), phase.AffectedAreas...),
		Steps:            append([]string(nil), phase.Steps...),
		ExpectedOutputs:  append([]string(nil), phase.ExpectedOutputs...),
		Validation:       phase.Validation,
		RisksHazards:     append([]string(nil), phase.RisksHazards...),
		HandoffNotes:     phase.HandoffNotes,
	}
}

// progressToProto maps the compact navigation snapshot to the wire. Every normal
// mutation returns this in place of the full session graph.
func progressToProto(p internalauthoring.AuthoringProgress) *authoringv1.AuthoringProgress {
	return &authoringv1.AuthoringProgress{
		SessionId:               p.SessionID,
		CurrentSectionKey:       p.CurrentSectionKey,
		CurrentPhaseId:          p.CurrentPhaseID,
		MandatorySectionsTotal:  int32Of(p.MandatorySectionsTotal),
		MandatorySectionsFilled: int32Of(p.MandatorySectionsFilled),
		PhasesTotal:             int32Of(p.PhasesTotal),
		PhasesComplete:          int32Of(p.PhasesComplete),
		RemainingRequiredInputs: append([]string(nil), p.RemainingRequiredInputs...),
		ReadyToFinalize:         p.ReadyToFinalize,
	}
}

// progressOf is the handler-edge shortcut: compute and map progress from a saved
// session in one step.
func progressOf(sess internalauthoring.Session) *authoringv1.AuthoringProgress {
	return progressToProto(internalauthoring.ComputeProgress(sess))
}

func mutationSummary(kind, objectID, field, summary string) *authoringv1.AuthoringMutationSummary {
	return &authoringv1.AuthoringMutationSummary{
		ObjectKind: kind,
		ObjectId:   objectID,
		Field:      field,
		Summary:    summary,
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

func referencesToProto(refs []planmodel.Reference) []*sharedv1.Reference {
	return planproto.ReferencesToProto(refs)
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
