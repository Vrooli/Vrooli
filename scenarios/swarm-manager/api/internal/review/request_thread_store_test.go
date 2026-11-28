package review

import (
	"errors"
	"testing"
)

func TestSaveRoundRejectsInvalidRequestThreadTransition(t *testing.T) {
	dir := t.TempDir()
	round := Round{
		RoundNum:    1,
		GeneratedAt: "2026-11-14T00:00:00Z",
		Status:      RoundStatusGathering,
		RequestThreads: []RequestThread{{
			ID:     "thread-1",
			Status: RequestThreadStatusPending,
		}},
	}
	if err := saveRound(dir, round); err != nil {
		t.Fatalf("save initial pending thread: %v", err)
	}

	if err := round.RequestThreads[0].TransitionStatus(RequestThreadStatusDismissed); err != nil {
		t.Fatal(err)
	}
	if err := saveRound(dir, round); err != nil {
		t.Fatalf("save legal dismissal: %v", err)
	}

	for _, next := range []RequestThreadStatus{RequestThreadStatusPending, RequestThreadStatusFulfilled, "unknown"} {
		round.RequestThreads[0].Status = next
		err := saveRound(dir, round)
		var transitionErr *InvalidRequestThreadTransitionError
		if !errors.As(err, &transitionErr) {
			t.Errorf("save terminal thread as %q error = %v, want InvalidRequestThreadTransitionError", next, err)
		}
	}

	stored, err := readRound(dir, 1)
	if err != nil {
		t.Fatalf("reload dismissed round: %v", err)
	}
	if stored.RequestThreads[0].Status != RequestThreadStatusDismissed {
		t.Fatalf("stored status = %q, want dismissed", stored.RequestThreads[0].Status)
	}
}

func TestSaveRoundRequiresNewRequestThreadsToStartPending(t *testing.T) {
	dir := t.TempDir()
	round := Round{
		RoundNum:    1,
		GeneratedAt: "2026-11-14T00:00:00Z",
		Status:      RoundStatusGathering,
		RequestThreads: []RequestThread{{
			ID:     "thread-1",
			Status: RequestThreadStatusFulfilled,
		}},
	}
	if err := saveRound(dir, round); err == nil {
		t.Fatal("save new fulfilled thread succeeded, want rejected non-pending initial state")
	}
	stored, err := readRound(dir, 1)
	if err != nil {
		t.Fatalf("check rejected new thread was not persisted: %v", err)
	}
	if stored != nil {
		t.Fatalf("stored round = %+v, want no round after rejected initial thread state", stored)
	}
}
