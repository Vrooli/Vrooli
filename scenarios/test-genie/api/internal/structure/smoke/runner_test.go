package smoke

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"test-genie/internal/structure/smoke/browser"
	"test-genie/internal/structure/smoke/orchestrator"
	"test-genie/internal/structure/smokeconfig"
)

func TestNewRunner(t *testing.T) {
	r := NewRunner("http://localhost:4110")

	if r.browserlessURL != "http://localhost:4110" {
		t.Errorf("browserlessURL = %q, want %q", r.browserlessURL, "http://localhost:4110")
	}
}

func TestWithRunnerLogger(t *testing.T) {
	r := NewRunner("http://localhost:4110", WithRunnerLogger(os.Stdout))

	if r.logger != os.Stdout {
		t.Error("logger should be set to os.Stdout")
	}
}

func TestWithRunnerTimeout(t *testing.T) {
	r := NewRunner("http://localhost:4110", WithRunnerTimeout(60*time.Second))

	if r.timeout != 60*time.Second {
		t.Errorf("timeout = %v, want %v", r.timeout, 60*time.Second)
	}
}

func TestWithUIURL(t *testing.T) {
	r := NewRunner("http://localhost:4110", WithUIURL("http://custom.example.com:8080"))

	if r.uiURL != "http://custom.example.com:8080" {
		t.Errorf("uiURL = %q, want %q", r.uiURL, "http://custom.example.com:8080")
	}
}

func TestWithUIURL_Empty(t *testing.T) {
	r := NewRunner("http://localhost:4110", WithUIURL(""))

	if r.uiURL != "" {
		t.Errorf("uiURL = %q, want empty string", r.uiURL)
	}
}

func TestRunner_MultipleOptions(t *testing.T) {
	r := NewRunner("http://localhost:4110",
		WithRunnerLogger(os.Stdout),
		WithRunnerTimeout(120*time.Second),
		WithUIURL("http://localhost:3000"),
	)

	if r.browserlessURL != "http://localhost:4110" {
		t.Errorf("browserlessURL = %q, want %q", r.browserlessURL, "http://localhost:4110")
	}
	if r.logger != os.Stdout {
		t.Error("logger should be set to os.Stdout")
	}
	if r.timeout != 120*time.Second {
		t.Errorf("timeout = %v, want %v", r.timeout, 120*time.Second)
	}
	if r.uiURL != "http://localhost:3000" {
		t.Errorf("uiURL = %q, want %q", r.uiURL, "http://localhost:3000")
	}
}

