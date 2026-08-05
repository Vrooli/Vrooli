package smoketest

import (
	"fmt"
	"strings"
	"time"

	"scenario-to-desktop-api/shared/errors"
)

func (s *DefaultService) processResults(smokeTestID, scenarioName, platform, artifactPath, displayCommand string, execResult *ExecutionResult, execErr error) {
	// Get the combined output for parsing
	output := ""
	if execResult != nil {
		output = execResult.Combined
	}

	// Debug: Log output details for lifecycle state extraction troubleshooting
	outputPreview := output
	if len(outputPreview) > 300 {
		outputPreview = outputPreview[:300] + "..."
	}
	s.logger.Info("smoke_test_processing_results",
		"smoke_test_id", smokeTestID,
		"output_length", len(output),
		"output_preview", outputPreview,
		"has_init_marker", strings.Contains(output, "SMOKE_TEST_INIT=started"),
		"has_session_id_marker", strings.Contains(output, "session_id="),
	)

	// Check for app-reported structured errors first (highest priority)
	if appError := s.outputParser.ExtractAppError(output); appError != nil {
		s.recordTypedFailure(smokeTestID, NewAppReportedError(appError))
		return
	}

	// Extract last lifecycle state for error context
	lastState := s.outputParser.ExtractLastLifecycleState(output)

	// Extract session ID from SMOKE_TEST_INIT marker for telemetry correlation
	appSessionID := s.outputParser.ExtractSessionID(output)

	// Debug: Log extraction results
	s.logger.Info("smoke_test_extraction_results",
		"smoke_test_id", smokeTestID,
		"lifecycle_state", lastState,
		"session_id", appSessionID,
		"session_id_empty", appSessionID == "",
	)

	// Store extracted lifecycle state and session ID in status for debugging
	s.store.Update(smokeTestID, func(status *Status) {
		status.ExtractedLifecycleState = lastState
		status.AppSessionID = appSessionID
	})

	result := s.outputParser.ParseResult(output)

	// Update telemetry upload status
	if result.TelemetryUploaded {
		s.transitionTo(smokeTestID, StateTelemetryUpload, "telemetry uploaded by app")
		s.store.Update(smokeTestID, func(status *Status) {
			status.TelemetryUploaded = true
			status.TelemetryUploadError = ""
		})
	} else if result.TelemetryUploadError {
		s.store.Update(smokeTestID, func(status *Status) {
			status.TelemetryUploadError = "telemetry upload failed (see logs)"
		})
	}

	// Attempt telemetry fallback if upload didn't succeed
	if !result.TelemetryUploaded {
		s.transitionTo(smokeTestID, StateTelemetryFallback, "attempting fallback")
		s.attemptTelemetryFallback(smokeTestID, scenarioName, platform, artifactPath, output, appSessionID)
	}

	// Check execution error
	if execErr != nil {
		smokeErr := s.buildExecutionError(execErr, execResult, displayCommand, platform, lastState)
		s.recordTypedFailure(smokeTestID, smokeErr)
		return
	}

	// Check for success marker
	if !result.Passed {
		context := map[string]string{
			"expected": "SMOKE_TEST_RESULT=passed",
			"platform": platform,
		}
		// Add lifecycle state to help diagnose where failure occurred
		if lastState != "" {
			context["last_lifecycle_state"] = lastState
		}
		s.recordTypedFailure(smokeTestID, NewValidationError(
			"smoke test did not report success",
			context,
		))
		return
	}

	// Success!
	s.transitionTo(smokeTestID, StatePassed, "smoke test passed")
	s.store.Update(smokeTestID, func(status *Status) {
		status.Status = "passed"
		status.Logs = append(status.Logs, "Smoke test passed")
		now := time.Now()
		status.CompletedAt = &now
	})

	s.logger.Info("smoke_test_passed",
		"smoke_test_id", smokeTestID,
		"scenario", scenarioName,
		"platform", platform,
	)
}

