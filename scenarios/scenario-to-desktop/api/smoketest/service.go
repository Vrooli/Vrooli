// DOC: docs/reference/smoke-test-pipeline.md
package smoketest

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"path/filepath"
	"strings"
	"time"

	"scenario-to-desktop-api/shared/errors"
)

// DefaultService is the default implementation of Service.
// See docs/reference/smoke-test-pipeline.md for execution flow and configuration.
type DefaultService struct {
	store             Store
	cancelManager     CancelManager
	telemetryIngestor TelemetryIngestor
	port              int
	logger            Logger

	// New injected components
	config            Config
	executor          ProcessExecutor
	platformResolver  PlatformResolver
	telemetryResolver TelemetryPathResolver
	outputParser      OutputParser
	fileSystem        FileSystem
	prereqChecker     PrerequisiteCheckerI
	envReader         EnvironmentReader
}

// NewService creates a new smoke test service with all required dependencies.
func NewService(
	store Store,
	cancelManager CancelManager,
	telemetryIngestor TelemetryIngestor,
	config Config,
	executor ProcessExecutor,
	platformResolver PlatformResolver,
	telemetryResolver TelemetryPathResolver,
	outputParser OutputParser,
	fileSystem FileSystem,
	logger Logger,
	port int,
) *DefaultService {
	return &DefaultService{
		store:             store,
		cancelManager:     cancelManager,
		telemetryIngestor: telemetryIngestor,
		config:            config,
		executor:          executor,
		platformResolver:  platformResolver,
		telemetryResolver: telemetryResolver,
		outputParser:      outputParser,
		fileSystem:        fileSystem,
		logger:            logger,
		port:              port,
	}
}

// NewDefaultSmokeTestService creates a new smoke test service with default implementations.
// This is the factory function for production wiring.
func NewDefaultSmokeTestService(
	store Store,
	cancelManager CancelManager,
	telemetryIngestor TelemetryIngestor,
	port int,
	logger Logger,
) *DefaultService {
	config := DefaultConfig()
	envReader := NewEnvironmentReader()
	fs := NewFileSystem()
	executor := NewProcessExecutorWithLimit(logger, config.MaxOutputBytes)
	platformResolver := NewPlatformResolver(executor, config, envReader, fs)
	telemetryResolver := NewTelemetryPathResolver(config, envReader, fs)
	outputParser := NewOutputParser(config)
	prereqChecker := NewPrerequisiteChecker(envReader, fs, executor)

	return &DefaultService{
		store:             store,
		cancelManager:     cancelManager,
		telemetryIngestor: telemetryIngestor,
		config:            config,
		executor:          executor,
		platformResolver:  platformResolver,
		telemetryResolver: telemetryResolver,
		outputParser:      outputParser,
		fileSystem:        fs,
		logger:            logger,
		port:              port,
		prereqChecker:     prereqChecker,
		envReader:         envReader,
	}
}

// CurrentPlatform returns the current platform identifier.
func (s *DefaultService) CurrentPlatform() string {
	return s.platformResolver.CurrentPlatform()
}

// PerformSmokeTest runs a smoke test on a built application.
func (s *DefaultService) PerformSmokeTest(ctx context.Context, smokeTestID, scenarioName, artifactPath, platform string) {
	// Check if smoke test exists before doing anything
	if _, ok := s.store.Get(smokeTestID); !ok {
		return
	}

	s.transitionTo(smokeTestID, StateInitializing, fmt.Sprintf("scenario=%s platform=%s", scenarioName, platform))

	// Validate preconditions
	s.transitionTo(smokeTestID, StateValidatingArtifact, artifactPath)
	if !s.validatePreconditions(ctx, smokeTestID, artifactPath, platform) {
		return
	}
	defer s.cancelManager.Clear(smokeTestID)

	// Panic recovery
	defer s.recoverFromPanic(smokeTestID)

	// Log start
	s.store.Update(smokeTestID, func(status *Status) {
		status.Logs = append(status.Logs, fmt.Sprintf(
			"Starting smoke test for %s on %s (artifact: %s)",
			scenarioName, platform, filepath.Base(artifactPath),
		))
	})

	// Resolve command
	s.transitionTo(smokeTestID, StateResolvingCommand, platform)
	cmd, args, displayCommand, err := s.resolveCommand(smokeTestID, platform, artifactPath)
	if err != nil {
		s.recordTypedFailure(smokeTestID, NewPlatformError(
			"artifact not runnable",
			err,
			platform,
		))
		return
	}

	// Execute smoke test with retry support
	s.transitionTo(smokeTestID, StateExecuting, displayCommand)
	execResult, execErr := s.executeWithRetry(ctx, smokeTestID, artifactPath, cmd, args, displayCommand)

	// Process results
	outputLen := 0
	if execResult != nil {
		outputLen = len(execResult.Combined)
	}
	s.transitionTo(smokeTestID, StateParsingOutput, fmt.Sprintf("%d bytes of output", outputLen))
	s.processResults(smokeTestID, scenarioName, platform, artifactPath, displayCommand, execResult, execErr)
}

