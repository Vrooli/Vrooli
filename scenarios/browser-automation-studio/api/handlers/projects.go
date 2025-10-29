package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	FolderPath  string `json:"folder_path"`
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	FolderPath  string `json:"folder_path,omitempty"`
}

// BulkDeleteProjectWorkflowsRequest represents the request payload for deleting multiple workflows
type BulkDeleteProjectWorkflowsRequest struct {
	WorkflowIDs []string `json:"workflow_ids"`
}

// ProjectWithStats represents a project with statistics
type ProjectWithStats struct {
	*database.Project
	Stats map[string]interface{} `json:"stats"`
}

// CreateProject handles POST /api/v1/projects
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode create project request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	if req.FolderPath == "" {
		http.Error(w, "Folder path is required", http.StatusBadRequest)
		return
	}

	// Validate folder path exists and is accessible
	absPath, err := filepath.Abs(req.FolderPath)
	if err != nil {
		http.Error(w, "Invalid folder path", http.StatusBadRequest)
		return
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0755); err != nil {
		h.log.WithError(err).WithField("folder_path", absPath).Error("Failed to create project directory")
		http.Error(w, "Failed to create project directory", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check if project with this name or folder path already exists
	if existingProject, _ := h.workflowService.GetProjectByName(ctx, req.Name); existingProject != nil {
		http.Error(w, "Project with this name already exists", http.StatusConflict)
		return
	}

	if existingProject, _ := h.workflowService.GetProjectByFolderPath(ctx, absPath); existingProject != nil {
		http.Error(w, "Project with this folder path already exists", http.StatusConflict)
		return
	}

	project := &database.Project{
		Name:        req.Name,
		Description: req.Description,
		FolderPath:  absPath,
	}

	if err := h.workflowService.CreateProject(ctx, project); err != nil {
		h.log.WithError(err).Error("Failed to create project")
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"project_id": project.ID,
		"status":     "created",
		"project":    project,
	})
}

// ListProjects handles GET /api/v1/projects
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	projects, err := h.workflowService.ListProjects(ctx, 100, 0)
	if err != nil {
		h.log.WithError(err).Error("Failed to list projects")
		http.Error(w, "Failed to list projects", http.StatusInternalServerError)
		return
	}

	// Get stats for each project
	projectsWithStats := make([]*ProjectWithStats, len(projects))
	for i, project := range projects {
		stats, err := h.workflowService.GetProjectStats(ctx, project.ID)
		if err != nil {
			h.log.WithError(err).WithField("project_id", project.ID).Warn("Failed to get project stats")
			stats = make(map[string]interface{})
		}

		projectsWithStats[i] = &ProjectWithStats{
			Project: project,
			Stats:   stats,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"projects": projectsWithStats,
	})
}

// GetProject handles GET /api/v1/projects/{id}
func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	project, err := h.workflowService.GetProject(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get project")
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// Get project stats
	stats, err := h.workflowService.GetProjectStats(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("project_id", id).Warn("Failed to get project stats")
		stats = make(map[string]interface{})
	}

	projectWithStats := &ProjectWithStats{
		Project: project,
		Stats:   stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projectWithStats)
}

// UpdateProject handles PUT /api/v1/projects/{id}
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode update project request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get existing project
	project, err := h.workflowService.GetProject(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get project for update")
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// Update fields if provided
	if req.Name != "" {
		project.Name = req.Name
	}
	if req.Description != "" {
		project.Description = req.Description
	}
	if req.FolderPath != "" {
		// Validate and update folder path
		absPath, err := filepath.Abs(req.FolderPath)
		if err != nil {
			http.Error(w, "Invalid folder path", http.StatusBadRequest)
			return
		}

		// Create directory if it doesn't exist
		if err := os.MkdirAll(absPath, 0755); err != nil {
			h.log.WithError(err).WithField("folder_path", absPath).Error("Failed to create project directory")
			http.Error(w, "Failed to create project directory", http.StatusInternalServerError)
			return
		}

		project.FolderPath = absPath
	}

	if err := h.workflowService.UpdateProject(ctx, project); err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to update project")
		http.Error(w, "Failed to update project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "updated",
		"project": project,
	})
}

// DeleteProject handles DELETE /api/v1/projects/{id}
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.workflowService.DeleteProject(ctx, id); err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to delete project")
		http.Error(w, "Failed to delete project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "deleted",
	})
}

// GetProjectWorkflows handles GET /api/v1/projects/{id}/workflows
func (h *Handler) GetProjectWorkflows(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	workflows, err := h.workflowService.ListWorkflowsByProject(ctx, projectID, 100, 0)
	if err != nil {
		h.log.WithError(err).WithField("project_id", projectID).Error("Failed to list project workflows")
		http.Error(w, "Failed to list workflows", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows": workflows,
	})
}

// BulkDeleteProjectWorkflows handles POST /api/v1/projects/{id}/workflows/bulk-delete
func (h *Handler) BulkDeleteProjectWorkflows(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	var req BulkDeleteProjectWorkflowsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).WithField("project_id", projectID).Error("Failed to decode bulk delete request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.WorkflowIDs) == 0 {
		http.Error(w, "workflow_ids is required", http.StatusBadRequest)
		return
	}

	workflowIDs := make([]uuid.UUID, 0, len(req.WorkflowIDs))
	for _, workflowIDStr := range req.WorkflowIDs {
		workflowID, err := uuid.Parse(workflowIDStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid workflow ID: %s", workflowIDStr), http.StatusBadRequest)
			return
		}
		workflowIDs = append(workflowIDs, workflowID)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.workflowService.DeleteProjectWorkflows(ctx, projectID, workflowIDs); err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"count":      len(workflowIDs),
		}).Error("Failed to bulk delete project workflows")
		http.Error(w, "Failed to delete workflows", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "deleted",
		"deleted_count": len(workflowIDs),
		"deleted_ids":   req.WorkflowIDs,
	})
}

// ExecuteAllProjectWorkflows handles POST /api/v1/projects/{id}/execute-all
func (h *Handler) ExecuteAllProjectWorkflows(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Get all workflows for this project
	workflows, err := h.workflowService.ListWorkflowsByProject(ctx, projectID, 1000, 0)
	if err != nil {
		h.log.WithError(err).WithField("project_id", projectID).Error("Failed to get project workflows")
		http.Error(w, "Failed to get project workflows", http.StatusInternalServerError)
		return
	}

	if len(workflows) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":    "No workflows found in project",
			"executions": []interface{}{},
		})
		return
	}

	// Start execution for each workflow
	executions := make([]map[string]interface{}, 0, len(workflows))
	for _, workflow := range workflows {
		execution, err := h.workflowService.ExecuteWorkflow(ctx, workflow.ID, map[string]interface{}{})
		if err != nil {
			h.log.WithError(err).WithField("workflow_id", workflow.ID).Warn("Failed to execute workflow in bulk operation")
			executions = append(executions, map[string]interface{}{
				"workflow_id":   workflow.ID.String(),
				"workflow_name": workflow.Name,
				"status":        "failed",
				"error":         err.Error(),
			})
			continue
		}

		executions = append(executions, map[string]interface{}{
			"workflow_id":   workflow.ID.String(),
			"workflow_name": workflow.Name,
			"execution_id":  execution.ID.String(),
			"status":        execution.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    fmt.Sprintf("Started execution for %d workflows", len(executions)),
		"executions": executions,
	})
}
