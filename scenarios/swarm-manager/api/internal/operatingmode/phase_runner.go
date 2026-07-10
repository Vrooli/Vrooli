package operatingmode

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/initiativelock"
)

func (s *Service) StartPhase(ctx context.Context, req StartPhaseRequest) (RoundEnvelope, error) {
	init, def, phaseDef, err := s.resolvePhase(req.InitiativeName, req.Phase)
	if err != nil {
		return RoundEnvelope{}, err
	}
	if def.Mode == ModeItemLevel {
		return RoundEnvelope{}, fmt.Errorf("item-level mode phases are owned by the existing backlog execution flow")
	}

	rc, err := s.collectRunContext(ctx, def, phaseDef, init.Name)
	if err != nil {
		return RoundEnvelope{}, err
	}
	return s.startResolvedPhase(ctx, rc, req.Note, req.Override, req.RequestedBy)
}

// StartTargetPhase is the plan-first entry point: it starts a mode round
// directly on a non-initiative target (a plan-manager plan or a plan-ref),
// with no initiative created and no backlog ceremony. Initiative-target modes
// keep the initiative-keyed StartPhase surface; naming one here is a typed
// error rather than a second way to drive initiatives.
func (s *Service) StartTargetPhase(ctx context.Context, req StartTargetPhaseRequest) (RoundEnvelope, error) {
	def, err := DefinitionFor(Mode(req.Mode))
	if err != nil {
		return RoundEnvelope{}, err
	}
	if !def.RunsModeRounds() {
		return RoundEnvelope{}, fmt.Errorf("mode %q runs no operating-mode rounds", def.Mode)
	}
	if def.Target.Kind == TargetInitiative {
		return RoundEnvelope{}, fmt.Errorf("mode %q targets an initiative; start its phases through the initiative surface (initiative name + phase), not the target surface", def.Mode)
	}
	phaseName := Phase(strings.ToLower(strings.TrimSpace(req.Phase)))
	if phaseName == "" {
		phaseName = def.PhaseGraph.StartPhase
	}
	phaseDef, err := def.PhaseDefinition(phaseName)
	if err != nil {
		return RoundEnvelope{}, err
	}
	rc, err := s.collectRunContext(ctx, def, phaseDef, req.TargetRef)
	if err != nil {
		return RoundEnvelope{}, err
	}
	return s.startResolvedPhase(ctx, rc, req.Note, req.Override, req.RequestedBy)
}

