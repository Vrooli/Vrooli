// Package handlers provides HTTP handlers for the agent-manager API.
//
// This package is the thin PRESENTATION layer. Handlers are responsible for:
// - HTTP request parsing and validation
// - Response formatting (JSON)
// - Error translation to HTTP status codes
// - Request ID tracking for observability
// - Authentication/authorization checks (when implemented)
//
// Handlers do NOT contain business logic - they delegate to the orchestration layer.
//
// ERROR HANDLING DESIGN:
// All errors are converted to domain.ErrorResponse for consistent API responses.
// Each response includes:
// - code: Machine-readable error identifier
// - message: Technical description
// - userMessage: Human-friendly explanation
// - recovery: Recommended action (retry, fix_input, wait, escalate)
// - retryable: Whether automatic retry may succeed
// - details: Structured context for debugging
// - requestId: Correlation ID for log aggregation
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler provides HTTP handlers for all API endpoints.
type Handler struct {
	svc orchestration.Service
	hub *WebSocketHub
}

// New creates a new Handler with the given orchestration service.
func New(svc orchestration.Service) *Handler {
	return &Handler{svc: svc}
}

// SetWebSocketHub sets the WebSocket hub for event broadcasting.
func (h *Handler) SetWebSocketHub(hub *WebSocketHub) {
	h.hub = hub
}

// GetWebSocketHub returns the WebSocket hub for external event broadcasting.
func (h *Handler) GetWebSocketHub() *WebSocketHub {
	return h.hub
}

// RegisterRoutes registers all API routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Apply request ID middleware to all routes
	r.Use(requestIDMiddleware)

	// Health endpoints
	r.HandleFunc("/health", h.Health).Methods("GET")
	r.HandleFunc("/api/v1/health", h.Health).Methods("GET")

	// WebSocket endpoint (registered separately, no middleware needed)
	if h.hub != nil {
		r.HandleFunc("/api/v1/ws", h.HandleWebSocket(h.hub))
	}

	// Profile endpoints
	r.HandleFunc("/api/v1/profiles", h.CreateProfile).Methods("POST")
	r.HandleFunc("/api/v1/profiles", h.ListProfiles).Methods("GET")
	r.HandleFunc("/api/v1/profiles/{id}", h.GetProfile).Methods("GET")
	r.HandleFunc("/api/v1/profiles/{id}", h.UpdateProfile).Methods("PUT")
	r.HandleFunc("/api/v1/profiles/{id}", h.DeleteProfile).Methods("DELETE")

	// Task endpoints
	r.HandleFunc("/api/v1/tasks", h.CreateTask).Methods("POST")
	r.HandleFunc("/api/v1/tasks", h.ListTasks).Methods("GET")
	r.HandleFunc("/api/v1/tasks/{id}", h.GetTask).Methods("GET")
	r.HandleFunc("/api/v1/tasks/{id}", h.UpdateTask).Methods("PUT")
	r.HandleFunc("/api/v1/tasks/{id}", h.DeleteTask).Methods("DELETE")
	r.HandleFunc("/api/v1/tasks/{id}/cancel", h.CancelTask).Methods("POST")

	// Run endpoints
	r.HandleFunc("/api/v1/runs", h.CreateRun).Methods("POST")
	r.HandleFunc("/api/v1/runs", h.ListRuns).Methods("GET")
	r.HandleFunc("/api/v1/runs/stop-all", h.StopAllRuns).Methods("POST") // Must be before /{id}
	r.HandleFunc("/api/v1/runs/tag/{tag}", h.GetRunByTag).Methods("GET")
	r.HandleFunc("/api/v1/runs/tag/{tag}/stop", h.StopRunByTag).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}", h.GetRun).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/stop", h.StopRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/events", h.GetRunEvents).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/diff", h.GetRunDiff).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/approve", h.ApproveRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/reject", h.RejectRun).Methods("POST")

	// Status endpoints
	r.HandleFunc("/api/v1/runners", h.GetRunnerStatus).Methods("GET")
	r.HandleFunc("/api/v1/runners/{type}/probe", h.ProbeRunner).Methods("POST")
}

// =============================================================================
// MIDDLEWARE
// =============================================================================

// requestIDMiddleware ensures each request has a unique ID for tracing.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// RESPONSE HELPERS
// =============================================================================

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes a structured error response using domain.ErrorResponse.
// This provides consistent error handling across all endpoints with:
// - Machine-readable error codes
// - User-friendly messages
// - Recovery guidance
// - Request ID for log correlation
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// Convert to structured error response
	errResp := domain.ToErrorResponse(err, requestID)

	// Map to HTTP status code
	status := mapErrorCodeToStatus(errResp.Code)

	// Add retry hint header for retryable errors
	if errResp.Retryable {
		w.Header().Set("X-Retryable", "true")
		if errResp.Recovery == domain.RecoveryRetryBackoff {
			w.Header().Set("Retry-After", "5")
		}
	}

	writeJSON(w, status, errResp)
}

