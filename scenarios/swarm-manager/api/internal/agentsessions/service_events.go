package agentsessions

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

// emitSessionEvent dispatches a session lifecycle event via the nil-guarded
// eventLogger. All seven session-state emitters share this path so the
// nil check and payload construction live in exactly one place.
func (s *Service) emitSessionEvent(session Session, emit func(EventLogger, string, any)) {
	if s.eventLogger != nil {
		emit(s.eventLogger, session.ID, eventPayload(session))
	}
}

func (s *Service) emitCreated(session Session) {
	s.emitSessionEvent(session, EventLogger.EmitAgentSessionCreated)
}

func (s *Service) emitStarted(session Session) {
	s.emitSessionEvent(session, EventLogger.EmitAgentSessionStarted)
}

func (s *Service) emitContinued(session Session) {
	s.emitSessionEvent(session, EventLogger.EmitAgentSessionContinued)
}

func (s *Service) emitFailed(session Session) {
	s.emitSessionEvent(session, EventLogger.EmitAgentSessionFailed)
}

func (s *Service) emitCanceled(session Session) {
	s.emitSessionEvent(session, EventLogger.EmitAgentSessionCanceled)
}

func (s *Service) emitCompleted(session Session) {
	s.emitSessionEvent(session, EventLogger.EmitAgentSessionCompleted)
}

func (s *Service) emitDeleted(session Session) {
	s.emitSessionEvent(session, EventLogger.EmitAgentSessionDeleted)
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
	SessionID         string `json:"session_id"`
	SessionKind       string `json:"session_kind"`
	Status            string `json:"status"`
	SkillID           string `json:"skill_id,omitempty"`
	RunID             string `json:"run_id,omitempty"`
	TaskID            string `json:"task_id,omitempty"`
	ProfileKey        string `json:"profile_key,omitempty"`
	FailureReason     string `json:"failure_reason,omitempty"`
	Disposition       string `json:"disposition,omitempty"`
	DispositionReason string `json:"disposition_reason,omitempty"`
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
		SessionID:         session.ID,
		SessionKind:       string(session.Kind),
		Status:            string(session.Status),
		SkillID:           session.SkillID,
		RunID:             session.RunID,
		TaskID:            session.TaskID,
		ProfileKey:        session.ProfileKey,
		FailureReason:     session.FailureReason,
		Disposition:       string(session.Disposition),
		DispositionReason: session.DispositionReason,
	}
}

func sessionKindFromAttribution(attr *Attribution) Kind {
	if attr == nil {
		return ""
	}
	return attr.SessionKind
}
