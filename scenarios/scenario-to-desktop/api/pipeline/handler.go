package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"scenario-to-desktop-api/shared/errors"
	httputil "scenario-to-desktop-api/shared/http"
)

// Handler provides HTTP handlers for pipeline operations.
type Handler struct {
	orchestrator Orchestrator
	manager      *Manager
	basePath     string
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// WithOrchestrator sets the orchestrator.
func WithOrchestrator(o Orchestrator) HandlerOption {
	return func(h *Handler) {
		h.orchestrator = o
	}
}

// WithBasePath sets the base path for status URLs.
func WithBasePath(path string) HandlerOption {
	return func(h *Handler) {
		h.basePath = path
	}
}

// WithManager sets the pipeline manager.
func WithManager(m *Manager) HandlerOption {
	return func(h *Handler) {
		h.manager = m
	}
}

// NewHandler creates a new pipeline HTTP handler.
func NewHandler(opts ...HandlerOption) *Handler {
	h := &Handler{
		basePath: "/api/v1/pipeline",
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// RegisterRoutes registers the pipeline routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// POST /api/v1/pipeline/run - start a new pipeline
	r.HandleFunc("/api/v1/pipeline/run", h.handleRun).Methods("POST")

	// GET /api/v1/pipeline/{id} - get pipeline status
	r.HandleFunc("/api/v1/pipeline/{id}", h.handleGetStatus).Methods("GET")

	// POST /api/v1/pipeline/{id}/resume - resume a stopped pipeline
	r.HandleFunc("/api/v1/pipeline/{id}/resume", h.handleResume).Methods("POST")

	// POST /api/v1/pipeline/{id}/cancel - cancel pipeline
	r.HandleFunc("/api/v1/pipeline/{id}/cancel", h.handleCancel).Methods("POST")

	// GET /api/v1/pipelines - list all pipelines
	r.HandleFunc("/api/v1/pipelines", h.handleList).Methods("GET")

	// Scenario-based pipeline management (new endpoints)
	// GET /api/v1/scenarios/{name}/pipeline/active - get active pipeline, auto-create if none
	r.HandleFunc("/api/v1/scenarios/{name}/pipeline/active", h.handleGetActivePipeline).Methods("GET")

	// POST /api/v1/scenarios/{name}/pipeline - create new active pipeline (archives old)
	r.HandleFunc("/api/v1/scenarios/{name}/pipeline", h.handleCreatePipeline).Methods("POST")

	// POST /api/v1/scenarios/{name}/pipeline/reset - archive current, clear active
	r.HandleFunc("/api/v1/scenarios/{name}/pipeline/reset", h.handleResetPipeline).Methods("POST")

	// GET /api/v1/scenarios/{name}/pipeline/history - get last N historical pipelines
	r.HandleFunc("/api/v1/scenarios/{name}/pipeline/history", h.handleGetPipelineHistory).Methods("GET")

	// POST /api/v1/scenarios/{name}/pipeline/start - start active pipeline with config
	r.HandleFunc("/api/v1/scenarios/{name}/pipeline/start", h.handleStartActivePipeline).Methods("POST")

	// POST /api/v1/scenarios/{name}/bundle/clean - clean bundle output directory for a scenario
	r.HandleFunc("/api/v1/scenarios/{name}/bundle/clean", h.handleBundleClean).Methods("POST")
}

type bundleCleanRequest struct {
	Framework    string `json:"framework,omitempty"`
	LocationMode string `json:"location_mode,omitempty"`
	PipelineID   string `json:"pipeline_id,omitempty"`
}

type bundleCleanResponse struct {
	ScenarioName string `json:"scenario_name"`
	Framework    string `json:"framework"`
	LocationMode string `json:"location_mode"`
	PipelineID   string `json:"pipeline_id,omitempty"`
	Path         string `json:"path"`
	Removed      bool   `json:"removed"`
}

func (h *Handler) handleBundleClean(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	var req bundleCleanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		httputil.WriteError(w, errors.ErrBadRequest("invalid request body").
			WithCause(err).
			WithRecovery(errors.RecoveryFixInput, "Ensure the request body is valid JSON"))
		return
	}

	framework := req.Framework
	if framework == "" {
		framework = FrameworkElectron
	}
	locationMode := req.LocationMode
	if locationMode == "" {
		locationMode = "proper"
	}
	if isStagingLocation(locationMode) && strings.TrimSpace(req.PipelineID) == "" {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline_id is required for staging/temp location_mode"))
		return
	}

	// Compute scenario path from conventional repo layout.
	home, _ := os.UserHomeDir()
	scenarioPath := filepath.Join(home, "Vrooli", "scenarios", scenarioName)

	cfg := &Config{
		ScenarioName:   scenarioName,
		LocationMode:   locationMode,
		Framework:      framework,
		Platforms:      nil,
		DeploymentMode: "",
	}
	_, desktopPath := resolvePipelineOutputPaths(cfg, scenarioPath, req.PipelineID, framework)
	if desktopPath == "" || !strings.Contains(desktopPath, filepath.Join("platforms", framework)) {
		httputil.WriteError(w, errors.ErrBadRequest("refusing to clean: computed output path is unsafe"))
		return
	}
	bundlePath := filepath.Join(desktopPath, "bundle")

	removed := false
	if _, statErr := os.Stat(bundlePath); statErr == nil {
		if err := removeAllRobust(bundlePath); err != nil {
			httputil.WriteError(w, errors.ErrBadRequest("failed to clean bundle directory").WithCause(err))
			return
		}
		removed = true
	} else if !os.IsNotExist(statErr) {
		httputil.WriteError(w, errors.ErrBadRequest("failed to stat bundle directory").WithCause(statErr))
		return
	}

	httputil.WriteJSON(w, http.StatusOK, bundleCleanResponse{
		ScenarioName: scenarioName,
		Framework:    framework,
		LocationMode: locationMode,
		PipelineID:   req.PipelineID,
		Path:         bundlePath,
		Removed:      removed,
	})
}

// extendWriteDeadline extends the HTTP response write deadline for long-running blocking requests.
// This prevents "i/o timeout" errors when the pipeline takes longer than the default WriteTimeout.
// Returns true if the deadline was successfully extended, false if the ResponseController is not supported.
func extendWriteDeadline(w http.ResponseWriter, timeout time.Duration) bool {
	rc := http.NewResponseController(w)
	// Add 30 seconds buffer to ensure we have time to write the response after pipeline completes
	deadline := time.Now().Add(timeout + 30*time.Second)
	if err := rc.SetWriteDeadline(deadline); err != nil {
		// ResponseController not supported by this ResponseWriter - continue without extension
		// This can happen with certain middleware that wraps the ResponseWriter
		return false
	}
	return true
}

// handleRun handles POST /api/v1/pipeline/run
// Query params:
//   - block: if "true", wait for pipeline completion before returning (default: false)
//   - timeout: max wait time in seconds when block=true (default: 600)
func (h *Handler) handleRun(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		httputil.WriteError(w, errors.ErrPipelineOrchestratorNotConfigured())
		return
	}

	// Parse request body
	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, errors.ErrBadRequest("invalid request body").
			WithCause(err).
			WithRecovery(errors.RecoveryFixInput, "Ensure the request body is valid JSON"))
		return
	}

	config := req.Config

	// Validate required fields
	if config.ScenarioName == "" {
		httputil.WriteError(w, errors.ErrPipelineScenarioRequired())
		return
	}

	if req.VersionUpdate != nil {
		if config.Version != "" {
			httputil.WriteError(w, errors.ErrBadRequest("version and version_update are mutually exclusive"))
			return
		}
		updater := newVersionUpdater("")
		version, rollback, derr := updater.Apply(config.ScenarioName, req.VersionUpdate)
		if derr != nil {
			httputil.WriteError(w, derr)
			return
		}
		if version != "" {
			config.Version = version
		}
		config.setVersionRollback(rollback)
	}

	// Check for blocking mode
	block := r.URL.Query().Get("block") == "true"
	timeoutSecs := 600 // default 10 minutes
	if ts := r.URL.Query().Get("timeout"); ts != "" {
		if parsed, err := strconv.Atoi(ts); err == nil && parsed > 0 {
			timeoutSecs = parsed
		}
	}

	if block {
		// Extend the write deadline to accommodate long-running pipelines
		// This prevents "i/o timeout" errors when writing the response
		extendWriteDeadline(w, time.Duration(timeoutSecs)*time.Second)

		// Blocking mode: wait for pipeline completion
		status, err := h.orchestrator.RunPipelineBlocking(r.Context(), &config, timeoutSecs)
		if err != nil {
			if status == nil || (status.IsComplete() && status.Status != StatusCompleted) {
				if rollback := config.takeVersionRollback(); rollback != nil {
					_ = rollback.Restore()
				}
			}
			// Check if we have a partial status to return
			if status != nil {
				// Timeout case - return the status with the error
				httputil.WriteError(w, errors.Wrap(errors.CodePipelineTimeout, err, "pipeline timed out").
					InDomain("pipeline").
					WithDetails(map[string]interface{}{
						"pipeline_id": status.PipelineID,
						"status":      status.Status,
					}).
					WithRecovery(errors.RecoveryRetry, "Increase the timeout or check pipeline progress"))
				return
			}
			httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to run pipeline").
				InDomain("pipeline").
				WithRecovery(errors.RecoveryRetry, "Check the configuration and try again"))
			return
		}

		// Return appropriate status code based on pipeline result
		if status.Status == StatusFailed {
			httputil.WriteJSON(w, http.StatusInternalServerError, status)
		} else {
			httputil.WriteJSONOK(w, status)
		}
		return
	}

	// Async mode (default): Start pipeline with background context - the pipeline runs asynchronously
	// and should not be cancelled when the HTTP request completes
	status, err := h.orchestrator.RunPipeline(context.Background(), &config)
	if err != nil {
		if rollback := config.takeVersionRollback(); rollback != nil {
			_ = rollback.Restore()
		}
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to start pipeline").
			InDomain("pipeline").
			WithRecovery(errors.RecoveryRetry, "Check the configuration and try again"))
		return
	}

	// Build status URL
	statusURL := fmt.Sprintf("%s/%s", h.basePath, status.PipelineID)

	// Return response
	response := RunResponse{
		PipelineID: status.PipelineID,
		StatusURL:  statusURL,
		Message:    "Pipeline started successfully",
	}

	httputil.WriteJSONAccepted(w, response)
}

