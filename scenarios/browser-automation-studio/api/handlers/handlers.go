package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services"
	"github.com/vrooli/browser-automation-studio/storage"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// Handler contains all HTTP handlers
type Handler struct {
	workflowService *services.WorkflowService
	repo            database.Repository
	browserless     *browserless.Client
	wsHub           *wsHub.Hub
	storage         *storage.MinIOClient
	log             *logrus.Logger
	upgrader        websocket.Upgrader
}

// NewHandler creates a new handler instance
func NewHandler(repo database.Repository, browserless *browserless.Client, wsHub *wsHub.Hub, log *logrus.Logger) *Handler {
	// Initialize MinIO client for screenshot serving
	storageClient, err := storage.NewMinIOClient(log)
	if err != nil {
		log.WithError(err).Warn("Failed to initialize MinIO client for handlers - screenshot serving will be disabled")
	}

	return &Handler{
		workflowService: services.NewWorkflowService(repo, browserless, wsHub, log),
		repo:            repo,
		browserless:     browserless,
		wsHub:           wsHub,
		storage:         storageClient,
		log:             log,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now - in production, validate origins properly
				return true
			},
		},
	}
}

// HealthResponse represents the health check response following Vrooli standards
type HealthResponse struct {
	Status       string                 `json:"status"`
	Service      string                 `json:"service"`
	Timestamp    string                 `json:"timestamp"`
	Readiness    bool                   `json:"readiness"`
	Version      string                 `json:"version,omitempty"`
	Dependencies map[string]interface{} `json:"dependencies,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
}

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

// ProjectWithStats represents a project with statistics
type ProjectWithStats struct {
	*database.Project
	Stats map[string]interface{} `json:"stats"`
}

// CreateWorkflowRequest represents the request to create a workflow
type CreateWorkflowRequest struct {
	ProjectID      *uuid.UUID             `json:"project_id,omitempty"`
	Name           string                 `json:"name"`
	FolderPath     string                 `json:"folder_path"`
	FlowDefinition map[string]interface{} `json:"flow_definition,omitempty"`
	AIPrompt       string                 `json:"ai_prompt,omitempty"`
}

// ExecuteWorkflowRequest represents the request to execute a workflow
type ExecuteWorkflowRequest struct {
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
	WaitForCompletion bool                   `json:"wait_for_completion"`
}

// Health handles GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	// Check database health
	var databaseHealthy bool
	var databaseLatency float64
	var databaseError map[string]interface{}
	
	start := time.Now()
	if h.repo != nil {
		// Try a simple database operation to check health
		_, err := h.repo.ListProjects(ctx, 1, 0)
		databaseLatency = float64(time.Since(start).Nanoseconds()) / 1e6 // Convert to milliseconds
		
		if err != nil {
			databaseHealthy = false
			databaseError = map[string]interface{}{
				"code":      "DATABASE_CONNECTION_ERROR",
				"message":   fmt.Sprintf("Database health check failed: %v", err),
				"category":  "resource",
				"retryable": true,
			}
		} else {
			databaseHealthy = true
		}
	} else {
		databaseHealthy = false
		databaseError = map[string]interface{}{
			"code":      "REPOSITORY_NOT_INITIALIZED",
			"message":   "Database repository not initialized",
			"category":  "internal",
			"retryable": false,
		}
	}
	
	// Check browserless health
	var browserlessHealthy bool
	var browserlessError map[string]interface{}
	
	if h.browserless != nil {
		if err := h.browserless.CheckBrowserlessHealth(); err != nil {
			browserlessHealthy = false
			browserlessError = map[string]interface{}{
				"code":      "BROWSERLESS_CONNECTION_ERROR",
				"message":   fmt.Sprintf("Browserless health check failed: %v", err),
				"category":  "resource",
				"retryable": true,
			}
		} else {
			browserlessHealthy = true
		}
	} else {
		browserlessHealthy = false
		browserlessError = map[string]interface{}{
			"code":      "BROWSERLESS_NOT_INITIALIZED",
			"message":   "Browserless client not initialized",
			"category":  "internal",
			"retryable": false,
		}
	}
	
	// Overall service status
	status := "healthy"
	readiness := true
	
	if !databaseHealthy {
		status = "unhealthy"
		readiness = false
	} else if !browserlessHealthy {
		status = "degraded"
		// Keep readiness true for degraded state - we can still serve some requests
	}
	
	// Build dependencies map
	dependencies := map[string]interface{}{
		"database": map[string]interface{}{
			"connected":  databaseHealthy,
			"latency_ms": databaseLatency,
			"error":      nil,
		},
		"external_services": []map[string]interface{}{
			{
				"name":      "browserless",
				"connected": browserlessHealthy,
				"error":     nil,
			},
		},
	}
	
	// Add errors if present
	if databaseError != nil {
		dependencies["database"].(map[string]interface{})["error"] = databaseError
	}
	if browserlessError != nil {
		dependencies["external_services"].([]map[string]interface{})[0]["error"] = browserlessError
	}
	
	response := HealthResponse{
		Status:       status,
		Service:      "browser-automation-studio-api",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Readiness:    readiness,
		Version:      "1.0.0",
		Dependencies: dependencies,
		Metrics: map[string]interface{}{
			"goroutines": 0, // Could be populated with runtime.NumGoroutine() if needed
		},
	}
	
	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// Project handlers

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

// CreateWorkflow handles POST /api/v1/workflows/create
func (h *Handler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode create workflow request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	
	if req.FolderPath == "" {
		req.FolderPath = "/"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	workflow, err := h.workflowService.CreateWorkflowWithProject(ctx, req.ProjectID, req.Name, req.FolderPath, req.FlowDefinition, req.AIPrompt)
	if err != nil {
		h.log.WithError(err).Error("Failed to create workflow")
		http.Error(w, "Failed to create workflow", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflow_id": workflow.ID,
		"status":      "created",
		"nodes":       workflow.FlowDefinition["nodes"],
		"edges":       workflow.FlowDefinition["edges"],
	})
}

// ListWorkflows handles GET /api/v1/workflows
func (h *Handler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	folderPath := r.URL.Query().Get("folder_path")
	
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	workflows, err := h.workflowService.ListWorkflows(ctx, folderPath, 100, 0)
	if err != nil {
		h.log.WithError(err).Error("Failed to list workflows")
		http.Error(w, "Failed to list workflows", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows": workflows,
	})
}

// GetWorkflow handles GET /api/v1/workflows/{id}
func (h *Handler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	workflow, err := h.workflowService.GetWorkflow(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get workflow")
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflow)
}

// ExecuteWorkflow handles POST /api/v1/workflows/{id}/execute
func (h *Handler) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	workflowID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}

	var req ExecuteWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode execute workflow request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	execution, err := h.workflowService.ExecuteWorkflow(ctx, workflowID, req.Parameters)
	if err != nil {
		h.log.WithError(err).Error("Failed to execute workflow")
		http.Error(w, "Failed to start execution", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"execution_id": execution.ID,
		"status":       execution.Status,
	})
}

// GetExecutionScreenshots handles GET /api/v1/executions/{id}/screenshots
func (h *Handler) GetExecutionScreenshots(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	executionID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	screenshots, err := h.workflowService.GetExecutionScreenshots(ctx, executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get screenshots")
		http.Error(w, "Failed to get screenshots", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"screenshots": screenshots,
	})
}

// HandleWebSocket handles WebSocket connections for real-time updates
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	// Check if client wants to subscribe to a specific execution
	var executionID *uuid.UUID
	if execIDStr := r.URL.Query().Get("execution_id"); execIDStr != "" {
		if id, err := uuid.Parse(execIDStr); err == nil {
			executionID = &id
		}
	}

	h.log.WithFields(logrus.Fields{
		"remote_addr":  r.RemoteAddr,
		"execution_id": executionID,
	}).Info("New WebSocket connection established")

	h.wsHub.ServeWS(conn, executionID)
}

// ServeScreenshot serves a screenshot from MinIO storage
func (h *Handler) ServeScreenshot(w http.ResponseWriter, r *http.Request) {
	// Extract object name from URL path
	objectName := chi.URLParam(r, "*")
	if objectName == "" {
		http.Error(w, "Screenshot path required", http.StatusBadRequest)
		return
	}

	// Remove any leading slash
	objectName = strings.TrimPrefix(objectName, "/")

	if h.storage == nil {
		http.Error(w, "Screenshot storage not available", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Get screenshot from MinIO
	object, info, err := h.storage.GetScreenshot(ctx, objectName)
	if err != nil {
		h.log.WithError(err).WithField("object_name", objectName).Error("Failed to get screenshot")
		http.Error(w, "Screenshot not found", http.StatusNotFound)
		return
	}
	defer object.Close()

	// Set appropriate headers
	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Stream the file to response
	if _, err := io.Copy(w, object); err != nil {
		h.log.WithError(err).Error("Failed to stream screenshot")
	}
}

// ServeThumbnail serves a thumbnail version of a screenshot
func (h *Handler) ServeThumbnail(w http.ResponseWriter, r *http.Request) {
	// For now, serve the same image as thumbnail
	// In a real implementation, you might generate actual thumbnails
	h.ServeScreenshot(w, r)
}

// GetExecution handles GET /api/v1/executions/{id}
func (h *Handler) GetExecution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	execution, err := h.workflowService.GetExecution(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get execution")
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// ListExecutions handles GET /api/v1/executions
func (h *Handler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	workflowIDStr := r.URL.Query().Get("workflow_id")
	var workflowID *uuid.UUID
	if workflowIDStr != "" {
		if id, err := uuid.Parse(workflowIDStr); err == nil {
			workflowID = &id
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	executions, err := h.workflowService.ListExecutions(ctx, workflowID, 100, 0)
	if err != nil {
		h.log.WithError(err).Error("Failed to list executions")
		http.Error(w, "Failed to list executions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"executions": executions,
	})
}

// StopExecution handles POST /api/v1/executions/{id}/stop
func (h *Handler) StopExecution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.workflowService.StopExecution(ctx, id); err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to stop execution")
		http.Error(w, "Failed to stop execution", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "stopped",
	})
}

// GetScenarioPort handles GET /api/v1/scenarios/{name}/port
func (h *Handler) GetScenarioPort(w http.ResponseWriter, r *http.Request) {
	scenarioName := chi.URLParam(r, "name")
	if scenarioName == "" {
		http.Error(w, "Scenario name is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	portInfo, err := h.getScenarioPortInfo(ctx, scenarioName)
	if err != nil {
		h.log.WithError(err).WithField("scenario", scenarioName).Error("Failed to get scenario port")
		http.Error(w, "Failed to get scenario port information", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(portInfo)
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
			"message": "No workflows found in project",
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
				"workflow_id": workflow.ID.String(),
				"workflow_name": workflow.Name,
				"status": "failed",
				"error": err.Error(),
			})
			continue
		}

		executions = append(executions, map[string]interface{}{
			"workflow_id": workflow.ID.String(),
			"workflow_name": workflow.Name,
			"execution_id": execution.ID.String(),
			"status": execution.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("Started execution for %d workflows", len(executions)),
		"executions": executions,
	})
}

// ScenarioPortInfo represents port information for a scenario
type ScenarioPortInfo struct {
	Port   int    `json:"port"`
	Status string `json:"status"`
	URL    string `json:"url"`
}

// getScenarioPortInfo executes vrooli CLI command to get scenario port information
// TakePreviewScreenshot handles POST /api/v1/preview-screenshot
func (h *Handler) TakePreviewScreenshot(w http.ResponseWriter, r *http.Request) {
	type PreviewRequest struct {
		URL string `json:"url"`
	}
	
	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode preview request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}
	
	h.log.WithField("url", req.URL).Info("Taking preview screenshot and capturing console logs")
	
	// Create temporary files for screenshot and console logs
	tmpScreenshotFile, err := os.CreateTemp("", "preview-*.png")
	if err != nil {
		h.log.WithError(err).Error("Failed to create temp screenshot file")
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpScreenshotFile.Name())
	defer tmpScreenshotFile.Close()
	
	tmpConsoleFile, err := os.CreateTemp("", "console-*.json")
	if err != nil {
		h.log.WithError(err).Error("Failed to create temp console file")
		http.Error(w, "Failed to create temporary console file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpConsoleFile.Name())
	defer tmpConsoleFile.Close()
	
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second) // Increased timeout for both operations
	defer cancel()
	
	// Take screenshot first
	screenshotCmd := exec.CommandContext(ctx, "resource-browserless", "screenshot", 
		"--url", req.URL,
		"--output", tmpScreenshotFile.Name())
	
	screenshotOutput, err := screenshotCmd.CombinedOutput()
	if err != nil {
		h.log.WithError(err).WithField("output", string(screenshotOutput)).Error("Failed to take screenshot with resource-browserless")
		
		// Provide more helpful error messages
		errorMsg := "Failed to take screenshot"
		if strings.Contains(string(screenshotOutput), "HTTP 500") {
			errorMsg = "Unable to access the URL - it may not be reachable from the browser automation service"
		} else if strings.Contains(string(screenshotOutput), "timeout") {
			errorMsg = "Screenshot request timed out - the page may be taking too long to load"
		} else if strings.Contains(string(screenshotOutput), "connection") {
			errorMsg = "Cannot connect to the URL - please check if it's accessible"
		}
		
		http.Error(w, errorMsg, http.StatusInternalServerError)
		return
	}
	
	// Capture console logs
	consoleCmd := exec.CommandContext(ctx, "resource-browserless", "console", 
		req.URL,
		"--output", tmpConsoleFile.Name(),
		"--wait-ms", "3000") // Wait a bit longer for dynamic content
	
	consoleOutput, consoleErr := consoleCmd.CombinedOutput()
	var consoleLogs interface{}
	if consoleErr != nil {
		h.log.WithError(consoleErr).WithField("output", string(consoleOutput)).Warn("Failed to capture console logs, continuing with screenshot only")
		// Set empty console logs if capture fails
		consoleLogs = []interface{}{}
	} else {
		// Read and parse console logs
		consoleData, err := os.ReadFile(tmpConsoleFile.Name())
		if err != nil {
			h.log.WithError(err).Warn("Failed to read console log file")
			consoleLogs = []interface{}{}
		} else {
			var consoleResult map[string]interface{}
			if err := json.Unmarshal(consoleData, &consoleResult); err != nil {
				h.log.WithError(err).Warn("Failed to parse console log JSON")
				consoleLogs = []interface{}{}
			} else {
				if logs, ok := consoleResult["logs"]; ok {
					consoleLogs = logs
				} else {
					consoleLogs = []interface{}{}
				}
			}
		}
	}
	
	// Check if the screenshot file was actually created and is valid
	fileInfo, err := os.Stat(tmpScreenshotFile.Name())
	if err != nil {
		h.log.WithError(err).Error("Screenshot file was not created")
		http.Error(w, "Screenshot file was not created", http.StatusInternalServerError)
		return
	}
	
	// Check if file has content
	if fileInfo.Size() == 0 {
		h.log.Error("Screenshot file is empty")
		http.Error(w, "Screenshot file is empty", http.StatusInternalServerError)
		return
	}
	
	// Read the screenshot file
	screenshotData, err := os.ReadFile(tmpScreenshotFile.Name())
	if err != nil {
		h.log.WithError(err).Error("Failed to read screenshot file")
		http.Error(w, "Failed to read screenshot", http.StatusInternalServerError)
		return
	}
	
	// Validate that it's a PNG file by checking the magic bytes
	if len(screenshotData) < 8 || !bytes.Equal(screenshotData[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		h.log.Error("Generated file is not a valid PNG")
		http.Error(w, "Generated file is not a valid PNG image", http.StatusInternalServerError)
		return
	}
	
	// Encode to base64
	base64Data := base64.StdEncoding.EncodeToString(screenshotData)
	
	// Return both screenshot and console logs
	response := map[string]interface{}{
		"success": true,
		"screenshot": fmt.Sprintf("data:image/png;base64,%s", base64Data),
		"consoleLogs": consoleLogs,
		"url": req.URL,
		"timestamp": time.Now().Unix(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) getScenarioPortInfo(ctx context.Context, scenarioName string) (*ScenarioPortInfo, error) {
	// For most scenarios, we'll try UI_PORT first, then API_PORT as fallback
	portNames := []string{"UI_PORT", "API_PORT"}
	
	var port int
	var portName string
	var err error
	
	// Try each port name until we find one that works
	for _, name := range portNames {
		cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", scenarioName, name)
		output, portErr := cmd.Output()
		if portErr == nil {
			portStr := strings.TrimSpace(string(output))
			port, err = strconv.Atoi(portStr)
			if err == nil {
				portName = name
				break
			}
		}
	}
	
	if err != nil || port == 0 {
		h.log.WithError(err).WithField("scenario", scenarioName).Error("Failed to get any port for scenario")
		return nil, fmt.Errorf("failed to get port for scenario %s: no valid ports found", scenarioName)
	}

	// Construct URL
	host := "localhost"
	url := fmt.Sprintf("http://%s:%d", host, port)

	// Check if the scenario is running by trying to get its status
	statusCmd := exec.CommandContext(ctx, "vrooli", "scenario", "status", scenarioName)
	statusOutput, err := statusCmd.Output()
	status := "unknown"
	if err == nil {
		statusStr := strings.TrimSpace(string(statusOutput))
		if strings.Contains(strings.ToLower(statusStr), "running") {
			status = "running"
		} else if strings.Contains(strings.ToLower(statusStr), "stopped") {
			status = "stopped"
		}
	}

	h.log.WithFields(logrus.Fields{
		"scenario": scenarioName,
		"port_name": portName,
		"port": port,
		"status": status,
	}).Info("Successfully retrieved scenario port info")

	return &ScenarioPortInfo{
		Port:   port,
		Status: status,
		URL:    url,
	}, nil
}