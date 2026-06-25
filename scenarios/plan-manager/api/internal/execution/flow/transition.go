package flow

import (
	"fmt"

	"plan-manager/internal/execution/flow/generated"
)

type (
	PhaseLifecycleStatus = generated.PhaseLifecycleStatus
	PhaseLifecycleEvent  = generated.PhaseLifecycleEvent
)

const (
	PhaseLifecycleTodo    = generated.PhaseLifecycleTodo
	PhaseLifecycleActive  = generated.PhaseLifecycleActive
	PhaseLifecycleBlocked = generated.PhaseLifecycleBlocked
	PhaseLifecycleDone    = generated.PhaseLifecycleDone
)

const (
	PhaseLifecycleBegin    = generated.PhaseLifecycleBegin
	PhaseLifecycleBlock    = generated.PhaseLifecycleBlock
	PhaseLifecycleUnblock  = generated.PhaseLifecycleUnblock
	PhaseLifecycleComplete = generated.PhaseLifecycleComplete
)

type PhaseLifecycleState struct {
	Status PhaseLifecycleStatus
}

func InitialPhaseLifecycleState() PhaseLifecycleState {
	return PhaseLifecycleState{Status: PhaseLifecycleTodo}
}

func TransitionPhaseLifecycle(state PhaseLifecycleState, event PhaseLifecycleEvent) (PhaseLifecycleState, error) {
	if err := CheckPhaseLifecycleInvariants(state); err != nil {
		return state, err
	}
	next, err := generated.TransitionPhaseLifecycleStatus(state.Status, event)
	return PhaseLifecycleState{Status: next}, err
}

func CheckPhaseLifecycleInvariants(state PhaseLifecycleState) error {
	switch state.Status {
	case PhaseLifecycleTodo,
		PhaseLifecycleActive,
		PhaseLifecycleBlocked,
		PhaseLifecycleDone:
		return nil
	default:
		return fmt.Errorf("unknown phase lifecycle status %q", state.Status)
	}
}
