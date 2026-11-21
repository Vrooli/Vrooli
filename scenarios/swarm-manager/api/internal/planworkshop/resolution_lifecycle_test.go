package planworkshop

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestResolutionTransitionMatrix(t *testing.T) {
	states := []ResolutionState{
		ResolutionPending,
		ResolutionDirectApplied,
		ResolutionReconciliation,
		ResolutionCandidateReady,
		ResolutionCandidateApplied,
		ResolutionCandidateDiscarded,
		ResolutionNeedsAttention,
		ResolutionStale,
		ResolutionUnavailable,
	}
	allowed := map[ResolutionState][]ResolutionState{
		ResolutionPending:            {ResolutionDirectApplied, ResolutionReconciliation, ResolutionUnavailable},
		ResolutionReconciliation:     {ResolutionCandidateReady, ResolutionNeedsAttention, ResolutionStale},
		ResolutionCandidateReady:     {ResolutionCandidateApplied, ResolutionCandidateDiscarded, ResolutionStale},
		ResolutionDirectApplied:      {},
		ResolutionCandidateApplied:   {},
		ResolutionCandidateDiscarded: {},
		ResolutionNeedsAttention:     {},
		ResolutionStale:              {},
		ResolutionUnavailable:        {},
	}

	for _, from := range states {
		for _, to := range states {
			t.Run(fmt.Sprintf("%s_to_%s", from, to), func(t *testing.T) {
				resolution := Resolution{ResponseID: "response-1", State: from}
				wantAllowed := from == to
				for _, next := range allowed[from] {
					wantAllowed = wantAllowed || next == to
				}
				err := resolution.Transition(to)
				if wantAllowed {
					if err != nil || resolution.State != to {
						t.Fatalf("Transition(%q) = %q, %v; want %q, nil", to, resolution.State, err, to)
					}
					return
				}
				if !errors.Is(err, ErrInvalidResolutionTransition) {
					t.Fatalf("Transition(%q) error = %v; want ErrInvalidResolutionTransition", to, err)
				}
				if resolution.State != from {
					t.Fatalf("rejected edge mutated state to %q; want %q", resolution.State, from)
				}
			})
		}
	}
}

func TestResolutionTransitionRejectsUnknownStates(t *testing.T) {
	unknownCurrent := Resolution{State: "future_state"}
	err := unknownCurrent.Transition(ResolutionPending)
	if !errors.Is(err, ErrInvalidResolutionTransition) || unknownCurrent.State != "future_state" {
		t.Fatalf("unknown current state result = %q, %v", unknownCurrent.State, err)
	}

	unknownTarget := Resolution{State: ResolutionPending}
	err = unknownTarget.Transition("future_state")
	if !errors.Is(err, ErrInvalidResolutionTransition) || unknownTarget.State != ResolutionPending {
		t.Fatalf("unknown target state result = %q, %v", unknownTarget.State, err)
	}

	var nilResolution *Resolution
	err = nilResolution.Transition(ResolutionPending)
	if !errors.Is(err, ErrInvalidResolutionTransition) {
		t.Fatalf("nil receiver transition error = %v; want ErrInvalidResolutionTransition", err)
	}
}

func TestSessionValidationRejectsUnknownResolutionState(t *testing.T) {
	session := Session{
		ID:             "workshop-1",
		Subject:        Subject{Kind: SubjectBacklog, Ref: "feature-1"},
		SubjectVersion: "v1",
		Resolutions:    []Resolution{{ResponseID: "response-1", State: "future_state"}},
	}
	err := session.Validate()
	if err == nil || !strings.Contains(err.Error(), "unknown state") {
		t.Fatalf("Session.Validate() error = %v; want unknown-state error", err)
	}
}
