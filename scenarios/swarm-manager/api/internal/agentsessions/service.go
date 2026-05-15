package agentsessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/idgen"

	agentdomainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	SkillMetaOrchestrator       = "swarm-manager-meta-orchestrator"
	SkillOperatingModeAuthoring = "swarm-manager-operating-mode-authoring"
	SkillSwarmOperations        = "swarm-manager-operations-session"

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
	EmitAgentSessionDeleted(sessionID string, payload any)
	EmitAgentSessionProposalCreated(sessionID string, payload any)
	EmitAgentSessionProposalApplied(sessionID string, payload any)
	EmitAgentSessionArtifactLinked(sessionID string, payload any)
}

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

type Service struct {
	store               Store
	spawner             SessionSpawner
	eventReader         RunEventReader
	backlogBatchApplier BacklogBatchApplier
	contextResolver     ContextResolver
	eventLogger         EventLogger
	projectRoot         string
	profileKey          string
}

type ServiceConfig struct {
	Store               Store
	Spawner             SessionSpawner
	EventReader         RunEventReader
	BacklogBatchApplier BacklogBatchApplier
	ContextResolver     ContextResolver
	EventLogger         EventLogger
	ProjectRoot         string
	ProfileKey          string
}

type CreateRequest struct {
	Kind  Kind
	Title string
}

type ContinueRequest struct {
	SessionID     string
	Message       string
	AttachmentIDs []string
	ContextRefs   []ContextRef
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

type ListEventsRequest struct {
	SessionID     string
	AfterSequence int64
	Limit         int32
}

type ListEventsResult struct {
	Events            []RunEvent
	HasMore           bool
	NextAfterSequence int64
}

type RunEvent struct {
	ID              string `json:"id"`
	RunID           string `json:"run_id"`
	Sequence        int64  `json:"sequence"`
	CreatedAt       string `json:"created_at"`
	EventType       string `json:"event_type"`
	Role            string `json:"role,omitempty"`
	Content         string `json:"content,omitempty"`
	ToolName        string `json:"tool_name,omitempty"`
	ToolCallID      string `json:"tool_call_id,omitempty"`
	Input           string `json:"input,omitempty"`
	Output          string `json:"output,omitempty"`
	Error           string `json:"error,omitempty"`
	Status          string `json:"status,omitempty"`
	PreviousStatus  string `json:"previous_status,omitempty"`
	ProgressPhase   string `json:"progress_phase,omitempty"`
	ProgressPercent int32  `json:"progress_percent,omitempty"`
	ProgressMessage string `json:"progress_message,omitempty"`
	Summary         string `json:"summary,omitempty"`
	RawJSON         string `json:"raw_json,omitempty"`
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
		store:               cfg.Store,
		spawner:             cfg.Spawner,
		eventReader:         cfg.EventReader,
		backlogBatchApplier: cfg.BacklogBatchApplier,
		contextResolver:     cfg.ContextResolver,
		eventLogger:         cfg.EventLogger,
		projectRoot:         cfg.ProjectRoot,
		profileKey:          cfg.ProfileKey,
	}, nil
}

func (s *Service) SetBacklogBatchApplier(applier BacklogBatchApplier) {
	s.backlogBatchApplier = applier
}

func (s *Service) SetContextResolver(resolver ContextResolver) {
	s.contextResolver = resolver
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Session, error) {
	req.Title = strings.TrimSpace(req.Title)
	if !IsKnownKind(req.Kind) {
		return Session{}, apierr.BadRequest("session kind is invalid")
	}
	if req.Title == "" {
		return Session{}, apierr.BadRequest("title is required")
	}

	now := nowRFC3339()
	session := Session{
		ID:         "sess_" + idgen.Generate(),
		Title:      req.Title,
		Kind:       req.Kind,
		Status:     StatusDraft,
		SkillID:    skillIDForKind(req.Kind),
		ProfileKey: s.profileKey,
		CreatedAt:  now,
		UpdatedAt:  now,
		CreatedBy:  attributionForContext(ctx),
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
	contextItems, err := s.resolveMessageContext(ctx, session, req.ContextRefs)
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
		_ = s.store.SaveSession(session)
		s.emitFailed(session)
		return Session{}, mapSpawnError(err)
	}
	s.emitContinued(session)
	return s.store.LoadSession(session.ID)
}