// handleGetStatus handles GET /api/v1/pipeline/{id}
// Supports ?verbose=true to include stage Details and Logs (default: false for minimal response)
func (h *Handler) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		httputil.WriteError(w, errors.ErrPipelineOrchestratorNotConfigured())
		return
	}

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	// Get status
	status, ok := h.orchestrator.GetStatus(pipelineID)
	if !ok {
		httputil.WriteError(w, errors.ErrPipelineNotFound(pipelineID))
		return
	}

	// Check if verbose output is requested (default: false for minimal AI-friendly response)
	verbose := r.URL.Query().Get("verbose") == "true"
	if !verbose {
		status = stripVerboseFields(status)
	}

	httputil.WriteJSONOK(w, status)
}

// handleResume handles POST /api/v1/pipeline/{id}/resume
func (h *Handler) handleResume(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		httputil.WriteError(w, errors.ErrPipelineOrchestratorNotConfigured())
		return
	}

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	// Parse optional request body for overrides
	// Note: ContentLength can be -1 for chunked transfer encoding, so we try to decode
	// regardless and only treat empty body as "no config provided"
	var resumeConfig *Config
	if r.ContentLength != 0 && r.Body != nil {
		var config Config
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			// EOF means empty body (e.g., POST with no content) - this is OK, proceed without config
			if err.Error() != "EOF" {
				httputil.WriteError(w, errors.ErrBadRequest("invalid request body").
					WithCause(err).
					WithRecovery(errors.RecoveryFixInput, "Ensure the request body is valid JSON"))
				return
			}
		} else {
			resumeConfig = &config
		}
	}

	// Resume the pipeline with background context - the pipeline runs asynchronously
	// and should not be cancelled when the HTTP request completes
	status, err := h.orchestrator.ResumePipeline(context.Background(), pipelineID, resumeConfig)
	if err != nil {
		// Check for domain errors from orchestrator
		if de, ok := errors.IsDomainError(err); ok {
			httputil.WriteError(w, de)
			return
		}
		// Fallback for non-domain errors
		httputil.WriteError(w, errors.ErrPipelineNotResumable(pipelineID, err.Error()))
		return
	}

	// Build status URL
	statusURL := fmt.Sprintf("%s/%s", h.basePath, status.PipelineID)

	// Return response
	response := ResumeResponse{
		PipelineID:       status.PipelineID,
		ParentPipelineID: pipelineID,
		StatusURL:        statusURL,
		ResumeFromStage:  status.Config.ResumeFromStage,
		Message:          fmt.Sprintf("Pipeline resumed from stage: %s", status.Config.ResumeFromStage),
	}

	httputil.WriteJSONAccepted(w, response)
}

