package pipeline

import (
	"context"
	"fmt"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/preflight"
	"scenario-to-desktop-api/shared/errors"

	runtimeapi "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/api"
)

// BundleabilityChecker checks if a scenario can run in bundled mode.
type BundleabilityChecker interface {
	CheckBundleability(scenarioName string) (*generation.BundleabilityResult, error)
}

// TargetBundleabilityChecker is implemented by analyzers that can evaluate
// every requested desktop OS instead of silently using the packager host.
type TargetBundleabilityChecker interface {
	CheckBundleabilityForPlatforms(scenarioName string, platforms []string, arch string) (*generation.BundleabilityResult, error)
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
	if failed := s.checkBundleability(input, result); failed {
		return result
	}

	if s.service == nil {
		failStage(result, s.timeProvider, errors.ErrPreflightServiceNotConfigured())
		return result
	}

	if input.BundleResult == nil {
		failStage(result, s.timeProvider, errors.ErrPreflightBundleNotAvailable())
		return result
	}

	// Run preflight validation and check response
	response, err := s.runPreflight(input, result)
	if err != nil {
		failStage(result, s.timeProvider, errors.ErrPreflightValidationFailed(err, nil))
		return result
	}

	// Validate response status and contents
	if failed := s.validateResponse(response, result); failed {
		return result
	}

	// Update input for next stage
	input.PreflightResult = response

	completeStage(result, s.timeProvider, response)
	result.Logs = append(result.Logs,
		fmt.Sprintf("Preflight status: %s", response.Status),
	)

	appendResponseSummaryLogs(response, result)

	return result
}

// checkBundleability checks if the scenario can run in bundled mode.
// Returns true if the stage should exit early (failure).
func (s *PreflightStage) checkBundleability(input *StageInput, result *StageResult) bool {
	if input.Config.GetDeploymentMode() != DeploymentModeBundled || s.bundleabilityChecker == nil {
		return false
	}

	var bundleability *generation.BundleabilityResult
	var err error
	if targetChecker, ok := s.bundleabilityChecker.(TargetBundleabilityChecker); ok {
		bundleability, err = targetChecker.CheckBundleabilityForPlatforms(input.Config.ScenarioName, input.Config.Platforms, "")
	} else {
		bundleability, err = s.bundleabilityChecker.CheckBundleability(input.Config.ScenarioName)
	}
	switch {
	case err != nil:
		// Resolution is already a required earlier stage. Keep this independent
		// analyzer as defense in depth, but never downgrade an admission error to
		// a warning after a bundle has been created.
		failStage(result, s.timeProvider, errors.ErrScenarioUnbundleable(input.Config.ScenarioName, "deployment-resolution", err.Error(), nil))
		return true
	case !bundleability.Bundleable:
		failStage(result, s.timeProvider, errors.ErrScenarioUnbundleable(
			input.Config.ScenarioName,
			bundleability.UnbundleableResource,
			bundleability.UnbundleableReason,
			bundleability.Alternatives,
		))
		return true
	default:
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
	return false
}

// runPreflight builds the preflight request, executes it, and appends path logs.
func (s *PreflightStage) runPreflight(input *StageInput, result *StageResult) (*preflight.Response, error) {
	manifestPath := input.BundleResult.ManifestPath
	bundleRoot := input.BundleResult.BundleDir

	result.Logs = append(result.Logs,
		fmt.Sprintf("Bundle root: %s", bundleRoot),
		fmt.Sprintf("Manifest: %s", manifestPath),
	)

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

	return s.service.RunBundlePreflight(request)
}

// validateResponse checks the preflight response for errors, invalid validation,
// fingerprint errors, and critical check failures.
// Returns true if the stage should exit early (failure).
func (s *PreflightStage) validateResponse(response *preflight.Response, result *StageResult) bool {
	// Check validation result
	if response.Status == "error" || response.Status == "failed" {
		failStage(result, s.timeProvider, errors.ErrPreflightValidationFailed(
			fmt.Errorf("preflight returned status: %s", response.Status),
			response.Errors,
		))
		result.Details = response
		return true
	}

	// Enforce bundle validation policy - fail if validation errors exist
	if response.Validation != nil && !response.Validation.Valid {
		validationErrors := extractValidationErrors(response.Validation)
		failStage(result, s.timeProvider, errors.ErrPreflightValidationFailed(
			fmt.Errorf("bundle validation failed with %d errors", len(validationErrors)),
			validationErrors,
		))
		result.Details = response
		return true
	}

	if failed := s.validateFingerprints(response, result); failed {
		return true
	}

	return s.validateCriticalChecks(response, result)
}

// validateFingerprints checks for service fingerprint errors (missing binaries, etc.).
// Returns true if the stage should exit early (failure).
func (s *PreflightStage) validateFingerprints(response *preflight.Response, result *StageResult) bool {
	var fingerprintErrors []string
	for _, fp := range response.Fingerprints {
		if fp.Error != "" {
			errMsg := fmt.Sprintf("service %s (%s): %s", fp.ServiceID, fp.Platform, fp.Error)
			if fp.BinaryResolvedPath != "" {
				errMsg = fmt.Sprintf("%s (resolved: %s)", errMsg, fp.BinaryResolvedPath)
			} else if fp.BinaryPath != "" {
				errMsg = fmt.Sprintf("%s (path: %s)", errMsg, fp.BinaryPath)
			}
			fingerprintErrors = append(fingerprintErrors, errMsg)
		}
	}
	if len(fingerprintErrors) == 0 {
		return false
	}

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
	return true
}

// validateCriticalChecks checks for critical validation check failures.
// Returns true if the stage should exit early (failure).
func (s *PreflightStage) validateCriticalChecks(response *preflight.Response, result *StageResult) bool {
	var criticalFailures []string
	for _, check := range response.Checks {
		if check.Status == "fail" && check.Step == "validation" {
			criticalFailures = append(criticalFailures,
				fmt.Sprintf("%s: %s", check.Name, check.Detail))
		}
	}
	if len(criticalFailures) == 0 {
		return false
	}

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
	return true
}

// appendResponseSummaryLogs adds fingerprint, validation, and readiness summary logs.
func appendResponseSummaryLogs(response *preflight.Response, result *StageResult) {
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