func (s *Service) UploadAttachments(_ context.Context, sessionID string, uploads []AttachmentUpload) ([]Attachment, error) {
	if len(uploads) == 0 {
		return []Attachment{}, nil
	}
	if len(uploads) > 6 {
		return nil, apierr.BadRequest("no more than 6 image attachments are allowed per message")
	}
	if _, err := s.store.LoadSession(strings.TrimSpace(sessionID)); err != nil {
		return nil, mapStoreError(err)
	}
	attachments := make([]Attachment, 0, len(uploads))
	for _, upload := range uploads {
		mediaType, _, _ := mime.ParseMediaType(upload.ContentType)
		if !allowedSessionImageTypes[mediaType] {
			return nil, apierr.BadRequest("unsupported file type: %s", mediaType)
		}
		if upload.Reader == nil {
			return nil, apierr.BadRequest("attachment file is required")
		}
		attachment := Attachment{
			ID:          "att_" + idgen.Generate(),
			Filename:    strings.TrimSpace(upload.Filename),
			ContentType: mediaType,
			SizeBytes:   upload.SizeBytes,
			CreatedAt:   nowRFC3339(),
		}
		if attachment.Filename == "" {
			attachment.Filename = "unnamed"
		}
		if err := s.store.SaveAttachment(sessionID, attachment, upload.Reader); err != nil {
			return nil, err
		}
		attachments = append(attachments, attachment)
	}
	return attachments, nil
}

func (s *Service) AttachmentPath(sessionID string, attachmentID string) (string, Attachment, error) {
	path, attachment, err := s.store.AttachmentPath(strings.TrimSpace(sessionID), strings.TrimSpace(attachmentID))
	if err != nil {
		return "", Attachment{}, mapStoreError(err)
	}
	return path, attachment, nil
}

