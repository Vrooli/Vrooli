package agentsessions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	kind := ProposalMutationList
	if !needsRevision && isNoChangeMutationProposal(ingested.PayloadJSON) {
		kind = ProposalNoChangeRecommendation
	}
	_, err = s.RecordProposal(ctx, session.ID, Proposal{
		Kind:             kind,
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

// proposalIsReady re-reads a mutation-list proposal and reports whether it is
// still eligible for a terminal apply/reject decision RIGHT NOW. It is the
// resolution-time readiness re-check the decision webhook runs before routing an
// operator answer to the authority: a proposal that is already applied/rejected,
// superseded (revised), failed, or gone is not ready, so the answer is a no-op. A
// not-found proposal is reported not-ready (false, nil), never an error, so a
// resolution for a vanished proposal is acknowledged rather than retried forever.
func (s *Service) proposalIsReady(ctx context.Context, sessionID, proposalID string) (bool, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return false, err
	}
	session, err := store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return false, nil
	}
	proposal, ok := findProposal(session, strings.TrimSpace(proposalID))
	if !ok || proposal.Kind != ProposalMutationList {
		return false, nil
	}
	return proposal.Status == ProposalStatusReady && !proposal.NeedsRevision, nil
}

// proposalCandidateMutationIDs returns every mutation id declared in a ready
// proposal's envelope. An "approve" answer with no explicit subset means "apply
// the whole proposal", which the authority expresses as the full candidate id set
// (an EMPTY accepted set is a rejection, so approve must never pass empty).
func (s *Service) proposalCandidateMutationIDs(ctx context.Context, sessionID, proposalID string) ([]string, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return nil, err
	}
	session, err := store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return nil, mapStoreError(err)
	}
	proposal, ok := findProposal(session, strings.TrimSpace(proposalID))
	if !ok {
		return nil, apierr.NotFound("session mutation proposal not found")
	}
	var payload struct {
		Mutations []struct {
			ID string `json:"id"`
		} `json:"mutations"`
	}
	if err := json.Unmarshal([]byte(proposal.PayloadJSON), &payload); err != nil {
		return nil, apierr.BadRequest("mutation proposal payload is invalid JSON: %v", err)
	}
	ids := make([]string, 0, len(payload.Mutations))
	for _, m := range payload.Mutations {
		if id := strings.TrimSpace(m.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
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

// isNoChangeMutationProposal recognizes the explicit mutation-list envelope
// used by proposal sessions to say that no graph change is recommended. The
// envelope's rationale remains the reviewer-facing explanation.
func isNoChangeMutationProposal(payloadJSON string) bool {
	var payload struct {
		Form      string            `json:"form"`
		Mutations []json.RawMessage `json:"mutations"`
	}
	if json.Unmarshal([]byte(payloadJSON), &payload) != nil {
		return false
	}
	return payload.Form == "mutation_list" && len(payload.Mutations) == 0
}

// DecideMutationListProposal records an explicit subset decision and delegates
// mutation to the API-composed processor, whose ApplyFlow revalidates current
// graph state immediately before changing anything.
func (s *Service) DecideMutationListProposal(ctx context.Context, sessionID, proposalID string, acceptedMutationIDs []string, note string) (Session, error) {
	session, proposal, err := s.loadMutationProposal(ctx, sessionID, proposalID)
	if err != nil {
		return Session{}, err
	}
	if proposal.Status != ProposalStatusReady || proposal.NeedsRevision {
		return Session{}, apierr.Conflict("mutation proposal must pass validation before apply")
	}
	rejected := len(acceptedMutationIDs) == 0
	application := MutationProposalApplication{}
	if !rejected {
		if s.mutationProposalProcessor == nil {
			return Session{}, apierr.Unavailable("session mutation proposal apply is unavailable")
		}
		var applyErr error
		application, applyErr = s.mutationProposalProcessor.Apply(ctx, *proposal.Target, proposal.PayloadJSON, acceptedMutationIDs, MutationProposalSource{SessionID: session.ID, RunID: session.RunID, DecidedAt: nowRFC3339()})
		if applyErr != nil {
			proposal.Status = ProposalStatusNeedsRevision
			proposal.NeedsRevision = true
			proposal.ValidationErrors = []string{applyErr.Error()}
			proposal.UpdatedAt = nowRFC3339()
			if saveErr := s.saveProposal(ctx, session.ID, proposal); saveErr != nil {
				return Session{}, saveErr
			}
			return s.loadSession(ctx, session.ID)
		}
	}
	proposal.Status = ProposalStatusApplied
	decisionKind := "apply"
	resolutionDecision := "applied"
	if rejected {
		proposal.Status = ProposalStatusRejected
		decisionKind = "reject"
		resolutionDecision = "rejected"
	}
	proposal.NeedsRevision = false
	proposal.UpdatedAt = nowRFC3339()
	proposal.Decisions = append(proposal.Decisions, ProposalDecision{
		Kind:                decisionKind,
		AcceptedMutationIDs: append([]string(nil), acceptedMutationIDs...),
		RejectedMutationIDs: rejectedMutationIDs(proposal.PayloadJSON, acceptedMutationIDs),
		Note:                strings.TrimSpace(note),
		Outcomes:            application.Outcomes,
		DecidedAt:           proposal.UpdatedAt,
	})
	if err := s.saveProposal(ctx, session.ID, proposal); err != nil {
		return Session{}, err
	}
	s.recordResolution(ctx, session, proposal, resolutionDecision, proposalResolutionReason(proposal, note))
	return s.loadSession(ctx, session.ID)
}

// AcceptNoChangeRecommendation records an explicit operator acceptance of a
// keep-as-is conclusion. Unlike Apply, this does not mutate item content; the
// composed processor records a separate review freshness signal instead.
func (s *Service) AcceptNoChangeRecommendation(ctx context.Context, sessionID, proposalID, note string) (Session, error) {
	session, proposal, err := s.loadNoChangeRecommendation(ctx, sessionID, proposalID)
	if err != nil {
		return Session{}, err
	}
	if proposal.Status != ProposalStatusReady || proposal.NeedsRevision {
		return Session{}, apierr.Conflict("no-change recommendation must be ready before acceptance")
	}
	if s.mutationProposalProcessor == nil {
		return Session{}, apierr.Unavailable("session proposal review is unavailable")
	}
	source := MutationProposalSource{SessionID: session.ID, RunID: session.RunID, DecidedAt: nowRFC3339(), ProposalID: proposal.ID}
	if err := s.mutationProposalProcessor.AcceptNoChange(ctx, *proposal.Target, proposal.PayloadJSON, source); err != nil {
		return Session{}, err
	}
	proposal.Status = ProposalStatusApplied
	proposal.UpdatedAt = source.DecidedAt
	proposal.Decisions = append(proposal.Decisions, ProposalDecision{Kind: "accept_keep", Note: strings.TrimSpace(note), DecidedAt: source.DecidedAt})
	if err := s.saveProposal(ctx, session.ID, proposal); err != nil {
		return Session{}, err
	}
	s.recordResolution(ctx, session, proposal, "no_change_recommendation", proposalResolutionReason(proposal, note))
	return s.loadSession(ctx, session.ID)
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
	return s.requestMutationProposalRevision(ctx, sessionID, proposalID, note, "")
}

// RequestMutationProposalRevisionForDecision applies a Hub answer with a
// stable decision identity. If delivery is retried after the proposal was
// durably marked superseded, the same message and Agent Manager continuation
// are replayed under the same idempotency key.
func (s *Service) RequestMutationProposalRevisionForDecision(ctx context.Context, sessionID, proposalID, note, askID string) (Session, error) {
	askID = strings.TrimSpace(askID)
	if askID == "" {
		return Session{}, apierr.BadRequest("request-changes decision requires an ask id")
	}
	if _, ok := s.spawner.(interface {
		ContinueRunWithIdempotencyKey(context.Context, string, string, string) error
	}); !ok {
		return Session{}, apierr.Unavailable("idempotent proposal revision continuation is unavailable")
	}
	return s.requestMutationProposalRevision(ctx, sessionID, proposalID, note, askID)
}

func (s *Service) requestMutationProposalRevision(ctx context.Context, sessionID, proposalID, note, askID string) (Session, error) {
	session, proposal, err := s.loadRevisableProposal(ctx, sessionID, proposalID)
	if err != nil {
		return Session{}, err
	}
	if askID == "" {
		return s.requestMutationProposalRevisionLegacy(ctx, session, proposal, note)
	}
	if s.spawner == nil {
		return Session{}, apierr.Conflict("proposal revision requires an attributed session run")
	}
	var priorDecision *ProposalDecision
	priorDecisionIndex := -1
	if askID != "" {
		for i := range proposal.Decisions {
			if proposal.Decisions[i].AskID == askID {
				priorDecision = &proposal.Decisions[i]
				priorDecisionIndex = i
				break
			}
		}
	}
	var continuationRunID, prompt, messageID string
	var decisionAt string
	if priorDecision != nil {
		if priorDecision.Kind != "request_changes" {
			return Session{}, apierr.Conflict("ask id is already recorded for another proposal decision")
		}
		if strings.TrimSpace(priorDecision.RunID) == "" || strings.TrimSpace(priorDecision.MessageID) == "" || strings.TrimSpace(priorDecision.Prompt) == "" {
			return Session{}, apierr.Conflict("stored proposal revision decision lacks its receiver identity")
		}
		if priorDecision.ContinuationAcked {
			return s.loadSession(ctx, session.ID)
		}
		note = priorDecision.Note
		continuationRunID, prompt, messageID = priorDecision.RunID, priorDecision.Prompt, priorDecision.MessageID
		decisionAt = priorDecision.DecidedAt
	} else {
		if strings.TrimSpace(session.RunID) == "" {
			return Session{}, apierr.Conflict("proposal revision requires an attributed session run")
		}
		if proposal.Status == ProposalStatusApplied || proposal.Status == ProposalStatusRejected || proposal.Status == ProposalStatusSuperseded {
			return Session{}, apierr.Conflict("proposal cannot be revised in its current state")
		}
		note = strings.TrimSpace(note)
		continuationRunID = strings.TrimSpace(session.RunID)
		prompt = revisionPrompt(proposal, note)
		messageID = proposalDecisionMessageID(askID)
		proposal.Status = ProposalStatusSuperseded
		proposal.UpdatedAt = nowRFC3339()
		decisionAt = proposal.UpdatedAt
		proposal.Decisions = append(proposal.Decisions, ProposalDecision{
			Kind: "request_changes", AskID: askID, RunID: continuationRunID, MessageID: messageID,
			Prompt: prompt, Note: note, DecidedAt: decisionAt,
		})
		priorDecisionIndex = len(proposal.Decisions) - 1
		if err := s.saveProposal(ctx, session.ID, proposal); err != nil {
			return Session{}, err
		}
	}
	messageExists := false
	for _, existing := range session.Messages {
		if existing.ID == messageID {
			messageExists = true
			if existing.Content != prompt {
				return Session{}, apierr.Conflict("stored revision message differs from the decision")
			}
			break
		}
	}
	message := Message{ID: messageID, Role: MessageRoleUser, Content: prompt, CreatedAt: decisionAt}
	store, err := s.storeFor(ctx)
	if err != nil {
		return Session{}, err
	}
	if !messageExists {
		if err := store.AppendMessage(session.ID, message); err != nil {
			return Session{}, err
		}
	}
	// Only the initial request, or a retry while the session still belongs to
	// the frozen receiver, updates session status. A later run must not be marked
	// running on behalf of an earlier Ask.
	if strings.TrimSpace(session.RunID) == continuationRunID {
		session.Status, session.UpdatedAt, session.FailureReason = StatusRunning, nowRFC3339(), ""
		if err := store.SaveSession(session); err != nil {
			return Session{}, err
		}
	}
	activityCtx := agentactivity.WithSpec(ctx, sessionActivitySpec(session, agentactivity.InteractionContinue))
	var continueErr error
	if spawner, ok := s.spawner.(interface {
		ContinueRunWithIdempotencyKey(context.Context, string, string, string) error
	}); !ok {
		continueErr = apierr.Unavailable("idempotent proposal revision continuation is unavailable")
	} else {
		continueErr = spawner.ContinueRunWithIdempotencyKey(activityCtx, continuationRunID, prompt, "swarm-manager/proposal-revision/"+askID)
	}
	if continueErr != nil {
		return Session{}, mapSpawnError(continueErr)
	}
	if priorDecisionIndex >= 0 {
		proposal.Decisions[priorDecisionIndex].ContinuationAcked = true
		proposal.UpdatedAt = nowRFC3339()
		if err := s.saveProposal(ctx, session.ID, proposal); err != nil {
			return Session{}, err
		}
	}
	return s.loadSession(ctx, session.ID)
}

// requestMutationProposalRevisionLegacy preserves the existing owner API's
// unkeyed behavior. Hub answers use the keyed path above because their stable
// Ask ID allows durable retry and acknowledgement.
func (s *Service) requestMutationProposalRevisionLegacy(ctx context.Context, session Session, proposal Proposal, note string) (Session, error) {
	if strings.TrimSpace(session.RunID) == "" || s.spawner == nil {
		return Session{}, apierr.Conflict("proposal revision requires an attributed session run")
	}
	if proposal.Status == ProposalStatusApplied || proposal.Status == ProposalStatusRejected || proposal.Status == ProposalStatusSuperseded {
		return Session{}, apierr.Conflict("proposal cannot be revised in its current state")
	}
	prompt := revisionPrompt(proposal, note)
	now := nowRFC3339()
	message := Message{ID: "msg_" + idgen.Generate(), Role: MessageRoleUser, Content: prompt, CreatedAt: now}
	store, err := s.storeFor(ctx)
	if err != nil {
		return Session{}, err
	}
	if err := store.AppendMessage(session.ID, message); err != nil {
		return Session{}, err
	}
	proposal.Status, proposal.UpdatedAt = ProposalStatusSuperseded, now
	if err := s.saveProposal(ctx, session.ID, proposal); err != nil {
		return Session{}, err
	}
	session.Status, session.UpdatedAt, session.FailureReason = StatusRunning, now, ""
	if err := store.SaveSession(session); err != nil {
		return Session{}, err
	}
	activityCtx := agentactivity.WithSpec(ctx, sessionActivitySpec(session, agentactivity.InteractionContinue))
	if err := s.spawner.ContinueRun(activityCtx, session.RunID, prompt); err != nil {
		return Session{}, mapSpawnError(err)
	}
	return s.loadSession(ctx, session.ID)
}

func proposalDecisionMessageID(askID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(askID)))
	return "msg_decision_" + hex.EncodeToString(sum[:])
}

