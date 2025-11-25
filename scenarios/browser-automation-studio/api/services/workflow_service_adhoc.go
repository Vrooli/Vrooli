package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// ExecuteAdhocWorkflow executes a workflow definition without persisting it to the database.
// This is useful for testing scenarios where workflows should be ephemeral and not pollute
// the database with test data. The workflow definition is validated and executed directly,
// with execution records still persisted for telemetry and replay purposes.
func (s *WorkflowService) ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error) {
	// Validate workflow definition structure
	if flowDefinition == nil {
		return nil, errors.New("flow_definition is required")
	}

	nodes, hasNodes := flowDefinition["nodes"]
	if !hasNodes {
		return nil, errors.New("flow_definition must contain 'nodes' array")
	}

	nodesArray, ok := nodes.([]interface{})
	if !ok {
		return nil, errors.New("'nodes' must be an array")
	}

	if len(nodesArray) == 0 {
		return nil, errors.New("workflow must contain at least one node")
	}

	_, hasEdges := flowDefinition["edges"]
	if !hasEdges {
		return nil, errors.New("flow_definition must contain 'edges' array")
	}

	// Create ephemeral workflow (temporarily persisted to satisfy FK constraint)
	// This workflow will be auto-deleted when execution is cleaned up (ON DELETE CASCADE)
	// Add timestamp to name to avoid unique constraint violations on (name, folder_path)
	ephemeralID := uuid.New()
	ephemeralName := fmt.Sprintf("%s [adhoc-%s]", name, ephemeralID.String()[:8])

	ephemeralWorkflow := &database.Workflow{
		ID:             ephemeralID,
		Name:           ephemeralName,
		FlowDefinition: database.JSONMap(flowDefinition),
		Version:        0, // Version 0 indicates adhoc/ephemeral workflow
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		// ProjectID is intentionally nil - adhoc workflows have no project context
	}

	// Temporarily persist ephemeral workflow to satisfy executions.workflow_id FK constraint
	// This allows executions table to maintain referential integrity while still being ephemeral
	if err := s.repo.CreateWorkflow(ctx, ephemeralWorkflow); err != nil {
		return nil, fmt.Errorf("failed to create ephemeral workflow: %w", err)
	}

	// Create execution record (persists for telemetry/replay)
	execution := &database.Execution{
		ID:              uuid.New(),
		WorkflowID:      ephemeralWorkflow.ID, // Reference ephemeral workflow ID
		WorkflowVersion: 0,                    // 0 indicates adhoc execution
		Status:          "pending",
		TriggerType:     "adhoc", // Special trigger type for ephemeral workflows
		Parameters:      database.JSONMap(parameters),
		StartedAt:       time.Now(),
		Progress:        0,
		CurrentStep:     "Initializing",
	}

	if err := s.repo.CreateExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Execute asynchronously (same pattern as normal workflow execution)
	// The executeWorkflowAsync method works with any workflow struct,
	// whether persisted or ephemeral
	s.startExecutionRunner(execution, ephemeralWorkflow)
	go s.scheduleAdhocWorkflowCleanup(ephemeralWorkflow.ID, execution.ID)

	return execution, nil
}

// scheduleAdhocWorkflowCleanup monitors an adhoc execution until it reaches a terminal state,
// then waits for the retention period before cleaning up both the execution and ephemeral workflow.
func (s *WorkflowService) scheduleAdhocWorkflowCleanup(workflowID, executionID uuid.UUID) {
	if s == nil || s.repo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), adhocExecutionCleanupTimeout)
	defer cancel()

	ticker := time.NewTicker(adhocExecutionCleanupInterval)
	defer ticker.Stop()

	var terminalObservedAt time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		execution, err := s.repo.GetExecution(ctx, executionID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				terminalObservedAt = time.Now()
				break
			}
			if s.log != nil {
				s.log.WithError(err).WithField("execution_id", executionID).Warn("adhoc cleanup: unable to load execution status")
			}
			continue
		}
		if execution == nil {
			terminalObservedAt = time.Now()
			break
		}
		if IsTerminalExecutionStatus(execution.Status) {
			if terminalObservedAt.IsZero() {
				terminalObservedAt = time.Now()
			}
			if time.Since(terminalObservedAt) >= adhocExecutionRetentionPeriod {
				break
			}
			continue
		}
		terminalObservedAt = time.Time{}
	}

	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()
	if err := s.repo.DeleteExecution(cleanupCtx, executionID); err != nil && s.log != nil {
		s.log.WithError(err).WithField("execution_id", executionID).Warn("adhoc cleanup: failed to delete execution")
	}
	if err := s.repo.DeleteWorkflow(cleanupCtx, workflowID); err != nil && s.log != nil {
		s.log.WithError(err).WithField("workflow_id", workflowID).Warn("adhoc cleanup: failed to delete workflow")
	}
}
