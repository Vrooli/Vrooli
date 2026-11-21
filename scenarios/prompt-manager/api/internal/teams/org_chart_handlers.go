package teams

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"prompt-manager/internal/store"

	"github.com/gorilla/mux"
)

func (h *Handlers) ReadOrgChart(ctx context.Context, id string) (OrgChartResponse, error) {
	if _, err := h.teamStore.Get(ctx, id); err != nil {
		return OrgChartResponse{}, fmt.Errorf("Team not found")
	}
	org, err := h.teamStore.GetOrgChart(ctx, id)
	if err != nil {
		return OrgChartResponse{}, err
	}
	resp := OrgChartResponse{TeamID: id, Edges: make([]OrgEdgeDTO, 0, len(org.Edges)), ManagedTeamEdges: make([]ManagedTeamEdgeDTO, 0)}
	for _, edge := range org.Edges {
		resp.Edges = append(resp.Edges, orgEdgeToDTO(edge))
	}
	for _, edge := range h.effectiveManagedTeamEdges(ctx, id) {
		resp.ManagedTeamEdges = append(resp.ManagedTeamEdges, managedTeamEdgeToDTO(edge))
	}
	return resp, nil
}

func (h *Handlers) WriteOrgChart(ctx context.Context, id string, req SetOrgChartRequest) (OrgChartResponse, error) {
	if _, err := h.teamStore.Get(ctx, id); err != nil {
		return OrgChartResponse{}, fmt.Errorf("Team not found")
	}
	if err := h.validateOrgChartEdges(ctx, id, req.Edges); err != nil {
		return OrgChartResponse{}, err
	}
	if err := h.validateManagedTeamEdges(ctx, id, req.ManagedTeamEdges); err != nil {
		return OrgChartResponse{}, err
	}
	org, err := h.teamStore.GetOrgChart(ctx, id)
	if err != nil {
		org = &store.OrgChart{TeamID: id}
	}
	org.TeamID = id
	org.Edges = make([]store.OrgEdge, 0, len(req.Edges))
	for _, edge := range req.Edges {
		org.Edges = append(org.Edges, orgEdgeFromDTO(edge))
	}
	if req.ManagedTeamEdges != nil {
		org.ManagedTeamEdges = make([]store.ManagedTeamEdge, 0, len(req.ManagedTeamEdges))
		for _, edge := range req.ManagedTeamEdges {
			org.ManagedTeamEdges = append(org.ManagedTeamEdges, managedTeamEdgeFromDTO(edge, id))
		}
	}
	if err := h.teamStore.SetOrgChart(ctx, id, org); err != nil {
		return OrgChartResponse{}, err
	}
	return OrgChartResponse{TeamID: id, Edges: req.Edges, ManagedTeamEdges: req.ManagedTeamEdges}, nil
}

func (h *Handlers) UpdateOrgEdge(ctx context.Context, teamID, reportID string, req UpdateOrgEdgeRequest) (OrgChartResponse, error) {
	if reportID == "" {
		return OrgChartResponse{}, fmt.Errorf("reportId is required")
	}
	if req.ManagerAgentID == "" {
		return OrgChartResponse{}, fmt.Errorf("managerAgentId is required")
	}
	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		return OrgChartResponse{}, fmt.Errorf("Team not found")
	}
	org, err := h.teamStore.GetOrgChart(ctx, teamID)
	if err != nil {
		return OrgChartResponse{}, err
	}
	edges := make([]OrgEdgeDTO, 0, len(org.Edges)+1)
	for _, edge := range org.Edges {
		if edge.ReportAgentID != reportID {
			edges = append(edges, orgEdgeToDTO(edge))
		}
	}
	edges = append(edges, OrgEdgeDTO{ManagerAgentID: req.ManagerAgentID, ReportAgentID: reportID})
	if err := h.validateOrgChartEdges(ctx, teamID, edges); err != nil {
		return OrgChartResponse{}, err
	}
	org.Edges = make([]store.OrgEdge, 0, len(edges))
	for _, edge := range edges {
		org.Edges = append(org.Edges, orgEdgeFromDTO(edge))
	}
	if err := h.teamStore.SetOrgChart(ctx, teamID, org); err != nil {
		return OrgChartResponse{}, err
	}
	return h.ReadOrgChart(ctx, teamID)
}

