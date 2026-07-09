package operatingmode

import (
	"encoding/json"

	"swarm-manager/internal/initiativelock"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/types/known/structpb"
)

// This file maps the operating-mode service's internal projection structs to
// their generated Connect wire messages. Each mapper mirrors one Go struct
// one-for-one so the typed contract stays a faithful projection of the same
// data the (transitional) REST surface serves. Message-typed fields map to nil
// when the source pointer is nil so `omitempty`-style absence is preserved on
// the wire.

func structFromMap(m map[string]any) *structpb.Struct {
	if len(m) == 0 {
		return nil
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return s
}

func structFromRaw(raw json.RawMessage) *structpb.Struct {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return structFromMap(m)
}

func capabilitiesToProto(c ModeCapabilities) *apipb.OperatingModeCapabilities {
	return &apipb.OperatingModeCapabilities{
		SupportsPhases:               c.SupportsPhases,
		CanStartPhases:               c.CanStartPhases,
		CanCompleteItems:             c.CanCompleteItems,
		CanApplyBacklogSyncProposals: c.CanApplyBacklogSyncProposals,
		RequiresAcceptanceCriteria:   c.RequiresAcceptanceCriteria,
		SupportsArtifacts:            c.SupportsArtifacts,
		SupportsHandoffs:             c.SupportsHandoffs,
		UsesItemExecutionFlow:        c.UsesItemExecutionFlow,
	}
}

func artifactDefToProto(a ArtifactDefinition) *apipb.OperatingModeArtifactDefinition {
	return &apipb.OperatingModeArtifactDefinition{
		Path:        a.Path,
		ContentType: a.ContentType,
		Required:    a.Required,
	}
}

func artifactDefsToProto(in []ArtifactDefinition) []*apipb.OperatingModeArtifactDefinition {
	if len(in) == 0 {
		return nil
	}
	out := make([]*apipb.OperatingModeArtifactDefinition, len(in))
	for i, a := range in {
		out[i] = artifactDefToProto(a)
	}
	return out
}

func resultBindingsToProto(in []ResultBinding) []*apipb.OperatingModeResultBindingSummary {
	if len(in) == 0 {
		return nil
	}
	out := make([]*apipb.OperatingModeResultBindingSummary, len(in))
	for i, b := range in {
		out[i] = &apipb.OperatingModeResultBindingSummary{
			Kind:     string(b.Kind),
			Artifact: artifactDefToProto(b.Artifact),
		}
	}
	return out
}

func outputContractToProto(c PhaseOutputContractSummary) *apipb.OperatingModePhaseOutputContractSummary {
	return &apipb.OperatingModePhaseOutputContractSummary{
		RequiresStructuredResult: c.RequiresStructuredResult,
		RequiresProgress:         c.RequiresProgress,
		RequiresVerdict:          c.RequiresVerdict,
		RequiresHandoff:          c.RequiresHandoff,
		RequiresBacklogSync:      c.RequiresBacklogSync,
		RequiredArtifactCount:    int32(c.RequiredArtifactCount),
	}
}

func catalogPhaseToProto(p ModeCatalogPhase) *apipb.OperatingModeCatalogPhase {
	return &apipb.OperatingModeCatalogPhase{
		Phase:                 p.Phase,
		PhaseKind:             p.PhaseKind,
		Label:                 p.Label,
		Title:                 p.Title,
		Purpose:               p.Purpose,
		Trigger:               p.Trigger,
		ProfileKey:            p.ProfileKey,
		WritesRepo:            p.WritesRepo,
		RequiresCriteria:      p.RequiresCriteria,
		IsStart:               p.IsStart,
		IsTerminal:            p.IsTerminal,
		OutputArtifacts:       artifactDefsToProto(p.OutputArtifacts),
		OutputContract:        outputContractToProto(p.OutputContract),
		CatalogId:             p.CatalogID,
		SkillId:               p.SkillID,
		ActivityPurpose:       p.ActivityPurpose,
		LockPurpose:           p.LockPurpose,
		ResultBindings:        resultBindingsToProto(p.ResultBindings),
		SamplesReplanRate:     p.SamplesReplanRate,
		SamplesAcceptanceRate: p.SamplesAcceptanceRate,
		AutoStartAfter:        p.AutoStartAfter,
		Reads:                 phaseReadsToProto(p.Reads),
		ExecutedBy:            p.ExecutedBy,
		Classification:        phaseClassificationToProto(p.Classification),
	}
}

func phaseClassificationToProto(c *PhaseClassificationSummary) *apipb.OperatingModeTransitionClassification {
	if c == nil {
		return nil
	}
	return &apipb.OperatingModeTransitionClassification{
		Field:       c.Field,
		Enum:        c.Enum,
		From:        c.From,
		Description: c.Description,
	}
}

func phaseReadsToProto(r PhaseReadsSummary) *apipb.OperatingModePhaseReads {
	if len(r.Base) == 0 && len(r.Target) == 0 {
		return nil
	}
	return &apipb.OperatingModePhaseReads{
		Base:   r.Base,
		Target: r.Target,
	}
}

func catalogPhaseGraphToProto(g *ModeCatalogPhaseGraph) *apipb.OperatingModeCatalogPhaseGraph {
	if g == nil {
		return nil
	}
	transitions := make([]*apipb.OperatingModeCatalogTransition, len(g.Transitions))
	for i, t := range g.Transitions {
		transitions[i] = &apipb.OperatingModeCatalogTransition{
			From:          t.From,
			To:            t.To,
			ConditionKind: t.ConditionKind,
			Label:         t.Label,
			Field:         t.Field,
			Value:         t.Value,
			Classified:    t.Classified,
		}
	}
	return &apipb.OperatingModeCatalogPhaseGraph{
		StartPhase:       g.StartPhase,
		Terminal:         g.Terminal,
		Transitions:      transitions,
		AcceptedVerdicts: g.AcceptedVerdicts,
	}
}

func catalogEntryToProto(e ModeCatalogEntry) *apipb.OperatingModeCatalogEntry {
	phases := make([]*apipb.OperatingModeCatalogPhase, len(e.Phases))
	for i, p := range e.Phases {
		phases[i] = catalogPhaseToProto(p)
	}
	return &apipb.OperatingModeCatalogEntry{
		Mode:                   e.Mode,
		Label:                  e.Label,
		Description:            e.Description,
		BestFor:                e.BestFor,
		NotFor:                 e.NotFor,
		Tradeoffs:              e.Tradeoffs,
		WhenInDoubtPickInstead: e.WhenInDoubtPickInstead,
		UsageCount:             int32(e.UsageCount),
		TargetKind:             e.TargetKind,
		RunStrategy:            e.RunStrategy,
		WorkspaceTabId:         e.WorkspaceTabID,
		Capabilities:           capabilitiesToProto(e.Capabilities),
		Default:                e.Default,
		Switchable:             e.Switchable,
		SupportsPhases:         e.SupportsPhases,
		Phases:                 phases,
		PhaseGraph:             catalogPhaseGraphToProto(e.PhaseGraph),
	}
}

func catalogToProto(c ModeCatalog) *apipb.OperatingModeCatalogResponse {
	modes := make([]*apipb.OperatingModeCatalogEntry, len(c.Modes))
	for i, e := range c.Modes {
		modes[i] = catalogEntryToProto(e)
	}
	return &apipb.OperatingModeCatalogResponse{Modes: modes}
}

func modeDetailToProto(d ModeDetail) *apipb.OperatingModeDetailResponse {
	linked := make([]*apipb.OperatingModeInitiativeRef, len(d.LinkedInitiatives))
	for i, r := range d.LinkedInitiatives {
		linked[i] = &apipb.OperatingModeInitiativeRef{
			Name:    r.Name,
			Title:   r.Title,
			Status:  r.Status,
			Updated: r.Updated,
		}
	}
	return &apipb.OperatingModeDetailResponse{
		Entry:             catalogEntryToProto(d.Entry),
		LinkedInitiatives: linked,
	}
}

func readinessToProto(r *ReadinessReport) *apipb.OperatingModeReadinessReport {
	if r == nil {
		return nil
	}
	dims := make([]*apipb.OperatingModeReadinessDimension, len(r.Dimensions))
	for i, d := range r.Dimensions {
		dims[i] = &apipb.OperatingModeReadinessDimension{
			Key:       d.Key,
			Label:     d.Label,
			Score:     d.Score,
			Rationale: d.Rationale,
		}
	}
	return &apipb.OperatingModeReadinessReport{
		Dimensions:   dims,
		OverallScore: r.OverallScore,
		Ready:        r.Ready,
	}
}

func roundItemsToProto(in []RoundItem) []*apipb.OperatingModeRoundItem {
	if len(in) == 0 {
		return nil
	}
	out := make([]*apipb.OperatingModeRoundItem, len(in))
	for i, it := range in {
		out[i] = &apipb.OperatingModeRoundItem{
			Ref:      it.Ref,
			Title:    it.Title,
			Status:   it.Status,
			Priority: int32(it.Priority),
			Effort:   it.Effort,
		}
	}
	return out
}

func handoffToProto(h *Handoff) *apipb.OperatingModeHandoff {
	if h == nil {
		return nil
	}
	return &apipb.OperatingModeHandoff{
		Summary:         h.Summary,
		CompletedPhases: h.CompletedPhases,
		ChangedFiles:    h.ChangedFiles,
		Tests:           h.Tests,
		Blockers:        h.Blockers,
		NextStep:        h.NextStep,
		Frontier:        h.Frontier,
		CreatedAt:       h.CreatedAt,
	}
}

func handoffsToProto(in []Handoff) []*apipb.OperatingModeHandoff {
	if len(in) == 0 {
		return nil
	}
	out := make([]*apipb.OperatingModeHandoff, len(in))
	for i := range in {
		out[i] = handoffToProto(&in[i])
	}
	return out
}

func artifactUpdatesToProto(in []ArtifactUpdate) []*apipb.OperatingModeArtifactUpdate {
	if len(in) == 0 {
		return nil
	}
	out := make([]*apipb.OperatingModeArtifactUpdate, len(in))
	for i, a := range in {
		out[i] = &apipb.OperatingModeArtifactUpdate{
			Path:        a.Path,
			ContentType: a.ContentType,
			Required:    a.Required,
			UpdatedAt:   a.UpdatedAt,
			Source:      a.Source,
		}
	}
	return out
}

func resolutionToProto(r PhaseResolutionRecord, ok bool) *apipb.OperatingModePhaseResolutionRecord {
	if !ok {
		return nil
	}
	return &apipb.OperatingModePhaseResolutionRecord{
		Outcome:            string(r.Outcome),
		Layer:              string(r.Layer),
		ChosenMessageIndex: int32(r.ChosenMessageIndex),
		MessagesScanned:    int32(r.MessagesScanned),
		Missing:            r.Missing,
		Violations:         r.Violations,
		Notes:              r.Notes,
		ClassifiedField:    r.ClassifiedField,
		ClassifiedValue:    r.ClassifiedValue,
	}
}

func roundEnvelopeToProto(r RoundEnvelope) *apipb.OperatingModeRoundEnvelope {
	resolution, resolutionOK := RoundPayload(r.Payload).Resolution()
	transitionClass, transitionClassOK := RoundPayload(r.Payload).TransitionClassification()
	return &apipb.OperatingModeRoundEnvelope{
		Round:                    int32(r.Round),
		Mode:                     r.Mode,
		ScopeKind:                r.ScopeKind,
		ScopeId:                  r.ScopeID,
		InitiativeName:           r.InitiativeName,
		Phase:                    r.Phase,
		RunStrategy:              r.RunStrategy,
		AgentProfileKey:          r.AgentProfileKey,
		GeneratedAt:              r.GeneratedAt,
		RunId:                    r.RunID,
		Status:                   string(r.Status),
		Readiness:                readinessToProto(r.Readiness),
		Items:                    roundItemsToProto(r.Items),
		ArtifactUpdates:          artifactUpdatesToProto(r.ArtifactUpdates),
		Handoffs:                 handoffsToProto(r.Handoffs),
		Payload:                  structFromMap(r.Payload),
		Error:                    r.Error,
		Resolution:               resolutionToProto(resolution, resolutionOK),
		TransitionClassification: resolutionToProto(transitionClass, transitionClassOK),
	}
}

func roundEnvelopesToProto(in []RoundEnvelope) []*apipb.OperatingModeRoundEnvelope {
	if len(in) == 0 {
		return nil
	}
	out := make([]*apipb.OperatingModeRoundEnvelope, len(in))
	for i, r := range in {
		out[i] = roundEnvelopeToProto(r)
	}
	return out
}

func artifactSnapshotsToProto(in []ArtifactSnapshot) []*apipb.OperatingModeArtifactSnapshot {
	if len(in) == 0 {
		return nil
	}
	out := make([]*apipb.OperatingModeArtifactSnapshot, len(in))
	for i, a := range in {
		out[i] = &apipb.OperatingModeArtifactSnapshot{
			Path:        a.Path,
			ContentType: a.ContentType,
			Required:    a.Required,
			Content:     a.Content,
			UpdatedAt:   a.UpdatedAt,
			SizeBytes:   a.SizeBytes,
		}
	}
	return out
}

func initiativeSnapshotToProto(s InitiativeSnapshot) *apipb.OperatingModeInitiativeSnapshot {
	return &apipb.OperatingModeInitiativeSnapshot{
		Name:               s.Name,
		Title:              s.Title,
		Description:        s.Description,
		Mode:               s.Mode,
		Items:              s.Items,
		AcceptanceCriteria: s.AcceptanceCriteria,
	}
}

func lockHolderToProto(h *initiativelock.Holder) *apipb.OperatingModeLockHolder {
	if h == nil {
		return nil
	}
	return &apipb.OperatingModeLockHolder{
		RunId:          h.RunID,
		Purpose:        h.Purpose,
		RoundNumber:    int32(h.RoundNumber),
		AcquiredAt:     h.AcquiredAt,
		AcquiredBy:     h.AcquiredBy,
		InitiativeName: h.InitiativeName,
	}
}

func workspacePhasesToProto(in []WorkspacePhase) []*apipb.OperatingModeWorkspacePhase {
	if len(in) == 0 {
		return nil
	}
	out := make([]*apipb.OperatingModeWorkspacePhase, len(in))
	for i, p := range in {
		out[i] = &apipb.OperatingModeWorkspacePhase{
			Phase:            p.Phase,
			PhaseKind:        p.PhaseKind,
			ActivityPurpose:  p.ActivityPurpose,
			ProfileKey:       p.ProfileKey,
			WritesRepo:       p.WritesRepo,
			OutputArtifacts:  artifactDefsToProto(p.OutputArtifacts),
			RequiresCriteria: p.RequiresCriteria,
			Startable:        p.Startable,
			Reason:           p.Reason,
			Next:             p.Next,
			AutoStartAfter:   p.AutoStartAfter,
			ExecutedBy:       p.ExecutedBy,
		}
	}
	return out
}

func transitionsToProto(in map[string][]string) map[string]*apipb.OperatingModeStringList {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]*apipb.OperatingModeStringList, len(in))
	for k, v := range in {
		out[k] = &apipb.OperatingModeStringList{Values: v}
	}
	return out
}

