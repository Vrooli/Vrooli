package recycler

import "github.com/ecosystem-manager/api/pkg/settings"

// RecyclerAPI defines the interface for task recycling services.
// This abstraction enables unit testing without requiring a real recycler.
type RecyclerAPI interface {
	// Enqueue schedules a task ID for recycler processing if enabled.
	// The recycler will evaluate whether to requeue, finalize, or block the task.
	Enqueue(taskID string)

	// SetWakeFunc registers a callback to nudge the queue processor after requeues.
	SetWakeFunc(fn func())

	// Start launches the background recycling loop if not already running.
	Start()

	// Stop terminates the background recycling loop.
	Stop()

	// Wake requests an immediate processing pass in addition to the scheduled interval.
	Wake()

	// OnSettingsUpdated reacts to settings changes by reseeding and waking.
	OnSettingsUpdated(previous, next settings.Settings)

	// Stats returns basic recycler counters for observability/testing.
	Stats() Stats
}

// Compile-time assertion that Recycler implements RecyclerAPI.
var _ RecyclerAPI = (*Recycler)(nil)
