package smoketest_test

import (
	"context"
	"errors"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
	"testing"
	"time"
)

// Test that Status fields work correctly
func TestStatus_Fields(t *testing.T) {
	now := time.Now()
	completed := now.Add(30 * time.Second)
	status := &smoketest.Status{
		SmokeTestID:          "test-123",
		ScenarioName:         "my-scenario",
		Platform:             "linux",
		Status:               "passed",
		ArtifactPath:         "/path/to/artifact",
		StartedAt:            now,
		CompletedAt:          &completed,
		TelemetryUploaded:    true,
		TelemetryUploadError: "",
	}

	if status.SmokeTestID != "test-123" {
		t.Errorf("SmokeTestID = %q, want %q", status.SmokeTestID, "test-123")
	}
	if status.ScenarioName != "my-scenario" {
		t.Errorf("ScenarioName = %q, want %q", status.ScenarioName, "my-scenario")
	}
	if status.Platform != "linux" {
		t.Errorf("Platform = %q, want %q", status.Platform, "linux")
	}
	if status.Status != "passed" {
		t.Errorf("Status = %q, want %q", status.Status, "passed")
	}
	if !status.TelemetryUploaded {
		t.Errorf("expected TelemetryUploaded to be true")
	}
}

func TestService_StateTransitions_Success(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
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
	executor.ExecuteResult.Output = "SMOKE_TEST_RESULT=passed\nSMOKE_TEST_UPLOAD=ok"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{
		Passed:            true,
		TelemetryUploaded: true,
	}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")

	// Verify all transitions are valid
	for i := 0; i < len(status.Transitions); i++ {
		transition := status.Transitions[i]
		if !isValidTransition(transition.From, transition.To) {
			t.Errorf("Invalid state transition: %s -> %s", transition.From, transition.To)
		}
	}

	// Verify expected state sequence for success path
	expectedStates := []smoketest.State{
		smoketest.StateInitializing,
		smoketest.StateValidatingArtifact,
		smoketest.StateResolvingCommand,
		smoketest.StateExecuting,
		smoketest.StateParsingOutput,
		smoketest.StateTelemetryUpload,
		smoketest.StatePassed,
	}

	if len(status.Transitions) != len(expectedStates) {
		t.Errorf("Expected %d transitions, got %d", len(expectedStates), len(status.Transitions))
		for i, tr := range status.Transitions {
			t.Logf("  Transition %d: %s -> %s", i, tr.From, tr.To)
		}
	} else {
		for i, expected := range expectedStates {
			if status.Transitions[i].To != expected {
				t.Errorf("Transition %d: expected To=%s, got To=%s", i, expected, status.Transitions[i].To)
			}
		}
	}

	// Verify final state
	if status.CurrentState != smoketest.StatePassed {
		t.Errorf("Expected final state %s, got %s", smoketest.StatePassed, status.CurrentState)
	}
}

func TestService_StateTransitions_ArtifactFailure(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	fs := mocks.NewMockFileSystem()
	// Don't add artifact file - it won't exist

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/nonexistent/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")

	// Verify all transitions are valid
	for _, transition := range status.Transitions {
		if !isValidTransition(transition.From, transition.To) {
			t.Errorf("Invalid state transition: %s -> %s", transition.From, transition.To)
		}
	}

	// Verify failure state sequence
	expectedStates := []smoketest.State{
		smoketest.StateInitializing,
		smoketest.StateValidatingArtifact,
		smoketest.StateFailed,
	}

	if len(status.Transitions) != len(expectedStates) {
		t.Errorf("Expected %d transitions, got %d", len(expectedStates), len(status.Transitions))
	} else {
		for i, expected := range expectedStates {
			if status.Transitions[i].To != expected {
				t.Errorf("Transition %d: expected To=%s, got To=%s", i, expected, status.Transitions[i].To)
			}
		}
	}

	// Verify final state
	if status.CurrentState != smoketest.StateFailed {
		t.Errorf("Expected final state %s, got %s", smoketest.StateFailed, status.CurrentState)
	}
}

func TestService_StateTransitions_ValidationFailure(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
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
	executor.ExecuteResult.Output = "App started but no success marker"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: false}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")

	// Verify all transitions are valid
	for _, transition := range status.Transitions {
		if !isValidTransition(transition.From, transition.To) {
			t.Errorf("Invalid state transition: %s -> %s", transition.From, transition.To)
		}
	}

	// Verify final state is failed
	if status.CurrentState != smoketest.StateFailed {
		t.Errorf("Expected final state %s, got %s", smoketest.StateFailed, status.CurrentState)
	}
}

