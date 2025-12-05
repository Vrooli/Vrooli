package structure

import (
	"context"
	"fmt"
	"io"

	"test-genie/internal/shared"
	"test-genie/internal/structure/content"
	"test-genie/internal/structure/existence"
	"test-genie/internal/structure/playbooks"
)

// Config holds configuration for structure validation.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario (typically the directory name).
	ScenarioName string

	// SchemasDir is the path to the directory containing JSON schemas.
	// If empty, schema validation is skipped.
	SchemasDir string

	// Expectations holds custom validation expectations from .vrooli/testing.json.
	Expectations *Expectations
}

// Runner orchestrates structure validation across existence and content checks.
type Runner struct {
	config Config

	// Validators (injectable for testing)
	existenceValidator existence.Validator
	cliValidator       existence.CLIValidator
	schemaValidator    content.SchemaValidatorInterface
	manifestValidator  content.ManifestValidator
	playbooksValidator playbooks.Validator

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
	if r.schemaValidator == nil && config.SchemasDir != "" {
		r.schemaValidator = content.NewSchemaValidator(config.ScenarioDir, config.SchemasDir, r.logWriter)
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
	if r.playbooksValidator == nil {
		playbooksEnabled := true
		playbooksStrict := false
		if config.Expectations != nil {
			playbooksEnabled = config.Expectations.ValidatePlaybooks
			playbooksStrict = config.Expectations.PlaybooksStrict
		}
		r.playbooksValidator = playbooks.New(playbooks.Config{
			ScenarioDir: config.ScenarioDir,
			Enabled:     playbooksEnabled,
			Strict:      playbooksStrict,
		}, r.logWriter)
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

// WithSchemaValidator sets a custom schema validator (for testing).
func WithSchemaValidator(v content.SchemaValidatorInterface) Option {
	return func(r *Runner) {
		r.schemaValidator = v
	}
}

// WithManifestValidator sets a custom manifest validator (for testing).
func WithManifestValidator(v content.ManifestValidator) Option {
	return func(r *Runner) {
		r.manifestValidator = v
	}
}

// WithPlaybooksValidator sets a custom playbooks validator (for testing).
func WithPlaybooksValidator(v playbooks.Validator) Option {
	return func(r *Runner) {
		r.playbooksValidator = v
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

	shared.LogInfo(r.logWriter, "Starting structure validation for %s", r.config.ScenarioName)

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
	shared.LogInfo(r.logWriter, "Checking required directories...")

	dirsResult := r.existenceValidator.ValidateDirs(requiredDirs)
	if !dirsResult.Success {
		return r.failFromResult(dirsResult, observations)
	}
	summary.DirsChecked = dirsResult.ItemsChecked
	shared.LogSuccess(r.logWriter, "All required directories present (%d)", len(requiredDirs))
	observations = append(observations, NewSuccessObservation(fmt.Sprintf("All required directories present (%d checked)", len(requiredDirs))))

	// Section: Files
	observations = append(observations, NewSectionObservation("🔍", "Checking required files..."))
	shared.LogInfo(r.logWriter, "Checking required files...")

	filesResult := r.existenceValidator.ValidateFiles(requiredFiles)
	if !filesResult.Success {
		return r.failFromResult(filesResult, observations)
	}
	summary.FilesChecked = filesResult.ItemsChecked
	shared.LogSuccess(r.logWriter, "All required files present (%d)", len(requiredFiles))
	observations = append(observations, NewSuccessObservation(fmt.Sprintf("All required files present (%d checked)", len(requiredFiles))))

	// Section: CLI Structure
	observations = append(observations, NewSectionObservation("🖥️", "Validating CLI structure..."))
	shared.LogInfo(r.logWriter, "Validating CLI structure...")

	cliResult := r.cliValidator.Validate()
	if !cliResult.Result.Success {
		return r.failFromResult(cliResult.Result, observations)
	}
	shared.LogSuccess(r.logWriter, "CLI structure valid (%s approach)", cliResult.Approach)
	observations = append(observations, cliResult.Result.Observations...)

	// Section: Service Manifest
	observations = append(observations, NewSectionObservation("📋", "Validating service manifest..."))
	shared.LogInfo(r.logWriter, "Validating service manifest...")

	manifestResult := r.manifestValidator.Validate()
	if !manifestResult.Success {
		return r.failFromResult(manifestResult, observations)
	}
	shared.LogSuccess(r.logWriter, "service.json validated")
	observations = append(observations, manifestResult.Observations...)

	// Section: Schema Validation (if schemas directory is configured)
	if r.schemaValidator != nil {
		observations = append(observations, NewSectionObservation("📋", "Validating .vrooli config files against schemas..."))
		shared.LogInfo(r.logWriter, "Validating .vrooli config files against schemas...")

		schemaResult := r.schemaValidator.Validate()
		if !schemaResult.Success {
			return r.failFromResult(schemaResult, observations)
		}
		summary.JSONFilesValid = schemaResult.ItemsChecked
		shared.LogSuccess(r.logWriter, "All config files valid (%d)", schemaResult.ItemsChecked)
		observations = append(observations, schemaResult.Observations...)
	}

	// Section: Playbooks Structure (informational by default)
	if r.playbooksValidator != nil {
		observations = append(observations, NewSectionObservation("🎭", "Validating playbooks structure..."))
		shared.LogInfo(r.logWriter, "Validating playbooks structure...")

		playbooksResult := r.playbooksValidator.Validate()
		if !playbooksResult.Success {
			// In strict mode, playbooks issues block the phase
			return r.failFromResult(playbooksResult, observations)
		}
		observations = append(observations, playbooksResult.Observations...)
	}

	// Final summary
	totalChecks := summary.TotalChecks() + 2 // +2 for manifest checks (name + health)
	observations = append(observations, Observation{
		Type:    ObservationSuccess,
		Icon:    "✅",
		Message: fmt.Sprintf("Structure validation completed (%d checks)", totalChecks),
	})

	shared.LogSuccess(r.logWriter, "Structure validation complete")

	return &RunResult{
		Success:      true,
		Observations: observations,
		Summary:      summary,
	}
}

// failFromResult constructs a failure RunResult from a Result.
// Since all packages now use the same types via type aliases, no conversion is needed.
func (r *Runner) failFromResult(result Result, observations []Observation) *RunResult {
	return &RunResult{
		Success:      false,
		Error:        result.Error,
		FailureClass: result.FailureClass,
		Remediation:  result.Remediation,
		Observations: append(observations, result.Observations...),
	}
}
