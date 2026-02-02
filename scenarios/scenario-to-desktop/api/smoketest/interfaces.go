// Package smoketest provides smoke testing services for desktop applications.
// This domain handles running smoke tests on built applications to verify
// they start correctly and can report telemetry.
package smoketest

import (
	"context"
	"os"
	"time"
)

// Service orchestrates smoke test operations.
type Service interface {
	// PerformSmokeTest runs a smoke test on a built application.
	PerformSmokeTest(ctx context.Context, smokeTestID, scenarioName, artifactPath, platform string)

	// CurrentPlatform returns the current platform identifier.
	CurrentPlatform() string
}

// Store manages smoke test status tracking.
type Store interface {
	// Save inserts or replaces a smoke test status.
	Save(status *Status)

	// Get returns the status for the given smoke test ID if it exists.
	Get(id string) (*Status, bool)

	// Update executes fn while holding a write lock on the requested smoke test.
	// It returns false when the smoke test ID is unknown.
	Update(id string, fn func(status *Status)) bool
}

// CancelManager manages cancellation of running smoke tests.
type CancelManager interface {
	// SetCancel registers a cancellation function for a smoke test.
	SetCancel(id string, cancel context.CancelFunc)

	// TakeCancel retrieves and removes the cancellation function for a smoke test.
	TakeCancel(id string) context.CancelFunc

	// Clear removes the cancellation function without calling it.
	Clear(id string)
}

// PackageFinder locates built packages in dist directories.
type PackageFinder interface {
	// FindBuiltPackage finds the built package file for a specific platform.
	FindBuiltPackage(distPath, platform string) (string, error)
}

// TelemetryIngestor ingests telemetry events from smoke tests.
type TelemetryIngestor interface {
	// IngestEvents ingests telemetry events from a smoke test.
	IngestEvents(scenarioName, instanceID, source string, events []map[string]interface{}) (string, int, error)
}

// Logger provides structured logging.
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// ExecutionResult contains the detailed output from command execution.
type ExecutionResult struct {
	// Stdout contains the standard output from the command.
	Stdout string

	// Stderr contains the standard error output from the command.
	Stderr string

	// Combined contains stdout and stderr interleaved (for backward compatibility).
	Combined string

	// ExitCode is the exit code from the command (-1 if process didn't start).
	ExitCode int

	// Duration is how long the command ran.
	Duration time.Duration

	// Truncated indicates if output was truncated due to size limits.
	Truncated bool

	// TruncatedBytes is the number of bytes that were truncated (0 if none).
	TruncatedBytes int
}

// ProcessExecutor abstracts command execution for testability.
type ProcessExecutor interface {
	// Execute runs a command and returns combined stdout/stderr output.
	// Deprecated: Use ExecuteWithResult for access to separated stdout/stderr.
	Execute(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (string, error)

	// ExecuteWithResult runs a command and returns detailed execution result.
	ExecuteWithResult(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (*ExecutionResult, error)

	// LookPath searches for an executable in the system PATH.
	LookPath(name string) (string, error)
}

// PlatformResolver handles platform-specific command resolution.
type PlatformResolver interface {
	// CurrentPlatform returns the current platform identifier (linux, mac, win).
	CurrentPlatform() string

	// ResolveCommand determines the command, args, and display string for running a smoke test.
	ResolveCommand(platform, artifactPath string) (cmd string, args []string, display string, err error)

	// RequiresHeadlessWrapper checks if a headless wrapper (xvfb-run) is needed and available.
	// Returns the wrapper command/args if needed and available, or an error if needed but unavailable.
	RequiresHeadlessWrapper() (needed bool, wrapperCmd string, wrapperArgs []string, err error)
}

// TelemetryPathResolver discovers telemetry file paths.
type TelemetryPathResolver interface {
	// ExtractFromOutput attempts to extract the telemetry path from smoke test output.
	ExtractFromOutput(output string) string

	// ResolveFromArtifact attempts to resolve the telemetry path based on platform and artifact.
	ResolveFromArtifact(platform, artifactPath, scenarioName string) string

	// ReadTelemetryEvents reads telemetry events from the given path.
	ReadTelemetryEvents(path string, limit int) ([]map[string]interface{}, error)
}

// OutputParser interprets smoke test output markers.
type OutputParser interface {
	// ParseResult analyzes smoke test output and returns the result.
	ParseResult(output string) OutputResult

	// ValidateSequence performs detailed validation of the smoke test output sequence.
	// It checks that markers appear in the correct order: init -> ready -> passed -> exit.
	ValidateSequence(output string) SequenceValidation

	// ExtractAppError parses SMOKE_TEST_ERROR markers from output.
	// Returns nil if no app error marker is found.
	ExtractAppError(output string) *AppError

	// ExtractLastLifecycleState returns the last lifecycle marker reached.
	// Returns empty string if no lifecycle markers are found.
	// Possible values: "init", "ready", "result", "exit"
	ExtractLastLifecycleState(output string) string
}

// TelemetryChainExecutor orchestrates the telemetry collection fallback chain.
type TelemetryChainExecutor interface {
	// Execute runs the telemetry collection chain and returns detailed results.
	Execute(ctx context.Context, params TelemetryChainParams) TelemetryResult
}

// EnvironmentReader abstracts environment variable access for testability.
type EnvironmentReader interface {
	// GetEnv retrieves the value of an environment variable.
	GetEnv(key string) string

	// UserHomeDir returns the current user's home directory.
	UserHomeDir() (string, error)
}

// FileSystem abstracts file operations for testability.
type FileSystem interface {
	// Stat returns file info for the given path.
	Stat(path string) (os.FileInfo, error)

	// ReadDir reads the contents of a directory.
	ReadDir(path string) ([]os.DirEntry, error)

	// Open opens a file for reading.
	Open(path string) (*os.File, error)

	// Chmod changes the mode of the named file.
	Chmod(path string, mode os.FileMode) error

	// ReadFile reads a file and returns its contents.
	ReadFile(path string) ([]byte, error)
}

// PrerequisiteCheckerI abstracts prerequisite checking for testability.
type PrerequisiteCheckerI interface {
	// CheckAll runs all prerequisite checks for a smoke test.
	CheckAll(artifactPath, platform string, telemetryPort int) []PrerequisiteResult

	// HasFatalFailure returns true if any prerequisite check failed fatally.
	HasFatalFailure(results []PrerequisiteResult) bool
}

// Clock abstracts time operations for testability.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// After waits for the duration to elapse and then sends the current time on the returned channel.
	After(d time.Duration) <-chan time.Time
}

// RealClock is a Clock implementation that uses the standard time package.
type RealClock struct{}

// Now returns the current time.
func (RealClock) Now() time.Time {
	return time.Now()
}

// After waits for the duration to elapse and then sends the current time.
func (RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
