package operatingmode

import (
	"context"
	"log/slog"
	"strings"
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
	if def.Mode == ModeItemLevel {
		return Workspace{
			InitiativeName: init.Name,
			Mode:           string(def.Mode),
			Definition:     workspaceMode(def, nil, init.AcceptanceCriteria),
		}, nil
	}
	rounds, err := s.store.ListRounds(init.Name, def.Mode)
	if err != nil {
		return Workspace{}, err
	}
	for i := range rounds {
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
	return Workspace{
		InitiativeName: init.Name,
		Mode:           string(def.Mode),
		Definition:     workspaceMode(def, rounds, init.AcceptanceCriteria),
		Lock:           holder,
		Artifacts:      artifacts,
		Rounds:         rounds,
	}, nil
}

func (s *Service) Catalog() (ModeCatalog, error) {
	if err := ValidateRegistry(); err != nil {
		return ModeCatalog{}, err
	}
	modes := Modes()
	entries := make([]ModeCatalogEntry, 0, len(modes))
	for _, mode := range modes {
		def, err := DefinitionFor(mode)
		if err != nil {
			return ModeCatalog{}, err
		}
		capabilities := modeCapabilities(def)
		entry := ModeCatalogEntry{
			Mode:           string(def.Mode),
			Label:          def.Label,
			ScopeKind:      string(def.Scope.Kind),
			RunStrategy:    string(def.RunStrategy.Kind),
			WorkspaceTabID: def.UI.WorkspaceTabID,
			Capabilities:   capabilities,
			Default:        def.Mode == DefaultMode(),
			Switchable:     true,
			SupportsPhases: capabilities.SupportsPhases,
		}
		if entry.SupportsPhases {
			for _, phaseName := range orderedPhases(def) {
				phase := def.PhaseGraph.Phases[phaseName]
				entry.Phases = append(entry.Phases, ModeCatalogPhase{
					Phase:            string(phase.Phase),
					ProfileKey:       phase.ProfileKey,
					WritesRepo:       phase.WritesRepo,
					RequiresCriteria: phase.RequiresCriteria,
				})
			}
		}
		entries = append(entries, entry)
	}
	return ModeCatalog{Modes: entries}, nil
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
			ActivityPurpose:  phase.ActivityPurpose,
			ProfileKey:       phase.ProfileKey,
			WritesRepo:       phase.WritesRepo,
			OutputArtifacts:  phase.OutputArtifacts,
			RequiresCriteria: phase.RequiresCriteria,
			Startable:        action.Startable,
			Reason:           action.Reason,
			Next:             action.Next,
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
		Mode:         string(def.Mode),
		Label:        def.Label,
		ScopeKind:    string(def.Scope.Kind),
		Capabilities: modeCapabilities(def),
		Phases:       phases,
		Terminal:     terminal,
		Transitions:  transitions,
		RunStrategy:  string(def.RunStrategy.Kind),
	}
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
