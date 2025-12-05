package unit

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/shared"
	"test-genie/internal/unit/golang"
	"test-genie/internal/unit/nodejs"
	"test-genie/internal/unit/python"
	"test-genie/internal/unit/shell"
)

// Config holds configuration for unit test execution.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario.
	ScenarioName string
}

// Runner orchestrates unit test execution across all languages.
type Runner struct {
	config    Config
	executor  CommandExecutor
	logWriter io.Writer
	runners   []LanguageRunner
}

// Option configures a Runner.
type Option func(*Runner)

// New creates a new unit test runner.
func New(config Config, opts ...Option) *Runner {
	r := &Runner{
		config:    config,
		executor:  NewDefaultExecutor(),
		logWriter: io.Discard,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithLogger sets the log writer for the runner.
func WithLogger(w io.Writer) Option {
	return func(r *Runner) {
		r.logWriter = w
	}
}

// WithExecutor sets a custom command executor (for testing).
func WithExecutor(e CommandExecutor) Option {
	return func(r *Runner) {
		r.executor = e
	}
}

// WithRunners sets custom language runners (for testing).
func WithRunners(runners ...LanguageRunner) Option {
	return func(r *Runner) {
		r.runners = runners
	}
}

// RunResult represents the complete outcome of running all unit tests.
type RunResult struct {
	// Success indicates whether all tests passed.
	Success bool

	// Error contains the first test error encountered.
	Error error

	// FailureClass categorizes the type of failure.
	FailureClass FailureClass

	// Remediation provides guidance on how to fix the issue.
	Remediation string

	// Observations contains all test observations.
	Observations []Observation

	// Summary provides counts by language.
	Summary RunSummary
}

// RunSummary tracks test execution by language.
type RunSummary struct {
	LanguagesDetected int
	LanguagesRun      int
	LanguagesSkipped  int
	LanguagesPassed   int
	LanguagesFailed   int
}

// TotalLanguages returns the total number of languages processed (run + skipped).
func (s RunSummary) TotalLanguages() int {
	return s.LanguagesRun + s.LanguagesSkipped
}

// String returns a human-readable summary.
func (s RunSummary) String() string {
	if s.LanguagesFailed > 0 {
		return fmt.Sprintf("%d passed, %d failed, %d skipped",
			s.LanguagesPassed, s.LanguagesFailed, s.LanguagesSkipped)
	}
	if s.LanguagesRun == 0 {
		return fmt.Sprintf("%d skipped (no languages detected)", s.LanguagesSkipped)
	}
	return fmt.Sprintf("%d passed, %d skipped", s.LanguagesPassed, s.LanguagesSkipped)
}

// Run executes all unit tests and returns the aggregated result.
func (r *Runner) Run(ctx context.Context) *RunResult {
	if err := ctx.Err(); err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassSystem,
		}
	}

	var observations []Observation
	var summary RunSummary
	var firstFailure *Result

	shared.LogInfo(r.logWriter, "Starting unit test execution for %s", r.config.ScenarioName)

	// Get runners (use injected runners or create defaults)
	runners := r.runners
	if len(runners) == 0 {
		runners = r.createDefaultRunners()
	}

	for _, runner := range runners {
		observations = append(observations, NewSectionObservation("🧪", fmt.Sprintf("Checking %s tests...", runner.Name())))

		if !runner.Detect() {
			summary.LanguagesSkipped++
			observations = append(observations, NewSkipObservation(fmt.Sprintf("%s not detected", runner.Name())))
			shared.LogInfo(r.logWriter, "%s not detected, skipping", runner.Name())
			continue
		}

		summary.LanguagesDetected++
		shared.LogInfo(r.logWriter, "Running %s tests...", runner.Name())

		result := runner.Run(ctx)

		if result.Skipped {
			observations = append(observations, result.Observations...)
			summary.LanguagesSkipped++
			continue
		}

		summary.LanguagesRun++
		observations = append(observations, result.Observations...)

		if !result.Success {
			summary.LanguagesFailed++
			// Record first failure but continue running remaining languages
			if firstFailure == nil {
				r := result
				firstFailure = &r
			}
			continue
		}

		summary.LanguagesPassed++
		successMsg := fmt.Sprintf("%s tests passed", runner.Name())
		if result.Coverage != "" {
			successMsg += fmt.Sprintf(" (coverage: %s%%)", result.Coverage)
		}
		observations = append(observations, NewSuccessObservation(successMsg))
		shared.LogSuccess(r.logWriter, "%s tests passed", runner.Name())
	}

	observations = append(observations, Observation{
		Type:    ObservationSuccess,
		Icon:    "✅",
		Message: fmt.Sprintf("Unit tests completed (%d languages passed)", summary.LanguagesPassed),
	})

	shared.LogSuccess(r.logWriter, "Unit test execution complete")

	// If any failures occurred, surface the first while still reporting full summary.
	if firstFailure != nil {
		return &RunResult{
			Success:      false,
			Error:        firstFailure.Error,
			FailureClass: firstFailure.FailureClass,
			Remediation:  firstFailure.Remediation,
			Observations: observations,
			Summary:      summary,
		}
	}

	return &RunResult{
		Success:      true,
		Observations: observations,
		Summary:      summary,
	}
}

// createDefaultRunners creates the default set of language runners.
// This is called when no custom runners are provided.
func (r *Runner) createDefaultRunners() []LanguageRunner {
	return []LanguageRunner{
		golang.New(golang.Config{
			ScenarioDir: r.config.ScenarioDir,
			Executor:    r.executor,
			LogWriter:   r.logWriter,
		}),
		nodejs.New(nodejs.Config{
			ScenarioDir: r.config.ScenarioDir,
			Executor:    r.executor,
			LogWriter:   r.logWriter,
		}),
		python.New(python.Config{
			ScenarioDir: r.config.ScenarioDir,
			Executor:    r.executor,
			LogWriter:   r.logWriter,
		}),
		shell.New(shell.Config{
			ScenarioDir:  r.config.ScenarioDir,
			ScenarioName: r.config.ScenarioName,
			Executor:     r.executor,
			LogWriter:    r.logWriter,
		}),
	}
}
