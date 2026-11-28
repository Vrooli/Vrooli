package evidence

import "fmt"

// TransitionPublicationState owns the persisted publication state vocabulary
// and allowed edges. Equal valid states are idempotent for request retries.
func TransitionPublicationState(from, to string) error {
	if !validPublicationState(from) {
		return fmt.Errorf("%w: unknown source state %q", ErrInvalidPublicationTransition, from)
	}
	if !validPublicationState(to) {
		return fmt.Errorf("%w: unknown target state %q", ErrInvalidPublicationTransition, to)
	}
	if from == to {
		return nil
	}
	switch from {
	case StateRequested:
		if to == StateReviewPrepared || to == StateApproved || to == StateRefused {
			return nil
		}
	case StateReviewPrepared:
		if to == StateApproved || to == StateRefused {
			return nil
		}
	case StateApproved:
		if to == StateActivating || to == StateRefused {
			return nil
		}
	case StateRefused:
		// A refused publication can be retried after its governance or
		// evidence prerequisite changes, under the same idempotency key.
		if to == StateApproved {
			return nil
		}
	case StateActivating:
		if to == StatePublished || to == StateFailed {
			return nil
		}
	}
	return fmt.Errorf("%w: %s -> %s", ErrInvalidPublicationTransition, from, to)
}

func validPublicationState(state string) bool {
	switch state {
	case StateRequested, StateReviewPrepared, StateApproved, StateActivating,
		StatePublished, StateRefused, StateFailed:
		return true
	default:
		return false
	}
}

// ValidateState rejects values that are not part of the durable lifecycle.
func (p Publication) ValidateState() error {
	if !validPublicationState(p.State) {
		return fmt.Errorf("%w: unknown state %q", ErrInvalidPublicationTransition, p.State)
	}
	return nil
}

// Transition applies a legal edge and leaves state unchanged on rejection.
func (p *Publication) Transition(to string) error {
	if p == nil {
		return fmt.Errorf("%w: publication is nil", ErrInvalidPublicationTransition)
	}
	if err := TransitionPublicationState(p.State, to); err != nil {
		return err
	}
	p.State = to
	return nil
}
