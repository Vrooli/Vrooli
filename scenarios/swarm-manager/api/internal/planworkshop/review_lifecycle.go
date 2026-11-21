package planworkshop

import (
	"errors"
	"fmt"
)

var ErrInvalidReviewRunTransition = errors.New("invalid Plan Workshop review transition")

var reviewRunTransitions = map[ReviewState]map[ReviewState]struct{}{
	"":            {ReviewPending: {}},
	ReviewPending: {ReviewRunning: {}, ReviewFailed: {}, ReviewStale: {}},
	ReviewRunning: {ReviewApplied: {}, ReviewStale: {}},
	ReviewApplied: {ReviewPending: {}},
	ReviewStale:   {ReviewPending: {}},
	ReviewFailed:  {ReviewPending: {}},
}

func (state ReviewState) Valid() bool {
	_, ok := reviewRunTransitions[state]
	return ok && state != ""
}

// Transition owns every ReviewRun.State write, including a fresh retry after a
// prior review reaches any terminal result.
func (run *ReviewRun) Transition(next ReviewState) error {
	if run == nil {
		return fmt.Errorf("%w: nil review run", ErrInvalidReviewRunTransition)
	}
	if !next.Valid() {
		return fmt.Errorf("%w: unknown target state %q", ErrInvalidReviewRunTransition, next)
	}
	if run.State == next {
		return nil
	}
	if _, ok := reviewRunTransitions[run.State][next]; !ok {
		return fmt.Errorf("%w: %q -> %q", ErrInvalidReviewRunTransition, run.State, next)
	}
	run.State = next
	return nil
}
