package main

import (
	"context"
	"testing"
	"time"

	testdb "github.com/vrooli/api-core/databasetest"
)

func newTestCommitCheckStore(t *testing.T) *CommitCheckStore {
	t.Helper()
	db := testdb.OpenSQLiteMemory(t)
	if err := ensureRepoSchema(db); err != nil {
		t.Fatalf("ensure repo schema: %v", err)
	}
	return NewCommitCheckStore(db)
}

func TestCommitCheckStoreSaveAndListForCommits(t *testing.T) {
	store := newTestCommitCheckStore(t)
	ctx := context.Background()
	when := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)

	run := CommitCheckRun{
		Kind:       CommitCheckKindPrecommit,
		Status:     CommitCheckStatusPassed,
		Command:    "custom check",
		ExitCode:   0,
		Summary:    "checks passed",
		Stdout:     "ok",
		DurationMs: 42,
		Timestamp:  when,
	}
	if err := store.Save(ctx, "/repo", "abc1234", run); err != nil {
		t.Fatalf("Save: %v", err)
	}

	byHash, err := store.ListForCommits(ctx, "/repo", []string{"abc1234", "missing"})
	if err != nil {
		t.Fatalf("ListForCommits: %v", err)
	}
	runs := byHash["abc1234"]
	if len(runs) != 1 {
		t.Fatalf("runs = %#v, want one run", runs)
	}
	if runs[0].Kind != CommitCheckKindPrecommit || runs[0].Status != CommitCheckStatusPassed || runs[0].Command != "custom check" {
		t.Fatalf("run = %#v", runs[0])
	}
	if _, ok := byHash["missing"]; ok {
		t.Fatalf("missing hash should not have runs")
	}
}
