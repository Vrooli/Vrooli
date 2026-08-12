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
	"swarm-manager/internal/transitionrunner"
)

func (s *Service) RecordProposal(ctx context.Context, sessionID string, proposal Proposal) (Proposal, error) {
	if proposal.ID == "" {
		proposal.ID = "prop_" + idgen.Generate()
	}
	now := nowRFC3339()
	if proposal.CreatedAt == "" {
		proposal.CreatedAt = now
	}
	proposal.UpdatedAt = now
	if err := s.saveProposal(ctx, sessionID, proposal); err != nil {
		return Proposal{}, err
	}
	s.emitProposalCreated(sessionID, proposal)
	return proposal, nil
}

// RecordWorkflowMutationProposals creates the durable proposal-session
// projection of a completed declared workflow. The workflow remains the
// immutable evidence source; this session is the single operator decision
// inbox and deliberately does not attempt to start an agent-session run.
func (s *Service) RecordWorkflowMutationProposals(ctx context.Context, title, summary, baseVersion string, target ProposalTarget, payloads []string, attribution Attribution) (Session, []Proposal, error) {
	if strings.TrimSpace(title) == "" {
		title = "Workflow proposals for " + target.Name
	}
	store, err := s.storeFor(ctx)
	if err != nil {
		return Session{}, nil, err
	}
	session, err := s.Create(ctx, CreateRequest{Kind: KindSwarmOperations, Title: title, ProposalTarget: &target})
	if err != nil {
		return Session{}, nil, err
	}
	session.Status = StatusProposalReady
	session.UpdatedAt = nowRFC3339()
	if err := store.SaveSession(session); err != nil {
		return Session{}, nil, err
	}
	if len(payloads) == 0 {
		payload, marshalErr := json.Marshal(map[string]any{"form": "mutation_list", "rationale": strings.TrimSpace(summary), "base_version": strings.TrimSpace(baseVersion), "mutations": []any{}})
		if marshalErr != nil {
			return Session{}, nil, marshalErr
		}
		payloads = []string{string(payload)}
	}
	proposals := make([]Proposal, 0, len(payloads))
	for _, payload := range payloads {
		kind := ProposalMutationList
		if isNoChangeMutationProposal(payload) {
			kind = ProposalNoChangeRecommendation
		}
		proposal, recordErr := s.RecordProposal(ctx, session.ID, Proposal{Kind: kind, Status: ProposalStatusReady, Summary: summary, PayloadJSON: payload, Target: &target, Attribution: &attribution})
		if recordErr != nil {
			return Session{}, nil, recordErr
		}
		proposals = append(proposals, proposal)
	}
	stored, err := store.LoadSession(session.ID)
	return stored, proposals, err
}

func (s *Service) ApplyProposal(ctx context.Context, sessionID, proposalID string) (Session, []Artifact, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(proposalID) == "" {
		return Session{}, nil, apierr.BadRequest("session_id and proposal_id are required")
	}
	store, err := s.storeFor(ctx)
	if err != nil {
		return Session{}, nil, err
	}
	session, err := store.LoadSession(strings.TrimSpace(sessionID))
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
	if err := store.SaveSession(session); err != nil {
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
	if err := s.saveProposal(ctx, session.ID, proposal); err != nil {
		return Session{}, nil, err
	}
	if err := store.SaveSession(session); err != nil {
		return Session{}, nil, err
	}
	s.emitProposalApplied(session, proposal, len(artifacts))
	applied, err := store.LoadSession(session.ID)
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
	case ProposalStartTransition:
		if _, err := s.validateStartTransitionProposal(proposal); err != nil {
			return err
		}
		if s.transitionStarter == nil {
			return apierr.Unavailable("transition proposal apply is unavailable")
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
			return nil, s.failProposalApply(ctx, session, proposal, err)
		}
	case ProposalStartTransition:
		start, validateErr := s.validateStartTransitionProposal(*proposal)
		if validateErr != nil {
			return nil, s.failProposalApply(ctx, session, proposal, validateErr)
		}
		correlation, startErr := s.transitionStarter.StartWith(identity.NewContext(ctx, prov), start.TransitionKey, start.SubjectValue, transitionrunner.PreparedInput{})
		if startErr != nil {
			return nil, s.failProposalApply(ctx, session, proposal, apierr.Wrapf(apierr.ErrBadGateway, http.StatusBadGateway, "start transition %q: %v", start.TransitionKey, startErr))
		}
		if strings.TrimSpace(correlation.ExecutionID) == "" {
			return nil, s.failProposalApply(ctx, session, proposal, apierr.BadGateway("start transition %q returned no execution id", start.TransitionKey))
		}
		artifact, artifactErr := s.AttachArtifact(ctx, Artifact{
			SessionID:      session.ID,
			ArtifactType:   ArtifactTransitionExecution,
			Action:         ArtifactActionCreated,
			EntityRef:      "transition/" + correlation.ExecutionID,
			Title:          start.TransitionKey,
			ProposalID:     proposal.ID,
			RunID:          session.RunID,
			MutationSource: string(ProposalStartTransition),
			Attribution: &Attribution{
				Type:        AttributionAgent,
				RunID:       prov.RunID,
				TaskID:      prov.TaskID,
				ProfileKey:  prov.ProfileKey,
				SessionID:   session.ID,
				SessionKind: session.Kind,
				Source:      "session/" + session.ID,
			},
		})
		if artifactErr != nil {
			return nil, s.failProposalApply(ctx, session, proposal, artifactErr)
		}
		artifacts = []Artifact{artifact}
	}
	return artifacts, nil
}

