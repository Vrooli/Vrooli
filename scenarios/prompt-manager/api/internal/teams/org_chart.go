package teams

import (
	"context"

	"prompt-manager/internal/store"
)

// effectiveManagedTeamEdges projects the standing supervisor's canonical
// effort scope into team relationships. Delivery teams own their effortRefs;
// the standing supervision team owns the policy to watch those efforts. The
// projection is read-only and therefore disappears automatically when a team
// is archived or no longer declares an effort. Explicit org edges remain
// supported for other stable team-to-team relationships.
func (h *Handlers) effectiveManagedTeamEdges(ctx context.Context, managerID string) []store.ManagedTeamEdge {
	org, _ := h.teamStore.GetOrgChart(ctx, managerID)
	if org == nil {
		org = &store.OrgChart{TeamID: managerID}
	}
	edges := append([]store.ManagedTeamEdge(nil), org.ManagedTeamEdges...)
	manager, err := h.teamStore.Get(ctx, managerID)
	if err != nil || manager.Archived || manager.Purpose != "supervision" {
		return edges
	}
	known := make(map[string]bool, len(edges))
	for _, edge := range edges {
		known[edge.ManagedTeamID] = true
	}
	teams, err := h.teamStore.List(ctx)
	if err != nil {
		return edges
	}
	for _, team := range teams {
		if team.ID == managerID || team.Archived || team.Purpose != "delivery" || len(team.EffortRefs) == 0 || known[team.ID] {
			continue
		}
		edges = append(edges, store.ManagedTeamEdge{
			ManagerTeamID: managerID,
			ManagedTeamID: team.ID,
			Relationship:  "supervises",
			AuthorityRef:  "agent-manager:effort-board",
			Status:        "active",
		})
	}
	return edges
}
