package smoke

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/structure/types"
)

func TestNewValidator(t *testing.T) {
	var buf bytes.Buffer
	v := NewValidator("/path/to/scenario", "test-scenario", &buf)

	if v == nil {
		t.Fatal("NewValidator returned nil")
	}

	// Verify it returns a validator interface implementation
	_, ok := v.(*validator)
	if !ok {
		t.Error("NewValidator should return *validator")
	}
}

func TestNewValidator_DefaultBrowserlessURL(t *testing.T) {
	// Clear any existing env var
	original := os.Getenv(BrowserlessURLEnvVar)
	os.Unsetenv(BrowserlessURLEnvVar)
	defer func() {
		if original != "" {
			os.Setenv(BrowserlessURLEnvVar, original)
		}
	}()

	var buf bytes.Buffer
	v := NewValidator("/path/to/scenario", "test-scenario", &buf)

	impl, ok := v.(*validator)
	if !ok {
		t.Fatal("NewValidator should return *validator")
	}

	if impl.browserlessURL != DefaultBrowserlessURL {
		t.Errorf("browserlessURL = %q, want %q", impl.browserlessURL, DefaultBrowserlessURL)
	}
}

func TestNewValidator_EnvBrowserlessURL(t *testing.T) {
	original := os.Getenv(BrowserlessURLEnvVar)
	os.Setenv(BrowserlessURLEnvVar, "http://custom:9999")
	defer func() {
		if original != "" {
			os.Setenv(BrowserlessURLEnvVar, original)
		} else {
			os.Unsetenv(BrowserlessURLEnvVar)
		}
	}()

	var buf bytes.Buffer
	v := NewValidator("/path/to/scenario", "test-scenario", &buf)

	impl, ok := v.(*validator)
	if !ok {
		t.Fatal("NewValidator should return *validator")
	}

	if impl.browserlessURL != "http://custom:9999" {
		t.Errorf("browserlessURL = %q, want %q", impl.browserlessURL, "http://custom:9999")
	}
}

func TestWithBrowserlessURL(t *testing.T) {
	var buf bytes.Buffer
	v := NewValidator("/path/to/scenario", "test-scenario", &buf, WithBrowserlessURL("http://override:8888"))

	impl, ok := v.(*validator)
	if !ok {
		t.Fatal("NewValidator should return *validator")
	}

	// Option should override env var
	if impl.browserlessURL != "http://override:8888" {
		t.Errorf("browserlessURL = %q, want %q", impl.browserlessURL, "http://override:8888")
	}
}

func TestValidator_Validate_NoUIDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	var buf bytes.Buffer
	v := NewValidator(tmpDir, "test-scenario", &buf)

	result := v.Validate(context.Background())

	// Should succeed with skip observation (no UI directory)
	if !result.Success {
		t.Errorf("Success = %v, want true (skipped)", result.Success)
	}
	if result.ItemsChecked != 1 {
		t.Errorf("ItemsChecked = %d, want 1", result.ItemsChecked)
	}
	if len(result.Observations) == 0 {
		t.Fatal("Expected at least one observation")
	}
	if result.Observations[0].Type != types.ObservationSkip {
		t.Errorf("Observation type = %v, want ObservationSkip", result.Observations[0].Type)
	}
}

func TestValidator_Validate_DisabledViaConfig(t *testing.T) {
	tmpDir := t.TempDir()
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Disable UI smoke in testing.json
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

	var buf bytes.Buffer
	v := NewValidator(tmpDir, "test-scenario", &buf)

	result := v.Validate(context.Background())

	// Should succeed with skip observation (disabled)
	if !result.Success {
		t.Errorf("Success = %v, want true (disabled)", result.Success)
	}
	if len(result.Observations) == 0 {
		t.Fatal("Expected at least one observation")
	}
	if result.Observations[0].Type != types.ObservationSkip {
		t.Errorf("Observation type = %v, want ObservationSkip", result.Observations[0].Type)
	}
}

