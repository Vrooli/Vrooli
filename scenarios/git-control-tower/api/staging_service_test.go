package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// [REQ:GCT-OT-P0-004] Stage/unstage operations

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
	_, err := StageFiles(ctx, StagingDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: "",
	}, StageRequest{Paths: []string{"file.txt"}})
	if err == nil || !strings.Contains(err.Error(), "repo dir is required") {
		t.Fatalf("expected 'repo dir is required' error, got %v", err)
	}
}

func TestStageFiles_EmptyPaths(t *testing.T) {
	ctx := context.Background()
	result, err := StageFiles(ctx, StagingDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: "/tmp",
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
	_, err := UnstageFiles(ctx, StagingDeps{
		Git:     &ExecGitRunner{GitPath: "git"},
		RepoDir: "",
	}, UnstageRequest{Paths: []string{"file.txt"}})
	if err == nil || !strings.Contains(err.Error(), "repo dir is required") {
		t.Fatalf("expected 'repo dir is required' error, got %v", err)
	}
}

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

func runGitStaging(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(out))
	}
}
