package execution

import (
	"encoding/json"

	"metareasoning-api/internal/domain/common"
)

// Use common types to avoid circular imports
type ExecutionRequest = common.ExecutionRequest
type ExecutionResponse = common.ExecutionResponse
type WorkflowEntity = common.WorkflowEntity

// ExecutionStrategy defines the interface for platform-specific execution strategies
type ExecutionStrategy interface {
	// Execute runs the workflow on the specific platform
	Execute(workflow *WorkflowEntity, req *ExecutionRequest) (*ExecutionResponse, error)
	
	// Validate checks if the workflow is valid for this platform
	Validate(workflow *WorkflowEntity) error
	
	// Export exports workflow to platform-specific format
	Export(workflow *WorkflowEntity, format string) (interface{}, error)
	
	// Import imports workflow from platform-specific format
	Import(data json.RawMessage, name string) (*WorkflowEntity, error)
	
	// GetPlatform returns the platform this strategy handles
	GetPlatform() common.Platform
}

// ExecutionEngine manages execution strategies for different platforms
type ExecutionEngine interface {
	// Execute dispatches execution to the appropriate strategy
	Execute(workflow *WorkflowEntity, req *ExecutionRequest) (*ExecutionResponse, error)
	
	// RegisterStrategy registers a new execution strategy
	RegisterStrategy(platform common.Platform, strategy ExecutionStrategy)
	
	// GetStrategy returns the strategy for a platform
	GetStrategy(platform common.Platform) (ExecutionStrategy, error)
	
	// ListPlatforms returns all supported platforms
	ListPlatforms() []common.Platform
	
	// ValidateWorkflow validates a workflow for its target platform
	ValidateWorkflow(workflow *WorkflowEntity) error
}