package smoketest

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"scenario-to-desktop-api/procmetrics"
	"scenario-to-desktop-api/screenrecording"
	"scenario-to-desktop-api/shared/errors"
)

// DefaultDemoHoldMs is the default duration (in milliseconds) to hold the
// demo app visible for screen recording.
const DefaultDemoHoldMs = 8000

// recordingState holds the active screen recording context during a smoke test.
type recordingState struct {
	captureID     string
	displayID     string
	displayWidth  int
	displayHeight int
	cleanup       func()
}

// PerformSmokeTest runs a smoke test on a built application.
func (s *DefaultService) PerformSmokeTest(ctx context.Context, smokeTestID, scenarioName, artifactPath, platform string) {
	if _, ok := s.store.Get(smokeTestID); !ok {
		return
	}

	s.transitionTo(smokeTestID, StateInitializing, fmt.Sprintf("scenario=%s platform=%s", scenarioName, platform))

	s.transitionTo(smokeTestID, StateValidatingArtifact, artifactPath)
	if !s.validatePreconditions(ctx, smokeTestID, artifactPath, platform) {
		return
	}
	defer s.cancelManager.Clear(smokeTestID)
	defer s.recoverFromPanic(smokeTestID)

	s.store.Update(smokeTestID, func(status *Status) {
		status.Logs = append(status.Logs, fmt.Sprintf(
			"Starting smoke test for %s on %s (artifact: %s)",
			scenarioName, platform, filepath.Base(artifactPath),
		))
	})

	rec := s.setupScreenRecording(ctx, smokeTestID)
	if rec.cleanup != nil {
		defer rec.cleanup()
	}

	// Resolve command
	s.transitionTo(smokeTestID, StateResolvingCommand, platform)
	cmd, args, displayCommand, err := s.resolveTestCommand(smokeTestID, platform, artifactPath, rec.captureID)
	if err != nil {
		s.recordTypedFailure(smokeTestID, NewPlatformError("artifact not runnable", err, platform))
		return
	}

	// Execute smoke test with retry support
	s.transitionTo(smokeTestID, StateExecuting, displayCommand)
	execResult, execErr := s.executeWithRetry(ctx, smokeTestID, artifactPath, cmd, args, displayCommand, rec.displayID, rec.displayWidth, rec.displayHeight)

	// Demo launch for screen recording if smoke test passed
	if rec.captureID != "" && rec.displayID != "" && execErr == nil && execResult != nil {
		if strings.Contains(execResult.Combined, s.config.SuccessMarker) {
			s.executeDemoLaunch(ctx, smokeTestID, artifactPath, platform, rec.displayID)
		}
	}

	s.finalizeRecording(ctx, smokeTestID, rec.captureID)

	// Process results
	outputLen := 0
	if execResult != nil {
		outputLen = len(execResult.Combined)
	}
	s.transitionTo(smokeTestID, StateParsingOutput, fmt.Sprintf("%d bytes of output", outputLen))
	s.processResults(smokeTestID, scenarioName, platform, artifactPath, displayCommand, execResult, execErr)
}

// setupScreenRecording initializes screen recording if configured. Returns a
// recordingState with zero values when recording is not enabled.
func (s *DefaultService) setupScreenRecording(ctx context.Context, smokeTestID string) recordingState {
	var recordingCfg *ScreenRecordingConfig
	if status, ok := s.store.Get(smokeTestID); ok {
		recordingCfg = status.RecordingConfig
	}

	if recordingCfg == nil || !recordingCfg.Enabled || s.recorder == nil || s.displayMgr == nil {
		return recordingState{}
	}

	width, height := recordingCfg.DisplayWidth, recordingCfg.DisplayHeight
	if width == 0 {
		width = 1920
	}
	if height == 0 {
		height = 1080
	}
	fps := recordingCfg.FPS
	if fps == 0 {
		fps = 15
	}

	s.logger.Info("screen_recording_setup",
		"smoke_test_id", smokeTestID,
		"width", width, "height", height, "fps", fps,
	)

	displayID, displayCleanup, err := s.displayMgr.CreateDisplay(width, height)
	if err != nil {
		s.logger.Warn("screen_recording_display_failed", "smoke_test_id", smokeTestID, "error", err.Error())
		s.store.Update(smokeTestID, func(status *Status) {
			status.ScreenRecording = &ScreenRecordingResult{Error: fmt.Sprintf("display creation failed: %v", err)}
		})
		return recordingState{}
	}

	cID, err := s.recorder.StartCapture(ctx, screenrecording.CaptureConfig{
		Display:    displayID,
		Width:      width,
		Height:     height,
		FPS:        fps,
		OutputPath: "", // let FFmpeg resource choose path
	})
	if err != nil {
		s.logger.Warn("screen_recording_start_failed", "smoke_test_id", smokeTestID, "error", err.Error())
		s.store.Update(smokeTestID, func(status *Status) {
			status.ScreenRecording = &ScreenRecordingResult{Error: fmt.Sprintf("capture start failed: %v", err)}
		})
		displayCleanup()
		return recordingState{}
	}

	s.store.Update(smokeTestID, func(status *Status) {
		status.Logs = append(status.Logs, fmt.Sprintf("Screen recording started (ID: %s, display: %s)", cID, displayID))
	})

	return recordingState{
		captureID:     cID,
		displayID:     displayID,
		displayWidth:  width,
		displayHeight: height,
		cleanup:       displayCleanup,
	}
}

