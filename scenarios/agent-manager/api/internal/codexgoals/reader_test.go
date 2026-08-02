package codexgoals

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRead(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "goals_1.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE thread_goals (thread_id TEXT PRIMARY KEY, goal_id TEXT NOT NULL, objective TEXT NOT NULL, status TEXT NOT NULL, token_budget INTEGER, tokens_used INTEGER NOT NULL, time_used_seconds INTEGER NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO thread_goals VALUES ('thread-1','goal-1','ship safely','active',1000,250,12)`); err != nil {
		t.Fatal(err)
	}
	goal, err := Read(context.Background(), dir, "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	if goal == nil || goal.GoalID != "goal-1" || goal.TokenBudget == nil || *goal.TokenBudget != 1000 || goal.TokensUsed != 250 {
		t.Fatalf("unexpected goal: %#v", goal)
	}
	missing, err := Read(context.Background(), dir, "missing")
	if err != nil || missing != nil {
		t.Fatalf("missing = %#v, %v", missing, err)
	}
}

func TestReadMissingStoreIsNotAnError(t *testing.T) {
	goal, err := Read(context.Background(), t.TempDir(), "thread")
	if err != nil || goal != nil {
		t.Fatalf("Read missing store = %#v, %v", goal, err)
	}
}

func TestReadUnreadableStoreReturnsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "goals_1.sqlite"), 0o700); err != nil {
		t.Fatal(err)
	}
	goal, err := Read(context.Background(), dir, "thread")
	if err == nil || goal != nil {
		t.Fatalf("Read unreadable store = %#v, %v", goal, err)
	}
}
