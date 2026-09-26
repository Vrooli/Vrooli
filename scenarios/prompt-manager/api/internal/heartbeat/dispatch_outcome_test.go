package heartbeat

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsDispatchUncertain_TypedOutcomes(t *testing.T) {
	cause := errors.New("boom")
	if IsDispatchUncertain(NewDispatchUncertainError(cause)) != true {
		t.Fatal("typed uncertain error must classify uncertain")
	}
	if IsDispatchUncertain(NewDispatchRejectedError(cause)) != false {
		t.Fatal("typed rejected error must classify deterministic")
	}
	if IsDispatchRejected(NewDispatchRejectedError(cause)) != true {
		t.Fatal("typed rejected error must be recognized")
	}
	// Wrapping must survive the way Execute wraps the client error.
	wrapped := fmt.Errorf("creating run: %w", NewDispatchUncertainError(cause))
	if !IsDispatchUncertain(wrapped) {
		t.Fatal("wrapped uncertain error must classify uncertain")
	}
}

func TestIsDispatchUncertain_TransportHeuristics(t *testing.T) {
	cases := map[string]bool{
		"connection refused":                         true,
		"connection reset by peer":                   true,
		"no such host":                               true,
		"i/o timeout":                                true,
		"context deadline exceeded":                  true,
		"agent-manager unreachable after 4 attempts": true,
		"validation error: title required":           false,
		"runner unavailable":                         false,
	}
	for message, want := range cases {
		if got := IsDispatchUncertain(errors.New(message)); got != want {
			t.Errorf("IsDispatchUncertain(%q) = %v, want %v", message, got, want)
		}
	}
	if IsDispatchUncertain(nil) {
		t.Fatal("nil error must not be uncertain")
	}
}
