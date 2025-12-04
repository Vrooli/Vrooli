package orchestrator

import (
	"context"
	"testing"
	"time"
)

func TestNewDefaultScenarioStarter(t *testing.T) {
	starter := NewDefaultScenarioStarter()
	if starter == nil {
		t.Fatal("expected non-nil starter")
	}
	if starter.StartTimeout != 120*time.Second {
		t.Errorf("expected StartTimeout=120s, got %v", starter.StartTimeout)
	}
	if starter.PollInterval != 2*time.Second {
		t.Errorf("expected PollInterval=2s, got %v", starter.PollInterval)
	}
}

func TestMockScenarioStarter_Start(t *testing.T) {
	mock := &MockScenarioStarter{}

	port, err := mock.Start(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 8080 {
		t.Errorf("expected port 8080, got %d", port)
	}
	if len(mock.StartedScenarios) != 1 {
		t.Errorf("expected 1 started scenario, got %d", len(mock.StartedScenarios))
	}
	if mock.StartedScenarios[0] != "test-scenario" {
		t.Errorf("expected scenario 'test-scenario', got '%s'", mock.StartedScenarios[0])
	}
}

func TestMockScenarioStarter_Start_CustomFunc(t *testing.T) {
	mock := &MockScenarioStarter{
		StartFunc: func(ctx context.Context, name string) (int, error) {
			return 9090, nil
		},
	}

	port, err := mock.Start(context.Background(), "custom-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 9090 {
		t.Errorf("expected port 9090, got %d", port)
	}
}

func TestMockScenarioStarter_Stop(t *testing.T) {
	mock := &MockScenarioStarter{}

	err := mock.Stop(context.Background(), "test-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.StoppedScenarios) != 1 {
		t.Errorf("expected 1 stopped scenario, got %d", len(mock.StoppedScenarios))
	}
	if mock.StoppedScenarios[0] != "test-scenario" {
		t.Errorf("expected scenario 'test-scenario', got '%s'", mock.StoppedScenarios[0])
	}
}

func TestOrchestrator_Run_AutoStart_Success(t *testing.T) {
	// Create mock scenario starter
	mockStarter := &MockScenarioStarter{
		StartFunc: func(ctx context.Context, name string) (int, error) {
			return 3000, nil
		},
	}

	// Create mock preflight that:
	// - Returns UI directory exists
	// - Browserless is available
	// - Bundle is fresh
	// - UI port is defined but not detected initially
	mockPreflight := &MockPreflightChecker{
		checkUIDirectory: true,
		browserlessAvail: true,
		bundleStatus:     &BundleStatus{Fresh: true},
		uiPortDefined:    &UIPortDefinition{Defined: true, EnvVar: "UI_PORT"},
		uiPort:           0, // Not detected initially
	}

	// Create mock browser client
	mockBrowser := &MockBrowserClient{
		response: &BrowserResponse{
			Success:   true,
			Handshake: HandshakeRaw{Signaled: true, DurationMs: 100},
		},
	}

	cfg := Config{
		ScenarioName:     "test-scenario",
		ScenarioDir:      "/tmp/test",
		BrowserlessURL:   "http://localhost:3000",
		Timeout:          30 * time.Second,
		HandshakeTimeout: 10 * time.Second,
		Viewport:         Viewport{Width: 1280, Height: 720},
		AutoStart:        true,
	}

	orch := New(cfg,
		WithPreflightChecker(mockPreflight),
		WithBrowserClient(mockBrowser),
		WithPayloadGenerator(&MockPayloadGenerator{}),
		WithScenarioStarter(mockStarter),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != StatusPassed {
		t.Errorf("expected status passed, got %s: %s", result.Status, result.Message)
	}

	// Verify scenario was auto-started
	if len(mockStarter.StartedScenarios) != 1 {
		t.Errorf("expected 1 auto-started scenario, got %d", len(mockStarter.StartedScenarios))
	}
	if mockStarter.StartedScenarios[0] != "test-scenario" {
		t.Errorf("expected 'test-scenario' to be started, got '%s'", mockStarter.StartedScenarios[0])
	}

	// Verify the UI URL was set correctly
	if result.UIURL != "http://localhost:3000" {
		t.Errorf("expected UIURL 'http://localhost:3000', got '%s'", result.UIURL)
	}
}

func TestOrchestrator_Run_AutoStart_Disabled(t *testing.T) {
	// Create mock preflight that:
	// - Returns UI directory exists
	// - Browserless is available
	// - Bundle is fresh
	// - UI port is defined but not detected
	mockPreflight := &MockPreflightChecker{
		checkUIDirectory: true,
		browserlessAvail: true,
		bundleStatus:     &BundleStatus{Fresh: true},
		uiPortDefined:    &UIPortDefinition{Defined: true, EnvVar: "UI_PORT"},
		uiPort:           0, // Not detected
	}

	cfg := Config{
		ScenarioName:     "test-scenario",
		ScenarioDir:      "/tmp/test",
		BrowserlessURL:   "http://localhost:3000",
		Timeout:          30 * time.Second,
		HandshakeTimeout: 10 * time.Second,
		Viewport:         Viewport{Width: 1280, Height: 720},
		AutoStart:        false, // Disabled
	}

	orch := New(cfg,
		WithPreflightChecker(mockPreflight),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return blocked since auto-start is disabled
	if result.Status != StatusBlocked {
		t.Errorf("expected status blocked, got %s", result.Status)
	}

	// Message should suggest using --auto-start
	if result.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestOrchestrator_Run_AutoStart_Fails(t *testing.T) {
	// Create mock scenario starter that fails
	mockStarter := &MockScenarioStarter{
		StartFunc: func(ctx context.Context, name string) (int, error) {
			return 0, context.DeadlineExceeded
		},
	}

	mockPreflight := &MockPreflightChecker{
		checkUIDirectory: true,
		browserlessAvail: true,
		bundleStatus:     &BundleStatus{Fresh: true},
		uiPortDefined:    &UIPortDefinition{Defined: true, EnvVar: "UI_PORT"},
		uiPort:           0, // Not detected
	}

	cfg := Config{
		ScenarioName:     "test-scenario",
		ScenarioDir:      "/tmp/test",
		BrowserlessURL:   "http://localhost:3000",
		Timeout:          30 * time.Second,
		HandshakeTimeout: 10 * time.Second,
		Viewport:         Viewport{Width: 1280, Height: 720},
		AutoStart:        true,
	}

	orch := New(cfg,
		WithPreflightChecker(mockPreflight),
		WithScenarioStarter(mockStarter),
	)

	result, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return blocked since auto-start failed
	if result.Status != StatusBlocked {
		t.Errorf("expected status blocked, got %s", result.Status)
	}

	// Verify auto-start was attempted
	if len(mockStarter.StartedScenarios) != 1 {
		t.Errorf("expected 1 auto-start attempt, got %d", len(mockStarter.StartedScenarios))
	}
}

// MockPreflightChecker for testing
type MockPreflightChecker struct {
	checkUIDirectory bool
	browserlessAvail bool
	browserlessErr   error
	bundleStatus     *BundleStatus
	uiPortDefined    *UIPortDefinition
	uiPort           int
	bridgeStatus     *BridgeStatus
}

func (m *MockPreflightChecker) CheckUIDirectory(scenarioDir string) bool {
	return m.checkUIDirectory
}

func (m *MockPreflightChecker) CheckBrowserless(ctx context.Context) error {
	if m.browserlessAvail {
		return nil
	}
	if m.browserlessErr != nil {
		return m.browserlessErr
	}
	return context.DeadlineExceeded
}

func (m *MockPreflightChecker) CheckBundleFreshness(ctx context.Context, scenarioDir string) (*BundleStatus, error) {
	return m.bundleStatus, nil
}

func (m *MockPreflightChecker) CheckUIPortDefined(scenarioDir string) (*UIPortDefinition, error) {
	return m.uiPortDefined, nil
}

func (m *MockPreflightChecker) CheckUIPort(ctx context.Context, scenarioName string) (int, error) {
	return m.uiPort, nil
}

func (m *MockPreflightChecker) CheckIframeBridge(ctx context.Context, scenarioDir string) (*BridgeStatus, error) {
	if m.bridgeStatus == nil {
		return &BridgeStatus{DependencyPresent: true}, nil
	}
	return m.bridgeStatus, nil
}

// MockBrowserClient for testing
type MockBrowserClient struct {
	response *BrowserResponse
	err      error
}

func (m *MockBrowserClient) ExecuteFunction(ctx context.Context, payload string) (*BrowserResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *MockBrowserClient) Health(ctx context.Context) error {
	return nil
}

// MockPayloadGenerator for testing
type MockPayloadGenerator struct{}

func (m *MockPayloadGenerator) Generate(uiURL string, timeout, handshakeTimeout interface{}, viewport Viewport, handshakeSignals []string) string {
	return "mock-payload"
}
