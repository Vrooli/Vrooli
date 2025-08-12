package common

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Platform represents the execution platform for a workflow
type Platform string

const (
	PlatformN8n      Platform = "n8n"
	PlatformWindmill Platform = "windmill"
	PlatformBoth     Platform = "both"
)

// String returns the string representation of the platform
func (p Platform) String() string {
	return string(p)
}

// IsValid checks if the platform value is valid
func (p Platform) IsValid() bool {
	switch p {
	case PlatformN8n, PlatformWindmill, PlatformBoth:
		return true
	default:
		return false
	}
}

// WorkflowType represents the type/category of a workflow
type WorkflowType string

const (
	TypeDataProcessing WorkflowType = "data-processing"
	TypeNotification   WorkflowType = "notification"
	TypeAnalysis       WorkflowType = "analysis"
	TypeIntegration    WorkflowType = "integration"
	TypeAutomation     WorkflowType = "automation"
	TypeGenerated      WorkflowType = "generated"
	TypeImported       WorkflowType = "imported"
	TypeCustom         WorkflowType = "custom"
)

// String returns the string representation of the workflow type
func (wt WorkflowType) String() string {
	return string(wt)
}

// ExecutionStatus represents the status of a workflow execution
type ExecutionStatus string

const (
	StatusSuccess ExecutionStatus = "success"
	StatusFailed  ExecutionStatus = "failed"
	StatusRunning ExecutionStatus = "running"
	StatusPending ExecutionStatus = "pending"
)

// String returns the string representation of the execution status
func (es ExecutionStatus) String() string {
	return string(es)
}

// IsValid checks if the execution status is valid
func (es ExecutionStatus) IsValid() bool {
	switch es {
	case StatusSuccess, StatusFailed, StatusRunning, StatusPending:
		return true
	default:
		return false
	}
}

// WorkflowEntity represents a minimal workflow for cross-domain operations
type WorkflowEntity struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Platform    Platform        `json:"platform"`
	Config      json.RawMessage `json:"config"`
	WebhookPath *string         `json:"webhook_path,omitempty"`
	JobPath     *string         `json:"job_path,omitempty"`
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
	ID          uuid.UUID       `json:"id"`
	WorkflowID  uuid.UUID       `json:"workflow_id"`
	Status      ExecutionStatus `json:"status"`
	Data        interface{}     `json:"data,omitempty"`
	Error       string          `json:"error,omitempty"`
	ExecutionMS int             `json:"execution_ms,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
}