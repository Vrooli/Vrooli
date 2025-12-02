package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/shared"
)

type suiteExecutionPayload struct {
	ScenarioName   string   `json:"scenarioName"`
	SuiteRequestID string   `json:"suiteRequestId"`
	Preset         string   `json:"preset"`
	Phases         []string `json:"phases"`
	Skip           []string `json:"skip"`
	FailFast       bool     `json:"failFast"`
}

func (s *Server) handleExecuteSuite(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload suiteExecutionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	scenario := strings.TrimSpace(payload.ScenarioName)
	if scenario == "" {
		s.writeError(w, http.StatusBadRequest, "scenarioName is required")
		return
	}

	execRequest := orchestrator.SuiteExecutionRequest{
		ScenarioName: scenario,
		Preset:       strings.TrimSpace(payload.Preset),
		Phases:       payload.Phases,
		Skip:         payload.Skip,
		FailFast:     payload.FailFast,
	}

	var suiteRequestID *uuid.UUID
	if id := strings.TrimSpace(payload.SuiteRequestID); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "suiteRequestId must be a valid UUID")
			return
		}
		suiteRequestID = &parsed
	}

	if s.executionSvc == nil {
		s.writeError(w, http.StatusInternalServerError, "execution service unavailable")
		return
	}

	result, err := s.executionSvc.Execute(r.Context(), execution.SuiteExecutionInput{
		Request:        execRequest,
		SuiteRequestID: suiteRequestID,
	})
	if err != nil {
		if errors.Is(err, execution.ErrSuiteRequestNotFound) {
			s.writeError(w, http.StatusNotFound, "suite request not found")
			return
		}
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			s.writeError(w, http.StatusBadRequest, vErr.Error())
			return
		}
		s.log("suite execution failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "suite execution failed")
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	scenario := strings.TrimSpace(params.Get("scenario"))
	limit := orchestrator.MaxExecutionHistory
	if raw := strings.TrimSpace(params.Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if raw := strings.TrimSpace(params.Get("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	executions, err := s.executionHistory.List(r.Context(), scenario, limit, offset)
	if err != nil {
		s.log("listing executions failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load execution history")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": executions,
		"count": len(executions),
	})
}

func (s *Server) handleGetExecution(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	rawID := strings.TrimSpace(params["id"])
	if rawID == "" {
		s.writeError(w, http.StatusBadRequest, "execution id is required")
		return
	}
	executionID, err := uuid.Parse(rawID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "execution id must be a valid UUID")
		return
	}

	result, err := s.executionHistory.Get(r.Context(), executionID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "execution not found")
			return
		}
		s.log("fetching execution failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load execution")
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}
