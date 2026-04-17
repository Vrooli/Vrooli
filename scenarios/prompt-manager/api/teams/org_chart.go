package teams

import (
	"context"
	"fmt"
	"prompt-manager/store"
)

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
