package main

import (
	"context"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/backlog"
)

// sessionDecisionProvider adapts durable ready mutation proposals into the
// backlog-owned next-action precedence seam. It deliberately returns only a
// count: rendering and application remain owned by agent sessions.
type sessionDecisionProvider struct{ sessions *agentsessions.Service }

func (p sessionDecisionProvider) PendingDecisions(ctx context.Context, item backlog.BacklogItem) (int, error) {
	if p.sessions == nil {
		return 0, nil
	}
	sessions, err := p.sessions.List(ctx, agentsessions.ListFilters{})
	if err != nil {
		return 0, err
	}
	ref := string(item.Kind) + "/" + item.Name
	count := 0
	for _, session := range sessions {
		for _, proposal := range session.Proposals {
			if proposal.Status == agentsessions.ProposalStatusReady && !proposal.NeedsRevision && proposal.Target != nil &&
				proposal.Target.Type == agentsessions.ContextBacklogItem && proposal.Target.Ref == ref {
				count++
			}
		}
	}
	return count, nil
}
