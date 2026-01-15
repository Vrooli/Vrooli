package steering

import (
	"time"

	"github.com/ecosystem-manager/api/pkg/autosteer"
)

// QueueStateRepository defines the interface for persisting steering queue state.
type QueueStateRepository interface {
	// Get retrieves the queue state for a task.
	// Returns nil, nil if no state exists.
	Get(taskID string) (*QueueState, error)

	// Save persists the queue state (upsert).
	Save(state *QueueState) error

	// Delete removes the queue state for a task.
	Delete(taskID string) error

	// ResetPosition resets the queue position to 0 without deleting the state.
	// Use this when reactivating a task to allow re-running the full queue.
	ResetPosition(taskID string) error
}

// NewQueueState creates a new QueueState for a task.
func NewQueueState(taskID string, queue []autosteer.SteerMode) *QueueState {
	now := time.Now().Format(time.RFC3339)
	return &QueueState{
		TaskID:       taskID,
		Queue:        queue,
		CurrentIndex: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Advance moves to the next item in the queue.
// Returns true if there are more items, false if exhausted.
func (q *QueueState) Advance() bool {
	q.CurrentIndex++
	q.UpdatedAt = time.Now().Format(time.RFC3339)
	return !q.IsExhausted()
}

// Reset sets the queue position back to the beginning.
func (q *QueueState) Reset() {
	q.CurrentIndex = 0
	q.UpdatedAt = time.Now().Format(time.RFC3339)
}

// Position returns a human-readable position string (e.g., "2/5").
func (q *QueueState) Position() string {
	if q.IsExhausted() {
		return "done"
	}
	return formatPosition(q.CurrentIndex+1, len(q.Queue))
}

// formatPosition formats position as "current/total".
func formatPosition(current, total int) string {
	return formatInt(current) + "/" + formatInt(total)
}

// formatInt converts int to string without importing strconv.
func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + formatInt(-n)
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
