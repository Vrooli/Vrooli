package feedback

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/proposals"
)

// RecordAgentTurn persists an agent-generated message into the round's
// thread and, when the message body carries a structured proposal JSON
// block, attaches it as a new ProposalRevision and makes it current.
//
// This is the inbound hook for the agent-manager listener: when the agent
// emits output on its run, the listener calls here with the raw text. The
// hook releases the lock and flips the round to awaiting_user — the user
// now owns the next move.
func (s *Service) RecordAgentTurn(initiativeName string, roundNumber int, body string) (Round, error) {
	round, err := s.store.LoadRound(initiativeName, roundNumber)
	if err != nil {
		return Round{}, err
	}
	if round.Status != RoundStatusAgentThinking {
		return Round{}, fmt.Errorf("round is in status %q; agent turn requires %q", round.Status, RoundStatusAgentThinking)
	}
	now := s.clock().UTC().Format(time.RFC3339)
	msg := Message{
		Role:      "agent",
		Content:   body,
		RunID:     round.RunID,
		CreatedAt: now,
	}

	extracted, rawProposal, warnings := extractProposal(body)
	msg.ParseWarnings = append([]string(nil), warnings...)
	if extracted != nil {
		revision := ProposalRevision{
			ID:              fmt.Sprintf("p%d", len(round.Proposals)+1),
			Proposal:        *extracted,
			CreatedAt:       now,
			ParseWarnings:   warnings,
			RawProposalText: rawProposal,
		}
		state, stateErr := s.StateBuilder(initiativeName)
		validationErrors := validateExtractedProposal(*extracted, state, stateErr)
		revision.ValidationErrors = append([]string(nil), validationErrors...)
		msg.ProposalID = revision.ID
		revision.MessageIndex = round.AppendThreadMessage(msg)
		round.Proposals = append(round.Proposals, revision)
		if len(validationErrors) == 0 {
			round.CurrentProposalID = revision.ID
			round.NeedsRevision = false
			round.LastParseWarnings = nil
			round.LastValidationErrors = nil
		} else {
			round.CurrentProposalID = ""
			round.NeedsRevision = true
			round.LastParseWarnings = nil
			round.LastValidationErrors = append([]string(nil), validationErrors...)
		}
	} else {
		round.AppendThreadMessage(msg)
		// No extractable proposal: the round returns to the user with a
		// structured "revision needed" signal. The UI reads NeedsRevision
		// to render the ask-for-revision CTA; warnings surface why.
		round.NeedsRevision = true
		round.LastParseWarnings = append([]string(nil), warnings...)
		round.LastValidationErrors = nil
		if len(round.LastParseWarnings) == 0 {
			round.LastParseWarnings = []string{"agent output did not contain a parseable proposal JSON block"}
		}
	}
	round.Status = RoundStatusAwaitingUser
	round.UpdatedAt = now
	// Terminal advance — clear poll-failure tracking so a subsequent
	// continue/revise round starts with a clean counter.
	round.LastPollError = ""
	round.PollFailureCount = 0
	if err := s.store.SaveRound(round); err != nil {
		return Round{}, err
	}
	if relErr := s.lock.Release(initiativeName, round.RunID); relErr != nil {
		slog.Warn("feedback: release lock failed", "err", relErr, "initiative", initiativeName, "run_id", round.RunID)
	}
	return round, nil
}

func validateExtractedProposal(p proposals.Proposal, state proposals.CurrentState, stateErr error) []string {
	if stateErr != nil {
		return []string{fmt.Sprintf("build initiative state for proposal validation: %s", stateErr.Error())}
	}
	normalized, err := proposals.Normalize(p, state)
	if err != nil {
		return splitValidationErrors(fmt.Errorf("normalize proposal: %w", err))
	}
	if err := proposals.Validate(normalized, state); err != nil {
		return splitValidationErrors(err)
	}
	return nil
}

func splitValidationErrors(err error) []string {
	if err == nil {
		return nil
	}
	parts := make([]string, 0)
	for _, child := range flattenErrors(err) {
		msg := strings.TrimSpace(child.Error())
		if msg == "" || msg == proposals.ErrInvalidProposal.Error() {
			continue
		}
		parts = append(parts, msg)
	}
	if len(parts) == 0 {
		parts = append(parts, strings.TrimSpace(err.Error()))
	}
	return parts
}

func flattenErrors(err error) []error {
	if err == nil {
		return nil
	}
	type unwrapper interface{ Unwrap() []error }
	if multi, ok := err.(unwrapper); ok {
		out := make([]error, 0)
		for _, child := range multi.Unwrap() {
			out = append(out, flattenErrors(child)...)
		}
		return out
	}
	return []error{err}
}

func (s *Service) normalizeRoundProposalState(round Round) Round {
	if round.Status != RoundStatusAwaitingUser {
		return round
	}
	current := round.CurrentProposal()
	if current == nil {
		return round
	}
	if len(current.ValidationErrors) > 0 {
		round.CurrentProposalID = ""
		round.NeedsRevision = true
		round.LastValidationErrors = append([]string(nil), current.ValidationErrors...)
		return round
	}
	state, err := s.StateBuilder(round.InitiativeName)
	validationErrors := validateExtractedProposal(current.Proposal, state, err)
	if len(validationErrors) == 0 {
		return round
	}
	current.ValidationErrors = append([]string(nil), validationErrors...)
	for i := range round.Proposals {
		if round.Proposals[i].ID == current.ID {
			round.Proposals[i].ValidationErrors = append([]string(nil), validationErrors...)
			break
		}
	}
	round.CurrentProposalID = ""
	round.NeedsRevision = true
	round.LastValidationErrors = append([]string(nil), validationErrors...)
	return round
}
