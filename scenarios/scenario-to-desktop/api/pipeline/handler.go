package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"scenario-to-desktop-api/shared/errors"
	httputil "scenario-to-desktop-api/shared/http"
)

// Handler provides HTTP handlers for pipeline operations.
type Handler struct {
	orchestrator Orchestrator
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
}

// handleRun handles POST /api/v1/pipeline/run
func (h *Handler) handleRun(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		httputil.WriteError(w, errors.ErrPipelineOrchestratorNotConfigured())
		return
	}

	// Parse request body
	var config Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		httputil.WriteError(w, errors.ErrBadRequest("invalid request body").
			WithCause(err).
			WithRecovery(errors.RecoveryFixInput, "Ensure the request body is valid JSON"))
		return
	}

	// Validate required fields
	if config.ScenarioName == "" {
		httputil.WriteError(w, errors.ErrPipelineScenarioRequired())
		return
	}

	// Start pipeline with background context - the pipeline runs asynchronously
	// and should not be cancelled when the HTTP request completes
	status, err := h.orchestrator.RunPipeline(context.Background(), &config)
	if err != nil {
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
