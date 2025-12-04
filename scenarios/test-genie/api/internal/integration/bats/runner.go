package bats

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"test-genie/internal/structure/types"
)

// Runner executes BATS test suites.
type Runner interface {
	// Run executes all BATS validation checks.
	Run(ctx context.Context) RunResult
}

// CommandExecutor executes commands and returns any error.
type CommandExecutor func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error

// CommandLookup checks if a command is available.
type CommandLookup func(name string) (string, error)

// AdaptExecutor adapts an integration.CommandExecutor to bats.CommandExecutor.
// Since they have the same signature, this is just a type conversion helper.
func AdaptExecutor(exec func(ctx context.Context, dir string, logWriter io.Writer, name string, args ...string) error) CommandExecutor {
	return CommandExecutor(exec)
}

// AdaptLookup adapts an integration.CommandLookup to bats.CommandLookup.
func AdaptLookup(lookup func(name string) (string, error)) CommandLookup {
	return CommandLookup(lookup)
}

// RunResult contains the outcome of BATS execution.
type RunResult struct {
	types.Result

	// PrimarySuite is the path to the primary BATS suite that was executed.
	PrimarySuite string

	// AdditionalSuitesRun is the number of additional suites executed.
	AdditionalSuitesRun int
}

// Config holds configuration for BATS execution.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario.
	ScenarioName string
}

// runner implements the Runner interface.
type runner struct {
	config    Config
	executor  CommandExecutor
	lookup    CommandLookup
	logWriter io.Writer
}

// Option configures a runner.
type Option func(*runner)

// New creates a new BATS runner.
func New(config Config, opts ...Option) Runner {
	r := &runner{
		config:    config,
		lookup:    exec.LookPath,
		logWriter: io.Discard,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithLogger sets the log writer.
func WithLogger(w io.Writer) Option {
	return func(r *runner) {
		r.logWriter = w
	}
}

// WithExecutor sets the command executor (for testing).
func WithExecutor(exec CommandExecutor) Option {
	return func(r *runner) {
		r.executor = exec
	}
}

// WithLookup sets the command lookup function (for testing).
func WithLookup(lookup CommandLookup) Option {
	return func(r *runner) {
		r.lookup = lookup
	}
}

// FailureClassMissingDependency indicates a required tool is not installed.
const FailureClassMissingDependency types.FailureClass = "missing_dependency"

// Run executes all BATS validation checks.
func (r *runner) Run(ctx context.Context) RunResult {
	if err := ctx.Err(); err != nil {
		return RunResult{
			Result: types.FailSystem(err, "Context cancelled"),
		}
	}

	var observations []types.Observation
	observations = append(observations, types.NewSectionObservation("🧪", "Running BATS acceptance tests..."))

	// Step 1: Verify bats is available
	if _, err := r.lookup("bats"); err != nil {
		return RunResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("required command 'bats' is not available: %w", err),
				FailureClass: FailureClassMissingDependency,
				Remediation:  "Install the Bats test runner to execute CLI acceptance suites.",
				Observations: observations,
			},
		}
	}
	r.logStep("bats command detected")
	observations = append(observations, types.NewSuccessObservation("bats runtime available"))

	cliDir := filepath.Join(r.config.ScenarioDir, "cli")

	// Step 2: Find primary BATS suite
	primarySuite, err := r.findPrimarySuite(cliDir)
	if err != nil {
		return RunResult{
			Result: types.Result{
				Success:      false,
				Error:        err,
				FailureClass: types.FailureClassMisconfiguration,
				Remediation:  "Add a .bats suite under cli/ to exercise CLI workflows.",
				Observations: observations,
			},
		}
	}

	// Step 3: Execute primary suite
	baseName := filepath.Base(primarySuite)
	if err := r.runSuite(ctx, cliDir, baseName); err != nil {
		return RunResult{
			Result: types.Result{
				Success:      false,
				Error:        fmt.Errorf("%s failed: %w", baseName, err),
				FailureClass: types.FailureClassSystem,
				Remediation:  fmt.Sprintf("Inspect %s to resolve acceptance test failures.", baseName),
				Observations: observations,
			},
			PrimarySuite: primarySuite,
		}
	}
	r.logStep("%s executed successfully", baseName)
	observations = append(observations, types.NewSuccessObservation("primary Bats suite passed"))

	// Step 4: Run additional suites
	additionalCount, err := r.runAdditionalSuites(ctx, cliDir)
	if err != nil {
		return RunResult{
			Result: types.Result{
				Success:      false,
				Error:        err,
				FailureClass: types.FailureClassSystem,
				Remediation:  "Ensure cli/test contains valid .bats files with readable permissions.",
				Observations: observations,
			},
			PrimarySuite: primarySuite,
		}
	}
	if additionalCount > 0 {
		observations = append(observations, types.NewInfoObservation(fmt.Sprintf("additional Bats suites: %d", additionalCount)))
	}

	r.logStep("BATS validation complete")

	return RunResult{
		Result: types.Result{
			Success:      true,
			Observations: observations,
		},
		PrimarySuite:        primarySuite,
		AdditionalSuitesRun: additionalCount,
	}
}

// findPrimarySuite locates the main BATS suite file.
func (r *runner) findPrimarySuite(cliDir string) (string, error) {
	// Build preferred candidates list
	var preferred []string
	name := strings.TrimSpace(r.config.ScenarioName)
	if name != "" {
		preferred = append(preferred,
			filepath.Join(cliDir, name+".bats"),
			filepath.Join(cliDir, name+"-cli.bats"),
		)
	}
	preferred = append(preferred,
		filepath.Join(cliDir, "test-genie.bats"),
	)

	// Check preferred candidates
	for _, candidate := range preferred {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// Fall back to any .bats file in cli directory
	entries, err := os.ReadDir(cliDir)
	if err != nil {
		return "", fmt.Errorf("failed to scan cli directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bats") {
			continue
		}
		return filepath.Join(cliDir, entry.Name()), nil
	}

	return "", fmt.Errorf("no .bats suites found under %s", cliDir)
}

// runSuite executes a single BATS suite.
func (r *runner) runSuite(ctx context.Context, dir, suitePath string) error {
	if r.executor == nil {
		return fmt.Errorf("command executor not configured")
	}
	return r.executor(ctx, dir, r.logWriter, "bats", "--tap", suitePath)
}

// runAdditionalSuites finds and runs any .bats files in cli/test/.
func (r *runner) runAdditionalSuites(ctx context.Context, cliDir string) (int, error) {
	testDir := filepath.Join(cliDir, "test")
	info, err := os.Stat(testDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // No test directory is fine
		}
		return 0, fmt.Errorf("failed to read cli/test directory: %w", err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("cli/test must be a directory, found file at %s", testDir)
	}

	entries, err := os.ReadDir(testDir)
	if err != nil {
		return 0, fmt.Errorf("failed to list cli/test suite files: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bats") {
			continue
		}
		relPath := filepath.Join("test", entry.Name())
		r.logStep("executing %s", relPath)
		if err := r.runSuite(ctx, cliDir, relPath); err != nil {
			return count, fmt.Errorf("%s failed: %w", relPath, err)
		}
		count++
	}

	return count, nil
}

// logStep writes a step message to the log.
func (r *runner) logStep(format string, args ...interface{}) {
	if r.logWriter == nil {
		return
	}
	fmt.Fprintf(r.logWriter, format+"\n", args...)
}
