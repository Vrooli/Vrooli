package errors

import (
	"fmt"
	"strings"
)

// ---- Domain-specific convenience constructors ----

// Bundle domain constructors

// ErrBundleNotFound creates a bundle not found error.
func ErrBundleNotFound(bundlePath string) *DomainError {
	return New(CodeBundleNotFound, "bundle not found").
		WithDetail("bundle_path", bundlePath).
		InDomain("bundle")
}

// ErrBundleManifest creates a bundle manifest error.
func ErrBundleManifest(cause error) *DomainError {
	return Wrap(CodeBundleManifestError, cause, "failed to parse bundle manifest").
		InDomain("bundle")
}

// Build domain constructors

// ErrBuildNotFound creates a build not found error.
func ErrBuildNotFound(buildID string) *DomainError {
	return New(CodeBuildNotFound, "build not found").
		WithDetail("build_id", buildID).
		InDomain("build")
}

// ErrBuildFailed creates a build failed error.
func ErrBuildFailed(cause error, platform string) *DomainError {
	return Wrap(CodeBuildFailed, cause, "build failed").
		WithDetail("platform", platform).
		InDomain("build")
}

// Generation domain constructors

// ErrWrapperNotFound creates a wrapper not found error.
func ErrWrapperNotFound(scenario string) *DomainError {
	return New(CodeWrapperNotFound, "wrapper not found").
		WithDetail("scenario", scenario).
		InDomain("generation")
}

// ErrScenarioNotFound creates a scenario not found error.
func ErrScenarioNotFound(scenario string) *DomainError {
	return New(CodeScenarioNotFound, "scenario not found").
		WithDetail("scenario", scenario).
		InDomain("generation")
}

// ErrScenarioUnbundleable creates an error indicating a scenario cannot run in bundled mode.
// This occurs when a scenario has required dependencies that cannot be packaged into a desktop app.
// The alternatives parameter can include swap options if available.
func ErrScenarioUnbundleable(scenario, resource, reason string, alternatives []string) *DomainError {
	steps := []string{
		fmt.Sprintf("Use --deployment-mode external-server to connect to a running server with %s", resource),
	}
	if len(alternatives) > 0 {
		steps = append(steps,
			fmt.Sprintf("Or configure dependency swap: %s → %s (requires scenario API support)", resource, strings.Join(alternatives, " or ")))
	}
	steps = append(steps,
		"Or ensure the required resource is running locally before starting the desktop app",
		"See docs/deployment/tiers/tier-2-desktop.md for deployment mode options",
	)

	err := New(CodeScenarioUnbundleable,
		fmt.Sprintf("scenario '%s' cannot run in bundled mode: requires '%s'", scenario, resource)).
		WithDetail("scenario", scenario).
		WithDetail("required_resource", resource).
		WithRecovery(RecoveryFixInput, reason).
		WithManualSteps(steps).
		InDomain("preflight")

	if len(alternatives) > 0 {
		err = err.WithDetail("alternatives", alternatives)
	}

	return err
}

// ErrTemplateNotFound creates a template not found error.
func ErrTemplateNotFound(templateType string) *DomainError {
	return New(CodeTemplateNotFound, "template not found").
		WithDetail("template_type", templateType).
		InDomain("generation")
}

// Preflight domain constructors

// ErrSessionNotFound creates a session not found error.
func ErrSessionNotFound(sessionID string) *DomainError {
	return New(CodeSessionNotFound, "session not found").
		WithDetail("session_id", sessionID).
		InDomain("preflight")
}

// ErrSessionExpired creates a session expired error.
func ErrSessionExpired(sessionID string) *DomainError {
	return New(CodeSessionExpired, "session expired").
		WithDetail("session_id", sessionID).
		InDomain("preflight")
}

// ErrJobNotFound creates a job not found error.
func ErrJobNotFound(jobID string) *DomainError {
	return New(CodeJobNotFound, "job not found").
		WithDetail("job_id", jobID).
		InDomain("preflight")
}

// ErrPreflightFailed creates a preflight failed error.
func ErrPreflightFailed(cause error) *DomainError {
	return Wrap(CodePreflightFailed, cause, "preflight validation failed").
		InDomain("preflight")
}

// Smoke test domain constructors

