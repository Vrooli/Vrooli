package execution

import (
	"errors"
	"testing"
)

func TestRecordStatusTransitionMatrix(t *testing.T) {
	states := []Status{
		StatusPending, StatusStarting, StatusRunning, StatusCancelling,
		StatusNeedsReview, StatusValidating, StatusNeedsFixup, StatusCompleted,
		StatusAbstained, StatusBudgetExhausted, StatusInterrupted,
		StatusNeedsAttention, StatusFailed, StatusCanceled,
	}
	want := map[Status][]Status{
		StatusPending:         {StatusStarting, StatusCancelling, StatusCanceled, StatusFailed},
		StatusStarting:        {StatusRunning, StatusCancelling, StatusNeedsReview, StatusValidating, StatusNeedsFixup, StatusCompleted, StatusAbstained, StatusBudgetExhausted, StatusInterrupted, StatusNeedsAttention, StatusFailed, StatusCanceled},
		StatusRunning:         {StatusCancelling, StatusNeedsReview, StatusValidating, StatusNeedsFixup, StatusCompleted, StatusAbstained, StatusBudgetExhausted, StatusInterrupted, StatusNeedsAttention, StatusFailed, StatusCanceled},
		StatusCancelling:      {StatusCanceled, StatusInterrupted, StatusFailed},
		StatusNeedsReview:     {StatusCancelling, StatusNeedsAttention, StatusValidating, StatusNeedsFixup, StatusCompleted, StatusAbstained, StatusInterrupted, StatusFailed, StatusCanceled},
		StatusValidating:      {StatusNeedsFixup, StatusCompleted, StatusFailed, StatusCanceled},
		StatusNeedsFixup:      {StatusStarting, StatusCompleted, StatusFailed, StatusCanceled},
		StatusInterrupted:     {StatusStarting, StatusCanceled},
		StatusNeedsAttention:  {StatusStarting, StatusCancelling, StatusFailed, StatusCanceled},
		StatusFailed:          {StatusStarting, StatusValidating, StatusCompleted},
		StatusAbstained:       {StatusStarting},
		StatusBudgetExhausted: {StatusStarting},
		StatusCompleted:       {StatusFailed, StatusValidating},
		StatusCanceled:        {},
	}
	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				record := Record{Status: from}
				err := record.TransitionStatus(to)
				allowed := containsStatus(want[from], to)
				_, implemented := recordStatusTransitions[from][to]
				if allowed != implemented {
					t.Fatalf("transition graph drift: %q -> %q allowed=%t, want %t", from, to, implemented, allowed)
				}
				if from == to {
					if err != nil || record.Status != from {
						t.Fatalf("same-state transition: status=%q err=%v", record.Status, err)
					}
					return
				}
				if allowed {
					if err != nil || record.Status != to {
						t.Fatalf("allowed transition: status=%q err=%v, want %q", record.Status, err, to)
					}
					return
				}
				if !errors.Is(err, ErrInvalidRecordStatusTransition) || record.Status != from {
					t.Fatalf("invalid transition mutated state: status=%q err=%v, want %q unchanged", record.Status, err, from)
				}
			})
		}
	}
}

func containsStatus(states []Status, target Status) bool {
	for _, state := range states {
		if state == target {
			return true
		}
	}
	return false
}

func TestRecordStatusTransitionRejectsUnknownStatesAndNilReceiver(t *testing.T) {
	for _, test := range []struct {
		name   string
		record *Record
		next   Status
	}{
		{name: "unknown current", record: &Record{Status: "future_state"}, next: StatusRunning},
		{name: "unknown target", record: &Record{Status: StatusPending}, next: "future_state"},
		{name: "nil receiver", record: nil, next: StatusRunning},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.record.TransitionStatus(test.next); !errors.Is(err, ErrInvalidRecordStatusTransition) {
				t.Fatalf("TransitionStatus error = %v, want invalid-transition error", err)
			}
		})
	}
}