func (s *DefaultService) validatePreconditions(ctx context.Context, smokeTestID, artifactPath, platform string) bool {
	// Check for cancellation
	if ctx.Err() != nil {
		s.recordTypedFailure(smokeTestID, NewCancelledError("smoke test cancelled"))
		return false
	}

	// Check artifact exists (fast path - don't run full prereqs if artifact missing)
	if _, err := s.fileSystem.Stat(artifactPath); err != nil {
		s.recordTypedFailure(smokeTestID, NewArtifactError(
			"artifact not found",
			err,
			artifactPath,
		))
		return false
	}

	// Run prerequisite checks if checker is available
	if s.prereqChecker != nil {
		s.transitionTo(smokeTestID, StateValidatingPrereqs, "checking system prerequisites")
		results := s.prereqChecker.CheckAll(artifactPath, platform, s.port)

		// Log all results
		for _, r := range results {
			logMsg := fmt.Sprintf("[prereq:%s] %s", r.Kind, r.Message)
			if !r.Passed && r.Suggestion != "" {
				logMsg += fmt.Sprintf(" (suggestion: %s)", r.Suggestion)
			}
			s.store.Update(smokeTestID, func(status *Status) {
				status.Logs = append(status.Logs, logMsg)
			})

			if !r.Passed {
				s.logger.Warn("smoke_test_prerequisite_failed",
					"smoke_test_id", smokeTestID,
					"kind", r.Kind.String(),
					"message", r.Message,
					"fatal", r.Fatal,
				)
			}
		}

		// Check for fatal failures
		if s.prereqChecker.HasFatalFailure(results) {
			// Find the first fatal failure for error reporting
			for _, r := range results {
				if !r.Passed && r.Fatal {
					s.recordTypedFailure(smokeTestID, &Error{
						Kind:            ErrKindArtifact,
						Message:         r.Message,
						Recoverable:     false,
						SuggestedAction: r.Suggestion,
						ManualSteps:     []string{r.Suggestion},
					})
					return false
				}
			}
		}
	}

	return true
}

func (s *DefaultService) recoverFromPanic(smokeTestID string) {
	if r := recover(); r != nil {
		kind := ErrKindExecution
		s.store.Update(smokeTestID, func(status *Status) {
			status.Status = "failed"
			status.Error = fmt.Sprintf("panic: %v", r)
			status.ErrorKind = &kind
			status.SuggestedAction = RecoveryPaths[kind]
			status.CurrentState = StateFailed
			now := time.Now()
			status.CompletedAt = &now
		})
		s.logger.Error("smoke_test_panic",
			"smoke_test_id", smokeTestID,
			"panic", fmt.Sprintf("%v", r),
		)
	}
}

func (s *DefaultService) resolveCommand(smokeTestID, platform, artifactPath string) (string, []string, string, error) {
	cmd, args, display, err := s.platformResolver.ResolveCommand(platform, artifactPath)
	if err != nil {
		return "", nil, "", err
	}

	// Check if headless wrapper is needed
	needed, wrapperCmd, wrapperArgs, wrapperErr := s.platformResolver.RequiresHeadlessWrapper()
	if wrapperErr != nil {
		s.store.Update(smokeTestID, func(status *Status) {
			status.Logs = append(status.Logs, fmt.Sprintf(
				"Smoke test failed: %s", wrapperErr.Error(),
			))
		})
		return "", nil, "", wrapperErr
	}

	if needed {
		// Prepend wrapper command and args
		args = append(append(wrapperArgs, cmd), args...)
		cmd = wrapperCmd
		display = fmt.Sprintf("%s %s %s", wrapperCmd, wrapperArgs[0], display)
	}

	return cmd, args, display, nil
}