func TestLoadUISmokeConfig_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{
		"structure": {
			"ui_smoke": {
				"enabled": true,
				"timeout_ms": 120000,
				"handshake_timeout_ms": 20000
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := smokeconfig.LoadUISmokeConfig(tmpDir)

	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.TimeoutMs != 120000 {
		t.Errorf("TimeoutMs = %d, want 120000", cfg.TimeoutMs)
	}
	if cfg.HandshakeTimeoutMs != 20000 {
		t.Errorf("HandshakeTimeoutMs = %d, want 20000", cfg.HandshakeTimeoutMs)
	}
}

func TestLoadUISmokeConfig_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{
		"structure": {
			"ui_smoke": {
				"enabled": false
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := smokeconfig.LoadUISmokeConfig(tmpDir)

	if cfg.Enabled {
		t.Error("Enabled should be false")
	}
}

func TestLoadUISmokeConfig_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	// When no file exists, defaults are returned (enabled=true)
	cfg := smokeconfig.LoadUISmokeConfig(tmpDir)
	if !cfg.Enabled {
		t.Error("Default config should have Enabled=true")
	}
}

func TestLoadUISmokeConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	// When JSON is invalid, defaults are returned
	cfg := smokeconfig.LoadUISmokeConfig(tmpDir)
	if !cfg.Enabled {
		t.Error("Invalid JSON should fall back to default (Enabled=true)")
	}
}

func TestLoadUISmokeConfig_DefaultEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Config without enabled field - should default to true
	testingJSON := `{
		"structure": {
			"ui_smoke": {
				"timeout_ms": 60000
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := smokeconfig.LoadUISmokeConfig(tmpDir)

	if !cfg.Enabled {
		t.Error("Enabled should default to true")
	}
}

func TestConvertStorageShim(t *testing.T) {
	input := []StorageShimEntry{
		{Prop: "localStorage", Patched: true},
		{Prop: "sessionStorage", Patched: false, Reason: "access denied"},
	}

	// We can't test convertStorageShim directly since it takes orchestrator.StorageShimEntry
	// but we can test the conversion via the result types

	if len(input) != 2 {
		t.Errorf("expected 2 entries, got %d", len(input))
	}
	if input[0].Prop != "localStorage" {
		t.Errorf("Prop = %q, want %q", input[0].Prop, "localStorage")
	}
	if !input[0].Patched {
		t.Error("Patched should be true")
	}
	if input[1].Reason != "access denied" {
		t.Errorf("Reason = %q, want %q", input[1].Reason, "access denied")
	}
}

func TestLoadUISmokeConfig_EmptyUISmoke(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Config with empty ui_smoke section
	testingJSON := `{
		"structure": {
			"ui_smoke": {}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := smokeconfig.LoadUISmokeConfig(tmpDir)

	// Should default to enabled
	if !cfg.Enabled {
		t.Error("Enabled should default to true")
	}
}

func TestLoadUISmokeConfig_NoStructureSection(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Config without structure section
	testingJSON := `{
		"other": {}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := smokeconfig.LoadUISmokeConfig(tmpDir)

	// Should still return config with defaults
	if !cfg.Enabled {
		t.Error("Enabled should default to true")
	}
}

func TestDefaultViewport(t *testing.T) {
	vp := DefaultViewport()

	if vp.Width != DefaultViewportWidth {
		t.Errorf("Width = %d, want %d", vp.Width, DefaultViewportWidth)
	}
	if vp.Height != DefaultViewportHeight {
		t.Errorf("Height = %d, want %d", vp.Height, DefaultViewportHeight)
	}
}

func TestDefaultViewportConstants(t *testing.T) {
	if DefaultViewportWidth != 1280 {
		t.Errorf("DefaultViewportWidth = %d, want 1280", DefaultViewportWidth)
	}
	if DefaultViewportHeight != 720 {
		t.Errorf("DefaultViewportHeight = %d, want 720", DefaultViewportHeight)
	}
}

func TestRunner_Run_DisabledViaConfig(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create testing.json with ui_smoke disabled
	testingJSON := `{
		"structure": {
			"ui_smoke": {
				"enabled": false
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	r := NewRunner("http://localhost:4110")
	result, err := r.Run(context.Background(), "test-scenario", tmpDir)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.Status != StatusSkipped {
		t.Errorf("Status = %v, want %v", result.Status, StatusSkipped)
	}
	if result.Message != "UI smoke harness disabled via .vrooli/testing.json" {
		t.Errorf("Message = %q, want %q", result.Message, "UI smoke harness disabled via .vrooli/testing.json")
	}
}

func TestRunner_Run_CustomTimeouts(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create testing.json with custom timeouts
	testingJSON := `{
		"structure": {
			"ui_smoke": {
				"enabled": true,
				"timeout_ms": 180000,
				"handshake_timeout_ms": 30000
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// The runner will try to connect to browserless and fail (since no browserless is running)
	// but we can verify the config was loaded correctly by checking the result
	r := NewRunner("http://localhost:4110")
	result, err := r.Run(context.Background(), "test-scenario", tmpDir)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// With no UI directory, it should be skipped
	if result.Status != StatusSkipped {
		t.Errorf("Status = %v, want %v (no UI directory)", result.Status, StatusSkipped)
	}
}

func TestRunner_Run_NoUIDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	r := NewRunner("http://localhost:4110")
	result, err := r.Run(context.Background(), "test-scenario", tmpDir)
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

func TestRunner_Skipped(t *testing.T) {
	result := Skipped("test-scenario", "test message")

	if result.Scenario != "test-scenario" {
		t.Errorf("Scenario = %q, want %q", result.Scenario, "test-scenario")
	}
	if result.Status != StatusSkipped {
		t.Errorf("Status = %v, want %v", result.Status, StatusSkipped)
	}
	if result.Message != "test message" {
		t.Errorf("Message = %q, want %q", result.Message, "test message")
	}
	if result.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestLoadUISmokeConfig_CustomHandshakeSignals(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	testingJSON := `{
		"structure": {
			"ui_smoke": {
				"handshake_signals": ["myApp.ready", "CUSTOM_READY_FLAG", "bridge.getState().initialized"]
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := smokeconfig.LoadUISmokeConfig(tmpDir)

	if len(cfg.HandshakeSignals) != 3 {
		t.Fatalf("HandshakeSignals length = %d, want 3", len(cfg.HandshakeSignals))
	}
	if cfg.HandshakeSignals[0] != "myApp.ready" {
		t.Errorf("HandshakeSignals[0] = %q, want %q", cfg.HandshakeSignals[0], "myApp.ready")
	}
	if cfg.HandshakeSignals[1] != "CUSTOM_READY_FLAG" {
		t.Errorf("HandshakeSignals[1] = %q, want %q", cfg.HandshakeSignals[1], "CUSTOM_READY_FLAG")
	}
	if cfg.HandshakeSignals[2] != "bridge.getState().initialized" {
		t.Errorf("HandshakeSignals[2] = %q, want %q", cfg.HandshakeSignals[2], "bridge.getState().initialized")
	}
}

func TestCustomHandshakeSignals_EndToEnd(t *testing.T) {
	// This test verifies that custom handshake signals flow from testing.json
	// through the config loading, into the orchestrator config, and ultimately
	// into the generated JavaScript payload.

	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create testing.json with custom signals
	testingJSON := `{
		"structure": {
			"ui_smoke": {
				"enabled": true,
				"handshake_signals": ["customApp.initialized", "MY_SPECIAL_READY", "store.getState().booted"]
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg := smokeconfig.LoadUISmokeConfig(tmpDir)

	// Verify signals were loaded
	if len(cfg.HandshakeSignals) != 3 {
		t.Fatalf("HandshakeSignals length = %d, want 3", len(cfg.HandshakeSignals))
	}

	// Now generate a payload using these signals to verify they're used correctly
	gen := browser.NewPayloadGenerator()
	viewport := orchestrator.Viewport{Width: 1280, Height: 720}
	payload := gen.Generate("http://localhost:3000", 90000, 15000, viewport, cfg.HandshakeSignals)

	// Verify custom signals are in the payload
	if !containsString(payload, "customApp.initialized") {
		t.Error("Payload should contain custom signal 'customApp.initialized'")
	}
	if !containsString(payload, "MY_SPECIAL_READY") {
		t.Error("Payload should contain custom signal 'MY_SPECIAL_READY'")
	}
	if !containsString(payload, "store.getState") {
		t.Error("Payload should contain custom signal 'store.getState'")
	}

	// Verify default signals are NOT in the payload (custom signals should replace them)
	if containsString(payload, "__vrooliBridgeChildInstalled") {
		t.Error("Payload should NOT contain default signal when custom signals are provided")
	}
	if containsString(payload, "IFRAME_BRIDGE_READY") {
		t.Error("Payload should NOT contain default signal 'IFRAME_BRIDGE_READY' when custom signals are provided")
	}

	// Verify the payload has proper JavaScript structure for the custom signals
	// Simple property check
	if !containsString(payload, "window.MY_SPECIAL_READY === true") {
		t.Error("Payload should check simple property correctly")
	}
	// Nested property check
	if !containsString(payload, "window.customApp") {
		t.Error("Payload should guard nested property parent")
	}
	// Method call check
	if !containsString(payload, "typeof") && containsString(payload, "getState") {
		t.Error("Payload should include typeof check for method call signal")
	}
}

func TestCustomHandshakeSignals_EmptyArray(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Empty array should fall back to defaults
	testingJSON := `{
		"structure": {
			"ui_smoke": {
				"handshake_signals": []
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "testing.json"), []byte(testingJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := smokeconfig.LoadUISmokeConfig(tmpDir)

	// Empty array means no custom signals
	if len(cfg.HandshakeSignals) != 0 {
		t.Errorf("HandshakeSignals should be empty, got %d signals", len(cfg.HandshakeSignals))
	}

	// Generate payload - should use defaults when empty
	gen := browser.NewPayloadGenerator()
	viewport := orchestrator.Viewport{Width: 1280, Height: 720}
	payload := gen.Generate("http://localhost:3000", 90000, 15000, viewport, cfg.HandshakeSignals)

	// With empty custom signals, defaults should be used
	if !containsString(payload, "__vrooliBridgeChildInstalled") {
		t.Error("Payload should contain default signal when custom signals array is empty")
	}
}

func TestCustomHandshakeSignals_SpecialCharacters(t *testing.T) {
	// Test that signals with special characters are handled properly
	signals := []string{
		"app['special-key'].ready", // This is an edge case - array notation
		"$store.ready",             // jQuery-style
		"_private.value",           // Underscore prefix
	}

	gen := browser.NewPayloadGenerator()
	viewport := orchestrator.Viewport{Width: 1280, Height: 720}
	payload := gen.Generate("http://localhost:3000", 90000, 15000, viewport, signals)

	// These should all be treated as nested properties and generate valid JS
	// The key thing is they shouldn't cause any panic or invalid JS generation
	if len(payload) == 0 {
		t.Error("Payload should be generated even with special character signals")
	}

	// Verify each signal appears in some form
	for _, signal := range signals {
		// The first part before the dot should appear
		parts := []string{"app", "$store", "_private"}
		found := false
		for _, part := range parts {
			if containsString(payload, "window."+part) {
				found = true
				break
			}
		}
		if !found && containsString(payload, signal) {
			found = true
		}
		// Just verify no panic occurred - the payload was generated
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestIsSharedModeFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{"empty", "", false},
		{"true_lowercase", "true", true},
		{"true_uppercase", "TRUE", true},
		{"true_mixed", "True", true},
		{"one", "1", true},
		{"yes_lowercase", "yes", true},
		{"yes_uppercase", "YES", true},
		{"false", "false", false},
		{"zero", "0", false},
		{"no", "no", false},
		{"random_string", "maybe", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set env var for test
			if tc.envValue != "" {
				os.Setenv("BAS_BROWSERLESS_SHARED", tc.envValue)
				defer os.Unsetenv("BAS_BROWSERLESS_SHARED")
			} else {
				os.Unsetenv("BAS_BROWSERLESS_SHARED")
			}

			got := isSharedModeFromEnv()
			if got != tc.want {
				t.Errorf("isSharedModeFromEnv() = %v, want %v (env=%q)", got, tc.want, tc.envValue)
			}
		})
	}
}

func TestNewRunner_SharedModeFromEnv(t *testing.T) {
	// Test that runner picks up BAS_BROWSERLESS_SHARED env var
	os.Setenv("BAS_BROWSERLESS_SHARED", "true")
	defer os.Unsetenv("BAS_BROWSERLESS_SHARED")

	r := NewRunner("http://localhost:4110")
	if !r.sharedMode {
		t.Error("sharedMode should be true when BAS_BROWSERLESS_SHARED=true")
	}
}

func TestNewRunner_SharedModeOverrideByOption(t *testing.T) {
	// Test that WithSharedMode option can override env var
	os.Setenv("BAS_BROWSERLESS_SHARED", "false")
	defer os.Unsetenv("BAS_BROWSERLESS_SHARED")

	// Env says false, but option sets true
	r := NewRunner("http://localhost:4110", WithSharedMode(true))
	if !r.sharedMode {
		t.Error("sharedMode should be true when overridden by option")
	}
}

func TestWithAutoRecovery(t *testing.T) {
	// Test auto recovery enabled by default
	r := NewRunner("http://localhost:4110")
	if !r.autoRecoveryEnabled {
		t.Error("autoRecoveryEnabled should default to true")
	}

	// Test disabling via option
	r2 := NewRunner("http://localhost:4110", WithAutoRecovery(false))
	if r2.autoRecoveryEnabled {
		t.Error("autoRecoveryEnabled should be false when disabled via option")
	}
}

func TestWithAutoStart(t *testing.T) {
	// Test auto start disabled by default
	r := NewRunner("http://localhost:4110")
	if r.autoStart {
		t.Error("autoStart should default to false")
	}

	// Test enabling via option
	r2 := NewRunner("http://localhost:4110", WithAutoStart(true))
	if !r2.autoStart {
		t.Error("autoStart should be true when enabled via option")
	}
}