// MarkProposalWaiting records an explicit operator decision to leave a ready
// proposal in the review queue. It does not require an attributed run and does
// not mutate the proposal target.
func (s *Service) MarkProposalWaiting(ctx context.Context, sessionID, proposalID, note string) (Session, error) {
	if strings.TrimSpace(note) == "" {
		return Session{}, apierr.BadRequest("waiting note is required")
	}
	session, proposal, err := s.loadProposal(ctx, sessionID, proposalID)
	if err != nil {
		return Session{}, err
	}
	if proposal.Status != ProposalStatusReady {
		return Session{}, apierr.Conflict("proposal must be ready before it can be marked waiting")
	}
	now := nowRFC3339()
	proposal.Status = ProposalStatusNeedsRevision
	proposal.NeedsRevision = true
	proposal.UpdatedAt = now
	proposal.Decisions = append(proposal.Decisions, ProposalDecision{Kind: "waiting", Note: strings.TrimSpace(note), DecidedAt: now})
	if err := s.saveProposal(ctx, session.ID, proposal); err != nil {
		return Session{}, err
	}
	return s.loadSession(ctx, session.ID)
}

func (s *Service) loadProposal(ctx context.Context, sessionID, proposalID string) (Session, Proposal, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return Session{}, Proposal{}, err
	}
	session, err := store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return Session{}, Proposal{}, mapStoreError(err)
	}
	proposal, ok := findProposal(session, strings.TrimSpace(proposalID))
	if !ok {
		return Session{}, Proposal{}, apierr.NotFound("session proposal not found")
	}
	return session, proposal, nil
}

