package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"
)

// mockPreflightChecker implements PreflightChecker for testing.
type mockPreflightChecker struct {
	browserlessErr   error
	bundleStatus     *BundleStatus
	bundleErr        error
	bridgeStatus     *BridgeStatus
	bridgeErr        error
	uiPort           int
	uiPortErr        error
	hasUIDir         bool
	uiPortDefinition *UIPortDefinition
	uiPortDefinedErr error
}

func (m *mockPreflightChecker) CheckBrowserless(ctx context.Context) error {
	return m.browserlessErr
}

func (m *mockPreflightChecker) CheckBundleFreshness(ctx context.Context, scenarioDir string) (*BundleStatus, error) {
	return m.bundleStatus, m.bundleErr
}

func (m *mockPreflightChecker) CheckIframeBridge(ctx context.Context, scenarioDir string) (*BridgeStatus, error) {
	return m.bridgeStatus, m.bridgeErr
}

func (m *mockPreflightChecker) CheckUIPort(ctx context.Context, scenarioName string) (int, error) {
	return m.uiPort, m.uiPortErr
}

func (m *mockPreflightChecker) CheckUIPortDefined(scenarioDir string) (*UIPortDefinition, error) {
	if m.uiPortDefinition != nil {
		return m.uiPortDefinition, m.uiPortDefinedErr
	}
	// Default: return a defined port if hasUIDir is true
	if m.hasUIDir {
		return &UIPortDefinition{Defined: true, EnvVar: "UI_PORT"}, nil
	}
	return &UIPortDefinition{Defined: false}, nil
}

func (m *mockPreflightChecker) CheckUIDirectory(scenarioDir string) bool {
	return m.hasUIDir
}

// mockBrowserClient implements BrowserClient for testing.
type mockBrowserClient struct {
	response  *BrowserResponse
	execErr   error
	healthErr error
}

func (m *mockBrowserClient) ExecuteFunction(ctx context.Context, payload string) (*BrowserResponse, error) {
	if m.execErr != nil {
		return nil, m.execErr
	}
	return m.response, nil
}

func (m *mockBrowserClient) Health(ctx context.Context) error {
	return m.healthErr
}

// mockArtifactWriter implements ArtifactWriter for testing.
type mockArtifactWriter struct {
	paths      *ArtifactPaths
	writeErr   error
	resultErr  error
	readmeErr  error
	readmePath string
}

func (m *mockArtifactWriter) WriteAll(ctx context.Context, scenarioDir, scenarioName string, response *BrowserResponse) (*ArtifactPaths, error) {
	return m.paths, m.writeErr
}

func (m *mockArtifactWriter) WriteResultJSON(ctx context.Context, scenarioDir, scenarioName string, result interface{}) error {
	return m.resultErr
}

func (m *mockArtifactWriter) WriteReadme(ctx context.Context, scenarioDir, scenarioName string, result *Result) (string, error) {
	return m.readmePath, m.readmeErr
}

// mockPayloadGenerator implements PayloadGenerator for testing.
type mockPayloadGenerator struct {
	payload string
}

func (m *mockPayloadGenerator) Generate(uiURL string, timeout, handshakeTimeout interface{}, viewport Viewport, customSignals []string) string {
	if m.payload != "" {
		return m.payload
	}
	return "module.exports = async () => ({ success: true })"
}

func validConfig() Config {
	return Config{
		ScenarioName:     "test-scenario",
		ScenarioDir:      "/path/to/scenario",
		BrowserlessURL:   "http://localhost:4110",
		Timeout:          90 * time.Second,
		HandshakeTimeout: 15 * time.Second,
		Viewport:         DefaultViewport(),
	}
}

func TestOrchestrator_Run_NoUIDirectory(t *testing.T) {
	cfg := validConfig()
	preflight := &mockPreflightChecker{hasUIDir: false}

	orch := New(cfg, WithPreflightChecker(preflight))

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusSkipped {
		t.Errorf("Status = %v, want %v", result.Status, StatusSkipped)
	}
	if result.Message != "UI directory not detected" {
		t.Errorf("Message = %q, want %q", result.Message, "UI directory not detected")
	}
}

func TestOrchestrator_Run_BrowserlessOffline(t *testing.T) {
	cfg := validConfig()
	preflight := &mockPreflightChecker{
		hasUIDir:       true,
		browserlessErr: context.DeadlineExceeded,
	}

	orch := New(cfg, WithPreflightChecker(preflight))

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusBlocked {
		t.Errorf("Status = %v, want %v", result.Status, StatusBlocked)
	}
}

