package agentsessions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/idgen"
)

const (
	SkillMetaOrchestrator       = "swarm-manager-meta-orchestrator"
	SkillOperatingModeAuthoring = "swarm-manager-operating-mode-authoring"

	EnvSessionID   = "VROOLI_SWARM_MANAGER_SESSION_ID"
	EnvSessionKind = "VROOLI_SWARM_MANAGER_SESSION_KIND"
	EnvSpawnSource = "VROOLI_SPAWN_SOURCE"
)

var nowUTC = func() time.Time { return time.Now().UTC() }

type EventLogger interface {
	EmitAgentSessionCreated(sessionID string, payload any)
	EmitAgentSessionStarted(sessionID string, payload any)
	EmitAgentSessionContinued(sessionID string, payload any)
	EmitAgentSessionFailed(sessionID string, payload any)
	EmitAgentSessionCanceled(sessionID string, payload any)
	EmitAgentSessionCompleted(sessionID string, payload any)
	EmitAgentSessionArtifactLinked(sessionID string, payload any)
}

type SessionSpawner interface {
	SpawnSession(ctx context.Context, req agentmanager.SessionSpawnRequest) (agentmanager.RunResult, error)
	ContinueRun(ctx context.Context, runID string, message string) error
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
	StopRun(ctx context.Context, runID string) error
}

type BacklogBatchApplier interface {
	ApplyAgentSessionBacklogBatchImport(ctx context.Context, payloadJSON string, prov identity.Provenance) ([]Artifact, error)
}

type Service struct {
	store               Store
	spawner             SessionSpawner
	backlogBatchApplier BacklogBatchApplier
	eventLogger         EventLogger
	projectRoot         string
	profileKey          string
}

type ServiceConfig struct {
	Store               Store
	Spawner             SessionSpawner
	BacklogBatchApplier BacklogBatchApplier
	EventLogger         EventLogger
	ProjectRoot         string
	ProfileKey          string
}

type CreateRequest struct {
	Kind           Kind
	Title          string
	InitialMessage string
	Initiative     string
}

type ContinueRequest struct {
	SessionID     string
	Message       string
	AttachmentIDs []string
}

func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Store == nil {
		return nil, errors.New("agentsessions.NewService: Store is required")
	}
	if cfg.ProfileKey == "" {
		cfg.ProfileKey = "swarm-manager/default"
	}
	if cfg.ProjectRoot == "" {
		cfg.ProjectRoot = "."
	}
	return &Service{
		store:               cfg.Store,
		spawner:             cfg.Spawner,
		backlogBatchApplier: cfg.BacklogBatchApplier,
		eventLogger:         cfg.EventLogger,
		projectRoot:         cfg.ProjectRoot,
		profileKey:          cfg.ProfileKey,
	}, nil
}

func (s *Service) SetBacklogBatchApplier(applier BacklogBatchApplier) {
	s.backlogBatchApplier = applier
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Session, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.InitialMessage = strings.TrimSpace(req.InitialMessage)
	if !IsKnownKind(req.Kind) {
		return Session{}, apierr.BadRequest("session kind is invalid")
	}
	if req.Title == "" {
		return Session{}, apierr.BadRequest("title is required")
	}
	if req.InitialMessage == "" {
		return Session{}, apierr.BadRequest("initial_message is required")
	}
	if s.spawner == nil {
		return Session{}, apierr.Unavailable("agent session spawning is unavailable")
	}

	now := nowRFC3339()
	session := Session{
		ID:         "sess_" + idgen.Generate(),
		Title:      req.Title,
		Kind:       req.Kind,
		Status:     StatusStarting,
		SkillID:    skillIDForKind(req.Kind),
		ProfileKey: s.profileKey,
		CreatedAt:  now,
		UpdatedAt:  now,
		CreatedBy:  attributionForContext(ctx),
	}
	if err := s.store.CreateSession(session); err != nil {
		return Session{}, err
	}

	userMessage := Message{
		ID:        "msg_" + idgen.Generate(),
		Role:      MessageRoleUser,
		Content:   req.InitialMessage,
		CreatedAt: now,
	}
	if err := s.store.AppendMessage(session.ID, userMessage); err != nil {
		return Session{}, err
	}
	s.emitCreated(session)

	spawnReq := agentmanager.SessionSpawnRequest{
		SessionID:   session.ID,
		Kind:        string(session.Kind),
		Title:       session.Title,
		Description: req.InitialMessage,
		Prompt:      buildInitialPrompt(session, req.InitialMessage, req.Initiative),
		ScopePath:   ".",
		ProjectRoot: s.projectRoot,
		CreatedBy:   sessionCreatedBy(session),
		Environment: sessionEnvironment(session),
		ProfileKey:  s.profileKey,
	}
	activityCtx := agentactivity.WithSpec(ctx, sessionActivitySpec(session, agentactivity.InteractionSpawn))
	run, err := s.spawner.SpawnSession(activityCtx, spawnReq)
	if err != nil {
		failed := session
		failed.Status = StatusFailed
		failed.FailureReason = err.Error()
		failed.UpdatedAt = nowRFC3339()
		_ = s.store.SaveSession(failed)
		s.emitFailed(failed)
		return Session{}, mapSpawnError(err)
	}

	session.TaskID = strings.TrimSpace(run.TaskID)
	session.RunID = strings.TrimSpace(run.RunID)
	session.Status = StatusRunning
	session.UpdatedAt = nowRFC3339()
	if err := s.store.SaveSession(session); err != nil {
		return Session{}, err
	}
	s.emitStarted(session)
	return s.store.LoadSession(session.ID)
}

