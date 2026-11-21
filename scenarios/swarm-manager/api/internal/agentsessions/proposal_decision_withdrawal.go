package agentsessions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vrooli/api-core/operatordecision"
)

// proposalDecisionSubjectKey is the hub subject key of a proposal's decision:
// one per session and proposal, so the hub keeps one open decision for it and
// the owner can close it when the proposal leaves review. An id outside the hub's
// key rule is shortened behind a hash, so two proposals never share a key.
func proposalDecisionSubjectKey(sessionID, proposalID string) string {
	key := "proposal:" + strings.TrimSpace(sessionID) + "/" + strings.TrimSpace(proposalID)
	if len(key) <= 128 && operatordecision.SubjectKeyRule.MatchString(key) {
		return key
	}
	sum := sha256.Sum256([]byte(key))
	return "proposal:" + hex.EncodeToString(sum[:16])
}

// proposalWithdrawalReason says, in the operator's words, why a decision that
// was open for a proposal no longer needs an answer.
func proposalWithdrawalReason(proposal Proposal) string {
	switch {
	case proposal.Status == ProposalStatusApplied:
		return "This proposal has been applied."
	case proposal.Status == ProposalStatusRejected:
		return "This proposal has been rejected."
	case proposal.Status == ProposalStatusSuperseded:
		return "A newer proposal replaced this one."
	case proposal.Status == ProposalStatusFailed:
		return "This proposal failed validation and no longer needs your decision."
	case proposal.Status == ProposalStatusNeedsRevision || proposal.NeedsRevision:
		return "This proposal was sent back for revision and no longer needs your decision."
	default:
		return "This proposal is no longer waiting for your decision."
	}
}

// withdrawClosedProposalDecision closes the operator's open decision for a
// proposal that has left review, however it left: decided in swarm-manager,
// superseded by a newer proposal, or sent back for revision. A decision answered
// through the hub is already closed there and the withdrawal changes nothing.
// It reports whether the candidate was handled, so the caller skips it.
//
// A decision raised before subject keys has none to name, so it is closed by the
// request event id its frozen snapshot holds. A proposal whose publication never
// reached the published state raised nothing the owner can prove, and is left.
func (s *Service) withdrawClosedProposalDecision(ctx context.Context, owner ProposalDecisionPageStore, sessionID string, proposal Proposal) bool {
	publication := proposal.DecisionPublication
	if publication == nil || publication.State != ProposalDecisionPublicationPublished || publication.WithdrawnAt != "" || eligibleProposalDecision(proposal) {
		return false
	}
	next := *publication
	next.WithdrawalAttempts++
	reason := proposalWithdrawalReason(proposal)
	err := s.publishProposalDecisionWithdrawal(ctx, publication, reason)
	if err != nil {
		next.WithdrawalError = truncateRunes(err.Error(), 500)
		slog.Warn("agentsessions: proposal decision withdrawal publish failed", "session", sessionID, "proposal", proposal.ID, "err", err)
	} else {
		next.WithdrawnAt = nowRFC3339()
		next.WithdrawalError = ""
	}
	if _, receiptErr := owner.UpdateProposalDecisionPublication(sessionID, proposal.ID, publication.EventID, next); receiptErr != nil {
		slog.Warn("agentsessions: proposal decision withdrawal receipt failed", "session", sessionID, "proposal", proposal.ID, "err", receiptErr)
	}
	return true
}

func (s *Service) publishProposalDecisionWithdrawal(ctx context.Context, publication *ProposalDecisionPublication, reason string) error {
	if s.decisionPublisher == nil {
		return fmt.Errorf("proposal decision event publisher is unavailable")
	}
	if availability, ok := s.decisionPublisher.(interface{ Enabled() bool }); ok && !availability.Enabled() {
		return fmt.Errorf("proposal decision event publisher is disabled")
	}
	withdrawal := operatordecision.Withdrawal{Reason: reason}
	if subject, _ := publication.Event.Payload["subject_key"].(string); strings.TrimSpace(subject) != "" {
		withdrawal.SubjectKey = subject
	} else {
		withdrawal.RequesterRef = publication.EventID
	}
	return s.decisionPublisher.PublishDomainEvent(ctx, operatordecision.BuildWithdrawalEvent(ProposalDecisionSource, ProposalDecisionKind, nowUTC(), withdrawal))
}