func workspaceModeToProto(m WorkspaceMode) *apipb.OperatingModeWorkspaceMode {
	return &apipb.OperatingModeWorkspaceMode{
		Mode:         m.Mode,
		Label:        m.Label,
		Description:  m.Description,
		TargetKind:   m.TargetKind,
		Capabilities: capabilitiesToProto(m.Capabilities),
		Phases:       workspacePhasesToProto(m.Phases),
		Terminal:     m.Terminal,
		Transitions:  transitionsToProto(m.Transitions),
		RunStrategy:  m.RunStrategy,
	}
}

func workspaceToProto(w Workspace) *apipb.OperatingModeWorkspace {
	return &apipb.OperatingModeWorkspace{
		InitiativeName: w.InitiativeName,
		Mode:           w.Mode,
		Definition:     workspaceModeToProto(w.Definition),
		Lock:           lockHolderToProto(w.Lock),
		Artifacts:      artifactSnapshotsToProto(w.Artifacts),
		Rounds:         roundEnvelopesToProto(w.Rounds),
	}
}

func activeExecutionsToProto(in []ActiveItemExecution) []*apipb.OperatingModeActiveItemExecution {
	if len(in) == 0 {
		return nil
	}
	out := make([]*apipb.OperatingModeActiveItemExecution, len(in))
	for i, e := range in {
		out[i] = &apipb.OperatingModeActiveItemExecution{
			ItemRef:     e.ItemRef,
			ExecutionId: e.ExecutionID,
			RunId:       e.RunID,
			Status:      e.Status,
		}
	}
	return out
}

