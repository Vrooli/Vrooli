package operatingmode

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"swarm-manager/internal/evidence"
)

func (s *Service) Workspace(ctx context.Context, initiativeName string) (Workspace, error) {
	init, err := s.initiatives.LoadInitiative(strings.TrimSpace(initiativeName))
	if err != nil {
		return Workspace{}, err
	}
	def, err := DefinitionFor(Mode(init.Mode))
	if err != nil {
		return Workspace{}, err
	}
	if overrides, overlayErr := s.loadOverlay(); overlayErr == nil {
		def = applyOverlay(def, overrides[def.Mode])
	}
	if def.Mode == ModeItemLevel {
		return Workspace{
			InitiativeName: init.Name,
			Mode:           string(def.Mode),
			Definition:     workspaceMode(def, nil, init.AcceptanceCriteria),
		}, nil
	}
	_, legacyAmbiguous, err := s.store.AdoptLegacyExecution(init.Name, def)
	if err != nil {
		return Workspace{}, err
	}
	rounds, err := s.store.ListRounds(init.Name, def.Mode)
	if err != nil {
		return Workspace{}, err
	}
	for i := range rounds {
		if legacyAmbiguous && rounds[i].ExecutionID == "" {
			continue
		}
		if isRoundActive(rounds[i]) {
			refreshed, refreshErr := s.RefreshRound(ctx, init.Name, def.Mode, rounds[i].Round)
			if refreshErr != nil {
				slog.Warn("operating mode: refresh round failed", "err", refreshErr, "initiative", init.Name, "round", rounds[i].Round)
				continue
			}
			rounds[i] = refreshed
		}
	}
	artifacts, err := s.store.ListDeclaredArtifacts(init.Name, def.Mode)
	if err != nil {
		return Workspace{}, err
	}
	holder, err := s.lock.Inspect(init.Name)
	if err != nil {
		return Workspace{}, err
	}
	executions, err := s.store.ListExecutions(init.Name, def.Mode)
	if err != nil {
		return Workspace{}, err
	}
	evidenceByExecution := map[string][]evidence.Record{}
	if s.evidenceService != nil {
		for _, execution := range executions {
			records, err := s.evidenceService.ListByOwnerID(ctx, evidence.OwnerOperatingModeExecution, execution.ExecutionID)
			if err != nil {
				return Workspace{}, err
			}
			evidenceByExecution[execution.ExecutionID] = records
		}
	}
	return Workspace{
		InitiativeName:      init.Name,
		Mode:                string(def.Mode),
		Definition:          workspaceMode(def, rounds, init.AcceptanceCriteria),
		Lock:                holder,
		Artifacts:           artifacts,
		Rounds:              rounds,
		Executions:          executions,
		EvidenceByExecution: evidenceByExecution,
	}, nil
}

// Catalog returns the user-visible mode list. Registry definitions are
// merged with persisted overrides (label/description) and annotated with the
// current per-mode initiative usage count. Sidebar and CLI both consume this.
func (s *Service) Catalog() (ModeCatalog, error) {
	if err := ValidateRegistry(); err != nil {
		return ModeCatalog{}, err
	}
	overrides, err := s.loadOverlay()
	if err != nil {
		return ModeCatalog{}, err
	}
	usage, err := s.modeUsageCounts()
	if err != nil {
		return ModeCatalog{}, err
	}
	modes := Modes()
	entries := make([]ModeCatalogEntry, 0, len(modes))
	for _, mode := range modes {
		def, err := DefinitionFor(mode)
		if err != nil {
			return ModeCatalog{}, err
		}
		def = applyOverlay(def, overrides[mode])
		entries = append(entries, buildCatalogEntry(def, usage[mode]))
	}
	return ModeCatalog{Modes: entries}, nil
}

// GetMode returns a single mode's catalog entry plus the initiatives
// currently using it. Backs the OperatingModeService.GetMode Connect RPC.
func (s *Service) GetMode(mode Mode) (ModeDetail, error) {
	if err := ValidateRegistry(); err != nil {
		return ModeDetail{}, err
	}
	def, err := DefinitionFor(mode)
	if err != nil {
		return ModeDetail{}, err
	}
	overrides, err := s.loadOverlay()
	if err != nil {
		return ModeDetail{}, err
	}
	def = applyOverlay(def, overrides[def.Mode])
	linked, err := s.InitiativesUsingMode(def.Mode)
	if err != nil {
		return ModeDetail{}, err
	}
	entry := buildCatalogEntry(def, len(linked))
	return ModeDetail{Entry: entry, LinkedInitiatives: linked}, nil
}

