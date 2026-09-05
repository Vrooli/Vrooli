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
		Id:                      p.ID,
		Slug:                    p.Slug,
		Title:                   p.Title,
		Status:                  PlanStatusToProto(p.Status),
		ContentHash:             p.ContentHash,
		CreatedAt:               p.CreatedAt,
		UpdatedAt:               p.UpdatedAt,
		WorkspaceId:             p.WorkspaceID,
		WorkspaceRoot:           p.WorkspaceRoot,
		Purpose:                 p.Purpose,
		Scope:                   p.Scope,
		Constraints:             p.Constraints,
		NonGoals:                p.NonGoals,
		References:              ReferencesToProto(p.References),
		RegressionAnchor:        AnchorToProto(p.RegressionAnchor),
		BaselineSet:             BaselineSetToProto(p.BaselineSet),
		DefinitionOfDone:        p.DefinitionOfDone,
		Phases:                  PhasesToProto(p.Phases),
		Supersedes:              p.Supersedes,
		SupersededBy:            p.SupersededBy,
		RelevantContext:         RelevantContextItemsToProto(p.RelevantContext),
		ProblemStatement:        p.ProblemStatement,
		TargetOutcome:           p.TargetOutcome,
		Assumptions:             p.Assumptions,
		TechnicalApproach:       p.TechnicalApproach,
		ValidationStrategy:      p.ValidationStrategy,
		FinalValidationCommands: p.FinalValidationCommands,
		RisksHazards:            p.RisksHazards,
		ProhibitedApproaches:    p.ProhibitedApproaches,
		WorkPosture:             WorkPostureToProto(p.WorkPosture),
		WorkPostureSource:       WorkPostureSourceToProto(p.WorkPostureSource),
		WorkPostureDetail:       p.WorkPostureDetail,
		ImportProvenance:        ImportProvenanceToProto(p.ImportProvenance),
		PreservedLegacySections: LegacySectionsToProto(p.PreservedLegacySections),
		ChangeBoundary:          ChangeBoundaryToProto(p.ChangeBoundary),
		Mirror:                  MirrorToProto(p.Mirror),
		Decisions:               PlanDecisionsToProto(p.Decisions),
		AssumptionRisks:         PlanAssumptionsToProto(p.AssumptionRisks),
		Definitions:             PlanDefinitionsToProto(p.Definitions),
	}
}

func PlanDecisionsToProto(items []planmodel.PlanDecision) []*sharedv1.PlanDecision {
	if len(items) == 0 {
		return nil
	}
	out := make([]*sharedv1.PlanDecision, 0, len(items))
	for _, d := range items {
		out = append(out, &sharedv1.PlanDecision{Title: d.Title, Statement: d.Statement})
	}
	return out
}

func PlanDecisionsFromProto(items []*sharedv1.PlanDecision) []planmodel.PlanDecision {
	if len(items) == 0 {
		return nil
	}
	out := make([]planmodel.PlanDecision, 0, len(items))
	for _, d := range items {
		out = append(out, planmodel.PlanDecision{Title: d.GetTitle(), Statement: d.GetStatement()})
	}
	return out
}

func PlanAssumptionsToProto(items []planmodel.PlanAssumption) []*sharedv1.PlanAssumption {
	if len(items) == 0 {
		return nil
	}
	out := make([]*sharedv1.PlanAssumption, 0, len(items))
	for _, a := range items {
		out = append(out, &sharedv1.PlanAssumption{Statement: a.Statement, Mitigation: a.Mitigation})
	}
	return out
}

func PlanAssumptionsFromProto(items []*sharedv1.PlanAssumption) []planmodel.PlanAssumption {
	if len(items) == 0 {
		return nil
	}
	out := make([]planmodel.PlanAssumption, 0, len(items))
	for _, a := range items {
		out = append(out, planmodel.PlanAssumption{Statement: a.GetStatement(), Mitigation: a.GetMitigation()})
	}
	return out
}

func PlanDefinitionsToProto(items []planmodel.PlanDefinition) []*sharedv1.PlanDefinition {
	if len(items) == 0 {
		return nil
	}
	out := make([]*sharedv1.PlanDefinition, 0, len(items))
	for _, item := range items {
		out = append(out, &sharedv1.PlanDefinition{Term: item.Term, Meaning: item.Meaning})
	}
	return out
}

