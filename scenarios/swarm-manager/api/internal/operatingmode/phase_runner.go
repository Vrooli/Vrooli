package operatingmode

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/initiativelock"
)

type phaseContext struct {
	init      InitiativeSnapshot
	def       Definition
	phaseDef  PhaseDefinition
	items     []RoundItem
	artifacts []ArtifactSnapshot
	rounds    []RoundEnvelope
}

func (s *Service) StartPhase(ctx context.Context, req StartPhaseRequest) (RoundEnvelope, error) {
	init, def, phaseDef, err := s.resolvePhase(req.InitiativeName, req.Phase)
	if err != nil {
		return RoundEnvelope{}, err
	}
	if def.Mode == ModeItemLevel {
		return RoundEnvelope{}, fmt.Errorf("item-level mode phases are owned by the existing backlog execution flow")
	}

	ctxData, err := s.collectPhaseContext(init, def, phaseDef)
	if err != nil {
		return RoundEnvelope{}, err
	}
	if err := ValidatePhaseStart(def, ctxData.rounds, phaseDef.Phase, init.AcceptanceCriteria); err != nil {
		return RoundEnvelope{}, err
	}
	round, err := s.store.CreateRound(RoundEnvelope{
		Mode:            string(def.Mode),
		InitiativeName:  init.Name,
		ScopeID:         init.Name,
		Phase:           string(phaseDef.Phase),
		AgentProfileKey: phaseDef.ProfileKey,
		Status:          RoundStatusReserved,
		Items:           ctxData.items,
		Payload: map[string]any{
			"operator_note": strings.TrimSpace(req.Note),
			"skill_id":      phaseDef.SkillID,
			"catalog_id":    phaseDef.CatalogID,
		},
	})
	if err != nil {
		return RoundEnvelope{}, err
	}

	holder := initiativelock.Holder{
		RunID:       fmt.Sprintf("provisional-operating-mode-%d-%d", round.Round, s.clock().UTC().UnixNano()),
		Purpose:     phaseDef.LockPurpose,
		RoundNumber: round.Round,
		AcquiredBy:  strings.TrimSpace(req.RequestedBy),
	}
	provisionalRunID := holder.RunID
	if req.Override {
		s.preemptLockHolder(ctx, init.Name)
		if err := s.lock.AcquireOverride(init.Name, holder); err != nil {
			s.persistPhaseStartFailure(round, err, "lock_failed_at")
			return RoundEnvelope{}, fmt.Errorf("override lock: %w", err)
		}
	} else if err := s.lock.Acquire(init.Name, holder); err != nil {
		s.persistPhaseStartFailure(round, err, "lock_failed_at")
		return RoundEnvelope{}, err
	}

	prompt, err := s.buildPrompt(ctx, ctxData, round, req.Note)
	if err != nil {
		_ = s.lock.Release(init.Name, provisionalRunID)
		round = s.persistPhaseStartFailure(round, err, "prompt_failed_at")
		slog.Warn("operating mode prompt rendering failed; phase start aborted",
			"err", err, "initiative", init.Name, "mode", def.Mode, "phase", phaseDef.Phase)
		return round, fmt.Errorf("render operating-mode prompt: %w", err)
	}

	round.Status = RoundStatusAgentRunning
	if err := s.store.SaveRound(round); err != nil {
		_ = s.lock.Release(init.Name, provisionalRunID)
		return RoundEnvelope{}, fmt.Errorf("save reserved round: %w", err)
	}

	spawnReq := agentmanager.InitiativeSpawnRequest{
		Name:        init.Name,
		Title:       fmt.Sprintf("%s: %s", def.Label, phaseDef.Phase),
		Description: strings.TrimSpace(init.Description),
		Prompt:      prompt,
		ScopePath:   ".",
		ProjectRoot: s.scenarioRoot,
		CreatedBy:   s.requestedBy,
		Purpose:     phaseDef.ActivityPurpose,
		RoundNumber: round.Round,
		RoundSlug:   string(phaseDef.Phase),
		ProfileKey:  phaseDef.ProfileKey,
	}
	if s.activity != nil {
		// PhaseKind drives lane assignment in agentactivity. The activity
		// purpose is a mode-defined dynamic string (e.g.
		// "holistic_loop_investigate"), so without phaseKind LaneOf would
		// return an error and the spawn would be rejected.
		spec := agentactivity.Spec{
			OwnerType:   agentactivity.OwnerInitiative,
			OwnerName:   init.Name,
			OwnerTitle:  init.Title,
			Purpose:     agentactivity.Purpose(phaseDef.ActivityPurpose),
			PhaseKind:   string(phaseDef.Kind),
			RequestedBy: s.requestedBy,
			Metadata: map[string]string{
				"entrypoint":        "initiative.operating_mode.phase",
				"operating_mode":    string(def.Mode),
				"phase":             string(phaseDef.Phase),
				"run_strategy":      string(def.RunStrategy.Kind),
				"round_number":      fmt.Sprintf("%d", round.Round),
				"artifact_set":      def.Artifact.Root,
				"agent_profile_key": phaseDef.ProfileKey,
			},
		}
		ctx = agentactivity.WithSpec(ctx, spec)
	}

	result, err := s.spawnInitiative(ctx, spawnReq)
	if err != nil {
		_ = s.lock.Release(init.Name, provisionalRunID)
		round = s.persistPhaseStartFailure(round, err, "spawn_failed_at")
		return round, fmt.Errorf("spawn operating-mode phase: %w", err)
	}

	round.RunID = strings.TrimSpace(result.RunID)
	round.GeneratedAt = s.clock().UTC().Format(time.RFC3339)
	if err := s.store.SaveRound(round); err != nil {
		slog.Warn("operating mode: persist run id failed", "err", err, "initiative", init.Name, "run_id", result.RunID)
	}
	holder.RunID = round.RunID
	if err := s.lock.AcquireOverride(init.Name, holder); err != nil {
		slog.Warn("operating mode: lock run-id swap failed", "err", err, "initiative", init.Name, "run_id", round.RunID)
		_ = s.lock.Release(init.Name, provisionalRunID)
		if s.agent != nil {
			if stopErr := s.agent.StopRun(ctx, round.RunID); stopErr != nil {
				slog.Warn("operating mode: stop run after lock swap failure failed", "err", stopErr, "initiative", init.Name, "run_id", round.RunID)
			}
		}
		round = s.persistPhaseStartFailure(round, err, "lock_swap_failed_at")
		return round, fmt.Errorf("swap operating-mode lock holder: %w", err)
	}
	s.emitPhaseStarted(round)
	return round, nil
}

