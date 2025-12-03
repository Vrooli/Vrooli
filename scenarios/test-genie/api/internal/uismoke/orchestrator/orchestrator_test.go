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
	browserlessErr error
	bundleStatus   *BundleStatus
	bundleErr      error
	bridgeStatus   *BridgeStatus
	bridgeErr      error
	uiPort         int
	uiPortErr      error
	hasUIDir       bool
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
	paths     *ArtifactPaths
	writeErr  error
	resultErr error
}

func (m *mockArtifactWriter) WriteAll(ctx context.Context, scenarioDir, scenarioName string, response *BrowserResponse) (*ArtifactPaths, error) {
	return m.paths, m.writeErr
}

func (m *mockArtifactWriter) WriteResultJSON(ctx context.Context, scenarioDir, scenarioName string, result interface{}) error {
	return m.resultErr
}

// mockPayloadGenerator implements PayloadGenerator for testing.
type mockPayloadGenerator struct {
	payload string
}

func (m *mockPayloadGenerator) Generate(uiURL string, timeout, handshakeTimeout interface{}) string {
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
		hasUIDir:     true,
		bundleStatus: &BundleStatus{Fresh: true},
		uiPort:       0,
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
			Screenshot: "coverage/test-scenario/ui-smoke/screenshot.png",
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
	if result.Message != "Iframe bridge never signaled ready" {
		t.Errorf("Message = %q, want %q", result.Message, "Iframe bridge never signaled ready")
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

	blocked := Blocked("test", "browserless offline")
	if blocked.Status != StatusBlocked {
		t.Errorf("Blocked status = %v, want %v", blocked.Status, StatusBlocked)
	}
}
