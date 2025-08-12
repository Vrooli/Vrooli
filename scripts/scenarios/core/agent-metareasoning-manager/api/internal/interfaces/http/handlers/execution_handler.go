package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"metareasoning-api/internal/domain/common"
	"metareasoning-api/internal/domain/workflow"
	"metareasoning-api/internal/pkg/errors"
	"metareasoning-api/internal/pkg/validation"
)

// ExecutionHandler handles HTTP requests for workflow execution operations
type ExecutionHandler struct {
	workflowService workflow.Service
	binder          validation.RequestBinder
	errorHandler    ErrorHandler
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler(workflowService workflow.Service, binder validation.RequestBinder, errorHandler ErrorHandler) *ExecutionHandler {
	return &ExecutionHandler{
		workflowService: workflowService,
		binder:          binder,
		errorHandler:    errorHandler,
	}
}

// ExecuteWorkflow handles POST /workflows/{id}/execute
func (h *ExecutionHandler) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) error {
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
	var execRequest common.ExecutionRequest
	if err := h.binder.BindJSON(r, &execRequest); err != nil {
		return errors.WrapValidation(err, "request_body", "execution_request")
	}
	
	// Set default model if not provided
	if execRequest.Model == "" {
		execRequest.Model = "llama3.2"
	}
	
	// Call service
	response, err := h.workflowService.ExecuteWorkflow(id, &execRequest)
	if err != nil {
		return errors.WithResource(err, "workflow_execution", id).
			WithDetail("model", execRequest.Model)
	}
	
	// Determine HTTP status based on execution status
	statusCode := http.StatusOK
	if response.Status == common.StatusFailed {
		statusCode = http.StatusUnprocessableEntity // 422 - execution failed but request was valid
	}
	
	// Success response
	return WriteJSON(w, statusCode, response)
}

// ExecuteWorkflowAsync handles POST /workflows/{id}/execute/async
func (h *ExecutionHandler) ExecuteWorkflowAsync(w http.ResponseWriter, r *http.Request) error {
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
	var execRequest common.ExecutionRequest
	if err := h.binder.BindJSON(r, &execRequest); err != nil {
		return errors.WrapValidation(err, "request_body", "execution_request")
	}
	
	// Set default model if not provided
	if execRequest.Model == "" {
		execRequest.Model = "llama3.2"
	}
	
	// For async execution, we would typically queue the job and return immediately
	// For now, we'll simulate async by executing normally but returning different response
	response, err := h.workflowService.ExecuteWorkflow(id, &execRequest)
	if err != nil {
		return errors.WithResource(err, "workflow_execution_async", id).
			WithDetail("model", execRequest.Model)
	}
	
	// Create async response
	asyncResponse := map[string]interface{}{
		"execution_id": response.ID,
		"workflow_id":  response.WorkflowID,
		"status":       "accepted",
		"message":      "Workflow execution started",
		"estimated_duration_ms": 5000, // placeholder
		"check_url":    "/workflows/" + id.String() + "/executions/" + response.ID.String(),
	}
	
	// Return 202 Accepted for async operations
	return WriteJSON(w, http.StatusAccepted, asyncResponse)
}

// GetExecutionStatus handles GET /workflows/{id}/executions/{execution_id}
func (h *ExecutionHandler) GetExecutionStatus(w http.ResponseWriter, r *http.Request) error {
	// Parse path parameters
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		return errors.NewValidationError("workflow ID is required")
	}
	
	executionIDStr, exists := vars["execution_id"]
	if !exists {
		return errors.NewValidationError("execution ID is required")
	}
	
	workflowID, err := uuid.Parse(idStr)
	if err != nil {
		return errors.NewValidationError("invalid workflow ID format").
			WithDetail("workflow_id", idStr)
	}
	
	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		return errors.NewValidationError("invalid execution ID format").
			WithDetail("execution_id", executionIDStr)
	}
	
	// Get execution history to find this specific execution
	history, err := h.workflowService.GetExecutionHistory(workflowID, 100)
	if err != nil {
		return errors.WithResource(err, "execution_history", workflowID)
	}
	
	// Find the specific execution
	var execution *workflow.ExecutionHistory
	for _, exec := range history {
		if exec.ID == executionID {
			execution = exec
			break
		}
	}
	
	if execution == nil {
		return errors.NewNotFoundError("execution", executionID)
	}
	
	// Success response
	return WriteJSON(w, http.StatusOK, execution)
}

