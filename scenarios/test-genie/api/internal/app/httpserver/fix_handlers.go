package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"test-genie/internal/fix"
)

// =============================================================================
// FIX HANDLERS - Agent-based test fixing
// =============================================================================

// spawnFixRequest is the request body for spawning a fix agent.
type spawnFixRequest struct {
	Phases []fix.PhaseInfo `json:"phases"`
}

// spawnFixResponse is the response for a spawned fix agent.
type spawnFixResponse struct {
	FixID  string     `json:"fixId"`
	RunID  string     `json:"runId,omitempty"`
	Tag    string     `json:"tag"`
	Status fix.Status `json:"status"`
	Error  string     `json:"error,omitempty"`
}

// handleSpawnFix spawns a fix agent for the specified scenario.
// POST /api/v1/scenarios/{name}/fix
func (s *Server) handleSpawnFix(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	var req spawnFixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if len(req.Phases) == 0 {
		s.writeError(w, http.StatusBadRequest, "at least one phase is required")
		return
	}

	// Check if agent-manager is available
	if !s.fixService.IsAgentAvailable(r.Context()) {
		s.writeError(w, http.StatusServiceUnavailable, "agent-manager is not available")
		return
	}

	result, err := s.fixService.Spawn(r.Context(), fix.SpawnRequest{
		ScenarioName: scenarioName,
		Phases:       req.Phases,
	})
	if err != nil {
		s.writeError(w, http.StatusConflict, err.Error())
		return
	}

	s.writeJSON(w, http.StatusCreated, spawnFixResponse{
		FixID:  result.FixID,
		RunID:  result.RunID,
		Tag:    result.Tag,
		Status: result.Status,
		Error:  result.Error,
	})
}

// fixResponse is the response for a fix record.
type fixResponse struct {
	ID           string          `json:"id"`
	ScenarioName string          `json:"scenarioName"`
	Phases       []fix.PhaseInfo `json:"phases"`
	Status       fix.Status      `json:"status"`
	RunID        string          `json:"runId,omitempty"`
	Tag          string          `json:"tag,omitempty"`
	StartedAt    string          `json:"startedAt"`
	CompletedAt  string          `json:"completedAt,omitempty"`
	Output       string          `json:"output,omitempty"`
	Error        string          `json:"error,omitempty"`
}

func recordToResponse(record *fix.Record) fixResponse {
	resp := fixResponse{
		ID:           record.ID,
		ScenarioName: record.ScenarioName,
		Phases:       record.Phases,
		Status:       record.Status,
		RunID:        record.RunID,
		Tag:          record.Tag,
		StartedAt:    record.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		Output:       record.Output,
		Error:        record.Error,
	}
	if record.CompletedAt != nil {
		resp.CompletedAt = record.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return resp
}

// handleGetFix returns the status of a specific fix.
// GET /api/v1/scenarios/{name}/fixes/{id}
func (s *Server) handleGetFix(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fixID := vars["id"]

	if fixID == "" {
		s.writeError(w, http.StatusBadRequest, "fix ID is required")
		return
	}

	record, ok := s.fixService.Get(fixID)
	if !ok {
		s.writeError(w, http.StatusNotFound, "fix not found")
		return
	}

	s.writeJSON(w, http.StatusOK, recordToResponse(record))
}

// handleListFixes returns recent fixes for a scenario.
// GET /api/v1/scenarios/{name}/fixes
func (s *Server) handleListFixes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	records := s.fixService.ListByScenario(scenarioName, 10)
	items := make([]fixResponse, 0, len(records))
	for _, record := range records {
		items = append(items, recordToResponse(record))
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"count": len(items),
	})
}

// handleGetActiveFix returns the active fix for a scenario, if any.
// GET /api/v1/scenarios/{name}/fixes/active
func (s *Server) handleGetActiveFix(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	record := s.fixService.GetActiveForScenario(scenarioName)
	if record == nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"active": false,
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"active": true,
		"fix":    recordToResponse(record),
	})
}

// handleStopFix stops a running fix.
// POST /api/v1/scenarios/{name}/fixes/{id}/stop
func (s *Server) handleStopFix(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fixID := vars["id"]

	if fixID == "" {
		s.writeError(w, http.StatusBadRequest, "fix ID is required")
		return
	}

	if err := s.fixService.Stop(r.Context(), fixID); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "fix stopped",
	})
}
