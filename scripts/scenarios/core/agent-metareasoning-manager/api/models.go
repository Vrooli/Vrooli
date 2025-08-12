package main

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Models for the API

// Config holds application configuration
type Config struct {
	Port              string
	DatabaseURL       string
	N8nBase          string
	WindmillBase     string
	WindmillWorkspace string
	OllamaBase       string
}

// Workflow represents a metareasoning workflow
type Workflow struct {
	ID                 uuid.UUID              `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Type              string                 `json:"type"`
	Platform          string                 `json:"platform"`
	Config            json.RawMessage        `json:"config"`
	WebhookPath       *string                `json:"webhook_path,omitempty"`
	JobPath           *string                `json:"job_path,omitempty"`
	Schema            json.RawMessage        `json:"schema"`
	EstimatedDuration int                    `json:"estimated_duration_ms"`
	Version           int                    `json:"version"`
	ParentID          *uuid.UUID             `json:"parent_id,omitempty"`
	IsActive          bool                   `json:"is_active"`
	IsBuiltin         bool                   `json:"is_builtin"`
	Tags              []string               `json:"tags"`
	UsageCount        int                    `json:"usage_count"`
	SuccessCount      int                    `json:"success_count"`
	FailureCount      int                    `json:"failure_count"`
	AvgExecutionTime  *int                   `json:"avg_execution_time_ms,omitempty"`
	CreatedBy         string                 `json:"created_by"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// WorkflowCreate represents the request to create/update a workflow
type WorkflowCreate struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Type              string                 `json:"type"`
	Platform          string                 `json:"platform"`
	Config            json.RawMessage        `json:"config"`
	WebhookPath       *string                `json:"webhook_path,omitempty"`
	JobPath           *string                `json:"job_path,omitempty"`
	Schema            json.RawMessage        `json:"schema"`
	EstimatedDuration int                    `json:"estimated_duration_ms"`
	Tags              []string               `json:"tags"`
}

// ExecutionRequest represents a workflow execution request
type ExecutionRequest struct {
	Input     interface{}            `json:"input"`
	Context   string                 `json:"context,omitempty"`
	Model     string                 `json:"model,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionResponse represents the result of a workflow execution
type ExecutionResponse struct {
	ID          uuid.UUID              `json:"id"`
	WorkflowID  uuid.UUID              `json:"workflow_id"`
	Status      string                 `json:"status"`
	Data        interface{}            `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
	ExecutionMS int                    `json:"execution_ms,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ListResponse represents a paginated list of workflows
type ListResponse struct {
	Workflows   []Workflow             `json:"workflows"`
	Total       int                    `json:"total"`
	Page        int                    `json:"page"`
	PageSize    int                    `json:"page_size"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status      string                 `json:"status"`
	Version     string                 `json:"version"`
	Database    bool                   `json:"database"`
	Services    map[string]bool        `json:"services"`
	Workflows   int                    `json:"workflows_count"`
	Timestamp   time.Time              `json:"timestamp"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Query   string                 `json:"query"`
	Results []Workflow             `json:"results"`
	Count   int                    `json:"count"`
}

// GenerateRequest represents a workflow generation request
type GenerateRequest struct {
	Prompt      string                 `json:"prompt"`
	Platform    string                 `json:"platform"`
	Model       string                 `json:"model,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
}

// ImportRequest represents a workflow import request
type ImportRequest struct {
	Platform string                 `json:"platform"`
	Data     json.RawMessage        `json:"data"`
	Name     string                 `json:"name,omitempty"`
}

// CloneRequest represents a workflow clone request
type CloneRequest struct {
	Name string                 `json:"name"`
}

// ExecutionHistory represents a single execution record
type ExecutionHistory struct {
	ID            uuid.UUID              `json:"id"`
	Status        string                 `json:"status"`
	ExecutionTime int                    `json:"execution_time_ms"`
	ModelUsed     string                 `json:"model_used"`
	CreatedAt     time.Time              `json:"created_at"`
	InputSummary  string                 `json:"input_summary,omitempty"`
	HasOutput     bool                   `json:"has_output"`
}

// HistoryResponse represents execution history for a workflow
type HistoryResponse struct {
	WorkflowID uuid.UUID              `json:"workflow_id"`
	History    []ExecutionHistory     `json:"history"`
	Count      int                    `json:"count"`
}

// MetricsResponse represents performance metrics for a workflow
type MetricsResponse struct {
	TotalExecutions int                    `json:"total_executions"`
	SuccessCount    int                    `json:"success_count"`
	FailureCount    int                    `json:"failure_count"`
	AvgExecutionTime int                   `json:"avg_execution_time"`
	MinExecutionTime int                   `json:"min_execution_time"`
	MaxExecutionTime int                   `json:"max_execution_time"`
	LastExecution   *time.Time             `json:"last_execution,omitempty"`
	SuccessRate     float64                `json:"success_rate"`
	ModelsUsed      []string               `json:"models_used"`
}

// StatsResponse represents system-wide statistics
type StatsResponse struct {
	TotalWorkflows    int                   `json:"total_workflows"`
	ActiveWorkflows   int                   `json:"active_workflows"`
	TotalExecutions   int                   `json:"total_executions"`
	SuccessfulExecs   int                   `json:"successful_execs"`
	FailedExecs       int                   `json:"failed_execs"`
	AvgExecutionTime  int                   `json:"avg_execution_time"`
	MostUsedWorkflow  string                `json:"most_used_workflow"`
	MostUsedModel     string                `json:"most_used_model"`
	LastExecution     *time.Time            `json:"last_execution,omitempty"`
}

// PlatformInfo represents information about an execution platform
type PlatformInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      bool                   `json:"status"`
	URL         string                 `json:"url"`
}

// PlatformsResponse represents available platforms
type PlatformsResponse struct {
	Platforms []PlatformInfo         `json:"platforms"`
}

// ModelInfo represents information about an AI model
type ModelInfo struct {
	Name       string                 `json:"name"`
	SizeMB     int                    `json:"size_mb"`
	ModifiedAt time.Time              `json:"modified_at"`
}

// ModelsResponse represents available AI models
type ModelsResponse struct {
	Models []ModelInfo            `json:"models"`
	Count  int                    `json:"count"`
}

// OllamaModel represents a model from Ollama API
type OllamaModel struct {
	Name       string                 `json:"name"`
	Size       int64                  `json:"size"`
	ModifiedAt string                 `json:"modified_at"`
}

// OllamaListResponse represents the response from Ollama list API
type OllamaListResponse struct {
	Models []OllamaModel          `json:"models"`
}

// OllamaGenerateRequest represents a generation request to Ollama
type OllamaGenerateRequest struct {
	Model       string                 `json:"model"`
	Prompt      string                 `json:"prompt"`
	Temperature float64                `json:"temperature,omitempty"`
	Stream      bool                   `json:"stream"`
}

// OllamaGenerateResponse represents Ollama's generation response
type OllamaGenerateResponse struct {
	Response string                 `json:"response"`
	Done     bool                   `json:"done"`
}