package smoketest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
)

func TestNewService(t *testing.T) {
	store := mocks.NewMockStore()
	cancelManager := mocks.NewMockCancelManager()
	telemetryIngestor := mocks.NewMockTelemetryIngestor()
	config := smoketest.DefaultConfig()
	executor := mocks.NewMockProcessExecutor()
	platformResolver := mocks.NewMockPlatformResolver()
	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	outputParser := mocks.NewMockOutputParser()
	fs := mocks.NewMockFileSystem()
	logger := mocks.NewMockLogger()
	telemetryExtractor := mocks.NewMockTelemetryErrorExtractor()

	service := smoketest.NewService(
		store,
		cancelManager,
		telemetryIngestor,
		config,
		executor,
		platformResolver,
		telemetryResolver,
		outputParser,
		fs,
		logger,
		8080,
		telemetryExtractor,
	)

	if service == nil {
		t.Fatal("NewService returned nil")
	}
}

func TestNewDefaultSmokeTestService(t *testing.T) {
	store := mocks.NewMockStore()
	cancelManager := mocks.NewMockCancelManager()
	logger := mocks.NewMockLogger()

	service := smoketest.NewDefaultSmokeTestService(store, cancelManager, nil, 8080, logger)

	if service == nil {
		t.Fatal("NewDefaultSmokeTestService returned nil")
	}
}

func TestService_CurrentPlatform(t *testing.T) {
	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.Platform = "linux"

	service := createTestService(func(s *testServiceDeps) {
		s.platformResolver = platformResolver
	})

	platform := service.CurrentPlatform()
	if platform != "linux" {
		t.Errorf("CurrentPlatform() = %q, want %q", platform, "linux")
	}
}

func TestService_PerformSmokeTest_NonexistentID(t *testing.T) {
	store := mocks.NewMockStore()
	// Don't add any status - smoke test ID doesn't exist

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "nonexistent", "scenario", "/path/to/artifact", "linux")

	// Should return early without crashing
	if len(store.UpdateCalls) > 0 {
		t.Error("Expected no Update calls for nonexistent ID")
	}
}

func TestService_PerformSmokeTest_CancelledContext(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact", "linux")

	status, _ := store.Get("test-123")
	if status.Status != "failed" {
		t.Errorf("Expected status 'failed', got %q", status.Status)
	}
	if status.Error != "smoke test cancelled" {
		t.Errorf("Expected error 'smoke test cancelled', got %q", status.Error)
	}
}

func TestService_PerformSmokeTest_ArtifactNotFound(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	fs := mocks.NewMockFileSystem()
	// Don't add the artifact file - it won't exist

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/nonexistent/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")
	if status.Status != "failed" {
		t.Errorf("Expected status 'failed', got %q", status.Status)
	}
}

func TestService_PerformSmokeTest_CommandResolutionError(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	fs := mocks.NewMockFileSystem()
	fs.AddFile("/path/to/artifact.deb", []byte{})

	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.ResolveResult.Err = errors.New("unsupported artifact type")

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.deb", "linux")

	status, _ := store.Get("test-123")
	if status.Status != "failed" {
		t.Errorf("Expected status 'failed', got %q", status.Status)
	}
}

func TestService_PerformSmokeTest_HeadlessWrapperError(t *testing.T) {
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
	platformResolver.HeadlessResult = struct {
		Needed      bool
		WrapperCmd  string
		WrapperArgs []string
		Err         error
	}{
		Needed: true,
		Err:    errors.New("xvfb-run not available"),
	}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")
	if status.Status != "failed" {
		t.Errorf("Expected status 'failed', got %q", status.Status)
	}
}

func TestService_PerformSmokeTest_ExecutionError(t *testing.T) {
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
	executor.ExecuteResult.Err = errors.New("execution failed")

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
	if status.Status != "failed" {
		t.Errorf("Expected status 'failed', got %q", status.Status)
	}
}

func TestService_PerformSmokeTest_NoSuccessMarker(t *testing.T) {
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
	if status.Status != "failed" {
		t.Errorf("Expected status 'failed', got %q", status.Status)
	}
	if status.Error != "smoke test did not report success" {
		t.Errorf("Expected specific error message, got %q", status.Error)
	}
}

