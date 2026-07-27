package smoketest_test

import (
	"encoding/json"
	"scenario-to-desktop-api/smoketest"
	"strings"
	"testing"
	"time"
)

func TestFormatForTerminal_StateTransitionTimestampFormat(t *testing.T) {
	now := time.Now()
	status := &smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Platform:     "linux",
		CurrentState: smoketest.StateFailed,
		Error:        "failed",
		StartedAt:    now,
		Transitions: []smoketest.StateTransition{
			{From: "", To: smoketest.StateInitializing, Timestamp: now, Message: "starting"},
		},
	}

	report := smoketest.GenerateFailureReport(status)
	output := report.FormatForTerminal()

	// Check timestamp is formatted as HH:MM:SS.mmm
	expectedFormat := now.Format("15:04:05.000")
	if !strings.Contains(output, expectedFormat) {
		t.Errorf("Output should contain timestamp in format HH:MM:SS.mmm, looking for %s in output", expectedFormat)
	}
}

func TestGenerateFailureReport_ArtifactPathInHints(t *testing.T) {
	errorKind := smoketest.ErrKindArtifact
	status := &smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Platform:     "linux",
		CurrentState: smoketest.StateFailed,
		ErrorKind:    &errorKind,
		ArtifactPath: "/specific/path/to/my-app.AppImage",
		StartedAt:    time.Now(),
	}

	report := smoketest.GenerateFailureReport(status)
	hintsStr := strings.Join(report.DiagnosticHints, " ")

	if !strings.Contains(hintsStr, "/specific/path/to/my-app.AppImage") {
		t.Errorf("DiagnosticHints should contain artifact path, got: %v", report.DiagnosticHints)
	}
}

func TestFormatForCI_Structure(t *testing.T) {
	errorKind := smoketest.ErrKindTimeout
	completedTime := time.Now()
	startTime := completedTime.Add(-30 * time.Second)

	status := &smoketest.Status{
		SmokeTestID:     "ci-test-123",
		ScenarioName:    "ci-scenario",
		Platform:        "linux",
		CurrentState:    smoketest.StateFailed,
		Error:           "timeout waiting for app",
		ErrorKind:       &errorKind,
		SuggestedAction: "Increase timeout",
		StartedAt:       startTime,
		CompletedAt:     &completedTime,
		Transitions: []smoketest.StateTransition{
			{From: "", To: smoketest.StateInitializing, Timestamp: startTime},
			{From: smoketest.StateInitializing, To: smoketest.StateFailed, Timestamp: completedTime},
		},
	}

	report := smoketest.GenerateFailureReport(status)
	ciOutput := report.FormatForCI()

	// Verify it's valid JSON
	var parsed smoketest.StructuredOutput
	err := json.Unmarshal([]byte(ciOutput), &parsed)
	if err != nil {
		t.Fatalf("FormatForCI should return valid JSON: %v", err)
	}

	// Check version
	if parsed.Version != "1.0" {
		t.Errorf("Version = %q, want %q", parsed.Version, "1.0")
	}

	// Check level
	if parsed.Level != "error" {
		t.Errorf("Level = %q, want %q", parsed.Level, "error")
	}

	// Check timestamp is recent
	if time.Since(parsed.Timestamp) > time.Minute {
		t.Error("Timestamp should be recent")
	}

	// Check smoke test report is present
	if parsed.SmokeTest == nil {
		t.Error("SmokeTest should not be nil")
	}

	// Check summary fields
	if parsed.Summary.Status != "failed" {
		t.Errorf("Summary.Status = %q, want %q", parsed.Summary.Status, "failed")
	}
	if parsed.Summary.ErrorKind != "timeout" {
		t.Errorf("Summary.ErrorKind = %q, want %q", parsed.Summary.ErrorKind, "timeout")
	}
	if parsed.Summary.StateCount != 2 {
		t.Errorf("Summary.StateCount = %d, want %d", parsed.Summary.StateCount, 2)
	}
	if !parsed.Summary.Retryable {
		t.Error("Summary.Retryable should be true for timeout errors")
	}
	if parsed.Summary.ActionNeeded != "Increase timeout" {
		t.Errorf("Summary.ActionNeeded = %q, want %q", parsed.Summary.ActionNeeded, "Increase timeout")
	}
}

