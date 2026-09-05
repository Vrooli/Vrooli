package budget

import "fmt"

type FreezeEvent string

const (
	FreezeEventEngage  FreezeEvent = "engage"
	FreezeEventRelease FreezeEvent = "release"
)

type FreezeState struct{ Frozen bool }

// CheckFreezeInvariants makes the model's single invariant executable. The
// boolean representation is intentionally total: there is no third/unknown
// state that could silently leave different callers interpreting the switch
// differently.
func CheckFreezeInvariants(state FreezeState) error {
	switch state.Frozen {
	case true, false:
		return nil
	default:
		return fmt.Errorf("freeze state is not boolean")
	}
}

// TransitionFreeze is the level-2 imperative model for the kill switch. Its
// invariant is deliberately small: every event yields one of two boolean
// states, and repeated operator actions are idempotent.
func TransitionFreeze(_ FreezeState, event FreezeEvent) (FreezeState, error) {
	var next FreezeState
	switch event {
	case FreezeEventEngage:
		next = FreezeState{Frozen: true}
	case FreezeEventRelease:
		next = FreezeState{Frozen: false}
	default:
		return FreezeState{}, fmt.Errorf("unknown freeze event %q", event)
	}
	return next, CheckFreezeInvariants(next)
}

func freezeEvent(frozen bool) FreezeEvent {
	if frozen {
		return FreezeEventEngage
	}
	return FreezeEventRelease
}