// resolveTestCommand resolves the command to run, using either the direct resolver
// (when recording manages the display) or the headless-wrapper-aware resolver.
func (s *DefaultService) resolveTestCommand(smokeTestID, platform, artifactPath, captureID string) (string, []string, string, error) {
	if captureID != "" {
		return s.platformResolver.ResolveCommand(platform, artifactPath)
	}
	return s.resolveCommand(smokeTestID, platform, artifactPath)
}

// finalizeRecording stops the screen recording and stores the result.
func (s *DefaultService) finalizeRecording(ctx context.Context, smokeTestID, captureID string) {
	if captureID == "" {
		return
	}

	captureResult, err := s.recorder.StopCapture(ctx, captureID)
	if err != nil {
		s.logger.Warn("screen_recording_stop_failed", "smoke_test_id", smokeTestID, "error", err.Error())
		s.store.Update(smokeTestID, func(status *Status) {
			status.ScreenRecording = &ScreenRecordingResult{Recorded: false, Error: fmt.Sprintf("capture stop failed: %v", err)}
		})
		return
	}

	s.store.Update(smokeTestID, func(status *Status) {
		status.ScreenRecording = &ScreenRecordingResult{
			Recorded:      true,
			VideoPath:     captureResult.VideoPath,
			DurationMs:    captureResult.DurationMs,
			FileSizeBytes: captureResult.FileSizeBytes,
		}
		status.Logs = append(status.Logs, fmt.Sprintf("Screen recording saved: %s", captureResult.VideoPath))
	})
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

func (s *DefaultService) executeSmokeTest(ctx context.Context, smokeTestID, artifactPath, cmd string, args []string, displayCommand, displayID string, displayWidth, displayHeight int) (*ExecutionResult, error) {
	// Build environment
	uploadURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/deployment/telemetry", s.port)
	env := []string{
		"SMOKE_TEST=1",
		fmt.Sprintf("SMOKE_TEST_TIMEOUT_MS=%d", s.config.TimeoutMS()),
		fmt.Sprintf("SMOKE_TEST_UPLOAD_URL=%s", uploadURL),
	}

	// When screen recording manages the display, tell Electron to render on it.
	if displayID != "" {
		env = append(env, fmt.Sprintf("DISPLAY=%s", displayID))
	}

	// Set up process monitor if factory is available.
	var monitor procmetrics.Monitor
	s.installMonitorHook(displayID, displayWidth, displayHeight, func(m procmetrics.Monitor) { monitor = m })

	// Execute
	workDir := filepath.Dir(artifactPath)
	result, err := s.executor.ExecuteWithResult(ctx, workDir, cmd, args, env, s.config.Timeout())

	// Harvest process metrics.
	s.harvestMonitor(monitor, smokeTestID)
	s.clearMonitorHook()

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

// executeDemoLaunch runs the app in normal mode for screen recording.
// This is purely for capturing a realistic startup experience on video.
// Errors are logged but do not affect the smoke test pass/fail result.
func (s *DefaultService) executeDemoLaunch(ctx context.Context, smokeTestID, artifactPath, platform, displayID string) {
	s.logger.Info("demo_launch_starting", "smoke_test_id", smokeTestID, "display", displayID)
	s.store.Update(smokeTestID, func(status *Status) {
		status.Logs = append(status.Logs, "Starting demo launch for screen recording...")
	})

	cmd, args, _, err := s.platformResolver.ResolveCommand(platform, artifactPath)
	if err != nil {
		s.logger.Warn("demo_launch_resolve_failed", "smoke_test_id", smokeTestID, "error", err.Error())
		return
	}

	// Strip --smoke-test from args so the app launches in normal mode
	args = StripSmokeTestFlag(args)

	env := []string{
		fmt.Sprintf("DISPLAY=%s", displayID),
		"SMOKE_TEST_DEMO=1",
		fmt.Sprintf("SMOKE_TEST_DEMO_HOLD_MS=%d", DefaultDemoHoldMs),
	}

	demoTimeout := 90 * time.Second
	result, err := s.executor.ExecuteWithResult(ctx, filepath.Dir(artifactPath), cmd, args, env, demoTimeout)

	if err != nil {
		s.logger.Warn("demo_launch_failed", "smoke_test_id", smokeTestID, "error", err.Error())
		s.store.Update(smokeTestID, func(status *Status) {
			status.Logs = append(status.Logs, fmt.Sprintf("Demo launch error (non-fatal): %v", err))
		})
	} else {
		exitCode := 0
		if result != nil {
			exitCode = result.ExitCode
		}
		s.logger.Info("demo_launch_completed", "smoke_test_id", smokeTestID, "exit_code", exitCode)
		s.store.Update(smokeTestID, func(status *Status) {
			status.Logs = append(status.Logs, fmt.Sprintf("Demo launch completed (exit code: %d)", exitCode))
		})
	}
}

// StripSmokeTestFlag removes "--smoke-test" from args so the app launches
// in normal mode for demo recording.
func StripSmokeTestFlag(args []string) []string {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		if arg != "--smoke-test" {
			result = append(result, arg)
		}
	}
	return result
}

// truncateOutput limits output to the specified maximum length.
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + fmt.Sprintf("... (%d bytes truncated)", len(s)-maxLen)
}

