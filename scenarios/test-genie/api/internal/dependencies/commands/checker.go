package commands

import (
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

	"test-genie/internal/structure/types"
)

// BaselineCommands defines commands required for local phase execution.
var BaselineCommands = []string{"bash", "curl", "jq"}

// LookupFunc is the function signature for command lookup.
// This allows dependency injection for testing.
type LookupFunc func(name string) (string, error)

// DefaultLookup uses exec.LookPath to find commands.
var DefaultLookup LookupFunc = exec.LookPath

// Checker validates that required commands are available in PATH.
type Checker interface {
	// Check verifies a command is available and returns its path or an error.
	Check(name string) (path string, err error)

	// CheckAll verifies multiple commands with reasons and returns a result.
	CheckAll(commands []CommandRequirement) Result
}

// CommandRequirement describes a command and why it's needed.
type CommandRequirement struct {
	Name   string // Command name (e.g., "bash", "go")
	Reason string // Why it's required
}

// Result represents the outcome of command checking.
type Result struct {
	// Success indicates whether all commands are available.
	Success bool

	// Error contains the validation error, if any.
	Error error

	// FailureClass categorizes the type of failure.
	FailureClass types.FailureClass

	// Remediation provides guidance on how to fix the issue.
	Remediation string

	// Observations contains detailed validation observations.
	Observations []types.Observation

	// Verified lists commands that were successfully found.
	Verified []string

	// Missing lists commands that could not be found.
	Missing []MissingCommand
}

// MissingCommand describes a command that wasn't found.
type MissingCommand struct {
	Name   string
	Reason string
}

// checker is the default implementation of Checker.
type checker struct {
	lookup    LookupFunc
	logWriter io.Writer
}

// New creates a new command checker.
func New(logWriter io.Writer, opts ...Option) Checker {
	c := &checker{
		lookup:    DefaultLookup,
		logWriter: logWriter,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Option configures a checker.
type Option func(*checker)

// WithLookup sets a custom lookup function (for testing).
func WithLookup(fn LookupFunc) Option {
	return func(c *checker) {
		c.lookup = fn
	}
}

// Check implements Checker.
func (c *checker) Check(name string) (string, error) {
	path, err := c.lookup(name)
	if err != nil {
		return "", fmt.Errorf("command '%s' not found in PATH: %w", name, err)
	}
	return path, nil
}

// CheckAll implements Checker.
func (c *checker) CheckAll(commands []CommandRequirement) Result {
	var observations []types.Observation
	var verified []string
	var missing []MissingCommand

	for _, cmd := range commands {
		path, err := c.Check(cmd.Name)
		if err != nil {
			c.logWarn("command missing: %s (%s)", cmd.Name, cmd.Reason)
			missing = append(missing, MissingCommand{
				Name:   cmd.Name,
				Reason: cmd.Reason,
			})
			continue
		}
		c.logStep("command verified: %s at %s", cmd.Name, path)
		verified = append(verified, cmd.Name)
		observations = append(observations, types.NewSuccessObservation(
			fmt.Sprintf("command available: %s", cmd.Name),
		))
	}

	if len(missing) > 0 {
		return Result{
			Success:      false,
			Error:        fmt.Errorf("missing required commands: %s", formatMissing(missing)),
			FailureClass: "missing_dependency",
			Remediation:  formatRemediation(missing),
			Observations: observations,
			Verified:     verified,
			Missing:      missing,
		}
	}

	return Result{
		Success:      true,
		Observations: observations,
		Verified:     verified,
	}
}

// formatMissing creates a summary of missing commands.
func formatMissing(missing []MissingCommand) string {
	var parts []string
	for _, m := range missing {
		parts = append(parts, fmt.Sprintf("%s (%s)", m.Name, m.Reason))
	}
	sort.Strings(parts)
	return strings.Join(parts, "; ")
}

// formatRemediation creates installation instructions.
func formatRemediation(missing []MissingCommand) string {
	names := make([]string, 0, len(missing))
	seen := make(map[string]struct{})
	for _, m := range missing {
		if _, exists := seen[m.Name]; exists {
			continue
		}
		seen[m.Name] = struct{}{}
		names = append(names, m.Name)
	}
	sort.Strings(names)
	return fmt.Sprintf("Install or expose these commands to PATH: %s", strings.Join(names, ", "))
}

// logStep writes a step message to the log.
func (c *checker) logStep(format string, args ...interface{}) {
	if c.logWriter == nil {
		return
	}
	fmt.Fprintf(c.logWriter, format+"\n", args...)
}

// logWarn writes a warning message to the log.
func (c *checker) logWarn(format string, args ...interface{}) {
	if c.logWriter == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(c.logWriter, "[WARNING] %s\n", msg)
}

// BaselineRequirements returns the baseline command requirements.
func BaselineRequirements() []CommandRequirement {
	return []CommandRequirement{
		{Name: "bash", Reason: "baseline tool required to run local phases"},
		{Name: "curl", Reason: "baseline tool required to run local phases"},
		{Name: "jq", Reason: "baseline tool required to run local phases"},
	}
}