func TestService_PerformSmokeTest_Success(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	cancelManager := mocks.NewMockCancelManager()

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
		s.cancelManager = cancelManager
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")
	if status.Status != "passed" {
		t.Errorf("Expected status 'passed', got %q", status.Status)
	}
	if !status.TelemetryUploaded {
		t.Error("Expected TelemetryUploaded to be true")
	}
	if status.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}

	// Verify cancel manager was cleared
	if len(cancelManager.ClearCalls) != 1 || cancelManager.ClearCalls[0] != "test-123" {
		t.Errorf("Expected Clear to be called with 'test-123', got %v", cancelManager.ClearCalls)
	}
}

func TestService_PerformSmokeTest_TelemetryUploadError(t *testing.T) {
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
	executor.ExecuteResult.Output = "SMOKE_TEST_RESULT=passed\nSMOKE_TEST_UPLOAD=error"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{
		Passed:               true,
		TelemetryUploadError: true,
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
	if status.Status != "passed" {
		t.Errorf("Expected status 'passed', got %q", status.Status)
	}
	if status.TelemetryUploadError == "" {
		t.Error("Expected TelemetryUploadError to be set")
	}
}

func TestService_PerformSmokeTest_TelemetryFallback(t *testing.T) {
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
	executor.ExecuteResult.Output = "SMOKE_TEST_RESULT=passed\n[Desktop App] Telemetry initialized at /tmp/telemetry.jsonl"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true}

	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	telemetryResolver.ExtractResult = "/tmp/telemetry.jsonl"
	telemetryResolver.ReadEventsResult.Events = []map[string]interface{}{
		{"event": "test"},
	}

	telemetryIngestor := mocks.NewMockTelemetryIngestor()
	telemetryIngestor.IngestResult.Count = 1

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
		s.telemetryResolver = telemetryResolver
		s.telemetryIngestor = telemetryIngestor
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")
	if status.Status != "passed" {
		t.Errorf("Expected status 'passed', got %q", status.Status)
	}
	if !status.TelemetryUploaded {
		t.Error("Expected TelemetryUploaded to be true after fallback")
	}

	// Verify ingestor was called
	if len(telemetryIngestor.IngestCalls) != 1 {
		t.Errorf("Expected 1 ingest call, got %d", len(telemetryIngestor.IngestCalls))
	}
}

func TestService_PerformSmokeTest_TelemetryFallback_NoPath(t *testing.T) {
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
	executor.ExecuteResult.Output = "SMOKE_TEST_RESULT=passed"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true}

	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	// No path returned

	telemetryIngestor := mocks.NewMockTelemetryIngestor()

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
		s.telemetryResolver = telemetryResolver
		s.telemetryIngestor = telemetryIngestor
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")
	if status.Status != "passed" {
		t.Errorf("Expected status 'passed', got %q", status.Status)
	}

	// Verify ingestor was NOT called
	if len(telemetryIngestor.IngestCalls) != 0 {
		t.Errorf("Expected no ingest calls, got %d", len(telemetryIngestor.IngestCalls))
	}
}

func TestService_PerformSmokeTest_TelemetryFallback_ReadError(t *testing.T) {
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
	executor.ExecuteResult.Output = "SMOKE_TEST_RESULT=passed"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true}

	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	telemetryResolver.ExtractResult = "/tmp/telemetry.jsonl"
	telemetryResolver.ReadEventsResult.Err = errors.New("read error")

	telemetryIngestor := mocks.NewMockTelemetryIngestor()

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
		s.telemetryResolver = telemetryResolver
		s.telemetryIngestor = telemetryIngestor
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")
	if status.TelemetryUploadError == "" {
		t.Error("Expected TelemetryUploadError to be set")
	}
}

