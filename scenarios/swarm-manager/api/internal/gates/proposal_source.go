package gates

import (
	"context"
	"strings"

	"swarm-manager/internal/agentsessions"
)

// ProposalSource projects ready mutation-list proposals without owning their
// decision flow. It intentionally exposes one gate per target, with Count
// reflecting all ready proposals from every session.
type ProposalSource struct{ Store agentsessions.Store }

func (ProposalSource) Name() string { return "proposal" }

func (s ProposalSource) Enumerate(_ context.Context) ([]Gate, error) {
	if s.Store == nil {
		return nil, nil
	}
	sessions, err := s.Store.ListSessions(agentsessions.ListFilters{})
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	titles := map[string]string{}
	for _, session := range sessions {
		for _, proposal := range session.Proposals {
			if proposal.Kind != agentsessions.ProposalMutationList || proposal.Status != agentsessions.ProposalStatusReady || proposal.Target == nil {
				continue
			}
			if proposal.Target.Type == agentsessions.ContextGoal {
				name := strings.TrimSpace(proposal.Target.Ref)
				if name == "" {
					continue
				}
				key := "goal/" + name
				counts[key]++
				titles[key] = proposal.Target.Name
				continue
			}
			if proposal.Target.Type != agentsessions.ContextBacklogItem {
				continue
			}
			ref := strings.TrimSpace(proposal.Target.Ref)
			if ref == "" {
				continue
			}
			counts[ref]++
			titles[ref] = proposal.Target.Name
		}
	}
	out := make([]Gate, 0, len(counts))
	for ref, count := range counts {
		if strings.HasPrefix(ref, "goal/") {
			name := strings.TrimPrefix(ref, "goal/")
			out = append(out, Gate{ID: GateID(KindProposal, "goal", name), Kind: KindProposal, OwnerType: "goal", OwnerName: name, OwnerTitle: titles[ref], Count: count})
			continue
		}
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			continue
		}
		out = append(out, Gate{ID: GateID(KindProposal, "backlog", ref), Kind: KindProposal, OwnerType: "backlog", OwnerKind: parts[0], OwnerName: parts[1], OwnerTitle: titles[ref], Count: count})
	}
	return out, nil
}