func (s *Service) List(_ context.Context, filters ListFilters) ([]Session, error) {
	return s.store.ListSessions(filters)
}

func (s *Service) ResolveSessionForRun(ctx context.Context, runID string) (identity.SessionReference, bool, error) {
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return identity.SessionReference{}, false, nil
	}
	sessions, err := s.List(ctx, ListFilters{})
	if err != nil {
		return identity.SessionReference{}, false, err
	}
	for _, session := range sessions {
		if strings.TrimSpace(session.RunID) != trimmed {
			continue
		}
		return identity.SessionReference{
			SessionID:   session.ID,
			SessionKind: string(session.Kind),
			Source:      "session/" + session.ID,
		}, true, nil
	}
	return identity.SessionReference{}, false, nil
}

func (s *Service) Get(_ context.Context, sessionID string) (Session, error) {
	session, err := s.store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return Session{}, mapStoreError(err)
	}
	return session, nil
}

func (s *Service) Continue(ctx context.Context, req ContinueRequest) (Session, error) {
	messageText := strings.TrimSpace(req.Message)
	if messageText == "" {
		return Session{}, apierr.BadRequest("message is required")
	}
	session, err := s.store.LoadSession(strings.TrimSpace(req.SessionID))
	if err != nil {
		return Session{}, mapStoreError(err)
	}
	if strings.TrimSpace(session.RunID) == "" {
		return Session{}, apierr.Conflict("session has no active agent run")
	}
	if s.spawner == nil {
		return Session{}, apierr.Unavailable("agent session continuation is unavailable")
	}

	now := nowRFC3339()
	message := Message{
		ID:            "msg_" + idgen.Generate(),
		Role:          MessageRoleUser,
		Content:       messageText,
		AttachmentIDs: append([]string(nil), req.AttachmentIDs...),
		CreatedAt:     now,
	}
	if err := s.store.AppendMessage(session.ID, message); err != nil {
		return Session{}, err
	}
	session.Status = StatusRunning
	session.UpdatedAt = now
	if err := s.store.SaveSession(session); err != nil {
		return Session{}, err
	}

	activityCtx := agentactivity.WithSpec(ctx, sessionActivitySpec(session, agentactivity.InteractionContinue))
	if err := s.spawner.ContinueRun(activityCtx, session.RunID, messageText); err != nil {
		session.Status = StatusFailed
		session.FailureReason = err.Error()
		session.UpdatedAt = nowRFC3339()
		_ = s.store.SaveSession(session)
		s.emitFailed(session)
		return Session{}, mapSpawnError(err)
	}
	s.emitContinued(session)
	return s.store.LoadSession(session.ID)
}

func (s *Service) Refresh(ctx context.Context, sessionID string) (Session, error) {
	session, err := s.store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return Session{}, mapStoreError(err)
	}
	if strings.TrimSpace(session.RunID) == "" || s.spawner == nil {
		return session, nil
	}
	state, err := s.spawner.GetRunState(ctx, session.RunID)
	if err != nil {
		return Session{}, mapSpawnError(err)
	}
	changed := false
	now := nowRFC3339()
	next := sessionStatusFromRunState(state.Status)
	if next != "" && next != session.Status {
		session.Status = next
		session.UpdatedAt = now
		if state.ErrorMsg != "" {
			session.FailureReason = state.ErrorMsg
		}
		changed = true
	}
	if changed {
		if err := s.store.SaveSession(session); err != nil {
			return Session{}, err
		}
	}
	if shouldAppendAssistantSummary(session, state.Summary) {
		if err := s.store.AppendMessage(session.ID, Message{
			ID:        "msg_" + idgen.Generate(),
			Role:      MessageRoleAssistant,
			Content:   strings.TrimSpace(state.Summary),
			CreatedAt: now,
		}); err != nil {
			return Session{}, err
		}
	}
	if changed {
		if next == StatusFailed {
			s.emitFailed(session)
		}
		if next == StatusComplete {
			s.emitCompleted(session)
		}
	}
	return s.store.LoadSession(session.ID)
}

