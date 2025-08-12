package workflow

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	
	"metareasoning-api/internal/domain/common"
)

// Entity represents a workflow domain entity
type Entity struct {
	ID                uuid.UUID       `json:"id"`
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	Type              common.WorkflowType `json:"type"`
	Platform          common.Platform     `json:"platform"`
	Config            json.RawMessage `json:"config"`
	WebhookPath       *string         `json:"webhook_path,omitempty"`
	JobPath           *string         `json:"job_path,omitempty"`
	Schema            json.RawMessage `json:"schema"`
	EstimatedDuration int             `json:"estimated_duration_ms"`
	Version           int             `json:"version"`
	ParentID          *uuid.UUID      `json:"parent_id,omitempty"`
	IsActive          *bool           `json:"is_active"`
	IsBuiltin         *bool           `json:"is_builtin"`
	Tags              []string        `json:"tags"`
	UsageCount        int             `json:"usage_count"`
	SuccessCount      int             `json:"success_count"`
	FailureCount      int             `json:"failure_count"`
	AvgExecutionTime  *int            `json:"avg_execution_time_ms,omitempty"`
	CreatedBy         string          `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// CreateRequest represents a request to create or update a workflow
type CreateRequest struct {
	Name              string          `json:"name" validate:"required,max=100"`
	Description       string          `json:"description"`
	Type              string          `json:"type" validate:"required,max=50"`
	Platform          common.Platform `json:"platform" validate:"required,oneof=n8n windmill both"`
	Config            json.RawMessage `json:"config" validate:"required"`
	WebhookPath       *string         `json:"webhook_path,omitempty"`
	JobPath           *string         `json:"job_path,omitempty"`
	Schema            json.RawMessage `json:"schema"`
	EstimatedDuration int             `json:"estimated_duration_ms" validate:"min=0"`
	Tags              []string        `json:"tags"`
	CreatedBy         string          `json:"created_by,omitempty"`
}

// Query represents a workflow query with filtering and pagination
type Query struct {
	Platform   common.Platform `query:"platform"`
	Type       string     `query:"type"`
	ActiveOnly bool       `query:"active" default:"true"`
	Pagination Pagination `query:",inline"`
}

// ListResponse represents a paginated list of workflows
type ListResponse struct {
	Workflows []*Entity `json:"workflows"`
	Total     int       `json:"total"`
	Page      int       `json:"page"`
	PageSize  int       `json:"page_size"`
	HasNext   bool      `json:"has_next"`
}

// ExecutionHistory represents a single execution record
type ExecutionHistory struct {
	ID            uuid.UUID `json:"id"`
	WorkflowID    uuid.UUID `json:"workflow_id"`
	Status        string    `json:"status"`
	ExecutionTime int       `json:"execution_time_ms"`
	ModelUsed     string    `json:"model_used"`
	CreatedAt     time.Time `json:"created_at"`
	InputSummary  string    `json:"input_summary,omitempty"`
	HasOutput     bool      `json:"has_output"`
}

// HistoryResponse represents execution history for a workflow
type HistoryResponse struct {
	WorkflowID uuid.UUID          `json:"workflow_id"`
	History    []*ExecutionHistory `json:"history"`
	Count      int                 `json:"count"`
}

// Metrics represents performance metrics for a workflow
type Metrics struct {
	TotalExecutions  int        `json:"total_executions"`
	SuccessCount     int        `json:"success_count"`
	FailureCount     int        `json:"failure_count"`
	AvgExecutionTime int        `json:"avg_execution_time"`
	MinExecutionTime int        `json:"min_execution_time"`
	MaxExecutionTime int        `json:"max_execution_time"`
	LastExecution    *time.Time `json:"last_execution,omitempty"`
	SuccessRate      float64    `json:"success_rate"`
	ModelsUsed       []string   `json:"models_used"`
}

// MetricsResponse represents performance metrics response
type MetricsResponse struct {
	TotalExecutions  int        `json:"total_executions"`
	SuccessCount     int        `json:"success_count"`
	FailureCount     int        `json:"failure_count"`
	AvgExecutionTime int        `json:"avg_execution_time"`
	MinExecutionTime int        `json:"min_execution_time"`
	MaxExecutionTime int        `json:"max_execution_time"`
	LastExecution    *time.Time `json:"last_execution,omitempty"`
	SuccessRate      float64    `json:"success_rate"`
	ModelsUsed       []string   `json:"models_used"`
}