func TestOrchestrator_Run_BundleStale(t *testing.T) {
	cfg := validConfig()
	preflight := &mockPreflightChecker{
		hasUIDir: true,
		bundleStatus: &BundleStatus{
			Fresh:  false,
			Reason: "Bundle outdated",
		},
	}

	orch := New(cfg, WithPreflightChecker(preflight))

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusBlocked {
		t.Errorf("Status = %v, want %v", result.Status, StatusBlocked)
	}
	if result.Bundle == nil {
		t.Error("Bundle should be set")
	}
}

func TestOrchestrator_Run_NoUIPort(t *testing.T) {
	cfg := validConfig()
	preflight := &mockPreflightChecker{
		hasUIDir:         true,
		bundleStatus:     &BundleStatus{Fresh: true},
		uiPort:           0,
		uiPortDefinition: &UIPortDefinition{Defined: false}, // No UI port defined - should skip
	}

	orch := New(cfg, WithPreflightChecker(preflight))

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusSkipped {
		t.Errorf("Status = %v, want %v", result.Status, StatusSkipped)
	}
}

func TestOrchestrator_Run_UIPortDefinedButNotDetected(t *testing.T) {
	cfg := validConfig()
	preflight := &mockPreflightChecker{
		hasUIDir:         true,
		bundleStatus:     &BundleStatus{Fresh: true},
		uiPort:           0,                                                   // Not detected
		uiPortDefinition: &UIPortDefinition{Defined: true, EnvVar: "UI_PORT"}, // But defined
	}

	orch := New(cfg, WithPreflightChecker(preflight))

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusBlocked {
		t.Errorf("Status = %v, want %v", result.Status, StatusBlocked)
	}
}

func TestOrchestrator_Run_MissingBridge(t *testing.T) {
	cfg := validConfig()
	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: false},
	}

	orch := New(cfg, WithPreflightChecker(preflight))

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, StatusFailed)
	}
}

func TestOrchestrator_Run_Success(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true, Version: "1.0.0"},
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success: true,
			Handshake: HandshakeRaw{
				Signaled:   true,
				DurationMs: 500,
			},
			Raw: json.RawMessage(`{}`),
		},
	}

	artifacts := &mockArtifactWriter{
		paths: &ArtifactPaths{
			Screenshot: "coverage/ui-smoke/screenshot.png",
		},
	}

	payloadGen := &mockPayloadGenerator{}

	var logBuf bytes.Buffer
	orch := New(cfg,
		WithLogger(&logBuf),
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithArtifactWriter(artifacts),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusPassed {
		t.Errorf("Status = %v, want %v", result.Status, StatusPassed)
	}
	if result.UIURL != "http://localhost:3000" {
		t.Errorf("UIURL = %q, want %q", result.UIURL, "http://localhost:3000")
	}
	if !result.Handshake.Signaled {
		t.Error("Handshake.Signaled should be true")
	}
}

func TestOrchestrator_Run_HandshakeTimeout(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success: true,
			Handshake: HandshakeRaw{
				Signaled: false,
				TimedOut: true,
				Error:    "Timeout exceeded",
			},
			Raw: json.RawMessage(`{}`),
		},
	}

	payloadGen := &mockPayloadGenerator{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, StatusFailed)
	}
	expectedMsg := "Iframe bridge never signaled ready. See: docs/phases/structure/ui-smoke.md#handshake-timeout"
	if result.Message != expectedMsg {
		t.Errorf("Message = %q, want %q", result.Message, expectedMsg)
	}
}

func TestOrchestrator_Run_NetworkFailure(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	status404 := 404
	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success: true,
			Handshake: HandshakeRaw{
				Signaled: true,
			},
			Network: []NetworkEntry{
				{
					URL:    "http://localhost:3000/api/data",
					Status: &status404,
				},
			},
			Raw: json.RawMessage(`{}`),
		},
	}

	payloadGen := &mockPayloadGenerator{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, StatusFailed)
	}
}

func TestOrchestrator_Run_PageError(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success: true,
			Handshake: HandshakeRaw{
				Signaled: true,
			},
			PageErrors: []PageError{
				{Message: "Uncaught TypeError: Cannot read property 'foo' of undefined"},
			},
			Raw: json.RawMessage(`{}`),
		},
	}

	payloadGen := &mockPayloadGenerator{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, StatusFailed)
	}
}

