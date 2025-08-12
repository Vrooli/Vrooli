package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"metareasoning-api/internal/domain/common"
	"metareasoning-api/internal/domain/integration"
	"metareasoning-api/internal/domain/workflow"
	"metareasoning-api/internal/pkg/errors"
	"metareasoning-api/internal/pkg/validation"
)

// IntegrationHandler handles HTTP requests for workflow integration operations
type IntegrationHandler struct {
	integrationService integration.Service
	workflowService    workflow.Service
	binder             validation.RequestBinder
	errorHandler       ErrorHandler
}

// NewIntegrationHandler creates a new integration handler
func NewIntegrationHandler(
	integrationService integration.Service,
	workflowService workflow.Service,
	binder validation.RequestBinder,
	errorHandler ErrorHandler,
) *IntegrationHandler {
	return &IntegrationHandler{
		integrationService: integrationService,
		workflowService:    workflowService,
		binder:             binder,
		errorHandler:       errorHandler,
	}
}

// ImportWorkflow handles POST /integration/import
func (h *IntegrationHandler) ImportWorkflow(w http.ResponseWriter, r *http.Request) error {
	// Parse request body
	var importRequest integration.ImportRequest
	if err := h.binder.BindJSON(r, &importRequest); err != nil {
		return errors.WrapValidation(err, "request_body", "import_request")
	}
	
	// Validate the import data
	if err := h.integrationService.ValidateImportData(importRequest.Platform, importRequest.Data); err != nil {
		return errors.WrapValidation(err, "import_data", string(importRequest.Platform))
	}
	
	// Import the workflow
	workflowRequest, err := h.integrationService.ImportWorkflow(&importRequest)
	if err != nil {
		return errors.WithOperation(err, "import_workflow").
			WithDetail("platform", importRequest.Platform)
	}
	
	// Create the imported workflow
	entity, err := h.workflowService.CreateWorkflow(workflowRequest)
	if err != nil {
		return errors.WithOperation(err, "create_imported_workflow").
			WithDetail("name", workflowRequest.Name)
	}
	
	// Prepare response
	response := map[string]interface{}{
		"workflow": entity,
		"imported_from": map[string]interface{}{
			"platform": importRequest.Platform,
			"name":     importRequest.Name,
		},
		"message": "Workflow imported successfully",
	}
	
	return WriteJSON(w, http.StatusCreated, response)
}

// ExportWorkflow handles GET /workflows/{id}/export
func (h *IntegrationHandler) ExportWorkflow(w http.ResponseWriter, r *http.Request) error {
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
	
	// Get format from query parameter
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json" // default format
	}
	
	// Export the workflow
	exportResponse, err := h.integrationService.ExportWorkflow(id, format)
	if err != nil {
		return errors.WithResource(err, "workflow_export", id).
			WithDetail("format", format)
	}
	
	return WriteJSON(w, http.StatusOK, exportResponse)
}

// GetPlatformStatus handles GET /integration/platforms
func (h *IntegrationHandler) GetPlatformStatus(w http.ResponseWriter, r *http.Request) error {
	// Get platform status from integration service
	platforms, err := h.integrationService.GetPlatformStatus()
	if err != nil {
		return errors.WithOperation(err, "get_platform_status")
	}
	
	response := map[string]interface{}{
		"platforms": platforms,
		"count":     len(platforms),
		"timestamp": getCurrentTimestamp(),
	}
	
	return WriteJSON(w, http.StatusOK, response)
}

// ValidateImportData handles POST /integration/validate
func (h *IntegrationHandler) ValidateImportData(w http.ResponseWriter, r *http.Request) error {
	// Parse validation request
	var validateRequest struct {
		Platform string          `json:"platform" validate:"required"`
		Data     json.RawMessage `json:"data" validate:"required"`
	}
	
	if err := h.binder.BindJSON(r, &validateRequest); err != nil {
		return errors.WrapValidation(err, "request_body", "validation_request")
	}
	
	// Parse platform
	platform, err := parseCommonPlatform(validateRequest.Platform)
	if err != nil {
		return errors.NewValidationError("invalid platform").
			WithDetail("platform", validateRequest.Platform)
	}
	
	// Validate the import data
	err = h.integrationService.ValidateImportData(platform, validateRequest.Data)
	
	response := map[string]interface{}{
		"platform":  validateRequest.Platform,
		"valid":     err == nil,
		"timestamp": getCurrentTimestamp(),
	}
	
	if err != nil {
		response["errors"] = []string{err.Error()}
		response["valid"] = false
	} else {
		response["message"] = "Import data is valid"
	}
	
	return WriteJSON(w, http.StatusOK, response)
}

