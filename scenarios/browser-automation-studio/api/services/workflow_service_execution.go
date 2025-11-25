package services

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// Package-level configuration for terminal execution statuses and adhoc workflow cleanup.
// These are shared across execution lifecycle methods (see workflow_service_lifecycle.go and workflow_service_adhoc.go).
var (
	terminalExecutionStatuses = map[string]struct{}{
		"completed": {},
		"failed":    {},
		"cancelled": {},
	}
	adhocExecutionCleanupInterval = 5 * time.Second
	adhocExecutionRetentionPeriod = 10 * time.Minute
	adhocExecutionCleanupTimeout  = 6 * time.Hour
)

// IsTerminalExecutionStatus reports whether the supplied status represents a terminal execution state.
func IsTerminalExecutionStatus(status string) bool {
	if strings.TrimSpace(status) == "" {
		return false
	}
	_, ok := terminalExecutionStatuses[strings.ToLower(status)]
	return ok
}

// ExecuteWorkflow executes a workflow.
// This is the primary entry point for executing persisted workflows.
// For ephemeral workflow execution (e.g., testing), see ExecuteAdhocWorkflow in workflow_service_adhoc.go.
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error) {
	// Verify workflow exists
	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if workflow.ProjectID != nil {
		if err := s.syncProjectWorkflows(ctx, *workflow.ProjectID); err != nil {
			return nil, err
		}
		workflow, err = s.repo.GetWorkflow(ctx, workflowID)
		if err != nil {
			return nil, err
		}
	}

	if err := s.ensureWorkflowChangeMetadata(ctx, workflow); err != nil {
		return nil, err
	}

	execution := &database.Execution{
		ID:              uuid.New(),
		WorkflowID:      workflowID,
		WorkflowVersion: workflow.Version,
		Status:          "pending",
		TriggerType:     "manual",
		Parameters:      database.JSONMap(parameters),
		StartedAt:       time.Now(),
		Progress:        0,
		CurrentStep:     "Initializing",
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, err
	}

	// Start async execution (implementation in workflow_service_lifecycle.go)
	s.startExecutionRunner(execution, workflow)

	return execution, nil
}

// GetExecutionScreenshots gets screenshots for an execution
func (s *WorkflowService) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error) {
	return s.repo.GetExecutionScreenshots(ctx, executionID)
}

// GetExecution gets an execution by ID
func (s *WorkflowService) GetExecution(ctx context.Context, id uuid.UUID) (*database.Execution, error) {
	return s.repo.GetExecution(ctx, id)
}

// ListExecutions lists executions with optional workflow filtering
func (s *WorkflowService) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error) {
	return s.repo.ListExecutions(ctx, workflowID, limit, offset)
}

// The following methods are implemented in separate focused files to improve maintainability:
//
// Adhoc workflow execution (workflow_service_adhoc.go):
//   - ExecuteAdhocWorkflow
//   - scheduleAdhocWorkflowCleanup
//
// Execution lifecycle management (workflow_service_lifecycle.go):
//   - StopExecution
//   - startExecutionRunner
//   - storeExecutionCancel
//   - cancelExecutionByID
//   - executeWorkflowAsync
//   - recordExecutionMarker
//   - applyCapabilityError
//   - closeEventSink
//
// Export preview and status (workflow_service_export.go):
//   - DescribeExecutionExport
//
// Automation engine integration (workflow_service_automation.go):
//   - executeWithAutomationEngine
//   - unsupportedAutomationNodes
//   - extractString
