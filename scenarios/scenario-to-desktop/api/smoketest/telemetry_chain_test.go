package smoketest_test

import (
	"context"
	"errors"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
	"testing"
)

func TestTelemetryChainExecutor_DirectUploadSuccess(t *testing.T) {
	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	telemetryIngestor := mocks.NewMockTelemetryIngestor()
	config := smoketest.DefaultConfig()
	logger := mocks.NewMockLogger()

	executor := smoketest.NewTelemetryChainExecutor(telemetryResolver, telemetryIngestor, config, logger)

	result := executor.Execute(context.Background(), smoketest.TelemetryChainParams{
		SmokeTestID:         "test-123",
		ScenarioName:        "test-scenario",
		Platform:            "linux",
		DirectUploadSuccess: true,
	})

	if result.Source != smoketest.TelemetrySourceUpload {
		t.Errorf("Expected source %q, got %q", smoketest.TelemetrySourceUpload, result.Source)
	}
	if len(result.AttemptedPaths) != 1 {
		t.Errorf("Expected 1 attempted path, got %d", len(result.AttemptedPaths))
	}
	if !result.AttemptedPaths[0].Success {
		t.Error("Expected first attempt to be successful")
	}
}

func TestTelemetryChainExecutor_OutputExtractionSuccess(t *testing.T) {
	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	telemetryResolver.ExtractResult = "/tmp/telemetry.jsonl"
	telemetryResolver.ReadEventsResult.Events = []map[string]interface{}{
		{"event": "test1"},
		{"event": "test2"},
	}

	telemetryIngestor := mocks.NewMockTelemetryIngestor()
	telemetryIngestor.IngestResult.Count = 2

	config := smoketest.DefaultConfig()
	logger := mocks.NewMockLogger()

	executor := smoketest.NewTelemetryChainExecutor(telemetryResolver, telemetryIngestor, config, logger)

	result := executor.Execute(context.Background(), smoketest.TelemetryChainParams{
		SmokeTestID:         "test-123",
		ScenarioName:        "test-scenario",
		Platform:            "linux",
		Output:              "[Desktop App] Telemetry initialized at /tmp/telemetry.jsonl",
		DirectUploadSuccess: false,
	})

	if result.Source != smoketest.TelemetrySourceOutputExtraction {
		t.Errorf("Expected source %q, got %q", smoketest.TelemetrySourceOutputExtraction, result.Source)
	}
	if result.Path != "/tmp/telemetry.jsonl" {
		t.Errorf("Expected path %q, got %q", "/tmp/telemetry.jsonl", result.Path)
	}
	if result.EventsFound != 2 {
		t.Errorf("Expected 2 events found, got %d", result.EventsFound)
	}
	if result.EventsIngested != 2 {
		t.Errorf("Expected 2 events ingested, got %d", result.EventsIngested)
	}
}

func TestTelemetryChainExecutor_ArtifactResolutionFallback(t *testing.T) {
	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	// No path extracted from output
	telemetryResolver.ExtractResult = ""
	// But artifact resolution works
	telemetryResolver.ResolveResult = "/home/user/.config/test-app/deployment-telemetry.jsonl"
	telemetryResolver.ReadEventsResult.Events = []map[string]interface{}{
		{"event": "test"},
	}

	telemetryIngestor := mocks.NewMockTelemetryIngestor()
	telemetryIngestor.IngestResult.Count = 1

	config := smoketest.DefaultConfig()
	logger := mocks.NewMockLogger()

	executor := smoketest.NewTelemetryChainExecutor(telemetryResolver, telemetryIngestor, config, logger)

	result := executor.Execute(context.Background(), smoketest.TelemetryChainParams{
		SmokeTestID:         "test-123",
		ScenarioName:        "test-scenario",
		Platform:            "linux",
		ArtifactPath:        "/path/to/artifact.AppImage",
		DirectUploadSuccess: false,
	})

	if result.Source != smoketest.TelemetrySourceArtifactResolution {
		t.Errorf("Expected source %q, got %q", smoketest.TelemetrySourceArtifactResolution, result.Source)
	}
	// Should have 2 attempts: output extraction (failed) and artifact resolution (succeeded)
	if len(result.AttemptedPaths) != 2 {
		t.Errorf("Expected 2 attempted paths, got %d", len(result.AttemptedPaths))
	}
}

