package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"scenario-to-desktop-api/domain"
	"scenario-to-desktop-api/tasks"
)

// handleCreateTask triggers a new task (investigate or fix) for a pipeline.
// POST /api/v1/pipeline/{id}/tasks
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pipelineID := vars["id"]

	if pipelineID == "" {
		http.Error(w, "pipeline ID is required", http.StatusBadRequest)
		return
	}

	if s.taskSvc == nil {
		http.Error(w, "task service not available", http.StatusServiceUnavailable)
		return
	}

	// Parse request body
	var req domain.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.PipelineID = pipelineID

	// Set defaults for focus if not specified
	if !req.Focus.Harness && !req.Focus.Subject {
		req.Focus.Harness = true
		req.Focus.Subject = true
	}

	// Trigger task
	inv, err := s.taskSvc.TriggerTask(r.Context(), req)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "not found"):
			http.Error(w, err.Error(), http.StatusNotFound)
		case strings.Contains(errMsg, "already in progress"):
			http.Error(w, err.Error(), http.StatusConflict)
		case strings.Contains(errMsg, "not available"):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		case strings.Contains(errMsg, "invalid"):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case strings.Contains(errMsg, "at least one"):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"task": inv,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleGetTask returns a single task by ID.
// GET /api/v1/pipeline/{id}/tasks/{taskId}
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pipelineID := vars["id"]
	taskID := vars["taskId"]

	if pipelineID == "" || taskID == "" {
		http.Error(w, "pipeline ID and task ID are required", http.StatusBadRequest)
		return
	}

	if s.taskSvc == nil {
		http.Error(w, "task service not available", http.StatusServiceUnavailable)
		return
	}

	task, err := s.taskSvc.GetTask(r.Context(), pipelineID, taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if task == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"task": task,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleListTasks returns all tasks for a pipeline.
// GET /api/v1/pipeline/{id}/tasks
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pipelineID := vars["id"]

	if pipelineID == "" {
		http.Error(w, "pipeline ID is required", http.StatusBadRequest)
		return
	}

	if s.taskSvc == nil {
		http.Error(w, "task service not available", http.StatusServiceUnavailable)
		return
	}

	// Get limit from query params (default 10)
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	taskList, err := s.taskSvc.ListTasks(r.Context(), pipelineID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to summaries
	summaries := make([]domain.InvestigationSummary, 0, len(taskList))
	for _, t := range taskList {
		summaries = append(summaries, t.ToSummary())
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": summaries,
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleStopTask stops a running task.
// POST /api/v1/pipeline/{id}/tasks/{taskId}/stop
func (s *Server) handleStopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pipelineID := vars["id"]
	taskID := vars["taskId"]

	if pipelineID == "" || taskID == "" {
		http.Error(w, "pipeline ID and task ID are required", http.StatusBadRequest)
		return
	}

	if s.taskSvc == nil {
		http.Error(w, "task service not available", http.StatusServiceUnavailable)
		return
	}

	if err := s.taskSvc.StopTask(r.Context(), pipelineID, taskID); err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "not found"):
			http.Error(w, err.Error(), http.StatusNotFound)
		case strings.Contains(errMsg, "not running"):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Task stopped",
	}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleGetAgentManagerStatus returns the agent-manager availability status.
// GET /api/v1/agent-manager/status
func (s *Server) handleGetAgentManagerStatus(w http.ResponseWriter, r *http.Request) {
	if s.taskSvc == nil {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"available": false,
			"reason":    "task service not configured",
		}); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
		return
	}

	available := s.taskSvc.IsAgentAvailable(r.Context())
	response := map[string]interface{}{
		"available": available,
	}

	if available {
		url, err := s.taskSvc.GetAgentManagerURL(r.Context())
		if err == nil {
			response["url"] = url
		}
	} else {
		response["reason"] = "agent-manager service not reachable"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// registerTaskRoutes registers the task routes on the router.
func (s *Server) registerTaskRoutes() {
	// Task endpoints under pipeline
	s.router.HandleFunc("/api/v1/pipeline/{id}/tasks", s.handleCreateTask).Methods("POST")
	s.router.HandleFunc("/api/v1/pipeline/{id}/tasks", s.handleListTasks).Methods("GET")
	s.router.HandleFunc("/api/v1/pipeline/{id}/tasks/{taskId}", s.handleGetTask).Methods("GET")
	s.router.HandleFunc("/api/v1/pipeline/{id}/tasks/{taskId}/stop", s.handleStopTask).Methods("POST")

	// Agent-manager status endpoint
	s.router.HandleFunc("/api/v1/agent-manager/status", s.handleGetAgentManagerStatus).Methods("GET")
}

// Ensure taskSvc type is available for use
var _ *tasks.Service
