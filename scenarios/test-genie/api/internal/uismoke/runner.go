package uismoke

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"test-genie/internal/uismoke/artifacts"
	"test-genie/internal/uismoke/browser"
	"test-genie/internal/uismoke/handshake"
	"test-genie/internal/uismoke/orchestrator"
	"test-genie/internal/uismoke/preflight"
)

// Runner provides a high-level API for running UI smoke tests.
type Runner struct {
	browserlessURL string
	logger         io.Writer
}

// NewRunner creates a new Runner.
func NewRunner(browserlessURL string, opts ...RunnerOption) *Runner {
	r := &Runner{
		browserlessURL: browserlessURL,
		logger:         io.Discard,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithRunnerLogger sets the logger for the runner.
func WithRunnerLogger(w io.Writer) RunnerOption {
	return func(r *Runner) {
		r.logger = w
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
	}

	// Load configuration from testing.json if available
	if testingConfig := loadTestingConfig(scenarioDir); testingConfig != nil {
		if testingConfig.Timeout > 0 {
			cfg.Timeout = testingConfig.Timeout
		}
		if testingConfig.HandshakeTimeout > 0 {
			cfg.HandshakeTimeout = testingConfig.HandshakeTimeout
		}
		if !testingConfig.Enabled {
			return Skipped(scenarioName, "UI smoke harness disabled via .vrooli/testing.json"), nil
		}
	}

	orch := orchestrator.New(cfg,
		orchestrator.WithLogger(r.logger),
		orchestrator.WithPreflightChecker(preflight.NewChecker(r.browserlessURL)),
		orchestrator.WithBrowserClient(browser.NewClient(r.browserlessURL, browser.WithTimeout(cfg.Timeout+cfg.HandshakeTimeout+20*time.Second))),
		orchestrator.WithArtifactWriter(artifacts.NewWriter()),
		orchestrator.WithHandshakeDetector(handshake.NewDetector()),
		orchestrator.WithPayloadGenerator(browser.NewPayloadGenerator()),
	)

	orchResult, err := orch.Run(ctx)
	if err != nil {
		return nil, err
	}

	return convertResult(orchResult), nil
}

// convertResult converts an orchestrator.Result to a uismoke.Result.
func convertResult(or *orchestrator.Result) *Result {
	r := &Result{
		Scenario:   or.Scenario,
		Status:     Status(or.Status),
		Message:    or.Message,
		Timestamp:  or.Timestamp,
		DurationMs: or.DurationMs,
		UIURL:      or.UIURL,
		Handshake: HandshakeResult{
			Signaled:   or.Handshake.Signaled,
			TimedOut:   or.Handshake.TimedOut,
			DurationMs: or.Handshake.DurationMs,
			Error:      or.Handshake.Error,
		},
		Artifacts: ArtifactPaths{
			Screenshot: or.Artifacts.Screenshot,
			Console:    or.Artifacts.Console,
			Network:    or.Artifacts.Network,
			HTML:       or.Artifacts.HTML,
			Raw:        or.Artifacts.Raw,
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

// testingConfig holds UI smoke settings from .vrooli/testing.json.
type testingConfig struct {
	Enabled          bool
	Timeout          time.Duration
	HandshakeTimeout time.Duration
}

// rawTestingJSON represents the structure of .vrooli/testing.json.
type rawTestingJSON struct {
	Structure struct {
		UISmoke struct {
			Enabled            *bool `json:"enabled"`
			TimeoutMs          int64 `json:"timeout_ms"`
			HandshakeTimeoutMs int64 `json:"handshake_timeout_ms"`
		} `json:"ui_smoke"`
	} `json:"structure"`
}

// loadTestingConfig loads UI smoke configuration from .vrooli/testing.json.
func loadTestingConfig(scenarioDir string) *testingConfig {
	configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var raw rawTestingJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	cfg := &testingConfig{
		Enabled: true, // enabled by default
	}

	if raw.Structure.UISmoke.Enabled != nil {
		cfg.Enabled = *raw.Structure.UISmoke.Enabled
	}

	if raw.Structure.UISmoke.TimeoutMs > 0 {
		cfg.Timeout = time.Duration(raw.Structure.UISmoke.TimeoutMs) * time.Millisecond
	}

	if raw.Structure.UISmoke.HandshakeTimeoutMs > 0 {
		cfg.HandshakeTimeout = time.Duration(raw.Structure.UISmoke.HandshakeTimeoutMs) * time.Millisecond
	}

	return cfg
}
