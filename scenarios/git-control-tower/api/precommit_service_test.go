package main

import (
	"context"
	"fmt"
	"testing"

	testdb "github.com/vrooli/api-core/databasetest"
)

type fakeCommandRunner struct {
	stdout   string
	stderr   string
	exitCode int
}

func (r fakeCommandRunner) Run(ctx context.Context, req CommandRunRequest) (CommandRunResult, error) {
	if r.exitCode != 0 {
		return CommandRunResult{Stdout: r.stdout, Stderr: r.stderr}, fakeExitError{code: r.exitCode}
	}
	return CommandRunResult{Stdout: r.stdout, Stderr: r.stderr}, nil
}

type fakeExitError struct {
	code int
}

func (e fakeExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.code)
}

func (e fakeExitError) ExitCode() int {
	return e.code
}

func newTestPrecommitService(t *testing.T) *PrecommitService {
	return newTestPrecommitServiceWithRunner(t, fakeCommandRunner{stdout: "ok"})
}

func newTestPrecommitServiceWithRunner(t *testing.T, runner CommandRunner) *PrecommitService {
	t.Helper()
	db := testdb.OpenSQLiteMemory(t)
	if err := ensureRepoSchema(db); err != nil {
		t.Fatalf("ensure repo schema: %v", err)
	}
	return NewPrecommitServiceWithRunner(db, runner)
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
	if result.Command != "printf ok" {
		t.Fatalf("result command = %q, want saved command", result.Command)
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
	svc := newTestPrecommitServiceWithRunner(t, fakeCommandRunner{stderr: "nope", exitCode: 7})
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
	if !passingGit.AssertCalledWith("Commit", "no_verify=true") {
		t.Fatalf("override commit must pass --no-verify so installed git hooks are bypassed; calls=%v", passingGit.Calls)
	}
}

func TestCreateCommitRecordsPassingPrecommitCheck(t *testing.T) {
	svc := newTestPrecommitServiceWithRunner(t, fakeCommandRunner{stdout: "lint ok"})
	store := newTestCommitCheckStore(t)
	repo := t.TempDir()
	ctx := context.Background()
	if _, err := svc.Save(ctx, repo, PrecommitConfig{
		Enabled:         true,
		Command:         "custom precommit",
		TimeoutSeconds:  30,
		RunBeforeCommit: true,
		AllowOverride:   true,
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	fakeGit := NewFakeGitRunner().AddStagedFile("file.go")
	result, err := CreateCommit(ctx, CommitDeps{
		Git:       fakeGit,
		RepoDir:   repo,
		Precommit: svc,
		Checks:    store,
	}, CommitRequest{Message: "fix: record check"})
	if err != nil {
		t.Fatalf("CreateCommit: %v", err)
	}
	if !result.Success {
		t.Fatalf("result = %#v", result)
	}

	checks, err := store.ListForCommits(ctx, repo, []string{result.Hash})
	if err != nil {
		t.Fatalf("ListForCommits: %v", err)
	}
	runs := checks[result.Hash]
	if len(runs) != 1 {
		t.Fatalf("runs = %#v, want one run", runs)
	}
	if runs[0].Kind != CommitCheckKindPrecommit || runs[0].Status != CommitCheckStatusPassed || runs[0].Command != "custom precommit" {
		t.Fatalf("run = %#v", runs[0])
	}
}
