package handlers

import (
	"net/http"

	"metareasoning-api/internal/domain/ai"
	"metareasoning-api/internal/domain/workflow"
	"metareasoning-api/internal/pkg/errors"
	"metareasoning-api/internal/pkg/validation"
)

// AIHandler handles HTTP requests for AI-powered operations
type AIHandler struct {
	aiService       ai.GenerationService
	workflowService workflow.Service
	binder          validation.RequestBinder
	errorHandler    ErrorHandler
}

// NewAIHandler creates a new AI handler
func NewAIHandler(
	aiService ai.GenerationService,
	workflowService workflow.Service,
	binder validation.RequestBinder,
	errorHandler ErrorHandler,
) *AIHandler {
	return &AIHandler{
		aiService:       aiService,
		workflowService: workflowService,
		binder:          binder,
		errorHandler:    errorHandler,
	}
}

// GenerateWorkflow handles POST /ai/generate
func (h *AIHandler) GenerateWorkflow(w http.ResponseWriter, r *http.Request) error {
	// Parse request body
	var genRequest ai.GenerationRequest
	if err := h.binder.BindJSON(r, &genRequest); err != nil {
		return errors.WrapValidation(err, "request_body", "generation_request")
	}
	
	// Set default temperature if not provided
	if genRequest.Temperature == 0 {
		genRequest.Temperature = 0.7
	}
	
	// Set default model if not provided
	if genRequest.Model == "" {
		genRequest.Model = "llama3.2"
	}
	
	// Generate workflow
	workflowRequest, err := h.aiService.GenerateWorkflow(&genRequest)
	if err != nil {
		return errors.WithOperation(err, "generate_workflow").
			WithDetail("prompt", genRequest.Prompt).
			WithDetail("platform", genRequest.Platform)
	}
	
	// Create the workflow
	entity, err := h.workflowService.CreateWorkflow(workflowRequest)
	if err != nil {
		return errors.WithOperation(err, "create_generated_workflow").
			WithDetail("name", workflowRequest.Name)
	}
	
	// Prepare response
	response := map[string]interface{}{
		"workflow":    entity,
		"generated_from": map[string]interface{}{
			"prompt":      genRequest.Prompt,
			"platform":    genRequest.Platform,
			"model":       genRequest.Model,
			"temperature": genRequest.Temperature,
		},
		"message": "Workflow generated successfully",
	}
	
	return WriteJSON(w, http.StatusCreated, response)
}

// ValidateModel handles POST /ai/models/{model}/validate
func (h *AIHandler) ValidateModel(w http.ResponseWriter, r *http.Request) error {
	// Get model from URL path
	model := r.URL.Path[len("/ai/models/"):] // Simple path extraction
	if model == "" {
		return errors.NewValidationError("model name is required")
	}
	
	// Remove "/validate" suffix if present
	if len(model) > 9 && model[len(model)-9:] == "/validate" {
		model = model[:len(model)-9]
	}
	
	// Validate model
	err := h.aiService.ValidateModel(model)
	
	response := map[string]interface{}{
		"model":     model,
		"valid":     err == nil,
		"timestamp": getCurrentTimestamp(),
	}
	
	if err != nil {
		response["error"] = err.Error()
	} else {
		response["message"] = "Model is available and ready"
	}
	
	return WriteJSON(w, http.StatusOK, response)
}

// ListModels handles GET /ai/models
func (h *AIHandler) ListModels(w http.ResponseWriter, r *http.Request) error {
	// Get models from AI service
	models, err := h.aiService.ListModels()
	if err != nil {
		return errors.WithOperation(err, "list_ai_models")
	}
	
	response := map[string]interface{}{
		"models":    models,
		"count":     len(models),
		"timestamp": getCurrentTimestamp(),
	}
	
	return WriteJSON(w, http.StatusOK, response)
}

