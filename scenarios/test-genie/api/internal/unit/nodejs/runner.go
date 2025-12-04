package nodejs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"test-genie/internal/unit/types"
)

// Runner executes Node.js unit tests.
type Runner struct {
	scenarioDir string
	executor    types.CommandExecutor
	logWriter   io.Writer
}

// Config holds configuration for the Node.js runner.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// Executor is the command executor to use.
	Executor types.CommandExecutor

	// LogWriter is where to write log output.
	LogWriter io.Writer
}

// New creates a new Node.js test runner.
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
	return "node"
}

// Detect returns true if a Node.js project exists in the scenario.
func (r *Runner) Detect() bool {
	return r.detectWorkspaceDir() != ""
}

// detectWorkspaceDir finds the Node.js workspace directory.
func (r *Runner) detectWorkspaceDir() string {
	candidates := []string{
		filepath.Join(r.scenarioDir, "ui"),
		r.scenarioDir,
	}
	for _, candidate := range candidates {
		if fileExists(filepath.Join(candidate, "package.json")) {
			return candidate
		}
	}
	return ""
}

// Run executes Node.js unit tests and returns the result.
func (r *Runner) Run(ctx context.Context) types.Result {
	if err := ctx.Err(); err != nil {
		return types.FailSystem(err, "Context cancelled")
	}

	nodeDir := r.detectWorkspaceDir()
	if nodeDir == "" {
		return types.Skip("No package.json found")
	}

	// Check Node.js is available
	if err := types.EnsureCommand(r.executor, "node"); err != nil {
		return types.FailMissingDependency(err, "Install Node.js so UI/unit suites can execute.")
	}

	// Load package.json
	manifest, err := LoadManifest(filepath.Join(nodeDir, "package.json"))
	if err != nil {
		return types.FailMisconfiguration(err, "Fix package.json so the Node workspace can be parsed.")
	}

	// Check for test script
	if manifest == nil || !manifest.HasTestScript() {
		return types.Skip("package.json exists but lacks a test script")
	}

	// Detect package manager
	packageManager := DetectPackageManager(manifest, nodeDir)
	if err := types.EnsureCommand(r.executor, packageManager); err != nil {
		return types.FailMissingDependency(err, fmt.Sprintf("Install %s to run Node test suites.", packageManager))
	}

	// Install dependencies if needed
	if !dirExists(filepath.Join(nodeDir, "node_modules")) {
		logStep(r.logWriter, "installing Node dependencies via %s", packageManager)
		if err := r.installDependencies(ctx, nodeDir, packageManager); err != nil {
			return types.FailSystem(
				fmt.Errorf("%s install failed: %w", packageManager, err),
				"Resolve dependency installation issues before re-running unit tests.",
			)
		}
	}

	// Run tests
	logStep(r.logWriter, "running Node unit tests with %s", packageManager)
	output, err := r.executor.Capture(ctx, nodeDir, r.logWriter, packageManager, "test")
	if err != nil {
		return types.FailTestFailure(
			fmt.Errorf("Node unit tests failed: %w", err),
			"Inspect the UI/unit test output above, fix failures, and rerun the suite.",
		)
	}

	// Extract coverage
	coverage := DetectCoverage(nodeDir, output)
	result := types.OKWithCoverage(coverage)

	successMsg := fmt.Sprintf("node unit tests passed via %s", packageManager)
	result = result.WithObservations(types.NewSuccessObservation(successMsg))

	if coverage != "" {
		result = result.WithObservations(types.NewInfoObservation(fmt.Sprintf("node coverage: %s%% statements", coverage)))
	}

	return result
}

// installDependencies installs Node.js dependencies using the appropriate package manager.
func (r *Runner) installDependencies(ctx context.Context, dir, manager string) error {
	switch manager {
	case "pnpm":
		return r.executor.Run(ctx, dir, r.logWriter, "pnpm", "install", "--frozen-lockfile", "--ignore-scripts")
	case "yarn":
		return r.executor.Run(ctx, dir, r.logWriter, "yarn", "install", "--frozen-lockfile")
	default:
		return r.executor.Run(ctx, dir, r.logWriter, "npm", "install")
	}
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// logStep writes a step message to the log.
func logStep(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, format+"\n", args...)
}
