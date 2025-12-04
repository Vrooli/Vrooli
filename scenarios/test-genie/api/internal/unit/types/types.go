// Package types provides shared types for unit test runners.
package types

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
)

// ObservationType categorizes the kind of observation.
type ObservationType int

const (
	ObservationSection ObservationType = iota
	ObservationSuccess
	ObservationWarning
	ObservationError
	ObservationInfo
	ObservationSkip
)

// Observation represents a single validation observation with formatting hints.
type Observation struct {
	Type    ObservationType
	Icon    string
	Message string
}

// NewSectionObservation creates a section header observation.
func NewSectionObservation(icon, message string) Observation {
	return Observation{Type: ObservationSection, Icon: icon, Message: message}
}

// NewSuccessObservation creates a success observation.
func NewSuccessObservation(message string) Observation {
	return Observation{Type: ObservationSuccess, Message: message}
}

// NewWarningObservation creates a warning observation.
func NewWarningObservation(message string) Observation {
	return Observation{Type: ObservationWarning, Message: message}
}

// NewErrorObservation creates an error observation.
func NewErrorObservation(message string) Observation {
	return Observation{Type: ObservationError, Message: message}
}

// NewInfoObservation creates an informational observation.
func NewInfoObservation(message string) Observation {
	return Observation{Type: ObservationInfo, Message: message}
}

// NewSkipObservation creates a skip observation.
func NewSkipObservation(message string) Observation {
	return Observation{Type: ObservationSkip, Message: message}
}

// FailureClass categorizes the type of unit test failure.
type FailureClass string

const (
	// FailureClassNone indicates no failure occurred.
	FailureClassNone FailureClass = ""
	// FailureClassMisconfiguration indicates the scenario is misconfigured.
	FailureClassMisconfiguration FailureClass = "misconfiguration"
	// FailureClassMissingDependency indicates a required tool is missing.
	FailureClassMissingDependency FailureClass = "missing_dependency"
	// FailureClassTestFailure indicates unit tests failed.
	FailureClassTestFailure FailureClass = "test_failure"
	// FailureClassSystem indicates a system-level error.
	FailureClassSystem FailureClass = "system"
)

// Result represents the outcome of running a language's unit tests.
type Result struct {
	// Success indicates whether tests passed.
	Success bool

	// Error contains the test error, if any.
	Error error

	// FailureClass categorizes the type of failure.
	FailureClass FailureClass

	// Remediation provides guidance on how to fix the issue.
	Remediation string

	// Observations contains detailed test observations.
	Observations []Observation

	// Coverage contains the test coverage percentage, if available.
	Coverage string

	// Skipped indicates the runner was skipped (language not detected).
	Skipped bool

	// SkipReason explains why the runner was skipped.
	SkipReason string
}

// OK creates a successful result.
func OK() Result {
	return Result{Success: true}
}

// OKWithCoverage creates a successful result with coverage information.
func OKWithCoverage(coverage string) Result {
	return Result{Success: true, Coverage: coverage}
}

// Skip creates a skipped result.
func Skip(reason string) Result {
	return Result{Success: true, Skipped: true, SkipReason: reason}
}

// Fail creates a failure result.
func Fail(err error, class FailureClass, remediation string) Result {
	return Result{
		Success:      false,
		Error:        err,
		FailureClass: class,
		Remediation:  remediation,
	}
}

// FailMisconfiguration creates a misconfiguration failure.
func FailMisconfiguration(err error, remediation string) Result {
	return Fail(err, FailureClassMisconfiguration, remediation)
}

// FailMissingDependency creates a missing dependency failure.
func FailMissingDependency(err error, remediation string) Result {
	return Fail(err, FailureClassMissingDependency, remediation)
}

// FailTestFailure creates a test failure result.
func FailTestFailure(err error, remediation string) Result {
	return Fail(err, FailureClassTestFailure, remediation)
}

// FailSystem creates a system-level failure.
func FailSystem(err error, remediation string) Result {
	return Fail(err, FailureClassSystem, remediation)
}

// WithObservations adds observations to the result.
func (r Result) WithObservations(obs ...Observation) Result {
	r.Observations = append(r.Observations, obs...)
	return r
}

// LanguageRunner executes unit tests for a specific language.
type LanguageRunner interface {
	// Name returns the runner's language name (e.g., "go", "node", "python").
	Name() string

	// Detect returns true if this runner should execute for the scenario.
	Detect() bool

	// Run executes the unit tests and returns the result.
	Run(ctx context.Context) Result
}

// CommandExecutor abstracts command execution for testing.
type CommandExecutor interface {
	// Run executes a command, streaming output to the writer.
	Run(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error

	// Capture executes a command and captures its output.
	Capture(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error)

	// LookPath checks if a command is available.
	LookPath(name string) (string, error)
}

// DefaultExecutor is the default command executor using os/exec.
type DefaultExecutor struct{}

// NewDefaultExecutor creates a new default command executor.
func NewDefaultExecutor() *DefaultExecutor {
	return &DefaultExecutor{}
}

// Run executes a command, streaming output to the writer.
func (e *DefaultExecutor) Run(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if logWriter == nil {
		logWriter = io.Discard
	}
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	return cmd.Run()
}

// Capture executes a command and captures its output.
func (e *DefaultExecutor) Capture(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var output bytes.Buffer
	if logWriter != nil {
		cmd.Stdout = io.MultiWriter(logWriter, &output)
		cmd.Stderr = logWriter
	} else {
		cmd.Stdout = &output
		cmd.Stderr = io.Discard
	}
	err := cmd.Run()
	return output.String(), err
}

// LookPath checks if a command is available.
func (e *DefaultExecutor) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// EnsureCommand checks if a command is available and returns an error if not.
func EnsureCommand(executor CommandExecutor, name string) error {
	if _, err := executor.LookPath(name); err != nil {
		return fmt.Errorf("required command '%s' is not available: %w", name, err)
	}
	return nil
}