func TestOrchestrator_Run_InvalidConfig(t *testing.T) {
	cfg := Config{} // Empty config - invalid

	orch := New(cfg)

	_, err := orch.Run(context.Background())
	if err == nil {
		t.Error("Run() should return error for invalid config")
	}
}

func TestOrchestrator_Run_BrowserExecutionError(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{
		execErr: context.DeadlineExceeded,
	}

	payloadGen := &mockPayloadGenerator{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, StatusFailed)
	}
}

func TestOrchestrator_Run_ExplicitUIURL(t *testing.T) {
	cfg := validConfig()
	cfg.UIURL = "http://custom.example.com:8080"

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success: true,
			Handshake: HandshakeRaw{
				Signaled: true,
			},
			Raw: json.RawMessage(`{}`),
		},
	}

	payloadGen := &mockPayloadGenerator{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.UIURL != "http://custom.example.com:8080" {
		t.Errorf("UIURL = %q, want %q", result.UIURL, "http://custom.example.com:8080")
	}
}

func TestOrchestrator_Run_NoBrowserClient(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	orch := New(cfg, WithPreflightChecker(preflight))

	_, err := orch.Run(context.Background())
	if err == nil {
		t.Error("Run() should return error when browser client not configured")
	}
}

func TestOrchestrator_Run_NoPayloadGenerator(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
	)

	_, err := orch.Run(context.Background())
	if err == nil {
		t.Error("Run() should return error when payload generator not configured")
	}
}

func TestOrchestrator_Run_BrowserlessFailure(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success: false,
			Error:   "Browser crashed",
			Raw:     json.RawMessage(`{}`),
		},
	}

	payloadGen := &mockPayloadGenerator{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, StatusFailed)
	}
	if result.Message != "Browser crashed" {
		t.Errorf("Message = %q, want %q", result.Message, "Browser crashed")
	}
}

func TestOrchestrator_Run_NetworkFailureWithErrorText(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success: true,
			Handshake: HandshakeRaw{
				Signaled: true,
			},
			Network: []NetworkEntry{
				{
					URL:       "http://localhost:3000/api/data",
					ErrorText: "net::ERR_CONNECTION_REFUSED",
				},
			},
			Raw: json.RawMessage(`{}`),
		},
	}

	payloadGen := &mockPayloadGenerator{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, StatusFailed)
	}
}

func TestOrchestrator_Run_WithHandshakeDetector(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success: true,
			Handshake: HandshakeRaw{
				Signaled:   true,
				DurationMs: 500,
			},
			Raw: json.RawMessage(`{}`),
		},
	}

	payloadGen := &mockPayloadGenerator{}
	handshakeDetector := &mockHandshakeDetector{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
		WithHandshakeDetector(handshakeDetector),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusPassed {
		t.Errorf("Status = %v, want %v", result.Status, StatusPassed)
	}
}

// mockHandshakeDetector implements HandshakeDetector for testing.
type mockHandshakeDetector struct{}

func (m *mockHandshakeDetector) Evaluate(raw *HandshakeRaw) HandshakeResult {
	return HandshakeResult{
		Signaled:   raw.Signaled,
		TimedOut:   raw.TimedOut,
		DurationMs: raw.DurationMs,
		Error:      raw.Error,
	}
}

func TestConfig_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Timeout != DefaultTimeout {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, DefaultTimeout)
	}
	if cfg.HandshakeTimeout != DefaultHandshakeTimeout {
		t.Errorf("HandshakeTimeout = %v, want %v", cfg.HandshakeTimeout, DefaultHandshakeTimeout)
	}
	if cfg.Viewport.Width != DefaultViewportWidth {
		t.Errorf("Viewport.Width = %d, want %d", cfg.Viewport.Width, DefaultViewportWidth)
	}
}

func TestConfig_TimeoutMs(t *testing.T) {
	cfg := Config{Timeout: 90 * time.Second}
	if got := cfg.TimeoutMs(); got != 90000 {
		t.Errorf("TimeoutMs() = %d, want 90000", got)
	}
}

func TestConfig_HandshakeTimeoutMs(t *testing.T) {
	cfg := Config{HandshakeTimeout: 15 * time.Second}
	if got := cfg.HandshakeTimeoutMs(); got != 15000 {
		t.Errorf("HandshakeTimeoutMs() = %d, want 15000", got)
	}
}