// UpdateMode persists user-editable overrides (label, description) for a
// mode via the overlay store. Returns the merged detail. Empty-string label
// is rejected; nil pointers leave the field unchanged; an explicit empty
// override clears the row (restoring defaults).
func (s *Service) UpdateMode(mode Mode, override Override) (ModeDetail, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return ModeDetail{}, err
	}
	if override.Label != nil && strings.TrimSpace(*override.Label) == "" {
		return ModeDetail{}, fmt.Errorf("label cannot be blank")
	}
	if s.overlay == nil {
		return ModeDetail{}, fmt.Errorf("operating-mode overlay store is not configured")
	}
	if err := s.overlay.Save(def.Mode, override); err != nil {
		return ModeDetail{}, err
	}
	return s.GetMode(def.Mode)
}

// InitiativesUsingMode walks the initiatives list and returns the
// compact references for those currently bound to the given mode. Used by
// catalog usage counts and the details page linked-initiative list.
func (s *Service) InitiativesUsingMode(mode Mode) ([]InitiativeRef, error) {
	if s.initLister == nil {
		return nil, nil
	}
	all, err := s.initLister.ListInitiatives()
	if err != nil {
		return nil, err
	}
	target := NormalizeMode(string(mode))
	out := make([]InitiativeRef, 0)
	for _, init := range all {
		if NormalizeMode(init.Mode) != target {
			continue
		}
		out = append(out, InitiativeRef{
			Name:    init.Name,
			Title:   init.Title,
			Status:  init.Status,
			Updated: init.Updated,
		})
	}
	return out, nil
}

func (s *Service) loadOverlay() (map[Mode]Override, error) {
	if s.overlay == nil {
		return map[Mode]Override{}, nil
	}
	overrides, err := s.overlay.Load()
	if err != nil {
		// Corrupted overlay must not break the catalog — fall back to
		// defaults but surface the cause via the slog logger.
		slog.Warn("operating-mode: overlay load failed; using registry defaults", "err", err)
		return map[Mode]Override{}, nil
	}
	return overrides, nil
}

func (s *Service) modeUsageCounts() (map[Mode]int, error) {
	counts := map[Mode]int{}
	if s.initLister == nil {
		return counts, nil
	}
	all, err := s.initLister.ListInitiatives()
	if err != nil {
		return nil, err
	}
	for _, init := range all {
		counts[NormalizeMode(init.Mode)]++
	}
	return counts, nil
}

