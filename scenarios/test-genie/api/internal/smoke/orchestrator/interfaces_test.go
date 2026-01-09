package orchestrator

import (
	"testing"
	"time"
)

// =============================================================================
// BlockedReason.ExitCode Tests
// =============================================================================

func TestBlockedReason_ExitCode(t *testing.T) {
	tests := []struct {
		reason   BlockedReason
		expected int
	}{
		{BlockedReasonBrowserlessOffline, 50},
		{BlockedReasonBundleStale, 60},
		{BlockedReasonUIPortMissing, 61},
		{BlockedReasonNone, 1},
		{BlockedReason("unknown"), 1},
	}

	for _, tc := range tests {
		t.Run(string(tc.reason), func(t *testing.T) {
			if got := tc.reason.ExitCode(); got != tc.expected {
				t.Errorf("ExitCode() = %d, want %d", got, tc.expected)
			}
		})
	}
}

// =============================================================================
// Config.Validate Tests
// =============================================================================

func TestConfig_Validate_MissingScenarioName(t *testing.T) {
	cfg := Config{
		ScenarioDir:      "/path/to/scenario",
		BrowserlessURL:   "http://localhost:3000",
		Timeout:          90 * time.Second,
		HandshakeTimeout: 15 * time.Second,
		Viewport:         DefaultViewport(),
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing scenario name")
	}
}

func TestConfig_Validate_MissingScenarioDir(t *testing.T) {
	cfg := Config{
		ScenarioName:     "test",
		BrowserlessURL:   "http://localhost:3000",
		Timeout:          90 * time.Second,
		HandshakeTimeout: 15 * time.Second,
		Viewport:         DefaultViewport(),
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing scenario dir")
	}
}

func TestConfig_Validate_MissingBrowserlessURL(t *testing.T) {
	cfg := Config{
		ScenarioName:     "test",
		ScenarioDir:      "/path/to/scenario",
		Timeout:          90 * time.Second,
		HandshakeTimeout: 15 * time.Second,
		Viewport:         DefaultViewport(),
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing browserless URL")
	}
}

func TestConfig_Validate_ZeroTimeout(t *testing.T) {
	cfg := Config{
		ScenarioName:     "test",
		ScenarioDir:      "/path/to/scenario",
		BrowserlessURL:   "http://localhost:3000",
		Timeout:          0,
		HandshakeTimeout: 15 * time.Second,
		Viewport:         DefaultViewport(),
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for zero timeout")
	}
}

func TestConfig_Validate_NegativeTimeout(t *testing.T) {
	cfg := Config{
		ScenarioName:     "test",
		ScenarioDir:      "/path/to/scenario",
		BrowserlessURL:   "http://localhost:3000",
		Timeout:          -1 * time.Second,
		HandshakeTimeout: 15 * time.Second,
		Viewport:         DefaultViewport(),
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for negative timeout")
	}
}

func TestConfig_Validate_ZeroHandshakeTimeout(t *testing.T) {
	cfg := Config{
		ScenarioName:     "test",
		ScenarioDir:      "/path/to/scenario",
		BrowserlessURL:   "http://localhost:3000",
		Timeout:          90 * time.Second,
		HandshakeTimeout: 0,
		Viewport:         DefaultViewport(),
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for zero handshake timeout")
	}
}

func TestConfig_Validate_ZeroViewportWidth(t *testing.T) {
	cfg := Config{
		ScenarioName:     "test",
		ScenarioDir:      "/path/to/scenario",
		BrowserlessURL:   "http://localhost:3000",
		Timeout:          90 * time.Second,
		HandshakeTimeout: 15 * time.Second,
		Viewport:         Viewport{Width: 0, Height: 720},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for zero viewport width")
	}
}

