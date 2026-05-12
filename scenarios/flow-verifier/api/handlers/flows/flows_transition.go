package flows

import (
	"flow-verifier/internal/flows/model"
)

type transitionFields struct {
	from         []string
	event        []string
	to           string
	wantError    bool
	wantErrorSet bool
}

// concreteTransition projects model.Transition into the shape the proto
// converter consumes. model.Transition's From/Event are scalar strings
// — the matrix flattens contract.Transition.From StringList into one
// row per (state,event) pair — so the proto's repeated fields each
// carry exactly one element. WantError is always set in model.Transition
// (defaults have been folded in), so wantErrorSet is always true.
func concreteTransition(t any) transitionFields {
	tr, ok := t.(model.Transition)
	if !ok {
		return transitionFields{}
	}
	return transitionFields{
		from:         []string{tr.From},
		event:        []string{tr.Event},
		to:           tr.To,
		wantError:    tr.WantError,
		wantErrorSet: true,
	}
}
