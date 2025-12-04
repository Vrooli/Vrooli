package golang

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"test-genie/internal/unit/types"
)

// Runner executes Go unit tests.
type Runner struct {
	scenarioDir string
	executor    types.CommandExecutor
	logWriter   io.Writer
}

// Config holds configuration for the Go runner.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// Executor is the command executor to use.
	Executor types.CommandExecutor

	// LogWriter is where to write log output.
	LogWriter io.Writer
}

// New creates a new Go test runner.
func New(cfg Config) *Runner {
	executor := cfg.Executor
	if executor == nil {
		executor = types.NewDefaultExecutor()
	}
	logWriter := cfg.LogWriter
	if logWriter == nil {
		logWriter = io.Discard
	}
	return &Runner{
		scenarioDir: cfg.ScenarioDir,
		executor:    executor,
		logWriter:   logWriter,
	}
}

// Name returns the runner's language name.
func (r *Runner) Name() string {
	return "go"
}

// Detect returns true if a Go project exists in the scenario.
func (r *Runner) Detect() bool {
	goModPath := filepath.Join(r.scenarioDir, "api", "go.mod")
	info, err := os.Stat(goModPath)
	return err == nil && !info.IsDir()
}

// Run executes Go unit tests and returns the result.
func (r *Runner) Run(ctx context.Context) types.Result {
	if err := ctx.Err(); err != nil {
		return types.FailSystem(err, "Context cancelled")
	}

	apiDir := filepath.Join(r.scenarioDir, "api")

	// Verify api/ directory exists
	if err := ensureDir(apiDir); err != nil {
		return types.FailMisconfiguration(err, "Ensure the api/ directory exists so Go unit tests can run.")
	}

	// Check Go is available
	if err := types.EnsureCommand(r.executor, "go"); err != nil {
		return types.FailMissingDependency(err, "Install the Go toolchain to execute API unit tests.")
	}

	logStep(r.logWriter, "executing go test ./... inside %s", apiDir)

	// Run go test
	if err := r.executor.Run(ctx, apiDir, r.logWriter, "go", "test", "./..."); err != nil {
		return types.FailTestFailure(
			fmt.Errorf("go test ./... failed: %w", err),
			"Fix failing Go tests under api/ before re-running the suite.",
		)
	}

	return types.OK().WithObservations(types.NewSuccessObservation("go test ./... passed"))
}

// ensureDir checks that a directory exists.
func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required directory missing: %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory but found file: %s", path)
	}
	return nil
}

// logStep writes a step message to the log.
func logStep(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, format+"\n", args...)
}
