package agentsessions

import (
	"context"
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
	if err := s.checkProposalApplyCapability(proposal); err != nil {
		return Session{}, nil, err
	}

	session.Status = StatusApplying
	session.UpdatedAt = nowRFC3339()
	if err := s.store.SaveSession(session); err != nil {
		return Session{}, nil, err
	}

	prov := proposalApplyProvenance(session, proposal)
	artifacts, err := s.executeProposalApplyWork(ctx, &session, &proposal, prov)
	if err != nil {
		return Session{}, nil, err
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

// checkProposalApplyCapability verifies that the service has the dependencies
// needed to apply this proposal kind. Returns an error if the kind is unknown
// or if a required service is unavailable.
func (s *Service) checkProposalApplyCapability(proposal Proposal) error {
	switch proposal.Kind {
	case ProposalBacklogBatchImport:
		if s.backlogBatchApplier == nil {
			return apierr.Unavailable("backlog batch proposal apply is unavailable")
		}
	case ProposalOperatingModeDraft, ProposalOperatingModeImplementationPlan:
		return apierr.Wrapf(apierr.ErrNotImplemented, http.StatusGone, "legacy agent session proposal kind %q is read-only", string(proposal.Kind))
	default:
		return apierr.Wrapf(apierr.ErrNotImplemented, http.StatusNotImplemented, "agent session proposal kind %q apply is not implemented yet", string(proposal.Kind))
	}
	return nil
}

// executeProposalApplyWork dispatches to the kind-specific apply logic and
// returns the produced artifacts. On error it rolls the session and proposal
// back via failProposalApply and returns the original error unchanged.
func (s *Service) executeProposalApplyWork(ctx context.Context, session *Session, proposal *Proposal, prov identity.Provenance) ([]Artifact, error) {
	var artifacts []Artifact
	var err error
	switch proposal.Kind {
	case ProposalBacklogBatchImport:
		artifacts, err = s.backlogBatchApplier.ApplyAgentSessionBacklogBatchImport(identity.NewContext(ctx, prov), proposal.PayloadJSON, prov)
		if err != nil {
			return nil, s.failProposalApply(session, proposal, err)
		}
	}
	return artifacts, nil
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
