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

// [REQ:GCT-OT-P0-003] File diff endpoint

// --- Unit Tests using FakeGitRunner (fast, safe, no real git) ---

func TestGetDiff_WithFakeGit(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		AddUnstagedFile("modified.txt")

	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, DiffRequest{
		Path:   "modified.txt",
		Staged: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !diff.HasDiff {
		t.Fatalf("expected HasDiff=true")
	}
	if !fakeGit.AssertCalled("Diff") {
		t.Fatalf("expected Diff to be called")
	}
}

func TestGetDiff_WithFakeGit_Staged(t *testing.T) {
	fakeGit := NewFakeGitRunner().
		AddStagedFile("staged.txt")

	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, DiffRequest{
		Path:   "staged.txt",
		Staged: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !diff.HasDiff {
		t.Fatalf("expected HasDiff=true for staged file")
	}
	if diff.Staged != true {
		t.Fatalf("expected Staged=true in response")
	}
}

func TestGetDiff_WithFakeGit_NoDiff(t *testing.T) {
	fakeGit := NewFakeGitRunner() // No files with changes

	diff, err := GetDiff(context.Background(), DiffDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, DiffRequest{
		Path:   "clean.txt",
		Staged: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff.HasDiff {
		t.Fatalf("expected HasDiff=false for clean file")
	}
}

func TestGetDiff_GitError(t *testing.T) {
	fakeGit := NewFakeGitRunner()
	fakeGit.DiffError = fmt.Errorf("simulated git diff failure")

	_, err := GetDiff(context.Background(), DiffDeps{
		Git:     fakeGit,
		RepoDir: "/fake/repo",
	}, DiffRequest{
		Path:   "file.txt",
		Staged: false,
	})
	if err == nil {
		t.Fatalf("expected error from git failure")
	}
	if !strings.Contains(err.Error(), "simulated git diff failure") {
		t.Fatalf("expected error to contain 'simulated git diff failure', got: %v", err)
	}
}

// --- Parser Tests (pure functions, no git needed) ---

func TestParseDiffOutput_EmptyDiff(t *testing.T) {
	result := ParseDiffOutput("")
	if result.HasDiff {
		t.Fatalf("expected HasDiff=false for empty input, got true")
	}
	if len(result.Hunks) != 0 {
		t.Fatalf("expected 0 hunks, got %d", len(result.Hunks))
	}
}

func TestParseDiffOutput_SingleHunk(t *testing.T) {
	input := `diff --git a/file.txt b/file.txt
index 1234567..abcdef0 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 line1
+added line
 line2
 line3
`
	result := ParseDiffOutput(input)
	if !result.HasDiff {
		t.Fatalf("expected HasDiff=true, got false")
	}
	if len(result.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(result.Hunks))
	}
	if result.Hunks[0].OldStart != 1 {
		t.Fatalf("expected OldStart=1, got %d", result.Hunks[0].OldStart)
	}
	if result.Hunks[0].OldCount != 3 {
		t.Fatalf("expected OldCount=3, got %d", result.Hunks[0].OldCount)
	}
	if result.Hunks[0].NewStart != 1 {
		t.Fatalf("expected NewStart=1, got %d", result.Hunks[0].NewStart)
	}
	if result.Hunks[0].NewCount != 4 {
		t.Fatalf("expected NewCount=4, got %d", result.Hunks[0].NewCount)
	}
	if result.Stats.Additions != 1 {
		t.Fatalf("expected 1 addition, got %d", result.Stats.Additions)
	}
	if result.Stats.Deletions != 0 {
		t.Fatalf("expected 0 deletions, got %d", result.Stats.Deletions)
	}
	if result.Stats.Files != 1 {
		t.Fatalf("expected 1 file, got %d", result.Stats.Files)
	}
}

func TestParseDiffOutput_MultipleHunks(t *testing.T) {
	input := `diff --git a/file.txt b/file.txt
index 1234567..abcdef0 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,2 @@
 line1
-removed
 line2
@@ -10,3 +9,4 @@
 line10
+added
 line11
 line12
`
	result := ParseDiffOutput(input)
	if len(result.Hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(result.Hunks))
	}
	if result.Hunks[0].OldStart != 1 {
		t.Fatalf("expected first hunk OldStart=1, got %d", result.Hunks[0].OldStart)
	}
	if result.Hunks[1].OldStart != 10 {
		t.Fatalf("expected second hunk OldStart=10, got %d", result.Hunks[1].OldStart)
	}
	if result.Stats.Additions != 1 {
		t.Fatalf("expected 1 addition, got %d", result.Stats.Additions)
	}
	if result.Stats.Deletions != 1 {
		t.Fatalf("expected 1 deletion, got %d", result.Stats.Deletions)
	}
}

func TestParseDiffOutput_MultipleFiles(t *testing.T) {
	input := `diff --git a/file1.txt b/file1.txt
index 1234567..abcdef0 100644
--- a/file1.txt
+++ b/file1.txt
@@ -1,1 +1,2 @@
 line1
+added
diff --git a/file2.txt b/file2.txt
index 1234567..abcdef0 100644
--- a/file2.txt
+++ b/file2.txt
@@ -1,2 +1,1 @@
 line1
-removed
`
	result := ParseDiffOutput(input)
	if result.Stats.Files != 2 {
		t.Fatalf("expected 2 files, got %d", result.Stats.Files)
	}
	if result.Stats.Additions != 1 {
		t.Fatalf("expected 1 addition, got %d", result.Stats.Additions)
	}
	if result.Stats.Deletions != 1 {
		t.Fatalf("expected 1 deletion, got %d", result.Stats.Deletions)
	}
}

func TestGetDiff_RequiresGitRunner(t *testing.T) {
	ctx := context.Background()
	_, err := GetDiff(ctx, DiffDeps{
		Git:     nil,
		RepoDir: "/tmp",
	}, DiffRequest{})
	if err == nil || !strings.Contains(err.Error(), "git runner is required") {
		t.Fatalf("expected 'git runner is required' error, got %v", err)
	}
}

func TestGetDiff_RequiresRepoDir(t *testing.T) {
	ctx := context.Background()
	_, err := GetDiff(ctx, DiffDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: "",
	}, DiffRequest{})
	if err == nil || !strings.Contains(err.Error(), "repo dir is required") {
		t.Fatalf("expected 'repo dir is required' error, got %v", err)
	}
}

func TestGetDiff_WithRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init")
	runGitCmd(t, repoDir, "checkout", "-b", "main")

	// Create and commit initial file
	filePath := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("initial content\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	runGitCmd(t, repoDir, "add", "test.txt")
	runGitCmd(t, repoDir, "commit", "-m", "initial")

	// Modify file
	if err := os.WriteFile(filePath, []byte("initial content\nadded line\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	diff, err := GetDiff(ctx, DiffDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	}, DiffRequest{
		Path:   "test.txt",
		Staged: false,
	})
	if err != nil {
		t.Fatalf("GetDiff failed: %v", err)
	}

	if !diff.HasDiff {
		t.Fatalf("expected HasDiff=true")
	}
	if diff.Stats.Additions != 1 {
		t.Fatalf("expected 1 addition, got %d", diff.Stats.Additions)
	}
	if diff.RepoDir != repoDir {
		t.Fatalf("expected RepoDir=%q, got %q", repoDir, diff.RepoDir)
	}
}

func TestGetDiff_StagedChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init")
	runGitCmd(t, repoDir, "checkout", "-b", "main")

	// Create and commit initial file
	filePath := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("initial content\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	runGitCmd(t, repoDir, "add", "test.txt")
	runGitCmd(t, repoDir, "commit", "-m", "initial")

	// Modify and stage file
	if err := os.WriteFile(filePath, []byte("initial content\nstaged line\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	runGitCmd(t, repoDir, "add", "test.txt")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check unstaged diff (should be empty since we staged)
	unstaged, err := GetDiff(ctx, DiffDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	}, DiffRequest{
		Path:   "test.txt",
		Staged: false,
	})
	if err != nil {
		t.Fatalf("GetDiff (unstaged) failed: %v", err)
	}
	if unstaged.HasDiff {
		t.Fatalf("expected no unstaged diff")
	}

	// Check staged diff
	staged, err := GetDiff(ctx, DiffDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: repoDir,
	}, DiffRequest{
		Path:   "test.txt",
		Staged: true,
	})
	if err != nil {
		t.Fatalf("GetDiff (staged) failed: %v", err)
	}
	if !staged.HasDiff {
		t.Fatalf("expected staged diff")
	}
	if staged.Stats.Additions != 1 {
		t.Fatalf("expected 1 staged addition, got %d", staged.Stats.Additions)
	}
}

// runGitCmd is an alias for RunGitCommand for backward compatibility.
// New tests should use RunGitCommand directly.
func runGitCmd(t *testing.T, dir string, args ...string) {
	RunGitCommand(t, dir, args...)
}