// ValidateWorkflow handles POST /workflows/{id}/validate
func (h *ExecutionHandler) ValidateWorkflow(w http.ResponseWriter, r *http.Request) error {
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
	
	// Get workflow to validate
	entity, err := h.workflowService.GetWorkflow(id)
	if err != nil {
		return errors.WithResource(err, "workflow", id)
	}
	
	if entity == nil {
		return errors.NewNotFoundError("workflow", id)
	}
	
	// Create minimal execution request for validation
	testRequest := &common.ExecutionRequest{
		Input:   map[string]interface{}{"test": true},
		Context: "validation",
		Model:   "llama3.2",
	}
	
	// Note: In a real implementation, we might have a separate validation method
	// For now, we'll try a dry-run execution to validate the workflow
	_, err = h.workflowService.ExecuteWorkflow(id, testRequest)
	
	validationResult := map[string]interface{}{
		"workflow_id": id,
		"valid":       err == nil,
		"timestamp":   getCurrentTimestamp(),
	}
	
	if err != nil {
		validationResult["errors"] = []string{err.Error()}
		validationResult["valid"] = false
	}
	
	// Always return 200 for validation results, even if workflow is invalid
	return WriteJSON(w, http.StatusOK, validationResult)
}

// EstimateExecutionTime handles POST /workflows/{id}/estimate
func (h *ExecutionHandler) EstimateExecutionTime(w http.ResponseWriter, r *http.Request) error {
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
	
	// Parse request body for estimation parameters
	var estimateRequest struct {
		Model string                 `json:"model,omitempty"`
		Input map[string]interface{} `json:"input,omitempty"`
	}
	
	if err := h.binder.BindJSON(r, &estimateRequest); err != nil {
		return errors.WrapValidation(err, "request_body", "estimation_request")
	}
	
	// Get workflow and its metrics
	entity, err := h.workflowService.GetWorkflow(id)
	if err != nil {
		return errors.WithResource(err, "workflow", id)
	}
	
	if entity == nil {
		return errors.NewNotFoundError("workflow", id)
	}
	
	// Get historical metrics
	metrics, err := h.workflowService.GetWorkflowMetrics(id)
	if err != nil {
		return errors.WithResource(err, "workflow_metrics", id)
	}
	
	// Calculate estimation based on historical data and workflow characteristics
	var estimatedTime int
	if metrics.TotalExecutions > 0 {
		estimatedTime = metrics.AvgExecutionTime
	} else {
		// Use configured estimated duration or default
		estimatedTime = entity.EstimatedDuration
		if estimatedTime == 0 {
			estimatedTime = 5000 // 5 second default
		}
	}
	
	// Apply model-specific multipliers (placeholder logic)
	modelMultiplier := 1.0
	if estimateRequest.Model != "" && estimateRequest.Model != "llama3.2" {
		modelMultiplier = 1.2 // Different models might be slower
	}
	
	finalEstimate := int(float64(estimatedTime) * modelMultiplier)
	
	response := map[string]interface{}{
		"workflow_id":               id,
		"estimated_duration_ms":     finalEstimate,
		"confidence":                getConfidenceLevel(metrics.TotalExecutions),
		"based_on_executions":       metrics.TotalExecutions,
		"avg_historical_time_ms":    metrics.AvgExecutionTime,
		"model":                     getModelName(estimateRequest.Model),
		"timestamp":                 getCurrentTimestamp(),
	}
	
	return WriteJSON(w, http.StatusOK, response)
}

// Helper functions

func getConfidenceLevel(executions int) string {
	if executions >= 100 {
		return "high"
	} else if executions >= 10 {
		return "medium"
	} else if executions >= 1 {
		return "low"
	}
	return "none"
}

func getModelName(model string) string {
	if model == "" {
		return "llama3.2"
	}
	return model
}

func getCurrentTimestamp() string {
	// Placeholder - would use time.Now().Format(time.RFC3339) in real implementation
	return "2025-08-12T10:00:00Z"
}