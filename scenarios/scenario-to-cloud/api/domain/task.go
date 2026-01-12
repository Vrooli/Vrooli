// Package domain defines the core domain types for the scenario-to-cloud scenario.
//
// This file contains task-related types for the unified task system that supports
// both investigation and fix workflows.
package domain

import (
	"fmt"
	"time"
)

// TaskType identifies the kind of task being executed.
type TaskType string

const (
	// TaskTypeInvestigate is for diagnosing deployment failures without making changes.
	TaskTypeInvestigate TaskType = "investigate"
	// TaskTypeFix is for applying fixes through an iterative diagnose-change-deploy-verify loop.
	TaskTypeFix TaskType = "fix"
)

// Valid returns true if the task type is valid.
func (t TaskType) Valid() bool {
	return t == TaskTypeInvestigate || t == TaskTypeFix
}

// InvestigationEffort defines how deep an investigation should go.
// Only applies to TaskTypeInvestigate.
type InvestigationEffort string

const (
	// EffortChecks performs quick health checks and basic diagnostics.
	EffortChecks InvestigationEffort = "checks"
	// EffortLogs performs log analysis and standard diagnostic procedures.
	EffortLogs InvestigationEffort = "logs"
	// EffortTrace performs full SSH tracing and deep analysis.
	EffortTrace InvestigationEffort = "trace"
)

// Valid returns true if the effort level is valid.
func (e InvestigationEffort) Valid() bool {
	return e == EffortChecks || e == EffortLogs || e == EffortTrace
}

// TaskFocus identifies which components to focus on during the task.
type TaskFocus struct {
	// Harness focuses on scenario-to-cloud deployment infrastructure.
	Harness bool `json:"harness"`
	// Subject focuses on the target scenario being deployed.
	Subject bool `json:"subject"`
}

// Validate ensures at least one focus is selected.
func (f TaskFocus) Validate() error {
	if !f.Harness && !f.Subject {
		return fmt.Errorf("at least one focus (harness or subject) must be selected")
	}
	return nil
}

// FixPermissions defines what kinds of fixes are allowed.
// Only applies to TaskTypeFix.
type FixPermissions struct {
	// Immediate allows VPS hotfixes (restart services, clear disk, etc.).
	Immediate bool `json:"immediate"`
	// Permanent allows code/configuration changes in the codebase.
	Permanent bool `json:"permanent"`
	// Prevention allows implementing monitoring, alerts, or pipeline improvements.
	Prevention bool `json:"prevention"`
}

// Validate ensures at least one permission is selected.
func (p FixPermissions) Validate() error {
	if !p.Immediate && !p.Permanent && !p.Prevention {
		return fmt.Errorf("at least one permission (immediate, permanent, or prevention) must be selected")
	}
	return nil
}

// String returns a comma-separated list of enabled permissions.
func (p FixPermissions) String() string {
	var parts []string
	if p.Immediate {
		parts = append(parts, "immediate")
	}
	if p.Permanent {
		parts = append(parts, "permanent")
	}
	if p.Prevention {
		parts = append(parts, "prevention")
	}
	if len(parts) == 0 {
		return "none"
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "," + parts[i]
	}
	return result
}

// CreateTaskRequest is the unified request structure for creating tasks.
type CreateTaskRequest struct {
	DeploymentID string `json:"deployment_id"`
	TaskType     TaskType `json:"task_type"`
	Focus        TaskFocus `json:"focus"`
	Note         string `json:"note,omitempty"`

	// Investigate-only fields
	Effort InvestigationEffort `json:"effort,omitempty"`

	// Fix-only fields
	Permissions           FixPermissions `json:"permissions,omitempty"`
	SourceInvestigationID string `json:"source_investigation_id,omitempty"`
	MaxIterations         int `json:"max_iterations,omitempty"`

	// Context selection (applies to both)
	IncludeContexts []string `json:"include_contexts,omitempty"`
}

// Validate ensures the request is valid for the given task type.
func (r *CreateTaskRequest) Validate() error {
	if r.DeploymentID == "" {
		return fmt.Errorf("deployment_id is required")
	}

	if !r.TaskType.Valid() {
		return fmt.Errorf("invalid task_type: %s", r.TaskType)
	}

	if err := r.Focus.Validate(); err != nil {
		return err
	}

	switch r.TaskType {
	case TaskTypeInvestigate:
		// Default effort to Logs if not specified
		if r.Effort == "" {
			r.Effort = EffortLogs
		}
		if !r.Effort.Valid() {
			return fmt.Errorf("invalid effort level: %s", r.Effort)
		}

	case TaskTypeFix:
		if err := r.Permissions.Validate(); err != nil {
			return err
		}
		// Default max iterations to 5
		if r.MaxIterations <= 0 {
			r.MaxIterations = 5
		}
		if r.MaxIterations > 10 {
			return fmt.Errorf("max_iterations cannot exceed 10")
		}
	}

	return nil
}

// FixIterationState tracks the state of an iterative fix loop.
// Stored in the Investigation.Details JSON blob.
type FixIterationState struct {
	CurrentIteration int                  `json:"current_iteration"`
	MaxIterations    int                  `json:"max_iterations"`
	Iterations       []FixIterationRecord `json:"iterations"`
	FinalStatus      string               `json:"final_status,omitempty"` // success, max_iterations, agent_gave_up, user_stopped, timeout
}

// FixIterationRecord tracks a single iteration of the fix loop.
type FixIterationRecord struct {
	Number           int       `json:"number"`
	StartedAt        time.Time `json:"started_at"`
	EndedAt          time.Time `json:"ended_at,omitempty"`
	DiagnosisSummary string    `json:"diagnosis_summary,omitempty"`
	ChangesSummary   string    `json:"changes_summary,omitempty"`
	DeployTriggered  bool      `json:"deploy_triggered"`
	VerifyResult     string    `json:"verify_result,omitempty"` // pass, fail, skip
	Outcome          string    `json:"outcome,omitempty"`       // success, continue, gave_up
	AgentRunID       string    `json:"agent_run_id,omitempty"`
}

// ExtendedInvestigationDetails extends InvestigationDetails with task-specific fields.
// This is used for the new task system while maintaining backward compatibility.
type ExtendedInvestigationDetails struct {
	// Base fields (from InvestigationDetails)
	Source                string  `json:"source"`
	RunID                 string  `json:"run_id,omitempty"`
	DurationSecs          int     `json:"duration_seconds,omitempty"`
	TokensUsed            int32   `json:"tokens_used,omitempty"`
	CostEstimate          float64 `json:"cost_estimate,omitempty"`
	OperationMode         string  `json:"operation_mode"`
	TriggerReason         string  `json:"trigger_reason"`
	DeploymentStep        string  `json:"deployment_step,omitempty"`
	SourceInvestigationID string  `json:"source_investigation_id,omitempty"`
	SourceFindings        string  `json:"source_findings,omitempty"`

	// New task fields
	TaskType    TaskType            `json:"task_type,omitempty"`
	Focus       TaskFocus           `json:"focus,omitempty"`
	Effort      InvestigationEffort `json:"effort,omitempty"`
	Permissions FixPermissions      `json:"permissions,omitempty"`

	// Fix loop state
	FixState *FixIterationState `json:"fix_state,omitempty"`
}