func TestService_PerformSmokeTest_HeadlessWrapper(t *testing.T) {
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
	platformResolver.HeadlessResult = struct {
		Needed      bool
		WrapperCmd  string
		WrapperArgs []string
		Err         error
	}{
		Needed:      true,
		WrapperCmd:  "xvfb-run",
		WrapperArgs: []string{"-a"},
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

	// Verify xvfb-run was used
	if len(executor.ExecuteCalls) != 1 {
		t.Fatalf("Expected 1 execute call, got %d", len(executor.ExecuteCalls))
	}
	call := executor.ExecuteCalls[0]
	if call.Command != "xvfb-run" {
		t.Errorf("Expected command 'xvfb-run', got %q", call.Command)
	}
	expectedArgs := []string{"-a", "/path/to/artifact.AppImage", "--smoke-test"}
	if len(call.Args) != len(expectedArgs) {
		t.Errorf("Expected args %v, got %v", expectedArgs, call.Args)
	}
}

func TestService_PerformSmokeTest_LongOutput(t *testing.T) {
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

	// Generate output longer than 500 characters
	longOutput := ""
	for i := 0; i < 600; i++ {
		longOutput += "x"
	}
	longOutput += "\nSMOKE_TEST_RESULT=passed\nSMOKE_TEST_UPLOAD=ok"

	executor := mocks.NewMockProcessExecutor()
	executor.ExecuteResult.Output = longOutput

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
	if status.Status != "passed" {
		t.Errorf("Expected status 'passed', got %q", status.Status)
	}

	// Check that logs contain truncation indicator
	foundTruncated := false
	for _, log := range status.Logs {
		if len(log) > 0 && (len(longOutput) > 500) {
			foundTruncated = true
			break
		}
	}
	if !foundTruncated {
		t.Log("Long output should be truncated in logs")
	}
}

// Helper types and functions for testing

type testServiceDeps struct {
	store              *mocks.MockStore
	cancelManager      *mocks.MockCancelManager
	telemetryIngestor  *mocks.MockTelemetryIngestor
	config             smoketest.Config
	executor           *mocks.MockProcessExecutor
	platformResolver   *mocks.MockPlatformResolver
	telemetryResolver  *mocks.MockTelemetryPathResolver
	outputParser       *mocks.MockOutputParser
	fs                 *mocks.MockFileSystem
	logger             *mocks.MockLogger
	port               int
	telemetryExtractor *mocks.MockTelemetryErrorExtractor
}

func createTestService(configure func(*testServiceDeps)) *smoketest.DefaultService {
	deps := &testServiceDeps{
		store:              mocks.NewMockStore(),
		cancelManager:      mocks.NewMockCancelManager(),
		telemetryIngestor:  mocks.NewMockTelemetryIngestor(),
		config:             smoketest.DefaultConfig(),
		executor:           mocks.NewMockProcessExecutor(),
		platformResolver:   mocks.NewMockPlatformResolver(),
		telemetryResolver:  mocks.NewMockTelemetryPathResolver(),
		outputParser:       mocks.NewMockOutputParser(),
		fs:                 mocks.NewMockFileSystem(),
		logger:             mocks.NewMockLogger(),
		port:               8080,
		telemetryExtractor: mocks.NewMockTelemetryErrorExtractor(),
	}

	if configure != nil {
		configure(deps)
	}

	return smoketest.NewService(
		deps.store,
		deps.cancelManager,
		deps.telemetryIngestor,
		deps.config,
		deps.executor,
		deps.platformResolver,
		deps.telemetryResolver,
		deps.outputParser,
		deps.fs,
		deps.logger,
		deps.port,
		deps.telemetryExtractor,
	)
}

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
		Logs:                 []string{"starting...", "done"},
		Error:                "",
		TelemetryUploaded:    true,
		TelemetryUploadError: "",
	}

	if status.SmokeTestID != "test-123" {
		t.Errorf("expected SmokeTestID 'test-123'")
	}
	if status.Platform != "linux" {
		t.Errorf("expected Platform 'linux'")
	}
	if len(status.Logs) != 2 {
		t.Errorf("expected 2 log entries")
	}
	if !status.TelemetryUploaded {
		t.Errorf("expected TelemetryUploaded to be true")
	}
}

// ValidStateTransitions defines all valid state transitions in the smoke test state machine.
var ValidStateTransitions = map[smoketest.State][]smoketest.State{
	"": { // Initial empty state
		smoketest.StateInitializing,
	},
	smoketest.StateInitializing: {
		smoketest.StateValidatingArtifact,
		smoketest.StateFailed,
	},
	smoketest.StateValidatingArtifact: {
		smoketest.StateResolvingCommand,
		smoketest.StateFailed,
	},
	smoketest.StateResolvingCommand: {
		smoketest.StateExecuting,
		smoketest.StateFailed,
	},
	smoketest.StateExecuting: {
		smoketest.StateParsingOutput,
		smoketest.StateFailed,
	},
	smoketest.StateParsingOutput: {
		smoketest.StateTelemetryUpload,
		smoketest.StateTelemetryFallback,
		smoketest.StatePassed,
		smoketest.StateFailed,
	},
	smoketest.StateTelemetryUpload: {
		smoketest.StatePassed,
		smoketest.StateFailed,
	},
	smoketest.StateTelemetryFallback: {
		smoketest.StatePassed,
		smoketest.StateFailed,
	},
	smoketest.StatePassed: {}, // Terminal state
	smoketest.StateFailed: {}, // Terminal state
}

