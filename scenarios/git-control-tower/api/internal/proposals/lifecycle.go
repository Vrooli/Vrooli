package proposals

import "fmt"

// transitionState is the only owner of Proposal.State changes. Revisions are
// content-review revisions, so activity and terminal transitions deliberately
// preserve the revision already guarded by the caller/store.
func transitionState(proposal *Proposal, next State) error {
	if proposal == nil {
		return fmt.Errorf("%w: proposal is required", ErrInvalid)
	}
	if err := validateStateTransition(proposal.State, next); err != nil {
		return err
	}
	proposal.State = next
	return nil
}

func validateStateTransition(current, next State) error {
	if !validState(current) || !validState(next) {
		return fmt.Errorf("%w: unknown proposal state transition %q -> %q", ErrInvalid, current, next)
	}
	if current == next {
		return nil
	}
	if current != StateOpen {
		return fmt.Errorf("%w: terminal proposal state %q cannot transition to %q", ErrInvalid, current, next)
	}
	return nil
}

func validState(state State) bool {
	switch state {
	case StateOpen, StateCommitted, StateWithdrawn, StateSuperseded:
		return true
	default:
		return false
	}
}