func shouldAppendAssistantSummary(session Session, summary string) bool {
	trimmed := strings.TrimSpace(summary)
	if trimmed == "" {
		return false
	}
	for i := len(session.Messages) - 1; i >= 0; i-- {
		message := session.Messages[i]
		if message.Role == MessageRoleAssistant && strings.TrimSpace(message.Content) == trimmed {
			return false
		}
	}
	return true
}

func (s *Service) Cancel(ctx context.Context, sessionID string) (Session, error) {
	session, err := s.store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return Session{}, mapStoreError(err)
	}
	if strings.TrimSpace(session.RunID) != "" && s.spawner != nil && isActiveSessionStatus(session.Status) {
		if err := s.spawner.StopRun(ctx, session.RunID); err != nil {
			return Session{}, mapSpawnError(err)
		}
	}
	session.Status = StatusCanceled
	session.UpdatedAt = nowRFC3339()
	if err := s.store.SaveSession(session); err != nil {
		return Session{}, err
	}
	s.emitCanceled(session)
	return s.store.LoadSession(session.ID)
}

func (s *Service) AttachArtifact(_ context.Context, artifact Artifact) (Artifact, error) {
	if artifact.ID == "" {
		artifact.ID = "art_" + idgen.Generate()
	}
	if artifact.CreatedAt == "" {
		artifact.CreatedAt = nowRFC3339()
	}
	if err := s.store.AppendArtifact(artifact.SessionID, artifact); err != nil {
		return Artifact{}, err
	}
	s.emitArtifactLinked(artifact)
	return artifact, nil
}

func (s *Service) RecordProposal(_ context.Context, sessionID string, proposal Proposal) (Proposal, error) {
	if proposal.ID == "" {
		proposal.ID = "prop_" + idgen.Generate()
	}
	now := nowRFC3339()
	if proposal.CreatedAt == "" {
		proposal.CreatedAt = now
	}
	proposal.UpdatedAt = now
	if err := s.store.SaveProposal(sessionID, proposal); err != nil {
		return Proposal{}, err
	}
	return proposal, nil
}

func (s *Service) ListArtifacts(_ context.Context, sessionID string) ([]Artifact, error) {
	artifacts, err := s.store.ListArtifacts(strings.TrimSpace(sessionID))
	if err != nil {
		return nil, mapStoreError(err)
	}
	return artifacts, nil
}

func (s *Service) ListArtifactsByEntity(_ context.Context, artifactType ArtifactType, entityRef string) ([]Artifact, error) {
	if !IsKnownArtifactType(artifactType) {
		return nil, apierr.BadRequest("artifact_type is invalid")
	}
	if strings.TrimSpace(entityRef) == "" {
		return nil, apierr.BadRequest("entity_ref is required")
	}
	return s.store.ListArtifactsByEntity(artifactType, entityRef)
}

