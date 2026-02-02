package teams

import (
	"encoding/json"
	"net/http"
	"strings"

	"prompt-manager/store"

	"github.com/gorilla/mux"
)

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
