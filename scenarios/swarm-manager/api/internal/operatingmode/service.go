package operatingmode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/promptmanager"
)

type InitiativeSnapshot struct {
	Name               string
	Title              string
	Description        string
	Mode               string
	Items              []string
	AcceptanceCriteria []string
}

type InitiativeReader interface {
	LoadInitiative(name string) (InitiativeSnapshot, error)
}

type InitiativeModeUpdater interface {
	UpdateInitiativeMode(name, mode string) (InitiativeSnapshot, error)
}

type BacklogItemSnapshot struct {
	Title    string
	Status   string
	Priority int
	Effort   string
}

type BacklogReader interface {
	LoadBacklogItem(kind, name string) (BacklogItemSnapshot, error)
}

type ActiveItemExecution struct {
	ItemRef     string `json:"item_ref"`
	ExecutionID string `json:"execution_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	Status      string `json:"status,omitempty"`
}

type ItemExecutionController interface {
	ActiveExecutionsForInitiative(ctx context.Context, initiative InitiativeSnapshot) ([]ActiveItemExecution, error)
	CancelActiveExecutionsForInitiative(ctx context.Context, initiative InitiativeSnapshot) ([]ActiveItemExecution, error)
}

type AgentSpawner interface {
	SpawnInitiative(ctx context.Context, req agentmanager.InitiativeSpawnRequest) (agentmanager.RunResult, error)
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
	StopRun(ctx context.Context, runID string) error
}

type Config struct {
	Store            *Store
	Lock             *initiativelock.Lock
	Initiatives      InitiativeReader
	ModeUpdater      InitiativeModeUpdater
	Backlog          BacklogReader
	ItemExecutions   ItemExecutionController
	Agent            AgentSpawner
	Activity         *agentactivity.Service
	PromptClient     promptmanager.Client
	Events           *eventlog.Emitter
	ScenarioRoot     string
	Clock            func() time.Time
	RequestedByLabel string
}

type Service struct {
	store        *Store
	lock         *initiativelock.Lock
	initiatives  InitiativeReader
	modeUpdater  InitiativeModeUpdater
	backlog      BacklogReader
	itemExecs    ItemExecutionController
	agent        AgentSpawner
	activity     *agentactivity.Service
	prompts      promptmanager.Client
	events       *eventlog.Emitter
	scenarioRoot string
	clock        func() time.Time
	requestedBy  string
}

type StartPhaseRequest struct {
	InitiativeName string
	Phase          string
	Note           string
	Override       bool
	RequestedBy    string
}

type SwitchModeRequest struct {
	InitiativeName             string
	Mode                       string
	CancelActiveItemExecutions bool
	RequestedBy                string
}

type SwitchModeResult struct {
	InitiativeName           string                `json:"initiative_name"`
	FromMode                 string                `json:"from_mode"`
	ToMode                   string                `json:"to_mode"`
	CanceledItemExecutions   []ActiveItemExecution `json:"canceled_item_executions,omitempty"`
	ActiveItemExecutions     []ActiveItemExecution `json:"active_item_executions,omitempty"`
	RequiresCancellation     bool                  `json:"requires_cancellation,omitempty"`
	OperatingModeWorkspaceID string                `json:"operating_mode_workspace_id,omitempty"`
}

type ActiveItemExecutionsConflict struct {
	InitiativeName string                `json:"initiative_name"`
	FromMode       string                `json:"from_mode"`
	ToMode         string                `json:"to_mode"`
	Executions     []ActiveItemExecution `json:"active_item_executions"`
}

func (e *ActiveItemExecutionsConflict) Error() string {
	return fmt.Sprintf("initiative %q has active item-level executions; confirm cancellation before switching to %q", e.InitiativeName, e.ToMode)
}

type Workspace struct {
	InitiativeName string                 `json:"initiative_name"`
	Mode           string                 `json:"mode"`
	Definition     WorkspaceMode          `json:"definition"`
	Lock           *initiativelock.Holder `json:"lock,omitempty"`
	Artifacts      []ArtifactSnapshot     `json:"artifacts"`
	Rounds         []RoundEnvelope        `json:"rounds"`
}

type WorkspaceMode struct {
	Mode        string              `json:"mode"`
	Label       string              `json:"label"`
	ScopeKind   string              `json:"scope_kind"`
	Phases      []WorkspacePhase    `json:"phases"`
	Terminal    []string            `json:"terminal"`
	Transitions map[string][]string `json:"transitions"`
	RunStrategy string              `json:"run_strategy"`
}

type WorkspacePhase struct {
	Phase            string               `json:"phase"`
	ActivityPurpose  string               `json:"activity_purpose"`
	ProfileKey       string               `json:"profile_key"`
	WritesRepo       bool                 `json:"writes_repo"`
	OutputArtifacts  []ArtifactDefinition `json:"output_artifacts,omitempty"`
	RequiresCriteria bool                 `json:"requires_criteria,omitempty"`
}

type phaseContext struct {
	init      InitiativeSnapshot
	def       Definition
	phaseDef  PhaseDefinition
	items     []RoundItem
	artifacts []ArtifactSnapshot
	rounds    []RoundEnvelope
}

func NewService(cfg Config) (*Service, error) {
	if cfg.Store == nil {
		return nil, errors.New("operatingmode: Store is required")
	}
	if cfg.Lock == nil {
		return nil, errors.New("operatingmode: Lock is required")
	}
	if cfg.Initiatives == nil {
		return nil, errors.New("operatingmode: InitiativeReader is required")
	}
	clk := cfg.Clock
	if clk == nil {
		clk = time.Now
	}
	requestedBy := strings.TrimSpace(cfg.RequestedByLabel)
	if requestedBy == "" {
		requestedBy = "swarm-manager"
	}
	return &Service{
		store:        cfg.Store,
		lock:         cfg.Lock,
		initiatives:  cfg.Initiatives,
		modeUpdater:  cfg.ModeUpdater,
		backlog:      cfg.Backlog,
		itemExecs:    cfg.ItemExecutions,
		agent:        cfg.Agent,
		activity:     cfg.Activity,
		prompts:      cfg.PromptClient,
		events:       cfg.Events,
		scenarioRoot: strings.TrimSpace(cfg.ScenarioRoot),
		clock:        clk,
		requestedBy:  requestedBy,
	}, nil
}

func (s *Service) SwitchMode(ctx context.Context, req SwitchModeRequest) (SwitchModeResult, error) {
	name := strings.TrimSpace(req.InitiativeName)
	if name == "" {
		return SwitchModeResult{}, fmt.Errorf("initiative name is required")
	}
	targetMode := NormalizeMode(req.Mode)
	if _, err := DefinitionFor(targetMode); err != nil {
		return SwitchModeResult{}, err
	}
	if s.modeUpdater == nil {
		return SwitchModeResult{}, errors.New("operatingmode: InitiativeModeUpdater is not configured")
	}
	init, err := s.initiatives.LoadInitiative(name)
	if err != nil {
		return SwitchModeResult{}, err
	}
	fromMode := NormalizeMode(init.Mode)
	result := SwitchModeResult{
		InitiativeName:           init.Name,
		FromMode:                 string(fromMode),
		ToMode:                   string(targetMode),
		OperatingModeWorkspaceID: string(targetMode),
	}
	if fromMode == targetMode {
		return result, nil
	}
	if fromMode == ModeItemLevel && targetMode != ModeItemLevel && s.itemExecs != nil {
		active, err := s.itemExecs.ActiveExecutionsForInitiative(ctx, init)
		if err != nil {
			return SwitchModeResult{}, err
		}
		if len(active) > 0 && !req.CancelActiveItemExecutions {
			return SwitchModeResult{}, &ActiveItemExecutionsConflict{
				InitiativeName: init.Name,
				FromMode:       string(fromMode),
				ToMode:         string(targetMode),
				Executions:     active,
			}
		}
		if len(active) > 0 {
			canceled, err := s.itemExecs.CancelActiveExecutionsForInitiative(ctx, init)
			if err != nil {
				return SwitchModeResult{}, err
			}
			result.CanceledItemExecutions = canceled
		}
	}
	updated, err := s.modeUpdater.UpdateInitiativeMode(init.Name, string(targetMode))
	if err != nil {
		return SwitchModeResult{}, err
	}
	result.ToMode = string(NormalizeMode(updated.Mode))
	return result, nil
}

func (s *Service) StartPhase(ctx context.Context, req StartPhaseRequest) (RoundEnvelope, error) {
	init, def, phaseDef, err := s.resolvePhase(req.InitiativeName, req.Phase)
	if err != nil {
		return RoundEnvelope{}, err
	}
	if def.Mode == ModeItemLevel {
		return RoundEnvelope{}, fmt.Errorf("item-level mode phases are owned by the existing backlog execution flow")
	}
	if phaseDef.RequiresCriteria && len(init.AcceptanceCriteria) == 0 {
		return RoundEnvelope{}, fmt.Errorf("phase %q requires initiative acceptance criteria", phaseDef.Phase)
	}

	ctxData, err := s.collectPhaseContext(init, def, phaseDef)
	if err != nil {
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
			return RoundEnvelope{}, fmt.Errorf("override lock: %w", err)
		}
	} else if err := s.lock.Acquire(init.Name, holder); err != nil {
		return RoundEnvelope{}, err
	}

	prompt, err := s.buildPrompt(ctx, ctxData, round, req.Note)
	if err != nil {
		slog.Warn("operating mode prompt rendering failed; using fallback prompt",
			"err", err, "initiative", init.Name, "mode", def.Mode, "phase", phaseDef.Phase)
		prompt = fallbackPrompt(ctxData, round, req.Note)
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
		spec := agentactivity.Spec{
			OwnerType:   agentactivity.OwnerInitiative,
			OwnerName:   init.Name,
			OwnerTitle:  init.Title,
			Purpose:     agentactivity.Purpose(phaseDef.ActivityPurpose),
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
		round.Status = RoundStatusFailed
		round.Error = err.Error()
		round.Payload["spawn_failed_at"] = s.clock().UTC().Format(time.RFC3339)
		_ = s.store.SaveRound(round)
		s.emitPhaseFailed(round, err.Error())
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
	}
	s.emitPhaseStarted(round)
	return round, nil
}

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
			Definition:     workspaceMode(def),
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
		Definition:     workspaceMode(def),
		Lock:           holder,
		Artifacts:      artifacts,
		Rounds:         rounds,
	}, nil
}

func (s *Service) RefreshRound(ctx context.Context, initiativeName string, mode Mode, number int) (RoundEnvelope, error) {
	round, err := s.store.LoadRound(initiativeName, mode, number)
	if err != nil {
		return RoundEnvelope{}, err
	}
	if !isRoundActive(round) || strings.TrimSpace(round.RunID) == "" || s.agent == nil {
		return round, nil
	}
	state, err := s.agent.GetRunState(ctx, round.RunID)
	if err != nil {
		return RoundEnvelope{}, err
	}
	switch strings.ToLower(strings.TrimSpace(state.Status)) {
	case "complete":
		round.Status = RoundStatusCompleted
		round.Payload = ensurePayload(round.Payload)
		round.Payload["agent_summary"] = strings.TrimSpace(state.Summary)
		round.Payload["finished_at"] = finishTime(state, s.clock)
		s.emitPhaseCompleted(round)
		_ = s.lock.Release(round.InitiativeName, round.RunID)
	case "failed":
		round.Status = RoundStatusFailed
		round.Error = strings.TrimSpace(state.ErrorMsg)
		if round.Error == "" {
			round.Error = "agent run failed"
		}
		round.Payload = ensurePayload(round.Payload)
		round.Payload["finished_at"] = finishTime(state, s.clock)
		s.emitPhaseFailed(round, round.Error)
		_ = s.lock.Release(round.InitiativeName, round.RunID)
	case "cancelled":
		round.Status = RoundStatusCanceled
		round.Payload = ensurePayload(round.Payload)
		round.Payload["finished_at"] = finishTime(state, s.clock)
		s.emitPhaseCanceled(round)
		_ = s.lock.Release(round.InitiativeName, round.RunID)
	default:
		return round, nil
	}
	if err := s.store.SaveRound(round); err != nil {
		return RoundEnvelope{}, err
	}
	return round, nil
}

func (s *Service) CancelRound(ctx context.Context, initiativeName string, mode Mode, number int) (RoundEnvelope, error) {
	round, err := s.store.LoadRound(initiativeName, mode, number)
	if err != nil {
		return RoundEnvelope{}, err
	}
	if strings.TrimSpace(round.RunID) != "" && s.agent != nil && isRoundActive(round) {
		if err := s.agent.StopRun(ctx, round.RunID); err != nil {
			return RoundEnvelope{}, err
		}
	}
	round.Status = RoundStatusCanceled
	round.Payload = ensurePayload(round.Payload)
	round.Payload["canceled_at"] = s.clock().UTC().Format(time.RFC3339)
	if err := s.store.SaveRound(round); err != nil {
		return RoundEnvelope{}, err
	}
	if round.RunID != "" {
		_ = s.lock.Release(initiativeName, round.RunID)
	}
	s.emitPhaseCanceled(round)
	return round, nil
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

func (s *Service) buildPrompt(ctx context.Context, data phaseContext, round RoundEnvelope, note string) (string, error) {
	if s.prompts == nil {
		return "", errors.New("prompt client not wired")
	}
	skillID := data.phaseDef.SkillID
	return s.prompts.ReadSkill(ctx, skillID, promptVariables(data, round, note), false)
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

func (s *Service) emitPhaseStarted(round RoundEnvelope) {
	if s.events != nil {
		s.events.EmitOperatingModePhaseStarted(round.ScopeID, phasePayload(round, "started", ""))
	}
}

func (s *Service) emitPhaseCompleted(round RoundEnvelope) {
	if s.events != nil {
		s.events.EmitOperatingModePhaseCompleted(round.ScopeID, phasePayload(round, "completed", ""))
	}
}

func (s *Service) emitPhaseFailed(round RoundEnvelope, reason string) {
	if s.events != nil {
		s.events.EmitOperatingModePhaseFailed(round.ScopeID, phasePayload(round, "failed", reason))
	}
}

func (s *Service) emitPhaseCanceled(round RoundEnvelope) {
	if s.events != nil {
		s.events.EmitOperatingModePhaseCanceled(round.ScopeID, phasePayload(round, "canceled", ""))
	}
}

func phasePayload(round RoundEnvelope, status, reason string) eventlog.OperatingModePhasePayload {
	payload := eventlog.OperatingModePhasePayload{
		Mode:            round.Mode,
		ScopeKind:       round.ScopeKind,
		ScopeID:         round.ScopeID,
		InitiativeName:  round.InitiativeName,
		Phase:           round.Phase,
		RunStrategy:     round.RunStrategy,
		AgentProfileKey: round.AgentProfileKey,
		RoundNumber:     round.Round,
		RunID:           round.RunID,
		Status:          status,
		ArtifactPaths:   artifactPaths(round.ArtifactUpdates),
	}
	if reason != "" {
		payload.Verdict = reason
	}
	payload.DurationSeconds = roundDuration(round)
	return payload
}

func artifactPaths(updates []ArtifactUpdate) []string {
	if len(updates) == 0 {
		return nil
	}
	paths := make([]string, 0, len(updates))
	for _, update := range updates {
		if strings.TrimSpace(update.Path) != "" {
			paths = append(paths, update.Path)
		}
	}
	return paths
}

func roundDuration(round RoundEnvelope) float64 {
	if round.GeneratedAt == "" || round.Payload == nil {
		return 0
	}
	finished, _ := round.Payload["finished_at"].(string)
	if finished == "" {
		return 0
	}
	start, err1 := time.Parse(time.RFC3339, round.GeneratedAt)
	end, err2 := time.Parse(time.RFC3339, finished)
	if err1 != nil || err2 != nil {
		return 0
	}
	return end.Sub(start).Seconds()
}

func finishTime(state agentmanager.RunState, clock func() time.Time) string {
	if strings.TrimSpace(state.FinishedAt) != "" {
		return strings.TrimSpace(state.FinishedAt)
	}
	return clock().UTC().Format(time.RFC3339)
}

func promptVariables(data phaseContext, round RoundEnvelope, note string) map[string]string {
	return map[string]string{
		"INITIATIVE_NAME":        data.init.Name,
		"INITIATIVE_TITLE":       data.init.Title,
		"INITIATIVE_DESCRIPTION": data.init.Description,
		"OPERATING_MODE":         string(data.def.Mode),
		"MODE_LABEL":             data.def.Label,
		"PHASE":                  string(data.phaseDef.Phase),
		"RUN_STRATEGY":           string(data.def.RunStrategy.Kind),
		"ROUND_NUMBER":           fmt.Sprintf("%d", round.Round),
		"AGENT_PROFILE_KEY":      data.phaseDef.ProfileKey,
		"ACCEPTANCE_CRITERIA":    strings.Join(data.init.AcceptanceCriteria, "\n"),
		"OPERATOR_NOTE":          strings.TrimSpace(note),
		"MEMBER_ITEMS_JSON":      mustJSON(data.items),
		"MODE_ARTIFACTS_JSON":    mustJSON(data.artifacts),
		"PRIOR_ROUNDS_JSON":      mustJSON(data.rounds),
	}
}

func fallbackPrompt(data phaseContext, round RoundEnvelope, note string) string {
	var b strings.Builder
	b.WriteString("Run the Swarm Manager operating-mode phase.\n\n")
	b.WriteString("Initiative: " + data.init.Name + "\n")
	b.WriteString("Title: " + data.init.Title + "\n")
	b.WriteString("Mode: " + string(data.def.Mode) + "\n")
	b.WriteString("Phase: " + string(data.phaseDef.Phase) + "\n")
	b.WriteString("Round: " + fmt.Sprintf("%d", round.Round) + "\n")
	b.WriteString("Profile key: " + data.phaseDef.ProfileKey + "\n\n")
	if strings.TrimSpace(note) != "" {
		b.WriteString("Operator note:\n" + strings.TrimSpace(note) + "\n\n")
	}
	if len(data.init.AcceptanceCriteria) > 0 {
		b.WriteString("Acceptance criteria:\n")
		for _, criterion := range data.init.AcceptanceCriteria {
			b.WriteString("- " + criterion + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Member items JSON:\n" + mustJSON(data.items) + "\n\n")
	b.WriteString("Mode artifacts JSON:\n" + mustJSON(data.artifacts) + "\n\n")
	b.WriteString("Prior rounds JSON:\n" + mustJSON(data.rounds) + "\n")
	return b.String()
}

func mustJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "null"
	}
	return string(data)
}

func workspaceMode(def Definition) WorkspaceMode {
	phases := make([]WorkspacePhase, 0, len(def.PhaseGraph.Phases))
	for _, phase := range def.PhaseGraph.Phases {
		phases = append(phases, WorkspacePhase{
			Phase:            string(phase.Phase),
			ActivityPurpose:  phase.ActivityPurpose,
			ProfileKey:       phase.ProfileKey,
			WritesRepo:       phase.WritesRepo,
			OutputArtifacts:  phase.OutputArtifacts,
			RequiresCriteria: phase.RequiresCriteria,
		})
	}
	sortWorkspacePhases(phases)
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
		Mode:        string(def.Mode),
		Label:       def.Label,
		ScopeKind:   string(def.Scope.Kind),
		Phases:      phases,
		Terminal:    terminal,
		Transitions: transitions,
		RunStrategy: string(def.RunStrategy.Kind),
	}
}

func sortWorkspacePhases(phases []WorkspacePhase) {
	order := map[string]int{
		"investigate":       1,
		"plan":              2,
		"execute":           3,
		"prepare_plan":      1,
		"execute_next":      2,
		"classify_progress": 3,
		"review":            4,
	}
	for i := 0; i < len(phases)-1; i++ {
		for j := i + 1; j < len(phases); j++ {
			if order[phases[j].Phase] < order[phases[i].Phase] {
				phases[i], phases[j] = phases[j], phases[i]
			}
		}
	}
}

func isRoundActive(round RoundEnvelope) bool {
	return round.Status == RoundStatusReserved || round.Status == RoundStatusAgentRunning
}

func ensurePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	return payload
}