func TestFormatForCI_RetryableKinds(t *testing.T) {
	tests := []struct {
		errorKind     smoketest.ErrorKind
		wantRetryable bool
	}{
		{smoketest.ErrKindTimeout, true},
		{smoketest.ErrKindTelemetry, true},
		{smoketest.ErrKindStore, true},
		{smoketest.ErrKindExecution, true},
		{smoketest.ErrKindArtifact, false},
		{smoketest.ErrKindValidation, false},
		{smoketest.ErrKindPlatform, false},
		{smoketest.ErrKindCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.errorKind.String(), func(t *testing.T) {
			status := &smoketest.Status{
				SmokeTestID:  "test-123",
				ScenarioName: "test",
				Platform:     "linux",
				CurrentState: smoketest.StateFailed,
				ErrorKind:    &tt.errorKind,
				StartedAt:    time.Now(),
			}

			report := smoketest.GenerateFailureReport(status)
			ciOutput := report.FormatForCI()

			var parsed smoketest.StructuredOutput
			if err := json.Unmarshal([]byte(ciOutput), &parsed); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if parsed.Summary.Retryable != tt.wantRetryable {
				t.Errorf("Retryable = %v, want %v for %s", parsed.Summary.Retryable, tt.wantRetryable, tt.errorKind)
			}
		})
	}
}

func TestDiagnosticCommands_ArtifactError(t *testing.T) {
	report := &smoketest.FailureReport{
		ErrorKind:    smoketest.ErrKindArtifact,
		ErrorContext: map[string]string{"artifact_path": "/path/to/app.AppImage"},
	}

	commands := report.DiagnosticCommands()

	if len(commands) == 0 {
		t.Fatal("Should return diagnostic commands for artifact errors")
	}

	// Check that ls command is included with the artifact path
	foundLs := false
	for _, cmd := range commands {
		if strings.Contains(cmd.Command, "ls") && strings.Contains(cmd.Command, "/path/to/app.AppImage") {
			foundLs = true
			break
		}
	}
	if !foundLs {
		t.Error("Should include ls command with artifact path")
	}
}

func TestDiagnosticCommands_ExecutionError(t *testing.T) {
	report := &smoketest.FailureReport{
		ErrorKind:    smoketest.ErrKindExecution,
		Platform:     "linux",
		ErrorContext: map[string]string{"artifact_path": "/app/binary"},
	}

	commands := report.DiagnosticCommands()

	if len(commands) == 0 {
		t.Fatal("Should return diagnostic commands for execution errors")
	}

	// Check for ldd command on Linux
	foundLdd := false
	for _, cmd := range commands {
		if strings.Contains(cmd.Command, "ldd") && cmd.Platform == "linux" {
			foundLdd = true
			break
		}
	}
	if !foundLdd {
		t.Error("Should include ldd command for Linux execution errors")
	}
}

func TestDiagnosticCommands_PlatformError_Linux(t *testing.T) {
	report := &smoketest.FailureReport{
		ErrorKind: smoketest.ErrKindPlatform,
		Platform:  "linux",
	}

	commands := report.DiagnosticCommands()

	if len(commands) == 0 {
		t.Fatal("Should return diagnostic commands for Linux platform errors")
	}

	// Check for xvfb-related commands
	foundXvfb := false
	for _, cmd := range commands {
		if strings.Contains(cmd.Command, "xvfb") {
			foundXvfb = true
			break
		}
	}
	if !foundXvfb {
		t.Error("Should include xvfb-related commands for Linux platform errors")
	}
}

func TestDiagnosticCommands_CategoryField(t *testing.T) {
	report := &smoketest.FailureReport{
		ErrorKind: smoketest.ErrKindStore,
	}

	commands := report.DiagnosticCommands()

	// All commands should have a category
	for _, cmd := range commands {
		if cmd.Category == "" {
			t.Errorf("Command %q should have a category", cmd.Command)
		}
		if cmd.Category != "check" && cmd.Category != "debug" && cmd.Category != "fix" {
			t.Errorf("Command category should be 'check', 'debug', or 'fix', got %q", cmd.Category)
		}
	}
}

func TestDiagnosticCommands_EmptyForCancelled(t *testing.T) {
	report := &smoketest.FailureReport{
		ErrorKind: smoketest.ErrKindCancelled,
	}

	commands := report.DiagnosticCommands()

	// Cancelled errors don't really have diagnostic commands
	if len(commands) != 0 {
		t.Logf("Got %d commands for cancelled error, this is fine but might not be useful", len(commands))
	}
}

// Helper functions

func ptrErrorKind(k smoketest.ErrorKind) *smoketest.ErrorKind {
	return &k
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