func PlanDefinitionsFromProto(items []*sharedv1.PlanDefinition) []planmodel.PlanDefinition {
	if len(items) == 0 {
		return nil
	}
	out := make([]planmodel.PlanDefinition, 0, len(items))
	for _, item := range items {
		if item != nil {
			out = append(out, planmodel.PlanDefinition{Term: item.GetTerm(), Meaning: item.GetMeaning()})
		}
	}
	return out
}

func PlanFromProto(p *sharedv1.Plan) planmodel.Plan {
	if p == nil {
		return planmodel.Plan{}
	}
	return planmodel.Plan{
		ID:                      p.GetId(),
		Slug:                    p.GetSlug(),
		Title:                   p.GetTitle(),
		Status:                  PlanStatusFromProto(p.GetStatus()),
		ContentHash:             p.GetContentHash(),
		CreatedAt:               p.GetCreatedAt(),
		UpdatedAt:               p.GetUpdatedAt(),
		WorkspaceID:             p.GetWorkspaceId(),
		WorkspaceRoot:           p.GetWorkspaceRoot(),
		Purpose:                 p.GetPurpose(),
		Scope:                   p.GetScope(),
		Constraints:             p.GetConstraints(),
		NonGoals:                p.GetNonGoals(),
		References:              ReferencesFromProto(p.GetReferences()),
		RegressionAnchor:        AnchorFromProto(p.GetRegressionAnchor()),
		BaselineSet:             BaselineSetFromProto(p.GetBaselineSet()),
		DefinitionOfDone:        p.GetDefinitionOfDone(),
		Phases:                  PhasesFromProto(p.GetPhases()),
		Supersedes:              p.GetSupersedes(),
		SupersededBy:            p.GetSupersededBy(),
		RelevantContext:         RelevantContextItemsFromProto(p.GetRelevantContext()),
		ProblemStatement:        p.GetProblemStatement(),
		TargetOutcome:           p.GetTargetOutcome(),
		Assumptions:             p.GetAssumptions(),
		TechnicalApproach:       p.GetTechnicalApproach(),
		ValidationStrategy:      p.GetValidationStrategy(),
		FinalValidationCommands: p.GetFinalValidationCommands(),
		RisksHazards:            p.GetRisksHazards(),
		ProhibitedApproaches:    p.GetProhibitedApproaches(),
		WorkPosture:             WorkPostureFromProto(p.GetWorkPosture()),
		WorkPostureSource:       WorkPostureSourceFromProto(p.GetWorkPostureSource()),
		WorkPostureDetail:       p.GetWorkPostureDetail(),
		ImportProvenance:        ImportProvenanceFromProto(p.GetImportProvenance()),
		PreservedLegacySections: LegacySectionsFromProto(p.GetPreservedLegacySections()),
		ChangeBoundary:          ChangeBoundaryFromProto(p.GetChangeBoundary()),
		Mirror:                  MirrorFromProto(p.GetMirror()),
		Decisions:               PlanDecisionsFromProto(p.GetDecisions()),
		AssumptionRisks:         PlanAssumptionsFromProto(p.GetAssumptionRisks()),
		Definitions:             PlanDefinitionsFromProto(p.GetDefinitions()),
	}
}

func MirrorToProto(m planmodel.RenderedPlanMirror) *sharedv1.RenderedPlanMirror {
	if m == (planmodel.RenderedPlanMirror{}) {
		return nil
	}
	return &sharedv1.RenderedPlanMirror{
		Path:          m.Path,
		RelativePath:  m.RelativePath,
		ContentHash:   m.ContentHash,
		RenderVersion: m.RenderVersion,
		RenderedAt:    m.RenderedAt,
		Status:        MirrorStatusToProto(m.Status),
		LastError:     m.LastError,
	}
}

