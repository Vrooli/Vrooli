package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string    `json:"status"`
	Time   time.Time `json:"time"`
}

// CreateWorkflowRequest represents the request to create a workflow
type CreateWorkflowRequest struct {
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
	
	status := h.workflowService.CheckHealth()
	
	response := HealthResponse{
		Status: status,
		Time:   time.Now(),
	}
	
	json.NewEncoder(w).Encode(response)
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

	workflow, err := h.workflowService.CreateWorkflow(ctx, req.Name, req.FolderPath, req.FlowDefinition, req.AIPrompt)
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