// handleCancel handles POST /api/v1/pipeline/{id}/cancel
func (h *Handler) handleCancel(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		httputil.WriteError(w, errors.ErrPipelineOrchestratorNotConfigured())
		return
	}

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	// Cancel the pipeline
	if !h.orchestrator.CancelPipeline(pipelineID) {
		// Pipeline might not be running or not found
		// Check if it exists
		status, ok := h.orchestrator.GetStatus(pipelineID)
		if !ok {
			httputil.WriteError(w, errors.ErrPipelineNotFound(pipelineID))
			return
		}
		if status.IsComplete() {
			httputil.WriteJSONOK(w, CancelResponse{
				Status:  status.Status,
				Message: "Pipeline has already completed",
			})
			return
		}
	}

	httputil.WriteJSONOK(w, CancelResponse{
		Status:  "cancelling",
		Message: "Pipeline cancellation requested",
	})
}

// handleList handles GET /api/v1/pipelines
func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		httputil.WriteError(w, errors.ErrPipelineOrchestratorNotConfigured())
		return
	}

	pipelines := h.orchestrator.ListPipelines()
	httputil.WriteJSONOK(w, ListResponse{
		Pipelines: pipelines,
	})
}

// handleGetActivePipeline handles GET /api/v1/scenarios/{name}/pipeline/active
// Returns the active pipeline for a scenario, optionally auto-creating one if none exists.
// Query params:
//   - auto_create: if "true", creates a new pipeline if none exists (default: true)
func (h *Handler) handleGetActivePipeline(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline manager not configured"))
		return
	}

	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	// Check auto_create query param (default: true)
	autoCreate := r.URL.Query().Get("auto_create") != "false"

	if !autoCreate {
		// Just return the current active pipeline, don't create
		status, ok := h.manager.GetActivePipelineStatus(scenarioName)
		if !ok {
			httputil.WriteJSONOK(w, ActivePipelineResponse{
				Pipeline: nil,
				Created:  false,
			})
			return
		}
		httputil.WriteJSONOK(w, ActivePipelineResponse{
			Pipeline: status,
			Created:  false,
		})
		return
	}

	// Get or create active pipeline
	status, created, err := h.manager.GetOrCreateActivePipeline(context.Background(), scenarioName, nil)
	if err != nil {
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to get or create active pipeline").
			InDomain("pipeline"))
		return
	}

	httputil.WriteJSONOK(w, ActivePipelineResponse{
		Pipeline: status,
		Created:  created,
	})
}