func TestService_StateTransitions_NoImpossibleJumps(t *testing.T) {
	// Test that no impossible jumps exist in the state machine definition
	allStates := []smoketest.State{
		smoketest.StateInitializing,
		smoketest.StateValidatingArtifact,
		smoketest.StateResolvingCommand,
		smoketest.StateExecuting,
		smoketest.StateParsingOutput,
		smoketest.StateTelemetryUpload,
		smoketest.StateTelemetryFallback,
		smoketest.StatePassed,
		smoketest.StateFailed,
	}

	// Verify each state has a defined set of valid transitions (including empty for terminal states)
	for _, state := range allStates {
		_, ok := ValidStateTransitions[state]
		if !ok {
			t.Errorf("State %s has no defined transitions", state)
		}
	}

	// Verify terminal states have no outgoing transitions
	terminalStates := []smoketest.State{smoketest.StatePassed, smoketest.StateFailed}
	for _, terminal := range terminalStates {
		transitions := ValidStateTransitions[terminal]
		if len(transitions) > 0 {
			t.Errorf("Terminal state %s should have no outgoing transitions, has %v", terminal, transitions)
		}
	}

	// Verify non-terminal states have at least one outgoing transition
	for state, transitions := range ValidStateTransitions {
		if state == "" || state == smoketest.StatePassed || state == smoketest.StateFailed {
			continue
		}
		if len(transitions) == 0 {
			t.Errorf("Non-terminal state %s has no outgoing transitions", state)
		}
	}
}

func TestService_PerformSmokeTest_PanicRecovery(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
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

	// Create executor that panics
	executor := mocks.NewMockProcessExecutor()
	executor.ExecuteWithResultFunc = func(ctx context.Context, workDir, command string, args, env []string, timeout time.Duration) (*smoketest.ExecutionResult, error) {
		panic("simulated executor panic")
	}

	logger := mocks.NewMockLogger()

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.logger = logger
	})

	// This should NOT panic - the service should recover
	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")

	// Verify the test completed without propagating panic
	if status.Status != "failed" {
		t.Errorf("Expected status 'failed' after panic, got %q", status.Status)
	}

	// Verify error contains panic information
	if status.Error == "" {
		t.Error("Expected Error to be set after panic")
	}
	if !contains(status.Error, "panic") {
		t.Errorf("Expected Error to contain 'panic', got %q", status.Error)
	}

	// Verify ErrorKind is set
	if status.ErrorKind == nil {
		t.Error("Expected ErrorKind to be set after panic")
	} else if *status.ErrorKind != smoketest.ErrKindExecution {
		t.Errorf("Expected ErrorKind to be ErrKindExecution, got %v", *status.ErrorKind)
	}

	// Verify final state is failed
	if status.CurrentState != smoketest.StateFailed {
		t.Errorf("Expected final state %s, got %s", smoketest.StateFailed, status.CurrentState)
	}

	// Verify logger recorded the panic
	foundPanicLog := false
	for _, call := range logger.ErrorCalls {
		if call.Msg == "smoke_test_panic" {
			foundPanicLog = true
			break
		}
	}
	if !foundPanicLog {
		t.Error("Expected logger to record smoke_test_panic error")
	}
}

func TestService_PerformSmokeTest_PanicRecovery_InOutputParser(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
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
	executor.ExecuteResult.Output = "some output"

	// Create output parser that panics
	outputParser := mocks.NewMockOutputParser()
	outputParser.ParseFunc = func(output string) smoketest.OutputResult {
		panic("output parser panic")
	}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})

	// This should NOT panic - the service should recover
	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")

	// Verify recovery occurred
	if status.Status != "failed" {
		t.Errorf("Expected status 'failed' after panic, got %q", status.Status)
	}
	if !contains(status.Error, "panic") {
		t.Errorf("Expected Error to contain 'panic', got %q", status.Error)
	}
}

// TestService_TimeoutErrorMessage_LateLifecycleStage tests that timeout errors
// with late lifecycle stages get descriptive error messages.
func TestService_TimeoutErrorMessage_LateLifecycleStage(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
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
	// Output shows app reached waiting_for_token stage before timeout
	executor.ExecuteResult.Output = `SMOKE_TEST_INIT=started session_id=test-session
SMOKE_TEST_STAGE=bundle_resolving
SMOKE_TEST_STAGE=runtime_starting
SMOKE_TEST_STAGE=waiting_for_token
runtime ready — IPC listening on 127.0.0.1:41605`
	executor.ExecuteResult.Err = errors.New("context deadline exceeded") // Timeout error

	outputParser := mocks.NewMockOutputParser()
	outputParser.LifecycleStateResult = "waiting_for_token"
	outputParser.SessionIDResult = "test-session"
	outputParser.Result = smoketest.OutputResult{Passed: false}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")

	// Verify error message reflects that app started successfully but timed out
	if status.Error == "" {
		t.Fatal("Expected Error to be set")
	}
	if !contains(status.Error, "started successfully") && !contains(status.Error, "timed out") {
		t.Errorf("Expected error to indicate successful start + timeout, got: %s", status.Error)
	}

	// Verify lifecycle state is in context
	if status.ErrorContext["last_lifecycle_state"] != "waiting_for_token" {
		t.Errorf("Expected lifecycle state in context, got: %v", status.ErrorContext)
	}

	// Verify error kind is timeout (not generic execution)
	if status.ErrorKind == nil {
		t.Fatal("Expected ErrorKind to be set")
	}
	if *status.ErrorKind != smoketest.ErrKindTimeout {
		t.Errorf("Expected ErrKindTimeout, got %v", *status.ErrorKind)
	}
}

