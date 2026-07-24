package agentsessions

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/idgen"

	agentdomainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

const (
	SkillMetaOrchestrator  = "swarm-manager-meta-orchestrator"
	SkillSwarmOperations   = "swarm-manager-operations-session"
	SkillWorkflowAuthoring = "swarm-manager-workflow-authoring"
	SkillProposals         = "swarm-manager-proposals"

	EnvSessionID   = "VROOLI_SWARM_MANAGER_SESSION_ID"
	EnvSessionKind = "VROOLI_SWARM_MANAGER_SESSION_KIND"
	EnvSpawnSource = "VROOLI_SPAWN_SOURCE"
)

var nowUTC = func() time.Time { return time.Now().UTC() }

type SessionSpawner interface {
	SpawnSession(ctx context.Context, req agentmanager.SessionSpawnRequest) (agentmanager.RunResult, error)
	ContinueRun(ctx context.Context, runID string, message string) error
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
	StopRun(ctx context.Context, runID string) error
}

type RunEventReader interface {
	GetRunEvents(ctx context.Context, runID string, opts agentmanager.RunEventsOptions) ([]*agentdomainpb.RunEvent, bool, error)
}

type BacklogBatchApplier interface {
	ApplyAgentSessionBacklogBatchImport(ctx context.Context, payloadJSON string, prov identity.Provenance) ([]Artifact, error)
}

type ContextResolver interface {
	ResolveSessionMessageContext(ctx context.Context, refs []ContextRef, limits ContextLimits) ([]ContextItem, error)
}

// MutationProposalProcessor keeps graph-aware extraction and ApplyFlow on the
// API composition side. Agent sessions own durable conversation and review
// state without importing proposal/backlog packages (which would form a cycle).
type MutationProposalProcessor interface {
	Ingest(ctx context.Context, target ProposalTarget, assistantReply string) (MutationProposalIngestion, error)
	Apply(ctx context.Context, target ProposalTarget, payloadJSON string, acceptedMutationIDs []string, source MutationProposalSource) (MutationProposalApplication, error)
	AcceptNoChange(ctx context.Context, target ProposalTarget, payloadJSON string, source MutationProposalSource) error
}

type MutationProposalIngestion struct {
	PayloadJSON      string
	ParseWarnings    []string
	ValidationErrors []string
}

type MutationProposalSource struct {
	SessionID  string
	RunID      string
	DecidedAt  string
	ProposalID string
}

type MutationProposalApplication struct{ Outcomes []MutationOutcome }

// MessageReferenceEnricher resolves typed entity references (`type:name` code
// spans) found in assistant message content into ContextItems whose NodeID the
// UI linkifies. Implemented by the session-context resolver; consumed at
// assistant-message append. Existence is the resolver's verdict — a reference
// to a record that does not exist is dropped, never attached.
type MessageReferenceEnricher interface {
	EnrichMessageReferences(ctx context.Context, content string) []ContextItem
}

type Service struct {
	store                     Store
	spawner                   SessionSpawner
	eventReader               RunEventReader
	backlogBatchApplier       BacklogBatchApplier
	contextResolver           ContextResolver
	eventLogger               EventLogger
	projectRoot               string
	profileKey                string
	mutationProposalProcessor MutationProposalProcessor
}

type ServiceConfig struct {
	Store                     Store
	Spawner                   SessionSpawner
	EventReader               RunEventReader
	BacklogBatchApplier       BacklogBatchApplier
	ContextResolver           ContextResolver
	EventLogger               EventLogger
	ProjectRoot               string
	ProfileKey                string
	MutationProposalProcessor MutationProposalProcessor
}

type CreateRequest struct {
	Kind           Kind
	Title          string
	ProposalTarget *ProposalTarget
}

type ContinueRequest struct {
	SessionID         string
	Message           string
	AttachmentIDs     []string
	ContextRefs       []ContextRef
	AutoContextPolicy AutoContextPolicy
}

type ContextLimits struct {
	Kind            Kind
	MaxTotal        int
	MaxPerType      map[ContextType]int
	MaxSummaryRunes int
}