// ErrSmokeTestNotFound creates a smoke test not found error.
func ErrSmokeTestNotFound(testID string) *DomainError {
	return New(CodeSmokeTestNotFound, "smoke test not found").
		WithDetail("smoke_test_id", testID).
		InDomain("smoketest")
}

// ErrArtifactNotFound creates an artifact not found error.
func ErrArtifactNotFound(artifactPath string) *DomainError {
	return New(CodeArtifactNotFound, "artifact not found").
		WithDetail("artifact_path", artifactPath).
		InDomain("smoketest")
}

// Pipeline domain constructors

// ErrPipelineNotFound creates a pipeline not found error.
func ErrPipelineNotFound(pipelineID string) *DomainError {
	return New(CodePipelineNotFound, "pipeline not found").
		WithDetail("pipeline_id", pipelineID).
		WithRecovery(RecoveryFixInput, "Verify the pipeline ID is correct or start a new pipeline").
		InDomain("pipeline")
}

// ErrPipelineCancelled creates a pipeline cancelled error.
func ErrPipelineCancelled(pipelineID string) *DomainError {
	return New(CodePipelineCancelled, "pipeline was cancelled").
		WithDetail("pipeline_id", pipelineID).
		WithRecovery(RecoveryNone, "The pipeline was cancelled as requested").
		InDomain("pipeline")
}

// ErrPipelineNotResumable creates an error when a pipeline cannot be resumed.
func ErrPipelineNotResumable(pipelineID string, reason string) *DomainError {
	return New(CodeBadRequest, "pipeline cannot be resumed: "+reason).
		WithDetail("pipeline_id", pipelineID).
		WithRecovery(RecoveryFixInput, "Start a new pipeline instead").
		InDomain("pipeline")
}

// ErrPipelineOrchestratorNotConfigured creates an error when the orchestrator is not configured.
func ErrPipelineOrchestratorNotConfigured() *DomainError {
	return New(CodeInternal, "pipeline orchestrator not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		InDomain("pipeline")
}

// ErrPipelineInvalidStage creates an error for invalid stage names.
func ErrPipelineInvalidStage(stageName string) *DomainError {
	return New(CodeValidation, "invalid stage name: "+stageName).
		WithDetail("stage_name", stageName).
		WithDetail("valid_stages", []string{"bundle", "preflight", "generate", "build", "smoketest", "deploy"}).
		WithRecovery(RecoveryFixInput, "Use one of the valid stage names").
		InDomain("pipeline")
}

// ErrPipelineScenarioRequired creates an error when scenario_name is missing.
func ErrPipelineScenarioRequired() *DomainError {
	return New(CodeValidation, "scenario_name is required").
		WithRecovery(RecoveryFixInput, "Provide a scenario_name in the request body").
		InDomain("pipeline")
}

// Signing domain constructors

// ErrCertificateNotFound creates a certificate not found error.
func ErrCertificateNotFound(certID string) *DomainError {
	return New(CodeCertificateNotFound, "certificate not found").
		WithDetail("certificate_id", certID).
		InDomain("signing")
}

// ErrCertificateExpired creates a certificate expired error.
func ErrCertificateExpired(certID string, expiresAt string) *DomainError {
	return New(CodeCertificateExpired, "certificate has expired").
		WithDetail("certificate_id", certID).
		WithDetail("expires_at", expiresAt).
		InDomain("signing")
}

// System domain constructors

// ErrWineNotInstalled creates a Wine not installed error.
func ErrWineNotInstalled() *DomainError {
	return New(CodeWineNotInstalled, "Wine is not installed").
		InDomain("system")
}

// ---- Stage-specific error constructors ----
// These provide rich error information for pipeline stage failures.

// Bundle stage errors

// ErrBundleManifestNotFound creates an error for missing bundle manifest.
func ErrBundleManifestNotFound(path string) *DomainError {
	return New(CodeBundleManifestError, "bundle manifest not found").
		WithDetail("manifest_path", path).
		WithRecovery(RecoveryFixInput, "Ensure the bundle manifest exists at the expected path").
		WithManualSteps([]string{
			fmt.Sprintf("Check if manifest exists: ls -la %s", path),
			"Verify scenario has platforms/<framework>/bundle/bundle.json",
			"Run deployment-manager to generate the manifest if missing",
		}).
		InDomain("bundle")
}

// ErrBundleManifestGeneration creates an error for manifest generation failure.
func ErrBundleManifestGeneration(cause error) *DomainError {
	return Wrap(CodeBundleManifestError, cause, "failed to generate bundle manifest").
		WithRecovery(RecoveryRetry, "Retry the operation or check manifest generation logs").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Check deployment-manager logs for details",
			"Verify scenario service.json is valid",
			"Ensure all required dependencies are available",
		}).
		InDomain("bundle")
}

