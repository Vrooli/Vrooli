package smoketest_test

import (
	"context"
	"testing"
	"time"

	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
)

// TestService_SessionIDExtraction validates that session ID is extracted from output
// and used for telemetry error correlation.
func TestService_SessionIDExtraction(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
		StartedAt:    time.Now(),
	})

	fs := mocks.NewMockFileSystem()
	fs.AddFile("/path/to/artifact.AppImage", []byte{})

	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.ResolveResult = struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}{
		Cmd:     "/path/to/artifact.AppImage",
		Args:    []string{"--smoke-test"},
		Display: "/path/to/artifact.AppImage --smoke-test",
	}

	executor := mocks.NewMockProcessExecutor()
	// Output contains session ID in the SMOKE_TEST_INIT marker
	executor.ExecuteResult.Output = "SMOKE_TEST_INIT=started session_id=abc12345-6789-0def-abcd-123456789abc\nSMOKE_TEST_RESULT=passed"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true}
	// Return session ID when ExtractSessionID is called
	outputParser.SessionIDResult = "abc12345-6789-0def-abcd-123456789abc"

	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	telemetryResolver.ExtractResult = "/tmp/telemetry.jsonl"

	telemetryExtractor := mocks.NewMockTelemetryErrorExtractor()
	// Session-matching error
	telemetryExtractor.ExtractLatestErrorForSessionResult.Error = &smoketest.TelemetryError{
		Event:     "smoke_test_failed",
		Message:   "Bundled payload is missing",
		SessionID: "abc12345-6789-0def-abcd-123456789abc",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
		s.telemetryResolver = telemetryResolver
		s.telemetryExtractor = telemetryExtractor
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")

	// Verify session ID was extracted and stored
	if status.AppSessionID != "abc12345-6789-0def-abcd-123456789abc" {
		t.Errorf("Expected AppSessionID to be abc12345-6789-0def-abcd-123456789abc, got %q", status.AppSessionID)
	}

	// Verify error was extracted
	if status.AppReportedError == nil {
		t.Error("Expected AppReportedError to be set")
	} else if status.AppReportedError.Message != "Bundled payload is missing" {
		t.Errorf("Expected error message 'Bundled payload is missing', got %q", status.AppReportedError.Message)
	}

	// Verify session-filtered extraction was called with correct session ID
	if len(telemetryExtractor.ExtractLatestErrorForSessionCalls) == 0 {
		t.Error("Expected ExtractLatestErrorForSession to be called")
	} else {
		call := telemetryExtractor.ExtractLatestErrorForSessionCalls[0]
		if call.SessionID != "abc12345-6789-0def-abcd-123456789abc" {
			t.Errorf("Expected session ID abc12345-6789-0def-abcd-123456789abc, got %q", call.SessionID)
		}
	}

	// Should NOT be flagged as mismatch since session matches
	if status.ErrorSessionMismatch {
		t.Error("Expected ErrorSessionMismatch to be false when session matches")
	}
}

// TestService_SessionMismatchDetection validates that when no session-matching error
// is found, we fall back to latest error but flag it as a mismatch.
func TestService_SessionMismatchDetection(t *testing.T) {
	startTime := time.Now()
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
		StartedAt:    startTime,
	})

	fs := mocks.NewMockFileSystem()
	fs.AddFile("/path/to/artifact.AppImage", []byte{})

	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.ResolveResult = struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}{
		Cmd:     "/path/to/artifact.AppImage",
		Args:    []string{"--smoke-test"},
		Display: "/path/to/artifact.AppImage --smoke-test",
	}

	executor := mocks.NewMockProcessExecutor()
	executor.ExecuteResult.Output = "SMOKE_TEST_INIT=started session_id=current-session-id\nSMOKE_TEST_RESULT=passed"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true}
	outputParser.SessionIDResult = "current-session-id"

	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	telemetryResolver.ExtractResult = "/tmp/telemetry.jsonl"

	telemetryExtractor := mocks.NewMockTelemetryErrorExtractor()
	// No session-matching error found (returns nil)
	telemetryExtractor.ExtractLatestErrorForSessionResult.Error = nil
	// But there IS an error from a previous session
	telemetryExtractor.ExtractLatestErrorResult.Error = &smoketest.TelemetryError{
		Event:     "smoke_test_failed",
		Message:   "Error from previous session",
		SessionID: "old-session-id",
		Timestamp: startTime.Add(-1 * time.Hour).Format(time.RFC3339), // 1 hour before test
	}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
		s.telemetryResolver = telemetryResolver
		s.telemetryExtractor = telemetryExtractor
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")

	// Should have the fallback error but flagged as mismatch
	if status.AppReportedError == nil {
		t.Error("Expected AppReportedError to be set (fallback)")
	} else if status.AppReportedError.Message != "Error from previous session" {
		t.Errorf("Expected fallback error message, got %q", status.AppReportedError.Message)
	}

	// SHOULD be flagged as mismatch
	if !status.ErrorSessionMismatch {
		t.Error("Expected ErrorSessionMismatch to be true when session doesn't match")
	}

	// SHOULD be flagged as stale (timestamp before smoke test start)
	if !status.AppReportedErrorStale {
		t.Error("Expected AppReportedErrorStale to be true when error predates test")
	}
}

