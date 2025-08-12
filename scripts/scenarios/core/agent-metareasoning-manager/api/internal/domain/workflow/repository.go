package workflow

import (
	"time"

	"github.com/google/uuid"
	
	"metareasoning-api/internal/domain/common"
)

// Repository defines the interface for workflow data access
type Repository interface {
	// Basic CRUD operations
	Create(entity *Entity) error
	GetByID(id uuid.UUID) (*Entity, error)
	Update(id uuid.UUID, entity *Entity) error
	Delete(id uuid.UUID) error
	
	// Query operations
	List(query *RepositoryQuery) (*ListResponse, error)
	Search(searchQuery string, limit int) ([]*Entity, error)
	
	// Advanced operations
	Clone(id uuid.UUID, newName string, clonedBy string) (*Entity, error)
	GetVersions(parentID uuid.UUID) ([]*Entity, error)
	
	// Transaction support
	WithTransaction(fn func(Repository) error) error
}

// ExecutionRepository defines the interface for execution history data access
type ExecutionRepository interface {
	// Record execution
	RecordExecution(workflowID uuid.UUID, req *common.ExecutionRequest, resp *common.ExecutionResponse) error
	
	// Query operations
	GetExecutionHistory(workflowID uuid.UUID, limit int) ([]*ExecutionHistory, error)
	GetMetrics(workflowID uuid.UUID) (*MetricsResponse, error)
	GetSystemStats() (*StatsResponse, error)
	
	// Cleanup operations
	CleanupOldExecutions(olderThan time.Time) error
}

// RepositoryQuery represents a repository-specific query (different from user Query)
type RepositoryQuery struct {
	Platform    common.Platform
	Type        common.WorkflowType
	ActiveOnly  bool
	CreatedBy   string
	Tags        []string
	Pagination  *Pagination
}

// StatsResponse represents system-wide statistics
type StatsResponse struct {
	TotalWorkflows    int        `json:"total_workflows"`
	ActiveWorkflows   int        `json:"active_workflows"`
	TotalExecutions   int        `json:"total_executions"`
	SuccessfulExecs   int        `json:"successful_executions"`
	FailedExecs       int        `json:"failed_executions"`
	AvgExecutionTime  int        `json:"avg_execution_time_ms"`
	LastExecution     *time.Time `json:"last_execution,omitempty"`
	MostUsedWorkflow  string     `json:"most_used_workflow,omitempty"`
	MostUsedModel     string     `json:"most_used_model,omitempty"`
}