func (s *Service) persistPhaseStartFailure(round RoundEnvelope, err error, timestampKey string) RoundEnvelope {
	round.Status = RoundStatusFailed
	if err != nil {
		round.Error = err.Error()
	}
	if strings.TrimSpace(timestampKey) != "" {
		MutableRoundPayload(&round).setString(timestampKey, s.clock().UTC().Format(time.RFC3339))
	}
	if saveErr := s.store.SaveRound(round); saveErr != nil {
		slog.Warn("operating mode: persist phase start failure failed", "err", saveErr, "initiative", round.InitiativeName, "round", round.Round)
	}
	s.emitPhaseFailed(round, round.Error)
	return round
}

func (s *Service) resolvePhase(initiativeName, phase string) (InitiativeSnapshot, Definition, PhaseDefinition, error) {
	init, err := s.initiatives.LoadInitiative(strings.TrimSpace(initiativeName))
	if err != nil {
		return InitiativeSnapshot{}, Definition{}, PhaseDefinition{}, err
	}
	def, err := DefinitionFor(Mode(init.Mode))
	if err != nil {
		return InitiativeSnapshot{}, Definition{}, PhaseDefinition{}, err
	}
	phaseName := Phase(strings.ToLower(strings.TrimSpace(phase)))
	if phaseName == "" {
		phaseName = def.PhaseGraph.StartPhase
	}
	phaseDef, err := def.PhaseDefinition(phaseName)
	if err != nil {
		return InitiativeSnapshot{}, Definition{}, PhaseDefinition{}, err
	}
	return init, def, phaseDef, nil
}

func (s *Service) collectPhaseContext(init InitiativeSnapshot, def Definition, phaseDef PhaseDefinition) (phaseContext, error) {
	rounds, err := s.store.ListRounds(init.Name, def.Mode)
	if err != nil {
		return phaseContext{}, err
	}
	artifacts, err := s.store.ListDeclaredArtifacts(init.Name, def.Mode)
	if err != nil {
		return phaseContext{}, err
	}
	items := s.collectItems(init.Items)
	return phaseContext{
		init:      init,
		def:       def,
		phaseDef:  phaseDef,
		items:     items,
		artifacts: artifacts,
		rounds:    rounds,
	}, nil
}

func (s *Service) collectItems(refs []string) []RoundItem {
	items := make([]RoundItem, 0, len(refs))
	for _, ref := range refs {
		trimmed := strings.TrimSpace(ref)
		if trimmed == "" {
			continue
		}
		item := RoundItem{Ref: trimmed}
		if s.backlog != nil {
			parts := strings.SplitN(trimmed, "/", 2)
			if len(parts) == 2 {
				loaded, err := s.backlog.LoadBacklogItem(parts[0], parts[1])
				if err == nil {
					item.Title = loaded.Title
					item.Status = loaded.Status
					item.Priority = loaded.Priority
					item.Effort = loaded.Effort
				}
			}
		}
		items = append(items, item)
	}
	return items
}

func (s *Service) spawnInitiative(ctx context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error) {
	if s.activity != nil {
		return s.activity.SpawnInitiative(ctx, req)
	}
	if s.agent == nil {
		return agentmanager.RunResult{}, agentmanager.ErrNotAvailable
	}
	return s.agent.SpawnInitiative(ctx, req)
}

func (s *Service) preemptLockHolder(ctx context.Context, initiativeName string) {
	holder, err := s.lock.Inspect(initiativeName)
	if err != nil || holder == nil || holder.RunID == "" || s.agent == nil {
		return
	}
	if err := s.agent.StopRun(ctx, holder.RunID); err != nil {
		slog.Warn("operating mode: failed to stop preempted lock holder", "err", err, "initiative", initiativeName, "run_id", holder.RunID)
	}
}
