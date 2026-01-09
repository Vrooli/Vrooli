package lint

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"test-genie/internal/lint/golang"
	"test-genie/internal/lint/nodejs"
	"test-genie/internal/lint/python"
	"test-genie/internal/shared"
)

// Config holds configuration for lint validation.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario.
	ScenarioName string

	// CommandLookup is an optional custom command lookup function.
	CommandLookup LookupFunc

	// Settings holds lint configuration from .vrooli/testing.json.
	// If nil, default settings are used.
	Settings *Settings
}

// Runner orchestrates lint validation across Go, Node.js, and Python.
type Runner struct {
	config   Config
	settings *Settings

	goLinter     *golang.Linter
	nodeLinter   *nodejs.Linter
	pythonLinter *python.Linter

	logWriter io.Writer
}

// Option configures a Runner.
type Option func(*Runner)

// New creates a new lint validation runner.
func New(config Config, opts ...Option) *Runner {
	// Use provided settings or defaults
	settings := config.Settings
	if settings == nil {
		settings = DefaultSettings()
	}

	r := &Runner{
		config:    config,
		settings:  settings,
		logWriter: io.Discard,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Set defaults for linters if not provided via options
	if r.goLinter == nil {
		r.goLinter = golang.New(golang.Config{
			Dir:           filepath.Join(config.ScenarioDir, "api"),
			CommandLookup: config.CommandLookup,
		}, golang.WithLogger(r.logWriter))
	}
	if r.nodeLinter == nil {
		r.nodeLinter = nodejs.New(nodejs.Config{
			Dir:           filepath.Join(config.ScenarioDir, "ui"),
			CommandLookup: config.CommandLookup,
		}, nodejs.WithLogger(r.logWriter))
	}
	if r.pythonLinter == nil {
		r.pythonLinter = python.New(python.Config{
			Dir:           config.ScenarioDir,
			CommandLookup: config.CommandLookup,
		}, python.WithLogger(r.logWriter))
	}

	return r
}

// WithLogger sets the log writer for the runner.
func WithLogger(w io.Writer) Option {
	return func(r *Runner) {
		r.logWriter = w
	}
}

// WithGoLinter sets a custom Go linter (for testing).
func WithGoLinter(l *golang.Linter) Option {
	return func(r *Runner) {
		r.goLinter = l
	}
}

// WithNodeLinter sets a custom Node.js linter (for testing).
func WithNodeLinter(l *nodejs.Linter) Option {
	return func(r *Runner) {
		r.nodeLinter = l
	}
}

// WithPythonLinter sets a custom Python linter (for testing).
func WithPythonLinter(l *python.Linter) Option {
	return func(r *Runner) {
		r.pythonLinter = l
	}
}

// Run executes all lint validations and returns the aggregated result.
func (r *Runner) Run(ctx context.Context) *RunResult {
	if err := ctx.Err(); err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassSystem,
		}
	}

	var observations []Observation
	var summary LintSummary
	var hasTypeErrors bool

	shared.LogInfo(r.logWriter, "Starting lint validation for %s", r.config.ScenarioName)

	// Check which languages are present and enabled
	hasGo := r.hasGoProject() && r.settings.Go.IsEnabled()
	hasNode := r.hasNodeProject() && r.settings.Node.IsEnabled()
	hasPython := r.hasPythonProject() && r.settings.Python.IsEnabled()

	// Log disabled languages
	if r.hasGoProject() && !r.settings.Go.IsEnabled() {
		observations = append(observations, NewSkipObservation("Go linting disabled via configuration"))
	}
	if r.hasNodeProject() && !r.settings.Node.IsEnabled() {
		observations = append(observations, NewSkipObservation("Node.js linting disabled via configuration"))
	}
	if r.hasPythonProject() && !r.settings.Python.IsEnabled() {
		observations = append(observations, NewSkipObservation("Python linting disabled via configuration"))
	}

	if !hasGo && !hasNode && !hasPython {
		observations = append(observations, NewInfoObservation("No lintable languages detected or all disabled"))
		return &RunResult{
			Success:      true,
			Observations: observations,
			Summary:      summary,
		}
	}

	// Section: Go Linting
	if hasGo {
		observations = append(observations, NewSectionObservation("🔷", "Linting Go code..."))
		shared.LogInfo(r.logWriter, "Linting Go code...")

		result := r.goLinter.Lint(ctx)
		summary.GoChecked = true
		summary.GoIssues = len(result.Issues)
		summary.TypeErrors += result.TypeErrors
		summary.LintErrors += result.LintWarnings

		// Convert observations from the linter
		observations = append(observations, result.Observations...)

		// In strict mode, all issues are treated as type errors
		if r.settings.Go.Strict && result.LintWarnings > 0 {
			hasTypeErrors = true
			summary.TypeErrors += result.LintWarnings
		} else if result.TypeErrors > 0 {
			hasTypeErrors = true
		}

		if result.Skipped {
			shared.LogInfo(r.logWriter, "Go linting skipped: %s", result.SkipReason)
		} else if len(result.Issues) == 0 {
			shared.LogSuccess(r.logWriter, "Go code passed all checks")
		} else {
			shared.LogWarn(r.logWriter, "Go linting found %d issues", len(result.Issues))
		}
	}

	// Section: Node.js Linting
	if hasNode {
		observations = append(observations, NewSectionObservation("🟨", "Linting TypeScript/JavaScript..."))
		shared.LogInfo(r.logWriter, "Linting TypeScript/JavaScript...")

		result := r.nodeLinter.Lint(ctx)
		summary.NodeChecked = true
		summary.NodeIssues = len(result.Issues)
		summary.TypeErrors += result.TypeErrors
		summary.LintErrors += result.LintWarnings

		// Convert observations from the linter
		observations = append(observations, result.Observations...)

		// In strict mode, all issues are treated as type errors
		if r.settings.Node.Strict && result.LintWarnings > 0 {
			hasTypeErrors = true
			summary.TypeErrors += result.LintWarnings
		} else if result.TypeErrors > 0 {
			hasTypeErrors = true
		}

		if result.Skipped {
			shared.LogInfo(r.logWriter, "Node linting skipped: %s", result.SkipReason)
		} else if len(result.Issues) == 0 {
			shared.LogSuccess(r.logWriter, "TypeScript/JavaScript passed all checks")
		} else {
			shared.LogWarn(r.logWriter, "Node linting found %d issues", len(result.Issues))
		}
	}

	// Section: Python Linting
	if hasPython {
		observations = append(observations, NewSectionObservation("🐍", "Linting Python code..."))
		shared.LogInfo(r.logWriter, "Linting Python code...")

		result := r.pythonLinter.Lint(ctx)
		summary.PythonChecked = true
		summary.PythonIssues = len(result.Issues)
		summary.TypeErrors += result.TypeErrors
		summary.LintErrors += result.LintWarnings

		// Convert observations from the linter
		observations = append(observations, result.Observations...)

		// In strict mode, all issues are treated as type errors
		if r.settings.Python.Strict && result.LintWarnings > 0 {
			hasTypeErrors = true
			summary.TypeErrors += result.LintWarnings
		} else if result.TypeErrors > 0 {
			hasTypeErrors = true
		}

		if result.Skipped {
			shared.LogInfo(r.logWriter, "Python linting skipped: %s", result.SkipReason)
		} else if len(result.Issues) == 0 {
			shared.LogSuccess(r.logWriter, "Python code passed all checks")
		} else {
			shared.LogWarn(r.logWriter, "Python linting found %d issues", len(result.Issues))
		}
	}

	// Determine success: type errors fail, lint warnings don't
	success := !hasTypeErrors
	var failureClass FailureClass
	var remediation string
	var err error

	if hasTypeErrors {
		failureClass = FailureClassMisconfiguration
		remediation = fmt.Sprintf("Fix %d type error(s) before proceeding. Run type checker locally to see details.", summary.TypeErrors)
		err = fmt.Errorf("type checking failed with %d error(s)", summary.TypeErrors)
	}

	// Final summary
	totalIssues := summary.TotalIssues()
	if success {
		if totalIssues > 0 {
			observations = append(observations, NewWarningObservation(
				fmt.Sprintf("Lint completed with %d warning(s) (%d languages checked)", totalIssues, summary.TotalChecks()),
			))
		} else {
			observations = append(observations, NewSuccessObservation(
				fmt.Sprintf("Lint validation passed (%d languages checked)", summary.TotalChecks()),
			))
		}
	} else {
		observations = append(observations, NewErrorObservation(
			fmt.Sprintf("Lint validation failed: %d type error(s)", summary.TypeErrors),
		))
	}

	shared.LogInfo(r.logWriter, "Lint validation complete: %s", summary.String())

	return &RunResult{
		Success:      success,
		Error:        err,
		FailureClass: failureClass,
		Remediation:  remediation,
		Observations: observations,
		Summary:      summary,
	}
}

// hasGoProject checks if the scenario has a Go project (api/go.mod).
func (r *Runner) hasGoProject() bool {
	goModPath := filepath.Join(r.config.ScenarioDir, "api", "go.mod")
	_, err := os.Stat(goModPath)
	return err == nil
}

// hasNodeProject checks if the scenario has a Node.js project (ui/package.json).
func (r *Runner) hasNodeProject() bool {
	packagePath := filepath.Join(r.config.ScenarioDir, "ui", "package.json")
	_, err := os.Stat(packagePath)
	return err == nil
}

// hasPythonProject checks if the scenario has Python files.
func (r *Runner) hasPythonProject() bool {
	// Check for common Python indicators
	indicators := []string{
		"pyproject.toml",
		"setup.py",
		"requirements.txt",
		"pytest.ini",
	}
	for _, indicator := range indicators {
		if _, err := os.Stat(filepath.Join(r.config.ScenarioDir, indicator)); err == nil {
			return true
		}
	}
	// Also check for any .py files in the root
	entries, err := os.ReadDir(r.config.ScenarioDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".py" {
			return true
		}
	}
	return false
}
