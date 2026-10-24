package teams

import (
	"context"
	"fmt"
	"strings"

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

func (h *Handlers) validateManagedTeamEdges(ctx context.Context, managerID string, edges []ManagedTeamEdgeDTO) error {
	teams, err := h.teamStore.List(ctx)
	if err != nil {
		return err
	}
	known := make(map[string]bool, len(teams))
	for _, team := range teams {
		known[team.ID] = true
	}
	managerByManaged := make(map[string]string, len(edges))
	for _, edge := range edges {
		if edge.ManagerTeamID == "" {
			edge.ManagerTeamID = managerID
		}
		if edge.ManagerTeamID != managerID {
			return newValidationError("managed team edges must belong to the requested manager team")
		}
		if !known[edge.ManagedTeamID] {
			return newValidationError(fmt.Sprintf("managedTeamId %s does not exist", edge.ManagedTeamID))
		}
		if edge.ManagedTeamID == managerID {
			return newValidationError("a team cannot manage itself")
		}
		if existing, ok := managerByManaged[edge.ManagedTeamID]; ok && existing == managerID {
			return newValidationError(fmt.Sprintf("duplicate managed team edge for %s", edge.ManagedTeamID))
		}
		managerByManaged[edge.ManagedTeamID] = managerID
	}
	// A team-level supervision graph must remain acyclic. Include existing
	// relationships from other manager teams when checking the replacement.
	parents := make(map[string]string)
	for _, team := range teams {
		org, readErr := h.teamStore.GetOrgChart(ctx, team.ID)
		if readErr != nil {
			continue
		}
		for _, edge := range org.ManagedTeamEdges {
			if edge.ManagerTeamID != managerID {
				parents[edge.ManagedTeamID] = edge.ManagerTeamID
			}
		}
	}
	for _, edge := range edges {
		parents[edge.ManagedTeamID] = managerID
	}
	for node := range parents {
		seen := map[string]bool{}
		for current := node; current != ""; current = parents[current] {
			if seen[current] {
				return newValidationError("managed team graph contains a cycle")
			}
			seen[current] = true
		}
	}
	return nil
}

func managedTeamEdgeToDTO(edge store.ManagedTeamEdge) ManagedTeamEdgeDTO {
	return ManagedTeamEdgeDTO{ManagerTeamID: edge.ManagerTeamID, ManagedTeamID: edge.ManagedTeamID, Relationship: edge.Relationship, AuthorityRef: edge.AuthorityRef, Status: edge.Status}
}

func managedTeamEdgeFromDTO(edge ManagedTeamEdgeDTO, managerID string) store.ManagedTeamEdge {
	if strings.TrimSpace(edge.ManagerTeamID) == "" {
		edge.ManagerTeamID = managerID
	}
	if strings.TrimSpace(edge.Relationship) == "" {
		edge.Relationship = "supervises"
	}
	if strings.TrimSpace(edge.Status) == "" {
		edge.Status = "active"
	}
	return store.ManagedTeamEdge{ManagerTeamID: edge.ManagerTeamID, ManagedTeamID: edge.ManagedTeamID, Relationship: edge.Relationship, AuthorityRef: edge.AuthorityRef, Status: edge.Status}
}

func (h *Handlers) validateOrgChartEdges(ctx context.Context, teamID string, edges []OrgEdgeDTO) error {
	memberSet, err := h.teamMemberSet(ctx, teamID)
	if err != nil {
		return err
	}

	managerByReport := make(map[string]string, len(edges))
	for _, edge := range edges {
		if err := validateMemberExists(memberSet, edge.ManagerAgentID, "managerAgentId"); err != nil {
			return err
		}
		if err := validateMemberExists(memberSet, edge.ReportAgentID, "reportAgentId"); err != nil {
			return err
		}
		if edge.ManagerAgentID == edge.ReportAgentID {
			return newValidationError("managerAgentId cannot equal reportAgentId")
		}
		if existingManager, ok := managerByReport[edge.ReportAgentID]; ok {
			if existingManager == edge.ManagerAgentID {
				return newValidationError(fmt.Sprintf("duplicate edge for reportAgentId %s", edge.ReportAgentID))
			}
			return newValidationError(fmt.Sprintf("reportAgentId %s already reports to %s", edge.ReportAgentID, existingManager))
		}
		managerByReport[edge.ReportAgentID] = edge.ManagerAgentID
	}

	if hasOrgChartCycle(managerByReport) {
		return newValidationError("org chart contains a reporting cycle")
	}

	return nil
}

func hasOrgChartCycle(managerByReport map[string]string) bool {
	visited := make(map[string]bool, len(managerByReport))
	stack := make(map[string]bool, len(managerByReport))

	var visit func(node string) bool
	visit = func(node string) bool {
		if stack[node] {
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		stack[node] = true

		if manager, ok := managerByReport[node]; ok {
			if visit(manager) {
				return true
			}
		}
		stack[node] = false
		return false
	}

	for reportID := range managerByReport {
		if visit(reportID) {
			return true
		}
	}

	return false
}

func orgEdgeToDTO(edge store.OrgEdge) OrgEdgeDTO {
	return OrgEdgeDTO{
		ManagerAgentID: edge.ManagerAgentID,
		ReportAgentID:  edge.ReportAgentID,
	}
}

func orgEdgeFromDTO(edge OrgEdgeDTO) store.OrgEdge {
	return store.OrgEdge{
		ManagerAgentID: edge.ManagerAgentID,
		ReportAgentID:  edge.ReportAgentID,
	}
}