type AttachmentUpload struct {
	Filename    string
	ContentType string
	SizeBytes   int64
	Reader      io.Reader
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
	if cfg.EventReader == nil {
		if reader, ok := cfg.Spawner.(RunEventReader); ok {
			cfg.EventReader = reader
		}
	}
	return &Service{
		store:                     cfg.Store,
		spawner:                   cfg.Spawner,
		eventReader:               cfg.EventReader,
		backlogBatchApplier:       cfg.BacklogBatchApplier,
		contextResolver:           cfg.ContextResolver,
		eventLogger:               cfg.EventLogger,
		projectRoot:               cfg.ProjectRoot,
		profileKey:                cfg.ProfileKey,
		mutationProposalProcessor: cfg.MutationProposalProcessor,
	}, nil
}

func (s *Service) SetBacklogBatchApplier(applier BacklogBatchApplier) {
	s.backlogBatchApplier = applier
}

func (s *Service) SetContextResolver(resolver ContextResolver) {
	s.contextResolver = resolver
}

func (s *Service) SetMutationProposalProcessor(processor MutationProposalProcessor) {
	s.mutationProposalProcessor = processor
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Session, error) {
	req.Title = strings.TrimSpace(req.Title)
	if !IsKnownKind(req.Kind) {
		return Session{}, apierr.BadRequest("session kind is invalid")
	}
	if req.Title == "" {
		return Session{}, apierr.BadRequest("title is required")
	}
	if req.ProposalTarget != nil {
		if err := req.ProposalTarget.Validate(); err != nil {
			return Session{}, apierr.BadRequest("invalid proposal target: %s", err)
		}
	}

	now := nowRFC3339()
	session := Session{
		ID:             "sess_" + idgen.Generate(),
		Title:          req.Title,
		Kind:           req.Kind,
		Status:         StatusDraft,
		SkillID:        skillIDForKind(req.Kind),
		ProfileKey:     s.profileKey,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      attributionForContext(ctx),
		ProposalTarget: req.ProposalTarget,
	}
	if req.ProposalTarget != nil {
		session.SkillID = SkillProposals
	}
	if err := s.store.CreateSession(session); err != nil {
		return Session{}, err
	}
	s.emitCreated(session)
	return s.store.LoadSession(session.ID)
}

