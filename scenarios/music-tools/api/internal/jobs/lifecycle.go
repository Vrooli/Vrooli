package jobs

import "fmt"

// ErrInvalidStateTransition identifies a job-state edge outside the durable
// job lifecycle. Initial state is assigned when a Job is created; every later
// state change goes through Transition.
var ErrInvalidStateTransition = fmt.Errorf("jobs: invalid state transition")

var transitions = map[State]map[State]struct{}{
	StateQueued:  edges(StateRunning, StateFailed, StateCanceled),
	StateRunning: edges(StateSucceeded, StateFailed, StateCanceled),
}

func edges(states ...State) map[State]struct{} {
	allowed := make(map[State]struct{}, len(states))
	for _, state := range states {
		allowed[state] = struct{}{}
	}
	return allowed
}

// Valid reports whether state is part of the persisted job lifecycle.
func (state State) Valid() bool {
	switch state {
	case StateQueued, StateRunning, StateSucceeded, StateFailed, StateCanceled:
		return true
	default:
		return false
	}
}

// mustTransition is for manager-owned edges whose source is protected by the
// job entry mutex. An invalid edge here is an internal programming error: do
// not persist or announce a state the lifecycle table rejected.
func mustTransition(job *Job, next State) {
	if err := job.Transition(next); err != nil {
		panic(err)
	}
}

// Transition changes a previously created job's state. Repeating the current
// state is idempotent; terminal states cannot be revived or replaced.
func (job *Job) Transition(next State) error {
	if job == nil {
		return fmt.Errorf("%w: nil job", ErrInvalidStateTransition)
	}
	if !job.State.Valid() || !next.Valid() {
		return fmt.Errorf("%w: %q -> %q contains unknown state", ErrInvalidStateTransition, job.State, next)
	}
	if job.State == next {
		return nil
	}
	if _, ok := transitions[job.State][next]; !ok {
		return fmt.Errorf("%w: %q -> %q", ErrInvalidStateTransition, job.State, next)
	}
	job.State = next
	return nil
}
