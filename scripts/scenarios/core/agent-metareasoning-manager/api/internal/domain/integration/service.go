package integration

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	
	"metareasoning-api/internal/domain/common"
	"metareasoning-api/internal/domain/execution"
	"metareasoning-api/internal/domain/workflow"
)

// Service handles workflow import/export and platform integration
type Service interface {
	// Import operations
	ImportWorkflow(request *ImportRequest) (*workflow.CreateRequest, error)
	
	// Export operations  
	ExportWorkflow(workflowID uuid.UUID, format string) (*ExportResponse, error)
	
	// Platform status
	GetPlatformStatus() ([]*PlatformInfo, error)
	
	// Validation
	ValidateImportData(platform common.Platform, data json.RawMessage) error
}

// ImportRequest represents a workflow import request
type ImportRequest struct {
	Platform common.Platform `json:"platform" validate:"required"`
	Data     json.RawMessage `json:"data" validate:"required"`
	Name     string          `json:"name,omitempty"`
}

// ExportResponse represents a workflow export response
type ExportResponse struct {
	Format   string      `json:"format"`
	Platform string      `json:"platform"`
	Data     interface{} `json:"data"`
}

// PlatformInfo represents information about an execution platform
type PlatformInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      bool   `json:"status"`
	URL         string `json:"url"`
}

// DefaultIntegrationService implements the integration Service
type DefaultIntegrationService struct {
	workflowRepo    workflow.Repository
	executionEngine execution.ExecutionEngine
	config          *Config
}

// Config holds integration service configuration
type Config struct {
	N8nBase          string
	WindmillBase     string
	WindmillWorkspace string
}

// NewDefaultIntegrationService creates a new integration service
func NewDefaultIntegrationService(workflowRepo workflow.Repository, executionEngine execution.ExecutionEngine, config *Config) Service {
	return &DefaultIntegrationService{
		workflowRepo:    workflowRepo,
		executionEngine: executionEngine,
		config:          config,
	}
}

// ImportWorkflow imports a workflow from external platform format
func (s *DefaultIntegrationService) ImportWorkflow(request *ImportRequest) (*workflow.CreateRequest, error) {
	// Validate platform
	if !request.Platform.IsValid() {
		return nil, fmt.Errorf("invalid platform: %s", request.Platform)
	}
	
	// Validate import data
	if err := s.ValidateImportData(request.Platform, request.Data); err != nil {
		return nil, fmt.Errorf("import data validation failed: %w", err)
	}
	
	// Get the appropriate execution strategy
	strategy, err := s.executionEngine.GetStrategy(request.Platform)
	if err != nil {
		return nil, fmt.Errorf("no strategy available for platform %s: %w", request.Platform, err)
	}
	
	// Import using the strategy
	workflowEntity, err := strategy.Import(request.Data, request.Name)
	if err != nil {
		return nil, fmt.Errorf("platform import failed: %w", err)
	}
	
	// Convert to workflow create request
	createReq := &workflow.CreateRequest{
		Name:              workflowEntity.Name,
		Description:       "Imported from " + request.Platform.String(),
		Type:              "imported",
		Platform:          workflowEntity.Platform,
		Config:            workflowEntity.Config,
		WebhookPath:       workflowEntity.WebhookPath,
		JobPath:           workflowEntity.JobPath,
		EstimatedDuration: 2000, // Default estimate
		Tags:              []string{"imported", request.Platform.String()},
	}
	
	return createReq, nil
}

// ExportWorkflow exports a workflow to platform-specific format
func (s *DefaultIntegrationService) ExportWorkflow(workflowID uuid.UUID, format string) (*ExportResponse, error) {
	// Get workflow from repository
	wf, err := s.workflowRepo.GetByID(workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}
	
	// Get the appropriate execution strategy
	strategy, err := s.executionEngine.GetStrategy(wf.Platform)
	if err != nil {
		return nil, fmt.Errorf("no strategy available for platform %s: %w", wf.Platform, err)
	}
	
	// Convert to execution entity
	workflowEntity := &execution.WorkflowEntity{
		ID:          wf.ID,
		Name:        wf.Name,
		Platform:    wf.Platform,
		Config:      wf.Config,
		WebhookPath: wf.WebhookPath,
		JobPath:     wf.JobPath,
	}
	
	// Export using the strategy
	exportData, err := strategy.Export(workflowEntity, format)
	if err != nil {
		return nil, fmt.Errorf("platform export failed: %w", err)
	}
	
	return &ExportResponse{
		Format:   format,
		Platform: wf.Platform.String(),
		Data:     exportData,
	}, nil
}

// GetPlatformStatus returns the status of all execution platforms
func (s *DefaultIntegrationService) GetPlatformStatus() ([]*PlatformInfo, error) {
	platforms := []*PlatformInfo{
		{
			Name:        "n8n",
			Description: "n8n workflow automation platform",
			URL:         s.config.N8nBase,
			Status:      s.checkPlatformHealth(s.config.N8nBase),
		},
		{
			Name:        "windmill",
			Description: "Windmill script execution platform",
			URL:         s.config.WindmillBase,
			Status:      s.checkPlatformHealth(s.config.WindmillBase),
		},
	}
	
	return platforms, nil
}

// ValidateImportData validates import data for a specific platform
func (s *DefaultIntegrationService) ValidateImportData(platform common.Platform, data json.RawMessage) error {
	// Basic JSON validation
	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	
	// Platform-specific validation
	switch platform {
	case common.PlatformN8n:
		return s.validateN8nImportData(data)
	case common.PlatformWindmill:
		return s.validateWindmillImportData(data)
	default:
		return fmt.Errorf("validation not implemented for platform: %s", platform)
	}
}

// validateN8nImportData validates n8n import data
func (s *DefaultIntegrationService) validateN8nImportData(data json.RawMessage) error {
	var n8nData struct {
		Data struct {
			Nodes       []interface{}          `json:"nodes"`
			Connections map[string]interface{} `json:"connections"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(data, &n8nData); err != nil {
		return fmt.Errorf("invalid n8n format: %w", err)
	}
	
	if len(n8nData.Data.Nodes) == 0 {
		return fmt.Errorf("n8n workflow must have at least one node")
	}
	
	return nil
}

// validateWindmillImportData validates Windmill import data
func (s *DefaultIntegrationService) validateWindmillImportData(data json.RawMessage) error {
	var windmillData struct {
		Data struct {
			Language string `json:"language"`
			Content  string `json:"content"`
		} `json:"data"`
	}
	
	if err := json.Unmarshal(data, &windmillData); err != nil {
		return fmt.Errorf("invalid windmill format: %w", err)
	}
	
	if windmillData.Data.Language == "" {
		return fmt.Errorf("windmill workflow must specify a language")
	}
	
	if windmillData.Data.Content == "" {
		return fmt.Errorf("windmill workflow must have content")
	}
	
	return nil
}

// checkPlatformHealth checks if a platform is accessible
func (s *DefaultIntegrationService) checkPlatformHealth(baseURL string) bool {
	// This would normally make an HTTP request to check health
	// For now, return true as a placeholder
	// TODO: Implement actual health checking
	return true
}