package agentsessions

import (
	"context"
	"fmt"
	"strings"
)

// CreateDecisionProposal persists a ready, operator-reviewed proposal without
// starting an agent run. It is used for server-generated migration reports;
// application remains a separate, explicitly authorized operation.
func (s *Service) CreateDecisionProposal(ctx context.Context, title, summary, payload string) (Session, Proposal, error) {
	session, err := s.Create(ctx, CreateRequest{Kind: KindSwarmOperations, Title: strings.TrimSpace(title)})
	if err != nil {
		return Session{}, Proposal{}, err
	}
	proposal, err := s.RecordProposal(ctx, session.ID, Proposal{Kind: ProposalGoalMigrationDisposition, Status: ProposalStatusReady, Summary: strings.TrimSpace(summary), PayloadJSON: payload})
	if err != nil {
		return Session{}, Proposal{}, err
	}
	session.Status = StatusProposalReady
	session.UpdatedAt = nowRFC3339()
	if err := s.saveSession(ctx, session); err != nil {
		return Session{}, Proposal{}, fmt.Errorf("persist migration proposal session: %w", err)
	}
	updated, err := s.loadSession(ctx, session.ID)
	return updated, proposal, err
}
