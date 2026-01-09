package nodejs

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

// Runner executes Node.js unit tests.
type Runner struct {
	scenarioDir string
	executor    types.CommandExecutor
	logWriter   io.Writer
	workspaces  []string
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
	return &Runner{
		scenarioDir: cfg.ScenarioDir,
		executor:    executor,
		logWriter:   shared.DefaultLogWriter(cfg.LogWriter),
	}
}

// Name returns the runner's language name.
func (r *Runner) Name() string {
	return "node"
}

// Detect returns true if a Node.js project exists in the scenario.
func (r *Runner) Detect() bool {
	r.workspaces = r.discoverWorkspaces()
	return len(r.workspaces) > 0
}

// Run executes Node.js unit tests and returns the result.
func (r *Runner) Run(ctx context.Context) types.Result {
	if err := ctx.Err(); err != nil {
		return types.FailSystem(err, "Context cancelled")
	}

	workspaces := r.workspaces
	if len(workspaces) == 0 {
		workspaces = r.discoverWorkspaces()
	}
	if len(workspaces) == 0 {
		return types.Skip("No package.json found")
	}

	builder := types.NewResultBuilder().
		Success().
		AddSection("📂", fmt.Sprintf("node workspaces (%d)", len(workspaces)))

	var rels []string
	for _, ws := range workspaces {
		rels = append(rels, r.relativePath(ws))
	}
	builder.AddInfof("workspaces: %s", strings.Join(rels, ", "))

	// Check Node.js is available
	if err := types.EnsureCommand(r.executor, "node"); err != nil {
		return types.FailMissingDependency(err, "Install Node.js so UI/unit suites can execute.")
	}

	runnedAny := false

	for _, nodeDir := range workspaces {
		rel := r.relativePath(nodeDir)

		// Load package.json
		manifest, err := LoadManifest(filepath.Join(nodeDir, "package.json"))
		if err != nil {
			return types.FailMisconfiguration(err, fmt.Sprintf("Fix package.json in %s so the Node workspace can be parsed.", rel))
		}

		if manifest == nil {
			builder.AddWarningf("package.json missing at %s; skipping", rel)
			continue
		}

		// Check for test script
		if !manifest.HasTestScript() {
			builder.AddWarningf("%s: package.json has no test script; skipping", rel)
			continue
		}

		// Detect package manager
		packageManager := DetectPackageManager(manifest, nodeDir)
		if err := types.EnsureCommand(r.executor, packageManager); err != nil {
			return types.FailMissingDependency(err, fmt.Sprintf("Install %s to run Node test suites.", packageManager))
		}

		builder.AddInfof("node test via %s in %s", packageManager, rel)

		// Install dependencies if needed
		if !dirExists(filepath.Join(nodeDir, "node_modules")) {
			shared.LogStep(r.logWriter, "installing Node dependencies via %s in %s", packageManager, nodeDir)
			if err := r.installDependencies(ctx, nodeDir, packageManager); err != nil {
				return types.FailSystem(
					fmt.Errorf("%s install failed in %s: %w", packageManager, rel, err),
					"Resolve dependency installation issues before re-running unit tests.",
				)
			}
		}

		// Run tests
		shared.LogStep(r.logWriter, "running Node unit tests with %s in %s", packageManager, nodeDir)
		output, err := r.executor.Capture(ctx, nodeDir, r.logWriter, packageManager, "test")
		if err != nil {
			result := types.FailTestFailure(
				fmt.Errorf("Node unit tests failed in %s: %w", rel, err),
				"Inspect the UI/unit test output above, fix failures, and rerun the suite.",
			)
			result.Observations = append(builder.Build().Observations, result.Observations...)
			return result
		}

		// Extract coverage and build result
		coverage := DetectCoverage(nodeDir, output)
		if coverage != "" {
			builder.AddInfof("%s coverage: %s%% statements", rel, coverage)
		}
		builder.AddSuccessf("node unit tests passed in %s via %s", rel, packageManager)
		runnedAny = true
	}

	if !runnedAny {
		builder.AddWarning("No runnable Node workspaces found (missing test scripts).")
		return builder.Skip("No runnable Node workspaces").Build()
	}

	return builder.Build()
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

// discoverWorkspaces finds all Node.js workspaces with a package.json.
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
		if d.Name() != "package.json" {
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

// relativePath returns a scenario-relative path for display.
func (r *Runner) relativePath(path string) string {
	rel, err := filepath.Rel(r.scenarioDir, path)
	if err != nil || rel == "." {
		return "."
	}
	return rel
}
