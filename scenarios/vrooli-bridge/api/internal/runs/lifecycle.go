package runs

import "fmt"

// runStatusTransitions is the sole owner of legal Run lifecycle edges. Delivery
// and execution share one persisted status for compatibility with the queue
// journal, so the table covers both protocols and their recovery paths.
var runStatusTransitions = map[RunStatus][]RunStatus{
	StatusQueued: {
		StatusPushed,
		StatusAcked,
		StatusRunning,
		StatusPassed,
		StatusFailed,
		StatusFailedDelivery,
		StatusAborted,
	},
	StatusPushed: {
		StatusQueued,
		StatusAcked,
		StatusRunning,
		StatusPassed,
		StatusFailed,
		StatusFailedDelivery,
		StatusCancelRequested,
		StatusUncertain,
		StatusAborted,
	},
	StatusAcked: {
		StatusRunning,
		StatusPassed,
		StatusFailed,
		StatusFailedDelivery,
		StatusCancelRequested,
		StatusUncertain,
		StatusAborted,
	},
	StatusRunning: {
		StatusPassed,
		StatusFailed,
		StatusCancelRequested,
		StatusUncertain,
		StatusAborted,
	},
	StatusCancelRequested: {
		StatusUncertain,
		StatusAborted,
	},
	StatusUncertain: {
		StatusCancelRequested,
		StatusAborted,
	},
	StatusPassed:         {},
	StatusFailed:         {},
	StatusAborted:        {},
	StatusFailedDelivery: {},
}

// Valid reports whether the value belongs to the persisted Run status set.
func (status RunStatus) Valid() bool {
	_, ok := runStatusTransitions[status]
	return ok
}

// CanTransitionTo reports whether the persisted Run status may move to next.
// Repeated writes of the same state are idempotent so metadata can be updated
// without inventing another edge.
func (status RunStatus) CanTransitionTo(next RunStatus) bool {
	if status == next {
		return status.Valid()
	}
	for _, allowed := range runStatusTransitions[status] {
		if allowed == next {
			return true
		}
	}
	return false
}

// ValidateStatusTransition is shared by service callers and repository
// boundaries. Persistence still uses the exact observed status in its SQL CAS.
func ValidateStatusTransition(from, to RunStatus) error {
	if !from.Valid() {
		return ErrInvalidRun{Field: "status", Reason: fmt.Sprintf("unknown current status %d", from)}
	}
	if !to.Valid() {
		return ErrInvalidRun{Field: "status", Reason: fmt.Sprintf("unknown target status %d", to)}
	}
	if !from.CanTransitionTo(to) {
		return ErrInvalidRun{Field: "status", Reason: fmt.Sprintf("transition %s -> %s is not allowed", from, to)}
	}
	return nil
}

// TransitionStatus applies the canonical transition policy to an in-memory
// Run. Repositories repeat the validation against the freshly loaded row and
// compare-and-swap its prior status before saving.
func (run *Run) TransitionStatus(next RunStatus) error {
	if run == nil {
		return ErrInvalidRun{Field: "status", Reason: "cannot transition a nil run"}
	}
	if err := ValidateStatusTransition(run.Status, next); err != nil {
		return err
	}
	run.Status = next
	return nil
}