func (s *Service) Start(ctx context.Context, req ContinueRequest) (Session, error) {
	messageText := strings.TrimSpace(req.Message)
	if messageText == "" && len(req.AttachmentIDs) == 0 && len(req.ContextRefs) == 0 {
		return Session{}, apierr.BadRequest("message, attachment, or context is required")
	}
	session, err := s.store.LoadSession(strings.TrimSpace(req.SessionID))
	if err != nil {
		return Session{}, mapStoreError(err)
	}
	if session.Status != StatusDraft || strings.TrimSpace(session.RunID) != "" {
		return Session{}, apierr.Conflict("agent session is already started")
	}
	if s.spawner == nil {
		return Session{}, apierr.Unavailable("agent session spawning is unavailable")
	}
	refs := refsWithAutoContext(session.Kind, req.ContextRefs, req.AutoContextPolicy, s.startupBriefResolverAvailable())
	if session.ProposalTarget != nil {
		refs = append(refs, ContextRef{Type: session.ProposalTarget.Type, Ref: session.ProposalTarget.Ref})
	}
	contextItems, err := s.resolveMessageContext(ctx, session, refs)
	if err != nil {
		return Session{}, err
	}

	now := nowRFC3339()
	userMessage := Message{
		ID:            "msg_" + idgen.Generate(),
		Role:          MessageRoleUser,
		Content:       messageText,
		CreatedAt:     now,
		AttachmentIDs: append([]string(nil), req.AttachmentIDs...),
		Context:       contextItems,
	}
	if err := s.store.AppendMessage(session.ID, userMessage); err != nil {
		return Session{}, err
	}
	session.Status = StatusStarting
	session.UpdatedAt = now
	if err := s.store.SaveSession(session); err != nil {
		return Session{}, err
	}

	spawnReq := agentmanager.SessionSpawnRequest{
		SessionID:   session.ID,
		Kind:        string(session.Kind),
		Title:       session.Title,
		Description: messageText,
		Prompt:      buildInitialPrompt(session, userMessage, sessionAttachmentsByID(session, req.AttachmentIDs)),
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
		if err := s.store.SaveSession(failed); err != nil {
			slog.Warn("agentsessions: persist session failed", "session", failed.ID, "err", err)
		}
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

func (s *Service) List(ctx context.Context, filters ListFilters) ([]Session, error) {
	sessions, err := s.store.ListSessions(filters)
	if err != nil {
		return nil, err
	}
	for index := range sessions {
		if err := s.hydrateArtifacts(ctx, &sessions[index]); err != nil {
			return nil, err
		}
	}
	return sessions, nil
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

func (s *Service) Get(ctx context.Context, sessionID string) (Session, error) {
	session, err := s.store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return Session{}, mapStoreError(err)
	}
	if err := s.hydrateArtifacts(ctx, &session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) hydrateArtifacts(ctx context.Context, session *Session) error {
	if s == nil || session == nil {
		return nil
	}
	artifacts, err := s.ListArtifacts(ctx, session.ID)
	if err != nil {
		return err
	}
	session.Artifacts = artifacts
	return nil
}

func (s *Service) Continue(ctx context.Context, req ContinueRequest) (Session, error) {
	messageText := strings.TrimSpace(req.Message)
	if messageText == "" && len(req.AttachmentIDs) == 0 && len(req.ContextRefs) == 0 {
		return Session{}, apierr.BadRequest("message, attachment, or context is required")
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
	contextItems, err := s.resolveMessageContext(ctx, session, req.ContextRefs)
	if err != nil {
		return Session{}, err
	}

	now := nowRFC3339()
	message := Message{
		ID:            "msg_" + idgen.Generate(),
		Role:          MessageRoleUser,
		Content:       messageText,
		AttachmentIDs: append([]string(nil), req.AttachmentIDs...),
		Context:       contextItems,
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
	if err := s.spawner.ContinueRun(activityCtx, session.RunID, buildContinuationPrompt(message, sessionAttachmentsByID(session, req.AttachmentIDs))); err != nil {
		session.Status = StatusFailed
		session.FailureReason = err.Error()
		session.UpdatedAt = nowRFC3339()
		if err := s.store.SaveSession(session); err != nil {
			slog.Warn("agentsessions: persist session failed", "session", session.ID, "err", err)
		}
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
		assistantMessage := Message{
			ID:        "msg_" + idgen.Generate(),
			Role:      MessageRoleAssistant,
			Content:   strings.TrimSpace(state.Summary),
			CreatedAt: now,
		}
		// Resolve typed entity references in the assistant output so the chat
		// UI can linkify them. Attaching survivors to Context is best-effort:
		// the resolver only returns references it confirmed exist, and an
		// absent enricher leaves Context empty.
		if enricher, ok := s.contextResolver.(MessageReferenceEnricher); ok {
			assistantMessage.Context = enricher.EnrichMessageReferences(ctx, assistantMessage.Content)
		}
		if err := s.store.AppendMessage(session.ID, assistantMessage); err != nil {
			return Session{}, err
		}
		if err := s.ingestMutationProposal(ctx, session, assistantMessage.Content); err != nil {
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
	if strings.TrimSpace(session.RunID) != "" && s.spawner != nil && hasStoppableRun(session.Status) {
		if err := s.spawner.StopRun(ctx, session.RunID); err != nil && !isTerminalRunStopConflict(err) {
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

func (s *Service) Delete(ctx context.Context, sessionID string) error {
	session, err := s.store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return mapStoreError(err)
	}
	if strings.TrimSpace(session.RunID) != "" && s.spawner != nil && hasStoppableRun(session.Status) {
		if err := s.spawner.StopRun(ctx, session.RunID); err != nil && !isTerminalRunStopConflict(err) {
			return mapSpawnError(err)
		}
	}
	if err := s.store.DeleteSession(session.ID); err != nil {
		return mapStoreError(err)
	}
	s.emitDeleted(session)
	return nil
}

func isTerminalRunStopConflict(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "status 409") && (strings.Contains(message, "state_terminal") || strings.Contains(message, "cannot stop run in"))
}
