package budget

import "fmt"

type FreezeEvent string

const (
	FreezeEventEngage  FreezeEvent = "engage"
	FreezeEventRelease FreezeEvent = "release"
)

type FreezeState struct{ Frozen bool }

// TransitionFreeze is the level-2 imperative model for the kill switch. Its
// invariant is deliberately small: every event yields one of two boolean
// states, and repeated operator actions are idempotent.
func TransitionFreeze(_ FreezeState, event FreezeEvent) (FreezeState, error) {
	switch event {
	case FreezeEventEngage:
		return FreezeState{Frozen: true}, nil
	case FreezeEventRelease:
		return FreezeState{Frozen: false}, nil
	default:
		return FreezeState{}, fmt.Errorf("unknown freeze event %q", event)
	}
}

func freezeEvent(frozen bool) FreezeEvent {
	if frozen {
		return FreezeEventEngage
	}
	return FreezeEventRelease
}