// executeWithRetry wraps executeSmokeTest with retry logic for recoverable errors.
func (s *DefaultService) executeWithRetry(ctx context.Context, smokeTestID, artifactPath, cmd string, args []string, displayCommand, displayID string, displayWidth, displayHeight int) (*ExecutionResult, error) {
	var lastResult *ExecutionResult

	for attempt := 0; ; attempt++ {
		result, err := s.executeSmokeTest(ctx, smokeTestID, artifactPath, cmd, args, displayCommand, displayID, displayWidth, displayHeight)
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

// installMonitorHook sets up the PID callback on the executor to start a monitor.
// The onCreated callback receives the monitor once started. No-op if monitorFactory is nil
// or executor is not a *DefaultProcessExecutor.
// expectedWidth/expectedHeight are the display dimensions for size-based window detection.
func (s *DefaultService) installMonitorHook(displayID string, expectedWidth, expectedHeight int, onCreated func(procmetrics.Monitor)) {
	if s.monitorFactory == nil {
		return
	}
	dpe, ok := s.executor.(*DefaultProcessExecutor)
	if !ok {
		return
	}
	dpe.onProcessStartedPID = func(pid int) {
		m := s.monitorFactory.NewMonitor()
		if err := m.Start(context.Background(), pid, displayID, expectedWidth, expectedHeight); err != nil {
			s.logger.Warn("failed to start process monitor", "pid", pid, "error", err)
			return
		}
		onCreated(m)
	}
}

// clearMonitorHook removes the PID callback from the executor.
func (s *DefaultService) clearMonitorHook() {
	if dpe, ok := s.executor.(*DefaultProcessExecutor); ok {
		dpe.onProcessStartedPID = nil
	}
}

// harvestMonitor stops a running monitor and writes its metrics into the smoke test status.
func (s *DefaultService) harvestMonitor(monitor procmetrics.Monitor, smokeTestID string) {
	if monitor == nil {
		return
	}
	monitor.Stop()
	report := monitor.Report()
	if report == nil {
		return
	}
	s.store.Update(smokeTestID, func(status *Status) {
		status.SplashDurationMs = report.Startup.SplashDurationMs
		status.ReadyDurationMs = report.Startup.ReadyMs
		status.ResourceSummary = report.Summary
	})
}
