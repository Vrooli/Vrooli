package smoketest_test

import (
	"encoding/json"
	"scenario-to-desktop-api/smoketest"
	"strings"
	"testing"
	"time"
)

func TestGenerateFailureReport_BasicFields(t *testing.T) {
	startTime := time.Now().Add(-30 * time.Second)
	completedTime := time.Now()
	errorKind := smoketest.ErrKindExecution

	status := &smoketest.Status{
		SmokeTestID:     "test-123",
		ScenarioName:    "my-scenario",
		Platform:        "linux",
		CurrentState:    smoketest.StateFailed,
		Error:           "smoke test failed",
		ErrorKind:       &errorKind,
		SuggestedAction: "Check app startup logs",
		ErrorContext:    map[string]string{"command": "/app/test"},
		StartedAt:       startTime,
		CompletedAt:     &completedTime,
		Transitions: []smoketest.StateTransition{
			{From: "", To: smoketest.StateInitializing, Timestamp: startTime},
			{From: smoketest.StateInitializing, To: smoketest.StateFailed, Timestamp: completedTime, Message: "failed"},
		},
	}

	report := smoketest.GenerateFailureReport(status)

	if report.SmokeTestID != "test-123" {
		t.Errorf("SmokeTestID = %q, want %q", report.SmokeTestID, "test-123")
	}
	if report.ScenarioName != "my-scenario" {
		t.Errorf("ScenarioName = %q, want %q", report.ScenarioName, "my-scenario")
	}
	if report.Platform != "linux" {
		t.Errorf("Platform = %q, want %q", report.Platform, "linux")
	}
	if report.FinalState != smoketest.StateFailed {
		t.Errorf("FinalState = %q, want %q", report.FinalState, smoketest.StateFailed)
	}
	if len(report.Transitions) != 2 {
		t.Errorf("Transitions length = %d, want %d", len(report.Transitions), 2)
	}
	if report.ErrorMessage != "smoke test failed" {
		t.Errorf("ErrorMessage = %q, want %q", report.ErrorMessage, "smoke test failed")
	}
	if report.SuggestedAction != "Check app startup logs" {
		t.Errorf("SuggestedAction = %q, want %q", report.SuggestedAction, "Check app startup logs")
	}
	if report.ErrorContext["command"] != "/app/test" {
		t.Errorf("ErrorContext[command] = %q, want %q", report.ErrorContext["command"], "/app/test")
	}
}

func TestGenerateFailureReport_ErrorKind(t *testing.T) {
	tests := []struct {
		name      string
		errorKind *smoketest.ErrorKind
		wantKind  smoketest.ErrorKind
	}{
		{
			name:      "with ErrorKind set",
			errorKind: ptrErrorKind(smoketest.ErrKindTimeout),
			wantKind:  smoketest.ErrKindTimeout,
		},
		{
			name:      "with nil ErrorKind",
			errorKind: nil,
			wantKind:  smoketest.ErrorKind(0), // Zero value when nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := &smoketest.Status{
				SmokeTestID:  "test-123",
				ScenarioName: "test-scenario",
				Platform:     "linux",
				CurrentState: smoketest.StateFailed,
				ErrorKind:    tt.errorKind,
				StartedAt:    time.Now(),
			}

			report := smoketest.GenerateFailureReport(status)

			if report.ErrorKind != tt.wantKind {
				t.Errorf("ErrorKind = %v, want %v", report.ErrorKind, tt.wantKind)
			}
		})
	}
}

func TestGenerateFailureReport_CompletedAt(t *testing.T) {
	tests := []struct {
		name         string
		completedAt  *time.Time
		wantDuration bool
		wantFailedAt bool
	}{
		{
			name:         "with CompletedAt set",
			completedAt:  ptrTime(time.Now()),
			wantDuration: true,
			wantFailedAt: true,
		},
		{
			name:         "with nil CompletedAt",
			completedAt:  nil,
			wantDuration: false,
			wantFailedAt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime := time.Now().Add(-30 * time.Second)
			status := &smoketest.Status{
				SmokeTestID:  "test-123",
				ScenarioName: "test-scenario",
				Platform:     "linux",
				CurrentState: smoketest.StateFailed,
				StartedAt:    startTime,
				CompletedAt:  tt.completedAt,
			}

			report := smoketest.GenerateFailureReport(status)

			if tt.wantDuration {
				if report.Duration <= 0 {
					t.Errorf("Duration = %v, want > 0", report.Duration)
				}
			} else {
				if report.Duration != 0 {
					t.Errorf("Duration = %v, want 0", report.Duration)
				}
			}

			if tt.wantFailedAt {
				if report.FailedAt.IsZero() {
					t.Error("FailedAt should not be zero when CompletedAt is set")
				}
			} else {
				if !report.FailedAt.IsZero() {
					t.Error("FailedAt should be zero when CompletedAt is nil")
				}
			}
		})
	}
}