// TestService_TimeoutErrorMessage_EarlyLifecycleStage tests that timeout errors
// with early/no lifecycle stages get appropriate error messages.
func TestService_TimeoutErrorMessage_EarlyLifecycleStage(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
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
	// Output shows app didn't produce any markers
	executor.ExecuteResult.Output = ""
	executor.ExecuteResult.Err = errors.New("process timed out")

	outputParser := mocks.NewMockOutputParser()
	outputParser.LifecycleStateResult = "" // No lifecycle state reached
	outputParser.Result = smoketest.OutputResult{Passed: false}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")

	// Error should indicate timeout before producing markers
	if status.Error == "" {
		t.Fatal("Expected Error to be set")
	}
	if !contains(status.Error, "timed out") {
		t.Errorf("Expected error to mention timeout, got: %s", status.Error)
	}
	// Should NOT say "started successfully" since no late lifecycle state reached
	if contains(status.Error, "started successfully") {
		t.Errorf("Error should NOT say 'started successfully' for early timeout: %s", status.Error)
	}
}

// TestService_TimeoutErrorMessage_RuntimeStartingStage tests that runtime_starting
// is considered a late lifecycle stage.
func TestService_TimeoutErrorMessage_RuntimeStartingStage(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
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
	executor.ExecuteResult.Output = `SMOKE_TEST_INIT=started
SMOKE_TEST_STAGE=bundle_resolving
SMOKE_TEST_STAGE=runtime_starting`
	executor.ExecuteResult.Err = errors.New("timeout waiting for process")

	outputParser := mocks.NewMockOutputParser()
	outputParser.LifecycleStateResult = "runtime_starting"
	outputParser.Result = smoketest.OutputResult{Passed: false}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")

	// Error should reflect successful initialization but timeout during runtime startup
	if status.Error == "" {
		t.Fatal("Expected Error to be set")
	}
	if !contains(status.Error, "initialized") && !contains(status.Error, "timed out") {
		t.Errorf("Expected error to indicate initialization + timeout, got: %s", status.Error)
	}

	// Error kind should be timeout
	if status.ErrorKind == nil || *status.ErrorKind != smoketest.ErrKindTimeout {
		t.Errorf("Expected ErrKindTimeout for late lifecycle timeout")
	}
}

// TestService_NonTimeoutError_KeepsGenericMessage tests that non-timeout errors
// keep the generic "smoke test process failed" message.
func TestService_NonTimeoutError_KeepsGenericMessage(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
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
	executor.ExecuteWithResultResult.Result = &smoketest.ExecutionResult{
		Stdout: `SMOKE_TEST_INIT=started
SMOKE_TEST_STAGE=bundle_resolving
SMOKE_TEST_STAGE=runtime_starting
SMOKE_TEST_STAGE=waiting_for_token`,
		Stderr: "",
		Combined: `SMOKE_TEST_INIT=started
SMOKE_TEST_STAGE=bundle_resolving
SMOKE_TEST_STAGE=runtime_starting
SMOKE_TEST_STAGE=waiting_for_token`,
		ExitCode: 139, // SIGSEGV
	}
	executor.ExecuteWithResultResult.Err = errors.New("segmentation fault") // Non-timeout error

	outputParser := mocks.NewMockOutputParser()
	outputParser.LifecycleStateResult = "waiting_for_token"
	outputParser.Result = smoketest.OutputResult{Passed: false}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")

	// Non-timeout error should have generic message
	if status.Error == "" {
		t.Fatal("Expected Error to be set")
	}
	if !contains(status.Error, "process failed") {
		t.Errorf("Non-timeout error should use generic message, got: %s", status.Error)
	}

	// Error kind should be execution (not timeout)
	if status.ErrorKind == nil || *status.ErrorKind != smoketest.ErrKindExecution {
		t.Errorf("Expected ErrKindExecution for non-timeout error")
	}

	// Lifecycle state should still be in context
	if status.ErrorContext["last_lifecycle_state"] != "waiting_for_token" {
		t.Errorf("Lifecycle state should be in context even for non-timeout errors")
	}
}