// ErrBundlePackagingFailed creates an error for bundle packaging failure.
func ErrBundlePackagingFailed(cause error, path string) *DomainError {
	return Wrap(CodeBundlePackageError, cause, "bundle packaging failed").
		WithDetail("bundle_path", path).
		WithRecovery(RecoveryRetry, "Check bundle configuration and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Review the bundle manifest for errors",
			"Check disk space and permissions",
			"Verify all referenced files exist",
		}).
		InDomain("bundle")
}

// ErrBundleServiceNotConfigured creates an error for missing bundle service.
func ErrBundleServiceNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Bundle packaging service unavailable").
		WithDetail("internal_error", "bundle packager not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the bundle packager is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("bundle")
}

// Preflight stage errors

// ErrPreflightServiceNotConfigured creates an error for missing preflight service.
func ErrPreflightServiceNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Preflight validation service unavailable").
		WithDetail("internal_error", "preflight service not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the preflight service is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("preflight")
}

// ErrPreflightValidationFailed creates an error for preflight validation failure.
func ErrPreflightValidationFailed(cause error, validationErrors []string) *DomainError {
	err := Wrap(CodePreflightFailed, cause, "preflight validation failed").
		WithRecovery(RecoveryFixInput, "Fix validation errors and retry").
		InDomain("preflight")

	if len(validationErrors) > 0 {
		err = err.WithDetail("validation_errors", validationErrors)
		steps := []string{"Fix the following validation errors:"}
		for _, e := range validationErrors {
			steps = append(steps, "  - "+e)
		}
		steps = append(steps, "Re-run the pipeline after fixing issues")
		err = err.WithManualSteps(steps)
	}

	return err
}

// ErrPreflightBundleNotAvailable creates an error when bundle result is missing.
func ErrPreflightBundleNotAvailable() *DomainError {
	return New(CodeDependencyError, "bundle result not available from previous stage").
		WithRecovery(RecoveryRetry, "Ensure bundle stage completes successfully first").
		WithManualSteps([]string{
			"Check if the bundle stage completed successfully",
			"Review bundle stage logs for errors",
			"Restart the pipeline from the bundle stage",
		}).
		InDomain("preflight")
}

// ErrPreflightTimeout creates an error for preflight timeout.
func ErrPreflightTimeout(duration string) *DomainError {
	return New(CodePreflightTimeout, "preflight validation timed out").
		WithDetail("timeout_duration", duration).
		WithRecovery(RecoveryRetryWithBackoff, "Increase timeout and retry").
		WithRetryStrategy(RetryConservative).
		WithManualSteps([]string{
			"Increase preflight_timeout_seconds in pipeline config",
			"Check if services are starting slowly",
			"Review resource usage during preflight",
		}).
		InDomain("preflight")
}

// Generate stage errors

// ErrGenerateAnalyzerNotConfigured creates an error for missing scenario analyzer.
func ErrGenerateAnalyzerNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Scenario analysis service unavailable").
		WithDetail("internal_error", "scenario analyzer not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the generation service is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("generation")
}

// ErrGenerateServiceNotConfigured creates an error for missing generation service.
func ErrGenerateServiceNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Code generation service unavailable").
		WithDetail("internal_error", "generation service not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the generation service is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("generation")
}

// ErrScenarioAnalysisFailed creates an error for scenario analysis failure.
func ErrScenarioAnalysisFailed(cause error, scenarioName string) *DomainError {
	return Wrap(CodeGenerationFailed, cause, "scenario analysis failed").
		WithDetail("scenario_name", scenarioName).
		WithRecovery(RecoveryFixInput, "Check scenario configuration and retry").
		WithManualSteps([]string{
			"Verify scenario exists in scenarios/" + scenarioName,
			"Check scenario service.json is valid JSON",
			"Ensure required fields are present in service.json",
		}).
		InDomain("generation")
}

