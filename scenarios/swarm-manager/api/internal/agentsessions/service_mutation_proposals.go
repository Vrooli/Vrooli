package agentsessions

import (
	"context"
	"encoding/json"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/idgen"
)

// ingestMutationProposal runs exactly once for each newly persisted assistant
// turn. There is intentionally no generation lock: concurrent session turns
// may produce stale advice, but ApplyFlow revalidates fresh state at apply.
func (s *Service) ingestMutationProposal(ctx context.Context, session Session, assistantReply string) error {
	if session.ProposalTarget == nil || s.mutationProposalProcessor == nil {
		return nil
	}
	ingested, err := s.mutationProposalProcessor.Ingest(ctx, *session.ProposalTarget, assistantReply)
	if err != nil {
		return err
	}
	if strings.TrimSpace(ingested.PayloadJSON) == "" {
		ingested.PayloadJSON = `{}`
	}
	needsRevision := len(ingested.ParseWarnings) > 0 || len(ingested.ValidationErrors) > 0
	if !needsRevision && ingested.PayloadJSON == `{}` {
		needsRevision = true
		ingested.ParseWarnings = append(ingested.ParseWarnings, "agent output did not contain a parseable proposal JSON block")
	}
	status := ProposalStatusReady
	if needsRevision {
		status = ProposalStatusNeedsRevision
	}
	_, err = s.RecordProposal(ctx, session.ID, Proposal{
		Kind:             ProposalMutationList,
		Status:           status,
		Summary:          mutationProposalSummary(session.ProposalTarget, needsRevision),
		PayloadJSON:      ingested.PayloadJSON,
		Target:           session.ProposalTarget,
		NeedsRevision:    needsRevision,
		ParseWarnings:    append([]string(nil), ingested.ParseWarnings...),
		ValidationErrors: append([]string(nil), ingested.ValidationErrors...),
		Attribution:      &Attribution{Type: AttributionAgent, RunID: session.RunID, TaskID: session.TaskID, ProfileKey: session.ProfileKey, SessionID: session.ID, SessionKind: session.Kind, Source: "session/" + session.ID},
	})
	return err
}

func mutationProposalSummary(target *ProposalTarget, needsRevision bool) string {
	if target == nil {
		return "Session mutation proposal"
	}
	if needsRevision {
		return "Revision requested for " + target.Name
	}
	return "Mutation proposal for " + target.Name
}

// DecideMutationListProposal records an explicit subset decision and delegates
// mutation to the API-composed processor, whose ApplyFlow revalidates current
// graph state immediately before changing anything.
func (s *Service) DecideMutationListProposal(ctx context.Context, sessionID, proposalID string, acceptedMutationIDs []string, note string) (Session, error) {
	session, proposal, err := s.loadMutationProposal(sessionID, proposalID)
	if err != nil {
		return Session{}, err
	}
	if proposal.Status != ProposalStatusReady || proposal.NeedsRevision {
		return Session{}, apierr.Conflict("mutation proposal must pass validation before apply")
	}
	if s.mutationProposalProcessor == nil {
		return Session{}, apierr.Unavailable("session mutation proposal apply is unavailable")
	}
	application, applyErr := s.mutationProposalProcessor.Apply(ctx, *proposal.Target, proposal.PayloadJSON, acceptedMutationIDs, MutationProposalSource{SessionID: session.ID, RunID: session.RunID, DecidedAt: nowRFC3339()})
	if applyErr != nil {
		proposal.Status = ProposalStatusNeedsRevision
		proposal.NeedsRevision = true
		proposal.ValidationErrors = []string{applyErr.Error()}
		proposal.UpdatedAt = nowRFC3339()
		if saveErr := s.store.SaveProposal(session.ID, proposal); saveErr != nil {
			return Session{}, saveErr
		}
		return s.store.LoadSession(session.ID)
	}
	proposal.Status = ProposalStatusApplied
	proposal.NeedsRevision = false
	proposal.UpdatedAt = nowRFC3339()
	proposal.Decisions = append(proposal.Decisions, ProposalDecision{
		Kind:                "apply",
		AcceptedMutationIDs: append([]string(nil), acceptedMutationIDs...),
		RejectedMutationIDs: rejectedMutationIDs(proposal.PayloadJSON, acceptedMutationIDs),
		Note:                strings.TrimSpace(note),
		Outcomes:            application.Outcomes,
		DecidedAt:           proposal.UpdatedAt,
	})
	if err := s.store.SaveProposal(session.ID, proposal); err != nil {
		return Session{}, err
	}
	return s.store.LoadSession(session.ID)
}

