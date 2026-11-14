package domain

import "fmt"

// Valid reports whether status belongs to the persisted Task vocabulary.
func (status TaskStatus) Valid() bool {
	switch status {
	case TaskStatusQueued, TaskStatusRunning, TaskStatusNeedsReview, TaskStatusApproved,
		TaskStatusRejected, TaskStatusFailed, TaskStatusCancelled, TaskStatusCompleted:
		return true
	default:
		return false
	}
}

// ValidateTaskStatusTransition is the shared policy check used by Task state
// writers. Persistence adapters still compare the observed status in SQL so a
// valid decision cannot overwrite a concurrent transition.
func ValidateTaskStatusTransition(id string, from, to TaskStatus, action string) error {
	if !from.Valid() || !to.Valid() {
		return NewStateErrorWithID("Task", id, string(from), action, fmt.Sprintf("unknown status transition target %q", to))
	}
	if from == to {
		return nil
	}
	if allowed, reason := from.CanTransitionTo(to); !allowed {
		return NewStateErrorWithID("Task", id, string(from), action, reason)
	}
	return nil
}

// TransitionStatus applies the canonical Task transition policy to an
// in-memory owner projection. Durable callers must perform the same check
// against the freshly observed row and include that value in their CAS.
func (task *Task) TransitionStatus(next TaskStatus) error {
	if task == nil {
		return fmt.Errorf("cannot transition a nil Task")
	}
	if err := ValidateTaskStatusTransition(task.ID.String(), task.Status, next, "transition"); err != nil {
		return err
	}
	task.Status = next
	return nil
}
