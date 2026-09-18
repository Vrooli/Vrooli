// Package entitlements is the server-side commercialization seam. It gates
// optional compute only; owned data, rules, nutrition safety, planning, and
// export are never gated here.
package entitlements

import (
	"context"
	"errors"
	"fmt"
)

var ErrBudgetExceeded = errors.New("optional research budget exceeded")

type BudgetError struct{ Requested, Remaining int64 }

func (e BudgetError) Error() string {
	return fmt.Sprintf("optional compute budget exceeded: requested %d units, %d remaining", e.Requested, e.Remaining)
}
func (BudgetError) Unwrap() error { return ErrBudgetExceeded }

type State struct {
	WorkspaceID     string
	Version         int64
	OptionalCompute bool
	MonthlyLimit    int64
	Used            int64
	AppliedEvents   map[string]int64
}
type Event struct {
	ID              string
	Version         int64
	OptionalCompute bool
	MonthlyLimit    int64
}

func New(workspaceID string) State {
	return State{WorkspaceID: workspaceID, OptionalCompute: true, MonthlyLimit: 0, AppliedEvents: map[string]int64{}}
}
func (s State) CoreAllowed() bool { return true }
func (s State) CanUseOptional(units int64) error {
	if !s.OptionalCompute {
		remaining := int64(0)
		if s.MonthlyLimit > 0 && s.MonthlyLimit > s.Used {
			remaining = s.MonthlyLimit - s.Used
		}
		return BudgetError{Requested: units, Remaining: remaining}
	}
	if units < 0 || (s.MonthlyLimit > 0 && s.Used+units > s.MonthlyLimit) {
		return ErrBudgetExceeded
	}
	return nil
}

// Repository is the server-side entitlement boundary. Callers never accept
// entitlement state supplied by a client.
type Repository interface {
	Get(context.Context, string) (State, error)
	ReserveOptional(context.Context, string, string, int64) (State, error)
	ApplyEvent(context.Context, string, Event) (State, error)
}

func (s State) ReserveOptional(units int64) (State, error) {
	if err := s.CanUseOptional(units); err != nil {
		return s, err
	}
	s.Used += units
	return s, nil
}
func (s State) ApplyEvent(event Event) (State, error) {
	if event.ID == "" || event.Version < 1 || event.Version < s.Version {
		return s, nil
	}
	if previous, ok := s.AppliedEvents[event.ID]; ok && previous == event.Version {
		return s, nil
	}
	if event.Version == s.Version && s.Version != 0 {
		return s, nil
	}
	s.Version = event.Version
	s.OptionalCompute = event.OptionalCompute
	s.MonthlyLimit = event.MonthlyLimit
	s.AppliedEvents[event.ID] = event.Version
	return s, nil
}
