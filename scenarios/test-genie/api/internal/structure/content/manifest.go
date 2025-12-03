package content

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"test-genie/internal/orchestrator/workspace"
)

// manifestValidator is the default implementation of ManifestValidator.
type manifestValidator struct {
	scenarioDir  string
	scenarioName string
	validateName bool
	logWriter    io.Writer
}

// ManifestValidatorOption configures a manifest validator.
type ManifestValidatorOption func(*manifestValidator)

// WithNameValidation enables or disables service name validation.
func WithNameValidation(enabled bool) ManifestValidatorOption {
	return func(v *manifestValidator) {
		v.validateName = enabled
	}
}

// NewManifestValidator creates a new manifest content validator.
func NewManifestValidator(scenarioDir, scenarioName string, logWriter io.Writer, opts ...ManifestValidatorOption) ManifestValidator {
	v := &manifestValidator{
		scenarioDir:  scenarioDir,
		scenarioName: scenarioName,
		validateName: true, // default: validate name matches directory
		logWriter:    logWriter,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Validate implements ManifestValidator.
func (v *manifestValidator) Validate() Result {
	return ValidateManifest(v.scenarioDir, v.scenarioName, v.validateName, v.logWriter)
}

// ValidateManifest validates the service.json manifest file.
func ValidateManifest(scenarioDir, scenarioName string, validateName bool, logWriter io.Writer) Result {
	manifestPath := filepath.Join(scenarioDir, ".vrooli", "service.json")

	// Check file exists
	if err := ensureFile(manifestPath); err != nil {
		logError(logWriter, "service.json not found")
		return FailMisconfiguration(
			err,
			"Run `vrooli scenario init` or restore .vrooli/service.json.",
		)
	}
	logStep(logWriter, "  ✓ service.json exists")

	// Parse the manifest
	manifest, err := workspace.LoadServiceManifest(manifestPath)
	if err != nil {
		logError(logWriter, "Failed to parse service.json: %v", err)
		return FailMisconfiguration(
			err,
			fmt.Sprintf("Fix JSON syntax in %s so the manifest can be parsed.", manifestPath),
		)
	}
	logStep(logWriter, "  ✓ service.json valid JSON")

	// Validate service name matches scenario directory (if enabled)
	if validateName && manifest.Service.Name != scenarioName {
		logError(logWriter, "service.name '%s' does not match scenario '%s'", manifest.Service.Name, scenarioName)
		return FailMisconfiguration(
			fmt.Errorf("service.name '%s' does not match scenario '%s'", manifest.Service.Name, scenarioName),
			fmt.Sprintf("Update service.name in %s to '%s'.", manifestPath, scenarioName),
		)
	}

	// Validate service name is not empty
	if manifest.Service.Name == "" {
		logError(logWriter, "service.name is required")
		return FailMisconfiguration(
			fmt.Errorf("service.name is required in %s", manifestPath),
			"Set service.name to the scenario folder name in .vrooli/service.json.",
		)
	}
	logStep(logWriter, "  ✓ service.name matches scenario")

	// Validate health checks are defined
	if len(manifest.Lifecycle.Health.Checks) == 0 {
		logError(logWriter, "No health checks defined")
		return FailMisconfiguration(
			fmt.Errorf("lifecycle.health.checks must define at least one entry in %s", manifestPath),
			"Add at least one health check entry under lifecycle.health.checks.",
		)
	}
	logStep(logWriter, "  ✓ health checks defined (%d)", len(manifest.Lifecycle.Health.Checks))

	return OK().WithObservations(
		NewSuccessObservation("service.json name matches scenario directory"),
		NewSuccessObservation("Health checks defined"),
	)
}

// ensureFile verifies that a path exists and is a file.
func ensureFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required file missing: %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but found directory: %s", path)
	}
	return nil
}