func (s *Service) ListEvents(ctx context.Context, req ListEventsRequest) (ListEventsResult, error) {
	session, err := s.store.LoadSession(strings.TrimSpace(req.SessionID))
	if err != nil {
		return ListEventsResult{}, mapStoreError(err)
	}
	if strings.TrimSpace(session.RunID) == "" {
		return ListEventsResult{Events: []RunEvent{}}, nil
	}
	if s.eventReader == nil {
		return ListEventsResult{}, apierr.Unavailable("agent session events are unavailable")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	events, hasMore, err := s.eventReader.GetRunEvents(ctx, session.RunID, agentmanager.RunEventsOptions{
		AfterSequence: req.AfterSequence,
		Limit:         limit,
	})
	if err != nil {
		return ListEventsResult{}, mapSpawnError(err)
	}
	result := ListEventsResult{
		Events:  make([]RunEvent, 0, len(events)),
		HasMore: hasMore,
	}
	for _, event := range events {
		mapped := mapRunEvent(event)
		if mapped.Sequence > result.NextAfterSequence {
			result.NextAfterSequence = mapped.Sequence
		}
		result.Events = append(result.Events, mapped)
	}
	return result, nil
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

func (s *Service) Delete(ctx context.Context, sessionID string) error {
	session, err := s.store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return mapStoreError(err)
	}
	if strings.TrimSpace(session.RunID) != "" && s.spawner != nil && isActiveSessionStatus(session.Status) {
		if err := s.spawner.StopRun(ctx, session.RunID); err != nil {
			return mapSpawnError(err)
		}
	}
	if err := s.store.DeleteSession(session.ID); err != nil {
		return mapStoreError(err)
	}
	s.emitDeleted(session)
	return nil
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

func (s *Service) AttachArtifacts(_ context.Context, artifacts []Artifact) ([]Artifact, error) {
	if len(artifacts) == 0 {
		return []Artifact{}, nil
	}
	for i := range artifacts {
		if artifacts[i].ID == "" {
			artifacts[i].ID = "art_" + idgen.Generate()
		}
		if artifacts[i].CreatedAt == "" {
			artifacts[i].CreatedAt = nowRFC3339()
		}
		if i > 0 && artifacts[i].SessionID != artifacts[0].SessionID {
			return nil, apierr.BadRequest("all artifacts must belong to the same session")
		}
	}
	if err := s.store.AppendArtifacts(artifacts[0].SessionID, artifacts); err != nil {
		return nil, err
	}
	for _, artifact := range artifacts {
		s.emitArtifactLinked(artifact)
	}
	return artifacts, nil
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
	s.emitProposalCreated(sessionID, proposal)
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
	case ProposalBacklogBatchImport, ProposalOperatingModeImplementationPlan:
		if s.backlogBatchApplier == nil {
			return Session{}, nil, apierr.Unavailable("backlog batch proposal apply is unavailable")
		}
	case ProposalOperatingModeDraft:
	default:
		return Session{}, nil, apierr.Wrapf(apierr.ErrNotImplemented, http.StatusNotImplemented, "agent session proposal kind %q apply is not implemented yet", string(proposal.Kind))
	}

	session.Status = StatusApplying
	session.UpdatedAt = nowRFC3339()
	if err := s.store.SaveSession(session); err != nil {
		return Session{}, nil, err
	}

	var artifacts []Artifact
	prov := proposalApplyProvenance(session, proposal)
	switch proposal.Kind {
	case ProposalBacklogBatchImport:
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
	case ProposalOperatingModeImplementationPlan:
		payloadJSON, err := backlogBatchPayloadForOperatingModePlan(proposal.PayloadJSON)
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
		artifacts, err = s.backlogBatchApplier.ApplyAgentSessionBacklogBatchImport(identity.NewContext(ctx, prov), payloadJSON, prov)
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
	case ProposalOperatingModeDraft:
		attr := AttributionFromProvenance(prov)
		artifact, err := s.AttachArtifact(ctx, Artifact{
			SessionID:      session.ID,
			ArtifactType:   ArtifactOperatingModeProposal,
			Action:         ArtifactActionProposed,
			EntityRef:      operatingModeProposalRef(proposal),
			Title:          proposal.Summary,
			ProposalID:     proposal.ID,
			RunID:          prov.RunID,
			MutationSource: "agent_sessions.apply.operating_mode_draft",
			Attribution:    &attr,
		})
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
		artifacts = append(artifacts, artifact)
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
	s.emitProposalApplied(session, proposal, len(artifacts))
	applied, err := s.store.LoadSession(session.ID)
	if err != nil {
		return Session{}, nil, err
	}
	return applied, artifacts, nil
}

func backlogBatchPayloadForOperatingModePlan(payloadJSON string) (string, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return "", apierr.BadRequest("invalid operating-mode implementation plan payload: %s", err)
	}
	for _, field := range []string{"backlog_batch_import", "batch_import", "backlog_batch"} {
		if raw, ok := payload[field]; ok {
			if !json.Valid(raw) {
				return "", apierr.BadRequest("operating-mode implementation plan field %q must be valid JSON", field)
			}
			return string(raw), nil
		}
	}
	if _, ok := payload["items"]; ok {
		return payloadJSON, nil
	}
	return "", apierr.BadRequest("operating-mode implementation plan payload must include items or backlog_batch_import")
}

func operatingModeProposalRef(proposal Proposal) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(proposal.PayloadJSON), &payload); err == nil {
		for _, field := range []string{"mode_id", "mode", "id", "name"} {
			if value, ok := payload[field].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return proposal.ID
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
	case KindSwarmOperations:
		return SkillSwarmOperations
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

var allowedSessionImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

func (s *Service) resolveMessageContext(ctx context.Context, session Session, refs []ContextRef) ([]ContextItem, error) {
	normalized, err := normalizeContextRefs(session.Kind, refs)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	if s.contextResolver == nil {
		return nil, apierr.Unavailable("agent session context resolution is unavailable")
	}
	items, err := s.contextResolver.ResolveSessionMessageContext(ctx, normalized, contextLimitsForKind(session.Kind))
	if err != nil {
		return nil, err
	}
	now := nowRFC3339()
	for i := range items {
		if items[i].SelectedAt == "" {
			items[i].SelectedAt = now
		}
		items[i].Summary = truncateRunes(strings.TrimSpace(items[i].Summary), contextLimitsForKind(session.Kind).MaxSummaryRunes)
	}
	return items, nil
}

func normalizeContextRefs(kind Kind, refs []ContextRef) ([]ContextRef, error) {
	limits := contextLimitsForKind(kind)
	if len(refs) > limits.MaxTotal {
		return nil, apierr.BadRequest("no more than %d context items are allowed for %s sessions", limits.MaxTotal, kind)
	}
	seen := make(map[string]struct{}, len(refs))
	counts := make(map[ContextType]int)
	normalized := make([]ContextRef, 0, len(refs))
	for _, ref := range refs {
		contextType := ContextType(strings.TrimSpace(string(ref.Type)))
		value := strings.TrimSpace(ref.Ref)
		if !IsKnownContextType(contextType) {
			return nil, apierr.BadRequest("context type is invalid")
		}
		if value == "" {
			return nil, apierr.BadRequest("context ref is required")
		}
		if max, ok := limits.MaxPerType[contextType]; !ok || max <= 0 {
			return nil, apierr.BadRequest("context type %q is not allowed for %s sessions", contextType, kind)
		}
		key := string(contextType) + "\x00" + value
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		counts[contextType]++
		if counts[contextType] > limits.MaxPerType[contextType] {
			return nil, apierr.BadRequest("too many %s context items; max is %d", contextType, limits.MaxPerType[contextType])
		}
		normalized = append(normalized, ContextRef{Type: contextType, Ref: value})
	}
	if len(normalized) > limits.MaxTotal {
		return nil, apierr.BadRequest("no more than %d context items are allowed for %s sessions", limits.MaxTotal, kind)
	}
	return normalized, nil
}

func contextLimitsForKind(kind Kind) ContextLimits {
	common := map[ContextType]int{
		ContextBacklogItem:   8,
		ContextInitiative:    4,
		ContextCapture:       4,
		ContextExecution:     6,
		ContextAgentActivity: 6,
		ContextScenario:      3,
		ContextOperatingMode: 3,
		ContextSession:       2,
	}
	switch kind {
	case KindOperatingModeAuthoring:
		return ContextLimits{Kind: kind, MaxTotal: 8, MaxPerType: common, MaxSummaryRunes: 1200}
	default:
		return ContextLimits{Kind: kind, MaxTotal: 12, MaxPerType: common, MaxSummaryRunes: 1200}
	}
}

func sessionAttachmentsByID(session Session, attachmentIDs []string) []Attachment {
	if len(attachmentIDs) == 0 || len(session.Attachments) == 0 {
		return nil
	}
	wanted := make(map[string]struct{}, len(attachmentIDs))
	for _, id := range attachmentIDs {
		wanted[strings.TrimSpace(id)] = struct{}{}
	}
	var attachments []Attachment
	for _, attachment := range session.Attachments {
		if _, ok := wanted[attachment.ID]; ok {
			attachments = append(attachments, attachment)
		}
	}
	return attachments
}

func buildInitialPrompt(session Session, message Message, attachments []Attachment) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are running a Swarm Manager %s agent session.\n\n", session.Kind)
	fmt.Fprintf(&b, "Use the Prompt Manager skill `%s` as your operating guide.\n", session.SkillID)
	fmt.Fprintf(&b, "Session ID: %s\n", session.ID)
	writeMessageContext(&b, message.Context)
	writeMessageAttachments(&b, attachments)
	b.WriteString("\nOperator message:\n")
	writeOperatorMessage(&b, message.Content)
	return b.String()
}

func buildContinuationPrompt(message Message, attachments []Attachment) string {
	if len(message.Context) == 0 && len(attachments) == 0 {
		return strings.TrimSpace(message.Content)
	}
	var b strings.Builder
	writeMessageContext(&b, message.Context)
	writeMessageAttachments(&b, attachments)
	b.WriteString("\nOperator message:\n")
	writeOperatorMessage(&b, message.Content)
	return b.String()
}

func writeMessageContext(b *strings.Builder, contextItems []ContextItem) {
	if len(contextItems) == 0 {
		return
	}
	b.WriteString("\nAttached context:\n")
	for i, item := range contextItems {
		fmt.Fprintf(b, "%d. [%s] %s (%s)\n", i+1, item.Type, item.Title, item.Ref)
		if strings.TrimSpace(item.MetadataJSON) != "" {
			fmt.Fprintf(b, "   Metadata: %s\n", strings.TrimSpace(item.MetadataJSON))
		}
		if strings.TrimSpace(item.Summary) != "" {
			fmt.Fprintf(b, "   Summary: %s\n", strings.TrimSpace(item.Summary))
		}
	}
}

func writeMessageAttachments(b *strings.Builder, attachments []Attachment) {
	if len(attachments) == 0 {
		return
	}
	b.WriteString("\nAttached images:\n")
	for _, attachment := range attachments {
		fmt.Fprintf(b, "- %s: %s (%s)\n", attachment.ID, attachment.Filename, attachment.ContentType)
	}
}

func writeOperatorMessage(b *strings.Builder, content string) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		b.WriteString("(no text supplied)")
		return
	}
	b.WriteString(trimmed)
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

