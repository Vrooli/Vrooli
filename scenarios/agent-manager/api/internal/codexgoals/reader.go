// Package codexgoals reads Codex's local goal accounting as an optional,
// read-only observability source. It never creates, alters, or writes Codex's
// private SQLite store.
package codexgoals

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // registers the read-only SQLite driver
)

type Goal struct {
	ThreadID        string
	GoalID          string
	Objective       string
	Status          string
	TokenBudget     *int64
	TokensUsed      int64
	TimeUsedSeconds int64
}

// Read returns the accounting row for threadID from sessionHome. Missing
// stores and absent rows are normal and reported as (nil, nil).
func Read(ctx context.Context, sessionHome, threadID string) (*Goal, error) {
	if sessionHome == "" || threadID == "" {
		return nil, nil
	}
	path := filepath.Join(sessionHome, "goals_1.sqlite")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat codex goals store: %w", err)
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open codex goals store read-only: %w", err)
	}
	defer db.Close()
	goal := &Goal{}
	err = db.QueryRowContext(ctx, `SELECT thread_id, goal_id, objective, status, token_budget, tokens_used, time_used_seconds FROM thread_goals WHERE thread_id = ?`, threadID).
		Scan(&goal.ThreadID, &goal.GoalID, &goal.Objective, &goal.Status, &goal.TokenBudget, &goal.TokensUsed, &goal.TimeUsedSeconds)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read codex goal: %w", err)
	}
	return goal, nil
}
