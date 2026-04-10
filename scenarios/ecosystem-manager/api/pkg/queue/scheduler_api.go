package queue

// SchedulerAPI defines the interface for task scheduling decisions.
// This interface enables testing and provides clear boundaries for scheduling logic.
// The Processor implements this interface directly, providing testability
// without requiring full extraction of the tightly-coupled scheduler logic.
type SchedulerAPI interface {
	// ProcessQueue processes pending tasks and starts available tasks.
	ProcessQueue()

	// ForceStartTask forces a task to start regardless of slot availability.
	// If allowOverflow is true, ignores slot limits completely.
	ForceStartTask(taskID string, allowOverflow bool) error

	// StartTaskIfSlotAvailable starts a task only if slots are available.
	// Returns error if no slots available or task not found.
	StartTaskIfSlotAvailable(taskID string) error

	// Wake signals the scheduler to process the queue.
	Wake()

	// GetSlotSnapshot returns current slot availability.
	// Uses SlotSnapshot type defined in concurrency_manager.go.
	GetSlotSnapshot() SlotSnapshot

	// IsRunning returns whether the scheduler is currently processing.
	IsRunning() bool

	// IsPaused returns whether the scheduler is in maintenance mode.
	IsPaused() bool
}

// Verify Processor implements SchedulerAPI at compile time.
var _ SchedulerAPI = (*Processor)(nil)