func TestTelemetryChainExecutor_AllMethodsFail(t *testing.T) {
	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	telemetryResolver.ExtractResult = ""
	telemetryResolver.ResolveResult = ""

	telemetryIngestor := mocks.NewMockTelemetryIngestor()
	config := smoketest.DefaultConfig()
	logger := mocks.NewMockLogger()

	executor := smoketest.NewTelemetryChainExecutor(telemetryResolver, telemetryIngestor, config, logger)

	result := executor.Execute(context.Background(), smoketest.TelemetryChainParams{
		SmokeTestID:         "test-123",
		ScenarioName:        "test-scenario",
		Platform:            "linux",
		DirectUploadSuccess: false,
	})

	if result.Source != smoketest.TelemetrySourceNone {
		t.Errorf("Expected source %q, got %q", smoketest.TelemetrySourceNone, result.Source)
	}
	if result.Error == "" {
		t.Error("Expected error to be set")
	}
	// Should have 2 failed attempts
	if len(result.AttemptedPaths) != 2 {
		t.Errorf("Expected 2 attempted paths, got %d", len(result.AttemptedPaths))
	}
}

func TestTelemetryChainExecutor_NoIngestor(t *testing.T) {
	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	config := smoketest.DefaultConfig()
	logger := mocks.NewMockLogger()

	// No ingestor configured
	executor := smoketest.NewTelemetryChainExecutor(telemetryResolver, nil, config, logger)

	result := executor.Execute(context.Background(), smoketest.TelemetryChainParams{
		SmokeTestID:         "test-123",
		ScenarioName:        "test-scenario",
		Platform:            "linux",
		DirectUploadSuccess: false,
	})

	if result.Source != smoketest.TelemetrySourceNone {
		t.Errorf("Expected source %q, got %q", smoketest.TelemetrySourceNone, result.Source)
	}
	if result.Error == "" {
		t.Error("Expected error about missing ingestor")
	}
}

func TestTelemetryChainExecutor_ReadEventsError(t *testing.T) {
	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	telemetryResolver.ExtractResult = "/tmp/telemetry.jsonl"
	telemetryResolver.ReadEventsResult.Err = errors.New("file not found")

	// Artifact resolution also fails
	telemetryResolver.ResolveResult = ""

	telemetryIngestor := mocks.NewMockTelemetryIngestor()
	config := smoketest.DefaultConfig()
	logger := mocks.NewMockLogger()

	executor := smoketest.NewTelemetryChainExecutor(telemetryResolver, telemetryIngestor, config, logger)

	result := executor.Execute(context.Background(), smoketest.TelemetryChainParams{
		SmokeTestID:         "test-123",
		ScenarioName:        "test-scenario",
		Platform:            "linux",
		Output:              "[Desktop App] Telemetry initialized at /tmp/telemetry.jsonl",
		DirectUploadSuccess: false,
	})

	if result.Source != smoketest.TelemetrySourceNone {
		t.Errorf("Expected source %q, got %q", smoketest.TelemetrySourceNone, result.Source)
	}
	// First attempt should have the error
	if len(result.AttemptedPaths) < 1 {
		t.Fatal("Expected at least 1 attempted path")
	}
	if result.AttemptedPaths[0].Error == "" {
		t.Error("Expected first attempt to have error")
	}
}

func TestTelemetryChainExecutor_IngestError(t *testing.T) {
	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	telemetryResolver.ExtractResult = "/tmp/telemetry.jsonl"
	telemetryResolver.ReadEventsResult.Events = []map[string]interface{}{
		{"event": "test"},
	}
	// Artifact resolution also provides a path
	telemetryResolver.ResolveResult = "/home/user/.config/app/telemetry.jsonl"

	telemetryIngestor := mocks.NewMockTelemetryIngestor()
	telemetryIngestor.IngestResult.Err = errors.New("ingest failed")

	config := smoketest.DefaultConfig()
	logger := mocks.NewMockLogger()

	executor := smoketest.NewTelemetryChainExecutor(telemetryResolver, telemetryIngestor, config, logger)

	result := executor.Execute(context.Background(), smoketest.TelemetryChainParams{
		SmokeTestID:         "test-123",
		ScenarioName:        "test-scenario",
		Platform:            "linux",
		ArtifactPath:        "/path/to/artifact.AppImage",
		Output:              "[Desktop App] Telemetry initialized at /tmp/telemetry.jsonl",
		DirectUploadSuccess: false,
	})

	// Both attempts should fail due to ingest error
	if result.Source != smoketest.TelemetrySourceNone {
		t.Errorf("Expected source %q, got %q", smoketest.TelemetrySourceNone, result.Source)
	}
}