func TestConfig_Validate_ZeroViewportHeight(t *testing.T) {
	cfg := Config{
		ScenarioName:     "test",
		ScenarioDir:      "/path/to/scenario",
		BrowserlessURL:   "http://localhost:3000",
		Timeout:          90 * time.Second,
		HandshakeTimeout: 15 * time.Second,
		Viewport:         Viewport{Width: 1280, Height: 0},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for zero viewport height")
	}
}

func TestConfig_Validate_NegativeViewportDimensions(t *testing.T) {
	cfg := Config{
		ScenarioName:     "test",
		ScenarioDir:      "/path/to/scenario",
		BrowserlessURL:   "http://localhost:3000",
		Timeout:          90 * time.Second,
		HandshakeTimeout: 15 * time.Second,
		Viewport:         Viewport{Width: -100, Height: -200},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for negative viewport dimensions")
	}
}

func TestConfig_Validate_ValidConfig(t *testing.T) {
	cfg := validConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("unexpected error for valid config: %v", err)
	}
}

// =============================================================================
// Viewport Tests
// =============================================================================

func TestDefaultViewport(t *testing.T) {
	vp := DefaultViewport()

	if vp.Width != DefaultViewportWidth {
		t.Errorf("expected width %d, got %d", DefaultViewportWidth, vp.Width)
	}
	if vp.Height != DefaultViewportHeight {
		t.Errorf("expected height %d, got %d", DefaultViewportHeight, vp.Height)
	}
}

// =============================================================================
// DiagnosisType Tests
// =============================================================================

func TestDiagnosisTypeConstants(t *testing.T) {
	tests := []struct {
		diagType DiagnosisType
		expected string
	}{
		{DiagnosisProcessLeak, "process_leak"},
		{DiagnosisMemoryExhaustion, "memory_exhaustion"},
		{DiagnosisChromeCrashes, "chrome_crashes"},
		{DiagnosisDegraded, "degraded"},
		{DiagnosisUnknown, "unknown"},
		{DiagnosisOffline, "offline"},
	}

	for _, tc := range tests {
		if string(tc.diagType) != tc.expected {
			t.Errorf("DiagnosisType = %q, want %q", tc.diagType, tc.expected)
		}
	}
}

// =============================================================================
// Status Tests
// =============================================================================

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusPassed, "passed"},
		{StatusFailed, "failed"},
		{StatusSkipped, "skipped"},
		{StatusBlocked, "blocked"},
	}

	for _, tc := range tests {
		if string(tc.status) != tc.expected {
			t.Errorf("Status = %q, want %q", tc.status, tc.expected)
		}
	}
}

// =============================================================================
// BlockedReason Tests
// =============================================================================

func TestBlockedReasonConstants(t *testing.T) {
	tests := []struct {
		reason   BlockedReason
		expected string
	}{
		{BlockedReasonNone, ""},
		{BlockedReasonBrowserlessOffline, "browserless_offline"},
		{BlockedReasonBundleStale, "bundle_stale"},
		{BlockedReasonUIPortMissing, "ui_port_missing"},
	}

	for _, tc := range tests {
		if string(tc.reason) != tc.expected {
			t.Errorf("BlockedReason = %q, want %q", tc.reason, tc.expected)
		}
	}
}

// =============================================================================
// Default Timeout Constants Tests
// =============================================================================

func TestDefaultTimeoutConstants(t *testing.T) {
	if DefaultTimeout != 90*time.Second {
		t.Errorf("DefaultTimeout = %v, want 90s", DefaultTimeout)
	}
	if DefaultHandshakeTimeout != 15*time.Second {
		t.Errorf("DefaultHandshakeTimeout = %v, want 15s", DefaultHandshakeTimeout)
	}
	if DefaultViewportWidth != 1280 {
		t.Errorf("DefaultViewportWidth = %d, want 1280", DefaultViewportWidth)
	}
	if DefaultViewportHeight != 720 {
		t.Errorf("DefaultViewportHeight = %d, want 720", DefaultViewportHeight)
	}
}