// buildExecutionError creates an Error with process diagnostics.
// The lifecycleState parameter allows customizing the error message based on
// how far the app progressed before failing.
func (s *DefaultService) buildExecutionError(execErr error, execResult *ExecutionResult, displayCommand, platform, lifecycleState string) *Error {
	// Determine error message based on lifecycle state and error type
	message := s.buildExecutionErrorMessage(execErr, lifecycleState)

	// Build context map
	context := map[string]string{
		"command":  displayCommand,
		"platform": platform,
	}
	if lifecycleState != "" {
		context["last_lifecycle_state"] = lifecycleState
	}

	err := NewExecutionError(message, execErr, context)

	// Use timeout error type if this is a timeout with good progress
	if isTimeout(execErr) && isLateLifecycleStage(lifecycleState) {
		err.Kind = ErrKindTimeout
		// Use lifecycle-specific recovery hints instead of generic timeout hint
		err.SuggestedAction = getLifecycleRecoveryHint(lifecycleState)
		err.ManualSteps = getLifecycleManualSteps(lifecycleState)
	}

	// Add process diagnostics if available
	if execResult != nil {
		err.Diagnostic = &errors.DiagnosticContext{
			Process: &errors.ProcessDiagnostic{
				ExitCode:   execResult.ExitCode,
				RuntimeMs:  execResult.Duration.Milliseconds(),
				LastOutput: truncateOutput(execResult.Stderr, 1000),
			},
		}
		// Add stderr to context for easier debugging
		if execResult.Stderr != "" {
			err.Context["stderr"] = truncateOutput(execResult.Stderr, 500)
		}
	}

	return err
}

// getLifecycleRecoveryHint returns a specific recovery hint based on the lifecycle state.
func getLifecycleRecoveryHint(state string) string {
	switch state {
	case "waiting_for_token":
		return "Runtime started but isn't creating auth token. Check if bundled API supports --token-path flag and is configured correctly in bundle.json"
	case "runtime_starting":
		return "Runtime binary started but may have crashed. Check if bundled API binary is compatible with this platform and has correct permissions"
	case "runtime_healthz":
		return "Runtime is not responding to health checks. The bundled API may have crashed during startup or is misconfigured"
	case "runtime_readyz":
		return "Runtime started but services aren't ready. This usually means the scenario requires external dependencies (like PostgreSQL) that cannot be bundled. Use --deployment-mode external-server to connect to a running server instead"
	case "runtime_ports":
		return "Runtime services are ready but port configuration failed. Check bundle.json for correct port mappings"
	case "ui_server_check":
		return "UI server is reachable but returned a non-2xx HTTP status. The server may be returning 404 (not found) or another error status"
	case "result":
		return "App completed smoke test but didn't exit cleanly. This is usually non-fatal - check for cleanup errors"
	default:
		return RecoveryPaths[ErrKindTimeout]
	}
}

// getLifecycleManualSteps returns specific manual steps based on the lifecycle state.
func getLifecycleManualSteps(state string) []string {
	switch state {
	case "waiting_for_token":
		return []string{
			"Check bundle.json for correct API binary configuration",
			"Verify the bundled API supports --token-path flag: ./api --help",
			"Run the bundled API manually with same flags to see startup errors",
			"Check if token directory is writable: ls -la ~/.config/<app>/runtime/",
			"Increase timeout if API startup is legitimately slow",
		}
	case "runtime_starting":
		return []string{
			"Check if bundled API binary has execute permissions: chmod +x ./api",
			"Verify binary is compatible with this platform (x64 vs arm64)",
			"Run the bundled API binary manually to see error messages",
			"Check for missing shared libraries: ldd ./api",
		}
	case "runtime_healthz":
		return []string{
			"Check bundled API logs for startup errors",
			"Verify API is listening on expected port: curl http://localhost:<port>/healthz",
			"Check if required environment variables are set",
			"Review bundle.json for correct healthz endpoint configuration",
		}
	case "runtime_readyz":
		return []string{
			"LIKELY CAUSE: Scenario requires external dependencies (PostgreSQL, Redis) that cannot be bundled",
			"SOLUTION: Use --deployment-mode external-server to connect to a running server with these dependencies",
			"Check scenario's .vrooli/service.json for required resources under dependencies.resources",
			"If bundled mode is needed, ensure all required resources are running and accessible",
			"Review API logs for specific connection errors: scenario-to-desktop pipeline status <id> --verbose",
		}
	case "runtime_ports":
		return []string{
			"Review bundle.json for correct port configuration",
			"Check if expected ports are free: netstat -tlnp | grep <port>",
			"Verify API returns expected port information from /ports endpoint",
		}
	case "ui_server_check":
		return []string{
			"The UI server is returning a non-2xx HTTP status code (e.g., 404)",
			"Check the app output for the exact status code received",
			"Verify the UI service is configured correctly in bundle.json",
			"For SPAs, ensure the server is configured to serve index.html for all routes",
			"Test the URL manually: curl -I http://127.0.0.1:<port>/",
		}
	case "result":
		return []string{
			"Check app logs for cleanup errors",
			"This is usually non-fatal - smoke test likely passed",
			"Verify app exits with code 0 after successful test",
		}
	default:
		return []string{
			"Increase SMOKE_TEST_TIMEOUT_MS environment variable",
			"Check if app startup is slow due to large assets",
			"Profile app initialization to identify bottlenecks",
			"Verify network connectivity if app makes startup requests",
		}
	}
}