// ErrScenarioValidationFailed creates an error for scenario validation failure.
func ErrScenarioValidationFailed(cause error, scenarioName string) *DomainError {
	return Wrap(CodeGenerationFailed, cause, "scenario validation failed").
		WithDetail("scenario_name", scenarioName).
		WithRecovery(RecoveryFixInput, "Fix scenario configuration for desktop deployment").
		WithManualSteps([]string{
			"Check scenario meets desktop deployment requirements",
			"Verify UI port and entry point are configured",
			"Review desktop deployment documentation",
		}).
		InDomain("generation")
}

// ErrDesktopConfigFailed creates an error for desktop config creation failure.
func ErrDesktopConfigFailed(cause error) *DomainError {
	return Wrap(CodeGenerationFailed, cause, "failed to create desktop config").
		WithRecovery(RecoveryRetry, "Check scenario metadata and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Verify scenario metadata is complete",
			"Check template type is supported",
			"Review generation service logs",
		}).
		InDomain("generation")
}

// ErrGenerationTimeout creates an error for generation timeout.
func ErrGenerationTimeout(buildID string, duration string) *DomainError {
	return New(CodeTimeout, "generation timed out").
		WithDetail("build_id", buildID).
		WithDetail("timeout_duration", duration).
		WithRecovery(RecoveryRetryWithBackoff, "Retry with longer timeout").
		WithRetryStrategy(RetryConservative).
		WithManualSteps([]string{
			"Check if generation is slow due to large template",
			"Review system resource usage",
			"Consider simplifying the desktop configuration",
		}).
		InDomain("generation")
}

// ErrGenerationFailed creates an error for generation failure.
func ErrGenerationFailed(cause error) *DomainError {
	return Wrap(CodeGenerationFailed, cause, "generation failed").
		WithRecovery(RecoveryRetry, "Check generation logs and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Review generation error logs",
			"Verify template files are intact",
			"Check disk space and permissions",
		}).
		InDomain("generation")
}

// Build stage errors

// ErrBuildServiceNotConfigured creates an error for missing build service.
func ErrBuildServiceNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Build service unavailable").
		WithDetail("internal_error", "build service not configured").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the build service is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("build")
}

// ErrBuildStoreNotConfigured creates an error for missing build store.
func ErrBuildStoreNotConfigured() *DomainError {
	return New(CodeServiceStartError, "Build tracking service unavailable").
		WithDetail("internal_error", "build store not configured for status polling").
		WithRecovery(RecoveryContactSupport, "Server configuration issue - contact support").
		WithManualSteps([]string{
			"Check server startup logs for initialization errors",
			"Verify the build store is properly configured",
			"Contact support if the issue persists",
		}).
		InDomain("build")
}

// ErrBuildDesktopPathMissing creates an error when desktop path is missing.
func ErrBuildDesktopPathMissing() *DomainError {
	return New(CodeDependencyError, "desktop path not available from generation stage").
		WithRecovery(RecoveryRetry, "Ensure generation stage completes successfully first").
		WithManualSteps([]string{
			"Check if the generation stage completed successfully",
			"Review generation stage logs for errors",
			"Restart the pipeline from the generation stage",
		}).
		InDomain("build")
}

// ErrBuildStartFailed creates an error for build start failure.
func ErrBuildStartFailed(cause error, platform string) *DomainError {
	return Wrap(CodeBuildFailed, cause, "build failed to start").
		WithDetail("platform", platform).
		WithRecovery(RecoveryRetry, "Check build configuration and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Verify electron-builder is installed",
			"Check platform-specific build tools are available",
			"Review build service logs for details",
		}).
		InDomain("build")
}

// ErrBuildTimedOut creates an error for build timeout.
func ErrBuildTimedOut(buildID string, duration string) *DomainError {
	return New(CodeTimeout, "build timed out").
		WithDetail("build_id", buildID).
		WithDetail("timeout_duration", duration).
		WithRecovery(RecoveryRetryWithBackoff, "Retry with longer timeout").
		WithRetryStrategy(RetryConservative).
		WithManualSteps([]string{
			"Build may be slow due to large assets",
			"Check system resource usage during build",
			"Consider building fewer platforms in parallel",
		}).
		InDomain("build")
}

