package agentactivity

import "fmt"

// Valid reports whether status belongs to the durable activity lifecycle.
func (status Status) Valid() bool {
	switch status {
	case StatusPending, StatusStarting, StatusRunning, StatusNeedsReview,
		StatusComplete, StatusAbstained, StatusBudgetExhausted,
		StatusFailed, StatusCancelled, StatusUnspecified:
		return true
	default:
		return false
	}
}

// Active identifies states that continue to occupy an execution lane.
func (status Status) Active() bool {
	switch status {
	case StatusPending, StatusStarting, StatusRunning, StatusNeedsReview:
		return true
	default:
		return false
	}
}

func (status Status) Terminal() bool {
	switch status {
	case StatusComplete, StatusAbstained, StatusBudgetExhausted,
		StatusFailed, StatusCancelled, StatusUnspecified:
		return true
	default:
		return false
	}
}

// CanTransition is the edge policy for a persisted agent activity. Pending
// records are written before an external call; starting/running may later be
// parked for operator review, resumed, or finalized by the owning system.
func (status Status) CanTransition(next Status) bool {
	if !status.Valid() || !next.Valid() {
		return false
	}
	if status == next {
		return true
	}
	switch status {
	case StatusPending:
		return next == StatusStarting || next == StatusRunning || next.Terminal()
	case StatusStarting:
		return next == StatusRunning || next == StatusNeedsReview || next.Terminal()
	case StatusRunning:
		return next == StatusNeedsReview || next.Terminal()
	case StatusNeedsReview:
		return next == StatusRunning || next.Terminal()
	default:
		return false
	}
}

// TransitionStatus is the sole mutation owner for a durable activity status.
func (record *Record) TransitionStatus(next Status) error {
	if record == nil || !record.Status.CanTransition(next) {
		if record == nil {
			return fmt.Errorf("%w: nil activity record", ErrInvalidStatusTransition)
		}
		return fmt.Errorf("%w: %q -> %q", ErrInvalidStatusTransition, record.Status, next)
	}
	record.Status = next
	return nil
}