// startResolvedPhase is the shared dispatch path behind every phase-start
// surface: it validates startability, reserves the round, acquires the
// target's exclusive lock, renders the prompt, and spawns the agent. When the
// phase is delegated (executed_by), the execution surface — prompt, reads,
// profile, purposes, declared output — comes from the sub-mode's next
// sub-phase, while persistence, locking, and routing stay keyed to the parent
// run context.
func (s *Service) startResolvedPhase(ctx context.Context, rc RunContext, note string, override bool, requestedBy string) (RoundEnvelope, error) {
	def, phaseDef := rc.Def, rc.PhaseDef
	ownershipKey, err := rc.OwnershipKey()
	if err != nil {
		return RoundEnvelope{}, err
	}
	if err := ValidatePhaseStart(def, rc.Rounds, phaseDef.Phase, rc.Target.Initiative.AcceptanceCriteria); err != nil {
		return RoundEnvelope{}, err
	}
	var execution OperatingModeExecution
	if rc.Execution != nil {
		execution = *rc.Execution
	} else {
		execution, err = s.store.ContinueOrCreateExecution(rc.Target.ID, def)
		if err != nil {
			return RoundEnvelope{}, err
		}
	}

	// exec is the run context whose prompt/reads/profile the spawned agent
	// receives: the phase's own for a regular phase, the sub-mode's next
	// sub-phase for a delegated one.
	exec := rc
	payload := map[string]any{
		"operator_note": strings.TrimSpace(note),
	}
	if phaseDef.Delegated() {
		exec, err = collectDelegatedRunContext(rc)
		if err != nil {
			return RoundEnvelope{}, err
		}
		payload[payloadDelegatedMode] = string(exec.Def.Mode)
		payload[payloadDelegatedPhase] = string(exec.PhaseDef.Phase)
	}
	payload["skill_id"] = exec.PhaseDef.SkillID
	payload["catalog_id"] = exec.PhaseDef.CatalogID
	payload["plan_context"] = exec.Target.Plan

	round, err := s.store.CreateRound(RoundEnvelope{
		ExecutionID:      execution.ExecutionID,
		DefinitionDigest: execution.DefinitionDigest,
		Mode:             string(def.Mode),
		InitiativeName:   rc.Target.Initiative.Name,
		ScopeID:          rc.Target.ID,
		Phase:            string(phaseDef.Phase),
		AgentProfileKey:  exec.PhaseDef.ProfileKey,
		Status:           RoundStatusReserved,
		Items:            rc.Target.Items,
		Payload:          payload,
	})
	if err != nil {
		return RoundEnvelope{}, err
	}

	holder := initiativelock.Holder{
		RunID:       fmt.Sprintf("provisional-operating-mode-%d-%d", round.Round, s.clock().UTC().UnixNano()),
		Purpose:     exec.PhaseDef.LockPurpose,
		RoundNumber: round.Round,
		AcquiredBy:  strings.TrimSpace(requestedBy),
	}
	provisionalRunID := holder.RunID
	if override {
		s.preemptLockHolder(ctx, ownershipKey)
		if err := s.lock.AcquireOverride(ownershipKey, holder); err != nil {
			s.persistPhaseStartFailure(round, err, "lock_failed_at")
			return RoundEnvelope{}, fmt.Errorf("override lock: %w", err)
		}
	} else if err := s.lock.Acquire(ownershipKey, holder); err != nil {
		s.persistPhaseStartFailure(round, err, "lock_failed_at")
		return RoundEnvelope{}, err
	}

	prompt, err := s.buildPrompt(ctx, exec, round, note)
	if err != nil {
		_ = s.lock.Release(ownershipKey, provisionalRunID)
		round = s.persistPhaseStartFailure(round, err, "prompt_failed_at")
		slog.Warn("operating mode prompt rendering failed; phase start aborted",
			"err", err, "target", rc.Target.ID, "mode", def.Mode, "phase", phaseDef.Phase)
		return round, fmt.Errorf("render operating-mode prompt: %w", err)
	}

	round.Status = RoundStatusAgentRunning
	if err := s.store.SaveRound(round); err != nil {
		_ = s.lock.Release(ownershipKey, provisionalRunID)
		return RoundEnvelope{}, fmt.Errorf("save reserved round: %w", err)
	}

	spawnReq := agentmanager.InitiativeSpawnRequest{
		Name:        rc.Target.ID,
		Title:       fmt.Sprintf("%s: %s", def.Label, phaseDef.Phase),
		Description: strings.TrimSpace(rc.Target.Description),
		Prompt:      prompt,
		ScopePath:   ".",
		ProjectRoot: s.scenarioRoot,
		CreatedBy:   s.requestedBy,
		Purpose:     exec.PhaseDef.ActivityPurpose,
		RoundNumber: round.Round,
		RoundSlug:   string(phaseDef.Phase),
		ProfileKey:  exec.PhaseDef.ProfileKey,
	}
	if s.activity != nil {
		// PhaseKind drives lane assignment in agentactivity. The activity
		// purpose is a mode-defined dynamic string (e.g.
		// "holistic_loop_investigate"), so without phaseKind LaneOf would
		// return an error and the spawn would be rejected.
		spec := agentactivity.Spec{
			OwnerType:   agentactivity.OwnerInitiative,
			OwnerName:   rc.Target.ID,
			OwnerTitle:  rc.Target.Title,
			Purpose:     agentactivity.Purpose(exec.PhaseDef.ActivityPurpose),
			PhaseKind:   string(exec.PhaseDef.Kind),
			RequestedBy: s.requestedBy,
			Metadata: map[string]string{
				"entrypoint":        "initiative.operating_mode.phase",
				"operating_mode":    string(def.Mode),
				"execution_id":      execution.ExecutionID,
				"target_kind":       string(def.Target.Kind),
				"phase":             string(phaseDef.Phase),
				"run_strategy":      string(def.RunStrategy.Kind),
				"round_number":      fmt.Sprintf("%d", round.Round),
				"artifact_set":      def.Artifact.Root,
				"agent_profile_key": exec.PhaseDef.ProfileKey,
			},
		}
		if phaseDef.Delegated() {
			spec.Metadata["executed_by"] = string(exec.Def.Mode)
			spec.Metadata["delegated_phase"] = string(exec.PhaseDef.Phase)
		}
		ctx = agentactivity.WithSpec(ctx, spec)
	}

	result, err := s.spawnInitiative(ctx, spawnReq)
	if err != nil {
		_ = s.lock.Release(ownershipKey, provisionalRunID)
		round = s.persistPhaseStartFailure(round, err, "spawn_failed_at")
		return round, fmt.Errorf("spawn operating-mode phase: %w", err)
	}

	round.RunID = strings.TrimSpace(result.RunID)
	round.GeneratedAt = s.clock().UTC().Format(time.RFC3339)
	if err := s.store.SaveRound(round); err != nil {
		slog.Warn("operating mode: persist run id failed", "err", err, "target", rc.Target.ID, "run_id", result.RunID)
	}
	if err := s.store.IndexRunOwner(execution, round.RunID, round.Round); err != nil {
		_ = s.lock.Release(ownershipKey, provisionalRunID)
		if s.agent != nil {
			_ = s.agent.StopRun(ctx, round.RunID)
		}
		round = s.persistPhaseStartFailure(round, err, "run_owner_index_failed_at")
		return round, fmt.Errorf("index operating-mode run owner: %w", err)
	}
	holder.RunID = round.RunID
	if err := s.lock.AcquireOverride(ownershipKey, holder); err != nil {
		slog.Warn("operating mode: lock run-id swap failed", "err", err, "target", rc.Target.ID, "run_id", round.RunID)
		_ = s.lock.Release(ownershipKey, provisionalRunID)
		if s.agent != nil {
			if stopErr := s.agent.StopRun(ctx, round.RunID); stopErr != nil {
				slog.Warn("operating mode: stop run after lock swap failure failed", "err", stopErr, "target", rc.Target.ID, "run_id", round.RunID)
			}
		}
		round = s.persistPhaseStartFailure(round, err, "lock_swap_failed_at")
		return round, fmt.Errorf("swap operating-mode lock holder: %w", err)
	}
	s.emitPhaseStarted(round)
	return round, nil
}