func (s *DefaultService) executeSmokeTest(ctx context.Context, smokeTestID, artifactPath, cmd string, args []string, displayCommand string) (*ExecutionResult, error) {
	// Build environment
	uploadURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/deployment/telemetry", s.port)
	env := []string{
		"SMOKE_TEST=1",
		fmt.Sprintf("SMOKE_TEST_TIMEOUT_MS=%d", s.config.TimeoutMS()),
		fmt.Sprintf("SMOKE_TEST_UPLOAD_URL=%s", uploadURL),
	}

	// Execute
	workDir := filepath.Dir(artifactPath)
	result, err := s.executor.ExecuteWithResult(ctx, workDir, cmd, args, env, s.config.Timeout())

	// Store execution details for debugging
	if result != nil {
		s.store.Update(smokeTestID, func(status *Status) {
			status.LastStdout = truncateOutput(result.Stdout, 10000)
			status.LastStderr = truncateOutput(result.Stderr, 10000)
			status.OutputTruncated = result.Truncated
		})
	}

	// Log execution result
	s.logExecutionResultWithDetails(smokeTestID, displayCommand, result, err)

	return result, err
}

// truncateOutput limits output to the specified maximum length.
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + fmt.Sprintf("... (%d bytes truncated)", len(s)-maxLen)
}

// executeWithRetry wraps executeSmokeTest with retry logic for recoverable errors.
func (s *DefaultService) executeWithRetry(ctx context.Context, smokeTestID, artifactPath, cmd string, args []string, displayCommand string) (*ExecutionResult, error) {
	var lastResult *ExecutionResult

	for attempt := 0; ; attempt++ {
		result, err := s.executeSmokeTest(ctx, smokeTestID, artifactPath, cmd, args, displayCommand)
		lastResult = result

		if err == nil {
			return result, nil
		}

		// Check if this is a retryable error
		retryStrategy := s.getRetryStrategy(err)
		if retryStrategy == nil {
			// Non-retryable error
			return result, err
		}

		if attempt >= retryStrategy.MaxAttempts-1 {
			// Max attempts reached
			s.logger.Warn("smoke_test_retry_exhausted",
				"smoke_test_id", smokeTestID,
				"attempts", attempt+1,
				"error", err.Error(),
			)
			return result, err
		}

		// Calculate backoff
		backoff := time.Duration(retryStrategy.BackoffMs) * time.Millisecond
		multiplier := retryStrategy.BackoffMultiplier
		if multiplier == 0 {
			multiplier = 2.0
		}
		backoff = time.Duration(float64(backoff) * math.Pow(multiplier, float64(attempt)))

		s.transitionTo(smokeTestID, StateRetrying, fmt.Sprintf("attempt %d/%d, backoff %v", attempt+2, retryStrategy.MaxAttempts, backoff))
		s.store.Update(smokeTestID, func(status *Status) {
			status.RetryCount = attempt + 1
			status.Logs = append(status.Logs, fmt.Sprintf("Retrying after error: %v (attempt %d/%d)", err, attempt+2, retryStrategy.MaxAttempts))
		})

		s.logger.Info("smoke_test_retrying",
			"smoke_test_id", smokeTestID,
			"attempt", attempt+2,
			"max_attempts", retryStrategy.MaxAttempts,
			"backoff_ms", backoff.Milliseconds(),
			"error", err.Error(),
		)

		select {
		case <-ctx.Done():
			return lastResult, ctx.Err()
		case <-time.After(backoff):
			s.transitionTo(smokeTestID, StateExecuting, fmt.Sprintf("retry attempt %d", attempt+2))
		}
	}
}

// getRetryStrategy returns the retry strategy for an error, or nil if not retryable.
func (s *DefaultService) getRetryStrategy(err error) *errors.RetryStrategy {
	// Check for timeout errors (retryable)
	if strings.Contains(err.Error(), "timed out") {
		return errors.RetryConservative
	}

	// Check for temporary execution errors (retryable)
	if strings.Contains(err.Error(), "resource temporarily unavailable") ||
		strings.Contains(err.Error(), "text file busy") {
		return errors.RetryDefault
	}

	// Not retryable
	return nil
}

