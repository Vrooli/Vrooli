package personas

import "fmt"

// TransitionStatus owns the durable persona status vocabulary and legal edges.
func TransitionStatus(from, to Status) error {
	if !from.Valid() {
		return fmt.Errorf("%w: unknown source status %q", ErrInvalidTransition, from)
	}
	if !to.Valid() {
		return fmt.Errorf("%w: unknown target status %q", ErrInvalidTransition, to)
	}
	if from == StatusActive && to == StatusArchived {
		return nil
	}
	return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
}

func (s Status) Valid() bool { return s == StatusActive || s == StatusArchived }

func (p *Persona) TransitionStatus(to Status) error {
	if p == nil {
		return fmt.Errorf("%w: persona is nil", ErrInvalidTransition)
	}
	if err := TransitionStatus(p.Status, to); err != nil {
		return err
	}
	p.Status = to
	return nil
}
