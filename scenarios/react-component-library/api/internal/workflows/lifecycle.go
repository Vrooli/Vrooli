package workflows

import (
	"errors"
	"fmt"
)

var ErrInvalidStatusTransition = errors.New("workflows: invalid status transition")

var workflowStatusTransitions = map[Status]map[Status]struct{}{
	StatusQueued: {
		StatusRunning: {}, StatusParked: {}, StatusSucceeded: {}, StatusFailed: {}, StatusStopped: {}, StatusUnavailable: {},
	},
	StatusRunning: {
		StatusParked: {}, StatusSucceeded: {}, StatusFailed: {}, StatusStopped: {}, StatusUnavailable: {},
	},
	StatusParked: {
		StatusRunning: {}, StatusSucceeded: {}, StatusFailed: {}, StatusStopped: {}, StatusUnavailable: {},
	},
	// Unavailable is a local observation gap. A successful reconciliation may
	// recover any state that Agent Manager persisted while RCL was disconnected.
	StatusUnavailable: {
		StatusQueued: {}, StatusRunning: {}, StatusParked: {}, StatusSucceeded: {}, StatusFailed: {}, StatusStopped: {},
	},
}

// TransitionStatus is the sole owner of changes to a durable workflow's status.
// Repeating the current status is idempotent because external execution status
// is projected into this record more than once during normal reconciliation.
func (w *Workflow) TransitionStatus(next Status) error {
	if w == nil {
		return fmt.Errorf("%w: nil workflow", ErrInvalidStatusTransition)
	}
	if w.Status == next && validStatus(next) {
		return nil
	}
	if !validStatus(next) {
		return fmt.Errorf("%w: unknown target %q", ErrInvalidStatusTransition, next)
	}
	if _, ok := workflowStatusTransitions[w.Status][next]; !ok {
		return fmt.Errorf("%w: %q -> %q", ErrInvalidStatusTransition, w.Status, next)
	}
	w.Status = next
	return nil
}

func validStatus(status Status) bool {
	switch status {
	case StatusQueued, StatusRunning, StatusParked, StatusSucceeded, StatusFailed, StatusStopped, StatusUnavailable:
		return true
	default:
		return false
	}
}
