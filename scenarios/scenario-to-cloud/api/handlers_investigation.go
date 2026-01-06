package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"scenario-to-cloud/domain"
)

// handleInvestigateDeployment triggers a new investigation for a failed deployment.
// POST /api/v1/deployments/{id}/investigate
func (s *Server) handleInvestigateDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	if deploymentID == "" {
		http.Error(w, "deployment ID is required", http.StatusBadRequest)
		return
	}

	// Check if investigation service is available
	if s.investigationSvc == nil {
		http.Error(w, "investigation service not available", http.StatusServiceUnavailable)
		return
	}

	// Parse request body
	var req domain.CreateInvestigationRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Trigger investigation
	inv, err := s.investigationSvc.TriggerInvestigation(r.Context(), TriggerInvestigationRequest{
		DeploymentID: deploymentID,
		AutoFix:      req.AutoFix,
		Note:         req.Note,
	})
	if err != nil {
		// Check for specific error types
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "not found"):
			http.Error(w, err.Error(), http.StatusNotFound)
		case strings.Contains(errMsg, "already in progress"):
			http.Error(w, err.Error(), http.StatusConflict)
		case strings.Contains(errMsg, "not available"):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		case strings.Contains(errMsg, "expected failed"):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return the investigation
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"investigation": inv,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleListInvestigations returns all investigations for a deployment.
// GET /api/v1/deployments/{id}/investigations
func (s *Server) handleListInvestigations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	if deploymentID == "" {
		http.Error(w, "deployment ID is required", http.StatusBadRequest)
		return
	}

	if s.investigationSvc == nil {
		http.Error(w, "investigation service not available", http.StatusServiceUnavailable)
		return
	}

	// Get limit from query params (default 10)
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		var l int
		if _, err := parseIntParam(limitStr, &l); err == nil && l > 0 {
			limit = l
		}
	}

	investigations, err := s.investigationSvc.ListInvestigations(r.Context(), deploymentID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to summaries for list response
	summaries := make([]domain.InvestigationSummary, 0, len(investigations))
	for _, inv := range investigations {
		summaries = append(summaries, inv.ToSummary())
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"investigations": summaries,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleGetInvestigation returns a single investigation by ID.
// GET /api/v1/deployments/{id}/investigations/{invId}
func (s *Server) handleGetInvestigation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]
	invID := vars["invId"]

	if deploymentID == "" || invID == "" {
		http.Error(w, "deployment ID and investigation ID are required", http.StatusBadRequest)
		return
	}

	if s.investigationSvc == nil {
		http.Error(w, "investigation service not available", http.StatusServiceUnavailable)
		return
	}

	inv, err := s.investigationSvc.GetInvestigation(r.Context(), deploymentID, invID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if inv == nil {
		http.Error(w, "investigation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"investigation": inv,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleStopInvestigation stops a running investigation.
// POST /api/v1/deployments/{id}/investigations/{invId}/stop
func (s *Server) handleStopInvestigation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]
	invID := vars["invId"]

	if deploymentID == "" || invID == "" {
		http.Error(w, "deployment ID and investigation ID are required", http.StatusBadRequest)
		return
	}

	if s.investigationSvc == nil {
		http.Error(w, "investigation service not available", http.StatusServiceUnavailable)
		return
	}

	if err := s.investigationSvc.StopInvestigation(r.Context(), deploymentID, invID); err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "not found"):
			http.Error(w, err.Error(), http.StatusNotFound)
		case strings.Contains(errMsg, "not running"):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Investigation stopped",
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleApplyFixes applies selected fixes from a completed investigation.
// POST /api/v1/deployments/{id}/investigations/{invId}/apply-fixes
func (s *Server) handleApplyFixes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]
	invID := vars["invId"]

	if deploymentID == "" || invID == "" {
		http.Error(w, "deployment ID and investigation ID are required", http.StatusBadRequest)
		return
	}

	if s.investigationSvc == nil {
		http.Error(w, "investigation service not available", http.StatusServiceUnavailable)
		return
	}

	// Parse request body
	var req domain.ApplyFixesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Apply fixes
	fixInv, err := s.investigationSvc.ApplyFixes(r.Context(), ApplyFixesRequest{
		InvestigationID: invID,
		DeploymentID:    deploymentID,
		Immediate:       req.Immediate,
		Permanent:       req.Permanent,
		Prevention:      req.Prevention,
		Note:            req.Note,
	})
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "not found"):
			http.Error(w, err.Error(), http.StatusNotFound)
		case strings.Contains(errMsg, "already in progress"):
			http.Error(w, err.Error(), http.StatusConflict)
		case strings.Contains(errMsg, "not available"):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		case strings.Contains(errMsg, "expected completed"):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case strings.Contains(errMsg, "no findings"):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case strings.Contains(errMsg, "at least one"):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"investigation": fixInv,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleCheckAgentManagerStatus returns the status of agent-manager.
// GET /api/v1/agent-manager/status
func (s *Server) handleCheckAgentManagerStatus(w http.ResponseWriter, r *http.Request) {
	if s.agentSvc == nil {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"enabled":   false,
			"available": false,
			"message":   "Agent manager integration is not configured",
		}); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
		return
	}

	available := s.agentSvc.IsAvailable(r.Context())

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled":   s.agentSvc.IsEnabled(),
		"available": available,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// parseIntParam is a helper to parse integer query parameters.
func parseIntParam(s string, result *int) (bool, error) {
	if s == "" {
		return false, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return false, err
	}
	*result = n
	return true, nil
}