// buildExecutionErrorMessage creates a descriptive error message based on
// the error type and how far the app progressed through its lifecycle.
func (s *DefaultService) buildExecutionErrorMessage(execErr error, lifecycleState string) string {
	isTimeoutErr := isTimeout(execErr)

	// If app reached a late lifecycle stage before timeout, provide informative message
	if isTimeoutErr && isLateLifecycleStage(lifecycleState) {
		switch lifecycleState {
		case "waiting_for_token":
			return "app started successfully but timed out waiting for authentication"
		case "runtime_starting":
			return "app initialized but timed out during runtime startup"
		case "result":
			return "app reached success state but timed out before completion"
		default:
			return "app started but timed out before completing smoke test"
		}
	}

	// Early stage timeout - app may have failed to start
	if isTimeoutErr {
		switch lifecycleState {
		case "init":
			return "app timed out during initialization"
		case "bundle_resolving":
			return "app timed out while resolving bundle"
		case "":
			return "app timed out before producing any lifecycle markers"
		default:
			return fmt.Sprintf("app timed out at %s stage", lifecycleState)
		}
	}

	// Non-timeout execution error
	return "smoke test process failed"
}

// isTimeout checks if the error indicates a timeout.
func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "timed out") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "context deadline exceeded")
}

// isLateLifecycleStage returns true if the app reached a stage indicating
// successful initialization (past bundle resolving).
func isLateLifecycleStage(state string) bool {
	switch state {
	case "runtime_starting", "waiting_for_token", "runtime_healthz", "runtime_readyz", "runtime_ports", "result":
		return true
	default:
		return false
	}
}

func (s *DefaultService) attemptTelemetryFallback(smokeTestID, scenarioName, platform, artifactPath, output, appSessionID string) {
	// Try to extract path from output first
	telemetryPath := s.telemetryResolver.ExtractFromOutput(output)
	if telemetryPath == "" {
		// Fall back to artifact-based resolution
		telemetryPath = s.telemetryResolver.ResolveFromArtifact(platform, artifactPath, scenarioName)
	}

	if telemetryPath == "" {
		s.store.Update(smokeTestID, func(status *Status) {
			status.Logs = append(status.Logs, "Telemetry fallback skipped: telemetry path not found")
		})
		s.logger.Info("smoke_test_telemetry_fallback_skipped",
			"smoke_test_id", smokeTestID,
			"reason", "telemetry path not found",
		)
		return
	}

	// Extract app-reported errors from telemetry for debugging
	s.extractAppErrors(smokeTestID, telemetryPath, appSessionID)

	// Ingest telemetry events as fallback
	s.ingestTelemetryFallback(smokeTestID, scenarioName, telemetryPath)
}