func (s *Service) ApplyProposal(ctx context.Context, sessionID, proposalID string) (Session, []Artifact, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(proposalID) == "" {
		return Session{}, nil, apierr.BadRequest("session_id and proposal_id are required")
	}
	session, err := s.store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return Session{}, nil, mapStoreError(err)
	}
	proposal, ok := findProposal(session, strings.TrimSpace(proposalID))
	if !ok {
		return Session{}, nil, apierr.NotFound("agent session proposal not found")
	}
	if proposal.Status != ProposalStatusReady {
		return Session{}, nil, apierr.Conflict("agent session proposal must be ready before apply")
	}
	if strings.TrimSpace(session.RunID) == "" {
		return Session{}, nil, apierr.Conflict("agent session proposal apply requires an attributed agent run")
	}
	switch proposal.Kind {
	case ProposalBacklogBatchImport:
		if s.backlogBatchApplier == nil {
			return Session{}, nil, apierr.Unavailable("backlog batch proposal apply is unavailable")
		}
	default:
		return Session{}, nil, apierr.Wrapf(apierr.ErrNotImplemented, http.StatusNotImplemented, "agent session proposal kind %q apply is not implemented yet", string(proposal.Kind))
	}

	session.Status = StatusApplying
	session.UpdatedAt = nowRFC3339()
	if err := s.store.SaveSession(session); err != nil {
		return Session{}, nil, err
	}

	var artifacts []Artifact
	switch proposal.Kind {
	case ProposalBacklogBatchImport:
		prov := proposalApplyProvenance(session, proposal)
		artifacts, err = s.backlogBatchApplier.ApplyAgentSessionBacklogBatchImport(identity.NewContext(ctx, prov), proposal.PayloadJSON, prov)
		if err != nil {
			proposal.Status = ProposalStatusFailed
			proposal.UpdatedAt = nowRFC3339()
			session.Status = StatusProposalReady
			session.FailureReason = err.Error()
			session.UpdatedAt = proposal.UpdatedAt
			_ = s.store.SaveProposal(session.ID, proposal)
			_ = s.store.SaveSession(session)
			return Session{}, nil, err
		}
	}

	proposal.Status = ProposalStatusApplied
	proposal.UpdatedAt = nowRFC3339()
	session.Status = StatusWaitingForUser
	session.FailureReason = ""
	session.UpdatedAt = proposal.UpdatedAt
	if err := s.store.SaveProposal(session.ID, proposal); err != nil {
		return Session{}, nil, err
	}
	if err := s.store.SaveSession(session); err != nil {
		return Session{}, nil, err
	}
	applied, err := s.store.LoadSession(session.ID)
	if err != nil {
		return Session{}, nil, err
	}
	return applied, artifacts, nil
}

func findProposal(session Session, proposalID string) (Proposal, bool) {
	for _, proposal := range session.Proposals {
		if strings.TrimSpace(proposal.ID) == proposalID {
			return proposal, true
		}
	}
	return Proposal{}, false
}

func proposalApplyProvenance(session Session, proposal Proposal) identity.Provenance {
	if proposal.Attribution != nil && proposal.Attribution.Type == AttributionAgent {
		return identity.Provenance{
			Type:        identity.TypeAgent,
			RunID:       proposal.Attribution.RunID,
			TaskID:      proposal.Attribution.TaskID,
			ProfileKey:  proposal.Attribution.ProfileKey,
			SessionID:   session.ID,
			SessionKind: string(session.Kind),
			Source:      "session/" + session.ID,
		}
	}
	return identity.Provenance{
		Type:        identity.TypeAgent,
		RunID:       session.RunID,
		TaskID:      session.TaskID,
		ProfileKey:  session.ProfileKey,
		SessionID:   session.ID,
		SessionKind: string(session.Kind),
		Source:      "session/" + session.ID,
	}
}

func skillIDForKind(kind Kind) string {
	switch kind {
	case KindMetaOrchestration:
		return SkillMetaOrchestrator
	case KindOperatingModeAuthoring:
		return SkillOperatingModeAuthoring
	default:
		return ""
	}
}

func sessionEnvironment(session Session) map[string]string {
	return map[string]string{
		EnvSessionID:   session.ID,
		EnvSessionKind: string(session.Kind),
		EnvSpawnSource: "session/" + session.ID,
	}
}

func buildInitialPrompt(session Session, initialMessage, initiative string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are running a Swarm Manager %s agent session.\n\n", session.Kind)
	fmt.Fprintf(&b, "Use the Prompt Manager skill `%s` as your operating guide.\n", session.SkillID)
	fmt.Fprintf(&b, "Session ID: %s\n", session.ID)
	if trimmed := strings.TrimSpace(initiative); trimmed != "" {
		fmt.Fprintf(&b, "Related initiative: %s\n", trimmed)
	}
	b.WriteString("\nOperator message:\n")
	b.WriteString(strings.TrimSpace(initialMessage))
	return b.String()
}

func sessionActivitySpec(session Session, interaction agentactivity.InteractionType) agentactivity.Spec {
	purpose := agentactivity.Purpose(session.Kind)
	return agentactivity.Spec{
		OwnerType:  agentactivity.OwnerSession,
		OwnerKind:  string(session.Kind),
		OwnerName:  session.ID,
		OwnerTitle: session.Title,
		Purpose:    purpose,
		Metadata: map[string]string{
			"entrypoint":       "agent_sessions." + string(session.Kind),
			"session_id":       session.ID,
			"skill_id":         session.SkillID,
			"interaction_type": string(interaction),
			"session_kind":     string(session.Kind),
			"swarm_source":     "session/" + session.ID,
		},
	}
}