func (h *Handlers) DeleteOrgEdge(ctx context.Context, teamID, reportID string) (OrgChartResponse, error) {
	if reportID == "" {
		return OrgChartResponse{}, fmt.Errorf("reportId is required")
	}
	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		return OrgChartResponse{}, fmt.Errorf("Team not found")
	}
	org, err := h.teamStore.GetOrgChart(ctx, teamID)
	if err != nil {
		return OrgChartResponse{}, err
	}
	edges := make([]OrgEdgeDTO, 0, len(org.Edges))
	found := false
	for _, edge := range org.Edges {
		if edge.ReportAgentID == reportID {
			found = true
			continue
		}
		edges = append(edges, orgEdgeToDTO(edge))
	}
	if !found {
		return OrgChartResponse{}, fmt.Errorf("reporting edge not found")
	}
	if err := h.validateOrgChartEdges(ctx, teamID, edges); err != nil {
		return OrgChartResponse{}, err
	}
	org.Edges = make([]store.OrgEdge, 0, len(edges))
	for _, edge := range edges {
		org.Edges = append(org.Edges, orgEdgeFromDTO(edge))
	}
	if err := h.teamStore.SetOrgChart(ctx, teamID, org); err != nil {
		return OrgChartResponse{}, err
	}
	return h.ReadOrgChart(ctx, teamID)
}

// UpdateOrgChartEdge handles PUT /teams/{id}/org/edges/{reportId} - sets a single reporting relationship.
func (h *Handlers) UpdateOrgChartEdge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	reportID := vars["reportId"]

	if reportID == "" {
		http.Error(w, "reportId is required", http.StatusBadRequest)
		return
	}

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	var req UpdateOrgEdgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.ManagerAgentID == "" {
		http.Error(w, "managerAgentId is required", http.StatusBadRequest)
		return
	}

	org, err := h.teamStore.GetOrgChart(ctx, teamID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updatedEdges := make([]OrgEdgeDTO, 0, len(org.Edges)+1)
	for _, edge := range org.Edges {
		if edge.ReportAgentID == reportID {
			continue
		}
		updatedEdges = append(updatedEdges, orgEdgeToDTO(edge))
	}
	updatedEdges = append(updatedEdges, OrgEdgeDTO{
		ManagerAgentID: req.ManagerAgentID,
		ReportAgentID:  reportID,
	})

	if err := h.validateOrgChartEdges(ctx, teamID, updatedEdges); err != nil {
		if isValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	org.Edges = make([]store.OrgEdge, 0, len(updatedEdges))
	for _, edge := range updatedEdges {
		org.Edges = append(org.Edges, orgEdgeFromDTO(edge))
	}

	if err := h.teamStore.SetOrgChart(ctx, teamID, org); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(OrgEdgeDTO{
		ManagerAgentID: req.ManagerAgentID,
		ReportAgentID:  reportID,
	})
}

// DeleteOrgChartEdge handles DELETE /teams/{id}/org/edges/{reportId} - removes a reporting relationship.
func (h *Handlers) DeleteOrgChartEdge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	reportID := vars["reportId"]

	if reportID == "" {
		http.Error(w, "reportId is required", http.StatusBadRequest)
		return
	}

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	org, err := h.teamStore.GetOrgChart(ctx, teamID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updatedEdges := make([]OrgEdgeDTO, 0, len(org.Edges))
	found := false
	for _, edge := range org.Edges {
		if edge.ReportAgentID == reportID {
			found = true
			continue
		}
		updatedEdges = append(updatedEdges, orgEdgeToDTO(edge))
	}

	if !found {
		http.Error(w, "reporting edge not found", http.StatusNotFound)
		return
	}

	if err := h.validateOrgChartEdges(ctx, teamID, updatedEdges); err != nil {
		if isValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	org.Edges = make([]store.OrgEdge, 0, len(updatedEdges))
	for _, edge := range updatedEdges {
		org.Edges = append(org.Edges, orgEdgeFromDTO(edge))
	}

	if err := h.teamStore.SetOrgChart(ctx, teamID, org); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
