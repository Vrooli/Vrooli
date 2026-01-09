package main

import (
	"context"
	"fmt"
	"testing"
)

// [REQ:GCT-OT-P0-005] Commit composition API

func TestCreateCommit_RequiresGitRunner(t *testing.T) {
	_, err := CreateCommit(context.Background(), CommitDeps{
		Git:     nil,
		RepoDir: "/fake/repo",
	}, CommitRequest{Message: "test"})
	if err == nil || err.Error() != "git runner is required" {
		t.Fatalf("expected 'git runner is required' error, got %v", err)
	}
}

func TestCreateCommit_RequiresRepoDir(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	_, err := CreateCommit(context.Background(), CommitDeps{
		Git:     fakeGit,
		RepoDir: "",
	}, CommitRequest{Message: "test"})
	if err == nil || err.Error() != "repo dir is required" {
		t.Fatalf("expected 'repo dir is required' error, got %v", err)
	}
}

func TestCreateCommit_RequiresMessage(t *testing.T) {
	fakeGit := NewFakeGitRunner().AddStagedFile("file.go")
	result, err := CreateCommit(context.Background(), CommitDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, CommitRequest{Message: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected success=false for empty message")
	}
	if len(result.ValidationErrors) == 0 {
		t.Fatalf("expected validation error for empty message")
	}
}

func TestCreateCommit_WithFakeGit(t *testing.T) {
	fakeGit := NewFakeGitRunner().AddStagedFile("file.go")
	result, err := CreateCommit(context.Background(), CommitDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, CommitRequest{Message: "feat: add new feature"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success=true, got error=%q", result.Error)
	}
	if result.Hash == "" {
		t.Fatalf("expected commit hash to be set")
	}
	if fakeGit.LastCommitMessage != "feat: add new feature" {
		t.Fatalf("expected message='feat: add new feature', got %q", fakeGit.LastCommitMessage)
	}
	if !fakeGit.AssertCalled("Commit") {
		t.Fatalf("expected Commit to be called")
	}
}

func TestCreateCommit_NothingToCommit(t *testing.T) {
	fakeGit := NewFakeGitRunner() // No staged files
	result, err := CreateCommit(context.Background(), CommitDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, CommitRequest{Message: "fix: something"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected success=false when nothing to commit")
	}
	if result.Error == "" {
		t.Fatalf("expected error message when nothing to commit")
	}
}

func TestCreateCommit_GitError(t *testing.T) {
	fakeGit := NewFakeGitRunner().AddStagedFile("file.go")
	fakeGit.CommitError = fmt.Errorf("permission denied")
	result, err := CreateCommit(context.Background(), CommitDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, CommitRequest{Message: "test commit"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected success=false for git error")
	}
	if result.Error != "permission denied" {
		t.Fatalf("expected error='permission denied', got %q", result.Error)
	}
}

// --- Conventional Commit Validation Tests ---

func TestValidateConventionalCommit_ValidFormats(t *testing.T) {
	validMessages := []string{
		"feat: add new feature",
		"fix: resolve null pointer",
		"docs: update readme",
		"feat(auth): add login endpoint",
		"fix(api): handle edge case",
		"refactor(core): simplify logic",
		"test(unit): add coverage",
		"chore: update dependencies",
		"feat!: breaking change",
		"feat(api)!: breaking change with scope",
		"ci: update workflow",
		"build: update toolchain",
		"perf: optimize query",
		"style: format code",
		"revert: undo previous commit",
	}
	for _, msg := range validMessages {
		errs := ValidateConventionalCommit(msg)
		if len(errs) > 0 {
			t.Errorf("expected %q to be valid, got errors: %v", msg, errs)
		}
	}
}

func TestValidateConventionalCommit_InvalidFormats(t *testing.T) {
	invalidMessages := []string{
		"",                       // Empty
		"add feature",            // No type prefix
		"feat",                   // No description
		"feat:",                  // No description after colon
		"feat: ",                 // Only whitespace after colon
		"Feat: capitalized",      // Uppercase type
		"unknown: invalid type",  // Invalid type
		"feat(CAPS): scope caps", // Uppercase scope
	}
	for _, msg := range invalidMessages {
		errs := ValidateConventionalCommit(msg)
		if len(errs) == 0 {
			t.Errorf("expected %q to be invalid, got no errors", msg)
		}
	}
}

func TestCreateCommit_ConventionalValidation_Pass(t *testing.T) {
	fakeGit := NewFakeGitRunner().AddStagedFile("file.go")
	result, err := CreateCommit(context.Background(), CommitDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, CommitRequest{
		Message:              "feat(api): add commit endpoint",
		ValidateConventional: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success=true, got validation_errors=%v error=%q",
			result.ValidationErrors, result.Error)
	}
}

func TestCreateCommit_ConventionalValidation_Fail(t *testing.T) {
	fakeGit := NewFakeGitRunner().AddStagedFile("file.go")
	result, err := CreateCommit(context.Background(), CommitDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, CommitRequest{
		Message:              "add new feature", // Invalid format
		ValidateConventional: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected success=false for invalid conventional commit")
	}
	if len(result.ValidationErrors) == 0 {
		t.Fatalf("expected validation errors")
	}
	// Should not have called git commit
	if fakeGit.AssertCalled("Commit") {
		t.Fatalf("expected Commit NOT to be called for invalid message")
	}
}

func TestCreateCommit_ConventionalValidation_Disabled(t *testing.T) {
	fakeGit := NewFakeGitRunner().AddStagedFile("file.go")
	result, err := CreateCommit(context.Background(), CommitDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, CommitRequest{
		Message:              "any message format works",
		ValidateConventional: false, // Validation disabled
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success=true when validation disabled, got error=%q", result.Error)
	}
}

func TestCreateCommit_WithRealRepo(t *testing.T) {
	repoDir := SetupTestRepo(t)

	// Create and stage a file
	WriteTestFile(t, repoDir+"/test.txt", "hello world")
	RunGitCommand(t, repoDir, "add", "test.txt")

	git := &ExecGitRunner{GitPath: "git"}
	result, err := CreateCommit(context.Background(), CommitDeps{
		Git:     git,
		RepoDir: repoDir,
	}, CommitRequest{
		Message:              "feat: add test file",
		ValidateConventional: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success=true, got error=%q validation_errors=%v",
			result.Error, result.ValidationErrors)
	}
	if result.Hash == "" {
		t.Fatalf("expected commit hash to be set")
	}
	if len(result.Hash) < 7 {
		t.Fatalf("expected short hash (7+ chars), got %q", result.Hash)
	}
}
