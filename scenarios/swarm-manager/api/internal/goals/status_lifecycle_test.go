package goals

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoalStatusTransitionMatrix(t *testing.T) {
	statuses := []GoalStatus{StatusActive, StatusAchieved, StatusArchived, "unknown"}
	allowed := map[GoalStatus]map[GoalStatus]bool{
		StatusActive:   {StatusActive: true, StatusAchieved: true, StatusArchived: true},
		StatusAchieved: {StatusAchieved: true, StatusArchived: true},
		StatusArchived: {StatusArchived: true, StatusActive: true},
	}
	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				goal := Goal{Status: from}
				err := goal.TransitionStatus(to)
				if allowed[from][to] {
					if err != nil {
						t.Fatalf("TransitionStatus(%q): %v", to, err)
					}
					if goal.Status != to {
						t.Fatalf("status = %q, want %q", goal.Status, to)
					}
					return
				}
				if err == nil {
					t.Fatalf("TransitionStatus(%q) unexpectedly succeeded", to)
				}
				if goal.Status != from {
					t.Fatalf("rejected transition changed status to %q", goal.Status)
				}
			})
		}
	}
	var nilGoal *Goal
	if err := nilGoal.TransitionStatus(StatusActive); err == nil {
		t.Fatal("nil goal transition unexpectedly succeeded")
	}
}

func TestGoalStoreRejectsInvalidStatusWithoutChangingDurableRecord(t *testing.T) {
	store := NewStore(t.TempDir())
	goal := &Goal{Name: "g", Status: StatusActive}
	if err := store.Save(goal); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.GoalDir(goal.Name), goalFileName)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	for _, next := range []GoalStatus{"unknown"} {
		candidate := *goal
		candidate.Status = next
		if err := store.Save(&candidate); err == nil {
			t.Fatalf("Save status %q unexpectedly succeeded", next)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Fatalf("rejected save changed durable record: before=%s after=%s", before, after)
		}
	}
	if err := goal.TransitionStatus(StatusAchieved); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(goal); err != nil {
		t.Fatal(err)
	}
	before, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	goal.Status = StatusActive
	if err := store.Save(goal); err == nil {
		t.Fatal("Save of achieved-to-active transition unexpectedly succeeded")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("rejected resurrection changed durable record: before=%s after=%s", before, after)
	}
}

func TestGoalStoreReloadsAchievementArchiveAndUnarchive(t *testing.T) {
	store := NewStore(t.TempDir())
	goal := &Goal{Name: "g", Status: StatusActive}
	if err := store.Save(goal); err != nil {
		t.Fatal(err)
	}
	for _, status := range []GoalStatus{StatusAchieved, StatusArchived, StatusActive} {
		if err := goal.TransitionStatus(status); err != nil {
			t.Fatalf("transition to %q: %v", status, err)
		}
		if err := store.Save(goal); err != nil {
			t.Fatalf("save %q: %v", status, err)
		}
		reloaded, err := store.Load(goal.Name)
		if err != nil {
			t.Fatalf("reload %q: %v", status, err)
		}
		if reloaded.Status != status {
			t.Fatalf("reloaded status = %q, want %q", reloaded.Status, status)
		}
	}
}
