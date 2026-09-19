package flow

import "fmt"

func Transition(state State, event Event) (State, error) {
	if err := CheckInvariants(state); err != nil {
		return state, err
	}
	switch state.Status {
	case StatusQueued:
		switch event {
		case EventDeliveryFailed:
			return state, nil
		case EventDeliveryAccepted:
			return State{Status: StatusAccepted}, nil
		}
	case StatusAccepted:
		return state, fmt.Errorf("accepted ledger emission is terminal")
	}
	return state, fmt.Errorf("invalid ledger emission transition %s + %s", state.Status, event)
}

func CheckInvariants(state State) error {
	switch state.Status {
	case StatusQueued, StatusAccepted:
		return nil
	default:
		return fmt.Errorf("unknown ledger emission status %q", state.Status)
	}
}
