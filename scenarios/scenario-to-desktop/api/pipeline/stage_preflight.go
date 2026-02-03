package pipeline

import (
	"context"
	"fmt"

	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/preflight"
	"scenario-to-desktop-api/shared/errors"

	runtimeapi "scenario-to-desktop-runtime/api"
)

// BundleabilityChecker checks if a scenario can run in bundled mode.
type BundleabilityChecker interface {
	CheckBundleability(scenarioName string) (*generation.BundleabilityResult, error)
}

// PreflightStage implements the preflight validation stage of the pipeline.
type PreflightStage struct {
	service              preflight.Service
	bundleabilityChecker BundleabilityChecker
	timeProvider         TimeProvider
}

// PreflightStageOption configures a PreflightStage.
type PreflightStageOption func(*PreflightStage)

// WithPreflightService sets the preflight service.
func WithPreflightService(svc preflight.Service) PreflightStageOption {
	return func(s *PreflightStage) {
		s.service = svc
	}
}

// WithBundleabilityChecker sets the bundleability checker for validating bundled mode.
func WithBundleabilityChecker(checker BundleabilityChecker) PreflightStageOption {
	return func(s *PreflightStage) {
		s.bundleabilityChecker = checker
	}
}

// WithPreflightTimeProvider sets the time provider.
func WithPreflightTimeProvider(tp TimeProvider) PreflightStageOption {
	return func(s *PreflightStage) {
		s.timeProvider = tp
	}
}

