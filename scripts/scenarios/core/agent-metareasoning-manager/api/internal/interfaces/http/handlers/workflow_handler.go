package handlers

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"metareasoning-api/internal/domain/workflow"
	"metareasoning-api/internal/pkg/errors"
	"metareasoning-api/internal/pkg/validation"
)

// WorkflowHandler handles HTTP requests for workflow operations
type WorkflowHandler struct {
	workflowService workflow.Service
	binder          validation.RequestBinder
	errorHandler    ErrorHandler
}

// NewWorkflowHandler creates a new workflow handler
func NewWorkflowHandler(workflowService workflow.Service, binder validation.RequestBinder, errorHandler ErrorHandler) *WorkflowHandler {
	return &WorkflowHandler{
		workflowService: workflowService,
		binder:          binder,
		errorHandler:    errorHandler,
	}
}

// ListWorkflows handles GET /workflows
func (h *WorkflowHandler) ListWorkflows(w http.ResponseWriter, r *http.Request) error {
	// Parse query parameters
	var query workflow.Query
	if err := h.binder.BindQuery(r, &query); err != nil {
		return errors.WrapValidation(err, "query", r.URL.RawQuery)
	}
	
	// Call service
	response, err := h.workflowService.ListWorkflows(&query)
	if err != nil {
		return errors.WithOperation(err, "list_workflows")
	}
	
	// Success response
	return WriteJSON(w, http.StatusOK, response)
}

// GetWorkflow handles GET /workflows/{id}
func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) error {
	// Parse path parameters
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		return errors.NewValidationError("workflow ID is required")
	}
	
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.NewValidationError("invalid workflow ID format").
			WithDetail("id", idStr)
	}
	
	// Call service
	entity, err := h.workflowService.GetWorkflow(id)
	if err != nil {
		return errors.WithResource(err, "workflow", id)
	}
	
	if entity == nil {
		return errors.NewNotFoundError("workflow", id)
	}
	
	// Success response
	return WriteJSON(w, http.StatusOK, entity)
}

// CreateWorkflow handles POST /workflows
func (h *WorkflowHandler) CreateWorkflow(w http.ResponseWriter, r *http.Request) error {
	// Parse request body
	var request workflow.CreateRequest
	if err := h.binder.BindJSON(r, &request); err != nil {
		return errors.WrapValidation(err, "request_body", "workflow_create")
	}
	
	// Set created by from context (would come from auth middleware)
	if request.CreatedBy == "" {
		request.CreatedBy = "system" // Default value
	}
	
	// Call service
	entity, err := h.workflowService.CreateWorkflow(&request)
	if err != nil {
		return errors.WithOperation(err, "create_workflow")
	}
	
	// Success response
	return WriteJSON(w, http.StatusCreated, entity)
}

// UpdateWorkflow handles PUT /workflows/{id}
func (h *WorkflowHandler) UpdateWorkflow(w http.ResponseWriter, r *http.Request) error {
	// Parse path parameters
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		return errors.NewValidationError("workflow ID is required")
	}
	
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.NewValidationError("invalid workflow ID format").
			WithDetail("id", idStr)
	}
	
	// Parse request body
	var request workflow.CreateRequest
	if err := h.binder.BindJSON(r, &request); err != nil {
		return errors.WrapValidation(err, "request_body", "workflow_update")
	}
	
	// Set updated by from context
	if request.CreatedBy == "" {
		request.CreatedBy = "system" // Default value
	}
	
	// Call service
	entity, err := h.workflowService.UpdateWorkflow(id, &request)
	if err != nil {
		return errors.WithResource(err, "workflow", id)
	}
	
	// Success response
	return WriteJSON(w, http.StatusOK, entity)
}

// DeleteWorkflow handles DELETE /workflows/{id}
func (h *WorkflowHandler) DeleteWorkflow(w http.ResponseWriter, r *http.Request) error {
	// Parse path parameters
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		return errors.NewValidationError("workflow ID is required")
	}
	
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.NewValidationError("invalid workflow ID format").
			WithDetail("id", idStr)
	}
	
	// Call service
	err = h.workflowService.DeleteWorkflow(id)
	if err != nil {
		return errors.WithResource(err, "workflow", id)
	}
	
	// Success response (204 No Content)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// CloneWorkflow handles POST /workflows/{id}/clone
func (h *WorkflowHandler) CloneWorkflow(w http.ResponseWriter, r *http.Request) error {
	// Parse path parameters
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		return errors.NewValidationError("workflow ID is required")
	}
	
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.NewValidationError("invalid workflow ID format").
			WithDetail("id", idStr)
	}
	
	// Parse request body for clone parameters
	var cloneRequest struct {
		Name string `json:"name" validate:"required,max=100"`
	}
	
	if err := h.binder.BindJSON(r, &cloneRequest); err != nil {
		return errors.WrapValidation(err, "request_body", "workflow_clone")
	}
	
	// Call service
	entity, err := h.workflowService.CloneWorkflow(id, cloneRequest.Name)
	if err != nil {
		return errors.WithResource(err, "workflow", id)
	}
	
	// Success response
	return WriteJSON(w, http.StatusCreated, entity)
}

// SearchWorkflows handles GET /workflows/search
func (h *WorkflowHandler) SearchWorkflows(w http.ResponseWriter, r *http.Request) error {
	// Get search query parameter
	query := r.URL.Query().Get("q")
	if query == "" {
		return errors.NewValidationError("search query 'q' is required")
	}
	
	// Get optional limit
	limitStr := r.URL.Query().Get("limit")
	limit := 20 // default
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	
	// Call service
	results, err := h.workflowService.SearchWorkflows(query)
	if err != nil {
		return errors.WithOperation(err, "search_workflows").
			WithDetail("query", query)
	}
	
	// Limit results if needed
	if len(results) > limit {
		results = results[:limit]
	}
	
	// Create response
	response := map[string]interface{}{
		"workflows": results,
		"count":     len(results),
		"query":     query,
		"limit":     limit,
	}
	
	// Success response
	return WriteJSON(w, http.StatusOK, response)
}

// GetWorkflowMetrics handles GET /workflows/{id}/metrics
func (h *WorkflowHandler) GetWorkflowMetrics(w http.ResponseWriter, r *http.Request) error {
	// Parse path parameters
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		return errors.NewValidationError("workflow ID is required")
	}
	
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.NewValidationError("invalid workflow ID format").
			WithDetail("id", idStr)
	}
	
	// Call service
	metrics, err := h.workflowService.GetWorkflowMetrics(id)
	if err != nil {
		return errors.WithResource(err, "workflow_metrics", id)
	}
	
	// Success response
	return WriteJSON(w, http.StatusOK, metrics)
}

// GetExecutionHistory handles GET /workflows/{id}/history
func (h *WorkflowHandler) GetExecutionHistory(w http.ResponseWriter, r *http.Request) error {
	// Parse path parameters
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		return errors.NewValidationError("workflow ID is required")
	}
	
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.NewValidationError("invalid workflow ID format").
			WithDetail("id", idStr)
	}
	
	// Get optional limit
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}
	
	// Call service
	history, err := h.workflowService.GetExecutionHistory(id, limit)
	if err != nil {
		return errors.WithResource(err, "workflow_history", id)
	}
	
	// Create response
	response := map[string]interface{}{
		"workflow_id": id,
		"history":     history,
		"count":       len(history),
		"limit":       limit,
	}
	
	// Success response
	return WriteJSON(w, http.StatusOK, response)
}