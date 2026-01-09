// Package diff provides diff generation and patch application for sandboxes.
//
// # Seam: CommandRunner Interface
//
// This file defines the CommandRunner interface, which is a critical architectural seam
// for testability. All external command execution (git, diff, patch) goes through this
// interface, allowing tests to inject mock implementations that don't touch the real
// filesystem or git repositories.
//
// # Why This Seam Exists
//
// The diff package shells out to several external commands:
//   - git (rev-parse, apply, add, commit, status, diff)
//   - diff (for generating unified diffs)
//   - patch (for applying diffs in non-git directories)
//
// Without this seam, tests would either:
//   1. Touch the real Vrooli git repository (DANGEROUS - could corrupt project)
//   2. Create and destroy real git repos in temp dirs (slow, flaky)
//   3. Skip testing these code paths entirely (inadequate coverage)
//
// With this seam, tests can inject a mock that returns predetermined outputs,
// enabling fast, deterministic, and safe testing.
//
// # Usage in Production
//
// Use DefaultCommandRunner() which wraps exec.CommandContext:
//
//	gen := diff.NewGeneratorWithRunner(diff.DefaultCommandRunner())
//	patcher := diff.NewPatcherWithRunner(diff.DefaultCommandRunner())
//
// # Usage in Tests
//
//	runner := &MockCommandRunner{
//	    Responses: map[string]CommandResult{
//	        "git rev-parse HEAD": {Stdout: "abc123\n"},
//	    },
//	}
//	gen := diff.NewGeneratorWithRunner(runner)
//
// # Test Safety Guarantees
//
// When using MockCommandRunner in tests:
//   - NO external processes are spawned
//   - NO filesystem modifications occur (except via test-controlled temp dirs)
//   - NO git operations affect the real repository
//   - Tests are deterministic and fast
package diff

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommandResult represents the output of a command execution.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// CommandRunner is the interface for executing external commands.
// This is the primary seam for test isolation in the diff package.
//
// Implementations:
//   - ExecCommandRunner: Production implementation using exec.CommandContext
//   - MockCommandRunner: Test implementation with configurable responses
type CommandRunner interface {
	// Run executes a command and returns the result.
	// The args slice contains the command followed by its arguments.
	// The dir parameter specifies the working directory (empty = current dir).
	// The stdin parameter provides input to the command (can be empty).
	Run(ctx context.Context, dir string, stdin string, args ...string) CommandResult
}

// ExecCommandRunner is the production implementation of CommandRunner.
// It wraps exec.CommandContext to run real external commands.
type ExecCommandRunner struct{}

// DefaultCommandRunner returns the production command runner.
func DefaultCommandRunner() CommandRunner {
	return &ExecCommandRunner{}
}

// Run executes a command using exec.CommandContext.
func (r *ExecCommandRunner) Run(ctx context.Context, dir string, stdin string, args ...string) CommandResult {
	if len(args) == 0 {
		return CommandResult{Err: fmt.Errorf("no command specified")}
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if dir != "" {
		cmd.Dir = dir
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
	}

	return result
}

// MockCommandRunner is a test implementation of CommandRunner.
// It returns predetermined responses based on command patterns.
//
// Example usage in tests:
//
//	mock := &MockCommandRunner{}
//	mock.AddResponse("git", "rev-parse", "HEAD").Returns("abc123\n", "", 0)
//	mock.AddResponse("git", "status").Returns("", "", 0)
type MockCommandRunner struct {
	// Responses maps command patterns to their results.
	// The key is a space-joined string of the command and arguments.
	// Partial matching is supported: "git rev-parse" matches "git rev-parse HEAD".
	Responses map[string]CommandResult

	// DefaultResponse is returned when no matching pattern is found.
	// If nil, an error is returned for unmatched commands.
	DefaultResponse *CommandResult

	// Calls records all commands that were executed.
	// This can be used to verify expected commands were called.
	Calls []MockCall

	// FailOnUnmatchedCommands controls behavior for unmatched commands.
	// If true (default), unmatched commands return an error.
	// If false, DefaultResponse is used (if set) or empty success.
	FailOnUnmatchedCommands bool
}

// MockCall records a single command execution for verification.
type MockCall struct {
	Dir   string
	Stdin string
	Args  []string
}

// NewMockCommandRunner creates a new mock runner for tests.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		Responses:               make(map[string]CommandResult),
		Calls:                   []MockCall{},
		FailOnUnmatchedCommands: true,
	}
}

