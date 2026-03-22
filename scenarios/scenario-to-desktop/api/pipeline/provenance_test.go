package pipeline

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCaptureProvenance_CleanRepo(t *testing.T) {
	dir := initTestRepo(t)

	p := CaptureProvenance(dir, "1.2.3")

	if p.GitCommitHash == "" {
		t.Fatal("expected non-empty commit hash")
	}
	if len(p.GitCommitHash) != 40 {
		t.Errorf("expected 40-char SHA, got %d chars: %s", len(p.GitCommitHash), p.GitCommitHash)
	}
	if p.GitBranch == "" {
		t.Fatal("expected non-empty branch")
	}
	if p.GitDirty {
		t.Error("expected clean repo, got dirty=true")
	}
	if p.Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", p.Version)
	}
	if p.BuiltAt.IsZero() {
		t.Error("expected non-zero BuiltAt")
	}
}

func TestCaptureProvenance_DirtyRepo(t *testing.T) {
	dir := initTestRepo(t)

	// Create an untracked file to make the repo dirty
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := CaptureProvenance(dir, "0.1.0")

	if !p.GitDirty {
		t.Error("expected dirty=true after adding untracked file")
	}
	if p.GitCommitHash == "" {
		t.Fatal("expected non-empty commit hash even when dirty")
	}
}

func TestCaptureProvenance_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	p := CaptureProvenance(dir, "0.0.0")

	// Should return partial provenance without crashing
	if p.GitCommitHash != "" {
		t.Errorf("expected empty commit hash for non-git dir, got %s", p.GitCommitHash)
	}
	if p.GitBranch != "" {
		t.Errorf("expected empty branch for non-git dir, got %s", p.GitBranch)
	}
	// Conservative default: assume dirty when git is unavailable
	if !p.GitDirty {
		t.Error("expected dirty=true when not in a git repo")
	}
	if p.Version != "0.0.0" {
		t.Errorf("expected version 0.0.0, got %s", p.Version)
	}
}

func TestCaptureProvenance_BranchName(t *testing.T) {
	dir := initTestRepo(t)

	// Create and switch to a feature branch
	run(t, dir, "git", "checkout", "-b", "feature/test-branch")

	p := CaptureProvenance(dir, "1.0.0")

	if p.GitBranch != "feature/test-branch" {
		t.Errorf("expected branch feature/test-branch, got %s", p.GitBranch)
	}
}

// initTestRepo creates a temporary git repo with one commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")

	// Create a file and commit it
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "initial commit")

	return dir
}

// run executes a command in a directory and fails the test on error.
func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}