// NewPreflightStage creates a new preflight stage.
func NewPreflightStage(opts ...PreflightStageOption) *PreflightStage {
	s := &PreflightStage{
		timeProvider: NewRealTimeProvider(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Name returns the stage name.
func (s *PreflightStage) Name() string {
	return StagePreflight
}

// Dependencies returns stages that must complete before this one.
func (s *PreflightStage) Dependencies() []string {
	return []string{StageBundle}
}

// CanSkip returns whether this stage can be skipped.
func (s *PreflightStage) CanSkip(input *StageInput) bool {
	return ShouldSkipPreflight(input.Config)
}

// Execute runs the preflight validation stage.
func (s *PreflightStage) Execute(ctx context.Context, input *StageInput) *StageResult {
	result := newStageResult(s.Name(), s.timeProvider)

	// Check if stage should be skipped
	if s.CanSkip(input) {
		reason := "Skipping preflight: deployment mode is proxy"
		if input.Config.SkipPreflight {
			reason = "Skipping preflight: explicitly skipped via config"
		}
		skipStage(result, s.timeProvider, reason)
		return result
	}

	if checkCancellation(ctx, result, s.timeProvider) {
		return result
	}

	// FAIL-FAST: Check if scenario can run in bundled mode BEFORE doing expensive work.
	// This catches scenarios with required external dependencies (like PostgreSQL)
	// that cannot be packaged into a desktop application.
	if input.Config.GetDeploymentMode() == DeploymentModeBundled && s.bundleabilityChecker != nil {
		bundleability, err := s.bundleabilityChecker.CheckBundleability(input.Config.ScenarioName)
		if err != nil {
			result.Logs = append(result.Logs,
				fmt.Sprintf("Warning: bundleability check failed: %v", err))
			// Continue anyway - the check is best-effort
		} else if !bundleability.Bundleable {
			failStage(result, s.timeProvider, errors.ErrScenarioUnbundleable(
				input.Config.ScenarioName,
				bundleability.UnbundleableResource,
				bundleability.UnbundleableReason,
				bundleability.Alternatives,
			))
			return result
		} else {
			// Log warnings about resources that require swaps
			for _, warning := range bundleability.Warnings {
				result.Logs = append(result.Logs,
					fmt.Sprintf("Warning: %s", warning.Message))
			}
			if len(bundleability.RequiredResources) > 0 {
				if len(bundleability.Warnings) > 0 {
					result.Logs = append(result.Logs,
						fmt.Sprintf("Scenario requires resources: %v (proceeding with declared swaps)", bundleability.RequiredResources))
				} else {
					result.Logs = append(result.Logs,
						fmt.Sprintf("Scenario requires resources: %v (all bundleable)", bundleability.RequiredResources))
				}
			}
		}
	}

	if s.service == nil {
		failStage(result, s.timeProvider, errors.ErrPreflightServiceNotConfigured())
		return result
	}

	if input.BundleResult == nil {
		failStage(result, s.timeProvider, errors.ErrPreflightBundleNotAvailable())
		return result
	}

	manifestPath := input.BundleResult.ManifestPath
	bundleRoot := input.BundleResult.BundleDir

	result.Logs = append(result.Logs,
		fmt.Sprintf("Bundle root: %s", bundleRoot),
		fmt.Sprintf("Manifest: %s", manifestPath),
	)

	// Build preflight request
	request := preflight.Request{
		BundleManifestPath: manifestPath,
		BundleRoot:         bundleRoot,
		Secrets:            input.Config.PreflightSecrets,
		StartServices:      true,
	}

	if input.Config.PreflightTimeoutSeconds > 0 {
		request.TimeoutSeconds = input.Config.PreflightTimeoutSeconds
	} else {
		request.TimeoutSeconds = 60 // Default timeout
	}

	// Run preflight validation
	response, err := s.service.RunBundlePreflight(request)
	if err != nil {
		failStage(result, s.timeProvider, errors.ErrPreflightValidationFailed(err, nil))
		return result
	}

	// Check validation result
	if response.Status == "error" || response.Status == "failed" {
		failStage(result, s.timeProvider, errors.ErrPreflightValidationFailed(
			fmt.Errorf("preflight returned status: %s", response.Status),
			response.Errors,
		))
		result.Details = response
		return result
	}

	// Enforce bundle validation policy - fail if validation errors exist
	if response.Validation != nil && !response.Validation.Valid {
		validationErrors := extractValidationErrors(response.Validation)
		failStage(result, s.timeProvider, errors.ErrPreflightValidationFailed(
			fmt.Errorf("bundle validation failed with %d errors", len(validationErrors)),
			validationErrors,
		))
		result.Details = response
		return result
	}

	// Fail if any service fingerprints have errors (missing binaries, etc.)
	var fingerprintErrors []string
	for _, fp := range response.Fingerprints {
		if fp.Error != "" {
			// Include resolved path in error message for easier debugging
			errMsg := fmt.Sprintf("service %s (%s): %s", fp.ServiceID, fp.Platform, fp.Error)
			if fp.BinaryResolvedPath != "" {
				errMsg = fmt.Sprintf("%s (resolved: %s)", errMsg, fp.BinaryResolvedPath)
			} else if fp.BinaryPath != "" {
				errMsg = fmt.Sprintf("%s (path: %s)", errMsg, fp.BinaryPath)
			}
			fingerprintErrors = append(fingerprintErrors, errMsg)
		}
	}
	if len(fingerprintErrors) > 0 {
		failStage(result, s.timeProvider, errors.ErrPreflightValidationFailed(
			fmt.Errorf("service binary validation failed for %d services", len(fingerprintErrors)),
			fingerprintErrors,
		).WithRecovery(errors.RecoveryFixInput, "Ensure all service binaries are built before running the pipeline. "+
			"Run 'make build' in the scenario directory to build binaries.").
			WithManualSteps([]string{
				"Check that the scenario has been fully built",
				"Verify binaries exist at the paths shown in the errors above",
				"For cross-platform builds, ensure binaries are compiled for each target platform",
			}))
		result.Details = response
		return result
	}

	// Fail if any critical validation checks failed
	var criticalFailures []string
	for _, check := range response.Checks {
		if check.Status == "fail" && check.Step == "validation" {
			criticalFailures = append(criticalFailures,
				fmt.Sprintf("%s: %s", check.Name, check.Detail))
		}
	}
	if len(criticalFailures) > 0 {
		failStage(result, s.timeProvider, errors.ErrPreflightValidationFailed(
			fmt.Errorf("%d critical validation checks failed", len(criticalFailures)),
			criticalFailures,
		).WithRecovery(errors.RecoveryFixInput, "Fix the validation errors listed above. "+
			"Common causes: missing UI build (run 'pnpm build' in ui/), missing assets, or incorrect manifest configuration.").
			WithManualSteps([]string{
				"Review each failed validation check above",
				"Build the UI if assets are missing: cd ui && pnpm build",
				"Verify the bundle manifest references correct file paths",
			}))
		result.Details = response
		return result
	}

	// Update input for next stage
	input.PreflightResult = response

	completeStage(result, s.timeProvider, response)
	result.Logs = append(result.Logs,
		fmt.Sprintf("Preflight status: %s", response.Status),
	)

	// Add fingerprint summary (successful binary validations)
	if len(response.Fingerprints) > 0 {
		validCount := 0
		for _, fp := range response.Fingerprints {
			if fp.Error == "" && fp.BinarySHA256 != "" {
				validCount++
			}
		}
		if validCount > 0 {
			result.Logs = append(result.Logs,
				fmt.Sprintf("Binary fingerprints validated: %d services", validCount),
			)
		}
	}

	// Add validation details with accurate pass/fail counts
	if response.Validation != nil && len(response.Checks) > 0 {
		passedCount, failedCount, warnCount := countCheckResults(response.Checks)
		if failedCount == 0 && warnCount == 0 {
			result.Logs = append(result.Logs,
				fmt.Sprintf("Validation checks: %d passed", passedCount),
			)
		} else {
			result.Logs = append(result.Logs,
				fmt.Sprintf("Validation checks: %d passed, %d failed, %d warnings", passedCount, failedCount, warnCount),
			)
		}
	}

	// Add readiness info
	if response.Ready != nil && response.Ready.Ready {
		result.Logs = append(result.Logs, "Bundle services are ready")
	}

	return result
}

// extractValidationErrors converts BundleValidationResult to string slice for error reporting.
func extractValidationErrors(v *runtimeapi.BundleValidationResult) []string {
	var errs []string
	for _, e := range v.Errors {
		errs = append(errs, fmt.Sprintf("[%s] %s: %s", e.Code, e.Service, e.Message))
	}
	for _, mb := range v.MissingBinaries {
		errs = append(errs, fmt.Sprintf("missing binary: %s (%s/%s)", mb.Path, mb.ServiceID, mb.Platform))
	}
	for _, ma := range v.MissingAssets {
		errs = append(errs, fmt.Sprintf("missing asset: %s (%s)", ma.Path, ma.ServiceID))
	}
	return errs
}

// countCheckResults counts checks by status.
func countCheckResults(checks []preflight.Check) (passed, failed, warnings int) {
	for _, check := range checks {
		switch check.Status {
		case "pass":
			passed++
		case "fail":
			failed++
		case "warning", "warn":
			warnings++
		}
	}
	return
}