const maxRunEventFieldBytes = 6000

func mapRunEvent(event *agentdomainpb.RunEvent) RunEvent {
	if event == nil {
		return RunEvent{}
	}
	mapped := RunEvent{
		ID:        strings.TrimSpace(event.GetId()),
		RunID:     strings.TrimSpace(event.GetRunId()),
		Sequence:  event.GetSequence(),
		EventType: agentRunEventType(event.GetEventType()),
	}
	if ts := event.GetTimestamp(); ts != nil {
		mapped.CreatedAt = ts.AsTime().UTC().Format(time.RFC3339)
	}
	switch data := event.GetData().(type) {
	case *agentdomainpb.RunEvent_Message:
		mapped.Role = strings.TrimSpace(data.Message.GetRole())
		mapped.Content = boundedText(data.Message.GetContent())
	case *agentdomainpb.RunEvent_ToolCall:
		mapped.ToolName = strings.TrimSpace(data.ToolCall.GetToolName())
		mapped.ToolCallID = strings.TrimSpace(data.ToolCall.GetToolCallId())
		mapped.Input = boundedText(structJSON(data.ToolCall.GetInput()))
	case *agentdomainpb.RunEvent_ToolResult:
		mapped.ToolName = strings.TrimSpace(data.ToolResult.GetToolName())
		mapped.ToolCallID = strings.TrimSpace(data.ToolResult.GetToolCallId())
		mapped.Output = boundedText(data.ToolResult.GetOutput())
		mapped.Error = boundedText(data.ToolResult.GetError())
	case *agentdomainpb.RunEvent_Status:
		mapped.PreviousStatus = strings.TrimSpace(data.Status.GetOldStatus())
		mapped.Status = strings.TrimSpace(data.Status.GetNewStatus())
		mapped.Summary = boundedText(data.Status.GetReason())
	case *agentdomainpb.RunEvent_Error:
		mapped.Error = boundedText(data.Error.GetMessage())
		if code := strings.TrimSpace(data.Error.GetCode()); code != "" {
			mapped.Status = code
		}
		mapped.RawJSON = boundedText(structJSON(data.Error.GetDetails()))
	case *agentdomainpb.RunEvent_Progress:
		mapped.ProgressPhase = strings.TrimSpace(data.Progress.GetPhase().String())
		mapped.ProgressPercent = data.Progress.GetPercentComplete()
		mapped.ProgressMessage = boundedText(data.Progress.GetCurrentAction())
	case *agentdomainpb.RunEvent_Compaction:
		mapped.Summary = boundedText(data.Compaction.GetSummary())
		mapped.ProgressMessage = boundedText(data.Compaction.GetTrigger())
	case *agentdomainpb.RunEvent_Log:
		mapped.Summary = boundedText(data.Log.GetMessage())
		mapped.Status = strings.TrimSpace(data.Log.GetLevel())
	case *agentdomainpb.RunEvent_Metric:
		mapped.Summary = boundedText(data.Metric.GetName())
	case *agentdomainpb.RunEvent_Artifact:
		mapped.Summary = boundedText(data.Artifact.GetPath())
	case *agentdomainpb.RunEvent_Cost:
		mapped.Summary = boundedText(fmt.Sprintf("$%.4f", data.Cost.GetTotalCostUsd()))
	case *agentdomainpb.RunEvent_RateLimit:
		mapped.Error = boundedText(data.RateLimit.GetMessage())
		mapped.Status = strings.TrimSpace(data.RateLimit.GetLimitType())
	default:
		mapped.RawJSON = boundedText(protoMessageJSON(event))
	}
	return mapped
}