func switchResultToProto(r SwitchModeResult) *apipb.OperatingModeSwitchResult {
	return &apipb.OperatingModeSwitchResult{
		InitiativeName:           r.InitiativeName,
		FromMode:                 r.FromMode,
		ToMode:                   r.ToMode,
		CanceledItemExecutions:   activeExecutionsToProto(r.CanceledItemExecutions),
		ActiveItemExecutions:     activeExecutionsToProto(r.ActiveItemExecutions),
		RequiresCancellation:     r.RequiresCancellation,
		OperatingModeWorkspaceId: r.OperatingModeWorkspaceID,
	}
}

func proposalResultToProto(r *ProposalApplyResult) *apipb.OperatingModeProposalApplyResult {
	if r == nil {
		return nil
	}
	outcomes := make([]*apipb.OperatingModeProposalOutcome, len(r.Outcomes))
	for i, o := range r.Outcomes {
		outcomes[i] = &apipb.OperatingModeProposalOutcome{
			MutationId: o.MutationID,
			Op:         o.Op,
			Target:     o.Target,
			Applied:    o.Applied,
			Skipped:    o.Skipped,
			Error:      o.Error,
		}
	}
	return &apipb.OperatingModeProposalApplyResult{
		Outcomes: outcomes,
		Applied:  int32(r.Applied),
		Failed:   int32(r.Failed),
		Skipped:  int32(r.Skipped),
		Created:  int32(r.Created),
		Updated:  int32(r.Updated),
	}
}

