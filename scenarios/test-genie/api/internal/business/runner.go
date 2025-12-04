package business

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/business/discovery"
	"test-genie/internal/business/existence"
	"test-genie/internal/business/parsing"
	"test-genie/internal/business/validation"
)

// Runner orchestrates business validation across existence, discovery, parsing, and validation.
type Runner struct {
	config Config

	// Validators (injectable for testing)
	existenceValidator  existence.Validator
	discoveryValidator  discovery.Validator
	parsingValidator    parsing.Validator
	structuralValidator validation.Validator

	logWriter io.Writer
}

// Option configures a Runner.
type Option func(*Runner)

// New creates a new business validation runner.
func New(config Config, opts ...Option) *Runner {
	r := &Runner{
		config:    config,
		logWriter: io.Discard,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Set defaults for validators if not provided via options
	if r.existenceValidator == nil {
		r.existenceValidator = existence.New(config.ScenarioDir, r.logWriter)
	}
	if r.discoveryValidator == nil {
		r.discoveryValidator = discovery.New(config.ScenarioDir, r.logWriter)
	}
	if r.parsingValidator == nil {
		r.parsingValidator = parsing.New(r.logWriter)
	}
	if r.structuralValidator == nil {
		r.structuralValidator = validation.New(r.logWriter)
	}

	return r
}

// WithLogger sets the log writer for the runner.
func WithLogger(w io.Writer) Option {
	return func(r *Runner) {
		r.logWriter = w
	}
}

// WithExistenceValidator sets a custom existence validator (for testing).
func WithExistenceValidator(v existence.Validator) Option {
	return func(r *Runner) {
		r.existenceValidator = v
	}
}

// WithDiscoveryValidator sets a custom discovery validator (for testing).
func WithDiscoveryValidator(v discovery.Validator) Option {
	return func(r *Runner) {
		r.discoveryValidator = v
	}
}

// WithParsingValidator sets a custom parsing validator (for testing).
func WithParsingValidator(v parsing.Validator) Option {
	return func(r *Runner) {
		r.parsingValidator = v
	}
}

// WithStructuralValidator sets a custom structural validator (for testing).
func WithStructuralValidator(v validation.Validator) Option {
	return func(r *Runner) {
		r.structuralValidator = v
	}
}

// Run executes all business validations and returns the aggregated result.
func (r *Runner) Run(ctx context.Context) *RunResult {
	if err := ctx.Err(); err != nil {
		return &RunResult{
			Success:      false,
			Error:        err,
			FailureClass: FailureClassSystem,
		}
	}

	var observations []Observation
	var summary ValidationSummary

	logInfo(r.logWriter, "Starting business validation for %s", r.config.ScenarioName)

	// Section: Requirements Directory
	observations = append(observations, NewSectionObservation("📋", "Validating requirements registry..."))

	existResult := r.existenceValidator.ValidateRequirementsDir()
	if !existResult.Success {
		return r.failFromResult(existResult, observations)
	}
	observations = append(observations, existResult.Observations...)

	// Section: Index File (if required)
	if r.config.Expectations == nil || r.config.Expectations.RequireIndex {
		indexResult := r.existenceValidator.ValidateIndexFile()
		if !indexResult.Success {
			return r.failFromResult(indexResult, observations)
		}
		observations = append(observations, indexResult.Observations...)
	}

	// Section: Discovery
	observations = append(observations, NewSectionObservation("🔍", "Discovering requirement modules..."))

	requireModules := r.config.Expectations == nil || r.config.Expectations.RequireModules
	discResult := r.discoveryValidator.Discover(ctx, requireModules)
	if !discResult.Success {
		return r.failFromResult(discResult.Result, observations)
	}
	summary.ModulesFound = discResult.ModuleCount
	observations = append(observations, discResult.Observations...)

	// Section: Parsing
	observations = append(observations, NewSectionObservation("📖", "Parsing requirement modules..."))

	// Convert discovery files to parsing files
	parsingFiles := make([]parsing.DiscoveredFile, len(discResult.Files))
	for i, f := range discResult.Files {
		parsingFiles[i] = parsing.DiscoveredFile{
			AbsolutePath: f.AbsolutePath,
			RelativePath: f.RelativePath,
			IsIndex:      f.IsIndex,
			ModuleDir:    f.ModuleDir,
		}
	}

	parseResult := r.parsingValidator.Parse(ctx, parsingFiles)
	if !parseResult.Success {
		return r.failFromResult(parseResult.Result, observations)
	}
	summary.RequirementsFound = parseResult.RequirementCount
	observations = append(observations, parseResult.Observations...)

	// Section: Structural Validation
	observations = append(observations, NewSectionObservation("✅", "Running structural validation rules..."))

	valResult := r.structuralValidator.Validate(ctx, parseResult.Index, r.config.ScenarioDir)
	summary.ValidationErrors = valResult.ErrorCount
	summary.ValidationWarns = valResult.WarningCount

	if !valResult.Success {
		return &RunResult{
			Success:      false,
			Error:        valResult.Error,
			FailureClass: valResult.FailureClass,
			Remediation:  valResult.Remediation,
			Observations: append(observations, valResult.Observations...),
			Summary:      summary,
		}
	}
	observations = append(observations, valResult.Observations...)

	// Final summary
	observations = append(observations, Observation{
		Type:    ObservationSuccess,
		Icon:    "✅",
		Message: fmt.Sprintf("Business validation completed (%s)", summary.String()),
	})

	logSuccess(r.logWriter, "Business validation complete")

	return &RunResult{
		Success:      true,
		Observations: observations,
		Summary:      summary,
	}
}

// failFromResult constructs a failure RunResult from a Result.
func (r *Runner) failFromResult(result Result, observations []Observation) *RunResult {
	return &RunResult{
		Success:      false,
		Error:        result.Error,
		FailureClass: result.FailureClass,
		Remediation:  result.Remediation,
		Observations: append(observations, result.Observations...),
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