func MirrorFromProto(m *sharedv1.RenderedPlanMirror) planmodel.RenderedPlanMirror {
	if m == nil {
		return planmodel.RenderedPlanMirror{}
	}
	return planmodel.RenderedPlanMirror{
		Path:          m.GetPath(),
		RelativePath:  m.GetRelativePath(),
		ContentHash:   m.GetContentHash(),
		RenderVersion: m.GetRenderVersion(),
		RenderedAt:    m.GetRenderedAt(),
		Status:        MirrorStatusFromProto(m.GetStatus()),
		LastError:     m.GetLastError(),
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
		AffectedAreas:   ph.AffectedAreas,
		Steps:           ph.Steps,
		ExpectedOutputs: ph.ExpectedOutputs,
		Validation:      ph.Validation,
		HandoffNotes:    ph.HandoffNotes,
		RisksHazards:    ph.RisksHazards,
		ChangeBoundary:  ChangeBoundaryToProto(ph.ChangeBoundary),
		ValidationScope: ValidationScopeToProto(ph.ValidationScope),
	}
}

// PhaseFromProto converts a wire Phase into the neutral kernel. Computed/joined
// fields (last_validation) are intentionally not carried back here; the plans
// write path rejects them at the handler edge before calling this.
func PhaseFromProto(ph *sharedv1.Phase) planmodel.Phase {
	if ph == nil {
		return planmodel.Phase{}
	}
	return planmodel.Phase{
		ID:              ph.GetId(),
		Order:           int(ph.GetOrder()),
		Title:           ph.GetTitle(),
		Intent:          ph.GetIntent(),
		RequiredReading: ph.GetRequiredReading(),
		Reminders:       ph.GetReminders(),
		BaselineScope:   ph.GetBaselineScope(),
		Acceptance:      ph.GetAcceptance(),
		Status:          PhaseStatusFromProto(ph.GetStatus()),
		References:      ReferencesFromProto(ph.GetReferences()),
		RelevantContext: RelevantContextItemsFromProto(ph.GetRelevantContext()),
		AffectedAreas:   ph.GetAffectedAreas(),
		Steps:           ph.GetSteps(),
		ExpectedOutputs: ph.GetExpectedOutputs(),
		Validation:      ph.GetValidation(),
		HandoffNotes:    ph.GetHandoffNotes(),
		RisksHazards:    ph.GetRisksHazards(),
		ChangeBoundary:  ChangeBoundaryFromProto(ph.GetChangeBoundary()),
		ValidationScope: ValidationScopeFromProto(ph.GetValidationScope()),
	}
}

func PhasesFromProto(phases []*sharedv1.Phase) []planmodel.Phase {
	if len(phases) == 0 {
		return nil
	}
	out := make([]planmodel.Phase, 0, len(phases))
	for _, ph := range phases {
		if ph == nil {
			continue
		}
		out = append(out, PhaseFromProto(ph))
	}
	return out
}

func ReferencesFromProto(refs []*sharedv1.Reference) []planmodel.Reference {
	if len(refs) == 0 {
		return nil
	}
	out := make([]planmodel.Reference, 0, len(refs))
	for _, r := range refs {
		if r == nil {
			continue
		}
		out = append(out, planmodel.Reference{
			ID:           r.GetId(),
			Kind:         RefKindFromProto(r.GetKind()),
			Target:       r.GetTarget(),
			Future:       r.GetFuture(),
			Resolution:   RefResolutionFromProto(r.GetResolution()),
			Staleness:    StalenessFromProto(r.GetStaleness()),
			ChangeFactor: r.GetChangeFactor(),
			Note:         r.GetNote(),
		})
	}
	return out
}

