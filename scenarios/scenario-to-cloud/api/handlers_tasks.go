package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/tasks"

	"github.com/gorilla/mux"
)

// handleCreateTask triggers a new task (investigate or fix) for a deployment.
// POST /api/v1/deployments/{id}/tasks
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	if deploymentID == "" {
		http.Error(w, "deployment ID is required", http.StatusBadRequest)
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
	req.DeploymentID = deploymentID

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
		case strings.Contains(errMsg, "expected failed"):
			http.Error(w, err.Error(), http.StatusBadRequest)
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
// GET /api/v1/deployments/{id}/tasks/{taskId}
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]
	taskID := vars["taskId"]

	if deploymentID == "" || taskID == "" {
		http.Error(w, "deployment ID and task ID are required", http.StatusBadRequest)
		return
	}

	if s.taskSvc == nil {
		http.Error(w, "task service not available", http.StatusServiceUnavailable)
		return
	}

	task, err := s.taskSvc.GetTask(r.Context(), deploymentID, taskID)
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

// handleListTasks returns all tasks for a deployment.
// GET /api/v1/deployments/{id}/tasks
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	if deploymentID == "" {
		http.Error(w, "deployment ID is required", http.StatusBadRequest)
		return
	}

	if s.taskSvc == nil {
		http.Error(w, "task service not available", http.StatusServiceUnavailable)
		return
	}

	// Get limit from query params (default 10)
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		var l int
		if _, err := parseIntParam(limitStr, &l); err == nil && l > 0 {
			limit = l
		}
	}

	taskList, err := s.taskSvc.ListTasks(r.Context(), deploymentID, limit)
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
// POST /api/v1/deployments/{id}/tasks/{taskId}/stop
func (s *Server) handleStopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]
	taskID := vars["taskId"]

	if deploymentID == "" || taskID == "" {
		http.Error(w, "deployment ID and task ID are required", http.StatusBadRequest)
		return
	}

	if s.taskSvc == nil {
		http.Error(w, "task service not available", http.StatusServiceUnavailable)
		return
	}

	if err := s.taskSvc.StopTask(r.Context(), deploymentID, taskID); err != nil {
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

// registerTaskRoutes registers the new task routes on the router.
func (s *Server) registerTaskRoutes(api *mux.Router) {
	// New unified task endpoints
	api.HandleFunc("/deployments/{id}/tasks", s.handleCreateTask).Methods("POST")
	api.HandleFunc("/deployments/{id}/tasks", s.handleListTasks).Methods("GET")
	api.HandleFunc("/deployments/{id}/tasks/{taskId}", s.handleGetTask).Methods("GET")
	api.HandleFunc("/deployments/{id}/tasks/{taskId}/stop", s.handleStopTask).Methods("POST")
}

// Ensure taskSvc type is available for use
var _ *tasks.Service
