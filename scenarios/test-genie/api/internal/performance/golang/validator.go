package golang

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Validator benchmarks Go builds and validates them against time thresholds.
type Validator interface {
	// Benchmark runs a Go build and returns the benchmark result.
	Benchmark(ctx context.Context, maxDuration time.Duration) BenchmarkResult
}

// CommandExecutor runs shell commands. Injectable for testing.
type CommandExecutor func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error

// CommandLookup checks if a command exists. Injectable for testing.
type CommandLookup func(name string) (string, error)

// validator is the default implementation of Validator.
type validator struct {
	scenarioDir     string
	logWriter       io.Writer
	commandExecutor CommandExecutor
	commandLookup   CommandLookup
}

// Option configures a validator.
type Option func(*validator)

// New creates a new Go build benchmark validator.
func New(scenarioDir string, opts ...Option) Validator {
	v := &validator{
		scenarioDir:     scenarioDir,
		logWriter:       io.Discard,
		commandExecutor: defaultCommandExecutor,
		commandLookup:   exec.LookPath,
	}

	for _, opt := range opts {
		opt(v)
	}

	return v
}

// WithLogger sets the log writer.
func WithLogger(w io.Writer) Option {
	return func(v *validator) {
		v.logWriter = w
	}
}

// WithCommandExecutor sets a custom command executor (for testing).
func WithCommandExecutor(executor CommandExecutor) Option {
	return func(v *validator) {
		v.commandExecutor = executor
	}
}

// WithCommandLookup sets a custom command lookup (for testing).
func WithCommandLookup(lookup CommandLookup) Option {
	return func(v *validator) {
		v.commandLookup = lookup
	}
}

// Benchmark implements Validator.
func (v *validator) Benchmark(ctx context.Context, maxDuration time.Duration) BenchmarkResult {
	// Check that Go is available
	if _, err := v.commandLookup("go"); err != nil {
		logError(v.logWriter, "Go toolchain not found")
		return BenchmarkResult{
			Result: FailMissingDependency(
				fmt.Errorf("go command not found: %w", err),
				"Install the Go toolchain so API builds can be benchmarked.",
			),
		}
	}

	// Check that api/ directory exists
	apiDir := filepath.Join(v.scenarioDir, "api")
	if err := ensureDir(apiDir); err != nil {
		logError(v.logWriter, "API directory not found: %s", apiDir)
		return BenchmarkResult{
			Result: FailMisconfiguration(
				err,
				"Restore the api/ directory before running performance benchmarks.",
			),
		}
	}

	// Create temp file for build output
	tmpFile, err := os.CreateTemp("", "test-genie-perf-*")
	if err != nil {
		return BenchmarkResult{
			Result: FailSystem(
				fmt.Errorf("failed to create temp binary: %w", err),
				"Verify the filesystem is writable for performance artifacts.",
			),
		}
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Run the build
	logInfo(v.logWriter, "Building Go API binary in %s", apiDir)
	start := time.Now()
	if err := v.commandExecutor(ctx, apiDir, v.logWriter, "go", "build", "-o", tmpPath, "./..."); err != nil {
		duration := time.Since(start)
		logError(v.logWriter, "Go build failed after %s", duration.Round(time.Second))
		return BenchmarkResult{
			Result: FailSystem(
				fmt.Errorf("go build failed: %w", err),
				"Fix compilation errors before re-running the performance phase.",
			),
			Duration: duration,
		}
	}
	duration := time.Since(start)
	seconds := int(duration.Round(time.Second) / time.Second)
	logSuccess(v.logWriter, "Go build completed in %ds", seconds)

	// Check against threshold
	if duration > maxDuration {
		return BenchmarkResult{
			Result: FailSystem(
				fmt.Errorf("go build exceeded %s (took %s)", maxDuration, duration.Round(time.Second)),
				"Investigate slow dependencies or remove unnecessary modules before building.",
			).WithObservations(NewSuccessObservation(fmt.Sprintf("go build duration: %ds", seconds))),
			Duration: duration,
		}
	}

	return BenchmarkResult{
		Result:   OK().WithObservations(NewSuccessObservation(fmt.Sprintf("go build duration: %ds", seconds))),
		Duration: duration,
	}
}

// defaultCommandExecutor runs commands using os/exec.
func defaultCommandExecutor(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	return cmd.Run()
}

// ensureDir verifies that a path exists and is a directory.
func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory not found: %s", path)
		}
		return fmt.Errorf("failed to stat %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}
	return nil
}

// Logging helpers

func logInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "🔍 %s\n", msg)
}

func logSuccess(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[SUCCESS] %s\n", msg)
}

func logError(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[ERROR] %s\n", msg)
}