// handleCreatePipeline handles POST /api/v1/scenarios/{name}/pipeline
// Creates a new active pipeline, archiving the current one if it exists.
func (h *Handler) handleCreatePipeline(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline manager not configured"))
		return
	}

	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	// Parse optional request body for config
	var config *Config
	if r.ContentLength > 0 {
		var c Config
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			httputil.WriteError(w, errors.ErrBadRequest("invalid request body").
				WithCause(err).
				WithRecovery(errors.RecoveryFixInput, "Ensure the request body is valid JSON"))
			return
		}
		config = &c
	}

	status, archivedID, err := h.manager.CreateNewPipeline(context.Background(), scenarioName, config)
	if err != nil {
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to create pipeline").
			InDomain("pipeline"))
		return
	}

	httputil.WriteJSONAccepted(w, CreatePipelineResponse{
		Pipeline:   status,
		ArchivedID: archivedID,
	})
}

// handleResetPipeline handles POST /api/v1/scenarios/{name}/pipeline/reset
// Archives the current active pipeline and clears the active slot.
func (h *Handler) handleResetPipeline(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline manager not configured"))
		return
	}

	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	archivedID, err := h.manager.ResetActivePipeline(scenarioName)
	if err != nil {
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to reset pipeline").
			InDomain("pipeline"))
		return
	}

	httputil.WriteJSONOK(w, ResetPipelineResponse{
		ArchivedID: archivedID,
		Cleared:    true,
	})
}

// handleGetPipelineHistory handles GET /api/v1/scenarios/{name}/pipeline/history
// Returns the history of pipelines for a scenario.
// Query params:
//   - limit: maximum number of pipelines to return (default: 10)
func (h *Handler) handleGetPipelineHistory(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline manager not configured"))
		return
	}

	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	// Parse limit query param
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	pipelines, total, err := h.manager.GetPipelineHistory(scenarioName, limit)
	if err != nil {
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to get pipeline history").
			InDomain("pipeline"))
		return
	}

	httputil.WriteJSONOK(w, PipelineHistoryResponse{
		Pipelines: pipelines,
		Total:     total,
	})
}