// RenderLivePrompt renders the literal agent prompt for one phase of a real
// linked initiative through the shared renderPhasePrompt seam — the same code
// path a live spawn uses, so the preview is byte-identical to what an agent
// receives. It performs no spawning, locking, or persistence. When the
// prompt-manager seam is unavailable it returns a typed degraded response
// (Degraded=true, Variables populated) rather than an error.
//
// roundNumber selects which round the prompt represents: prior rounds are
// filtered to those before it (matching the live trace's Reads projection), so
// the substituted PRIOR_ROUNDS_JSON is consistent with what the operator sees.
// A non-positive roundNumber renders the next round.
func (s *Service) RenderLivePrompt(ctx context.Context, initiativeName, phase string, roundNumber int, note string) (RenderPromptResponse, error) {
	init, def, phaseDef, err := s.resolvePhase(initiativeName, phase)
	if err != nil {
		return RenderPromptResponse{}, err
	}
	if def.Mode == ModeItemLevel {
		return RenderPromptResponse{}, fmt.Errorf("item-level mode has no operating-mode phase prompt to render")
	}
	rc, err := s.collectRunContext(ctx, def, phaseDef, init.Name)
	if err != nil {
		return RenderPromptResponse{}, err
	}
	effectiveRound := roundNumber
	if effectiveRound <= 0 {
		effectiveRound = len(rc.Rounds) + 1
	}
	priorRounds := make([]RoundEnvelope, 0, len(rc.Rounds))
	for _, r := range rc.Rounds {
		if r.Round < effectiveRound {
			priorRounds = append(priorRounds, r)
		}
	}
	rc.Rounds = priorRounds
	// A delegated phase renders the sub-mode's next sub-phase prompt — the
	// same execution context a live start would spawn.
	exec := rc
	if phaseDef.Delegated() {
		exec, err = collectDelegatedRunContext(rc)
		if err != nil {
			return RenderPromptResponse{}, err
		}
	}
	round := RoundEnvelope{
		Round:           effectiveRound,
		Mode:            string(def.Mode),
		InitiativeName:  rc.Target.Initiative.Name,
		ScopeID:         rc.Target.ID,
		Phase:           string(phaseDef.Phase),
		AgentProfileKey: exec.PhaseDef.ProfileKey,
	}
	resp := RenderPromptResponse{
		Mode:       string(def.Mode),
		Phase:      string(phaseDef.Phase),
		SkillID:    exec.PhaseDef.SkillID,
		ProfileKey: exec.PhaseDef.ProfileKey,
		Variables:  promptVariables(exec, round, note),
	}
	rendered, err := s.renderPhasePrompt(ctx, exec, round, note)
	if err != nil {
		if errors.Is(err, ErrPromptRenderUnavailable) {
			resp.Degraded = true
			resp.DegradedReason = err.Error()
			return resp, nil
		}
		return RenderPromptResponse{}, err
	}
	resp.Prompt = rendered.Prompt
	resp.Variables = rendered.Variables
	return resp, nil
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
		slog.Warn("operating mode: persist phase start failure failed", "err", saveErr, "scope", round.ScopeID, "round", round.Round)
	}
	if syncErr := s.syncExecutionStatus(round); syncErr != nil {
		slog.Warn("operating mode: persist execution failure status failed", "err", syncErr, "scope", round.ScopeID, "execution_id", round.ExecutionID)
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
	// Initiative-keyed phase actions only run initiative-target modes. A
	// plan-target mode (e.g. the generic phased-plan-drain) operates on a plan
	// directly — its rounds cannot be driven through an initiative's mode
	// field, and an initiative that still carries one names it explicitly.
	if def.Target.Kind != TargetInitiative {
		return InitiativeSnapshot{}, Definition{}, PhaseDefinition{}, fmt.Errorf("initiative %q declares mode %q, which targets %s (not an initiative); switch the initiative to an initiative-target mode", init.Name, def.Mode, def.Target.Kind)
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

// collectRunContext resolves the mode's target instance through its adapter
// and assembles the generic run context (target + durable rounds + declared
// artifacts). It contains no target-kind branching: all target-specific work
// lives behind the adapter.
func (s *Service) collectRunContext(ctx context.Context, def Definition, phaseDef PhaseDefinition, targetRef string) (RunContext, error) {
	adapter, err := AdapterFor(def.Target.Kind)
	if err != nil {
		return RunContext{}, err
	}
	target, err := adapter.Resolve(ctx, s, def, phaseDef, targetRef)
	if err != nil {
		return RunContext{}, err
	}
	_, legacyAmbiguous, err := s.store.AdoptLegacyExecution(target.ID, def)
	if err != nil {
		return RunContext{}, err
	}
	executions, err := s.store.ListExecutions(target.ID, def.Mode)
	if err != nil {
		return RunContext{}, err
	}
	var resumable []OperatingModeExecution
	for _, execution := range executions {
		if !execution.Terminal() {
			resumable = append(resumable, execution)
		}
	}
	if len(resumable) > 1 {
		return RunContext{}, fmt.Errorf("%w: mode %q scope %q has %d resumable manifests", ErrExecutionAmbiguous, def.Mode, target.ID, len(resumable))
	}
	var execution *OperatingModeExecution
	if len(resumable) == 1 {
		pinned := resumable[0]
		pinnedDef, err := pinned.DefinitionBundle.RootDefinition()
		if err != nil {
			return RunContext{}, err
		}
		pinnedPhase, err := pinnedDef.PhaseDefinition(phaseDef.Phase)
		if err != nil {
			return RunContext{}, err
		}
		def = pinnedDef
		phaseDef = pinnedPhase
		execution = &pinned
		pinnedAdapter, err := AdapterFor(def.Target.Kind)
		if err != nil {
			return RunContext{}, err
		}
		pinnedTarget, err := pinnedAdapter.Resolve(ctx, s, def, phaseDef, targetRef)
		if err != nil {
			return RunContext{}, err
		}
		if pinnedTarget.ID != target.ID {
			return RunContext{}, fmt.Errorf("pinned target %q differs from live target %q for execution %q", pinnedTarget.ID, target.ID, pinned.ExecutionID)
		}
		target = pinnedTarget
	}
	rounds, err := s.store.ListRounds(target.ID, def.Mode)
	if err != nil {
		return RunContext{}, err
	}
	if execution != nil {
		filtered := rounds[:0]
		for _, round := range rounds {
			if round.ExecutionID == execution.ExecutionID {
				filtered = append(filtered, round)
			}
		}
		rounds = filtered
	} else if len(executions) > 0 || legacyAmbiguous {
		// Terminal manifests and ambiguous legacy histories are compatibility
		// projections only. A new execution never inherits either history.
		rounds = nil
	}
	artifacts, err := s.store.ListDeclaredArtifactsForDefinition(target.ID, def)
	if err != nil {
		return RunContext{}, err
	}
	return RunContext{
		Def:       def,
		PhaseDef:  phaseDef,
		Execution: execution,
		Target:    target,
		Artifacts: artifacts,
		Rounds:    rounds,
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

func (s *Service) preemptLockHolder(ctx context.Context, ownershipKey string) {
	holder, err := s.lock.Inspect(ownershipKey)
	if err != nil || holder == nil || holder.RunID == "" || s.agent == nil {
		return
	}
	if err := s.agent.StopRun(ctx, holder.RunID); err != nil {
		slog.Warn("operating mode: failed to stop preempted lock holder", "err", err, "ownership_key", ownershipKey, "run_id", holder.RunID)
	}
}