func AnchorFromProto(a *sharedv1.RegressionAnchor) planmodel.RegressionAnchor {
	if a == nil {
		return planmodel.RegressionAnchor{}
	}
	return planmodel.RegressionAnchor{
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

// ChangeBoundaryToProto converts the neutral boundary to its wire shape. A zero
// boundary still produces a (zero) message so consumers see consistent shape;
// callers that omit boundaries entirely should pass a zero value.
func ChangeBoundaryToProto(b planmodel.ChangeBoundary) *sharedv1.ChangeBoundary {
	if b.IsZero() {
		return nil
	}
	return &sharedv1.ChangeBoundary{
		AcceptanceAllow:    b.AcceptanceAllow,
		AcceptanceDeny:     b.AcceptanceDeny,
		OperatorOnlyReason: b.OperatorOnlyReason,
	}
}

func ChangeBoundaryFromProto(b *sharedv1.ChangeBoundary) planmodel.ChangeBoundary {
	if b == nil {
		return planmodel.ChangeBoundary{}
	}
	return planmodel.ChangeBoundary{
		AcceptanceAllow:    b.GetAcceptanceAllow(),
		AcceptanceDeny:     b.GetAcceptanceDeny(),
		OperatorOnlyReason: b.GetOperatorOnlyReason(),
	}
}

func ValidationScopeToProto(scope planmodel.ValidationScope) *sharedv1.ValidationScope {
	if scope.Mode == planmodel.ValidationScopeUnspecified {
		return nil
	}
	mode := sharedv1.ValidationScopeMode_VALIDATION_SCOPE_MODE_UNSPECIFIED
	if scope.Mode == planmodel.ValidationScopeNarrow {
		mode = sharedv1.ValidationScopeMode_VALIDATION_SCOPE_MODE_NARROW
	}
	if scope.Mode == planmodel.ValidationScopeFullPlan {
		mode = sharedv1.ValidationScopeMode_VALIDATION_SCOPE_MODE_FULL_PLAN
	}
	return &sharedv1.ValidationScope{Mode: mode, Boundary: ChangeBoundaryToProto(scope.Boundary), Rationale: scope.Rationale}
}

func ValidationScopeFromProto(scope *sharedv1.ValidationScope) planmodel.ValidationScope {
	if scope == nil {
		return planmodel.ValidationScope{}
	}
	mode := planmodel.ValidationScopeUnspecified
	if scope.GetMode() == sharedv1.ValidationScopeMode_VALIDATION_SCOPE_MODE_NARROW {
		mode = planmodel.ValidationScopeNarrow
	}
	if scope.GetMode() == sharedv1.ValidationScopeMode_VALIDATION_SCOPE_MODE_FULL_PLAN {
		mode = planmodel.ValidationScopeFullPlan
	}
	return planmodel.ValidationScope{Mode: mode, Boundary: ChangeBoundaryFromProto(scope.GetBoundary()), Rationale: scope.GetRationale()}
}

func WorkPostureToProto(p planmodel.WorkPosture) sharedv1.WorkPosture {
	switch p {
	case planmodel.WorkPostureGreenfield:
		return sharedv1.WorkPosture_WORK_POSTURE_GREENFIELD
	case planmodel.WorkPostureBrownfield:
		return sharedv1.WorkPosture_WORK_POSTURE_BROWNFIELD
	default:
		return sharedv1.WorkPosture_WORK_POSTURE_UNSPECIFIED
	}
}

func WorkPostureFromProto(p sharedv1.WorkPosture) planmodel.WorkPosture {
	switch p {
	case sharedv1.WorkPosture_WORK_POSTURE_GREENFIELD:
		return planmodel.WorkPostureGreenfield
	case sharedv1.WorkPosture_WORK_POSTURE_BROWNFIELD:
		return planmodel.WorkPostureBrownfield
	default:
		return planmodel.WorkPostureUnspecified
	}
}

func WorkPostureSourceToProto(s planmodel.WorkPostureSource) sharedv1.WorkPostureSource {
	switch s {
	case planmodel.WorkPostureSourceDefault:
		return sharedv1.WorkPostureSource_WORK_POSTURE_SOURCE_DEFAULT
	case planmodel.WorkPostureSourceServiceMaturity:
		return sharedv1.WorkPostureSource_WORK_POSTURE_SOURCE_SERVICE_MATURITY
	case planmodel.WorkPostureSourceExplicitOverride:
		return sharedv1.WorkPostureSource_WORK_POSTURE_SOURCE_EXPLICIT_OVERRIDE
	case planmodel.WorkPostureSourceImportLegacy:
		return sharedv1.WorkPostureSource_WORK_POSTURE_SOURCE_IMPORT_LEGACY
	default:
		return sharedv1.WorkPostureSource_WORK_POSTURE_SOURCE_UNSPECIFIED
	}
}

func WorkPostureSourceFromProto(s sharedv1.WorkPostureSource) planmodel.WorkPostureSource {
	switch s {
	case sharedv1.WorkPostureSource_WORK_POSTURE_SOURCE_DEFAULT:
		return planmodel.WorkPostureSourceDefault
	case sharedv1.WorkPostureSource_WORK_POSTURE_SOURCE_SERVICE_MATURITY:
		return planmodel.WorkPostureSourceServiceMaturity
	case sharedv1.WorkPostureSource_WORK_POSTURE_SOURCE_EXPLICIT_OVERRIDE:
		return planmodel.WorkPostureSourceExplicitOverride
	case sharedv1.WorkPostureSource_WORK_POSTURE_SOURCE_IMPORT_LEGACY:
		return planmodel.WorkPostureSourceImportLegacy
	default:
		return planmodel.WorkPostureSourceUnspecified
	}
}

func MirrorStatusToProto(s planmodel.RenderedMirrorStatus) sharedv1.RenderedMirrorStatus {
	switch s {
	case planmodel.RenderedMirrorStatusFresh:
		return sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_FRESH
	case planmodel.RenderedMirrorStatusMissing:
		return sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_MISSING
	case planmodel.RenderedMirrorStatusStale:
		return sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_STALE
	case planmodel.RenderedMirrorStatusWriteFailed:
		return sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_WRITE_FAILED
	case planmodel.RenderedMirrorStatusUnknown:
		return sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_UNKNOWN
	default:
		return sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_UNSPECIFIED
	}
}

func MirrorStatusFromProto(s sharedv1.RenderedMirrorStatus) planmodel.RenderedMirrorStatus {
	switch s {
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_FRESH:
		return planmodel.RenderedMirrorStatusFresh
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_MISSING:
		return planmodel.RenderedMirrorStatusMissing
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_STALE:
		return planmodel.RenderedMirrorStatusStale
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_WRITE_FAILED:
		return planmodel.RenderedMirrorStatusWriteFailed
	case sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_UNKNOWN:
		return planmodel.RenderedMirrorStatusUnknown
	default:
		return planmodel.RenderedMirrorStatusUnspecified
	}
}

func LegacySectionsToProto(sections []planmodel.LegacySection) []*sharedv1.LegacySection {
	if len(sections) == 0 {
		return nil
	}
	out := make([]*sharedv1.LegacySection, 0, len(sections))
	for _, s := range sections {
		out = append(out, &sharedv1.LegacySection{
			Heading:            s.Heading,
			Content:            s.Content,
			MappedTo:           s.MappedTo,
			PreservationReason: s.PreservationReason,
		})
	}
	return out
}

func LegacySectionsFromProto(sections []*sharedv1.LegacySection) []planmodel.LegacySection {
	if len(sections) == 0 {
		return nil
	}
	out := make([]planmodel.LegacySection, 0, len(sections))
	for _, s := range sections {
		if s == nil {
			continue
		}
		out = append(out, planmodel.LegacySection{
			Heading:            s.GetHeading(),
			Content:            s.GetContent(),
			MappedTo:           s.GetMappedTo(),
			PreservationReason: s.GetPreservationReason(),
		})
	}
	return out
}

func ImportProvenanceToProto(p *planmodel.ImportProvenance) *sharedv1.ImportProvenance {
	if p == nil {
		return nil
	}
	return &sharedv1.ImportProvenance{
		SourcePath:     p.SourcePath,
		ImportedAt:     p.ImportedAt,
		OriginalFormat: p.OriginalFormat,
		Note:           p.Note,
		WorkspaceId:    p.WorkspaceID,
		WorkspaceRoot:  p.WorkspaceRoot,
	}
}

func ImportProvenanceFromProto(p *sharedv1.ImportProvenance) *planmodel.ImportProvenance {
	if p == nil {
		return nil
	}
	return &planmodel.ImportProvenance{
		SourcePath:     p.GetSourcePath(),
		ImportedAt:     p.GetImportedAt(),
		OriginalFormat: p.GetOriginalFormat(),
		Note:           p.GetNote(),
		WorkspaceID:    p.GetWorkspaceId(),
		WorkspaceRoot:  p.GetWorkspaceRoot(),
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

func BaselineSetToProto(intent planmodel.BaselineSetIntent) *sharedv1.BaselineSetIntent {
	if intent.Name == "" && len(intent.ScenarioTargets) == 0 && len(intent.RepoPaths) == 0 && intent.CapturePolicy == "" && intent.Compatibility == "" {
		return nil
	}
	return &sharedv1.BaselineSetIntent{
		Name:            intent.Name,
		ScenarioTargets: intent.ScenarioTargets,
		RepoPaths:       intent.RepoPaths,
		CapturePolicy:   intent.CapturePolicy,
		Compatibility:   intent.Compatibility,
	}
}

func BaselineSetFromProto(intent *sharedv1.BaselineSetIntent) planmodel.BaselineSetIntent {
	if intent == nil {
		return planmodel.BaselineSetIntent{}
	}
	return planmodel.BaselineSetIntent{
		Name:            intent.GetName(),
		ScenarioTargets: append([]string(nil), intent.GetScenarioTargets()...),
		RepoPaths:       append([]string(nil), intent.GetRepoPaths()...),
		CapturePolicy:   intent.GetCapturePolicy(),
		Compatibility:   intent.GetCompatibility(),
	}
}

func EdgeToProto(e planmodel.PlanEdge) *sharedv1.PlanEdge {
	return &sharedv1.PlanEdge{
		FromPlanId: e.FromPlanID,
		ToPlanId:   e.ToPlanID,
		Kind:       e.Kind,
	}
}

var (
	planStatusPairs = []enumPair[planmodel.PlanStatus, sharedv1.PlanStatus]{
		{planmodel.PlanStatusDraft, sharedv1.PlanStatus_PLAN_STATUS_DRAFT},
		{planmodel.PlanStatusActive, sharedv1.PlanStatus_PLAN_STATUS_ACTIVE},
		{planmodel.PlanStatusComplete, sharedv1.PlanStatus_PLAN_STATUS_COMPLETE},
		{planmodel.PlanStatusArchived, sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED},
	}
	phaseStatusPairs = []enumPair[planmodel.PhaseStatus, sharedv1.PhaseStatus]{
		{planmodel.PhaseStatusTodo, sharedv1.PhaseStatus_PHASE_STATUS_TODO},
		{planmodel.PhaseStatusActive, sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE},
		{planmodel.PhaseStatusDone, sharedv1.PhaseStatus_PHASE_STATUS_DONE},
		{planmodel.PhaseStatusBlocked, sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED},
	}
	refKindPairs = []enumPair[planmodel.ReferenceKind, sharedv1.ReferenceKind]{
		{planmodel.ReferenceCode, sharedv1.ReferenceKind_REFERENCE_KIND_CODE},
		{planmodel.ReferenceReq, sharedv1.ReferenceKind_REFERENCE_KIND_REQ},
		{planmodel.ReferenceDoc, sharedv1.ReferenceKind_REFERENCE_KIND_DOC},
	}
	refResolutionPairs = []enumPair[planmodel.ReferenceResolution, sharedv1.ReferenceResolution]{
		{planmodel.ResolutionResolved, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED},
		{planmodel.ResolutionUnresolved, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED},
		{planmodel.ResolutionFuture, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE},
		{planmodel.ResolutionMissing, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING},
	}
)

func PlanStatusToProto(s planmodel.PlanStatus) sharedv1.PlanStatus {
	return enumToProto(s, planStatusPairs, sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED)
}

func PlanStatusFromProto(s sharedv1.PlanStatus) planmodel.PlanStatus {
	return enumFromProto(s, planStatusPairs, "")
}

func PhaseStatusToProto(s planmodel.PhaseStatus) sharedv1.PhaseStatus {
	return enumToProto(s, phaseStatusPairs, sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED)
}

func PhaseStatusFromProto(s sharedv1.PhaseStatus) planmodel.PhaseStatus {
	return enumFromProto(s, phaseStatusPairs, "")
}

func RefKindToProto(k planmodel.ReferenceKind) sharedv1.ReferenceKind {
	return enumToProto(k, refKindPairs, sharedv1.ReferenceKind_REFERENCE_KIND_UNSPECIFIED)
}

func RefKindFromProto(k sharedv1.ReferenceKind) planmodel.ReferenceKind {
	return enumFromProto(k, refKindPairs, planmodel.ReferenceCode)
}

func RefResolutionToProto(r planmodel.ReferenceResolution) sharedv1.ReferenceResolution {
	return enumToProto(r, refResolutionPairs, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNSPECIFIED)
}

func RefResolutionFromProto(r sharedv1.ReferenceResolution) planmodel.ReferenceResolution {
	return enumFromProto(r, refResolutionPairs, planmodel.ResolutionUnspecified)
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
