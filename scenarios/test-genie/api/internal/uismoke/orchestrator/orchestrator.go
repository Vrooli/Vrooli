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
	config     Config
	preflight  PreflightChecker
	browser    BrowserClient
	artifacts  ArtifactWriter
	handshake  HandshakeDetector
	logger     io.Writer
	payloadGen PayloadGenerator
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

	// Step 2: Check browserless availability
	if o.preflight != nil {
		if err := o.preflight.CheckBrowserless(ctx); err != nil {
			o.log("Browserless check failed: %v", err)
			result := Blocked(o.config.ScenarioName,
				"Browserless resource is offline. Run 'resource-browserless manage start' then rerun the smoke test.")
			o.persistResult(ctx, result)
			return result, nil
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
					bundleStatus.Reason, o.config.ScenarioName, o.config.ScenarioName))
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
		// If service.json defines a UI port but we couldn't detect it, that's an error
		if uiPortDef != nil && uiPortDef.Defined {
			o.log("UI port defined in service.json (%s) but not detected - scenario may not be running", uiPortDef.EnvVar)
			result := Blocked(o.config.ScenarioName,
				fmt.Sprintf("UI port is defined in service.json (%s) but not detected.\n"+
					"  ↳ The scenario may not be running or the UI server failed to start.\n"+
					"  ↳ Fix: vrooli scenario restart %s\n"+
					"  ↳ Then check: vrooli scenario logs %s --step start-ui",
					uiPortDef.EnvVar, o.config.ScenarioName, o.config.ScenarioName))
			o.persistResult(ctx, result)
			return result, nil
		}

		// No UI port defined - this scenario genuinely has no UI
		o.log("No UI port defined in service.json, skipping smoke test")
		result := Skipped(o.config.ScenarioName, "Scenario does not define a UI port")
		o.persistResult(ctx, result)
		return result, nil
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

	payload := o.payloadGen.Generate(uiURL, o.config.Timeout, o.config.HandshakeTimeout)
	o.log("Executing browser session for %s", uiURL)

	response, err := o.browser.ExecuteFunction(ctx, payload)
	if err != nil {
		o.log("Browser execution failed: %v", err)
		result := Failed(o.config.ScenarioName, fmt.Sprintf("Failed to reach Browserless API: %v", err))
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
		message = "Iframe bridge never signaled ready"
	} else if len(response.Network) > 0 {
		status = StatusFailed
		entry := response.Network[0]
		if entry.Status != nil {
			message = fmt.Sprintf("Network failure: HTTP %d → %s", *entry.Status, entry.URL)
		} else if entry.ErrorText != "" {
			message = fmt.Sprintf("Network failure: %s → %s", entry.ErrorText, entry.URL)
		} else {
			message = fmt.Sprintf("Network failure: Request error → %s", entry.URL)
		}
	} else if len(response.PageErrors) > 0 {
		status = StatusFailed
		message = fmt.Sprintf("UI exception: %s", response.PageErrors[0].Message)
	}

	result := &Result{
		Scenario:     o.config.ScenarioName,
		Status:       status,
		Message:      message,
		Timestamp:    time.Now().UTC(),
		DurationMs:   duration.Milliseconds(),
		UIURL:        uiURL,
		Handshake:    handshake,
		Bundle:       bundle,
		IframeBridge: bridge,
		StorageShim:  response.StorageShim,
		Raw:          response.Raw,
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
		if err := o.artifacts.WriteReadme(ctx, o.config.ScenarioDir, o.config.ScenarioName, result); err != nil {
			o.log("Failed to write README: %v", err)
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

// Skipped creates a skipped result.
func Skipped(scenario, message string) *Result {
	return &Result{
		Scenario:  scenario,
		Status:    StatusSkipped,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// Blocked creates a blocked result.
func Blocked(scenario, message string) *Result {
	return &Result{
		Scenario:  scenario,
		Status:    StatusBlocked,
		Message:   message,
		Timestamp: time.Now().UTC(),
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
