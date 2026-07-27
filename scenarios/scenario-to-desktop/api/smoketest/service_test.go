package smoketest_test

import (
	"context"
	"errors"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
	"testing"
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
