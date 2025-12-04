package smoke

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"test-genie/internal/smoke/artifacts"
	"test-genie/internal/smoke/browser"
	"test-genie/internal/smoke/handshake"
	"test-genie/internal/smoke/orchestrator"
	"test-genie/internal/smoke/preflight"
	"test-genie/internal/smoke/smokeconfig"
)

// Runner provides a high-level API for running UI smoke tests.
type Runner struct {
	browserlessURL      string
	logger              io.Writer
	timeout             time.Duration
	uiURL               string // Optional override for UI URL (skips auto-detection)
	autoRecoveryEnabled bool   // Enable auto-recovery (default: true)
	sharedMode          bool   // Shared mode prevents restart if sessions are active
	autoStart           bool   // Auto-start scenario if UI port not detected
}

// NewRunner creates a new Runner.
func NewRunner(browserlessURL string, opts ...RunnerOption) *Runner {
	r := &Runner{
		browserlessURL:      browserlessURL,
		logger:              io.Discard,
		autoRecoveryEnabled: true,  // Enable auto-recovery by default
		sharedMode:          isSharedModeFromEnv(), // Read from BAS_BROWSERLESS_SHARED env var
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// isSharedModeFromEnv checks the BAS_BROWSERLESS_SHARED environment variable.
// Returns true if the env var is set to a truthy value (true, 1, yes).
func isSharedModeFromEnv() bool {
	val := os.Getenv("BAS_BROWSERLESS_SHARED")
	if val == "" {
		return false
	}
	val = strings.ToLower(val)
	return val == "true" || val == "1" || val == "yes"
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithRunnerLogger sets the logger for the runner.
func WithRunnerLogger(w io.Writer) RunnerOption {
	return func(r *Runner) {
		r.logger = w
	}
}

// WithRunnerTimeout sets a custom timeout for the runner.
func WithRunnerTimeout(timeout time.Duration) RunnerOption {
	return func(r *Runner) {
		r.timeout = timeout
	}
}

// WithUIURL sets a custom UI URL, bypassing auto-detection.
func WithUIURL(url string) RunnerOption {
	return func(r *Runner) {
		r.uiURL = url
	}
}

// WithAutoRecovery enables or disables automatic browserless recovery.
func WithAutoRecovery(enabled bool) RunnerOption {
	return func(r *Runner) {
		r.autoRecoveryEnabled = enabled
	}
}

// WithSharedMode enables shared mode, which prevents browserless restart
// if other sessions are active.
func WithSharedMode(enabled bool) RunnerOption {
	return func(r *Runner) {
		r.sharedMode = enabled
	}
}

// WithAutoStart enables automatic scenario startup if UI port is not detected.
func WithAutoStart(enabled bool) RunnerOption {
	return func(r *Runner) {
		r.autoStart = enabled
	}
}

// Run executes a UI smoke test for the given scenario.
func (r *Runner) Run(ctx context.Context, scenarioName, scenarioDir string) (*Result, error) {
	cfg := orchestrator.Config{
		ScenarioName:     scenarioName,
		ScenarioDir:      scenarioDir,
		BrowserlessURL:   r.browserlessURL,
		Timeout:          DefaultTimeout,
		HandshakeTimeout: DefaultHandshakeTimeout,
		Viewport:         orchestrator.Viewport{Width: DefaultViewportWidth, Height: DefaultViewportHeight},
		UIURL:            r.uiURL,     // Use custom URL if provided (bypasses auto-detection)
		AutoStart:        r.autoStart, // Enable auto-start if requested
	}

	// Apply runner-level timeout override (from CLI --timeout flag)
	if r.timeout > 0 {
		cfg.Timeout = r.timeout
	}

	// Load configuration from testing.json if available (only override if not already set by CLI)
	smokeCfg := smokeconfig.LoadUISmokeConfig(scenarioDir)
	if smokeCfg.TimeoutMs > 0 && r.timeout == 0 {
		cfg.Timeout = time.Duration(smokeCfg.TimeoutMs) * time.Millisecond
	}
	if smokeCfg.HandshakeTimeoutMs > 0 {
		cfg.HandshakeTimeout = time.Duration(smokeCfg.HandshakeTimeoutMs) * time.Millisecond
	}
	if len(smokeCfg.HandshakeSignals) > 0 {
		cfg.HandshakeSignals = smokeCfg.HandshakeSignals
	}
	if !smokeCfg.Enabled {
		return Skipped(scenarioName, "UI smoke harness disabled via .vrooli/testing.json"), nil
	}

	// Create the preflight checker (which also implements HealthChecker)
	checker := preflight.NewChecker(r.browserlessURL)

	// Build orchestrator options
	orchOpts := []orchestrator.Option{
		orchestrator.WithLogger(r.logger),
		orchestrator.WithPreflightChecker(checker),
		orchestrator.WithBrowserClient(browser.NewClient(r.browserlessURL, browser.WithTimeout(cfg.Timeout+cfg.HandshakeTimeout+20*time.Second))),
		orchestrator.WithArtifactWriter(artifacts.NewWriter()),
		orchestrator.WithHandshakeDetector(handshake.NewDetector()),
		orchestrator.WithPayloadGenerator(browser.NewPayloadGenerator()),
	}

	// Enable auto-recovery with health checker if requested
	if r.autoRecoveryEnabled {
		orchOpts = append(orchOpts,
			orchestrator.WithHealthChecker(checker),
			orchestrator.WithAutoRecoveryOptions(orchestrator.AutoRecoveryOptions{
				SharedMode:    r.sharedMode,
				MaxRetries:    1,
				ContainerName: "vrooli-browserless",
			}),
		)
	}

	// Enable auto-start if requested
	if r.autoStart {
		orchOpts = append(orchOpts,
			orchestrator.WithScenarioStarter(orchestrator.NewDefaultScenarioStarter()),
		)
	}

	orch := orchestrator.New(cfg, orchOpts...)

	orchResult, err := orch.Run(ctx)
	if err != nil {
		return nil, err
	}

	return convertResult(orchResult), nil
}

// convertResult converts an orchestrator.Result to a smoke.Result.
func convertResult(or *orchestrator.Result) *Result {
	r := &Result{
		Scenario:            or.Scenario,
		Status:              Status(or.Status),
		BlockedReason:       BlockedReason(or.BlockedReason),
		Message:             or.Message,
		Timestamp:           or.Timestamp,
		DurationMs:          or.DurationMs,
		UIURL:               or.UIURL,
		Handshake: HandshakeResult{
			Signaled:   or.Handshake.Signaled,
			TimedOut:   or.Handshake.TimedOut,
			DurationMs: or.Handshake.DurationMs,
			Error:      or.Handshake.Error,
		},
		NetworkFailureCount: or.NetworkFailureCount,
		PageErrorCount:      or.PageErrorCount,
		ConsoleErrorCount:   or.ConsoleErrorCount,
		Artifacts: ArtifactPaths{
			Screenshot: or.Artifacts.Screenshot,
			Console:    or.Artifacts.Console,
			Network:    or.Artifacts.Network,
			HTML:       or.Artifacts.HTML,
			Raw:        or.Artifacts.Raw,
			Readme:     or.Artifacts.Readme,
		},
		StorageShim: convertStorageShim(or.StorageShim),
		Raw:         or.Raw,
	}

	if or.Bundle != nil {
		r.Bundle = &BundleStatus{
			Fresh:  or.Bundle.Fresh,
			Reason: or.Bundle.Reason,
			Config: or.Bundle.Config,
		}
	}

	if or.IframeBridge != nil {
		r.IframeBridge = &BridgeStatus{
			DependencyPresent: or.IframeBridge.DependencyPresent,
			Version:           or.IframeBridge.Version,
			Details:           or.IframeBridge.Details,
		}
	}

	return r
}

// convertStorageShim converts orchestrator storage shim entries to uismoke entries.
func convertStorageShim(entries []orchestrator.StorageShimEntry) []StorageShimEntry {
	if len(entries) == 0 {
		return nil
	}
	result := make([]StorageShimEntry, len(entries))
	for i, e := range entries {
		result[i] = StorageShimEntry{
			Prop:    e.Prop,
			Patched: e.Patched,
			Reason:  e.Reason,
		}
	}
	return result
}
