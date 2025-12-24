package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// Resume validation errors
var (
	// ErrExecutionNotFound indicates the execution ID doesn't exist.
	ErrExecutionNotFound = errors.New("execution not found")

	// ErrExecutionNotResumable indicates the execution is in a state that cannot be resumed.
	ErrExecutionNotResumable = errors.New("execution is not resumable")

	// ErrExecutionStillRunning indicates the execution is still in progress.
	ErrExecutionStillRunning = errors.New("execution is still running")

	// ErrExecutionPending indicates the execution hasn't started yet.
	ErrExecutionPending = errors.New("execution has not started")

	// ErrNoCheckpointAvailable indicates no successful steps were completed.
	ErrNoCheckpointAvailable = errors.New("no checkpoint available - no steps completed successfully")

	// ErrWorkflowChanged indicates the workflow was modified after execution started.
	ErrWorkflowChanged = errors.New("workflow has been modified since execution - step indices may not match")

	// ErrWorkflowNotFound indicates the workflow no longer exists.
	ErrWorkflowNotFound = errors.New("workflow not found")
)

// ValidateResumable checks if an execution can be resumed.
// Returns nil if resumable, or a specific error explaining why not.
func (s *WorkflowService) ValidateResumable(ctx context.Context, executionID uuid.UUID) error {
	if s == nil {
		return errors.New("workflow service is nil")
	}

	// 1. Check execution exists
	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return ErrExecutionNotFound
		}
		return fmt.Errorf("get execution: %w", err)
	}

	// 2. Check execution status
	switch execution.Status {
	case database.ExecutionStatusRunning:
		return ErrExecutionStillRunning
	case database.ExecutionStatusPending:
		return ErrExecutionPending
	case database.ExecutionStatusCompleted:
		return ErrExecutionNotResumable // Already succeeded, nothing to resume
	case database.ExecutionStatusFailed:
		// Failed executions can be resumed - continue validation
	default:
		// Unknown status - allow resume attempt
	}

	// 3. Check checkpoint exists by reading timeline
	checkpoint, err := s.ExtractCheckpointState(ctx, executionID)
	if err != nil {
		return fmt.Errorf("extract checkpoint: %w", err)
	}
	if checkpoint.LastStepIndex < 0 {
		return ErrNoCheckpointAvailable
	}

	// 4. Verify workflow still exists
	workflow, err := s.repo.GetWorkflow(ctx, execution.WorkflowID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return ErrWorkflowNotFound
		}
		return fmt.Errorf("get workflow: %w", err)
	}

	// 5. Check workflow version hasn't changed (user-confirmed design decision)
	// This prevents step index mismatches when workflow structure changed
	if checkpoint.WorkflowVersion > 0 && workflow.Version != checkpoint.WorkflowVersion {
		return fmt.Errorf("%w: execution used version %d, current version is %d",
			ErrWorkflowChanged, checkpoint.WorkflowVersion, workflow.Version)
	}

	return nil
}

// ResumeRequest contains parameters for resuming an execution.
type ResumeRequest struct {
	// Parameters are optional overrides merged with original execution params.
	Parameters map[string]any `json:"parameters,omitempty"`

	// ResumeURL is an optional URL to navigate the browser to before resuming.
	// Useful when the browser state needs to be re-established after session loss.
	ResumeURL string `json:"resume_url,omitempty"`
}

// CanResume is a quick check to determine if an execution can potentially be resumed.
// Unlike ValidateResumable, this doesn't load the full checkpoint state.
func (s *WorkflowService) CanResume(ctx context.Context, executionID uuid.UUID) (bool, string) {
	if s == nil {
		return false, "service not available"
	}

	execution, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return false, "execution not found"
	}

	switch execution.Status {
	case database.ExecutionStatusRunning:
		return false, "execution is still running"
	case database.ExecutionStatusPending:
		return false, "execution has not started"
	case database.ExecutionStatusCompleted:
		return false, "execution already completed successfully"
	case database.ExecutionStatusFailed:
		return true, ""
	default:
		return true, ""
	}
}
