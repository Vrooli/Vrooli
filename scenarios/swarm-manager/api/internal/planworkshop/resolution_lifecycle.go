package planworkshop

import (
	"errors"
	"fmt"
)

var ErrInvalidResolutionTransition = errors.New("invalid Plan Workshop resolution transition")

// resolutionTransitions is the sole owner of durable Resolution.State edges.
// Terminal decisions stay terminal; a new operator response creates a new
// Resolution instead of reviving an old one.
var resolutionTransitions = map[ResolutionState]map[ResolutionState]struct{}{
	ResolutionPending: {
		ResolutionDirectApplied:  {},
		ResolutionReconciliation: {},
		ResolutionUnavailable:    {},
	},
	ResolutionReconciliation: {
		ResolutionCandidateReady: {},
		ResolutionNeedsAttention: {},
		ResolutionStale:          {},
	},
	ResolutionCandidateReady: {
		ResolutionCandidateApplied:   {},
		ResolutionCandidateDiscarded: {},
		ResolutionStale:              {},
	},
}

// Valid reports whether the value belongs to the persisted Resolution state
// vocabulary. It is used when reading or writing sessions so unknown states
// fail closed instead of silently bypassing a lifecycle guard.
func (state ResolutionState) Valid() bool {
	switch state {
	case ResolutionPending, ResolutionDirectApplied, ResolutionReconciliation,
		ResolutionCandidateReady, ResolutionCandidateApplied, ResolutionCandidateDiscarded,
		ResolutionNeedsAttention, ResolutionStale, ResolutionUnavailable:
		return true
	default:
		return false
	}
}

// Transition is the only way to change an existing resolution's durable
// state. Reapplying the current value is idempotent; all other changes must
// follow the explicit graph above.
func (r *Resolution) Transition(next ResolutionState) error {
	if r == nil || !r.State.Valid() || !next.Valid() {
		return fmt.Errorf("%w: unknown state %q -> %q", ErrInvalidResolutionTransition, stateOf(r), next)
	}
	if r.State == next {
		return nil
	}
	if _, ok := resolutionTransitions[r.State][next]; !ok {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidResolutionTransition, r.State, next)
	}
	r.State = next
	return nil
}

func stateOf(r *Resolution) ResolutionState {
	if r == nil {
		return "<nil>"
	}
	return r.State
}
