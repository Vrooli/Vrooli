package performance

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/performance/golang"
	"test-genie/internal/performance/nodejs"
)

// Runner orchestrates performance validation across Go and Node.js builds.
type Runner struct {
	config Config

	// Validators (injectable for testing)
	golangValidator golang.Validator
	nodejsValidator nodejs.Validator

	logWriter io.Writer
}

// Option configures a Runner.
type Option func(*Runner)

// New creates a new performance validation runner.
func New(config Config, opts ...Option) *Runner {
	r := &Runner{
		config:    config,
		logWriter: io.Discard,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Set defaults for validators if not provided via options
	if r.golangValidator == nil {
		r.golangValidator = golang.New(config.ScenarioDir, golang.WithLogger(r.logWriter))
	}
	if r.nodejsValidator == nil {
		r.nodejsValidator = nodejs.New(config.ScenarioDir, nodejs.WithLogger(r.logWriter))
	}

	return r
}

// WithLogger sets the log writer for the runner.
func WithLogger(w io.Writer) Option {
	return func(r *Runner) {
		r.logWriter = w
	}
}

// WithGolangValidator sets a custom Go validator (for testing).
func WithGolangValidator(v golang.Validator) Option {
	return func(r *Runner) {
		r.golangValidator = v
	}
}

// WithNodejsValidator sets a custom Node.js validator (for testing).
func WithNodejsValidator(v nodejs.Validator) Option {
	return func(r *Runner) {
		r.nodejsValidator = v
	}
}

// Run executes all performance validations and returns the aggregated result.
func (r *Runner) Run(ctx context.Context) *RunResult {
	if err := ctx.Err(); err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassSystem,
		}
	}

	var observations []Observation
	var summary BenchmarkSummary

	logInfo(r.logWriter, "Starting performance validation for %s", r.config.ScenarioName)

	// Get expectations (with defaults)
	expectations := r.config.Expectations
	if expectations == nil {
		expectations = DefaultExpectations()
	}

	// Section: Go Build
	observations = append(observations, NewSectionObservation("🔨", "Benchmarking Go API build..."))

	goResult := r.golangValidator.Benchmark(ctx, expectations.GoBuildMaxDuration)
	summary.GoBuildDuration = goResult.Duration
	observations = append(observations, goResult.Observations...)

	if !goResult.Success {
		summary.GoBuildPassed = false
		return &RunResult{
			Success:      false,
			Error:        goResult.Error,
			FailureClass: goResult.FailureClass,
			Remediation:  goResult.Remediation,
			Observations: observations,
			Summary:      summary,
		}
	}
	summary.GoBuildPassed = true

	// Section: UI Build
	observations = append(observations, NewSectionObservation("📦", "Benchmarking UI build..."))

	uiResult := r.nodejsValidator.Benchmark(ctx, expectations.UIBuildMaxDuration)
	summary.UIBuildDuration = uiResult.Duration
	summary.UIBuildSkipped = uiResult.Skipped
	observations = append(observations, uiResult.Observations...)

	if !uiResult.Success && !uiResult.Skipped {
		summary.UIBuildPassed = false
		return &RunResult{
			Success:      false,
			Error:        uiResult.Error,
			FailureClass: uiResult.FailureClass,
			Remediation:  uiResult.Remediation,
			Observations: observations,
			Summary:      summary,
		}
	}
	summary.UIBuildPassed = uiResult.Success || uiResult.Skipped

	// Final summary
	observations = append(observations, Observation{
		Type:    ObservationSuccess,
		Icon:    "✅",
		Message: fmt.Sprintf("Performance validation completed (%s)", summary.String()),
	})

	logSuccess(r.logWriter, "Performance validation complete")

	return &RunResult{
		Success:      true,
		Observations: observations,
		Summary:      summary,
	}
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
	fmt.Fprintf(w, "[SUCCESS] ✅ %s\n", msg)
}
