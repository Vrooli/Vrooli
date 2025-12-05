package python

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/shared"
	"test-genie/internal/unit/types"
)

// Runner executes Python unit tests.
type Runner struct {
	scenarioDir string
	executor    types.CommandExecutor
	logWriter   io.Writer
	workspaces  []string
}

// Config holds configuration for the Python runner.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// Executor is the command executor to use.
	Executor types.CommandExecutor

	// LogWriter is where to write log output.
	LogWriter io.Writer
}

// New creates a new Python test runner.
func New(cfg Config) *Runner {
	executor := cfg.Executor
	if executor == nil {
		executor = types.NewDefaultExecutor()
	}
	return &Runner{
		scenarioDir: cfg.ScenarioDir,
		executor:    executor,
		logWriter:   shared.DefaultLogWriter(cfg.LogWriter),
	}
}

// Name returns the runner's language name.
func (r *Runner) Name() string {
	return "python"
}

// Detect returns true if a Python project exists in the scenario.
func (r *Runner) Detect() bool {
	r.workspaces = r.discoverWorkspaces()
	return len(r.workspaces) > 0
}

// Run executes Python unit tests and returns the result.
func (r *Runner) Run(ctx context.Context) types.Result {
	if err := ctx.Err(); err != nil {
		return types.FailSystem(err, "Context cancelled")
	}

	workspaces := r.workspaces
	if len(workspaces) == 0 {
		workspaces = r.discoverWorkspaces()
	}
	if len(workspaces) == 0 {
		return types.Skip("No Python workspace detected")
	}

	builder := types.NewResultBuilder().
		Success().
		AddSection("📂", fmt.Sprintf("python workspaces (%d)", len(workspaces)))

	// Find Python command
	pythonCmd, err := r.resolvePythonCommand()
	if err != nil {
		return types.FailMissingDependency(err, "Install python3 so scenario-specific Python suites can run.")
	}

	runnedAny := false

	for _, pythonDir := range workspaces {
		rel := r.relativePath(pythonDir)

		// Check for test files
		if !HasTestFiles(pythonDir) {
			builder.AddWarningf("%s: python workspace detected but no test_*.py files found", rel)
			continue
		}

		builder.AddInfof("python tests in %s", rel)

		// Try pytest first, fall back to unittest
		if r.supportsPytest(ctx, pythonCmd) {
			if res := r.runPytest(ctx, pythonDir, pythonCmd); !res.Success {
				return res
			} else {
				for _, obs := range res.Observations {
					builder.AddObservation(obs)
				}
			}
		} else {
			if res := r.runUnittest(ctx, pythonDir, pythonCmd); !res.Success {
				return res
			} else {
				for _, obs := range res.Observations {
					builder.AddObservation(obs)
				}
			}
		}

		runnedAny = true
	}

	if !runnedAny {
		builder.AddWarning("No runnable Python workspaces found (no test files).")
		return builder.Skip("No runnable Python workspaces").Build()
	}

	return builder.Build()
}

// resolvePythonCommand finds the available Python command.
func (r *Runner) resolvePythonCommand() (string, error) {
	candidates := []string{"python3", "python"}
	for _, cmd := range candidates {
		if err := types.EnsureCommand(r.executor, cmd); err == nil {
			return cmd, nil
		}
	}
	return "", fmt.Errorf("python runtime not available")
}

// supportsPytest checks if pytest is available.
func (r *Runner) supportsPytest(ctx context.Context, pythonCmd string) bool {
	script := `import importlib.util, sys; sys.exit(0 if importlib.util.find_spec('pytest') else 1)`
	if err := r.executor.Run(ctx, "", io.Discard, pythonCmd, "-c", script); err != nil {
		return false
	}
	return true
}

// runPytest executes tests using pytest.
func (r *Runner) runPytest(ctx context.Context, dir, pythonCmd string) types.Result {
	shared.LogStep(r.logWriter, "running pytest under %s", dir)
	if err := r.executor.Run(ctx, dir, r.logWriter, pythonCmd, "-m", "pytest", "-q"); err != nil {
		return types.FailTestFailure(
			fmt.Errorf("pytest failed: %w", err),
			"Inspect pytest output above, fix failing tests, and rerun the suite.",
		)
	}
	return types.NewResultBuilder().
		Success().
		AddSuccess("python unit tests passed via pytest").
		Build()
}

// runUnittest executes tests using unittest.
func (r *Runner) runUnittest(ctx context.Context, dir, pythonCmd string) types.Result {
	shared.LogStep(r.logWriter, "running unittest discover under %s", dir)
	if err := r.executor.Run(ctx, dir, r.logWriter, pythonCmd, "-m", "unittest", "discover"); err != nil {
		return types.FailTestFailure(
			fmt.Errorf("python -m unittest discover failed: %w", err),
			"Ensure the default unittest suites pass or install pytest for richer reporting.",
		)
	}
	return types.NewResultBuilder().
		Success().
		AddSuccess("python unit tests passed via unittest").
		Build()
}

// hasIndicators checks if common Python project indicators are present.
func hasIndicators(dir string) bool {
	indicators := []string{
		"requirements.txt",
		"pyproject.toml",
		"setup.py",
		filepath.Join("tests", "__init__.py"),
	}
	for _, indicator := range indicators {
		if fileExists(filepath.Join(dir, indicator)) {
			return true
		}
	}
	// Also check for tests directory
	if dirExists(filepath.Join(dir, "tests")) {
		return true
	}
	if HasTestFiles(dir) {
		return true
	}
	return false
}

// HasTestFiles checks if Python test files exist in the directory.
func HasTestFiles(dir string) bool {
	found := false
	errStopWalk := errors.New("stop-walk")
	skipDirs := map[string]struct{}{
		".git":         {},
		"node_modules": {},
		"dist":         {},
		"build":        {},
		".next":        {},
		"coverage":     {},
		"__pycache__":  {},
		".venv":        {},
		"venv":         {},
	}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || found {
			return walkErr
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".py") && (strings.HasPrefix(d.Name(), "test_") || strings.HasSuffix(d.Name(), "_test.py")) {
			found = true
			return errStopWalk
		}
		return nil
	})
	if err != nil && !errors.Is(err, errStopWalk) {
		return false
	}
	return found
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// discoverWorkspaces finds all Python workspaces that have indicators.
func (r *Runner) discoverWorkspaces() []string {
	var workspaces []string
	seen := map[string]struct{}{}
	skipDirs := map[string]struct{}{
		".git":              {},
		"node_modules":      {},
		"dist":              {},
		"build":             {},
		"coverage":          {},
		".next":             {},
		".pnpm-store":       {},
		"playwright-report": {},
		"test-results":      {},
		"__pycache__":       {},
		".venv":             {},
		"venv":              {},
	}

	filepath.WalkDir(r.scenarioDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			if _, ok := seen[path]; ok {
				return nil
			}
			if hasIndicators(path) {
				seen[path] = struct{}{}
				workspaces = append(workspaces, path)
			}
			return nil
		}

		return nil
	})

	return workspaces
}

// relativePath returns a scenario-relative path for display.
func (r *Runner) relativePath(path string) string {
	rel, err := filepath.Rel(r.scenarioDir, path)
	if err != nil || rel == "." {
		return "."
	}
	return rel
}