func agentRunEventType(eventType agentdomainpb.RunEventType) string {
	switch eventType {
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_LOG:
		return "log"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE:
		return "message"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL:
		return "tool_call"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT:
		return "tool_result"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_STATUS:
		return "status"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_METRIC:
		return "metric"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_ARTIFACT:
		return "artifact"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_ERROR:
		return "error"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED:
		return "message_deleted"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_COMPACTION:
		return "compaction"
	case agentdomainpb.RunEventType_RUN_EVENT_TYPE_LIFECYCLE:
		return "lifecycle"
	default:
		return "unknown"
	}
}

func structJSON(value *structpb.Struct) string {
	if value == nil {
		return ""
	}
	return protoMessageJSON(value)
}

func protoMessageJSON(value interface{ ProtoReflect() protoreflect.Message }) string {
	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func boundedText(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= maxRunEventFieldBytes {
		return value
	}
	return value[:maxRunEventFieldBytes] + "\n[truncated]"
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

func (s *Service) emitDeleted(session Session) {
	if s.eventLogger != nil {
		s.eventLogger.EmitAgentSessionDeleted(session.ID, eventPayload(session))
	}
}

func (s *Service) emitArtifactLinked(artifact Artifact) {
	if s.eventLogger != nil {
		sessionKind := ""
		if artifact.Attribution != nil {
			sessionKind = string(artifact.Attribution.SessionKind)
		}
		s.eventLogger.EmitAgentSessionArtifactLinked(artifact.SessionID, AgentSessionArtifactEventPayload{
			SessionID:      artifact.SessionID,
			SessionKind:    sessionKind,
			ArtifactType:   string(artifact.ArtifactType),
			Action:         string(artifact.Action),
			EntityRef:      artifact.EntityRef,
			ProposalID:     artifact.ProposalID,
			RunID:          artifact.RunID,
			MutationSource: artifact.MutationSource,
		})
	}
}

func (s *Service) emitProposalCreated(sessionID string, proposal Proposal) {
	if s.eventLogger != nil {
		s.eventLogger.EmitAgentSessionProposalCreated(sessionID, AgentSessionProposalEventPayload{
			SessionID:    sessionID,
			SessionKind:  string(sessionKindFromAttribution(proposal.Attribution)),
			ProposalID:   proposal.ID,
			ProposalKind: string(proposal.Kind),
			Status:       string(proposal.Status),
		})
	}
}

func (s *Service) emitProposalApplied(session Session, proposal Proposal, artifactCount int) {
	if s.eventLogger != nil {
		s.eventLogger.EmitAgentSessionProposalApplied(session.ID, AgentSessionProposalEventPayload{
			SessionID:     session.ID,
			SessionKind:   string(session.Kind),
			ProposalID:    proposal.ID,
			ProposalKind:  string(proposal.Kind),
			Status:        string(proposal.Status),
			ArtifactCount: artifactCount,
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

type AgentSessionProposalEventPayload struct {
	SessionID     string `json:"session_id"`
	SessionKind   string `json:"session_kind,omitempty"`
	ProposalID    string `json:"proposal_id"`
	ProposalKind  string `json:"proposal_kind"`
	Status        string `json:"status"`
	ArtifactCount int    `json:"artifact_count,omitempty"`
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

func sessionKindFromAttribution(attr *Attribution) Kind {
	if attr == nil {
		return ""
	}
	return attr.SessionKind
}