func TestGenerateDiagnosticHints_AllErrorKinds(t *testing.T) {
	tests := []struct {
		errorKind     smoketest.ErrorKind
		platform      string
		artifactPath  string
		wantHintCount int // Minimum expected hints
		wantContains  []string
	}{
		{
			errorKind:     smoketest.ErrKindArtifact,
			artifactPath:  "/path/to/artifact.AppImage",
			wantHintCount: 2,
			wantContains:  []string{"build stage", "permissions"},
		},
		{
			errorKind:     smoketest.ErrKindExecution,
			wantHintCount: 3,
			wantContains:  []string{"entry point", "shared libraries", "--smoke-test"},
		},
		{
			errorKind:     smoketest.ErrKindTimeout,
			wantHintCount: 3,
			wantContains:  []string{"waiting", "SMOKE_TEST_TIMEOUT_MS"},
		},
		{
			errorKind:     smoketest.ErrKindValidation,
			wantHintCount: 3,
			wantContains:  []string{"SMOKE_TEST_RESULT=passed", "smoke test mode"},
		},
		{
			errorKind:     smoketest.ErrKindPlatform,
			platform:      "linux",
			wantHintCount: 2,
			wantContains:  []string{"xvfb", "DISPLAY"},
		},
		{
			errorKind:     smoketest.ErrKindPlatform,
			platform:      "mac",
			wantHintCount: 2,
			wantContains:  []string{"macOS", "signed"},
		},
		{
			errorKind:     smoketest.ErrKindPlatform,
			platform:      "win",
			wantHintCount: 2,
			wantContains:  []string{"Windows", "headless"},
		},
		{
			errorKind:     smoketest.ErrKindTelemetry,
			wantHintCount: 3,
			wantContains:  []string{"telemetry service", "network connectivity"},
		},
		{
			errorKind:     smoketest.ErrKindStore,
			wantHintCount: 2,
			wantContains:  []string{"disk space", "permissions"},
		},
		{
			errorKind:     smoketest.ErrKindCancelled,
			wantHintCount: 2,
			wantContains:  []string{"cancelled", "Re-run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.errorKind.String(), func(t *testing.T) {
			errorKind := tt.errorKind
			status := &smoketest.Status{
				SmokeTestID:  "test-123",
				ScenarioName: "test-scenario",
				Platform:     tt.platform,
				CurrentState: smoketest.StateFailed,
				ErrorKind:    &errorKind,
				ArtifactPath: tt.artifactPath,
				StartedAt:    time.Now(),
			}

			report := smoketest.GenerateFailureReport(status)

			if len(report.DiagnosticHints) < tt.wantHintCount {
				t.Errorf("DiagnosticHints count = %d, want >= %d", len(report.DiagnosticHints), tt.wantHintCount)
			}

			hintsStr := strings.Join(report.DiagnosticHints, " ")
			for _, want := range tt.wantContains {
				if !strings.Contains(strings.ToLower(hintsStr), strings.ToLower(want)) {
					t.Errorf("DiagnosticHints should contain %q, got: %v", want, report.DiagnosticHints)
				}
			}
		})
	}
}

func TestGenerateDiagnosticHints_NilErrorKind(t *testing.T) {
	status := &smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Platform:     "linux",
		CurrentState: smoketest.StateFailed,
		ErrorKind:    nil,
		StartedAt:    time.Now(),
	}

	report := smoketest.GenerateFailureReport(status)

	if len(report.DiagnosticHints) != 0 {
		t.Errorf("DiagnosticHints should be empty for nil ErrorKind, got %v", report.DiagnosticHints)
	}
}

