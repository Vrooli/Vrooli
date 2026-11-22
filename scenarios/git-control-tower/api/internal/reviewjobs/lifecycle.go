package reviewjobs

import "fmt"

const (
	Dispatching   State = "dispatching"
	AwaitingInput State = "awaiting_input"
	Stale         State = "stale"
	Cancelling    State = "cancelling"
)

var lifecycleEdges = map[State]map[State]struct{}{
	Queued: {
		Dispatching: {}, Running: {}, Failed: {}, Cancelled: {}, Cancelling: {}, Interrupted: {},
	},
	Dispatching: {
		Queued: {}, Running: {}, AwaitingInput: {}, AwaitingProvider: {}, Succeeded: {}, Failed: {}, Cancelled: {}, Cancelling: {}, Stale: {}, Interrupted: {},
	},
	Running: {
		Queued: {}, AwaitingInput: {}, AwaitingProvider: {}, Partial: {}, Succeeded: {}, Failed: {}, Cancelled: {}, Cancelling: {}, Stale: {}, Interrupted: {},
	},
	AwaitingInput: {
		Queued: {}, Running: {}, Succeeded: {}, Failed: {}, Cancelled: {}, Cancelling: {}, Stale: {}, Interrupted: {},
	},
	AwaitingProvider: {Cancelled: {}},
	Partial: {
		Queued: {}, Running: {}, Succeeded: {}, Failed: {}, Cancelled: {}, Cancelling: {}, Interrupted: {},
	},
	Cancelling: {
		Failed: {}, Cancelled: {}, Stale: {},
	},
}

// Valid reports whether state is part of the durable review-job lifecycle.
func (state State) Valid() bool {
	switch state {
	case Queued, Dispatching, Running, AwaitingInput, AwaitingProvider, Partial, Succeeded, Failed, Cancelled, Interrupted, Cancelling, Stale:
		return true
	default:
		return false
	}
}

// Terminal reports whether state has no permitted outgoing lifecycle edge.
func (state State) Terminal() bool {
	return state == Succeeded || state == Failed || state == Cancelled || state == Interrupted || state == Stale
}

// Transition changes one job state through the fleet lifecycle owner.
// Repeating the current state is idempotent; invalid transitions do not mutate.
func (job *Job) Transition(next State) error {
	if job == nil {
		return fmt.Errorf("review job: cannot transition a nil job")
	}
	if !job.State.Valid() {
		return fmt.Errorf("review job: unknown source state %q", job.State)
	}
	if !next.Valid() {
		return fmt.Errorf("review job: unknown target state %q", next)
	}
	if job.State == next {
		return nil
	}
	if _, allowed := lifecycleEdges[job.State][next]; !allowed {
		return fmt.Errorf("review job: invalid state transition %q -> %q", job.State, next)
	}
	job.State = next
	return nil
}
