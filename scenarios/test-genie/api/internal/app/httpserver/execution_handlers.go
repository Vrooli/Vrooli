package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
	"test-genie/internal/shared"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/cli-core/cliutil"
)

type suiteExecutionPayload struct {
	ScenarioName      string   `json:"scenarioName"`
	Target            string   `json:"target"`
	SuiteRequestID    string   `json:"suiteRequestId"`
	Preset            string   `json:"preset"`
	Phases            []string `json:"phases"`
	Skip              []string `json:"skip"`
	FailFast          bool     `json:"failFast"`
	DiagnosticsPreset string   `json:"diagnosticsPreset"`
	CaptureProfile    string   `json:"captureProfile"`
	UIURL             string   `json:"uiUrl"`
	APIURL            string   `json:"apiUrl"`
	// ScenarioPath is the absolute physical scenario directory to read and write.
	ScenarioPath string `json:"scenarioPath"`
	// LogicalRepoRoot and LogicalScenarioRelPath describe where repo-relative
	// validation should treat the physical scenario as living.
	LogicalRepoRoot        string `json:"logicalRepoRoot"`
	LogicalScenarioRelPath string `json:"logicalScenarioRelPath"`
}

// handleAdmissionStatus exposes bounded-admission occupancy, configured caps,
// and process-lifetime overload/coalescing counters for diagnosis.
func (s *Server) handleAdmissionStatus(w http.ResponseWriter, _ *http.Request) {
	if s.runManager == nil {
		s.writeError(w, http.StatusServiceUnavailable, "run admission is unavailable")
		return
	}
	s.writeJSON(w, http.StatusOK, s.runManager.AdmissionStatus())
}

// admissionCaller is an intentionally small, bounded attribution seam. Trusted
// gateways may attach X-Vrooli-Caller; absent identity is handled by the
// runmanager's conservative anonymous bucket.
func admissionCaller(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get(cliutil.HeaderCaller))
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
	target := strings.TrimSpace(payload.Target)
	if scenario == "" && target == "" {
		return execution.SuiteExecutionInput{}, shared.NewValidationError("target or scenarioName is required")
	}
	if target != "" {
		scenario = target
	}
	if strings.TrimSpace(payload.SuiteRequestID) != "" {
		return execution.SuiteExecutionInput{}, shared.NewValidationError("suiteRequestId has been removed; execute the scenario, then create a remediation job from its execution evidence")
	}

	request := orchestrator.SuiteExecutionRequest{
		ScenarioName:           scenario,
		Target:                 target,
		Preset:                 strings.TrimSpace(payload.Preset),
		Phases:                 payload.Phases,
		Skip:                   payload.Skip,
		FailFast:               payload.FailFast,
		DiagnosticsPreset:      strings.TrimSpace(payload.DiagnosticsPreset),
		CaptureProfile:         strings.TrimSpace(payload.CaptureProfile),
		UIURL:                  strings.TrimSpace(payload.UIURL),
		APIURL:                 strings.TrimSpace(payload.APIURL),
		ScenarioPath:           strings.TrimSpace(payload.ScenarioPath),
		LogicalRepoRoot:        strings.TrimSpace(payload.LogicalRepoRoot),
		LogicalScenarioRelPath: strings.TrimSpace(payload.LogicalScenarioRelPath),
	}

	return execution.SuiteExecutionInput{Request: request}, nil
}

// handleExecuteSuite is the blocking REST adapter over the run manager: it
// starts a durable, request-decoupled run and blocks until it completes,
// returning the full result. Because the run is owned by the server (not this
// request), a client disconnect detaches this handler without aborting the run
// — the caller can re-attach by run id. Consumed by git-control-tower and any
// programmatic blocking caller.
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

	if s.runManager == nil {
		s.writeError(w, http.StatusInternalServerError, "execution service unavailable")
		return
	}
	if runID := s.runManager.CoalescedRunID(input.Request); runID != "" {
		status, waitErr := s.runManager.Wait(r.Context(), input.Request.ScenarioName, runID)
		if waitErr != nil {
			return
		}
		s.writeJSON(w, http.StatusOK, status.Result)
		return
	}
	caller := admissionCaller(r)
	releasePreview, err := s.runManager.TryAcquirePreviewFor(caller)
	if err != nil {
		w.Header().Set("Retry-After", "5")
		s.writeError(w, http.StatusTooManyRequests, err.Error())
		return
	}
	defer releasePreview()

	// Synchronous plan validation (bad preset/phase → 400) + ETA.
	preview, eta, err := s.previewExecutionPlan(r.Context(), input.Request)
	if err != nil {
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			s.writeError(w, http.StatusBadRequest, vErr.Error())
			return
		}
	}
	applyPreviewPhaseSelection(&input, preview)

	res, err := s.runManager.Start(runmanager.StartOptions{Input: input, Caller: caller, EstimatedTotalSeconds: eta})
	if err != nil {
		var busy *runmanager.BusyError
		if errors.As(err, &busy) {
			s.writeError(w, http.StatusConflict, busy.Error())
			return
		}
		var saturated *runmanager.SaturatedError
		if errors.As(err, &saturated) {
			w.Header().Set("Retry-After", "5")
			s.writeError(w, http.StatusTooManyRequests, saturated.Error())
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	runID := res.RunID

	status, err := s.runManager.Wait(r.Context(), input.Request.ScenarioName, runID)
	if err != nil {
		// Client disconnected (or its deadline elapsed): the run continues under
		// the server-lifetime context. Nothing to write back.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if status.Result == nil {
		detail := status.Error
		if detail == "" {
			detail = "suite execution failed"
		}
		s.log("suite execution failed", map[string]interface{}{"error": detail})
		s.writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "suite execution failed",
			"errors":  []string{detail},
			"metadata": map[string]interface{}{
				"scenarioName": input.Request.ScenarioName,
				"scenarioPath": input.Request.ScenarioPath,
			},
		})
		return
	}

	s.writeJSON(w, http.StatusOK, status.Result)
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
