package queue

import (
	"context"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

// ExecutionResult represents the outcome of a task execution.
type ExecutionResult struct {
	Success          bool
	Error            string
	Output           string
	Message          string
	RateLimited      bool
	RetryAfter       int
	MaxTurnsExceeded bool
	IdleTimeout      bool
}

// ExecutionManagerAPI defines the interface for task execution management.
// This separates the "doing the work" from the "deciding what work to do".
type ExecutionManagerAPI interface {
	// ExecuteTask executes a single task and returns the result.
	// This is the main entry point for task execution.
	ExecuteTask(ctx context.Context, task tasks.TaskItem) (*ExecutionResult, error)

	// GetExecutionDir returns the directory path for a specific execution.
	GetExecutionDir(taskID, executionID string) string

	// GetExecutionFilePath returns the full path to a file within an execution directory.
	GetExecutionFilePath(taskID, executionID, filename string) string

	// LatestExecutionOutputPath returns the absolute path to the most recent execution output.
	LatestExecutionOutputPath(taskID string) string
}

// Verify ExecutionManager implements the interface at compile time.
// This will be added once the implementation is complete.