func TestFormatForTerminal_AllSections(t *testing.T) {
	startTime := time.Now().Add(-30 * time.Second)
	completedTime := time.Now()
	errorKind := smoketest.ErrKindExecution

	status := &smoketest.Status{
		SmokeTestID:     "test-abc-123",
		ScenarioName:    "my-test-scenario",
		Platform:        "linux",
		CurrentState:    smoketest.StateFailed,
		Error:           "process crashed",
		ErrorKind:       &errorKind,
		SuggestedAction: "Check the logs",
		ErrorContext:    map[string]string{"exit_code": "1", "command": "test-app"},
		StartedAt:       startTime,
		CompletedAt:     &completedTime,
		Transitions: []smoketest.StateTransition{
			{From: "", To: smoketest.StateInitializing, Timestamp: startTime, Message: "starting"},
			{From: smoketest.StateInitializing, To: smoketest.StateFailed, Timestamp: completedTime, Message: "crashed"},
		},
	}

	report := smoketest.GenerateFailureReport(status)
	output := report.FormatForTerminal()

	// Check header
	if !strings.Contains(output, "SMOKE TEST FAILURE REPORT") {
		t.Error("Output should contain header")
	}

	// Check basic fields
	if !strings.Contains(output, "test-abc-123") {
		t.Error("Output should contain SmokeTestID")
	}
	if !strings.Contains(output, "my-test-scenario") {
		t.Error("Output should contain ScenarioName")
	}
	if !strings.Contains(output, "linux") {
		t.Error("Output should contain Platform")
	}

	// Check error section
	if !strings.Contains(output, "Error") {
		t.Error("Output should contain Error section")
	}
	if !strings.Contains(output, "execution") {
		t.Error("Output should contain ErrorKind string")
	}
	if !strings.Contains(output, "process crashed") {
		t.Error("Output should contain error message")
	}

	// Check context section
	if !strings.Contains(output, "Context") {
		t.Error("Output should contain Context section")
	}
	if !strings.Contains(output, "exit_code") {
		t.Error("Output should contain error context keys")
	}

	// Check state timeline
	if !strings.Contains(output, "State Timeline") {
		t.Error("Output should contain State Timeline section")
	}
	if !strings.Contains(output, "initializing") {
		t.Error("Output should contain state transitions")
	}

	// Check suggested action
	if !strings.Contains(output, "Suggested Action") {
		t.Error("Output should contain Suggested Action section")
	}
	if !strings.Contains(output, "Check the logs") {
		t.Error("Output should contain suggested action text")
	}

	// Check diagnostic hints
	if !strings.Contains(output, "Diagnostic Hints") {
		t.Error("Output should contain Diagnostic Hints section")
	}
}

func TestFormatForTerminal_EmptyOptionalSections(t *testing.T) {
	status := &smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Platform:     "linux",
		CurrentState: smoketest.StateFailed,
		Error:        "simple error",
		StartedAt:    time.Now(),
		// No ErrorContext, no Transitions, no SuggestedAction, no ErrorKind
	}

	report := smoketest.GenerateFailureReport(status)
	output := report.FormatForTerminal()

	// Should still have basic sections
	if !strings.Contains(output, "SMOKE TEST FAILURE REPORT") {
		t.Error("Output should contain header")
	}
	if !strings.Contains(output, "test-123") {
		t.Error("Output should contain SmokeTestID")
	}

	// Context section should not appear when ErrorContext is empty
	if strings.Contains(output, "--- Context ---") {
		t.Error("Output should not contain Context section when ErrorContext is empty")
	}

	// State Timeline should not appear when Transitions is empty
	if strings.Contains(output, "--- State Timeline ---") {
		t.Error("Output should not contain State Timeline section when Transitions is empty")
	}

	// Suggested Action should not appear when SuggestedAction is empty
	if strings.Contains(output, "--- Suggested Action ---") {
		t.Error("Output should not contain Suggested Action section when SuggestedAction is empty")
	}

	// Diagnostic Hints should not appear when ErrorKind is nil
	if strings.Contains(output, "--- Diagnostic Hints ---") {
		t.Error("Output should not contain Diagnostic Hints section when ErrorKind is nil")
	}
}

func TestFormatForJSON(t *testing.T) {
	errorKind := smoketest.ErrKindValidation
	completedTime := time.Now()

	status := &smoketest.Status{
		SmokeTestID:     "test-json-123",
		ScenarioName:    "json-scenario",
		Platform:        "linux",
		CurrentState:    smoketest.StateFailed,
		Error:           "validation failed",
		ErrorKind:       &errorKind,
		SuggestedAction: "Fix validation",
		ErrorContext:    map[string]string{"marker": "missing"},
		StartedAt:       time.Now().Add(-10 * time.Second),
		CompletedAt:     &completedTime,
		Transitions: []smoketest.StateTransition{
			{From: "", To: smoketest.StateInitializing, Timestamp: time.Now()},
		},
	}

	report := smoketest.GenerateFailureReport(status)

	// FormatForJSON should return the report itself (identity)
	jsonReport := report.FormatForJSON()
	if jsonReport != report {
		t.Error("FormatForJSON should return the same report")
	}

	// Verify it can be marshaled to JSON
	data, err := json.Marshal(jsonReport)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	expectedFields := []string{
		"smoke_test_id",
		"scenario_name",
		"platform",
		"final_state",
		"error_kind",
		"error_message",
		"suggested_action",
		"transitions",
		"diagnostic_hints",
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON should contain field %q", field)
		}
	}

	// Verify values
	if !strings.Contains(jsonStr, "test-json-123") {
		t.Error("JSON should contain smoke test ID value")
	}
	if !strings.Contains(jsonStr, "json-scenario") {
		t.Error("JSON should contain scenario name value")
	}
}