func rejectedMutationIDs(payloadJSON string, accepted []string) []string {
	var payload struct {
		Mutations []struct {
			ID string `json:"id"`
		} `json:"mutations"`
	}
	if json.Unmarshal([]byte(payloadJSON), &payload) != nil {
		return nil
	}
	acceptedSet := make(map[string]struct{}, len(accepted))
	for _, id := range accepted {
		acceptedSet[strings.TrimSpace(id)] = struct{}{}
	}
	rejected := make([]string, 0, len(payload.Mutations))
	for _, mutation := range payload.Mutations {
		id := strings.TrimSpace(mutation.ID)
		if id == "" {
			continue
		}
		if _, ok := acceptedSet[id]; !ok {
			rejected = append(rejected, id)
		}
	}
	return rejected
}

func (s *Service) RequestMutationProposalRevision(ctx context.Context, sessionID, proposalID, note string) (Session, error) {
	session, proposal, err := s.loadMutationProposal(sessionID, proposalID)
	if err != nil {
		return Session{}, err
	}
	if strings.TrimSpace(session.RunID) == "" || s.spawner == nil {
		return Session{}, apierr.Conflict("proposal revision requires an attributed session run")
	}
	if proposal.Status == ProposalStatusApplied || proposal.Status == ProposalStatusRejected || proposal.Status == ProposalStatusSuperseded {
		return Session{}, apierr.Conflict("proposal cannot be revised in its current state")
	}
	prompt := revisionPrompt(proposal, note)
	now := nowRFC3339()
	message := Message{ID: "msg_" + idgen.Generate(), Role: MessageRoleUser, Content: prompt, CreatedAt: now}
	if err := s.store.AppendMessage(session.ID, message); err != nil {
		return Session{}, err
	}
	proposal.Status, proposal.UpdatedAt = ProposalStatusSuperseded, now
	if err := s.store.SaveProposal(session.ID, proposal); err != nil {
		return Session{}, err
	}
	session.Status, session.UpdatedAt, session.FailureReason = StatusRunning, now, ""
	if err := s.store.SaveSession(session); err != nil {
		return Session{}, err
	}
	activityCtx := agentactivity.WithSpec(ctx, sessionActivitySpec(session, agentactivity.InteractionContinue))
	if err := s.spawner.ContinueRun(activityCtx, session.RunID, prompt); err != nil {
		return Session{}, mapSpawnError(err)
	}
	return s.store.LoadSession(session.ID)
}

func (s *Service) loadMutationProposal(sessionID, proposalID string) (Session, Proposal, error) {
	session, err := s.store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return Session{}, Proposal{}, mapStoreError(err)
	}
	proposal, ok := findProposal(session, strings.TrimSpace(proposalID))
	if !ok || proposal.Kind != ProposalMutationList {
		return Session{}, Proposal{}, apierr.NotFound("session mutation proposal not found")
	}
	if proposal.Target == nil {
		return Session{}, Proposal{}, apierr.BadRequest("session mutation proposal target is missing")
	}
	return session, proposal, nil
}

func revisionPrompt(proposal Proposal, note string) string {
	parts := []string{"Revise the mutation_list proposal for this session. Emit one complete fenced JSON mutation_list envelope."}
	if len(proposal.ParseWarnings) > 0 {
		parts = append(parts, "Parse warnings:\n- "+strings.Join(proposal.ParseWarnings, "\n- "))
	}
	if len(proposal.ValidationErrors) > 0 {
		parts = append(parts, "Validation errors (address each verbatim):\n- "+strings.Join(proposal.ValidationErrors, "\n- "))
	}
	if strings.TrimSpace(note) != "" {
		parts = append(parts, "Reviewer note:\n"+strings.TrimSpace(note))
	}
	return strings.Join(parts, "\n\n")
}