func (s *Service) loadMutationProposal(ctx context.Context, sessionID, proposalID string) (Session, Proposal, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return Session{}, Proposal{}, err
	}
	session, err := store.LoadSession(strings.TrimSpace(sessionID))
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

func (s *Service) loadRevisableProposal(ctx context.Context, sessionID, proposalID string) (Session, Proposal, error) {
	session, proposal, err := s.loadMutationProposal(ctx, sessionID, proposalID)
	if err == nil {
		return session, proposal, nil
	}
	// A no-change recommendation has the same target and revision lifecycle as
	// a mutation proposal, but is intentionally excluded from the Apply path.
	store, storeErr := s.storeFor(ctx)
	if storeErr != nil {
		return Session{}, Proposal{}, storeErr
	}
	session, loadErr := store.LoadSession(strings.TrimSpace(sessionID))
	if loadErr != nil {
		return Session{}, Proposal{}, mapStoreError(loadErr)
	}
	proposal, ok := findProposal(session, strings.TrimSpace(proposalID))
	if !ok || proposal.Kind != ProposalNoChangeRecommendation || proposal.Target == nil {
		return Session{}, Proposal{}, err
	}
	return session, proposal, nil
}

func (s *Service) loadNoChangeRecommendation(ctx context.Context, sessionID, proposalID string) (Session, Proposal, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return Session{}, Proposal{}, err
	}
	session, err := store.LoadSession(strings.TrimSpace(sessionID))
	if err != nil {
		return Session{}, Proposal{}, mapStoreError(err)
	}
	proposal, ok := findProposal(session, strings.TrimSpace(proposalID))
	if !ok || proposal.Target == nil || (proposal.Kind != ProposalNoChangeRecommendation && !isNoChangeMutationProposal(proposal.PayloadJSON)) {
		return Session{}, Proposal{}, apierr.NotFound("session no-change recommendation not found")
	}
	return session, proposal, nil
}

func revisionPrompt(proposal Proposal, note string) string {
	intro := "Revise the mutation_list proposal for this session. Emit one complete fenced JSON mutation_list envelope."
	if proposal.Kind == ProposalNoChangeRecommendation {
		intro = "Revisit the no-change recommendation for this session. If the target should remain unchanged, emit a fenced mutation_list JSON envelope with an empty mutations array and a clear rationale. If changes are warranted, emit the complete mutation_list instead."
	}
	parts := []string{intro}
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
