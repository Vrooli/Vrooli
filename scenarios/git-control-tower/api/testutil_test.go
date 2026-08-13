package main

import (
	"testing"

	"git-control-tower/internal/testutil/fixtures"
	"github.com/vrooli/api-core/apihttptest"
)

// --- Shared Test Utilities ---
// This file contains test helpers shared across all test files.
// Use these helpers to avoid duplication and ensure consistent test setup.

// RunGitCommand executes a git command in the given directory with proper environment.
// Use this for integration tests that require real git operations.
// For unit tests, prefer FakeGitRunner instead.
func RunGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	fixtures.RunGitCommand(t, dir, args...)
}

// SetupTestRepo creates a temporary git repository for integration testing.
// Returns the path to the repo directory.
// The repo is automatically cleaned up when the test finishes.
func SetupTestRepo(t *testing.T) string {
	t.Helper()
	return fixtures.SetupGitRepo(t)
}

// WriteTestFile creates a file in the given directory with the specified content.
func WriteTestFile(t *testing.T, path string, contents string) {
	t.Helper()
	fixtures.WriteFile(t, path, contents)
}

// AssertContains checks if the expected value exists in the slice.
func AssertContains(t *testing.T, values []string, expected string) {
	t.Helper()
	apihttptest.ContainsString(t, values, expected)
}