func backlogSyncResultToProto(r BacklogSyncResult) *apipb.OperatingModeBacklogSyncResult {
	completed := make([]*apipb.OperatingModeBacklogCompletionResult, len(r.CompletedItems))
	for i, c := range r.CompletedItems {
		completed[i] = &apipb.OperatingModeBacklogCompletionResult{
			ItemRef:    c.ItemRef,
			FromStatus: c.FromStatus,
			ToStatus:   c.ToStatus,
		}
	}
	return &apipb.OperatingModeBacklogSyncResult{
		InitiativeName: r.InitiativeName,
		Mode:           r.Mode,
		Phase:          r.Phase,
		Round:          int32(r.Round),
		RunId:          r.RunID,
		CompletedItems: completed,
		ProposalResult: proposalResultToProto(r.ProposalResult),
		Noop:           r.Noop,
	}
}

func progressToProto(p *ProgressState) *apipb.OperatingModeProgressState {
	if p == nil {
		return nil
	}
	return &apipb.OperatingModeProgressState{
		Decision:        string(p.Decision),
		CompletedPhases: p.CompletedPhases,
		CurrentPhase:    p.CurrentPhase,
		Rationale:       p.Rationale,
		UpdatedAt:       p.UpdatedAt,
	}
}

func backlogSyncPlanToProto(p *BacklogSyncPlan) *apipb.OperatingModeBacklogSyncPlan {
	if p == nil {
		return nil
	}
	return &apipb.OperatingModeBacklogSyncPlan{
		CompletedItems: p.CompletedItems,
		CreatedItems:   p.CreatedItems,
		UpdatedItems:   p.UpdatedItems,
		Proposal:       structFromRaw(p.Proposal),
		Rationale:      p.Rationale,
	}
}

