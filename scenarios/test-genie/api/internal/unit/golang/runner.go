package golang

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/shared"
	"test-genie/internal/unit/types"
)

// Runner executes Go unit tests.
type Runner struct {
	scenarioDir string
	executor    types.CommandExecutor
	logWriter   io.Writer
	workspaces  []string
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
	return &Runner{
		scenarioDir: cfg.ScenarioDir,
		executor:    executor,
		logWriter:   shared.DefaultLogWriter(cfg.LogWriter),
	}
}

// Name returns the runner's language name.
func (r *Runner) Name() string {
	return "go"
}

// Detect returns true if a Go project exists in the scenario.
func (r *Runner) Detect() bool {
	r.workspaces = r.discoverWorkspaces()
	return len(r.workspaces) > 0
}

// Run executes Go unit tests and returns the result.
func (r *Runner) Run(ctx context.Context) types.Result {
	if err := ctx.Err(); err != nil {
		return types.FailSystem(err, "Context cancelled")
	}

	workspaces := r.workspaces
	if len(workspaces) == 0 {
		workspaces = r.discoverWorkspaces()
	}
	if len(workspaces) == 0 {
		return types.Skip("No Go workspaces detected")
	}

	builder := types.NewResultBuilder().
		Success().
		AddSection("📂", fmt.Sprintf("go workspaces (%d)", len(workspaces)))

	var rels []string
	for _, ws := range workspaces {
		rels = append(rels, r.relativePath(ws))
	}
	builder.AddInfof("workspaces: %s", strings.Join(rels, ", "))

	for _, workspace := range workspaces {
		relPath := r.relativePath(workspace)
		builder.AddInfof("go test ./... in %s", relPath)

		// Verify directory exists
		if err := ensureDir(workspace); err != nil {
			return types.FailMisconfiguration(err, fmt.Sprintf("Ensure the %s directory exists so Go unit tests can run.", relPath))
		}

		// Prepare coverage output location
		coveragePath := r.coveragePath(workspace)
		if err := os.MkdirAll(filepath.Dir(coveragePath), 0o755); err != nil {
			return types.FailSystem(err, "Create coverage directory before running Go unit tests.")
		}

		// Check Go is available
		if err := types.EnsureCommand(r.executor, "go"); err != nil {
			return types.FailMissingDependency(err, "Install the Go toolchain to execute API unit tests.")
		}

		shared.LogStep(r.logWriter, "executing go test ./... inside %s", workspace)

		// Run go test
		if err := r.executor.Run(ctx, workspace, r.logWriter, "go", "test",
			"-coverprofile="+coveragePath,
			"-covermode=atomic",
			"./...",
		); err != nil {
			result := types.FailTestFailure(
				fmt.Errorf("go test ./... failed in %s: %w", relPath, err),
				fmt.Sprintf("Fix failing Go tests under %s before re-running the suite.", relPath),
			)
			result.Observations = append(builder.Build().Observations, result.Observations...)
			return result
		}
		builder.AddSuccessf("go test passed in %s (coverage recorded)", relPath)
	}

	return builder.Build()
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

// discoverWorkspaces finds all Go workspaces by locating go.mod files.
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
	}

	filepath.WalkDir(r.scenarioDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		dir := filepath.Dir(path)
		if _, ok := seen[dir]; ok {
			return nil
		}
		seen[dir] = struct{}{}
		workspaces = append(workspaces, dir)
		return nil
	})

	return workspaces
}

// coveragePath returns a unique coverage path per workspace.
func (r *Runner) coveragePath(workspace string) string {
	rel := r.relativePath(workspace)
	rel = strings.ReplaceAll(rel, string(os.PathSeparator), "_")
	if rel == "." || rel == "" {
		rel = "root"
	}
	filename := fmt.Sprintf("go-%s.coverage.out", rel)
	return filepath.Join(r.scenarioDir, "coverage", "go", filename)
}

// relativePath returns a scenario-relative path for display.
func (r *Runner) relativePath(path string) string {
	rel, err := filepath.Rel(r.scenarioDir, path)
	if err != nil || rel == "." {
		return "."
	}
	return rel
}