func TestConfig_ResolveUIURL(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "explicit UIURL",
			cfg:  Config{UIURL: "http://example.com:8080"},
			want: "http://example.com:8080",
		},
		{
			name: "UIPort only",
			cfg:  Config{UIPort: 3000},
			want: "http://localhost:3000",
		},
		{
			name: "neither set",
			cfg:  Config{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.ResolveUIURL(); got != tt.want {
				t.Errorf("ResolveUIURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatus_Constants(t *testing.T) {
	if StatusPassed != "passed" {
		t.Errorf("StatusPassed = %q, want %q", StatusPassed, "passed")
	}
	if StatusFailed != "failed" {
		t.Errorf("StatusFailed = %q, want %q", StatusFailed, "failed")
	}
	if StatusSkipped != "skipped" {
		t.Errorf("StatusSkipped = %q, want %q", StatusSkipped, "skipped")
	}
	if StatusBlocked != "blocked" {
		t.Errorf("StatusBlocked = %q, want %q", StatusBlocked, "blocked")
	}
}

func TestResultConstructors(t *testing.T) {
	passed := Passed("test", "http://localhost:3000", 5*time.Second)
	if passed.Status != StatusPassed {
		t.Errorf("Passed status = %v, want %v", passed.Status, StatusPassed)
	}

	failed := Failed("test", "error message")
	if failed.Status != StatusFailed {
		t.Errorf("Failed status = %v, want %v", failed.Status, StatusFailed)
	}

	skipped := Skipped("test", "no UI")
	if skipped.Status != StatusSkipped {
		t.Errorf("Skipped status = %v, want %v", skipped.Status, StatusSkipped)
	}

	blocked := Blocked("test", "browserless offline", BlockedReasonBrowserlessOffline)
	if blocked.Status != StatusBlocked {
		t.Errorf("Blocked status = %v, want %v", blocked.Status, StatusBlocked)
	}
	if blocked.BlockedReason != BlockedReasonBrowserlessOffline {
		t.Errorf("Blocked reason = %v, want %v", blocked.BlockedReason, BlockedReasonBrowserlessOffline)
	}
}

func TestFormatNetworkFailures_Empty(t *testing.T) {
	result := formatNetworkFailures(nil)
	if result != "" {
		t.Errorf("formatNetworkFailures(nil) = %q, want empty string", result)
	}

	result = formatNetworkFailures([]NetworkEntry{})
	if result != "" {
		t.Errorf("formatNetworkFailures([]) = %q, want empty string", result)
	}
}

func TestFormatNetworkFailures_SingleHTTPError(t *testing.T) {
	status := 404
	failures := []NetworkEntry{
		{URL: "http://example.com/api", Status: &status},
	}
	result := formatNetworkFailures(failures)
	if result != "HTTP 404 → http://example.com/api" {
		t.Errorf("formatNetworkFailures() = %q, want HTTP error format", result)
	}
}

func TestFormatNetworkFailures_SingleConnectionError(t *testing.T) {
	failures := []NetworkEntry{
		{URL: "http://example.com/api", ErrorText: "net::ERR_CONNECTION_REFUSED"},
	}
	result := formatNetworkFailures(failures)
	if result != "net::ERR_CONNECTION_REFUSED → http://example.com/api" {
		t.Errorf("formatNetworkFailures() = %q, want connection error format", result)
	}
}

func TestFormatNetworkFailures_MultipleFailures(t *testing.T) {
	status404 := 404
	status500 := 500
	failures := []NetworkEntry{
		{URL: "http://example.com/api", Status: &status404},
		{URL: "http://example.com/other", Status: &status500},
		{URL: "http://example.com/ws", ErrorText: "net::ERR_CONNECTION_REFUSED"},
	}
	result := formatNetworkFailures(failures)

	// Should contain count
	if !containsString(result, "(3 total)") {
		t.Errorf("result should contain '(3 total)', got: %s", result)
	}

	// Should contain all failures
	if !containsString(result, "HTTP 404") {
		t.Errorf("result should contain 'HTTP 404', got: %s", result)
	}
	if !containsString(result, "HTTP 500") {
		t.Errorf("result should contain 'HTTP 500', got: %s", result)
	}
	if !containsString(result, "ERR_CONNECTION_REFUSED") {
		t.Errorf("result should contain 'ERR_CONNECTION_REFUSED', got: %s", result)
	}
}

func TestFormatNetworkFailures_TruncatesAtFive(t *testing.T) {
	status := 500
	failures := make([]NetworkEntry, 8)
	for i := range failures {
		failures[i] = NetworkEntry{URL: "http://example.com/api" + string(rune('0'+i)), Status: &status}
	}

	result := formatNetworkFailures(failures)

	// Should contain count
	if !containsString(result, "(8 total)") {
		t.Errorf("result should contain '(8 total)', got: %s", result)
	}

	// Should contain "... and X more"
	if !containsString(result, "... and 3 more") {
		t.Errorf("result should contain '... and 3 more', got: %s", result)
	}
}

func TestFormatSingleNetworkFailure_GenericError(t *testing.T) {
	entry := NetworkEntry{URL: "http://example.com/api"}
	result := formatSingleNetworkFailure(entry)
	if result != "Request error → http://example.com/api" {
		t.Errorf("formatSingleNetworkFailure() = %q, want generic error format", result)
	}
}

func containsString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// =============================================================================
// Auto-Recovery Tests
// =============================================================================

func TestOrchestrator_Run_AutoRecovery_Success(t *testing.T) {
	cfg := validConfig()

	mockHealth := &mockHealthCheckerFull{
		mockPreflightChecker: mockPreflightChecker{
			hasUIDir:       true,
			browserlessErr: context.DeadlineExceeded, // Initial failure
			bundleStatus:   &BundleStatus{Fresh: true},
			uiPort:         3000,
			bridgeStatus:   &BridgeStatus{DependencyPresent: true},
		},
		recoveryResult: &RecoveryResult{
			Attempted: true,
			Success:   true,
			Action:    "restarted container",
		},
		browserlessRecovered: true, // After recovery, browserless works
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success:   true,
			Handshake: HandshakeRaw{Signaled: true},
			Raw:       json.RawMessage(`{}`),
		},
	}

	payloadGen := &mockPayloadGenerator{}

	orch := New(cfg,
		WithHealthChecker(mockHealth),
		WithAutoRecovery(true),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusPassed {
		t.Errorf("Status = %v, want %v (message: %s)", result.Status, StatusPassed, result.Message)
	}
}

func TestOrchestrator_Run_AutoRecovery_Failed(t *testing.T) {
	cfg := validConfig()

	mockHealth := &mockHealthCheckerFull{
		mockPreflightChecker: mockPreflightChecker{
			hasUIDir:       true,
			browserlessErr: context.DeadlineExceeded,
			bundleStatus:   &BundleStatus{Fresh: true},
			uiPort:         3000,
			bridgeStatus:   &BridgeStatus{DependencyPresent: true},
		},
		recoveryResult: &RecoveryResult{
			Attempted: true,
			Success:   false,
			Error:     "container failed to start",
		},
		diagnosis: &Diagnosis{
			Type:           DiagnosisOffline,
			Message:        "Browserless is offline",
			Recommendation: "Run: resource-browserless manage start",
		},
	}

	orch := New(cfg,
		WithHealthChecker(mockHealth),
		WithAutoRecovery(true),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusBlocked {
		t.Errorf("Status = %v, want %v", result.Status, StatusBlocked)
	}
}

func TestOrchestrator_Run_WithDiagnosis_NoBrowserClient(t *testing.T) {
	cfg := validConfig()

	mockHealth := &mockHealthCheckerFull{
		mockPreflightChecker: mockPreflightChecker{
			hasUIDir:     true,
			bundleStatus: &BundleStatus{Fresh: true},
			uiPort:       3000,
			bridgeStatus: &BridgeStatus{DependencyPresent: true},
		},
		diagnosis: &Diagnosis{
			Type:           DiagnosisProcessLeak,
			Message:        "Too many Chrome processes",
			Recommendation: "Restart browserless",
		},
	}

	browser := &mockBrowserClient{
		execErr: context.DeadlineExceeded,
	}

	payloadGen := &mockPayloadGenerator{}

	orch := New(cfg,
		WithHealthChecker(mockHealth),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, StatusFailed)
	}
	// Should contain diagnosis info in message
	if !containsString(result.Message, "Browserless") {
		t.Errorf("Message should contain diagnosis, got: %s", result.Message)
	}
}

func TestOrchestrator_Run_ConsoleErrorsCounted(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success: true,
			Handshake: HandshakeRaw{
				Signaled: true,
			},
			Console: []ConsoleEntry{
				{Level: "error", Message: "Error 1"},
				{Level: "error", Message: "Error 2"},
				{Level: "warn", Message: "Warning"},
				{Level: "log", Message: "Log"},
			},
			Raw: json.RawMessage(`{}`),
		},
	}

	payloadGen := &mockPayloadGenerator{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.ConsoleErrorCount != 2 {
		t.Errorf("ConsoleErrorCount = %d, want 2", result.ConsoleErrorCount)
	}
}

func TestOrchestrator_Run_ArtifactWriteError(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success:   true,
			Handshake: HandshakeRaw{Signaled: true},
			Raw:       json.RawMessage(`{}`),
		},
	}

	artifacts := &mockArtifactWriter{
		writeErr: context.DeadlineExceeded,
	}

	payloadGen := &mockPayloadGenerator{}
	var logBuf bytes.Buffer

	orch := New(cfg,
		WithLogger(&logBuf),
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithArtifactWriter(artifacts),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should still pass even if artifact write fails
	if result.Status != StatusPassed {
		t.Errorf("Status = %v, want %v", result.Status, StatusPassed)
	}
}

func TestOrchestrator_Run_PersistResultError(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success:   true,
			Handshake: HandshakeRaw{Signaled: true},
			Raw:       json.RawMessage(`{}`),
		},
	}

	artifacts := &mockArtifactWriter{
		resultErr: context.DeadlineExceeded,
		readmeErr: context.DeadlineExceeded,
		paths:     &ArtifactPaths{Screenshot: "test.png"},
	}

	payloadGen := &mockPayloadGenerator{}
	var logBuf bytes.Buffer

	orch := New(cfg,
		WithLogger(&logBuf),
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithArtifactWriter(artifacts),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Should still pass even if persist fails
	if result.Status != StatusPassed {
		t.Errorf("Status = %v, want %v", result.Status, StatusPassed)
	}
}