// BatchImport handles POST /integration/import/batch
func (h *IntegrationHandler) BatchImport(w http.ResponseWriter, r *http.Request) error {
	// Parse batch import request
	var batchRequest struct {
		Imports []integration.ImportRequest `json:"imports" validate:"required,min=1,max=10"`
	}
	
	if err := h.binder.BindJSON(r, &batchRequest); err != nil {
		return errors.WrapValidation(err, "request_body", "batch_import_request")
	}
	
	results := make([]map[string]interface{}, 0, len(batchRequest.Imports))
	errors_list := make([]map[string]interface{}, 0)
	
	// Process each import request
	for i, importRequest := range batchRequest.Imports {
		// Validate import data
		if err := h.integrationService.ValidateImportData(importRequest.Platform, importRequest.Data); err != nil {
			errors_list = append(errors_list, map[string]interface{}{
				"index":    i,
				"error":    err.Error(),
				"platform": importRequest.Platform,
				"name":     importRequest.Name,
			})
			continue
		}
		
		// Import workflow
		workflowRequest, err := h.integrationService.ImportWorkflow(&importRequest)
		if err != nil {
			errors_list = append(errors_list, map[string]interface{}{
				"index":    i,
				"error":    err.Error(),
				"platform": importRequest.Platform,
				"name":     importRequest.Name,
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
			"platform": importRequest.Platform,
			"success":  true,
		}
		results = append(results, result)
	}
	
	// Prepare response
	response := map[string]interface{}{
		"results":       results,
		"success_count": len(results),
		"total_count":   len(batchRequest.Imports),
		"timestamp":     getCurrentTimestamp(),
	}
	
	if len(errors_list) > 0 {
		response["errors"] = errors_list
		response["error_count"] = len(errors_list)
	}
	
	statusCode := http.StatusCreated
	if len(results) == 0 {
		statusCode = http.StatusBadRequest // All imports failed
	} else if len(errors_list) > 0 {
		statusCode = http.StatusPartialContent // Some imports failed
	}
	
	return WriteJSON(w, statusCode, response)
}

// GetExportFormats handles GET /integration/export/formats
func (h *IntegrationHandler) GetExportFormats(w http.ResponseWriter, r *http.Request) error {
	// Get platform from query parameter
	platform := r.URL.Query().Get("platform")
	
	// Define supported export formats
	formats := map[string]interface{}{
		"json": map[string]interface{}{
			"name":        "JSON",
			"description": "Standard JSON format",
			"extension":   ".json",
			"mime_type":   "application/json",
		},
		"yaml": map[string]interface{}{
			"name":        "YAML",
			"description": "YAML format",
			"extension":   ".yaml",
			"mime_type":   "application/x-yaml",
		},
	}
	
	// Platform-specific formats
	if platform == "n8n" {
		formats["n8n"] = map[string]interface{}{
			"name":        "n8n Native",
			"description": "n8n native workflow format",
			"extension":   ".json",
			"mime_type":   "application/json",
		}
	} else if platform == "windmill" {
		formats["windmill"] = map[string]interface{}{
			"name":        "Windmill Script",
			"description": "Windmill script format",
			"extension":   ".ts",
			"mime_type":   "text/typescript",
		}
	}
	
	response := map[string]interface{}{
		"formats":   formats,
		"platform":  platform,
		"count":     len(formats),
		"timestamp": getCurrentTimestamp(),
	}
	
	return WriteJSON(w, http.StatusOK, response)
}

// Helper functions

func parseCommonPlatform(platform string) (common.Platform, error) {
	p := common.Platform(platform)
	if !p.IsValid() {
		return "", errors.NewValidationError("invalid platform value")
	}
	return p, nil
}