// TestService_SessionID_Integration_WithRealParser tests that session ID extraction
// works with the REAL DefaultOutputParser, not mocks. This catches bugs where
// mocks return preset values but the actual regex extraction fails.
func TestService_SessionID_Integration_WithRealParser(t *testing.T) {
	// Use REAL output parser - this is the key difference from other tests
	realParser := smoketest.NewOutputParser(smoketest.DefaultConfig())

	// Sample output that matches actual desktop app format
	testOutput := `SMOKE_TEST_INIT=started session_id=abc12345-6789-0def-abcd-123456789abc
[Desktop App] Telemetry initialized at /home/user/.config/app/deployment-telemetry.jsonl
[Desktop App] Bundle found at: /tmp/.mount_App123/resources/bundle
SMOKE_TEST_STAGE=bundle_resolving
[Desktop App] Pre-flight bundle validation passed
SMOKE_TEST_STAGE=runtime_starting
SMOKE_TEST_STAGE=waiting_for_token
runtime ready — IPC listening on 127.0.0.1:41605 (dry-run=false)
SMOKE_TEST_RESULT=passed`

	// Verify the real parser extracts session ID correctly
	extractedID := realParser.ExtractSessionID(testOutput)
	if extractedID != "abc12345-6789-0def-abcd-123456789abc" {
		t.Fatalf("Real parser failed to extract session ID: got %q, want %q",
			extractedID, "abc12345-6789-0def-abcd-123456789abc")
	}

	// Verify lifecycle state extraction works
	lifecycleState := realParser.ExtractLastLifecycleState(testOutput)
	if lifecycleState != "result" { // "result" because SMOKE_TEST_RESULT marker is present
		t.Fatalf("Real parser lifecycle state: got %q, want %q", lifecycleState, "result")
	}
}

// TestService_SessionID_Integration_WithEmptyOutput verifies that session ID extraction
// gracefully handles empty or malformed output.
func TestService_SessionID_Integration_WithEmptyOutput(t *testing.T) {
	realParser := smoketest.NewOutputParser(smoketest.DefaultConfig())

	tests := []struct {
		name     string
		output   string
		wantID   string
		wantDesc string
	}{
		{
			name:   "empty output",
			output: "",
			wantID: "",
		},
		{
			name:   "output without session ID marker",
			output: "SMOKE_TEST_INIT=started\nSome other output",
			wantID: "",
		},
		{
			name:   "malformed session ID (uppercase)",
			output: "SMOKE_TEST_INIT=started session_id=ABC12345-6789-0DEF-ABCD-123456789ABC",
			wantID: "", // regex only matches lowercase
		},
		{
			name:   "partial output with session ID",
			output: "SMOKE_TEST_INIT=started session_id=a1b2c3d4-5678-90ab-cdef-1234567890ab",
			wantID: "a1b2c3d4-5678-90ab-cdef-1234567890ab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := realParser.ExtractSessionID(tt.output)
			if got != tt.wantID {
				t.Errorf("ExtractSessionID() = %q, want %q", got, tt.wantID)
			}
		})
	}
}

// TestService_StalenessDetection_Integration tests the full staleness detection flow
// using real timestamp parsing.
func TestService_StalenessDetection_Integration(t *testing.T) {
	// Test case: Error timestamp before smoke test start time
	staleError := &smoketest.TelemetryError{
		Event:     "smoke_test_failed",
		Message:   "Bundled payload is missing",
		SessionID: "old-session-id",
		Timestamp: "2026-02-02T10:00:00.000Z", // 10:00 UTC
	}

	// Smoke test started at 15:00 UTC
	smokeTestStart, _ := time.Parse(time.RFC3339, "2026-02-02T15:00:00Z")

	isStale := smoketest.IsErrorStale(staleError, smokeTestStart)
	if !isStale {
		t.Error("Expected error to be stale when timestamp (10:00) is before smoke test start (15:00)")
	}

	// Test case: Error timestamp after smoke test start time
	freshError := &smoketest.TelemetryError{
		Event:     "smoke_test_failed",
		Message:   "Network error",
		SessionID: "current-session-id",
		Timestamp: "2026-02-02T16:00:00.000Z", // 16:00 UTC
	}

	isStale = smoketest.IsErrorStale(freshError, smokeTestStart)
	if isStale {
		t.Error("Expected error to NOT be stale when timestamp (16:00) is after smoke test start (15:00)")
	}

	// Test case: Error timestamp with milliseconds format
	millisecondsError := &smoketest.TelemetryError{
		Event:     "smoke_test_failed",
		Message:   "Bundle validation failed",
		SessionID: "test-session",
		Timestamp: "2026-02-02T12:30:45.123Z", // Before smoke test
	}

	isStale = smoketest.IsErrorStale(millisecondsError, smokeTestStart)
	if !isStale {
		t.Error("Expected error with milliseconds timestamp to be detected as stale")
	}
}
