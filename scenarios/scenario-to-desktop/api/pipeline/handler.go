package pipeline

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
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

	// POST /api/v1/pipeline/{id}/cancel - cancel pipeline
	r.HandleFunc("/api/v1/pipeline/{id}/cancel", h.handleCancel).Methods("POST")

	// GET /api/v1/pipelines - list all pipelines
	r.HandleFunc("/api/v1/pipelines", h.handleList).Methods("GET")
}

// handleRun handles POST /api/v1/pipeline/run
func (h *Handler) handleRun(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		h.writeError(w, http.StatusInternalServerError, "pipeline orchestrator not configured")
		return
	}

	// Parse request body
	var config Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Validate required fields
	if config.ScenarioName == "" {
		h.writeError(w, http.StatusBadRequest, "scenario_name is required")
		return
	}

	// Start pipeline
	status, err := h.orchestrator.RunPipeline(r.Context(), &config)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to start pipeline: %v", err))
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

	h.writeJSON(w, http.StatusAccepted, response)
}

// handleGetStatus handles GET /api/v1/pipeline/{id}
func (h *Handler) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		h.writeError(w, http.StatusInternalServerError, "pipeline orchestrator not configured")
		return
	}

	vars := mux.Vars(r)
	pipelineID := vars["id"]

	// Get status
	status, ok := h.orchestrator.GetStatus(pipelineID)
	if !ok {
		h.writeError(w, http.StatusNotFound, fmt.Sprintf("pipeline not found: %s", pipelineID))
		return
	}

	h.writeJSON(w, http.StatusOK, status)
}

// handleCancel handles POST /api/v1/pipeline/{id}/cancel
func (h *Handler) handleCancel(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		h.writeError(w, http.StatusInternalServerError, "pipeline orchestrator not configured")
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
			h.writeError(w, http.StatusNotFound, fmt.Sprintf("pipeline not found: %s", pipelineID))
			return
		}
		if status.IsComplete() {
			h.writeJSON(w, http.StatusOK, CancelResponse{
				Status:  status.Status,
				Message: "Pipeline has already completed",
			})
			return
		}
	}

	h.writeJSON(w, http.StatusOK, CancelResponse{
		Status:  "cancelling",
		Message: "Pipeline cancellation requested",
	})
}

// handleList handles GET /api/v1/pipelines
func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		h.writeError(w, http.StatusInternalServerError, "pipeline orchestrator not configured")
		return
	}

	pipelines := h.orchestrator.ListPipelines()
	h.writeJSON(w, http.StatusOK, ListResponse{
		Pipelines: pipelines,
	})
}

// writeJSON writes a JSON response.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