func buildCatalogEntry(def Definition, usageCount int) ModeCatalogEntry {
	capabilities := modeCapabilities(def)
	inputContract := compiledInputContractForProjection(def)
	entry := ModeCatalogEntry{
		Mode:                   string(def.Mode),
		Label:                  def.Label,
		Description:            def.Description,
		BestFor:                append([]string(nil), def.BestFor...),
		NotFor:                 append([]string(nil), def.NotFor...),
		Tradeoffs:              append([]string(nil), def.Tradeoffs...),
		WhenInDoubtPickInstead: string(def.WhenInDoubtPickInstead),
		UsageCount:             usageCount,
		TargetKind:             string(def.Target.Kind),
		RunStrategy:            string(def.RunStrategy.Kind),
		WorkspaceTabID:         def.UI.WorkspaceTabID,
		Capabilities:           capabilities,
		Default:                def.Mode == DefaultMode(),
		Switchable:             true,
		SupportsPhases:         capabilities.SupportsPhases,
		InputContract:          inputContract,
	}
	if !entry.SupportsPhases {
		return entry
	}
	terminalSet := make(map[Phase]struct{}, len(def.PhaseGraph.Terminal))
	for _, terminal := range def.PhaseGraph.Terminal {
		terminalSet[terminal] = struct{}{}
	}
	for _, phaseName := range orderedPhases(def) {
		phase := def.PhaseGraph.Phases[phaseName]
		_, isTerminal := terminalSet[phaseName]
		entry.Phases = append(entry.Phases, ModeCatalogPhase{
			Phase:                 string(phase.Phase),
			PhaseKind:             string(phase.Kind),
			Label:                 humanizeToken(string(phase.Phase)),
			Title:                 phase.PromptCatalog.Title,
			Purpose:               phase.PromptCatalog.Purpose,
			Trigger:               phase.PromptCatalog.Trigger,
			ProfileKey:            phase.ProfileKey,
			WritesRepo:            phase.WritesRepo,
			RequiresCriteria:      phase.RequiresCriteria,
			IsStart:               phaseName == def.PhaseGraph.StartPhase,
			IsTerminal:            isTerminal,
			OutputArtifacts:       phase.OutputArtifacts,
			Reads:                 summarizePhaseReads(def, phase),
			OutputContract:        summarizeContract(phase.OutputContract),
			CatalogID:             phase.CatalogID,
			SkillID:               phase.SkillID,
			ActivityPurpose:       phase.ActivityPurpose,
			LockPurpose:           phase.LockPurpose,
			ResultBindings:        phase.ResultBindings,
			SamplesReplanRate:     def.Metrics.CountsReplanSample(phaseName),
			SamplesAcceptanceRate: def.Metrics.CountsAcceptanceSample(phaseName),
			AutoStartAfter:        phasesToStrings(phase.AutoStartAfter),
			ExecutedBy:            string(phase.ExecutedBy),
			Classification:        summarizePhaseClassification(phase.TransitionClassification),
		})
	}
	entry.PhaseGraph = buildCatalogPhaseGraph(def)
	return entry
}

// summarizePhaseClassification projects a phase's classification-on-transition
// contract for the catalog. Nil in, nil out — a phase whose edges all route on
// directly-emitted fields carries no classification.
func summarizePhaseClassification(c *TransitionClassification) *PhaseClassificationSummary {
	if c == nil {
		return nil
	}
	return &PhaseClassificationSummary{
		Field:       c.Field,
		Enum:        append([]string(nil), c.Enum...),
		From:        c.From,
		Description: c.Description,
	}
}

func buildCatalogPhaseGraph(def Definition) *ModeCatalogPhaseGraph {
	terminal := make([]string, 0, len(def.PhaseGraph.Terminal))
	for _, phase := range def.PhaseGraph.Terminal {
		terminal = append(terminal, string(phase))
	}
	transitions := make([]ModeCatalogTransition, 0)
	for _, from := range orderedPhases(def) {
		// A phase's classified edge expands into eq-guards over the classification
		// field; mark those guards so surfaces render them as classification-on-
		// transition rather than a directly-emitted routing field.
		classifiedField := ""
		if c := def.PhaseGraph.Phases[from].TransitionClassification; c != nil {
			classifiedField = c.Field
		}
		if guards := def.PhaseGraph.Guards[from]; len(guards) > 0 {
			for _, gt := range guards {
				kind := GuardKind(gt.When)
				label := GuardLabel(gt.When)
				field := gt.When.Field
				value := renderGuardValue(gt.When.Value)
				for _, to := range gt.To {
					transitions = append(transitions, ModeCatalogTransition{
						From:          string(from),
						To:            string(to),
						ConditionKind: kind,
						Label:         label,
						Field:         field,
						Value:         value,
						Classified:    classifiedField != "" && field == classifiedField,
					})
				}
			}
			continue
		}
		for _, to := range def.PhaseGraph.Transitions[from] {
			transitions = append(transitions, ModeCatalogTransition{
				From:          string(from),
				To:            string(to),
				ConditionKind: GuardOpAlways,
				Label:         "always",
			})
		}
	}
	sort.SliceStable(transitions, func(i, j int) bool {
		if transitions[i].From != transitions[j].From {
			return transitions[i].From < transitions[j].From
		}
		if transitions[i].To != transitions[j].To {
			return transitions[i].To < transitions[j].To
		}
		return transitions[i].Label < transitions[j].Label
	})
	graph := &ModeCatalogPhaseGraph{
		StartPhase:  string(def.PhaseGraph.StartPhase),
		Terminal:    terminal,
		Transitions: transitions,
	}
	if len(def.Metrics.AcceptedVerdicts) > 0 {
		graph.AcceptedVerdicts = append([]string(nil), def.Metrics.AcceptedVerdicts...)
	}
	return graph
}

