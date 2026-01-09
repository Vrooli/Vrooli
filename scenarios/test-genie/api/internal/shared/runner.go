package shared

import (
	"context"
	"io"
)

// BaseConfig contains configuration common to all phase runners.
type BaseConfig struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario (typically the directory name).
	ScenarioName string
}

// BaseRunner provides common functionality for phase runners.
// Embed this in your runner struct to get common options and utilities.
type BaseRunner struct {
	// LogWriter is the destination for log output.
	LogWriter io.Writer
}

// NewBaseRunner creates a new BaseRunner with default settings.
func NewBaseRunner() BaseRunner {
	return BaseRunner{
		LogWriter: io.Discard,
	}
}

// BaseOption is a function that configures a BaseRunner.
type BaseOption func(*BaseRunner)

// ApplyBaseOptions applies a list of base options to the runner.
func (r *BaseRunner) ApplyBaseOptions(opts ...BaseOption) {
	for _, opt := range opts {
		opt(r)
	}
}

// WithBaseLogger sets the log writer for the runner.
func WithBaseLogger(w io.Writer) BaseOption {
	return func(r *BaseRunner) {
		if w != nil {
			r.LogWriter = w
		}
	}
}

// CheckContext checks if the context is cancelled and returns a system failure if so.
// This is a utility for consistent context checking across all runners.
func CheckContext[TSummary any](ctx context.Context) *RunResult[TSummary] {
	if err := ctx.Err(); err != nil {
		return SystemFailure[TSummary](err)
	}
	return nil
}

// Runner is an interface for phase runners.
// This allows for generic handling of different phase runners.
type Runner[TConfig any, TSummary any] interface {
	// Run executes the phase and returns the result.
	Run(ctx context.Context) *RunResult[TSummary]
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