// ErrBuildPlatformFailed creates an error for platform build failure.
func ErrBuildPlatformFailed(cause error, platform string, lastOutput string) *DomainError {
	err := Wrap(CodeBuildFailed, cause, "build failed").
		WithDetail("platform", platform).
		WithRecovery(RecoveryRetry, "Check build logs and retry").
		WithRetryStrategy(RetryDefault).
		InDomain("build")

	if lastOutput != "" {
		err = err.WithDiagnostic(&DiagnosticContext{
			Process: &ProcessDiagnostic{
				LastOutput: lastOutput,
			},
		})
	}

	// Platform-specific manual steps
	switch platform {
	case "linux":
		err = err.WithManualSteps([]string{
			"Check Linux build dependencies are installed",
			"Verify fpm is available for package creation",
			"Review electron-builder output for errors",
		})
	case "mac":
		err = err.WithManualSteps([]string{
			"macOS builds may require code signing",
			"Check Xcode command line tools are installed",
			"Review electron-builder output for errors",
		})
	case "win":
		err = err.WithManualSteps([]string{
			"Windows builds on Linux require Wine",
			"Check if Wine is installed and configured",
			"Review electron-builder output for errors",
		})
	default:
		err = err.WithManualSteps([]string{
			"Review electron-builder output for errors",
			"Check platform-specific requirements",
			"Verify build configuration is correct",
		})
	}

	return err
}

// Deploy stage errors

// ErrDeployFailed creates an error for deploy failure.
func ErrDeployFailed(cause error, target string) *DomainError {
	return Wrap(CodeInternal, cause, "deploy failed").
		WithDetail("target", target).
		WithRecovery(RecoveryRetry, "Check LPBS connectivity and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Verify LPBS is running and reachable",
			"Check remote profile session is active",
			"Review deploy logs for details",
		}).
		InDomain("deploy")
}

// ErrDeployTimeout creates an error for deploy timeout.
func ErrDeployTimeout(deployID string, duration string) *DomainError {
	return New(CodeTimeout, "deploy timed out").
		WithDetail("deploy_id", deployID).
		WithDetail("timeout_duration", duration).
		WithRecovery(RecoveryRetryWithBackoff, "Retry with longer timeout").
		WithRetryStrategy(RetryConservative).
		WithManualSteps([]string{
			"Large artifacts may need more upload time",
			"Check network speed to LPBS remote",
			"Consider uploading fewer artifacts in parallel",
		}).
		InDomain("deploy")
}

// Smoke test stage errors (for use by smoketest package)

// ErrSmokeTestArtifactNotFound creates an error for missing smoke test artifact.
func ErrSmokeTestArtifactNotFound(artifactPath string) *DomainError {
	return New(CodeArtifactNotFound, "smoke test artifact not found").
		WithDetail("artifact_path", artifactPath).
		WithRecovery(RecoveryFixInput, "Ensure build stage completed and produced artifacts").
		WithManualSteps([]string{
			"Verify the build stage completed successfully",
			fmt.Sprintf("Check if artifact exists: ls -la %s", artifactPath),
			"Review build logs for errors",
			"Ensure build output directory is correct",
		}).
		InDomain("smoketest")
}

// ErrSmokeTestExecutionFailed creates an error for smoke test execution failure.
func ErrSmokeTestExecutionFailed(cause error, context map[string]string) *DomainError {
	err := Wrap(CodeSmokeTestFailed, cause, "smoke test execution failed").
		WithRecovery(RecoveryRetry, "Check app startup logs and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Check if the application can run manually",
			"Verify all dependencies are installed",
			"Check system logs for crash information",
			"Use --show-output with 'pipeline run' to see full app output",
		}).
		InDomain("smoketest")

	for k, v := range context {
		err = err.WithDetail(k, v)
	}

	return err
}

// ErrSmokeTestTimeout creates an error for smoke test timeout.
func ErrSmokeTestTimeout(duration string, context map[string]string) *DomainError {
	err := New(CodeTimeout, "smoke test timed out").
		WithDetail("timeout_duration", duration).
		WithRecovery(RecoveryRetryWithBackoff, "Increase timeout and retry").
		WithRetryStrategy(RetryConservative).
		WithManualSteps([]string{
			"Increase SMOKE_TEST_TIMEOUT_MS environment variable",
			"Check if app startup is slow due to large assets",
			"Profile app initialization to identify bottlenecks",
			"Verify network connectivity if app makes startup requests",
		}).
		InDomain("smoketest")

	for k, v := range context {
		err = err.WithDetail(k, v)
	}

	return err
}