func workspaceMode(def Definition, rounds []RoundEnvelope, acceptanceCriteria []string) WorkspaceMode {
	actions := map[Phase]PhaseAction{}
	for _, action := range ComputePhaseActions(PhaseStateInput{
		Definition:         def,
		Rounds:             rounds,
		AcceptanceCriteria: acceptanceCriteria,
		RequireRunStrategy: true,
	}) {
		actions[action.Phase] = action
	}
	phases := make([]WorkspacePhase, 0, len(def.PhaseGraph.Phases))
	for _, phaseName := range orderedPhases(def) {
		phase := def.PhaseGraph.Phases[phaseName]
		action := actions[phaseName]
		phases = append(phases, WorkspacePhase{
			Phase:            string(phase.Phase),
			PhaseKind:        string(phase.Kind),
			ActivityPurpose:  phase.ActivityPurpose,
			ProfileKey:       phase.ProfileKey,
			WritesRepo:       phase.WritesRepo,
			OutputArtifacts:  phase.OutputArtifacts,
			RequiresCriteria: phase.RequiresCriteria,
			Startable:        action.Startable,
			Reason:           action.Reason,
			Next:             action.Next,
			AutoStartAfter:   phasesToStrings(phase.AutoStartAfter),
			ExecutedBy:       string(phase.ExecutedBy),
		})
	}
	terminal := make([]string, 0, len(def.PhaseGraph.Terminal))
	for _, phase := range def.PhaseGraph.Terminal {
		terminal = append(terminal, string(phase))
	}
	transitions := make(map[string][]string, len(def.PhaseGraph.Transitions))
	for from, to := range def.PhaseGraph.Transitions {
		key := string(from)
		transitions[key] = make([]string, 0, len(to))
		for _, next := range to {
			transitions[key] = append(transitions[key], string(next))
		}
	}
	return WorkspaceMode{
		Mode:          string(def.Mode),
		Label:         def.Label,
		Description:   def.Description,
		TargetKind:    string(def.Target.Kind),
		Capabilities:  modeCapabilities(def),
		Phases:        phases,
		Terminal:      terminal,
		Transitions:   transitions,
		RunStrategy:   string(def.RunStrategy.Kind),
		InputContract: compiledInputContractForProjection(def),
	}
}

func compiledInputContractForProjection(def Definition) CompiledInputContract {
	bundle, _, err := pinDefinitionBundle(def, DefinitionFor)
	if err != nil {
		return CompiledInputContract{}
	}
	root, err := bundle.RootDefinition()
	if err != nil {
		return CompiledInputContract{}
	}
	compiled, err := CompileInputContract(bundle.Definitions, root)
	if err != nil {
		return CompiledInputContract{}
	}
	return compiled
}

// phasesToStrings projects a slice of typed Phase values to plain strings for
// JSON wire shapes. Returns nil for empty inputs to keep `omitempty` honest.
func phasesToStrings(phases []Phase) []string {
	if len(phases) == 0 {
		return nil
	}
	out := make([]string, 0, len(phases))
	for _, p := range phases {
		out = append(out, string(p))
	}
	return out
}

func modeCapabilities(def Definition) ModeCapabilities {
	capabilities := ModeCapabilities{
		SupportsPhases:        len(def.PhaseGraph.Phases) > 0,
		UsesItemExecutionFlow: def.RunStrategy.Kind == RunStrategyExistingItemFlow,
	}
	capabilities.CanStartPhases = capabilities.SupportsPhases
	capabilities.CanCompleteItems = hasBacklogSyncCapability(def.BacklogSync, BacklogSyncMarkComplete)
	capabilities.CanApplyBacklogSyncProposals = hasBacklogSyncCapability(def.BacklogSync, BacklogSyncProposeMutations)
	for _, phase := range def.PhaseGraph.Phases {
		if phase.RequiresCriteria {
			capabilities.RequiresAcceptanceCriteria = true
		}
		if len(phase.OutputArtifacts) > 0 || len(phase.ResultBindings) > 0 {
			capabilities.SupportsArtifacts = true
		}
		if phase.OutputContract.RequiresHandoff {
			capabilities.SupportsHandoffs = true
		}
	}
	return capabilities
}
