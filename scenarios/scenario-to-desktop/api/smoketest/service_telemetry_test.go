package smoketest_test

import (
	"context"
	"errors"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
	"testing"
)

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
