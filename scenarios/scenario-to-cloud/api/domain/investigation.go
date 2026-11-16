// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// InvestigationStatus represents the current state of an investigation.
type InvestigationStatus string

const (
	InvestigationStatusPending   InvestigationStatus = "pending"
	InvestigationStatusRunning   InvestigationStatus = "running"
	InvestigationStatusCompleted InvestigationStatus = "completed"
	InvestigationStatusFailed    InvestigationStatus = "failed"
	InvestigationStatusCancelled InvestigationStatus = "cancelled"
)

// TransitionStatus advances a persisted investigation through its lifecycle.
// Repeated writes of the current status are safe; terminal states cannot be
// reopened or replaced by a different outcome.
func (i *Investigation) TransitionStatus(next InvestigationStatus) error {
	if i == nil {
		return fmt.Errorf("investigation is nil")
	}
	if !validInvestigationStatus(i.Status) {
		return fmt.Errorf("unknown investigation status %q", i.Status)
	}
	if !validInvestigationStatus(next) {
		return fmt.Errorf("unknown investigation status %q", next)
	}
	if i.Status == next {
		return nil
	}
	allowed := false
	switch i.Status {
	case InvestigationStatusPending:
		allowed = next == InvestigationStatusRunning || next == InvestigationStatusFailed || next == InvestigationStatusCancelled
	case InvestigationStatusRunning:
		allowed = next == InvestigationStatusCompleted || next == InvestigationStatusFailed || next == InvestigationStatusCancelled
	}
	if !allowed {
		return fmt.Errorf("invalid investigation transition %q -> %q", i.Status, next)
	}
	i.Status = next
	return nil
}

func validInvestigationStatus(status InvestigationStatus) bool {
	switch status {
	case InvestigationStatusPending, InvestigationStatusRunning, InvestigationStatusCompleted, InvestigationStatusFailed, InvestigationStatusCancelled:
		return true
	default:
		return false
	}
}

// Investigation represents an investigation record for a deployment failure.
// Investigations are triggered by users to diagnose deployment issues via an agent.
type Investigation struct {
	ID              string              `json:"id"`
	DeploymentID    string              `json:"deployment_id"`
	DeploymentRunID *string             `json:"deployment_run_id,omitempty"`
	Status          InvestigationStatus `json:"status"`
	Findings        *string             `json:"findings,omitempty"`
	Progress        int                 `json:"progress"` // 0-100
	Details         NullRawMessage      `json:"details,omitempty"`
	AgentRunID      *string             `json:"agent_run_id,omitempty"`
	ErrorMessage    *string             `json:"error_message,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	CompletedAt     *time.Time          `json:"completed_at,omitempty"`
}

// InvestigationDetails contains structured metadata about an investigation.
type InvestigationDetails struct {
	Source                string  `json:"source"`                            // "agent-manager"
	RunID                 string  `json:"run_id,omitempty"`                  // Agent run ID
	DurationSecs          int     `json:"duration_seconds,omitempty"`        // Total duration
	TokensUsed            int64   `json:"tokens_used,omitempty"`             // Token consumption
	CostEstimate          float64 `json:"cost_estimate,omitempty"`           // Estimated cost
	OperationMode         string  `json:"operation_mode"`                    // "report-only", "auto-fix", or "fix-application:..."
	TriggerReason         string  `json:"trigger_reason"`                    // "user_requested"
	DeploymentStep        string  `json:"deployment_step,omitempty"`         // Step that failed
	SourceInvestigationID string  `json:"source_investigation_id,omitempty"` // For fix applications: ID of the original investigation
	SourceFindings        string  `json:"source_findings,omitempty"`         // For fix applications: findings from original investigation
}

// CreateInvestigationRequest is the request body for triggering an investigation.
type CreateInvestigationRequest struct {
	AutoFix         bool     `json:"auto_fix,omitempty"`         // Whether to attempt automatic fixes
	Note            string   `json:"note,omitempty"`             // User-provided context
	IncludeContexts []string `json:"include_contexts,omitempty"` // Which context items to include (e.g., "error-info", "deployment-manifest")
}

// ApplyFixesRequest is the request body for applying fixes from an investigation.
type ApplyFixesRequest struct {
	Immediate  bool   `json:"immediate"`  // Apply immediate fixes (VPS commands)
	Permanent  bool   `json:"permanent"`  // Apply permanent fixes (code/config changes)
	Prevention bool   `json:"prevention"` // Apply prevention measures
	Note       string `json:"note,omitempty"`
}

// InvestigationSummary is a lightweight view for list responses.
type InvestigationSummary struct {
	ID                    string              `json:"id"`
	DeploymentID          string              `json:"deployment_id"`
	DeploymentRunID       *string             `json:"deployment_run_id,omitempty"`
	Status                InvestigationStatus `json:"status"`
	Progress              int                 `json:"progress"`
	HasFindings           bool                `json:"has_findings"`
	ErrorMessage          *string             `json:"error_message,omitempty"`
	SourceInvestigationID *string             `json:"source_investigation_id,omitempty"`
	CreatedAt             time.Time           `json:"created_at"`
	CompletedAt           *time.Time          `json:"completed_at,omitempty"`
}

// ToSummary converts an Investigation to its summary form.
func (i *Investigation) ToSummary() InvestigationSummary {
	summary := InvestigationSummary{
		ID:              i.ID,
		DeploymentID:    i.DeploymentID,
		DeploymentRunID: i.DeploymentRunID,
		Status:          i.Status,
		Progress:        i.Progress,
		HasFindings:     i.Findings != nil && *i.Findings != "",
		ErrorMessage:    i.ErrorMessage,
		CreatedAt:       i.CreatedAt,
		CompletedAt:     i.CompletedAt,
	}

	// Extract source_investigation_id from details if present
	if details, err := i.ParseDetails(); err == nil && details != nil && details.SourceInvestigationID != "" {
		summary.SourceInvestigationID = &details.SourceInvestigationID
	}

	return summary
}

// ParseDetails unmarshals the Details JSON into InvestigationDetails.
func (i *Investigation) ParseDetails() (*InvestigationDetails, error) {
	if !i.Details.Valid || len(i.Details.Data) == 0 {
		return nil, nil
	}
	var details InvestigationDetails
	if err := json.Unmarshal(i.Details.Data, &details); err != nil {
		return nil, err
	}
	return &details, nil
}