// handleStartActivePipeline handles POST /api/v1/scenarios/{name}/pipeline/start
// Starts the active pipeline for a scenario with optional config overrides.
// This is the correct way to run stages - it uses the existing active pipeline
// rather than creating orphaned new ones.
// Query params:
//   - block: if "true", wait for pipeline completion before returning (default: false)
//   - timeout: max wait time in seconds when block=true (default: 600)
func (h *Handler) handleStartActivePipeline(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline manager not configured"))
		return
	}

	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	// Parse optional request body for config overrides
	var config *Config
	if r.ContentLength > 0 {
		var req RunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteError(w, errors.ErrBadRequest("invalid request body").
				WithCause(err).
				WithRecovery(errors.RecoveryFixInput, "Ensure the request body is valid JSON"))
			return
		}

		c := req.Config
		// Ensure scenario name is set
		c.ScenarioName = scenarioName

		if req.VersionUpdate != nil {
			if c.Version != "" {
				httputil.WriteError(w, errors.ErrBadRequest("version and version_update are mutually exclusive"))
				return
			}
			updater := newVersionUpdater("")
			version, rollback, derr := updater.Apply(c.ScenarioName, req.VersionUpdate)
			if derr != nil {
				httputil.WriteError(w, derr)
				return
			}
			if version != "" {
				c.Version = version
			}
			c.setVersionRollback(rollback)
		}

		config = &c
	}

	// Check for blocking mode
	block := r.URL.Query().Get("block") == "true"
	timeoutSecs := 600 // default 10 minutes
	if ts := r.URL.Query().Get("timeout"); ts != "" {
		if parsed, err := strconv.Atoi(ts); err == nil && parsed > 0 {
			timeoutSecs = parsed
		}
	}

	if block {
		// Extend the write deadline to accommodate long-running pipelines
		// This prevents "i/o timeout" errors when writing the response
		extendWriteDeadline(w, time.Duration(timeoutSecs)*time.Second)

		// Blocking mode: wait for pipeline completion
		status, err := h.manager.StartActivePipelineBlocking(r.Context(), scenarioName, config, timeoutSecs)
		if err != nil {
			if status == nil || (status.IsComplete() && status.Status != StatusCompleted) {
				if config != nil {
					if rollback := config.takeVersionRollback(); rollback != nil {
						_ = rollback.Restore()
					}
				}
			}
			// Check if we have a partial status to return
			if status != nil {
				// Timeout case - return the status with the error
				httputil.WriteError(w, errors.Wrap(errors.CodePipelineTimeout, err, "pipeline timed out").
					InDomain("pipeline").
					WithDetails(map[string]interface{}{
						"pipeline_id": status.PipelineID,
						"status":      status.Status,
					}).
					WithRecovery(errors.RecoveryRetry, "Increase the timeout or check pipeline progress"))
				return
			}
			httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to run pipeline").
				InDomain("pipeline"))
			return
		}

		// Return appropriate status code based on pipeline result
		if status.Status == StatusFailed {
			httputil.WriteJSON(w, http.StatusInternalServerError, status)
		} else {
			httputil.WriteJSONOK(w, status)
		}
		return
	}

	// Async mode (default): Start the active pipeline with background context - the pipeline runs asynchronously
	// and should not be cancelled when the HTTP request completes
	status, err := h.manager.StartActivePipeline(context.Background(), scenarioName, config)
	if err != nil {
		if config != nil {
			if rollback := config.takeVersionRollback(); rollback != nil {
				_ = rollback.Restore()
			}
		}
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to start active pipeline").
			InDomain("pipeline"))
		return
	}

	// Build status URL
	statusURL := fmt.Sprintf("%s/%s", h.basePath, status.PipelineID)

	httputil.WriteJSONAccepted(w, StartActivePipelineResponse{
		Pipeline:  status,
		StatusURL: statusURL,
		Message:   "Pipeline started successfully",
	})
}

// stripVerboseFields returns a copy of the status with Details and Logs removed from each stage.
// This provides a minimal response suitable for AI consumption and reduces payload size.
func stripVerboseFields(status *Status) *Status {
	stripped := *status
	stripped.Stages = make(map[string]*StageResult)
	for name, stage := range status.Stages {
		s := *stage
		s.Details = nil // Remove stage-specific details
		s.Logs = nil    // Remove log messages
		stripped.Stages[name] = &s
	}
	return &stripped
}
