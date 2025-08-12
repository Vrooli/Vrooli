package workflow

import (
	"fmt"

	"github.com/google/uuid"
	
	"metareasoning-api/internal/domain/common"
	"metareasoning-api/internal/domain/execution"
)

// Service provides core workflow business operations
type Service interface {
	// Core CRUD operations
	GetWorkflow(id uuid.UUID) (*Entity, error)
	ListWorkflows(query *Query) (*ListResponse, error)
	CreateWorkflow(request *CreateRequest) (*Entity, error)
	UpdateWorkflow(id uuid.UUID, request *CreateRequest) (*Entity, error)
	DeleteWorkflow(id uuid.UUID) error
	
	// Workflow operations
	CloneWorkflow(id uuid.UUID, name string) (*Entity, error)
	SearchWorkflows(query string) ([]*Entity, error)
	
	// Execution operations
	ExecuteWorkflow(id uuid.UUID, request *common.ExecutionRequest) (*common.ExecutionResponse, error)
	GetExecutionHistory(id uuid.UUID, limit int) ([]*ExecutionHistory, error)
	GetWorkflowMetrics(id uuid.UUID) (*MetricsResponse, error)
}

// DefaultService implements the workflow Service interface
type DefaultService struct {
	repo            Repository
	executionRepo   ExecutionRepository
	executionEngine execution.ExecutionEngine
}

// NewDefaultService creates a new workflow service
func NewDefaultService(repo Repository, executionRepo ExecutionRepository, executionEngine execution.ExecutionEngine) Service {
	return &DefaultService{
		repo:            repo,
		executionRepo:   executionRepo,
		executionEngine: executionEngine,
	}
}

// GetWorkflow retrieves a workflow by ID
func (s *DefaultService) GetWorkflow(id uuid.UUID) (*Entity, error) {
	return s.repo.GetByID(id)
}

// ListWorkflows returns paginated workflows with filtering
func (s *DefaultService) ListWorkflows(query *Query) (*ListResponse, error) {
	// Convert user query to repository query
	repoQuery := &RepositoryQuery{
		Platform:   query.Platform,
		Type:       common.WorkflowType(query.Type),
		ActiveOnly: query.ActiveOnly,
		Pagination: &query.Pagination,
	}
	return s.repo.List(repoQuery)
}

// CreateWorkflow creates a new workflow
func (s *DefaultService) CreateWorkflow(request *CreateRequest) (*Entity, error) {
	// Convert request to entity
	entity := &Entity{
		Name:              request.Name,
		Description:       request.Description,
		Type:              common.WorkflowType(request.Type),
		Platform:          request.Platform,
		Config:            request.Config,
		WebhookPath:       request.WebhookPath,
		JobPath:           request.JobPath,
		Schema:            request.Schema,
		EstimatedDuration: request.EstimatedDuration,
		Tags:              request.Tags,
		CreatedBy:         request.CreatedBy,
	}
	
	// Validate the workflow for its target platform
	execWorkflow := &common.WorkflowEntity{
		ID:          entity.ID,
		Name:        entity.Name,
		Platform:    entity.Platform,
		Config:      entity.Config,
		WebhookPath: entity.WebhookPath,
		JobPath:     entity.JobPath,
	}
	
	if err := s.executionEngine.ValidateWorkflow(execWorkflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}
	
	err := s.repo.Create(entity)
	return entity, err
}

// UpdateWorkflow updates an existing workflow (creates new version)
func (s *DefaultService) UpdateWorkflow(id uuid.UUID, request *CreateRequest) (*Entity, error) {
	// Create new version entity
	updated := &Entity{
		Name:              request.Name,
		Description:       request.Description,
		Type:              common.WorkflowType(request.Type),
		Platform:          request.Platform,
		Config:            request.Config,
		WebhookPath:       request.WebhookPath,
		JobPath:           request.JobPath,
		Schema:            request.Schema,
		EstimatedDuration: request.EstimatedDuration,
		Tags:              request.Tags,
		CreatedBy:         request.CreatedBy,
	}
	
	// Validate the updated workflow
	execWorkflow := &common.WorkflowEntity{
		ID:          updated.ID,
		Name:        updated.Name,
		Platform:    updated.Platform,
		Config:      updated.Config,
		WebhookPath: updated.WebhookPath,
		JobPath:     updated.JobPath,
	}
	
	if err := s.executionEngine.ValidateWorkflow(execWorkflow); err != nil {
		return nil, fmt.Errorf("updated workflow validation failed: %w", err)
	}
	
	err := s.repo.Update(id, updated)
	return updated, err
}

// DeleteWorkflow soft deletes a workflow
func (s *DefaultService) DeleteWorkflow(id uuid.UUID) error {
	return s.repo.Delete(id)
}

// CloneWorkflow creates a copy of an existing workflow
func (s *DefaultService) CloneWorkflow(id uuid.UUID, name string) (*Entity, error) {
	return s.repo.Clone(id, name, "system")
}

// SearchWorkflows searches workflows by text query
func (s *DefaultService) SearchWorkflows(query string) ([]*Entity, error) {
	return s.repo.Search(query, 20)
}

// ExecuteWorkflow executes a workflow using the execution engine
func (s *DefaultService) ExecuteWorkflow(id uuid.UUID, request *common.ExecutionRequest) (*common.ExecutionResponse, error) {
	// Get workflow
	wf, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}
	
	// Convert to execution entity
	execWorkflow := &common.WorkflowEntity{
		ID:          wf.ID,
		Name:        wf.Name,
		Platform:    wf.Platform,
		Config:      wf.Config,
		WebhookPath: wf.WebhookPath,
		JobPath:     wf.JobPath,
	}
	
	// Execute workflow
	response, err := s.executionEngine.Execute(execWorkflow, request)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}
	
	// Record execution in database
	if recordErr := s.executionRepo.RecordExecution(id, request, response); recordErr != nil {
		// Log error but don't fail the execution
		// TODO: Add proper logging
		fmt.Printf("Failed to record execution: %v\n", recordErr)
	}
	
	return response, nil
}

// GetExecutionHistory returns execution history for a workflow
func (s *DefaultService) GetExecutionHistory(id uuid.UUID, limit int) ([]*ExecutionHistory, error) {
	return s.executionRepo.GetExecutionHistory(id, limit)
}

// GetWorkflowMetrics returns performance metrics for a workflow
func (s *DefaultService) GetWorkflowMetrics(id uuid.UUID) (*MetricsResponse, error) {
	return s.executionRepo.GetMetrics(id)
}