func TestValidator_Validate_SuccessObservation(t *testing.T) {
	// This tests the observation message formatting for successful results
	// The actual success path requires a running Browserless, so we just verify
	// the observation message format
	msg := "ui smoke passed (1500ms)"
	obs := types.NewSuccessObservation(msg)

	if obs.Type != types.ObservationSuccess {
		t.Errorf("Type = %v, want ObservationSuccess", obs.Type)
	}
	if obs.Message != msg {
		t.Errorf("Message = %q, want %q", obs.Message, msg)
	}
}

func TestValidator_Validate_ErrorObservation(t *testing.T) {
	// Verify error observation creation
	msg := "browserless offline"
	obs := types.NewErrorObservation(msg)

	if obs.Type != types.ObservationError {
		t.Errorf("Type = %v, want ObservationError", obs.Type)
	}
	if obs.Message != msg {
		t.Errorf("Message = %q, want %q", obs.Message, msg)
	}
}

func TestValidator_ScenarioFields(t *testing.T) {
	var buf bytes.Buffer
	v := NewValidator("/test/scenario/dir", "my-scenario", &buf)

	impl, ok := v.(*validator)
	if !ok {
		t.Fatal("NewValidator should return *validator")
	}

	if impl.scenarioDir != "/test/scenario/dir" {
		t.Errorf("scenarioDir = %q, want %q", impl.scenarioDir, "/test/scenario/dir")
	}
	if impl.scenarioName != "my-scenario" {
		t.Errorf("scenarioName = %q, want %q", impl.scenarioName, "my-scenario")
	}
	if impl.logWriter != &buf {
		t.Error("logWriter should be set to the provided buffer")
	}
}

func TestValidator_MultipleOptions(t *testing.T) {
	var buf bytes.Buffer
	customOpt := func(v *validator) {
		v.scenarioName = "overridden-name"
	}

	v := NewValidator("/path", "original", &buf,
		WithBrowserlessURL("http://custom:1234"),
		customOpt,
	)

	impl, ok := v.(*validator)
	if !ok {
		t.Fatal("NewValidator should return *validator")
	}

	if impl.browserlessURL != "http://custom:1234" {
		t.Errorf("browserlessURL = %q, want %q", impl.browserlessURL, "http://custom:1234")
	}
	if impl.scenarioName != "overridden-name" {
		t.Errorf("scenarioName = %q, want %q", impl.scenarioName, "overridden-name")
	}
}

func TestGetBrowserlessURL_Default(t *testing.T) {
	original := os.Getenv(BrowserlessURLEnvVar)
	os.Unsetenv(BrowserlessURLEnvVar)
	defer func() {
		if original != "" {
			os.Setenv(BrowserlessURLEnvVar, original)
		}
	}()

	url := GetBrowserlessURL()
	if url != DefaultBrowserlessURL {
		t.Errorf("GetBrowserlessURL() = %q, want %q", url, DefaultBrowserlessURL)
	}
}

func TestGetBrowserlessURL_FromEnv(t *testing.T) {
	original := os.Getenv(BrowserlessURLEnvVar)
	os.Setenv(BrowserlessURLEnvVar, "http://env-override:7777")
	defer func() {
		if original != "" {
			os.Setenv(BrowserlessURLEnvVar, original)
		} else {
			os.Unsetenv(BrowserlessURLEnvVar)
		}
	}()

	url := GetBrowserlessURL()
	if url != "http://env-override:7777" {
		t.Errorf("GetBrowserlessURL() = %q, want %q", url, "http://env-override:7777")
	}
}

func TestBrowserlessURLEnvVar_Constant(t *testing.T) {
	// Verify the constant has the expected value
	if BrowserlessURLEnvVar != "BROWSERLESS_URL" {
		t.Errorf("BrowserlessURLEnvVar = %q, want %q", BrowserlessURLEnvVar, "BROWSERLESS_URL")
	}
}