func TestOrchestrator_Run_BrowserResponseFailureEmptyError(t *testing.T) {
	cfg := validConfig()

	preflight := &mockPreflightChecker{
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       3000,
		bridgeStatus: &BridgeStatus{DependencyPresent: true},
	}

	browser := &mockBrowserClient{
		response: &BrowserResponse{
			Success: false,
			Error:   "", // Empty error
			Raw:     json.RawMessage(`{}`),
		},
	}

	payloadGen := &mockPayloadGenerator{}

	orch := New(cfg,
		WithPreflightChecker(preflight),
		WithBrowserClient(browser),
		WithPayloadGenerator(payloadGen),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, StatusFailed)
	}
	// Should have default message when error is empty
	if result.Message != "Browserless execution failed" {
		t.Errorf("Message = %q, want default failure message", result.Message)
	}
}

// mockHealthCheckerFull is a more complete mock for testing auto-recovery flows.
type mockHealthCheckerFull struct {
	mockPreflightChecker
	healthy              bool
	healthErr            error
	diagnostics          *HealthDiagnostics
	recoveryResult       *RecoveryResult
	diagnosis            *Diagnosis
	browserlessRecovered bool
	checkCount           int
}

func (m *mockHealthCheckerFull) CheckBrowserless(ctx context.Context) error {
	m.checkCount++
	// First check fails, subsequent checks succeed if recovered
	if m.browserlessRecovered && m.checkCount > 1 {
		return nil
	}
	return m.mockPreflightChecker.CheckBrowserless(ctx)
}

func (m *mockHealthCheckerFull) IsHealthy(ctx context.Context) (bool, *HealthDiagnostics, error) {
	return m.healthy, m.diagnostics, m.healthErr
}

func (m *mockHealthCheckerFull) EnsureHealthy(ctx context.Context, opts AutoRecoveryOptions) (*RecoveryResult, error) {
	if m.recoveryResult != nil {
		return m.recoveryResult, nil
	}
	return &RecoveryResult{Attempted: false, Success: true}, nil
}

func (m *mockHealthCheckerFull) DiagnoseBrowserlessFailure(ctx context.Context, scenarioName string) *Diagnosis {
	if m.diagnosis != nil {
		return m.diagnosis
	}
	return &Diagnosis{
		Type:           DiagnosisUnknown,
		Message:        "Unknown failure",
		Recommendation: "Check logs",
	}
}
