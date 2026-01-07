package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"test-genie/internal/requirementsimprove"
)

// =============================================================================
// REQUIREMENTS IMPROVE HANDLERS - Agent-based requirements improvement
// =============================================================================

// spawnRequirementsImproveRequest is the request body for spawning a requirements improve agent.
type spawnRequirementsImproveRequest struct {
	Requirements []requirementsimprove.RequirementInfo `json:"requirements"`
	ActionType   requirementsimprove.ActionType        `json:"actionType"`
}

// spawnRequirementsImproveResponse is the response for a spawned requirements improve agent.
type spawnRequirementsImproveResponse struct {
	ImproveID string                      `json:"improveId"`
	RunID     string                      `json:"runId,omitempty"`
	Tag       string                      `json:"tag"`
	Status    requirementsimprove.Status  `json:"status"`
	Error     string                      `json:"error,omitempty"`
}

// handleSpawnRequirementsImprove spawns a requirements improve agent for the specified scenario.
// POST /api/v1/scenarios/{name}/requirements/improve
func (s *Server) handleSpawnRequirementsImprove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	var req spawnRequirementsImproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if len(req.Requirements) == 0 {
		s.writeError(w, http.StatusBadRequest, "at least one requirement is required")
		return
	}

	// Default action type to write_tests
	if req.ActionType == "" {
		req.ActionType = requirementsimprove.ActionWriteTests
	}

	// Check if agent-manager is available
	if !s.requirementsImproveService.IsAgentAvailable(r.Context()) {
		s.writeError(w, http.StatusServiceUnavailable, "agent-manager is not available")
		return
	}

	result, err := s.requirementsImproveService.Spawn(r.Context(), requirementsimprove.SpawnRequest{
		ScenarioName: scenarioName,
		Requirements: req.Requirements,
		ActionType:   req.ActionType,
	})
	if err != nil {
		s.writeError(w, http.StatusConflict, err.Error())
		return
	}

	s.writeJSON(w, http.StatusCreated, spawnRequirementsImproveResponse{
		ImproveID: result.ImproveID,
		RunID:     result.RunID,
		Tag:       result.Tag,
		Status:    result.Status,
		Error:     result.Error,
	})
}

// requirementsImproveResponse is the response for a requirements improve record.
type requirementsImproveResponse struct {
	ID           string                                `json:"id"`
	ScenarioName string                                `json:"scenarioName"`
	Requirements []requirementsimprove.RequirementInfo `json:"requirements"`
	ActionType   requirementsimprove.ActionType        `json:"actionType"`
	Status       requirementsimprove.Status            `json:"status"`
	RunID        string                                `json:"runId,omitempty"`
	Tag          string                                `json:"tag,omitempty"`
	StartedAt    string                                `json:"startedAt"`
	CompletedAt  string                                `json:"completedAt,omitempty"`
	Output       string                                `json:"output,omitempty"`
	Error        string                                `json:"error,omitempty"`
}

func requirementsImproveRecordToResponse(record *requirementsimprove.Record) requirementsImproveResponse {
	resp := requirementsImproveResponse{
		ID:           record.ID,
		ScenarioName: record.ScenarioName,
		Requirements: record.Requirements,
		ActionType:   record.ActionType,
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

// handleGetRequirementsImprove returns the status of a specific requirements improve operation.
// GET /api/v1/scenarios/{name}/requirements/improve/{id}
func (s *Server) handleGetRequirementsImprove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	improveID := vars["id"]

	if improveID == "" {
		s.writeError(w, http.StatusBadRequest, "improve ID is required")
		return
	}

	record, ok := s.requirementsImproveService.Get(improveID)
	if !ok {
		s.writeError(w, http.StatusNotFound, "improve not found")
		return
	}

	s.writeJSON(w, http.StatusOK, requirementsImproveRecordToResponse(record))
}

// handleListRequirementsImproves returns recent requirements improve operations for a scenario.
// GET /api/v1/scenarios/{name}/requirements/improve
func (s *Server) handleListRequirementsImproves(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	records := s.requirementsImproveService.ListByScenario(scenarioName, 10)
	items := make([]requirementsImproveResponse, 0, len(records))
	for _, record := range records {
		items = append(items, requirementsImproveRecordToResponse(record))
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"count": len(items),
	})
}

// handleGetActiveRequirementsImprove returns the active requirements improve for a scenario, if any.
// GET /api/v1/scenarios/{name}/requirements/improve/active
func (s *Server) handleGetActiveRequirementsImprove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	record := s.requirementsImproveService.GetActiveForScenario(scenarioName)
	if record == nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"active": false,
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"active":  true,
		"improve": requirementsImproveRecordToResponse(record),
	})
}

// handleStopRequirementsImprove stops a running requirements improve operation.
// POST /api/v1/scenarios/{name}/requirements/improve/{id}/stop
func (s *Server) handleStopRequirementsImprove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	improveID := vars["id"]

	if improveID == "" {
		s.writeError(w, http.StatusBadRequest, "improve ID is required")
		return
	}

	if err := s.requirementsImproveService.Stop(r.Context(), improveID); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "improve stopped",
	})
}