func TestTelemetryChainExecutor_NoEventsFound(t *testing.T) {
	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	telemetryResolver.ExtractResult = "/tmp/telemetry.jsonl"
	telemetryResolver.ReadEventsResult.Events = []map[string]interface{}{} // Empty
	telemetryResolver.ResolveResult = ""

	telemetryIngestor := mocks.NewMockTelemetryIngestor()
	config := smoketest.DefaultConfig()
	logger := mocks.NewMockLogger()

	executor := smoketest.NewTelemetryChainExecutor(telemetryResolver, telemetryIngestor, config, logger)

	result := executor.Execute(context.Background(), smoketest.TelemetryChainParams{
		SmokeTestID:         "test-123",
		ScenarioName:        "test-scenario",
		Platform:            "linux",
		Output:              "[Desktop App] Telemetry initialized at /tmp/telemetry.jsonl",
		DirectUploadSuccess: false,
	})

	if result.Source != smoketest.TelemetrySourceNone {
		t.Errorf("Expected source %q, got %q", smoketest.TelemetrySourceNone, result.Source)
	}
	// First attempt should indicate no events
	if len(result.AttemptedPaths) < 1 {
		t.Fatal("Expected at least 1 attempted path")
	}
	if result.AttemptedPaths[0].Error == "" {
		t.Error("Expected first attempt to have error about no events")
	}
}

func TestTelemetryChainExecutor_AttemptedPathsVisibility(t *testing.T) {
	telemetryResolver := mocks.NewMockTelemetryPathResolver()
	telemetryResolver.ExtractResult = "/tmp/output-path.jsonl"
	telemetryResolver.ResolveResult = "/home/user/.config/app/artifact-path.jsonl"
	// First read fails, second succeeds
	callCount := 0
	telemetryResolver.ReadEventsFunc = func(path string, limit int) ([]map[string]interface{}, error) {
		callCount++
		if callCount == 1 {
			return nil, errors.New("first read failed")
		}
		return []map[string]interface{}{{"event": "test"}}, nil
	}

	telemetryIngestor := mocks.NewMockTelemetryIngestor()
	telemetryIngestor.IngestResult.Count = 1

	config := smoketest.DefaultConfig()
	logger := mocks.NewMockLogger()

	executor := smoketest.NewTelemetryChainExecutor(telemetryResolver, telemetryIngestor, config, logger)

	result := executor.Execute(context.Background(), smoketest.TelemetryChainParams{
		SmokeTestID:         "test-123",
		ScenarioName:        "test-scenario",
		Platform:            "linux",
		ArtifactPath:        "/path/to/artifact.AppImage",
		Output:              "[Desktop App] Telemetry initialized at /tmp/output-path.jsonl",
		DirectUploadSuccess: false,
	})

	// Should succeed via artifact resolution
	if result.Source != smoketest.TelemetrySourceArtifactResolution {
		t.Errorf("Expected source %q, got %q", smoketest.TelemetrySourceArtifactResolution, result.Source)
	}

	// Should have 2 attempts with full visibility
	if len(result.AttemptedPaths) != 2 {
		t.Fatalf("Expected 2 attempted paths, got %d", len(result.AttemptedPaths))
	}

	// First attempt: output extraction failed
	if result.AttemptedPaths[0].Source != smoketest.TelemetrySourceOutputExtraction {
		t.Errorf("First attempt source: expected %q, got %q", smoketest.TelemetrySourceOutputExtraction, result.AttemptedPaths[0].Source)
	}
	if result.AttemptedPaths[0].Success {
		t.Error("First attempt should have failed")
	}
	if result.AttemptedPaths[0].Path != "/tmp/output-path.jsonl" {
		t.Errorf("First attempt path: expected %q, got %q", "/tmp/output-path.jsonl", result.AttemptedPaths[0].Path)
	}

	// Second attempt: artifact resolution succeeded
	if result.AttemptedPaths[1].Source != smoketest.TelemetrySourceArtifactResolution {
		t.Errorf("Second attempt source: expected %q, got %q", smoketest.TelemetrySourceArtifactResolution, result.AttemptedPaths[1].Source)
	}
	if !result.AttemptedPaths[1].Success {
		t.Error("Second attempt should have succeeded")
	}
	if result.AttemptedPaths[1].Path != "/home/user/.config/app/artifact-path.jsonl" {
		t.Errorf("Second attempt path: expected %q, got %q", "/home/user/.config/app/artifact-path.jsonl", result.AttemptedPaths[1].Path)
	}
}
