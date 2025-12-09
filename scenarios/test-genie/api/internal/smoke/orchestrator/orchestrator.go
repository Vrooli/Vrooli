package orchestrator

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// Orchestrator coordinates the UI smoke test workflow.
type Orchestrator struct {
	config              Config
	preflight           PreflightChecker
	healthChecker       HealthChecker
	browser             BrowserClient
	artifacts           ArtifactWriter
	handshake           HandshakeDetector
	logger              io.Writer
	payloadGen          PayloadGenerator
	autoRecoveryEnabled bool
	autoRecoveryOpts    AutoRecoveryOptions
	scenarioStarter     ScenarioStarter
	autoStartedScenario string // Track if we auto-started to cleanup on exit
}

// New creates a new Orchestrator with the given configuration and options.
func New(config Config, opts ...Option) *Orchestrator {
	o := &Orchestrator{
		config: config,
		logger: io.Discard,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Run executes the UI smoke test workflow.
func (o *Orchestrator) Run(ctx context.Context) (*Result, error) {
	startTime := time.Now()

	// Validate configuration
	if err := o.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	o.log("Starting UI smoke test for %s", o.config.ScenarioName)

	// Step 1: Check if UI directory exists
	if o.preflight != nil && !o.preflight.CheckUIDirectory(o.config.ScenarioDir) {
		o.log("No UI directory detected, skipping smoke test")
		result := Skipped(o.config.ScenarioName, "UI directory not detected")
		o.persistResult(ctx, result)
		return result, nil
	}

	// Step 2: Check browserless availability with auto-recovery
	if o.preflight != nil {
		browserlessErr := o.preflight.CheckBrowserless(ctx)
		if browserlessErr != nil {
			o.log("Browserless check failed: %v", browserlessErr)

			// Attempt auto-recovery if enabled and health checker is available
			if o.autoRecoveryEnabled && o.healthChecker != nil {
				o.log("Attempting auto-recovery...")
				opts := o.autoRecoveryOpts
				if opts.ContainerName == "" {
					opts = DefaultAutoRecoveryOptions()
				}

				recoveryResult, recoveryErr := o.healthChecker.EnsureHealthy(ctx, opts)
				if recoveryErr == nil && recoveryResult.Success {
					o.log("Auto-recovery succeeded: %s", recoveryResult.Action)
					// Retry the browserless check
					browserlessErr = o.preflight.CheckBrowserless(ctx)
				} else {
					if recoveryResult != nil && recoveryResult.Attempted {
						o.log("Auto-recovery failed: %s", recoveryResult.Error)
					}
				}
			}

			// If still failing, generate detailed diagnosis
			if browserlessErr != nil {
				var message string
				if o.healthChecker != nil {
					diagnosis := o.healthChecker.DiagnoseBrowserlessFailure(ctx, o.config.ScenarioName)
					message = fmt.Sprintf("%s\n\n%s", diagnosis.Message, diagnosis.Recommendation)
				} else {
					message = "Browserless resource is offline. Run 'resource-browserless manage start' then rerun the smoke test."
				}

				result := Blocked(o.config.ScenarioName, message, BlockedReasonBrowserlessOffline)
				o.persistResult(ctx, result)
				return result, nil
			}
		}
		o.log("Browserless is available")
	}

	// Step 3: Check bundle freshness
	var bundleStatus *BundleStatus
	if o.preflight != nil {
		var err error
		bundleStatus, err = o.preflight.CheckBundleFreshness(ctx, o.config.ScenarioDir)
		if err != nil {
			o.log("Bundle freshness check failed: %v", err)
		}
		if bundleStatus != nil && !bundleStatus.Fresh {
			o.log("UI bundle is stale: %s", bundleStatus.Reason)
			result := Blocked(o.config.ScenarioName,
				fmt.Sprintf("%s\n  ↳ Fix: vrooli scenario restart %s\n  ↳ Then verify: vrooli scenario ui-smoke %s",
					bundleStatus.Reason, o.config.ScenarioName, o.config.ScenarioName),
				BlockedReasonBundleStale)
			result.Bundle = bundleStatus
			o.persistResult(ctx, result)
			return result, nil
		}
		o.log("UI bundle is fresh")
	}

	// Step 4: Check if UI port is defined in service.json
	var uiPortDef *UIPortDefinition
	if o.preflight != nil {
		var err error
		uiPortDef, err = o.preflight.CheckUIPortDefined(o.config.ScenarioDir)
		if err != nil {
			o.log("UI port definition check failed: %v", err)
		}
	}

	// Step 5: Discover UI port
	uiURL := o.config.ResolveUIURL()
	if uiURL == "" && o.preflight != nil {
		port, err := o.preflight.CheckUIPort(ctx, o.config.ScenarioName)
		if err != nil {
			o.log("UI port discovery failed: %v", err)
		}
		if port > 0 {
			uiURL = fmt.Sprintf("http://localhost:%d", port)
			o.log("Discovered UI port: %d", port)
		}
	}

	if uiURL == "" {
		// If service.json defines a UI port but we couldn't detect it, the scenario may not be running
		if uiPortDef != nil && uiPortDef.Defined {
			o.log("UI port defined in service.json (%s) but not detected - scenario may not be running", uiPortDef.EnvVar)

			// Attempt auto-start if enabled
			if o.config.AutoStart && o.scenarioStarter != nil {
				o.log("Attempting auto-start of scenario %s...", o.config.ScenarioName)
				port, startErr := o.scenarioStarter.Start(ctx, o.config.ScenarioName)
				if startErr != nil {
					o.log("Auto-start failed: %v", startErr)
				} else {
					o.autoStartedScenario = o.config.ScenarioName // Track for cleanup
					uiURL = fmt.Sprintf("http://localhost:%d", port)
					o.log("Auto-start succeeded, UI available at %s", uiURL)
				}
			}

			// If still no UI URL after auto-start attempt, return blocked
			if uiURL == "" {
				var message string
				if o.config.AutoStart {
					message = fmt.Sprintf("UI port is defined in service.json (%s) but not detected.\n"+
						"  ↳ Auto-start was attempted but failed.\n"+
						"  ↳ Fix: vrooli scenario restart %s\n"+
						"  ↳ Then check: vrooli scenario logs %s --step start-ui",
						uiPortDef.EnvVar, o.config.ScenarioName, o.config.ScenarioName)
				} else {
					message = fmt.Sprintf("UI port is defined in service.json (%s) but not detected.\n"+
						"  ↳ The scenario may not be running or the UI server failed to start.\n"+
						"  ↳ Fix: vrooli scenario restart %s\n"+
						"  ↳ Or use: --auto-start to automatically start the scenario\n"+
						"  ↳ Then check: vrooli scenario logs %s --step start-ui",
						uiPortDef.EnvVar, o.config.ScenarioName, o.config.ScenarioName)
				}
				result := Blocked(o.config.ScenarioName, message, BlockedReasonUIPortMissing)
				o.persistResult(ctx, result)
				return result, nil
			}
		} else {
			// No UI port defined - this scenario genuinely has no UI
			o.log("No UI port defined in service.json, skipping smoke test")
			result := Skipped(o.config.ScenarioName, "Scenario does not define a UI port")
			o.persistResult(ctx, result)
			return result, nil
		}
	}

	// Step 6: Check iframe-bridge dependency
	var bridgeStatus *BridgeStatus
	if o.preflight != nil {
		var err error
		bridgeStatus, err = o.preflight.CheckIframeBridge(ctx, o.config.ScenarioDir)
		if err != nil {
			o.log("iframe-bridge check failed: %v", err)
		}
		if bridgeStatus != nil && !bridgeStatus.DependencyPresent {
			o.log("iframe-bridge dependency missing")
			result := Failed(o.config.ScenarioName,
				"@vrooli/iframe-bridge dependency missing in ui/package.json")
			result.IframeBridge = bridgeStatus
			o.persistResult(ctx, result)
			return result, nil
		}
		o.log("iframe-bridge dependency present")
	}

	// Step 7: Execute browser session
	if o.browser == nil {
		return nil, fmt.Errorf("browser client not configured")
	}
	if o.payloadGen == nil {
		return nil, fmt.Errorf("payload generator not configured")
	}

	payload := o.payloadGen.Generate(uiURL, o.config.Timeout, o.config.HandshakeTimeout, o.config.Viewport, o.config.HandshakeSignals)
	o.log("Executing browser session for %s", uiURL)

	response, err := o.browser.ExecuteFunction(ctx, payload)
	if err != nil {
		o.log("Browser execution failed: %v", err)

		// Generate detailed diagnosis for browser execution failures
		var message string
		if o.healthChecker != nil {
			diagnosis := o.healthChecker.DiagnoseBrowserlessFailure(ctx, o.config.ScenarioName)
			message = fmt.Sprintf("Failed to reach Browserless API: %v\n\n%s\n\n%s",
				err, diagnosis.Message, diagnosis.Recommendation)
		} else {
			message = fmt.Sprintf("Failed to reach Browserless API: %v", err)
		}

		result := Failed(o.config.ScenarioName, message)
		o.persistResult(ctx, result)
		return result, nil
	}

	// Step 8: Evaluate handshake
	var handshakeResult HandshakeResult
	if o.handshake != nil {
		handshakeResult = o.handshake.Evaluate(&response.Handshake)
	} else {
		handshakeResult = HandshakeResult{
			Signaled:   response.Handshake.Signaled,
			TimedOut:   response.Handshake.TimedOut,
			DurationMs: response.Handshake.DurationMs,
			Error:      response.Handshake.Error,
		}
	}

	// Step 9: Write artifacts
	var artifactPaths *ArtifactPaths
	if o.artifacts != nil {
		artifactPaths, err = o.artifacts.WriteAll(ctx, o.config.ScenarioDir, o.config.ScenarioName, response)
		if err != nil {
			o.log("Failed to write artifacts: %v", err)
		}
	}

	// Step 10: Build final result
	duration := time.Since(startTime)
	result := o.buildResult(response, handshakeResult, bundleStatus, bridgeStatus, artifactPaths, uiURL, duration)

	// Log additional telemetry for richer diagnostics.
	o.log("Handshake: signaled=%t timed_out=%t duration=%dms error=%s", handshakeResult.Signaled, handshakeResult.TimedOut, handshakeResult.DurationMs, handshakeResult.Error)
	if response != nil {
		o.log("UI telemetry: network_failures=%d page_errors=%d console_errors=%d storage_patches=%d", len(response.Network), len(response.PageErrors), result.ConsoleErrorCount, len(response.StorageShim))
	}
	if artifactPaths != nil {
		o.log("Artifacts: screenshot=%s console=%s network=%s html=%s raw=%s readme=%s",
			artifactPaths.Screenshot, artifactPaths.Console, artifactPaths.Network, artifactPaths.HTML, artifactPaths.Raw, artifactPaths.Readme)
	}

	// Step 11: Persist result
	o.persistResult(ctx, result)

	o.log("UI smoke test completed: %s (%dms)", result.Status, result.DurationMs)
	return result, nil
}

func (o *Orchestrator) buildResult(
	response *BrowserResponse,
	handshake HandshakeResult,
	bundle *BundleStatus,
	bridge *BridgeStatus,
	artifacts *ArtifactPaths,
	uiURL string,
	duration time.Duration,
) *Result {
	// Determine status and message
	status := StatusPassed
	message := "UI loaded successfully"

	if !response.Success {
		status = StatusFailed
		message = response.Error
		if message == "" {
			message = "Browserless execution failed"
		}
	} else if !handshake.Signaled {
		status = StatusFailed
		message = "Iframe bridge never signaled ready. See: docs/phases/structure/ui-smoke.md#handshake-timeout"
	} else if len(response.Network) > 0 {
		status = StatusFailed
		message = formatNetworkFailures(response.Network)
	} else if len(response.PageErrors) > 0 {
		status = StatusFailed
		message = fmt.Sprintf("UI exception: %s", response.PageErrors[0].Message)
	}

	// Count console.error() calls
	consoleErrorCount := 0
	for _, entry := range response.Console {
		if entry.Level == "error" {
			consoleErrorCount++
		}
	}

	result := &Result{
		Scenario:            o.config.ScenarioName,
		Status:              status,
		Message:             message,
		Timestamp:           time.Now().UTC(),
		DurationMs:          duration.Milliseconds(),
		UIURL:               uiURL,
		Handshake:           handshake,
		NetworkFailureCount: len(response.Network),
		PageErrorCount:      len(response.PageErrors),
		ConsoleErrorCount:   consoleErrorCount,
		Bundle:              bundle,
		IframeBridge:        bridge,
		StorageShim:         response.StorageShim,
		Raw:                 response.Raw,
	}

	if artifacts != nil {
		result.Artifacts = *artifacts
	}

	return result
}

func (o *Orchestrator) persistResult(ctx context.Context, result *Result) {
	if o.artifacts != nil {
		if err := o.artifacts.WriteResultJSON(ctx, o.config.ScenarioDir, o.config.ScenarioName, result); err != nil {
			o.log("Failed to persist result: %v", err)
		}
		readmePath, err := o.artifacts.WriteReadme(ctx, o.config.ScenarioDir, o.config.ScenarioName, result)
		if err != nil {
			o.log("Failed to write README: %v", err)
		} else {
			result.Artifacts.Readme = readmePath
		}
	}
}

func (o *Orchestrator) log(format string, args ...interface{}) {
	if o.logger != nil {
		msg := fmt.Sprintf(format, args...)
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		fmt.Fprint(o.logger, msg)
	}
}

// formatNetworkFailures creates a message describing all network failures.
func formatNetworkFailures(failures []NetworkEntry) string {
	if len(failures) == 0 {
		return ""
	}

	if len(failures) == 1 {
		return formatSingleNetworkFailure(failures[0])
	}

	// Multiple failures - list them all
	var messages []string
	for i, entry := range failures {
		if i >= 5 {
			// Limit to 5 failures to keep message readable
			messages = append(messages, fmt.Sprintf("... and %d more", len(failures)-5))
			break
		}
		messages = append(messages, formatSingleNetworkFailure(entry))
	}

	return fmt.Sprintf("Network failures (%d total): %s", len(failures), strings.Join(messages, "; "))
}

// formatSingleNetworkFailure formats a single network failure entry.
func formatSingleNetworkFailure(entry NetworkEntry) string {
	if entry.Status != nil {
		return fmt.Sprintf("HTTP %d → %s", *entry.Status, entry.URL)
	} else if entry.ErrorText != "" {
		return fmt.Sprintf("%s → %s", entry.ErrorText, entry.URL)
	}
	return fmt.Sprintf("Request error → %s", entry.URL)
}

// Skipped creates a skipped result.
func Skipped(scenario, message string) *Result {
	return &Result{
		Scenario:  scenario,
		Status:    StatusSkipped,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// Blocked creates a blocked result with the specified reason.
func Blocked(scenario, message string, reason BlockedReason) *Result {
	return &Result{
		Scenario:      scenario,
		Status:        StatusBlocked,
		BlockedReason: reason,
		Message:       message,
		Timestamp:     time.Now().UTC(),
	}
}

// Failed creates a failed result.
func Failed(scenario, message string) *Result {
	return &Result{
		Scenario:  scenario,
		Status:    StatusFailed,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// Passed creates a successful result.
func Passed(scenario, uiURL string, duration time.Duration) *Result {
	return &Result{
		Scenario:   scenario,
		Status:     StatusPassed,
		Message:    "UI loaded successfully",
		Timestamp:  time.Now().UTC(),
		DurationMs: duration.Milliseconds(),
		UIURL:      uiURL,
	}
}
