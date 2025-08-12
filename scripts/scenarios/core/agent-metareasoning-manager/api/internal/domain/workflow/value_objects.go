package workflow

import (
	"fmt"
	"strings"
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

// ParsePlatform parses a string to Platform enum
func ParsePlatform(s string) (Platform, error) {
	platform := Platform(strings.ToLower(s))
	if !platform.IsValid() {
		return "", fmt.Errorf("invalid platform: %s. Valid options: n8n, windmill, both", s)
	}
	return platform, nil
}

// AllPlatforms returns all valid platform values
func AllPlatforms() []Platform {
	return []Platform{PlatformN8n, PlatformWindmill, PlatformBoth}
}

// WorkflowType represents the type/category of a workflow
type WorkflowType string

const (
	TypeDataProcessing  WorkflowType = "data-processing"
	TypeNotification    WorkflowType = "notification"
	TypeAnalysis        WorkflowType = "analysis"
	TypeIntegration     WorkflowType = "integration"
	TypeAutomation      WorkflowType = "automation"
	TypeGenerated       WorkflowType = "generated"
	TypeImported        WorkflowType = "imported"
	TypeCustom          WorkflowType = "custom"
)

// String returns the string representation of the workflow type
func (wt WorkflowType) String() string {
	return string(wt)
}

// IsValid checks if the workflow type value is valid
func (wt WorkflowType) IsValid() bool {
	switch wt {
	case TypeDataProcessing, TypeNotification, TypeAnalysis, 
		 TypeIntegration, TypeAutomation, TypeGenerated, 
		 TypeImported, TypeCustom:
		return true
	default:
		return true // Allow custom types for flexibility
	}
}

// ParseWorkflowType parses a string to WorkflowType
func ParseWorkflowType(s string) WorkflowType {
	return WorkflowType(strings.ToLower(s))
}

// CommonWorkflowTypes returns commonly used workflow types
func CommonWorkflowTypes() []WorkflowType {
	return []WorkflowType{
		TypeDataProcessing,
		TypeNotification,
		TypeAnalysis,
		TypeIntegration,
		TypeAutomation,
		TypeGenerated,
		TypeImported,
		TypeCustom,
	}
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

// ParseExecutionStatus parses a string to ExecutionStatus
func ParseExecutionStatus(s string) (ExecutionStatus, error) {
	status := ExecutionStatus(strings.ToLower(s))
	if !status.IsValid() {
		return "", fmt.Errorf("invalid execution status: %s. Valid options: success, failed, running, pending", s)
	}
	return status, nil
}

// Pagination represents pagination parameters
type Pagination struct {
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"page_size" validate:"min=1,max=100"`
	SortBy   string `json:"sort_by,omitempty"`
	SortDir  string `json:"sort_dir,omitempty" validate:"omitempty,oneof=ASC DESC"`
}

// NewPagination creates a new Pagination with defaults
func NewPagination(page, pageSize int) Pagination {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset calculates the SQL offset for the pagination
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the SQL limit for the pagination
func (p Pagination) Limit() int {
	return p.PageSize
}