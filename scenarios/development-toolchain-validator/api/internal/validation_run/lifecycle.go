package validation_run

import (
	"errors"
	"fmt"
)

var ErrInvalidTransition = errors.New("invalid validation run transition")
var ErrStateConflict = errors.New("validation run state changed before update")

// TransitionStatus owns the lifecycle for one durable validation run.
func TransitionStatus(from, to Status) error {
	if !from.Valid() || !to.Valid() {
		return fmt.Errorf("%w: unknown status %d -> %d", ErrInvalidTransition, from, to)
	}
	if from == to {
		return nil
	}
	switch from {
	case StatusQueued:
		if to == StatusRunning {
			return nil
		}
	case StatusRunning:
		if to == StatusEvaluating || to == StatusTerminal {
			return nil
		}
	case StatusEvaluating:
		if to == StatusTerminal {
			return nil
		}
	}
	return fmt.Errorf("%w: %d -> %d", ErrInvalidTransition, from, to)
}

func (s Status) Valid() bool {
	return s == StatusQueued || s == StatusRunning || s == StatusEvaluating || s == StatusTerminal
}

func (r *Run) TransitionStatus(to Status) error {
	if r == nil {
		return fmt.Errorf("%w: run is nil", ErrInvalidTransition)
	}
	if err := TransitionStatus(r.Status, to); err != nil {
		return err
	}
	r.Status = to
	return nil
}
