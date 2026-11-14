package agentactivity

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestActivityStatusLifecycleTransitionMatrix(t *testing.T) {
	states := []Status{
		StatusPending, StatusStarting, StatusRunning, StatusNeedsReview,
		StatusComplete, StatusAbstained, StatusBudgetExhausted,
		StatusFailed, StatusCancelled, StatusUnspecified,
	}
	want := map[Status]map[Status]bool{
		StatusPending: {
			StatusPending: true, StatusStarting: true, StatusRunning: true,
			StatusComplete: true, StatusAbstained: true, StatusBudgetExhausted: true,
			StatusFailed: true, StatusCancelled: true, StatusUnspecified: true,
		},
		StatusStarting: {
			StatusStarting: true, StatusRunning: true, StatusNeedsReview: true,
			StatusComplete: true, StatusAbstained: true, StatusBudgetExhausted: true,
			StatusFailed: true, StatusCancelled: true, StatusUnspecified: true,
		},
		StatusRunning: {
			StatusRunning: true, StatusNeedsReview: true, StatusComplete: true,
			StatusAbstained: true, StatusBudgetExhausted: true, StatusFailed: true,
			StatusCancelled: true, StatusUnspecified: true,
		},
		StatusNeedsReview: {
			StatusNeedsReview: true, StatusRunning: true, StatusComplete: true,
			StatusAbstained: true, StatusBudgetExhausted: true, StatusFailed: true,
			StatusCancelled: true, StatusUnspecified: true,
		},
		StatusComplete:        {StatusComplete: true},
		StatusAbstained:       {StatusAbstained: true},
		StatusBudgetExhausted: {StatusBudgetExhausted: true},
		StatusFailed:          {StatusFailed: true},
		StatusCancelled:       {StatusCancelled: true},
		StatusUnspecified:     {StatusUnspecified: true},
	}

	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"->"+string(to), func(t *testing.T) {
				record := Record{Status: from}
				err := record.TransitionStatus(to)
				if want[from][to] {
					if err != nil || record.Status != to {
						t.Fatalf("transition status=%s want=%s err=%v", record.Status, to, err)
					}
					return
				}
				if !errors.Is(err, ErrInvalidStatusTransition) {
					t.Fatalf("transition %s -> %s error=%v, want invalid transition", from, to, err)
				}
				if record.Status != from {
					t.Fatalf("rejected transition mutated status to %s", record.Status)
				}
			})
		}
	}

	var nilRecord *Record
	if err := nilRecord.TransitionStatus(StatusRunning); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("nil record transition error=%v", err)
	}
	for _, status := range []Status{"", "invalid"} {
		if err := (&Record{Status: StatusPending}).TransitionStatus(status); !errors.Is(err, ErrInvalidStatusTransition) {
			t.Fatalf("invalid target %q error=%v", status, err)
		}
	}
}

func TestFileStoreRejectsUnknownDurableActivityStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activities.json")
	store := NewStore(path)
	if err := store.Save([]Record{{ActivityID: "write-invalid", Status: Status("invalid")}}); err == nil {
		t.Fatal("store accepted an unknown status for a durable write")
	}
	payload, err := json.Marshal([]Record{{ActivityID: "read-invalid", Status: Status("invalid")}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); err == nil {
		t.Fatal("store accepted an unknown persisted status")
	}
}
