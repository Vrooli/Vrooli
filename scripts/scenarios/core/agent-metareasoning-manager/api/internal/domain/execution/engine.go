package execution

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	
	"metareasoning-api/internal/domain/common"
)

// DefaultExecutionEngine implements ExecutionEngine
type DefaultExecutionEngine struct {
	strategies map[common.Platform]ExecutionStrategy
	mutex      sync.RWMutex
}

// NewDefaultExecutionEngine creates a new execution engine
func NewDefaultExecutionEngine() ExecutionEngine {
	return &DefaultExecutionEngine{
		strategies: make(map[common.Platform]ExecutionStrategy),
	}
}

// Execute dispatches execution to the appropriate strategy
func (e *DefaultExecutionEngine) Execute(wf *WorkflowEntity, req *ExecutionRequest) (*ExecutionResponse, error) {
	startTime := time.Now()
	
	// Set default model if not specified
	if req.Model == "" {
		req.Model = "llama3.2"
	}
	
	// Get strategy for platform
	strategy, err := e.GetStrategy(wf.Platform)
	if err != nil {
		return nil, err
	}
	
	// Validate workflow before execution
	if err := strategy.Validate(wf); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}
	
	// Execute workflow
	response, err := strategy.Execute(wf, req)
	if err != nil {
		// Create error response
		response = &ExecutionResponse{
			ID:          uuid.New(),
			WorkflowID:  wf.ID,
			Status:      common.StatusFailed,
			Error:       err.Error(),
			ExecutionMS: int(time.Since(startTime).Milliseconds()),
			Timestamp:   time.Now(),
		}
	} else {
		// Ensure response has correct metadata
		if response.ID == uuid.Nil {
			response.ID = uuid.New()
		}
		response.WorkflowID = wf.ID
		response.ExecutionMS = int(time.Since(startTime).Milliseconds())
		response.Timestamp = time.Now()
		
		// Set status if not already set
		if response.Status == "" {
			response.Status = common.StatusSuccess
		}
	}
	
	return response, nil
}

// RegisterStrategy registers a new execution strategy
func (e *DefaultExecutionEngine) RegisterStrategy(platform common.Platform, strategy ExecutionStrategy) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	e.strategies[platform] = strategy
}

// GetStrategy returns the strategy for a platform
func (e *DefaultExecutionEngine) GetStrategy(platform common.Platform) (ExecutionStrategy, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	strategy, exists := e.strategies[platform]
	if !exists {
		return nil, fmt.Errorf("no strategy registered for platform: %s", platform)
	}
	
	return strategy, nil
}

// ListPlatforms returns all supported platforms
func (e *DefaultExecutionEngine) ListPlatforms() []common.Platform {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	platforms := make([]common.Platform, 0, len(e.strategies))
	for platform := range e.strategies {
		platforms = append(platforms, platform)
	}
	
	return platforms
}

// ValidateWorkflow validates a workflow for its target platform
func (e *DefaultExecutionEngine) ValidateWorkflow(wf *WorkflowEntity) error {
	strategy, err := e.GetStrategy(wf.Platform)
	if err != nil {
		return err
	}
	
	return strategy.Validate(wf)
}