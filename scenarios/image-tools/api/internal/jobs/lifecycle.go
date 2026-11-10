package jobs

import (
	"fmt"
	"time"
)

// CanTransition is the sole job-lifecycle transition table. A cancellation is
// not completion: active jobs pass through canceling until their leases have
// been released. Admission may requeue a contended job without executing it.
func (s State) CanTransition(next State) bool {
	switch s {
	case StateQueued:
		return next == StateAdmitting || next == StateCanceled || next == StateFailed
	case StateAdmitting:
		return next == StateQueued || next == StateRunning || next == StateCanceling || next == StateFailed || next == StateCanceled
	case StateRunning:
		return next == StateSucceeded || next == StateFailed || next == StateCanceling || next == StateCanceled
	case StateCanceling:
		return next == StateCanceled || next == StateFailed
	default:
		return false
	}
}

func (j *Job) transition(next State, at time.Time, message string) error {
	if !j.State.CanTransition(next) {
		return fmt.Errorf("jobs: invalid lifecycle transition %q -> %q", j.State, next)
	}
	j.State = next
	j.Message = message
	if next == StateRunning && j.StartedAt == nil {
		j.StartedAt = &at
	}
	if next.Terminal() {
		j.FinishedAt = &at
	}
	return nil
}

func normalizePriority(priority Priority) Priority {
	switch priority {
	case PriorityInteractive, PriorityService, PriorityBatch:
		return priority
	default:
		return PriorityService
	}
}

func priorityRank(priority Priority) int {
	switch normalizePriority(priority) {
	case PriorityInteractive:
		return 2
	case PriorityService:
		return 1
	default:
		return 0
	}
}
