package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// [REQ:GCT-OT-P0-002] Repository status API

// --- Unit Tests using FakeGitRunner (fast, safe, no real git) ---

func TestGetRepoStatus_WithFakeGit(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		WithBranch("feature/test", "origin/feature/test", 3, 1).
		AddStagedFile("scenarios/alpha/file.go").
		AddUnstagedFile("resources/beta/config.yaml").
		AddUntrackedFile("notes.txt")
	// Note: Not adding conflicts here as they have complex XY behavior

	status, err := GetRepoStatus(context.Background(), RepoStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify branch info
	if status.Branch.Head != "feature/test" {
		t.Fatalf("expected branch head=feature/test, got %q", status.Branch.Head)
	}
	if status.Branch.Upstream != "origin/feature/test" {
		t.Fatalf("expected upstream origin/feature/test, got %q", status.Branch.Upstream)
	}
	if status.Branch.Ahead != 3 {
		t.Fatalf("expected ahead=3, got %d", status.Branch.Ahead)
	}
	if status.Branch.Behind != 1 {
		t.Fatalf("expected behind=1, got %d", status.Branch.Behind)
	}

	// Verify summary counts
	if status.Summary.Staged != 1 {
		t.Fatalf("expected 1 staged file, got %d", status.Summary.Staged)
	}
	if status.Summary.Unstaged != 1 {
		t.Fatalf("expected 1 unstaged file, got %d", status.Summary.Unstaged)
	}
	if status.Summary.Untracked != 1 {
		t.Fatalf("expected 1 untracked file, got %d", status.Summary.Untracked)
	}

	// Verify git was called
	if !fakeGit.AssertCalled("StatusPorcelainV2") {
		t.Fatalf("expected StatusPorcelainV2 to be called")
	}
}

func TestGetRepoStatus_WithConflicts(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		AddConflictFile("conflicted.txt")

	status, err := GetRepoStatus(context.Background(), RepoStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Conflicts with UU status appear in both staged and unstaged
	// This is correct git behavior - both index and worktree differ from base
	if status.Summary.Conflicts != 1 {
		t.Fatalf("expected 1 conflict file, got %d", status.Summary.Conflicts)
	}
	if status.Summary.Staged < 1 {
		t.Fatalf("expected at least 1 staged (from conflict), got %d", status.Summary.Staged)
	}
	if status.Summary.Unstaged < 1 {
		t.Fatalf("expected at least 1 unstaged (from conflict), got %d", status.Summary.Unstaged)
	}
}

func TestGetRepoStatus_GitError(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	fakeGit.StatusError = fmt.Errorf("simulated git status failure")

	_, err := GetRepoStatus(context.Background(), RepoStatusDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	})
	if err == nil {
		t.Fatalf("expected error from git failure")
	}
	if !strings.Contains(err.Error(), "simulated git status failure") {
		t.Fatalf("expected error to contain 'simulated git status failure', got: %v", err)
	}
}

func TestGetRepoStatus_RequiresGitRunner(t *testing.T) {
	_, err := GetRepoStatus(context.Background(), RepoStatusDeps{
		Git:     nil,
		RepoDir: "/fake/repo",
	})
	if err == nil || !strings.Contains(err.Error(), "git runner is required") {
		t.Fatalf("expected 'git runner is required' error, got %v", err)
	}
}

func TestGetRepoStatus_RequiresRepoDir(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	_, err := GetRepoStatus(context.Background(), RepoStatusDeps{
		Git:     fakeGit,
		RepoDir: "",
	})
	if err == nil || !strings.Contains(err.Error(), "repo dir is required") {
		t.Fatalf("expected 'repo dir is required' error, got %v", err)
	}
}

// --- Parser Tests (pure functions, no git needed) ---

func TestParsePorcelainV2Status_BranchAndFiles(t *testing.T) {
	out := []byte(strings.Join([]string{
		"# branch.oid 0123456789abcdef",
		"# branch.head main",
		"# branch.upstream origin/main",
		"# branch.ab +2 -1",
		"1 M. N... 100644 100644 100644 abcdef1 abcdef2 file1.txt",
		"1 .M N... 100644 100644 100644 abcdef1 abcdef2 file2.txt",
		"? untracked.txt",
		"! ignored.log",
		"u UU N... 100644 100644 100644 100644 abcdef1 abcdef2 abcdef3 conflict.txt",
		"",
	}, "\x00"))

	parsed, err := ParsePorcelainV2Status(out)
	if err != nil {
		t.Fatalf("ParsePorcelainV2Status failed: %v", err)
	}

	if parsed.Branch.Head != "main" {
		t.Fatalf("expected branch head=main, got %q", parsed.Branch.Head)
	}
	if parsed.Branch.Upstream != "origin/main" {
		t.Fatalf("expected upstream origin/main, got %q", parsed.Branch.Upstream)
	}
	if parsed.Branch.Ahead != 2 || parsed.Branch.Behind != 1 {
		t.Fatalf("expected ahead=2 behind=1, got ahead=%d behind=%d", parsed.Branch.Ahead, parsed.Branch.Behind)
	}

	assertContains(t, parsed.Files.Staged, "file1.txt")
	assertContains(t, parsed.Files.Unstaged, "file2.txt")
	assertContains(t, parsed.Files.Untracked, "untracked.txt")
	assertContains(t, parsed.Files.Ignored, "ignored.log")
	assertContains(t, parsed.Files.Conflicts, "conflict.txt")
}

func TestGetRepoStatus_UsesGitAndDetectsScopes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "checkout", "-b", "main")

	writeFile(t, filepath.Join(repoDir, "scenarios", "alpha", "README.md"), "alpha")
	writeFile(t, filepath.Join(repoDir, "resources", "beta", "README.md"), "beta")
	writeFile(t, filepath.Join(repoDir, "notes.txt"), "notes")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := GetRepoStatus(ctx, RepoStatusDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	})
	if err != nil {
		t.Fatalf("GetRepoStatus failed: %v", err)
	}

	if status.Branch.Head == "" {
		t.Fatalf("expected branch head to be set")
	}
	if len(status.Files.Untracked) < 3 {
		t.Fatalf("expected at least 3 untracked files, got %d (%v)", len(status.Files.Untracked), status.Files.Untracked)
	}
	if len(status.Scopes["scenario:alpha"]) == 0 {
		t.Fatalf("expected scenario:alpha scope to be detected, got scopes=%v", status.Scopes)
	}
	if len(status.Scopes["resource:beta"]) == 0 {
		t.Fatalf("expected resource:beta scope to be detected, got scopes=%v", status.Scopes)
	}
	if len(status.Scopes["other"]) == 0 {
		t.Fatalf("expected other scope to be detected, got scopes=%v", status.Scopes)
	}
}

// runGit is an alias for RunGitCommand for backward compatibility.
// New tests should use RunGitCommand directly.
func runGit(t *testing.T, dir string, args ...string) {
	RunGitCommand(t, dir, args...)
}

// writeFile is an alias for WriteTestFile for backward compatibility.
// New tests should use WriteTestFile directly.
func writeFile(t *testing.T, path string, contents string) {
	WriteTestFile(t, path, contents)
}

// assertContains is an alias for AssertContains for backward compatibility.
// New tests should use AssertContains directly.
func assertContains(t *testing.T, values []string, expected string) {
	AssertContains(t, values, expected)
}
