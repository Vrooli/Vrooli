package structure

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/structure/content"
	"test-genie/internal/structure/existence"
)

// Config holds configuration for structure validation.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario (typically the directory name).
	ScenarioName string

	// Expectations holds custom validation expectations from .vrooli/testing.json.
	Expectations *Expectations
}

// Runner orchestrates structure validation across existence and content checks.
type Runner struct {
	config Config

	// Validators (injectable for testing)
	existenceValidator existence.Validator
	cliValidator       existence.CLIValidator
	jsonValidator      content.JSONValidator
	manifestValidator  content.ManifestValidator

	logWriter io.Writer
}

// Option configures a Runner.
type Option func(*Runner)

// New creates a new structure validation runner.
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
	if r.cliValidator == nil {
		r.cliValidator = existence.NewCLIValidator(config.ScenarioDir, config.ScenarioName, r.logWriter)
	}
	if r.jsonValidator == nil {
		r.jsonValidator = content.NewJSONValidator(config.ScenarioDir, r.logWriter)
	}
	if r.manifestValidator == nil {
		validateName := true
		if config.Expectations != nil {
			validateName = config.Expectations.ValidateServiceName
		}
		r.manifestValidator = content.NewManifestValidator(
			config.ScenarioDir,
			config.ScenarioName,
			r.logWriter,
			content.WithNameValidation(validateName),
		)
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

// WithCLIValidator sets a custom CLI validator (for testing).
func WithCLIValidator(v existence.CLIValidator) Option {
	return func(r *Runner) {
		r.cliValidator = v
	}
}

// WithJSONValidator sets a custom JSON validator (for testing).
func WithJSONValidator(v content.JSONValidator) Option {
	return func(r *Runner) {
		r.jsonValidator = v
	}
}

// WithManifestValidator sets a custom manifest validator (for testing).
func WithManifestValidator(v content.ManifestValidator) Option {
	return func(r *Runner) {
		r.manifestValidator = v
	}
}

// Run executes all structure validations and returns the aggregated result.
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

	logInfo(r.logWriter, "Starting structure validation for %s", r.config.ScenarioName)

	// Resolve requirements based on expectations
	var additionalDirs, additionalFiles, excludedDirs, excludedFiles []string
	if r.config.Expectations != nil {
		additionalDirs = r.config.Expectations.AdditionalDirs
		additionalFiles = r.config.Expectations.AdditionalFiles
		excludedDirs = r.config.Expectations.ExcludedDirs
		excludedFiles = r.config.Expectations.ExcludedFiles
	}
	requiredDirs, requiredFiles := existence.ResolveRequirementsWithOverrides(
		additionalDirs, additionalFiles, excludedDirs, excludedFiles,
	)

	// Section: Scenario Structure
	observations = append(observations, NewSectionObservation("🏗️", "Validating scenario structure..."))

	// Section: Directories
	observations = append(observations, NewSectionObservation("🔍", "Checking required directories..."))
	logInfo(r.logWriter, "Checking required directories...")

	dirsResult := r.existenceValidator.ValidateDirs(requiredDirs)
	if !dirsResult.Success {
		return r.failFromExistence(dirsResult, observations)
	}
	summary.DirsChecked = dirsResult.ItemsChecked
	logSuccess(r.logWriter, "All required directories present (%d)", len(requiredDirs))
	observations = append(observations, NewSuccessObservation(fmt.Sprintf("All required directories present (%d checked)", len(requiredDirs))))

	// Section: Files
	observations = append(observations, NewSectionObservation("🔍", "Checking required files..."))
	logInfo(r.logWriter, "Checking required files...")

	filesResult := r.existenceValidator.ValidateFiles(requiredFiles)
	if !filesResult.Success {
		return r.failFromExistence(filesResult, observations)
	}
	summary.FilesChecked = filesResult.ItemsChecked
	logSuccess(r.logWriter, "All required files present (%d)", len(requiredFiles))
	observations = append(observations, NewSuccessObservation(fmt.Sprintf("All required files present (%d checked)", len(requiredFiles))))

	// Section: CLI Structure
	observations = append(observations, NewSectionObservation("🖥️", "Validating CLI structure..."))
	logInfo(r.logWriter, "Validating CLI structure...")

	cliResult := r.cliValidator.Validate()
	if !cliResult.Result.Success {
		return r.failFromExistence(cliResult.Result, observations)
	}
	logSuccess(r.logWriter, "CLI structure valid (%s approach)", cliResult.Approach)
	observations = append(observations, convertExistenceObservations(cliResult.Result.Observations)...)

	// Section: Service Manifest
	observations = append(observations, NewSectionObservation("📋", "Validating service manifest..."))
	logInfo(r.logWriter, "Validating service manifest...")

	manifestResult := r.manifestValidator.Validate()
	if !manifestResult.Success {
		return r.failFromContent(manifestResult, observations)
	}
	logSuccess(r.logWriter, "service.json validated")
	observations = append(observations, convertContentObservations(manifestResult.Observations)...)

	// Section: JSON Validation (if enabled)
	if r.config.Expectations == nil || r.config.Expectations.ValidateJSONFiles {
		observations = append(observations, NewSectionObservation("📄", "Validating JSON files..."))
		logInfo(r.logWriter, "Validating JSON files...")

		jsonResult := r.jsonValidator.Validate()
		if !jsonResult.Success {
			return r.failFromContent(jsonResult, observations)
		}
		summary.JSONFilesValid = jsonResult.ItemsChecked
		logSuccess(r.logWriter, "All JSON files valid (%d)", jsonResult.ItemsChecked)
		observations = append(observations, NewSuccessObservation(fmt.Sprintf("All JSON files are valid (%d checked)", jsonResult.ItemsChecked)))
	}

	// Final summary
	totalChecks := summary.TotalChecks() + 2 // +2 for manifest checks (name + health)
	observations = append(observations, Observation{
		Type:    ObservationSuccess,
		Icon:    "✅",
		Message: fmt.Sprintf("Structure validation completed (%d checks)", totalChecks),
	})

	logSuccess(r.logWriter, "Structure validation complete")

	return &RunResult{
		Success:      true,
		Observations: observations,
		Summary:      summary,
	}
}

