package main

import (
	"context"

	"swarm-manager/internal/agentsessions"
)

// sessionDecisionProvider adapts durable ready mutation proposals into the
// backlog-owned next-action precedence seam. It deliberately returns only
// counts: rendering and application remain owned by agent sessions.
type sessionDecisionProvider struct{ sessions *agentsessions.Service }

// readyDecisionCounts groups ready proposal counts by the entity domain they
// target. Both halves of the operator inbox — backlog items and goals — ask
// the same store the same question, so one pass answers both.
type readyDecisionCounts struct {
	items    map[string]int
	goals    map[string]int
	captures map[string]int
}

// countReadyDecisions counts ready, revision-free mutation proposals per
// target reference in a single pass over the session store. The seam is
// batch-shaped on purpose: the answer is a property of the whole store, so a
// per-entity variant would rescan every session once per entity.
func (p sessionDecisionProvider) countReadyDecisions(ctx context.Context) (readyDecisionCounts, error) {
	counts := readyDecisionCounts{items: map[string]int{}, goals: map[string]int{}, captures: map[string]int{}}
	if p.sessions == nil {
		return counts, nil
	}
	// Artifacts are never read while counting proposals, and hydrating them
	// per session is what made this scan expensive enough to matter.
	sessions, err := p.sessions.ListWithoutArtifacts(ctx, agentsessions.ListFilters{})
	if err != nil {
		return readyDecisionCounts{}, err
	}
	for _, session := range sessions {
		for _, proposal := range session.Proposals {
			if proposal.Status != agentsessions.ProposalStatusReady || proposal.NeedsRevision || proposal.Target == nil {
				continue
			}
			switch proposal.Target.Type {
			case agentsessions.ContextBacklogItem:
				counts.items[proposal.Target.Ref]++
			case agentsessions.ContextGoal:
				counts.goals[proposal.Target.Ref]++
			case agentsessions.ContextCapture:
				counts.captures[proposal.Target.Ref]++
			}
		}
	}
	return counts, nil
}

// PendingDecisionCounts satisfies the backlog decision seam for callers that
// resolve actions outside the feed projection.
func (p sessionDecisionProvider) PendingDecisionCounts(ctx context.Context) (map[string]int, error) {
	counts, err := p.countReadyDecisions(ctx)
	if err != nil {
		return nil, err
	}
	return counts.items, nil
}