type validatedStartTransitionProposal struct {
	StartTransitionProposal
	SubjectValue string
}

// validateStartTransitionProposal keeps the proposal envelope deliberately
// small while preserving the registry's typed subject contract. The wire
// subject_ref is `<declared-subject>/<value>` (or the equivalent `:` form);
// only the value is handed to transitionrunner, which owns domain parsing.
func (s *Service) validateStartTransitionProposal(proposal Proposal) (validatedStartTransitionProposal, error) {
	var payload StartTransitionProposal
	if err := json.Unmarshal([]byte(proposal.PayloadJSON), &payload); err != nil {
		return validatedStartTransitionProposal{}, apierr.BadRequest("start_transition payload is invalid JSON: %v", err)
	}
	payload.TransitionKey = strings.TrimSpace(payload.TransitionKey)
	payload.SubjectRef = strings.TrimSpace(payload.SubjectRef)
	payload.ProjectionAction = strings.TrimSpace(payload.ProjectionAction)
	payload.Reason = strings.TrimSpace(payload.Reason)
	if payload.TransitionKey == "" || payload.SubjectRef == "" || payload.ProjectionAction == "" {
		return validatedStartTransitionProposal{}, apierr.BadRequest("start_transition requires transition_key, subject_ref, and projection_action")
	}
	definition, ok := s.transitionRegistry.Get(payload.TransitionKey)
	if !ok {
		return validatedStartTransitionProposal{}, apierr.BadRequest("transition %q is not declared", payload.TransitionKey)
	}
	subjectValue, ok := splitDeclaredSubjectRef(payload.SubjectRef, definition.Subject)
	if !ok {
		return validatedStartTransitionProposal{}, apierr.BadRequest("subject_ref must match declared subject %q", definition.Subject)
	}
	if !payload.ProjectionAgrees && payload.Reason == "" {
		return validatedStartTransitionProposal{}, apierr.BadRequest("start_transition requires a reason when projection_agrees is false")
	}
	return validatedStartTransitionProposal{StartTransitionProposal: payload, SubjectValue: subjectValue}, nil
}

func splitDeclaredSubjectRef(subjectRef, declaredSubject string) (string, bool) {
	for _, separator := range []string{"/", ":"} {
		prefix, value, found := strings.Cut(subjectRef, separator)
		if found && strings.TrimSpace(prefix) == declaredSubject && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), true
		}
	}
	return "", false
}

// failProposalApply rolls the proposal back to failed and the session back to
// proposal_ready, best-effort persisting both, and returns applyErr unchanged so
// callers can `return ..., s.failProposalApply(...)`.
func (s *Service) failProposalApply(ctx context.Context, session *Session, proposal *Proposal, applyErr error) error {
	proposal.Status = ProposalStatusFailed
	proposal.UpdatedAt = nowRFC3339()
	session.Status = StatusProposalReady
	session.FailureReason = applyErr.Error()
	session.UpdatedAt = proposal.UpdatedAt
	if err := s.saveProposal(ctx, session.ID, *proposal); err != nil {
		slog.Warn("agentsessions: persist proposal failed", "session", session.ID, "proposal", proposal.ID, "err", err)
	}
	if store, err := s.storeFor(ctx); err != nil {
		slog.Warn("agentsessions: resolve session store failed", "session", session.ID, "err", err)
	} else if err := store.SaveSession(*session); err != nil {
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
			Actor:       identity.TypeAgent,
			RunID:       proposal.Attribution.RunID,
			TaskID:      proposal.Attribution.TaskID,
			ProfileKey:  proposal.Attribution.ProfileKey,
			SessionID:   session.ID,
			SessionKind: string(session.Kind),
			Source:      "session/" + session.ID,
		}
	}
	return identity.Provenance{
		Actor:       identity.TypeAgent,
		RunID:       session.RunID,
		TaskID:      session.TaskID,
		ProfileKey:  session.ProfileKey,
		SessionID:   session.ID,
		SessionKind: string(session.Kind),
		Source:      "session/" + session.ID,
	}
}
