package agentsessions

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vrooli/api-core/operatordecision"
)

const proposalDecisionPageSize = 25

// ReconcileProposalDecisionPage processes at most one stable-key page from the
// routed proposal owner. Callers retain Next in memory and reset it to the zero
// cursor when HasMore is false; this yields bounded, fair passes without a
// second durable queue or cursor table.
func (s *Service) ReconcileProposalDecisionPage(ctx context.Context, after ProposalDecisionCursor, limit int) (ProposalDecisionPage, error) {
	store, err := s.storeFor(ctx)
	if err != nil {
		return ProposalDecisionPage{}, err
	}
	owner, ok := store.(ProposalDecisionPageStore)
	if !ok {
		return ProposalDecisionPage{}, fmt.Errorf("agent-session store does not support proposal decision reconciliation")
	}
	if limit <= 0 {
		limit = proposalDecisionPageSize
	}
	page, err := owner.ListProposalDecisionPage(after, limit)
	if err != nil {
		return ProposalDecisionPage{}, err
	}
	for _, issue := range page.Issues {
		slog.Warn("agentsessions: proposal decision record remains unresolved",
			"session", issue.SessionID, "proposal_file", issue.ProposalFile, "reason", issue.Reason)
	}
	for _, candidate := range page.Candidates {
		proposal := candidate.Proposal
		if !eligibleProposalDecision(proposal) {
			continue
		}
		if proposal.DecisionPublication == nil {
			// A row that predates the immutable snapshot cannot be rebuilt under
			// current producer code: doing so could create a second ask. Preserve
			// the uncertainty as a durable owner state and wait for exact event
			// proof or an explicit migration.
			_, markErr := owner.UpdateProposalDecisionPublication(candidate.SessionID, proposal.ID, "", ProposalDecisionPublication{
				Version:   1,
				State:     ProposalDecisionPublicationLegacyUnresolved,
				LastError: "legacy ready proposal has no frozen event snapshot; publication cannot be inferred",
			})
			if markErr != nil {
				slog.Warn("agentsessions: legacy proposal decision remains unresolved", "session", candidate.SessionID, "proposal", proposal.ID, "err", markErr)
			}
			continue
		}
		publication := proposal.DecisionPublication
		if publication.State != ProposalDecisionPublicationPending || publication.EventID == "" {
			continue
		}
		if got := operatordecision.EventID(publication.Event); got != publication.EventID {
			slog.Warn("agentsessions: frozen proposal decision snapshot does not match its event_id", "session", candidate.SessionID, "proposal", proposal.ID)
			continue
		}
		s.emitProposalDecisionRequested(ctx, candidate.SessionID, proposal)
	}
	return page, nil
}