// writeSimpleError creates a simple validation error for request parsing issues.
func writeSimpleError(w http.ResponseWriter, r *http.Request, field, message string) {
	err := domain.NewValidationError(field, message)
	writeError(w, r, err)
}

// parseUUID extracts and parses a UUID from the request path.
func parseUUID(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(mux.Vars(r)[param])
}

// mapErrorCodeToStatus maps domain error codes to HTTP status codes.
// This centralizes the error-to-status mapping based on error semantics.
func mapErrorCodeToStatus(code domain.ErrorCode) int {
	category := code.Category()

	switch category {
	case "NOT":
		// NOT_FOUND_* errors
		return http.StatusNotFound

	case "VALIDATION":
		// VALIDATION_* errors
		return http.StatusBadRequest

	case "STATE":
		// STATE_* errors (conflict)
		return http.StatusConflict

	case "POLICY":
		// POLICY_* errors (forbidden)
		return http.StatusForbidden

	case "CAPACITY":
		// CAPACITY_* errors (temporarily unavailable)
		return http.StatusServiceUnavailable

	case "RUNNER":
		// RUNNER_* errors
		if code == domain.ErrCodeRunnerTimeout {
			return http.StatusGatewayTimeout
		}
		if code == domain.ErrCodeRunnerUnavailable {
			return http.StatusServiceUnavailable
		}
		return http.StatusBadGateway

	case "SANDBOX":
		// SANDBOX_* errors
		if strings.Contains(string(code), "CREATE") {
			return http.StatusServiceUnavailable
		}
		return http.StatusBadGateway

	case "DATABASE":
		// DATABASE_* errors
		if code == domain.ErrCodeDatabaseConnection {
			return http.StatusServiceUnavailable
		}
		return http.StatusInternalServerError

	case "CONFIG":
		// CONFIG_* errors
		return http.StatusInternalServerError

	case "INTERNAL":
		// INTERNAL_* errors
		return http.StatusInternalServerError

	default:
		return http.StatusInternalServerError
	}
}

// mapDomainError maps domain errors to HTTP status codes.
// Kept for backwards compatibility; prefer writeError() for new code.
func mapDomainError(err error) int {
	return mapErrorCodeToStatus(domain.GetErrorCode(err))
}

// =============================================================================
// HEALTH HANDLERS
// =============================================================================

// Health returns system health status.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.GetHealth(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// =============================================================================
// PROFILE HANDLERS
// =============================================================================

// CreateProfile creates a new agent profile.
func (h *Handler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var profile domain.AgentProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	// Validate before sending to service
	if err := profile.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.svc.CreateProfile(r.Context(), &profile)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// GetProfile retrieves a profile by ID.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for profile ID")
		return
	}

	profile, err := h.svc.GetProfile(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// ListProfiles returns all agent profiles.
func (h *Handler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.svc.ListProfiles(r.Context(), orchestration.ListOptions{})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, profiles)
}

// UpdateProfile updates an existing profile.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for profile ID")
		return
	}

	var profile domain.AgentProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	profile.ID = id

	// Validate before sending to service
	if err := profile.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.svc.UpdateProfile(r.Context(), &profile)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// DeleteProfile removes a profile.
func (h *Handler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for profile ID")
		return
	}

	if err := h.svc.DeleteProfile(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// =============================================================================
// TASK HANDLERS
// =============================================================================

// CreateTask creates a new task.
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	// Validate before sending to service
	if err := task.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.svc.CreateTask(r.Context(), &task)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// GetTask retrieves a task by ID.
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for task ID")
		return
	}

	task, err := h.svc.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// ListTasks returns all tasks.
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.ListTasks(r.Context(), orchestration.ListOptions{})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

// UpdateTask updates an existing task.
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for task ID")
		return
	}

	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}
	task.ID = id

	// Validate before sending to service
	if err := task.Validate(); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := h.svc.UpdateTask(r.Context(), &task)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// CancelTask cancels a queued or running task.