func phaseResultToProto(r PhaseResult) *apipb.OperatingModePhaseResult {
	artifacts := make([]*apipb.OperatingModeArtifactResult, len(r.Artifacts))
	for i, a := range r.Artifacts {
		artifacts[i] = &apipb.OperatingModeArtifactResult{
			Path:        a.Path,
			Content:     a.Content,
			ContentType: a.ContentType,
		}
	}
	return &apipb.OperatingModePhaseResult{
		Artifacts:    artifacts,
		Handoff:      handoffToProto(r.Handoff),
		Handoffs:     handoffsToProto(r.Handoffs),
		Readiness:    readinessToProto(r.Readiness),
		Progress:     progressToProto(r.Progress),
		Verdict:      r.Verdict,
		ReplanNeeded: r.ReplanNeeded,
		BacklogSync:  backlogSyncPlanToProto(r.BacklogSync),
	}
}

func simulationTransitionToProto(t *SimulationTransition) *apipb.OperatingModeSimulationTransition {
	if t == nil {
		return nil
	}
	return &apipb.OperatingModeSimulationTransition{
		From:          t.From,
		To:            t.To,
		ConditionKind: t.ConditionKind,
		Label:         t.Label,
		Field:         t.Field,
		Value:         t.Value,
	}
}

func simulationInputsToProto(in SimulationInputs) *apipb.OperatingModeSimulationInputs {
	return &apipb.OperatingModeSimulationInputs{
		Initiative:         initiativeSnapshotToProto(in.Initiative),
		Items:              roundItemsToProto(in.Items),
		Artifacts:          artifactSnapshotsToProto(in.Artifacts),
		PriorRounds:        roundEnvelopesToProto(in.PriorRounds),
		AcceptanceCriteria: in.AcceptanceCriteria,
	}
}

