package queue

// HistoryManagerAPI defines the interface for execution history management.
// This separates history persistence from task execution and scheduling.
type HistoryManagerAPI interface {
	// GetExecutionDir returns the directory for a specific execution.
	GetExecutionDir(taskID, executionID string) string

	// GetExecutionFilePath returns the full path to a file within an execution directory.
	GetExecutionFilePath(taskID, executionID, filename string) string

	// SavePromptToHistory stores the assembled prompt in execution history.
	// Returns the relative path to the saved prompt file.
	SavePromptToHistory(taskID, executionID, prompt string) (string, error)

	// SaveExecutionMetadata persists execution metadata to disk.
	SaveExecutionMetadata(history ExecutionHistory) error

	// SaveOutputToHistory moves the execution log to the history directory.
	// Returns the relative path to the saved output file.
	SaveOutputToHistory(taskID, executionID string) (string, error)

	// LoadExecutionHistory loads all execution history for a task.
	// Results are sorted by start time (newest first).
	LoadExecutionHistory(taskID string) ([]ExecutionHistory, error)

	// LoadAllExecutionHistory loads execution history for all tasks.
	// Uses caching to avoid expensive filesystem operations on repeated calls.
	LoadAllExecutionHistory() ([]ExecutionHistory, error)

	// LatestExecutionOutputPath returns the absolute path to the most recent execution output.
	// Prefers clean_output.txt when available, otherwise output.log.
	LatestExecutionOutputPath(taskID string) string

	// CleanupOldExecutions removes execution history older than the specified retention period.
	CleanupOldExecutions(taskID string, retentionDays int) error

	// InvalidateCache clears the execution history cache.
	InvalidateCache()
}

// Verify HistoryManager implements the interface at compile time.
var _ HistoryManagerAPI = (*HistoryManager)(nil)