// ErrSmokeTestValidationFailed creates an error for missing success marker.
func ErrSmokeTestValidationFailed(context map[string]string) *DomainError {
	err := New(CodeSmokeTestFailed, "smoke test validation failed: success marker not found").
		WithRecovery(RecoveryFixInput, "Ensure app outputs SMOKE_TEST_RESULT=passed").
		WithManualSteps([]string{
			"Verify app outputs SMOKE_TEST_RESULT=passed on successful startup",
			"Check if app is detecting SMOKE_TEST=1 environment variable",
			"Review app smoke test handler implementation",
			"Ensure app doesn't crash before outputting success marker",
		}).
		InDomain("smoketest")

	for k, v := range context {
		err = err.WithDetail(k, v)
	}

	return err
}

// ErrSmokeTestPlatformError creates an error for platform-specific issues.
func ErrSmokeTestPlatformError(cause error, platform string) *DomainError {
	err := Wrap(CodeSmokeTestFailed, cause, "platform-specific smoke test error").
		WithDetail("platform", platform).
		WithRecovery(RecoveryInstallDependency, "Install required platform dependencies").
		InDomain("smoketest")

	// Platform-specific recovery steps
	switch platform {
	case "linux":
		err = err.WithManualSteps([]string{
			"Install xvfb for headless display: sudo apt-get install xvfb",
			"Set DISPLAY environment variable or ensure X11 is running",
			"Verify libgtk and other Electron dependencies are installed",
		}).WithAutoFix(&AutoFix{
			Command:     "sudo apt-get install -y xvfb libgtk-3-0 libnotify4 libnss3 libxss1 libxtst6 xdg-utils libatspi2.0-0 libdrm2 libgbm1 libasound2",
			Description: "Install common Electron dependencies for Linux",
			Safe:        false,
		})
	case "mac":
		err = err.WithManualSteps([]string{
			"Ensure app is properly signed for macOS",
			"Check Gatekeeper settings: spctl --status",
			"Verify app bundle structure: Contents/MacOS/ exists",
		})
	case "win":
		err = err.WithManualSteps([]string{
			"Ensure .exe file is not blocked by Windows Defender",
			"Check Windows Firewall settings",
			"Verify Visual C++ Redistributable is installed",
		})
	default:
		err = err.WithManualSteps([]string{
			"Verify platform is supported (linux, mac, win)",
			"Check platform-specific documentation",
		})
	}

	return err
}

// ErrSmokeTestTelemetryFailed creates an error for telemetry failures.
func ErrSmokeTestTelemetryFailed(cause error, context map[string]string) *DomainError {
	err := Wrap(CodeTelemetryError, cause, "smoke test telemetry failed").
		WithRecovery(RecoveryRetry, "Check telemetry service and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Check telemetry service is running and accessible",
			"Verify network connectivity to telemetry endpoint",
			"Check telemetry file permissions if using file-based fallback",
			"Review telemetry API logs for errors",
		}).
		InDomain("smoketest")

	for k, v := range context {
		err = err.WithDetail(k, v)
	}

	return err
}

// ErrSmokeTestStoreFailed creates an error for persistence failures.
func ErrSmokeTestStoreFailed(cause error) *DomainError {
	return Wrap(CodeInternal, cause, "Could not save test results").
		WithDetail("internal_error", "smoke test store operation failed").
		WithRecovery(RecoveryRetry, "Check disk space and retry").
		WithRetryStrategy(RetryDefault).
		WithManualSteps([]string{
			"Check available disk space: df -h",
			"Verify file system permissions",
			"Check if data directory exists and is writable",
		}).
		InDomain("smoketest")
}

// ErrSmokeTestCancelled creates an error for cancelled operations.
func ErrSmokeTestCancelled() *DomainError {
	return New(CodePipelineCancelled, "smoke test cancelled").
		WithRecovery(RecoveryNone, "Re-run smoke test if cancellation was unintentional").
		WithManualSteps([]string{
			"Re-run the smoke test if cancellation was unintentional",
			"Check if timeout was too short",
		}).
		InDomain("smoketest")
}
