// Package handlers provides HTTP handlers for the agent-manager API.
//
// This package is the thin PRESENTATION layer. Handlers are responsible for:
// - HTTP request parsing and validation
// - Response formatting (JSON)
// - Error translation to HTTP status codes
// - Authentication/authorization checks (when implemented)
//
// Handlers do NOT contain business logic - they delegate to the orchestration layer.
package handlers

import (
	"encoding/json"
	"net/http"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler provides HTTP handlers for all API endpoints.
type Handler struct {
	svc orchestration.Service
}

// New creates a new Handler with the given orchestration service.
func New(svc orchestration.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers all API routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Health endpoints
	r.HandleFunc("/health", h.Health).Methods("GET")
	r.HandleFunc("/api/v1/health", h.Health).Methods("GET")

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
	r.HandleFunc("/api/v1/tasks/{id}/cancel", h.CancelTask).Methods("POST")

	// Run endpoints
	r.HandleFunc("/api/v1/runs", h.CreateRun).Methods("POST")
	r.HandleFunc("/api/v1/runs", h.ListRuns).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}", h.GetRun).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/stop", h.StopRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/events", h.GetRunEvents).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/diff", h.GetRunDiff).Methods("GET")
	r.HandleFunc("/api/v1/runs/{id}/approve", h.ApproveRun).Methods("POST")
	r.HandleFunc("/api/v1/runs/{id}/reject", h.RejectRun).Methods("POST")

	// Status endpoints
	r.HandleFunc("/api/v1/runners", h.GetRunnerStatus).Methods("GET")
}

// -----------------------------------------------------------------------------
// Response Helpers
// -----------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func parseUUID(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(mux.Vars(r)[param])
}

func mapDomainError(err error) int {
	switch err.(type) {
	case *domain.NotFoundError:
		return http.StatusNotFound
	case *domain.ValidationError:
		return http.StatusBadRequest
	case *domain.StateError:
		return http.StatusConflict
	case *domain.PolicyViolationError:
		return http.StatusForbidden
	case *domain.CapacityExceededError:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// -----------------------------------------------------------------------------
// Health Handlers
// -----------------------------------------------------------------------------

// Health returns system health status.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.GetHealth(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// -----------------------------------------------------------------------------
// Profile Handlers
// -----------------------------------------------------------------------------

// CreateProfile creates a new agent profile.
func (h *Handler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var profile domain.AgentProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.CreateProfile(r.Context(), &profile)
	if err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// GetProfile retrieves a profile by ID.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid profile ID")
		return
	}

	profile, err := h.svc.GetProfile(r.Context(), id)
	if err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// ListProfiles returns all agent profiles.
func (h *Handler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.svc.ListProfiles(r.Context(), orchestration.ListOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, profiles)
}

// UpdateProfile updates an existing profile.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid profile ID")
		return
	}

	var profile domain.AgentProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	profile.ID = id

	result, err := h.svc.UpdateProfile(r.Context(), &profile)
	if err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// DeleteProfile removes a profile.
func (h *Handler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid profile ID")
		return
	}

	if err := h.svc.DeleteProfile(r.Context(), id); err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// -----------------------------------------------------------------------------
// Task Handlers
// -----------------------------------------------------------------------------

// CreateTask creates a new task.
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.CreateTask(r.Context(), &task)
	if err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// GetTask retrieves a task by ID.
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	task, err := h.svc.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// ListTasks returns all tasks.
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.ListTasks(r.Context(), orchestration.ListOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

// UpdateTask updates an existing task.
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	task.ID = id

	result, err := h.svc.UpdateTask(r.Context(), &task)
	if err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// CancelTask cancels a queued or running task.
func (h *Handler) CancelTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	if err := h.svc.CancelTask(r.Context(), id); err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// -----------------------------------------------------------------------------
// Run Handlers
// -----------------------------------------------------------------------------

// CreateRun creates a new run.
func (h *Handler) CreateRun(w http.ResponseWriter, r *http.Request) {
	var req orchestration.CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	run, err := h.svc.CreateRun(r.Context(), req)
	if err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, run)
}

// GetRun retrieves a run by ID.
func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid run ID")
		return
	}

	run, err := h.svc.GetRun(r.Context(), id)
	if err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, run)
}

// ListRuns returns all runs.
func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := h.svc.ListRuns(r.Context(), orchestration.RunListOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, runs)
}

// StopRun stops a running run.
func (h *Handler) StopRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid run ID")
		return
	}

	if err := h.svc.StopRun(r.Context(), id); err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

// GetRunEvents returns events for a run.
func (h *Handler) GetRunEvents(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid run ID")
		return
	}

	events, err := h.svc.GetRunEvents(r.Context(), id, event.GetOptions{})
	if err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, events)
}

// GetRunDiff returns the diff for a run.
func (h *Handler) GetRunDiff(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid run ID")
		return
	}

	diff, err := h.svc.GetRunDiff(r.Context(), id)
	if err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, diff)
}

// ApproveRun approves a run's changes.
func (h *Handler) ApproveRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid run ID")
		return
	}

	var req struct {
		Actor     string `json:"actor"`
		CommitMsg string `json:"commitMsg"`
		Force     bool   `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.ApproveRun(r.Context(), orchestration.ApproveRequest{
		RunID:     id,
		Actor:     req.Actor,
		CommitMsg: req.CommitMsg,
		Force:     req.Force,
	})
	if err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// RejectRun rejects a run's changes.
func (h *Handler) RejectRun(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid run ID")
		return
	}

	var req struct {
		Actor  string `json:"actor"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.RejectRun(r.Context(), id, req.Actor, req.Reason); err != nil {
		writeError(w, mapDomainError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// GetRunnerStatus returns status of all runners.
func (h *Handler) GetRunnerStatus(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.svc.GetRunnerStatus(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, statuses)
}
