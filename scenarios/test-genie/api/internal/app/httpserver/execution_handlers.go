package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/shared"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type suiteExecutionPayload struct {
	ScenarioName   string   `json:"scenarioName"`
	SuiteRequestID string   `json:"suiteRequestId"`
	Preset         string   `json:"preset"`
	Phases         []string `json:"phases"`
	Skip           []string `json:"skip"`
	FailFast       bool     `json:"failFast"`
	UIURL          string   `json:"uiUrl"`
	APIURL         string   `json:"apiUrl"`
	BrowserlessURL string   `json:"browserlessUrl"`
	// ScenarioPath overrides scenario directory resolution. Set by the CLI
	// when running inside a sandboxed agent. When empty, the API resolves
	// the path from ScenarioName using VROOLI_ROOT.
	ScenarioPath string `json:"scenarioPath"`
}

func decodeSuiteExecutionInput(r *http.Request) (execution.SuiteExecutionInput, error) {
	defer r.Body.Close()

	var payload suiteExecutionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return execution.SuiteExecutionInput{}, shared.NewValidationError("invalid JSON payload")
	}
	return buildSuiteExecutionInput(payload)
}

func buildSuiteExecutionInput(payload suiteExecutionPayload) (execution.SuiteExecutionInput, error) {
	scenario := strings.TrimSpace(payload.ScenarioName)
	if scenario == "" {
		return execution.SuiteExecutionInput{}, shared.NewValidationError("scenarioName is required")
	}

	request := orchestrator.SuiteExecutionRequest{
		ScenarioName:   scenario,
		Preset:         strings.TrimSpace(payload.Preset),
		Phases:         payload.Phases,
		Skip:           payload.Skip,
		FailFast:       payload.FailFast,
		UIURL:          strings.TrimSpace(payload.UIURL),
		APIURL:         strings.TrimSpace(payload.APIURL),
		BrowserlessURL: strings.TrimSpace(payload.BrowserlessURL),
		ScenarioPath:   strings.TrimSpace(payload.ScenarioPath),
	}

	var suiteRequestID *uuid.UUID
	if id := strings.TrimSpace(payload.SuiteRequestID); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return execution.SuiteExecutionInput{}, shared.NewValidationError("suiteRequestId must be a valid UUID")
		}
		suiteRequestID = &parsed
	}

	return execution.SuiteExecutionInput{
		Request:        request,
		SuiteRequestID: suiteRequestID,
	}, nil
}

func (s *Server) handleExecuteSuite(w http.ResponseWriter, r *http.Request) {
	input, err := decodeSuiteExecutionInput(r)
	if err != nil {
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			s.writeError(w, http.StatusBadRequest, vErr.Error())
			return
		}
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	if s.executionSvc == nil {
		s.writeError(w, http.StatusInternalServerError, "execution service unavailable")
		return
	}

	result, err := s.executionSvc.Execute(r.Context(), input)
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