// GenerateWorkflowBatch handles POST /ai/generate/batch
func (h *AIHandler) GenerateWorkflowBatch(w http.ResponseWriter, r *http.Request) error {
	// Parse batch request
	var batchRequest struct {
		Requests []ai.GenerationRequest `json:"requests" validate:"required,min=1,max=10"`
		Parallel bool                   `json:"parallel,omitempty"`
	}
	
	if err := h.binder.BindJSON(r, &batchRequest); err != nil {
		return errors.WrapValidation(err, "request_body", "batch_generation_request")
	}
	
	results := make([]map[string]interface{}, 0, len(batchRequest.Requests))
	errors_list := make([]map[string]interface{}, 0)
	
	// Process each request
	for i, genRequest := range batchRequest.Requests {
		// Set defaults
		if genRequest.Temperature == 0 {
			genRequest.Temperature = 0.7
		}
		if genRequest.Model == "" {
			genRequest.Model = "llama3.2"
		}
		
		// Generate workflow
		workflowRequest, err := h.aiService.GenerateWorkflow(&genRequest)
		if err != nil {
			errors_list = append(errors_list, map[string]interface{}{
				"index": i,
				"error": err.Error(),
				"prompt": genRequest.Prompt,
			})
			continue
		}
		
		// Create workflow
		entity, err := h.workflowService.CreateWorkflow(workflowRequest)
		if err != nil {
			errors_list = append(errors_list, map[string]interface{}{
				"index": i,
				"error": err.Error(),
				"name":  workflowRequest.Name,
			})
			continue
		}
		
		// Add successful result
		result := map[string]interface{}{
			"index":    i,
			"workflow": entity,
			"prompt":   genRequest.Prompt,
			"success":  true,
		}
		results = append(results, result)
	}
	
	// Prepare response
	response := map[string]interface{}{
		"results":       results,
		"success_count": len(results),
		"total_count":   len(batchRequest.Requests),
		"timestamp":     getCurrentTimestamp(),
	}
	
	if len(errors_list) > 0 {
		response["errors"] = errors_list
		response["error_count"] = len(errors_list)
	}
	
	statusCode := http.StatusCreated
	if len(results) == 0 {
		statusCode = http.StatusBadRequest // All requests failed
	} else if len(errors_list) > 0 {
		statusCode = http.StatusPartialContent // Some requests failed
	}
	
	return WriteJSON(w, statusCode, response)
}

// ImproveWorkflow handles POST /ai/improve/{id}
func (h *AIHandler) ImproveWorkflow(w http.ResponseWriter, r *http.Request) error {
	// This would be a feature to improve existing workflows using AI
	// Parse path parameters and improvement request
	var improvementRequest struct {
		Improvements []string `json:"improvements" validate:"required,min=1"`
		Model        string   `json:"model,omitempty"`
		Temperature  float64  `json:"temperature,omitempty"`
	}
	
	if err := h.binder.BindJSON(r, &improvementRequest); err != nil {
		return errors.WrapValidation(err, "request_body", "improvement_request")
	}
	
	// For now, return a placeholder response
	response := map[string]interface{}{
		"message":    "Workflow improvement feature coming soon",
		"requested_improvements": improvementRequest.Improvements,
		"status":     "not_implemented",
		"timestamp":  getCurrentTimestamp(),
	}
	
	return WriteJSON(w, http.StatusNotImplemented, response)
}

// ExplainWorkflow handles POST /ai/explain/{id}
func (h *AIHandler) ExplainWorkflow(w http.ResponseWriter, r *http.Request) error {
	// This would be a feature to explain workflows using AI
	var explanationRequest struct {
		Level   string `json:"level,omitempty" validate:"omitempty,oneof=basic detailed technical"`
		Format  string `json:"format,omitempty" validate:"omitempty,oneof=text markdown html"`
		Model   string `json:"model,omitempty"`
	}
	
	if err := h.binder.BindJSON(r, &explanationRequest); err != nil {
		return errors.WrapValidation(err, "request_body", "explanation_request")
	}
	
	// Set defaults
	if explanationRequest.Level == "" {
		explanationRequest.Level = "basic"
	}
	if explanationRequest.Format == "" {
		explanationRequest.Format = "text"
	}
	
	// For now, return a placeholder response
	response := map[string]interface{}{
		"message":   "Workflow explanation feature coming soon",
		"level":     explanationRequest.Level,
		"format":    explanationRequest.Format,
		"status":    "not_implemented",
		"timestamp": getCurrentTimestamp(),
	}
	
	return WriteJSON(w, http.StatusNotImplemented, response)
}