package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
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
	Stats map[string]any `json:"stats"`
}

// CreateProject handles POST /api/v1/projects
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode create project request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "name"}))
		return
	}

	if req.FolderPath == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "folder_path"}))
		return
	}

	// Validate folder path exists and is accessible
	absPath, err := filepath.Abs(req.FolderPath)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "folder_path", "error": "invalid path"}))
		return
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0755); err != nil {
		h.log.WithError(err).WithField("folder_path", absPath).Error("Failed to create project directory")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "create_directory"}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Check if project with this name or folder path already exists
	if existingProject, _ := h.workflowService.GetProjectByName(ctx, req.Name); existingProject != nil {
		h.respondError(w, ErrProjectAlreadyExists.WithMessage("Project with this name already exists"))
		return
	}

	if existingProject, _ := h.workflowService.GetProjectByFolderPath(ctx, absPath); existingProject != nil {
		h.respondError(w, ErrProjectAlreadyExists.WithMessage("Project with this folder path already exists"))
		return
	}

	project := &database.Project{
		Name:        req.Name,
		Description: req.Description,
		FolderPath:  absPath,
	}

	if err := h.workflowService.CreateProject(ctx, project); err != nil {
		h.log.WithError(err).Error("Failed to create project")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "create_project"}))
		return
	}

	h.respondSuccess(w, http.StatusCreated, map[string]any{
		"project_id": project.ID,
		"status":     "created",
		"project":    project,
	})
}

// ListProjects handles GET /api/v1/projects
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Parse pagination parameters
	limit := 100 // default
	offset := 0  // default

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	projects, err := h.workflowService.ListProjects(ctx, limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Failed to list projects")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_projects"}))
		return
	}

	// Get stats for each project
	projectsWithStats := make([]*ProjectWithStats, len(projects))
	for i, project := range projects {
		stats, err := h.workflowService.GetProjectStats(ctx, project.ID)
		if err != nil {
			h.log.WithError(err).WithField("project_id", project.ID).Warn("Failed to get project stats")
			stats = make(map[string]any)
		}

		projectsWithStats[i] = &ProjectWithStats{
			Project: project,
			Stats:   stats,
		}
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"projects": projectsWithStats,
	})
}

// GetProject handles GET /api/v1/projects/{id}
func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidProjectID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	project, err := h.workflowService.GetProject(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get project")
		h.respondError(w, ErrProjectNotFound.WithDetails(map[string]string{"project_id": id.String()}))
		return
	}

	// Get project stats
	stats, err := h.workflowService.GetProjectStats(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("project_id", id).Warn("Failed to get project stats")
		stats = make(map[string]any)
	}

	projectWithStats := &ProjectWithStats{
		Project: project,
		Stats:   stats,
	}

	h.respondSuccess(w, http.StatusOK, projectWithStats)
}

// UpdateProject handles PUT /api/v1/projects/{id}
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidProjectID)
		return
	}

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode update project request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Get existing project
	project, err := h.workflowService.GetProject(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get project for update")
		h.respondError(w, ErrProjectNotFound.WithDetails(map[string]string{"project_id": id.String()}))
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
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "folder_path", "error": "invalid path"}))
			return
		}

		// Create directory if it doesn't exist
		if err := os.MkdirAll(absPath, 0755); err != nil {
			h.log.WithError(err).WithField("folder_path", absPath).Error("Failed to create project directory")
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "create_directory"}))
			return
		}

		project.FolderPath = absPath
	}

	if err := h.workflowService.UpdateProject(ctx, project); err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to update project")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "update_project"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"status":  "updated",
		"project": project,
	})
}

// DeleteProject handles DELETE /api/v1/projects/{id}
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidProjectID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	if err := h.workflowService.DeleteProject(ctx, id); err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to delete project")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "delete_project"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"status": "deleted",
	})
}

// GetProjectWorkflows handles GET /api/v1/projects/{id}/workflows
func (h *Handler) GetProjectWorkflows(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidProjectID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	workflows, err := h.workflowService.ListWorkflowsByProject(ctx, projectID, 100, 0)
	if err != nil {
		h.log.WithError(err).WithField("project_id", projectID).Error("Failed to list project workflows")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_workflows"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"workflows": workflows,
	})
}

// BulkDeleteProjectWorkflows handles POST /api/v1/projects/{id}/workflows/bulk-delete
func (h *Handler) BulkDeleteProjectWorkflows(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	projectID, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, ErrInvalidProjectID)
		return
	}

	var req BulkDeleteProjectWorkflowsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).WithField("project_id", projectID).Error("Failed to decode bulk delete request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	if len(req.WorkflowIDs) == 0 {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "workflow_ids"}))
		return
	}

	workflowIDs := make([]uuid.UUID, 0, len(req.WorkflowIDs))
	for _, workflowIDStr := range req.WorkflowIDs {
		workflowID, err := uuid.Parse(workflowIDStr)
		if err != nil {
			h.respondError(w, ErrInvalidWorkflowID.WithDetails(map[string]string{"workflow_id": workflowIDStr}))
			return
		}
		workflowIDs = append(workflowIDs, workflowID)
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	if err := h.workflowService.DeleteProjectWorkflows(ctx, projectID, workflowIDs); err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"count":      len(workflowIDs),
		}).Error("Failed to bulk delete project workflows")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "bulk_delete_workflows"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
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
		h.respondError(w, ErrInvalidProjectID)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	// Get all workflows for this project
	workflows, err := h.workflowService.ListWorkflowsByProject(ctx, projectID, 1000, 0)
	if err != nil {
		h.log.WithError(err).WithField("project_id", projectID).Error("Failed to get project workflows")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_workflows"}))
		return
	}

	if len(workflows) == 0 {
		h.respondSuccess(w, http.StatusOK, map[string]any{
			"message":    "No workflows found in project",
			"executions": []any{},
		})
		return
	}

	// Start execution for each workflow
	executions := make([]map[string]any, 0, len(workflows))
	for _, workflow := range workflows {
		execution, err := h.workflowService.ExecuteWorkflow(ctx, workflow.ID, map[string]any{})
		if err != nil {
			h.log.WithError(err).WithField("workflow_id", workflow.ID).Warn("Failed to execute workflow in bulk operation")
			executions = append(executions, map[string]any{
				"workflow_id":   workflow.ID.String(),
				"workflow_name": workflow.Name,
				"status":        "failed",
				"error":         err.Error(),
			})
			continue
		}

		executions = append(executions, map[string]any{
			"workflow_id":   workflow.ID.String(),
			"workflow_name": workflow.Name,
			"execution_id":  execution.ID.String(),
			"status":        execution.Status,
		})
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"message":    fmt.Sprintf("Started execution for %d workflows", len(executions)),
		"executions": executions,
	})
}