// failFromExistence constructs a failure RunResult from an existence.Result.
func (r *Runner) failFromExistence(result existence.Result, observations []Observation) *RunResult {
	return &RunResult{
		Success:      false,
		Error:        result.Error,
		FailureClass: convertExistenceFailureClass(result.FailureClass),
		Remediation:  result.Remediation,
		Observations: append(observations, convertExistenceObservations(result.Observations)...),
	}
}

// failFromContent constructs a failure RunResult from a content.Result.
func (r *Runner) failFromContent(result content.Result, observations []Observation) *RunResult {
	return &RunResult{
		Success:      false,
		Error:        result.Error,
		FailureClass: convertContentFailureClass(result.FailureClass),
		Remediation:  result.Remediation,
		Observations: append(observations, convertContentObservations(result.Observations)...),
	}
}

// convertExistenceFailureClass converts existence.FailureClass to structure.FailureClass.
func convertExistenceFailureClass(fc existence.FailureClass) FailureClass {
	switch fc {
	case existence.FailureClassMisconfiguration:
		return FailureClassMisconfiguration
	case existence.FailureClassSystem:
		return FailureClassSystem
	default:
		return FailureClassSystem
	}
}

// convertContentFailureClass converts content.FailureClass to structure.FailureClass.
func convertContentFailureClass(fc content.FailureClass) FailureClass {
	switch fc {
	case content.FailureClassMisconfiguration:
		return FailureClassMisconfiguration
	case content.FailureClassSystem:
		return FailureClassSystem
	default:
		return FailureClassSystem
	}
}

// convertExistenceObservations converts existence observations to structure observations.
func convertExistenceObservations(obs []existence.Observation) []Observation {
	result := make([]Observation, len(obs))
	for i, o := range obs {
		result[i] = Observation{
			Type:    convertExistenceObservationType(o.Type),
			Icon:    o.Icon,
			Message: o.Message,
		}
	}
	return result
}

// convertContentObservations converts content observations to structure observations.
func convertContentObservations(obs []content.Observation) []Observation {
	result := make([]Observation, len(obs))
	for i, o := range obs {
		result[i] = Observation{
			Type:    convertContentObservationType(o.Type),
			Icon:    o.Icon,
			Message: o.Message,
		}
	}
	return result
}

// convertExistenceObservationType converts existence.ObservationType to structure.ObservationType.
func convertExistenceObservationType(t existence.ObservationType) ObservationType {
	switch t {
	case existence.ObservationSection:
		return ObservationSection
	case existence.ObservationSuccess:
		return ObservationSuccess
	case existence.ObservationWarning:
		return ObservationWarning
	case existence.ObservationError:
		return ObservationError
	case existence.ObservationInfo:
		return ObservationInfo
	case existence.ObservationSkip:
		return ObservationSkip
	default:
		return ObservationInfo
	}
}

// convertContentObservationType converts content.ObservationType to structure.ObservationType.
func convertContentObservationType(t content.ObservationType) ObservationType {
	switch t {
	case content.ObservationSection:
		return ObservationSection
	case content.ObservationSuccess:
		return ObservationSuccess
	case content.ObservationWarning:
		return ObservationWarning
	case content.ObservationError:
		return ObservationError
	case content.ObservationInfo:
		return ObservationInfo
	case content.ObservationSkip:
		return ObservationSkip
	default:
		return ObservationInfo
	}
}

// logInfo writes an info message.
func logInfo(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "🔍 %s\n", msg)
}

// logSuccess writes a success message.
func logSuccess(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "[SUCCESS] ✅ %s\n", msg)
}
