package agentsessions

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/idgen"
)

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
			return Session{}, nil, s.failProposalApply(&session, &proposal, err)
		}
	case ProposalOperatingModeImplementationPlan:
		payloadJSON, err := backlogBatchPayloadForOperatingModePlan(proposal.PayloadJSON)
		if err != nil {
			return Session{}, nil, s.failProposalApply(&session, &proposal, err)
		}
		artifacts, err = s.backlogBatchApplier.ApplyAgentSessionBacklogBatchImport(identity.NewContext(ctx, prov), payloadJSON, prov)
		if err != nil {
			return Session{}, nil, s.failProposalApply(&session, &proposal, err)
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
			return Session{}, nil, s.failProposalApply(&session, &proposal, err)
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

// failProposalApply rolls the proposal back to failed and the session back to
// proposal_ready, best-effort persisting both, and returns applyErr unchanged so
// callers can `return ..., s.failProposalApply(...)`.
func (s *Service) failProposalApply(session *Session, proposal *Proposal, applyErr error) error {
	proposal.Status = ProposalStatusFailed
	proposal.UpdatedAt = nowRFC3339()
	session.Status = StatusProposalReady
	session.FailureReason = applyErr.Error()
	session.UpdatedAt = proposal.UpdatedAt
	if err := s.store.SaveProposal(session.ID, *proposal); err != nil {
		slog.Warn("agentsessions: persist proposal failed", "session", session.ID, "proposal", proposal.ID, "err", err)
	}
	if err := s.store.SaveSession(*session); err != nil {
		slog.Warn("agentsessions: persist session failed", "session", session.ID, "err", err)
	}
	return applyErr
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