// AddResponse registers a response for a command pattern.
// The pattern is matched against the space-joined command and arguments.
// Earlier (more specific) patterns take precedence over later ones.
//
// Example:
//
//	mock.AddResponse("git rev-parse HEAD", CommandResult{Stdout: "abc123\n"})
//	mock.AddResponse("git rev-parse", CommandResult{Stdout: "other\n"}) // fallback
func (m *MockCommandRunner) AddResponse(pattern string, result CommandResult) *MockCommandRunner {
	m.Responses[pattern] = result
	return m
}

// Run looks up the command in the response map and returns the matching result.
func (m *MockCommandRunner) Run(ctx context.Context, dir string, stdin string, args ...string) CommandResult {
	// Record the call
	m.Calls = append(m.Calls, MockCall{Dir: dir, Stdin: stdin, Args: args})

	if len(args) == 0 {
		return CommandResult{Err: fmt.Errorf("no command specified")}
	}

	// Build the full command string for matching
	fullCmd := strings.Join(args, " ")

	// Try exact match first
	if result, ok := m.Responses[fullCmd]; ok {
		return result
	}

	// Try prefix matches (most specific first - longer patterns)
	var matchedPattern string
	var matchedResult CommandResult
	for pattern, result := range m.Responses {
		if strings.HasPrefix(fullCmd, pattern) {
			// Take the longest matching pattern
			if len(pattern) > len(matchedPattern) {
				matchedPattern = pattern
				matchedResult = result
			}
		}
	}

	if matchedPattern != "" {
		return matchedResult
	}

	// No match found
	if m.DefaultResponse != nil {
		return *m.DefaultResponse
	}

	if m.FailOnUnmatchedCommands {
		return CommandResult{
			Err:      fmt.Errorf("MockCommandRunner: no response configured for command: %s", fullCmd),
			ExitCode: 1,
		}
	}

	// Return empty success
	return CommandResult{}
}

// WasCalled returns true if a command matching the pattern was called.
func (m *MockCommandRunner) WasCalled(pattern string) bool {
	for _, call := range m.Calls {
		fullCmd := strings.Join(call.Args, " ")
		if strings.HasPrefix(fullCmd, pattern) {
			return true
		}
	}
	return false
}

// CallCount returns the number of times a command matching the pattern was called.
func (m *MockCommandRunner) CallCount(pattern string) int {
	count := 0
	for _, call := range m.Calls {
		fullCmd := strings.Join(call.Args, " ")
		if strings.HasPrefix(fullCmd, pattern) {
			count++
		}
	}
	return count
}

// Reset clears all recorded calls (but keeps configured responses).
func (m *MockCommandRunner) Reset() {
	m.Calls = []MockCall{}
}

// AssertCalled verifies that a command was called, returning an error if not.
// This is useful in test assertions.
func (m *MockCommandRunner) AssertCalled(pattern string) error {
	if !m.WasCalled(pattern) {
		return fmt.Errorf("expected command '%s' to be called, but it wasn't. Calls: %v", pattern, m.Calls)
	}
	return nil
}

// AssertNotCalled verifies that a command was NOT called, returning an error if it was.
func (m *MockCommandRunner) AssertNotCalled(pattern string) error {
	if m.WasCalled(pattern) {
		return fmt.Errorf("expected command '%s' NOT to be called, but it was", pattern)
	}
	return nil
}

// NoOpCommandRunner always returns success with empty output.
// Useful for tests that need to suppress command execution without specific expectations.
type NoOpCommandRunner struct {
	Calls []MockCall
}

// Run records the call but returns empty success.
func (r *NoOpCommandRunner) Run(ctx context.Context, dir string, stdin string, args ...string) CommandResult {
	r.Calls = append(r.Calls, MockCall{Dir: dir, Stdin: stdin, Args: args})
	return CommandResult{}
}