func (s *DefaultService) logExecutionResultWithDetails(smokeTestID, displayCommand string, result *ExecutionResult, err error) {
	logEntry := fmt.Sprintf("[smoke-test] %s", displayCommand)
	if err != nil {
		logEntry += fmt.Sprintf("\nFAILED: %v", err)
	} else {
		logEntry += "\nSUCCESS"
	}

	if result != nil {
		// Add exit code
		logEntry += fmt.Sprintf("\nExit code: %d", result.ExitCode)

		// Log stdout
		stdout := result.Stdout
		if len(stdout) > 0 {
			if len(stdout) < 500 {
				logEntry += fmt.Sprintf("\nStdout: %s", stdout)
			} else {
				logEntry += fmt.Sprintf("\nStdout: %s... (%d bytes)", stdout[:500], len(stdout))
			}
		}

		// Log stderr separately for debugging
		stderr := result.Stderr
		if len(stderr) > 0 {
			if len(stderr) < 500 {
				logEntry += fmt.Sprintf("\nStderr: %s", stderr)
			} else {
				logEntry += fmt.Sprintf("\nStderr: %s... (%d bytes)", stderr[:500], len(stderr))
			}
		}

		// Note truncation
		if result.Truncated {
			logEntry += fmt.Sprintf("\n[Output truncated: %d bytes exceeded limit]", result.TruncatedBytes)
		}
	}

	s.store.Update(smokeTestID, func(status *Status) {
		status.Logs = append(status.Logs, logEntry)
	})
}

func (s *DefaultService) processResults(smokeTestID, scenarioName, platform, artifactPath, displayCommand string, execResult *ExecutionResult, execErr error) {
	// Get the combined output for parsing
	output := ""
	if execResult != nil {
		output = execResult.Combined
	}

	// Check for app-reported structured errors first (highest priority)
	if appError := s.outputParser.ExtractAppError(output); appError != nil {
		s.recordTypedFailure(smokeTestID, NewAppReportedError(appError))
		return
	}

	// Extract last lifecycle state for error context
	lastState := s.outputParser.ExtractLastLifecycleState(output)

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
		s.attemptTelemetryFallback(smokeTestID, scenarioName, platform, artifactPath, output)
	}

	// Check execution error
	if execErr != nil {
		smokeErr := s.buildExecutionError(execErr, execResult, displayCommand, platform)
		// Add lifecycle state to error context for debugging
		if lastState != "" {
			smokeErr.Context["last_lifecycle_state"] = lastState
		}
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
func (s *DefaultService) buildExecutionError(execErr error, execResult *ExecutionResult, displayCommand, platform string) *Error {
	err := NewExecutionError(
		"smoke test process failed",
		execErr,
		map[string]string{
			"command":  displayCommand,
			"platform": platform,
		},
	)

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

func (s *DefaultService) attemptTelemetryFallback(smokeTestID, scenarioName, platform, artifactPath, output string) {
	if s.telemetryIngestor == nil {
		return
	}

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

func (s *DefaultService) recordTypedFailure(smokeTestID string, err *Error) {
	s.transitionTo(smokeTestID, StateFailed, err.Message)
	s.store.Update(smokeTestID, func(status *Status) {
		status.Status = "failed"
		status.Error = err.Error()
		status.ErrorKind = &err.Kind
		status.ErrorContext = err.Context
		status.SuggestedAction = err.SuggestedAction
		status.Logs = append(status.Logs, fmt.Sprintf("FAILED: %s", err.Message))
		now := time.Now()
		status.CompletedAt = &now
	})

	s.logger.Error("smoke_test_failed",
		"smoke_test_id", smokeTestID,
		"error_kind", err.Kind.String(),
		"error", err.Error(),
		"recoverable", err.Recoverable,
	)
}

func (s *DefaultService) transitionTo(smokeTestID string, newState State, message string) {
	s.store.Update(smokeTestID, func(status *Status) {
		now := time.Now()
		var durationMs int64
		if len(status.Transitions) > 0 {
			lastTransition := status.Transitions[len(status.Transitions)-1]
			durationMs = now.Sub(lastTransition.Timestamp).Milliseconds()
		}
		transition := StateTransition{
			From:       status.CurrentState,
			To:         newState,
			Timestamp:  now,
			Message:    message,
			DurationMs: durationMs,
		}
		status.Transitions = append(status.Transitions, transition)
		status.CurrentState = newState
		status.Logs = append(status.Logs, fmt.Sprintf("[%s] %s", newState, message))
	})

	s.logger.Info("smoke_test_state_transition",
		"smoke_test_id", smokeTestID,
		"state", string(newState),
		"message", message,
	)
}

// SlogAdapter adapts slog.Logger to the Logger interface.
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new slog adapter.
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	return &SlogAdapter{logger: logger}
}

// Info logs an info message.
func (a *SlogAdapter) Info(msg string, args ...interface{}) {
	a.logger.Info(msg, args...)
}

// Warn logs a warning message.
func (a *SlogAdapter) Warn(msg string, args ...interface{}) {
	a.logger.Warn(msg, args...)
}

// Error logs an error message.
func (a *SlogAdapter) Error(msg string, args ...interface{}) {
	a.logger.Error(msg, args...)
}