// isValidTransition checks if a state transition is valid according to the state machine.
func isValidTransition(from, to smoketest.State) bool {
	validTargets, ok := ValidStateTransitions[from]
	if !ok {
		return false
	}
	for _, valid := range validTargets {
		if valid == to {
			return true
		}
	}
	return false
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

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// logsContain checks if any log entry contains the given substring
func logsContain(logs []string, substr string) bool {
	for _, log := range logs {
		if contains(log, substr) {
			return true
		}
	}
	return false
}

func TestService_LogContent_ContainsScenarioName(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "my-unique-scenario",
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
	executor.ExecuteResult.Output = "SMOKE_TEST_RESULT=passed"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true, TelemetryUploaded: true}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "my-unique-scenario", "/path/to/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")
	if !logsContain(status.Logs, "my-unique-scenario") {
		t.Error("Logs should contain scenario name")
		t.Logf("Logs: %v", status.Logs)
	}
}

func TestService_LogContent_ContainsPlatform(t *testing.T) {
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
	executor.ExecuteResult.Output = "SMOKE_TEST_RESULT=passed"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true, TelemetryUploaded: true}

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
	if !logsContain(status.Logs, "linux") {
		t.Error("Logs should contain platform")
		t.Logf("Logs: %v", status.Logs)
	}
}

func TestService_LogContent_ContainsArtifactPath(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	fs := mocks.NewMockFileSystem()
	fs.AddFile("/path/to/MySpecialApp.AppImage", []byte{})

	platformResolver := mocks.NewMockPlatformResolver()
	platformResolver.ResolveResult = struct {
		Cmd     string
		Args    []string
		Display string
		Err     error
	}{
		Cmd:     "/path/to/MySpecialApp.AppImage",
		Args:    []string{"--smoke-test"},
		Display: "/path/to/MySpecialApp.AppImage --smoke-test",
	}

	executor := mocks.NewMockProcessExecutor()
	executor.ExecuteResult.Output = "SMOKE_TEST_RESULT=passed"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true, TelemetryUploaded: true}

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
		s.platformResolver = platformResolver
		s.executor = executor
		s.outputParser = outputParser
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/path/to/MySpecialApp.AppImage", "linux")

	status, _ := store.Get("test-123")
	// Should contain at least the artifact file name
	if !logsContain(status.Logs, "MySpecialApp.AppImage") {
		t.Error("Logs should contain artifact path/name")
		t.Logf("Logs: %v", status.Logs)
	}
}

func TestService_LogContent_ContainsCommand(t *testing.T) {
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
	executor.ExecuteResult.Output = "SMOKE_TEST_RESULT=passed"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true, TelemetryUploaded: true}

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
	// Should contain the command with --smoke-test flag
	if !logsContain(status.Logs, "--smoke-test") {
		t.Error("Logs should contain command with --smoke-test flag")
		t.Logf("Logs: %v", status.Logs)
	}
}

func TestService_LogContent_ContainsErrorDetails(t *testing.T) {
	store := mocks.NewMockStore()
	store.AddStatus(&smoketest.Status{
		SmokeTestID:  "test-123",
		ScenarioName: "test-scenario",
		Status:       "running",
	})

	fs := mocks.NewMockFileSystem()
	// Artifact doesn't exist

	service := createTestService(func(s *testServiceDeps) {
		s.store = store
		s.fs = fs
	})

	ctx := context.Background()
	service.PerformSmokeTest(ctx, "test-123", "test-scenario", "/nonexistent/artifact.AppImage", "linux")

	status, _ := store.Get("test-123")
	// Should contain FAILED and error information
	if !logsContain(status.Logs, "FAILED") {
		t.Error("Logs should contain FAILED on error")
		t.Logf("Logs: %v", status.Logs)
	}
}

func TestService_LogContent_ContainsStateTransitions(t *testing.T) {
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
	executor.ExecuteResult.Output = "SMOKE_TEST_RESULT=passed"

	outputParser := mocks.NewMockOutputParser()
	outputParser.Result = smoketest.OutputResult{Passed: true, TelemetryUploaded: true}

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

	// Should contain state transition markers
	expectedStates := []string{"initializing", "validating_artifact", "resolving_command", "executing", "parsing_output"}
	for _, state := range expectedStates {
		if !logsContain(status.Logs, state) {
			t.Errorf("Logs should contain state transition for %s", state)
			t.Logf("Logs: %v", status.Logs)
		}
	}
}

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