func attributionForContext(ctx context.Context) *Attribution {
	attr := AttributionFromProvenance(identity.FromContext(ctx))
	if attr.Type == "" {
		attr.Type = AttributionOperator
	}
	return &attr
}

func sessionCreatedBy(session Session) string {
	if session.CreatedBy == nil {
		return "swarm-manager"
	}
	if session.CreatedBy.Type == AttributionAgent && session.CreatedBy.RunID != "" {
		return "agent:" + session.CreatedBy.RunID
	}
	return string(session.CreatedBy.Type)
}

func sessionStatusFromRunState(status string) Status {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "starting", "running":
		return StatusRunning
	case "needs_review":
		return StatusWaitingForUser
	case "complete", "completed":
		return StatusComplete
	case "failed":
		return StatusFailed
	case "cancelled", "canceled":
		return StatusCanceled
	default:
		return ""
	}
}

func mapStoreError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apierr.NotFound("agent session not found")
	}
	if errors.Is(err, ErrValidation) {
		return apierr.BadRequest("%s", err.Error())
	}
	return err
}

func mapSpawnError(err error) error {
	if errors.Is(err, agentmanager.ErrNotAvailable) {
		return apierr.Unavailable("agent-manager is unavailable")
	}
	if errors.Is(err, agentmanager.ErrRequestFailed) {
		return apierr.Wrap(apierr.ErrBadGateway, http.StatusBadGateway, err.Error())
	}
	return err
}

func nowRFC3339() string {
	return nowUTC().Format(time.RFC3339)
}

func (s *Service) emitCreated(session Session) {
	if s.eventLogger != nil {
		s.eventLogger.EmitAgentSessionCreated(session.ID, eventPayload(session))
	}
}

func (s *Service) emitStarted(session Session) {
	if s.eventLogger != nil {
		s.eventLogger.EmitAgentSessionStarted(session.ID, eventPayload(session))
	}
}

func (s *Service) emitContinued(session Session) {
	if s.eventLogger != nil {
		s.eventLogger.EmitAgentSessionContinued(session.ID, eventPayload(session))
	}
}

func (s *Service) emitFailed(session Session) {
	if s.eventLogger != nil {
		s.eventLogger.EmitAgentSessionFailed(session.ID, eventPayload(session))
	}
}

func (s *Service) emitCanceled(session Session) {
	if s.eventLogger != nil {
		s.eventLogger.EmitAgentSessionCanceled(session.ID, eventPayload(session))
	}
}

func (s *Service) emitCompleted(session Session) {
	if s.eventLogger != nil {
		s.eventLogger.EmitAgentSessionCompleted(session.ID, eventPayload(session))
	}
}

func (s *Service) emitArtifactLinked(artifact Artifact) {
	if s.eventLogger != nil {
		s.eventLogger.EmitAgentSessionArtifactLinked(artifact.SessionID, AgentSessionArtifactEventPayload{
			SessionID:      artifact.SessionID,
			SessionKind:    "",
			ArtifactType:   string(artifact.ArtifactType),
			Action:         string(artifact.Action),
			EntityRef:      artifact.EntityRef,
			ProposalID:     artifact.ProposalID,
			RunID:          artifact.RunID,
			MutationSource: artifact.MutationSource,
		})
	}
}

type AgentSessionEventPayload struct {
	SessionID     string `json:"session_id"`
	SessionKind   string `json:"session_kind"`
	Status        string `json:"status"`
	SkillID       string `json:"skill_id,omitempty"`
	RunID         string `json:"run_id,omitempty"`
	TaskID        string `json:"task_id,omitempty"`
	ProfileKey    string `json:"profile_key,omitempty"`
	FailureReason string `json:"failure_reason,omitempty"`
}

type AgentSessionArtifactEventPayload struct {
	SessionID      string `json:"session_id"`
	SessionKind    string `json:"session_kind,omitempty"`
	ArtifactType   string `json:"artifact_type"`
	Action         string `json:"action"`
	EntityRef      string `json:"entity_ref"`
	ProposalID     string `json:"proposal_id,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	MutationSource string `json:"mutation_source,omitempty"`
}

func eventPayload(session Session) AgentSessionEventPayload {
	return AgentSessionEventPayload{
		SessionID:     session.ID,
		SessionKind:   string(session.Kind),
		Status:        string(session.Status),
		SkillID:       session.SkillID,
		RunID:         session.RunID,
		TaskID:        session.TaskID,
		ProfileKey:    session.ProfileKey,
		FailureReason: session.FailureReason,
	}
}
