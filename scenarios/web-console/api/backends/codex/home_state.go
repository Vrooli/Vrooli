package codex

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/cli-core/agenttranscript"
)

// A per-session Codex home keeps its durable record in files Codex owns: the
// rollout tree (one JSONL per thread, sub-agent threads included) and the goal
// store. These readers observe them without a running Codex.

// HomeGoal is one row of a Codex home's goal store (goals_1.sqlite).
type HomeGoal struct {
	ThreadID        string
	Status          string // active | paused | blocked | usage_limited | budget_limited | complete
	Updated         time.Time
	TokensUsed      int64
	TimeUsedSeconds int64
	TokenBudget     *int64
}

// NativeGoal is the goal as the app-server protocol names it (camelCase
// statuses), so a stored goal reads the same as one from a live owner.
func (g HomeGoal) NativeGoal() *NativeGoalState {
	status := g.Status
	switch status {
	case "usage_limited":
		status = "usageLimited"
	case "budget_limited":
		status = "budgetLimited"
	}
	return &NativeGoalState{ThreadID: g.ThreadID, Status: status, TokensUsed: g.TokensUsed, TimeUsedSeconds: g.TimeUsedSeconds, TokenBudget: g.TokenBudget}
}

// ReadHomeGoals returns every goal in a Codex home. A home without a goal
// store has no goals; a store that exists but cannot be read is an error, so a
// caller never mistakes "unreadable" for "no active goal".
func ReadHomeGoals(ctx context.Context, home string) ([]HomeGoal, error) {
	path := filepath.Join(home, "goals_1.sqlite")
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	uri := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}).String()
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	rows, err := db.QueryContext(ctx, `SELECT thread_id, status, updated_at_ms, tokens_used, time_used_seconds, token_budget FROM thread_goals ORDER BY updated_at_ms DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var goals []HomeGoal
	for rows.Next() {
		var goal HomeGoal
		var updated int64
		var budget sql.NullInt64
		if err := rows.Scan(&goal.ThreadID, &goal.Status, &updated, &goal.TokensUsed, &goal.TimeUsedSeconds, &budget); err != nil {
			return nil, err
		}
		goal.Updated = time.UnixMilli(updated)
		if budget.Valid {
			goal.TokenBudget = &budget.Int64
		}
		goals = append(goals, goal)
	}
	return goals, rows.Err()
}

// NewestRolloutWrite is the latest modification time of any rollout in a
// Codex home: the last moment any thread, a sub-agent's included, recorded
// work. ok is false when the home has no rollout tree.
func NewestRolloutWrite(home string) (newest time.Time, ok bool, err error) {
	root := filepath.Join(home, "sessions")
	if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
		if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
			return time.Time{}, false, statErr
		}
		return time.Time{}, false, nil
	}
	err = filepath.WalkDir(root, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return time.Time{}, false, err
	}
	return newest, true, nil
}

// RolloutRoot names the conversation a rollout belongs to. A sub-agent's
// rollout carries its own id plus the id of the root thread it was spawned
// under (session_id); a root rollout's session_id is its own id or absent.
type RolloutRoot struct {
	ThreadID string // this rollout's thread
	RootID   string // the root thread of its conversation
	CWD      string
	SubAgent bool
}

// ReadRolloutRoot reads the session_meta line that opens a rollout.
func ReadRolloutRoot(path string) (RolloutRoot, bool) {
	f, err := os.Open(path)
	if err != nil {
		return RolloutRoot{}, false
	}
	defer f.Close()
	line, _ := bufio.NewReaderSize(f, 128*1024).ReadBytes('\n')
	if len(line) == 0 {
		return RolloutRoot{}, false
	}
	meta, ok := agenttranscript.ParseCodexRolloutSessionMeta(line)
	if !ok || meta.ID == "" {
		return RolloutRoot{}, false
	}
	root := RolloutRoot{ThreadID: meta.ID, RootID: meta.ID, CWD: meta.CWD}
	if meta.SessionID != "" && meta.SessionID != meta.ID {
		root.RootID = meta.SessionID
		root.SubAgent = true
	}
	if meta.ThreadSource == "subagent" {
		root.SubAgent = true
	}
	return root, true
}

// RootThreadID resolves a thread recorded for a session to the root thread of
// its conversation, the one `codex resume` must open (and the one a goal is
// set on). It returns threadID unchanged when no rollout for it is found.
func RootThreadID(home, threadID string) string {
	if threadID == "" {
		return threadID
	}
	matches, err := filepath.Glob(filepath.Join(home, "sessions", "*", "*", "*", "rollout-*-"+threadID+".jsonl"))
	if err != nil || len(matches) == 0 {
		return threadID
	}
	if root, ok := ReadRolloutRoot(matches[0]); ok && root.ThreadID == threadID {
		return root.RootID
	}
	return threadID
}