// extractAppErrors attempts to find app-reported errors in telemetry for this smoke test session.
func (s *DefaultService) extractAppErrors(smokeTestID, telemetryPath, appSessionID string) {
	if s.telemetryExtractor == nil {
		return
	}

	appErr, err := s.telemetryExtractor.ExtractLatestErrorForSession(telemetryPath, appSessionID)
	sessionMismatch := false

	if err == nil && appErr == nil && appSessionID != "" {
		appErr, err = s.telemetryExtractor.ExtractLatestError(telemetryPath)
		if appErr != nil {
			sessionMismatch = true
		}
	}

	if err != nil || appErr == nil {
		return
	}

	var startTime time.Time
	if status, ok := s.store.Get(smokeTestID); ok {
		startTime = status.StartedAt
	}
	isStale := IsErrorStale(appErr, startTime)

	s.store.Update(smokeTestID, func(status *Status) {
		status.AppReportedError = appErr
		status.ErrorSessionMismatch = sessionMismatch
		status.AppReportedErrorStale = isStale

		logMsg := fmt.Sprintf("App-reported error: %s", FormatTelemetryError(appErr))
		if sessionMismatch {
			logMsg += " (WARNING: session mismatch - may be from previous run)"
		} else if isStale {
			logMsg += " (WARNING: timestamp predates this smoke test)"
		}
		status.Logs = append(status.Logs, logMsg)
	})
	s.logger.Info("smoke_test_app_error_extracted",
		"smoke_test_id", smokeTestID,
		"error_event", appErr.Event,
		"error_message", appErr.Message,
		"error_session_id", appErr.SessionID,
		"app_session_id", appSessionID,
		"session_mismatch", sessionMismatch,
		"is_stale", isStale,
	)
}

// ingestTelemetryFallback reads and uploads telemetry events when regular upload didn't happen.
func (s *DefaultService) ingestTelemetryFallback(smokeTestID, scenarioName, telemetryPath string) {
	if s.telemetryIngestor == nil {
		return
	}

	events, err := s.telemetryResolver.ReadTelemetryEvents(telemetryPath, s.config.MaxTelemetryEvents)
	if err != nil {
		s.store.Update(smokeTestID, func(status *Status) {
			status.TelemetryUploadError = fmt.Sprintf("telemetry fallback read failed: %v", err)
			status.Logs = append(status.Logs, fmt.Sprintf("Telemetry fallback failed: %v", err))
		})
		s.logger.Error("smoke_test_telemetry_fallback_read_failed",
			"smoke_test_id", smokeTestID,
			"telemetry_path", telemetryPath,
			"error", err.Error(),
		)
		return
	}

	if len(events) == 0 {
		s.store.Update(smokeTestID, func(status *Status) {
			status.Logs = append(status.Logs, "Telemetry fallback skipped: no events found")
		})
		s.logger.Info("smoke_test_telemetry_fallback_skipped",
			"smoke_test_id", smokeTestID,
			"telemetry_path", telemetryPath,
			"reason", "no events found",
		)
		return
	}

	_, ingested, err := s.telemetryIngestor.IngestEvents(scenarioName, "", "smoke-test-timeout", events)
	if err != nil {
		s.store.Update(smokeTestID, func(status *Status) {
			status.TelemetryUploadError = fmt.Sprintf("telemetry fallback upload failed: %v", err)
			status.Logs = append(status.Logs, fmt.Sprintf("Telemetry fallback upload failed: %v", err))
		})
		s.logger.Error("smoke_test_telemetry_fallback_upload_failed",
			"smoke_test_id", smokeTestID,
			"telemetry_path", telemetryPath,
			"events_found", len(events),
			"error", err.Error(),
		)
		return
	}

	s.store.Update(smokeTestID, func(status *Status) {
		status.TelemetryUploaded = true
		status.TelemetryUploadError = ""
		status.Logs = append(status.Logs, fmt.Sprintf("Telemetry fallback uploaded %d events from %s", ingested, telemetryPath))
	})
	s.logger.Info("smoke_test_telemetry_fallback",
		"smoke_test_id", smokeTestID,
		"telemetry_path", telemetryPath,
		"events_found", len(events),
		"events_ingested", ingested,
	)
}