func (h *Handler) CancelTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for task ID")
		return
	}

	if err := h.svc.CancelTask(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// DeleteTask permanently removes a cancelled task.
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for task ID")
		return
	}

	if err := h.svc.DeleteTask(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// =============================================================================
// RUN HANDLERS
// =============================================================================

// CreateRun creates a new run.
func (h *Handler) CreateRun(w http.ResponseWriter, r *http.Request) {
	var req orchestration.CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	run, err := h.svc.CreateRun(r.Context(), req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, run)
}

// GetRun retrieves a run by ID.
func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	run, err := h.svc.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, run)
}

// ListRuns returns all runs, with optional filtering.
// Query parameters:
//   - status: Filter by run status (e.g., "running", "pending", "complete")
//   - taskId: Filter by task ID
//   - profileId: Filter by agent profile ID
//   - tagPrefix: Filter by tag prefix (e.g., "ecosystem-" to get all ecosystem-manager runs)
func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	opts := orchestration.RunListOptions{}

	// Parse status filter
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := domain.RunStatus(statusStr)
		opts.Status = &status
	}

	// Parse task ID filter
	if taskIDStr := r.URL.Query().Get("taskId"); taskIDStr != "" {
		if taskID, err := uuid.Parse(taskIDStr); err == nil {
			opts.TaskID = &taskID
		}
	}

	// Parse profile ID filter
	if profileIDStr := r.URL.Query().Get("profileId"); profileIDStr != "" {
		if profileID, err := uuid.Parse(profileIDStr); err == nil {
			opts.AgentProfileID = &profileID
		}
	}

	// Parse tag prefix filter
	if tagPrefix := r.URL.Query().Get("tagPrefix"); tagPrefix != "" {
		opts.TagPrefix = tagPrefix
	}

	runs, err := h.svc.ListRuns(r.Context(), opts)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, runs)
}

// StopRun stops a running run.
func (h *Handler) StopRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	if err := h.svc.StopRun(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

// GetRunByTag retrieves a run by its custom tag.
func (h *Handler) GetRunByTag(w http.ResponseWriter, r *http.Request) {
	tag := mux.Vars(r)["tag"]
	if tag == "" {
		writeSimpleError(w, r, "tag", "tag is required")
		return
	}

	run, err := h.svc.GetRunByTag(r.Context(), tag)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, run)
}

// StopRunByTag stops a run identified by its custom tag.
func (h *Handler) StopRunByTag(w http.ResponseWriter, r *http.Request) {
	tag := mux.Vars(r)["tag"]
	if tag == "" {
		writeSimpleError(w, r, "tag", "tag is required")
		return
	}

	if err := h.svc.StopRunByTag(r.Context(), tag); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "tag": tag})
}

// StopAllRuns stops all running runs, optionally filtered by tag prefix.
// POST /api/v1/runs/stop-all
// Body: {"tagPrefix": "ecosystem-", "force": true}
func (h *Handler) StopAllRuns(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TagPrefix string `json:"tagPrefix"`
		Force     bool   `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	result, err := h.svc.StopAllRuns(r.Context(), orchestration.StopAllOptions{
		TagPrefix: req.TagPrefix,
		Force:     req.Force,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetRunEvents returns events for a run.
func (h *Handler) GetRunEvents(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	events, err := h.svc.GetRunEvents(r.Context(), id, event.GetOptions{})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, events)
}

// GetRunDiff returns the diff for a run.
func (h *Handler) GetRunDiff(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	diff, err := h.svc.GetRunDiff(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, diff)
}

// ApproveRun approves a run's changes.
func (h *Handler) ApproveRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	var req struct {
		Actor     string `json:"actor"`
		CommitMsg string `json:"commitMsg"`
		Force     bool   `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	result, err := h.svc.ApproveRun(r.Context(), orchestration.ApproveRequest{
		RunID:     id,
		Actor:     req.Actor,
		CommitMsg: req.CommitMsg,
		Force:     req.Force,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// RejectRun rejects a run's changes.
func (h *Handler) RejectRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeSimpleError(w, r, "id", "invalid UUID format for run ID")
		return
	}

	var req struct {
		Actor  string `json:"actor"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid JSON request body")
		return
	}

	if err := h.svc.RejectRun(r.Context(), id, req.Actor, req.Reason); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// GetRunnerStatus returns status of all runners.
func (h *Handler) GetRunnerStatus(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.svc.GetRunnerStatus(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, statuses)
}

// ProbeRunner sends a test request to verify a runner can respond.
func (h *Handler) ProbeRunner(w http.ResponseWriter, r *http.Request) {
	runnerType := mux.Vars(r)["type"]
	if runnerType == "" {
		writeSimpleError(w, r, "type", "runner type is required")
		return
	}

	result, err := h.svc.ProbeRunner(r.Context(), domain.RunnerType(runnerType))
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
