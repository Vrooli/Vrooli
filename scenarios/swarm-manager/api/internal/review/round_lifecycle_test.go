package review

import (
	"errors"
	"testing"
)

func TestRoundTransitionStatusEnforcesLifecycleEdges(t *testing.T) {
	for from, allowed := range roundStatusTransitions {
		for to := range allowed {
			round := Round{Status: from}
			if err := round.TransitionStatus(to); err != nil {
				t.Errorf("TransitionStatus(%s -> %s) error = %v", from, to, err)
			}
			if round.Status != to {
				t.Errorf("TransitionStatus(%s -> %s) left status %s", from, to, round.Status)
			}
		}
	}

	for _, edge := range [][2]RoundStatus{
		{RoundStatusGathering, RoundStatusPending},
		{RoundStatusFailed, RoundStatusGathering},
		{RoundStatusComplete, RoundStatusGathering},
	} {
		round := Round{Status: edge[0]}
		var transitionErr *InvalidRoundTransitionError
		if err := round.TransitionStatus(edge[1]); !errors.As(err, &transitionErr) {
			t.Errorf("TransitionStatus(%s -> %s) error = %v, want InvalidRoundTransitionError", edge[0], edge[1], err)
		}
		if round.Status != edge[0] {
			t.Errorf("rejected TransitionStatus(%s -> %s) mutated status to %s", edge[0], edge[1], round.Status)
		}
	}
}