func simulationToProto(s SimulationResponse) *apipb.OperatingModeSimulationResponse {
	presets := make([]*apipb.OperatingModeSimulationPreset, len(s.Presets))
	for i, p := range s.Presets {
		presets[i] = &apipb.OperatingModeSimulationPreset{
			Id:          p.ID,
			Label:       p.Label,
			Description: p.Description,
			Branch:      p.Branch,
			Scenario:    p.Scenario,
		}
	}
	trace := make([]*apipb.OperatingModeSimulationStep, len(s.Trace))
	for i, step := range s.Trace {
		trace[i] = &apipb.OperatingModeSimulationStep{
			Index:           int32(step.Index),
			Phase:           step.Phase,
			PhaseKind:       step.PhaseKind,
			Inputs:          simulationInputsToProto(step.Inputs),
			Output:          phaseResultToProto(step.Output),
			Round:           roundEnvelopeToProto(step.Round),
			Transition:      simulationTransitionToProto(step.Transition),
			Terminal:        step.Terminal,
			SkillId:         step.SkillID,
			ProfileKey:      step.ProfileKey,
			PromptVariables: step.PromptVariables,
		}
	}
	return &apipb.OperatingModeSimulationResponse{
		Mode:         s.Mode,
		Label:        s.Label,
		Presets:      presets,
		ActivePreset: s.ActivePreset,
		Initiative:   initiativeSnapshotToProto(s.Initiative),
		Trace:        trace,
	}
}

func renderPromptToProto(r RenderPromptResponse) *apipb.OperatingModeRenderPromptResponse {
	return &apipb.OperatingModeRenderPromptResponse{
		Mode:           r.Mode,
		Preset:         r.Preset,
		StepIndex:      int32(r.StepIndex),
		Phase:          r.Phase,
		SkillId:        r.SkillID,
		ProfileKey:     r.ProfileKey,
		Variables:      r.Variables,
		Prompt:         r.Prompt,
		Degraded:       r.Degraded,
		DegradedReason: r.DegradedReason,
	}
}

func scaffoldResultToProto(r ScaffoldResult) *apipb.OperatingModeScaffoldResponse {
	return &apipb.OperatingModeScaffoldResponse{
		Mode:         r.Mode,
		Dir:          r.Dir,
		CreatedFiles: append([]string(nil), r.CreatedFiles...),
	}
}

func validationReportToProto(r ValidationReport) *apipb.OperatingModeValidateResponse {
	return &apipb.OperatingModeValidateResponse{
		Mode:              r.Mode,
		Ok:                r.OK,
		Errors:            append([]string(nil), r.Errors...),
		PhaseCount:        int32(r.PhaseCount),
		ExampleRuns:       int32(r.ExampleRuns),
		Summary:           r.Summary,
		UncoveredBranches: append([]string(nil), r.UncoveredBranches...),
	}
}
