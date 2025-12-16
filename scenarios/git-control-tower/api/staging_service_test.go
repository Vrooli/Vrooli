package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// [REQ:GCT-OT-P0-004] Stage/unstage operations

// --- Unit Tests using FakeGitRunner (fast, safe, no real git) ---

func TestStageFiles_RequiresGitRunner(t *testing.T) {
	ctx := context.Background()
	_, err := StageFiles(ctx, StagingDeps{
		Git:     nil,
		RepoDir: "/tmp",
	}, StageRequest{Paths: []string{"file.txt"}})
	if err == nil || !strings.Contains(err.Error(), "git runner is required") {
		t.Fatalf("expected 'git runner is required' error, got %v", err)
	}
}

func TestStageFiles_RequiresRepoDir(t *testing.T) {
	ctx := context.Background()
	fakeGit := NewFakeGitRunner()
	_, err := StageFiles(ctx, StagingDeps{
		Git:     fakeGit,
		RepoDir: "",
	}, StageRequest{Paths: []string{"file.txt"}})
	if err == nil || !strings.Contains(err.Error(), "repo dir is required") {
		t.Fatalf("expected 'repo dir is required' error, got %v", err)
	}
}

func TestStageFiles_EmptyPaths(t *testing.T) {
	ctx := context.Background()
	fakeGit := NewFakeGitRunner()
	result, err := StageFiles(ctx, StagingDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, StageRequest{Paths: []string{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success for empty paths")
	}
	if len(result.Staged) != 0 {
		t.Fatalf("expected no staged files")
	}
	// Ensure git was NOT called for empty paths
	if fakeGit.AssertCalled("Stage") {
		t.Fatalf("git stage should not be called for empty paths")
	}
}

func TestStageFiles_WithFakeGit(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		AddUntrackedFile("newfile.txt").
		AddUnstagedFile("modified.txt")

	result, err := StageFiles(context.Background(), StagingDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, StageRequest{
		Paths: []string{"newfile.txt", "modified.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got errors: %v", result.Errors)
	}
	if len(result.Staged) != 2 {
		t.Fatalf("expected 2 staged files, got %d", len(result.Staged))
	}
	if !fakeGit.AssertCalled("Stage") {
		t.Fatalf("expected Stage to be called")
	}
}

func TestStageFiles_PathTraversalBlocked(t *testing.T) {
	fakeGit := NewFakeGitRunner()

	result, err := StageFiles(context.Background(), StagingDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, StageRequest{
		Paths: []string{"../../../etc/passwd"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Path traversal paths should be filtered out
	if result.Success && len(result.Staged) > 0 {
		t.Fatalf("expected path traversal to be blocked, got staged: %v", result.Staged)
	}
}

func TestStageFiles_GitError(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	fakeGit.StageError = fmt.Errorf("simulated git add failure")

	result, err := StageFiles(context.Background(), StagingDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, StageRequest{
		Paths: []string{"file.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected failure due to git error")
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected errors to be populated")
	}
}

func TestUnstageFiles_RequiresGitRunner(t *testing.T) {
	ctx := context.Background()
	_, err := UnstageFiles(ctx, StagingDeps{
		Git:     nil,
		RepoDir: "/tmp",
	}, UnstageRequest{Paths: []string{"file.txt"}})
	if err == nil || !strings.Contains(err.Error(), "git runner is required") {
		t.Fatalf("expected 'git runner is required' error, got %v", err)
	}
}

func TestUnstageFiles_RequiresRepoDir(t *testing.T) {
	ctx := context.Background()
	fakeGit := NewFakeGitRunner()
	_, err := UnstageFiles(ctx, StagingDeps{
		Git:     fakeGit,
		RepoDir: "",
	}, UnstageRequest{Paths: []string{"file.txt"}})
	if err == nil || !strings.Contains(err.Error(), "repo dir is required") {
		t.Fatalf("expected 'repo dir is required' error, got %v", err)
	}
}

func TestUnstageFiles_WithFakeGit(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		AddStagedFile("staged.txt")

	result, err := UnstageFiles(context.Background(), StagingDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, UnstageRequest{
		Paths: []string{"staged.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got errors: %v", result.Errors)
	}
	if len(result.Unstaged) != 1 {
		t.Fatalf("expected 1 unstaged file, got %d", len(result.Unstaged))
	}
	if !fakeGit.AssertCalled("Unstage") {
		t.Fatalf("expected Unstage to be called")
	}
}

func TestUnstageFiles_GitError(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	fakeGit.UnstageError = fmt.Errorf("simulated git reset failure")

	result, err := UnstageFiles(context.Background(), StagingDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, UnstageRequest{
		Paths: []string{"file.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected failure due to git error")
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected errors to be populated")
	}
}

// --- Integration Tests using real git (marked with suffix _Integration) ---
// These tests require git to be installed and create real repos in temp directories.
// Keep these as a safety net to verify ExecGitRunner works with the actual git binary.

func TestStageFiles_WithRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := t.TempDir()
	runGitStaging(t, repoDir, "init")
	runGitStaging(t, repoDir, "checkout", "-b", "main")

	// Create a file
	filePath := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test content\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := StageFiles(ctx, StagingDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	}, StageRequest{
		Paths: []string{"test.txt"},
	})
	if err != nil {
		t.Fatalf("StageFiles failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got errors: %v", result.Errors)
	}
	if len(result.Staged) != 1 {
		t.Fatalf("expected 1 staged file, got %d", len(result.Staged))
	}
	if result.Staged[0] != "test.txt" {
		t.Fatalf("expected staged file 'test.txt', got %q", result.Staged[0])
	}

	// Verify file is actually staged
	out, _ := exec.Command("git", "-C", repoDir, "diff", "--cached", "--name-only").Output()
	if !strings.Contains(string(out), "test.txt") {
		t.Fatalf("file not actually staged, git diff --cached shows: %s", string(out))
	}
}

func TestUnstageFiles_WithRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := t.TempDir()
	runGitStaging(t, repoDir, "init")
	runGitStaging(t, repoDir, "checkout", "-b", "main")

	// Create and commit initial file so HEAD exists
	filePath := filepath.Join(repoDir, "initial.txt")
	if err := os.WriteFile(filePath, []byte("initial\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	runGitStaging(t, repoDir, "add", "initial.txt")
	runGitStaging(t, repoDir, "commit", "-m", "initial")

	// Create and stage a new file
	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	runGitStaging(t, repoDir, "add", "test.txt")

	// Verify it's staged
	out, _ := exec.Command("git", "-C", repoDir, "diff", "--cached", "--name-only").Output()
	if !strings.Contains(string(out), "test.txt") {
		t.Fatalf("file not staged before test: %s", string(out))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := UnstageFiles(ctx, StagingDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	}, UnstageRequest{
		Paths: []string{"test.txt"},
	})
	if err != nil {
		t.Fatalf("UnstageFiles failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got errors: %v", result.Errors)
	}
	if len(result.Unstaged) != 1 {
		t.Fatalf("expected 1 unstaged file, got %d", len(result.Unstaged))
	}

	// Verify file is no longer staged
	out, _ = exec.Command("git", "-C", repoDir, "diff", "--cached", "--name-only").Output()
	if strings.Contains(string(out), "test.txt") {
		t.Fatalf("file still staged after unstage: %s", string(out))
	}
}

func TestExpandScope(t *testing.T) {
	tests := []struct {
		scope    string
		expected []string
	}{
		{"scenario:foo", []string{"scenarios/foo/"}},
		{"resource:bar", []string{"resources/bar/"}},
		{"package:baz", []string{"packages/baz/"}},
		{"invalid", nil},
		{"", nil},
	}

	for _, tc := range tests {
		result := expandScope(tc.scope)
		if len(result) != len(tc.expected) {
			t.Errorf("expandScope(%q) = %v, want %v", tc.scope, result, tc.expected)
			continue
		}
		for i := range result {
			if result[i] != tc.expected[i] {
				t.Errorf("expandScope(%q)[%d] = %q, want %q", tc.scope, i, result[i], tc.expected[i])
			}
		}
	}
}

func TestCleanFilePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"file.txt", "file.txt"},
		{"/file.txt", "file.txt"},
		{"./file.txt", "file.txt"},
		{"../file.txt", "../file.txt"},
		{"  file.txt  ", "file.txt"},
		{"", ""},
		{"dir/file.txt", "dir/file.txt"},
		{"/dir/file.txt", "dir/file.txt"},
	}

	for _, tc := range tests {
		result := cleanFilePath(tc.input)
		if result != tc.expected {
			t.Errorf("cleanFilePath(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

// runGitStaging is an alias for RunGitCommand for backward compatibility.
// New tests should use RunGitCommand directly.
func runGitStaging(t *testing.T, dir string, args ...string) {
	RunGitCommand(t, dir, args...)
}
