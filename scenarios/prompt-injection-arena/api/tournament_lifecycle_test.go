package main

import (
	"errors"
	"testing"
)

func TestTournamentTransitionGraph(t *testing.T) {
	allowed := map[TournamentStatus][]TournamentStatus{
		TournamentScheduled: {TournamentRunning, TournamentCancelled},
		TournamentRunning:   {TournamentCompleted},
	}
	states := []TournamentStatus{TournamentScheduled, TournamentRunning, TournamentCompleted, TournamentCancelled}
	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				tournament := &Tournament{Status: from}
				err := tournament.TransitionStatus(to)
				isAllowed := from == to
				for _, candidate := range allowed[from] {
					isAllowed = isAllowed || candidate == to
				}
				if isAllowed {
					if err != nil || tournament.Status != to {
						t.Fatalf("transition %s -> %s failed: status=%s err=%v", from, to, tournament.Status, err)
					}
					return
				}
				if !errors.Is(err, errInvalidTournamentTransition) || tournament.Status != from {
					t.Fatalf("illegal transition %s -> %s mutated state or returned wrong error: status=%s err=%v", from, to, tournament.Status, err)
				}
			})
		}
	}
}

func TestTournamentTransitionRejectsUnknownAndNil(t *testing.T) {
	for _, tournament := range []*Tournament{
		{Status: "future_state"},
		{Status: TournamentScheduled},
		nil,
	} {
		next := TournamentRunning
		if tournament != nil && tournament.Status == TournamentScheduled {
			next = "future_state"
		}
		if err := tournament.TransitionStatus(next); !errors.Is(err, errInvalidTournamentTransition) {
			t.Fatalf("transition from %+v to %q returned %v", tournament, next, err)
		}
	}
}
