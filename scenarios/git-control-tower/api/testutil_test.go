package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// --- Shared Test Utilities ---
// This file contains test helpers shared across all test files.
// Use these helpers to avoid duplication and ensure consistent test setup.

// RunGitCommand executes a git command in the given directory with proper environment.
// Use this for integration tests that require real git operations.
// For unit tests, prefer FakeGitRunner instead.
func RunGitCommand(t *testing.T, dir string, args ...string) {
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

// SetupTestRepo creates a temporary git repository for integration testing.
// Returns the path to the repo directory.
// The repo is automatically cleaned up when the test finishes.
func SetupTestRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	repoDir := t.TempDir()
	RunGitCommand(t, repoDir, "init")
	RunGitCommand(t, repoDir, "checkout", "-b", "main")
	return repoDir
}

// WriteTestFile creates a file in the given directory with the specified content.
func WriteTestFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
}

// AssertContains checks if the expected value exists in the slice.
func AssertContains(t *testing.T, values []string, expected string) {
	t.Helper()
	for _, v := range values {
		if v == expected {
			return
		}
	}
	t.Fatalf("expected %q in %v", expected, values)
}
