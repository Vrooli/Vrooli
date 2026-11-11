package execution

import "fmt"

// BaselineSetState owns the durable lifecycle of an execution's behavioral-
// before receipt. A recapture creates a new ticket; it does not revive an old
// terminal receipt.
func (s BaselineSetState) CanTransitionTo(next BaselineSetStatus) bool {
	if !validBaselineSetStatus(s.Status) || !validBaselineSetStatus(next) {
		return false
	}
	if s.Status == next {
		return true
	}
	switch s.Status {
	case BaselineSetStatusRequired:
		return next == BaselineSetStatusScopeRepairRequired || next == BaselineSetStatusComplete || next == BaselineSetStatusPartial || next == BaselineSetStatusDegraded
	case BaselineSetStatusScopeRepairRequired:
		return next == BaselineSetStatusRequired || next == BaselineSetStatusDegraded
	case BaselineSetStatusPartial:
		return next == BaselineSetStatusDegraded
	default:
		return false
	}
}

func (s *BaselineSetState) TransitionTo(next BaselineSetStatus) error {
	if s == nil {
		return fmt.Errorf("baseline transition requires a state")
	}
	if !s.CanTransitionTo(next) {
		return fmt.Errorf("invalid baseline transition %q -> %q", s.Status, next)
	}
	s.Status = next
	return nil
}

func validBaselineSetStatus(status BaselineSetStatus) bool {
	switch status {
	case BaselineSetStatusRequired, BaselineSetStatusScopeRepairRequired,
		BaselineSetStatusComplete, BaselineSetStatusPartial, BaselineSetStatusDegraded:
		return true
	default:
		return false
	}
}

func (s BaselineSetState) Validate() error {
	if s.Status == "" {
		return nil // pre-lifecycle legacy rows are read-compatible
	}
	if !validBaselineSetStatus(s.Status) {
		return fmt.Errorf("unknown baseline status %q", s.Status)
	}
	return nil
}
