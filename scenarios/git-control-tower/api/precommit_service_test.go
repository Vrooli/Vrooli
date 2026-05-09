package main

import (
	"context"
	"testing"

	testdb "git-control-tower/internal/testutil/db"
)

func newTestPrecommitService(t *testing.T) *PrecommitService {
	t.Helper()
	db := testdb.OpenSQLiteMemory(t)
	if err := ensureRepoSchema(db); err != nil {
		t.Fatalf("ensure repo schema: %v", err)
	}
	return NewPrecommitService(db)
}

func TestPrecommitServiceSaveGetAndRun(t *testing.T) {
	svc := newTestPrecommitService(t)
	repo := t.TempDir()
	ctx := context.Background()

	cfg, err := svc.Save(ctx, repo, PrecommitConfig{
		Enabled:         true,
		Command:         "printf ok",
		TimeoutSeconds:  30,
		RunBeforeCommit: true,
		AllowOverride:   true,
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !cfg.Enabled || cfg.Command != "printf ok" || cfg.WorkingDirectory != repo {
		t.Fatalf("saved config = %#v", cfg)
	}

	result, err := svc.Run(ctx, repo, PrecommitRunRequest{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "passed" || result.Stdout != "ok" {
		t.Fatalf("result = %#v, want passed with stdout ok", result)
	}

	loaded, err := svc.Get(ctx, repo)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if loaded.LastResult == nil || loaded.LastResult.Status != "passed" {
		t.Fatalf("loaded last result = %#v", loaded.LastResult)
	}
}

func TestCreateCommitPrecommitBlocksAndOverrideSkips(t *testing.T) {
	svc := newTestPrecommitService(t)
	repo := t.TempDir()
	ctx := context.Background()
	if _, err := svc.Save(ctx, repo, PrecommitConfig{
		Enabled:         true,
		Command:         "printf nope >&2; exit 7",
		TimeoutSeconds:  30,
		RunBeforeCommit: true,
		AllowOverride:   true,
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	failingGit := NewFakeGitRunner().AddStagedFile("file.go")
	blocked, err := CreateCommit(ctx, CommitDeps{
		Git:       failingGit,
		RepoDir:   repo,
		Precommit: svc,
	}, CommitRequest{Message: "fix: blocked"})
	if err != nil {
		t.Fatalf("CreateCommit blocked: %v", err)
	}
	if blocked.Success || blocked.Precommit == nil || blocked.Precommit.ExitCode != 7 {
		t.Fatalf("blocked response = %#v", blocked)
	}
	if failingGit.AssertCalled("Commit") {
		t.Fatalf("git commit should not run when precommit fails")
	}

	passingGit := NewFakeGitRunner().AddStagedFile("file.go")
	committed, err := CreateCommit(ctx, CommitDeps{
		Git:       passingGit,
		RepoDir:   repo,
		Precommit: svc,
	}, CommitRequest{Message: "fix: override", SkipPrecommitOnce: true})
	if err != nil {
		t.Fatalf("CreateCommit override: %v", err)
	}
	if !committed.Success || !passingGit.AssertCalled("Commit") {
		t.Fatalf("override response = %#v", committed)
	}
}
