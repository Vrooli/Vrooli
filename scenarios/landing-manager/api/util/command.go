// Package util provides utility functions and interfaces for the landing-manager API.
package util

import (
	"context"
	"os/exec"
)

// CommandResult represents the result of a command execution
type CommandResult struct {
	Output   string
	ExitCode int
	Err      error
}

// CommandExecutor is a seam for executing external commands.
// This interface allows tests to substitute command execution with mock implementations,
// making it possible to test code that shells out to CLI tools without requiring
// the actual tools or running scenarios.
//
// Implementations:
//   - RealCommandExecutor: Executes actual commands via os/exec (production)
//   - MockCommandExecutor: Returns configurable responses (testing)
type CommandExecutor interface {
	// Execute runs a command with the given name and arguments.
	// Returns the combined stdout/stderr output and any error.
	Execute(name string, args ...string) CommandResult

	// ExecuteWithContext runs a command with context for timeout/cancellation support.
	ExecuteWithContext(ctx context.Context, name string, args ...string) CommandResult
}

// RealCommandExecutor implements CommandExecutor using os/exec.
// This is the production implementation that executes actual system commands.
type RealCommandExecutor struct{}

// NewRealCommandExecutor creates a new RealCommandExecutor
func NewRealCommandExecutor() *RealCommandExecutor {
	return &RealCommandExecutor{}
}

// Execute runs a command and returns the combined output
func (r *RealCommandExecutor) Execute(name string, args ...string) CommandResult {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return CommandResult{
		Output:   string(output),
		ExitCode: exitCode,
		Err:      err,
	}
}

// ExecuteWithContext runs a command with context support
func (r *RealCommandExecutor) ExecuteWithContext(ctx context.Context, name string, args ...string) CommandResult {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return CommandResult{
		Output:   string(output),
		ExitCode: exitCode,
		Err:      err,
	}
}

// MockCommandExecutor implements CommandExecutor for testing.
// Configure responses using the Responses map keyed by command name,
// or set DefaultResponse for a catch-all response.
type MockCommandExecutor struct {
	// Responses maps command names to their mock responses
	Responses map[string]CommandResult
	// DefaultResponse is returned when no specific response is configured
	DefaultResponse CommandResult
	// Calls records all command invocations for verification
	Calls []CommandCall
}

// CommandCall records a single command invocation
type CommandCall struct {
	Name string
	Args []string
}

// NewMockCommandExecutor creates a new MockCommandExecutor
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		Responses: make(map[string]CommandResult),
		Calls:     make([]CommandCall, 0),
	}
}

// Execute returns a mock response based on the command name
func (m *MockCommandExecutor) Execute(name string, args ...string) CommandResult {
	m.Calls = append(m.Calls, CommandCall{Name: name, Args: args})

	if response, ok := m.Responses[name]; ok {
		return response
	}
	return m.DefaultResponse
}

// ExecuteWithContext returns a mock response (context is ignored in mock)
func (m *MockCommandExecutor) ExecuteWithContext(ctx context.Context, name string, args ...string) CommandResult {
	return m.Execute(name, args...)
}

// SetResponse configures a mock response for a specific command
func (m *MockCommandExecutor) SetResponse(name string, output string, exitCode int, err error) {
	m.Responses[name] = CommandResult{
		Output:   output,
		ExitCode: exitCode,
		Err:      err,
	}
}

// Reset clears all recorded calls and responses
func (m *MockCommandExecutor) Reset() {
	m.Calls = make([]CommandCall, 0)
	m.Responses = make(map[string]CommandResult)
}

// DefaultCommandExecutor is the package-level executor used when none is explicitly provided.
// Tests can replace this with a MockCommandExecutor to avoid shelling out.
var DefaultCommandExecutor CommandExecutor = NewRealCommandExecutor()

// RunCommand executes a command using the DefaultCommandExecutor.
// This is a convenience function for code that doesn't need to inject an executor.
func RunCommand(name string, args ...string) CommandResult {
	return DefaultCommandExecutor.Execute(name, args...)
}

// RunCommandWithContext executes a command with context using the DefaultCommandExecutor.
func RunCommandWithContext(ctx context.Context, name string, args ...string) CommandResult {
	return DefaultCommandExecutor.ExecuteWithContext(ctx, name, args...)
}
