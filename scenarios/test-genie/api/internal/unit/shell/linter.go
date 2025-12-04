package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"test-genie/internal/unit/types"
)

// Linter validates shell script syntax.
type Linter struct {
	scenarioDir  string
	scenarioName string
	executor     types.CommandExecutor
	logWriter    io.Writer
}

// Config holds configuration for the shell linter.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario.
	ScenarioName string

	// Executor is the command executor to use.
	Executor types.CommandExecutor

	// LogWriter is where to write log output.
	LogWriter io.Writer
}

// New creates a new shell linter.
func New(cfg Config) *Linter {
	executor := cfg.Executor
	if executor == nil {
		executor = types.NewDefaultExecutor()
	}
	logWriter := cfg.LogWriter
	if logWriter == nil {
		logWriter = io.Discard
	}
	return &Linter{
		scenarioDir:  cfg.ScenarioDir,
		scenarioName: cfg.ScenarioName,
		executor:     executor,
		logWriter:    logWriter,
	}
}

// Name returns the linter's language name.
func (l *Linter) Name() string {
	return "shell"
}

// Detect returns true if shell scripts exist to lint.
func (l *Linter) Detect() bool {
	_, err := l.discoverCLIBinary()
	return err == nil
}

// Run lints shell scripts and returns the result.
func (l *Linter) Run(ctx context.Context) types.Result {
	if err := ctx.Err(); err != nil {
		return types.FailSystem(err, "Context cancelled")
	}

	cliPath, err := l.discoverCLIBinary()
	if err != nil {
		logWarn(l.logWriter, "CLI binary not linted: %v", err)
		return types.Skip("No shell entrypoints detected")
	}

	// Check bash is available
	if err := types.EnsureCommand(l.executor, "bash"); err != nil {
		return types.FailMissingDependency(err, "Install bash so shell entrypoints can be linted.")
	}

	// Verify file exists
	if err := ensureFile(cliPath); err != nil {
		return types.FailMisconfiguration(err, "Restore the CLI binary so syntax checks can run.")
	}

	// Run bash -n
	logStep(l.logWriter, "running bash -n %s", cliPath)
	if err := l.executor.Run(ctx, "", l.logWriter, "bash", "-n", cliPath); err != nil {
		return types.FailTestFailure(
			fmt.Errorf("bash -n %s failed: %w", cliPath, err),
			fmt.Sprintf("Fix syntax errors in %s and re-run the suite.", cliPath),
		)
	}

	return types.OK().WithObservations(types.NewSuccessObservation(fmt.Sprintf("bash -n verified: %s", cliPath)))
}

// discoverCLIBinary finds the CLI binary for the scenario.
func (l *Linter) discoverCLIBinary() (string, error) {
	cliDir := filepath.Join(l.scenarioDir, "cli")
	info, err := os.Stat(cliDir)
	if err != nil {
		return "", fmt.Errorf("cli directory missing: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("cli path is not a directory: %s", cliDir)
	}

	// Try scenario-specific names first
	var candidates []string
	name := strings.TrimSpace(l.scenarioName)
	if name != "" {
		candidates = append(candidates,
			filepath.Join(cliDir, name),
			filepath.Join(cliDir, name+".sh"),
			filepath.Join(cliDir, name+".exe"),
		)
	}
	// Add fallback patterns
	candidates = append(candidates,
		filepath.Join(cliDir, "test-genie"),
		filepath.Join(cliDir, "test-genie.exe"),
	)

	for _, candidate := range candidates {
		if err := ensureExecutable(candidate); err == nil {
			return candidate, nil
		}
	}

	// Scan directory for any executable
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return "", fmt.Errorf("failed to list cli directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(cliDir, entry.Name())
		if err := ensureExecutable(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no executable CLI binary found under %s", cliDir)
}

// ensureFile checks that a file exists and is not a directory.
func ensureFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required file missing: %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but found directory: %s", path)
	}
	return nil
}

// ensureExecutable checks that a file exists and is executable.
func ensureExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required executable missing: %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected executable but found directory: %s", path)
	}
	if runtime.GOOS == "windows" {
		// Windows does not expose POSIX execute bits, so existence is enough
		return nil
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("file is not executable: %s", path)
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

// logWarn writes a warning message to the log.
func logWarn(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[WARNING] ⚠️ %s\n", msg)
}
