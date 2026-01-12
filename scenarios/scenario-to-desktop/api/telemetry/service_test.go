package telemetry

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServiceCreation(t *testing.T) {
	service := NewService("/tmp/test-vrooli")
	if service == nil {
		t.Fatalf("expected service to be created")
	}
}

func TestGetFilePath(t *testing.T) {
	service := NewService("/home/test")
	path := service.GetFilePath("my-scenario")
	expected := "/home/test/.vrooli/deployment/telemetry/my-scenario.jsonl"
	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}
}

func TestSummaryMissing(t *testing.T) {
	vrooliRoot := t.TempDir()
	service := NewService(vrooliRoot)

	ctx := context.Background()
	summary, err := service.GetSummary(ctx, "nonexistent-scenario")
	if err != nil {
		t.Fatalf("expected no error for missing telemetry, got %v", err)
	}
	if summary.Exists {
		t.Errorf("expected exists=false for missing telemetry")
	}
}

func TestSummaryWithData(t *testing.T) {
	vrooliRoot := t.TempDir()
	telemetryDir := filepath.Join(vrooliRoot, ".vrooli", "deployment", "telemetry")
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		t.Fatalf("failed to create telemetry dir: %v", err)
	}

	filePath := filepath.Join(telemetryDir, "test-scenario.jsonl")
	content := strings.Join([]string{
		`{"event":"app_start","ingested_at":"2026-01-01T00:00:00Z"}`,
		`{"event":"app_ready","ingested_at":"2026-01-02T00:00:00Z"}`,
		"",
	}, "\n")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write telemetry file: %v", err)
	}

	service := NewService(vrooliRoot)
	ctx := context.Background()
	summary, err := service.GetSummary(ctx, "test-scenario")
	if err != nil {
		t.Fatalf("GetSummary error: %v", err)
	}

	if !summary.Exists {
		t.Errorf("expected exists=true")
	}
	if summary.EventCount != 2 {
		t.Errorf("expected 2 events, got %d", summary.EventCount)
	}
	if summary.LastIngestedAt != "2026-01-02T00:00:00Z" {
		t.Errorf("expected last ingested at '2026-01-02T00:00:00Z', got %q", summary.LastIngestedAt)
	}
}

func TestIngestEvents(t *testing.T) {
	vrooliRoot := t.TempDir()
	service := NewService(vrooliRoot)
	ctx := context.Background()

	t.Run("successful ingest", func(t *testing.T) {
		events := []map[string]interface{}{
			{"event": "test_event_1", "data": "value1"},
			{"event": "test_event_2", "data": "value2"},
		}

		filePath, count, err := service.IngestEvents(ctx, "test-scenario", "external-server", "test-source", events)
		if err != nil {
			t.Fatalf("IngestEvents error: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 events ingested, got %d", count)
		}
		if filePath == "" {
			t.Errorf("expected non-empty file path")
		}

		// Verify file contents
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read telemetry file: %v", err)
		}
		if !strings.Contains(string(content), "test_event_1") {
			t.Errorf("expected content to contain test_event_1")
		}
	})

	t.Run("empty scenario name", func(t *testing.T) {
		_, _, err := service.IngestEvents(ctx, "", "external-server", "source", []map[string]interface{}{{"event": "test"}})
		if err == nil {
			t.Fatalf("expected error for empty scenario name")
		}
	})

	t.Run("empty events", func(t *testing.T) {
		_, _, err := service.IngestEvents(ctx, "test-scenario", "external-server", "source", []map[string]interface{}{})
		if err == nil {
			t.Fatalf("expected error for empty events")
		}
	})

	t.Run("default deployment mode", func(t *testing.T) {
		events := []map[string]interface{}{{"event": "test"}}
		_, _, err := service.IngestEvents(ctx, "test-scenario-default", "", "source", events)
		if err != nil {
			t.Fatalf("IngestEvents error: %v", err)
		}
	})
}

func TestNormalizeEvent(t *testing.T) {
	t.Run("adds required fields", func(t *testing.T) {
		event := map[string]interface{}{"event": "test"}
		result := normalizeEvent(event, "my-scenario", "external", "test-source", "2026-01-01T00:00:00Z")

		if result["scenario_name"] != "my-scenario" {
			t.Errorf("expected scenario_name 'my-scenario', got %v", result["scenario_name"])
		}
		if result["deployment_mode"] != "external" {
			t.Errorf("expected deployment_mode 'external', got %v", result["deployment_mode"])
		}
		if result["source"] != "test-source" {
			t.Errorf("expected source 'test-source', got %v", result["source"])
		}
		if result["ingested_at"] != "2026-01-01T00:00:00Z" {
			t.Errorf("expected ingested_at '2026-01-01T00:00:00Z', got %v", result["ingested_at"])
		}
	})

	t.Run("defaults level to info", func(t *testing.T) {
		event := map[string]interface{}{"event": "test"}
		result := normalizeEvent(event, "scenario", "mode", "source", "time")

		if result["level"] != "info" {
			t.Errorf("expected level 'info', got %v", result["level"])
		}
	})

	t.Run("preserves existing level", func(t *testing.T) {
		event := map[string]interface{}{"event": "test", "level": "error"}
		result := normalizeEvent(event, "scenario", "mode", "source", "time")

		if result["level"] != "error" {
			t.Errorf("expected level 'error', got %v", result["level"])
		}
	})

	t.Run("normalizes serverType to server_type", func(t *testing.T) {
		event := map[string]interface{}{"event": "test", "serverType": "proxy"}
		result := normalizeEvent(event, "scenario", "mode", "source", "time")

		if result["server_type"] != "proxy" {
			t.Errorf("expected server_type 'proxy', got %v", result["server_type"])
		}
	})

	t.Run("falls back to event deployment mode", func(t *testing.T) {
		event := map[string]interface{}{"event": "test", "deploymentMode": "bundled"}
		result := normalizeEvent(event, "scenario", "", "source", "time")

		if result["deployment_mode"] != "bundled" {
			t.Errorf("expected deployment_mode 'bundled', got %v", result["deployment_mode"])
		}
	})

	t.Run("defaults empty source to desktop-runtime", func(t *testing.T) {
		event := map[string]interface{}{"event": "test"}
		result := normalizeEvent(event, "scenario", "mode", "", "time")

		if result["source"] != "desktop-runtime" {
			t.Errorf("expected source 'desktop-runtime', got %v", result["source"])
		}
	})
}

func TestGetInsights(t *testing.T) {
	vrooliRoot := t.TempDir()
	service := NewService(vrooliRoot)
	ctx := context.Background()

	t.Run("missing file", func(t *testing.T) {
		insights, err := service.GetInsights(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("GetInsights error: %v", err)
		}
		if insights.Exists {
			t.Errorf("expected exists=false for missing file")
		}
	})

	t.Run("with session data", func(t *testing.T) {
		telemetryDir := filepath.Join(vrooliRoot, ".vrooli", "deployment", "telemetry")
		if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
			t.Fatalf("failed to create telemetry dir: %v", err)
		}

		content := strings.Join([]string{
			`{"event":"app_start","ingested_at":"2026-01-01T00:00:00Z"}`,
			`{"event":"app_ready","ingested_at":"2026-01-01T00:01:00Z"}`,
			`{"event":"app_session_succeeded","ingested_at":"2026-01-01T00:02:00Z","session_id":"sess-1","details":{"started_at":"2026-01-01T00:00:00Z","ready_at":"2026-01-01T00:01:00Z"}}`,
		}, "\n")

		filePath := filepath.Join(telemetryDir, "insights-scenario.jsonl")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		insights, err := service.GetInsights(ctx, "insights-scenario")
		if err != nil {
			t.Fatalf("GetInsights error: %v", err)
		}

		if !insights.Exists {
			t.Errorf("expected exists=true")
		}
		if insights.LastSession == nil {
			t.Fatalf("expected LastSession to be set")
		}
		if insights.LastSession.Status != "succeeded" {
			t.Errorf("expected session status 'succeeded', got %q", insights.LastSession.Status)
		}
	})

	t.Run("with smoke test data", func(t *testing.T) {
		telemetryDir := filepath.Join(vrooliRoot, ".vrooli", "deployment", "telemetry")
		content := strings.Join([]string{
			`{"event":"smoke_test_passed","ingested_at":"2026-01-01T00:00:00Z","session_id":"smoke-1"}`,
		}, "\n")

		filePath := filepath.Join(telemetryDir, "smoke-scenario.jsonl")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		insights, err := service.GetInsights(ctx, "smoke-scenario")
		if err != nil {
			t.Fatalf("GetInsights error: %v", err)
		}

		if insights.LastSmokeTest == nil {
			t.Fatalf("expected LastSmokeTest to be set")
		}
		if insights.LastSmokeTest.Status != "passed" {
			t.Errorf("expected smoke test status 'passed', got %q", insights.LastSmokeTest.Status)
		}
	})

	t.Run("with error data", func(t *testing.T) {
		telemetryDir := filepath.Join(vrooliRoot, ".vrooli", "deployment", "telemetry")
		content := strings.Join([]string{
			`{"event":"service_error","level":"error","ingested_at":"2026-01-01T00:00:00Z","details":{"error":"test error"}}`,
		}, "\n")

		filePath := filepath.Join(telemetryDir, "error-scenario.jsonl")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		insights, err := service.GetInsights(ctx, "error-scenario")
		if err != nil {
			t.Fatalf("GetInsights error: %v", err)
		}

		if insights.LastError == nil {
			t.Fatalf("expected LastError to be set")
		}
		if insights.LastError.Message != "test error" {
			t.Errorf("expected error message 'test error', got %q", insights.LastError.Message)
		}
	})

	t.Run("infers session from app lifecycle", func(t *testing.T) {
		telemetryDir := filepath.Join(vrooliRoot, ".vrooli", "deployment", "telemetry")
		content := strings.Join([]string{
			`{"event":"app_start","ingested_at":"2026-01-01T00:00:00Z"}`,
			`{"event":"app_ready","ingested_at":"2026-01-01T00:01:00Z"}`,
			`{"event":"app_shutdown","ingested_at":"2026-01-01T00:02:00Z"}`,
		}, "\n")

		filePath := filepath.Join(telemetryDir, "lifecycle-scenario.jsonl")
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		insights, err := service.GetInsights(ctx, "lifecycle-scenario")
		if err != nil {
			t.Fatalf("GetInsights error: %v", err)
		}

		if insights.LastSession == nil {
			t.Fatalf("expected LastSession to be inferred from lifecycle")
		}
		if insights.LastSession.Status != "succeeded" {
			t.Errorf("expected inferred session status 'succeeded', got %q", insights.LastSession.Status)
		}
	})
}

func TestGetTail(t *testing.T) {
	vrooliRoot := t.TempDir()
	service := NewService(vrooliRoot)
	ctx := context.Background()

	t.Run("missing file", func(t *testing.T) {
		result, err := service.GetTail(ctx, "nonexistent", 10)
		if err != nil {
			t.Fatalf("GetTail error: %v", err)
		}
		if result.Exists {
			t.Errorf("expected exists=false for missing file")
		}
	})

	t.Run("returns last N entries", func(t *testing.T) {
		telemetryDir := filepath.Join(vrooliRoot, ".vrooli", "deployment", "telemetry")
		if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
			t.Fatalf("failed to create telemetry dir: %v", err)
		}

		var lines []string
		for i := 1; i <= 20; i++ {
			lines = append(lines, `{"event":"event_`+string(rune('A'+i-1))+`","num":`+string(rune('0'+i%10))+`}`)
		}

		filePath := filepath.Join(telemetryDir, "tail-scenario.jsonl")
		if err := os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		result, err := service.GetTail(ctx, "tail-scenario", 5)
		if err != nil {
			t.Fatalf("GetTail error: %v", err)
		}

		if !result.Exists {
			t.Errorf("expected exists=true")
		}
		if result.TotalLines != 20 {
			t.Errorf("expected 20 total lines, got %d", result.TotalLines)
		}
		if len(result.Entries) != 5 {
			t.Errorf("expected 5 entries, got %d", len(result.Entries))
		}
	})
}

func TestDelete(t *testing.T) {
	vrooliRoot := t.TempDir()
	service := NewService(vrooliRoot)
	ctx := context.Background()

	t.Run("delete missing file", func(t *testing.T) {
		err := service.Delete(ctx, "nonexistent")
		if err == nil {
			t.Fatalf("expected error for deleting nonexistent file")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got %v", err)
		}
	})

	t.Run("delete existing file", func(t *testing.T) {
		telemetryDir := filepath.Join(vrooliRoot, ".vrooli", "deployment", "telemetry")
		if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
			t.Fatalf("failed to create telemetry dir: %v", err)
		}

		filePath := filepath.Join(telemetryDir, "delete-scenario.jsonl")
		if err := os.WriteFile(filePath, []byte(`{"event":"test"}`), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		err := service.Delete(ctx, "delete-scenario")
		if err != nil {
			t.Fatalf("Delete error: %v", err)
		}

		// Verify file is deleted
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Errorf("expected file to be deleted")
		}
	})
}

func TestIsErrorEvent(t *testing.T) {
	tests := []struct {
		event    string
		expected bool
	}{
		{"smoke_test_failed", true},
		{"startup_error", true},
		{"runtime_error", true},
		{"migration_failed", true},
		{"asset_missing", true},
		{"app_start", false},
		{"app_ready", false},
		{"app_shutdown", false},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			got := IsErrorEvent(tt.event)
			if got != tt.expected {
				t.Errorf("IsErrorEvent(%q) = %v, want %v", tt.event, got, tt.expected)
			}
		})
	}
}

func TestParseEventTimestamp(t *testing.T) {
	t.Run("parses ingested_at", func(t *testing.T) {
		payload := map[string]interface{}{
			"ingested_at": "2026-01-01T12:00:00Z",
		}
		ts, ok := ParseEventTimestamp(payload)
		if !ok {
			t.Fatalf("expected timestamp to be parsed")
		}
		if ts.Year() != 2026 {
			t.Errorf("expected year 2026, got %d", ts.Year())
		}
	})

	t.Run("parses timestamp field", func(t *testing.T) {
		payload := map[string]interface{}{
			"timestamp": "2026-02-01T12:00:00Z",
		}
		ts, ok := ParseEventTimestamp(payload)
		if !ok {
			t.Fatalf("expected timestamp to be parsed")
		}
		if ts.Month() != 2 {
			t.Errorf("expected month 2, got %d", ts.Month())
		}
	})

	t.Run("missing timestamp returns false", func(t *testing.T) {
		payload := map[string]interface{}{
			"event": "test",
		}
		_, ok := ParseEventTimestamp(payload)
		if ok {
			t.Errorf("expected false for missing timestamp")
		}
	})
}

func TestFormatTimeIfSet(t *testing.T) {
	t.Run("zero time returns empty", func(t *testing.T) {
		var zero time.Time
		result := FormatTimeIfSet(zero)
		if result != "" {
			t.Errorf("expected empty string for zero time, got %q", result)
		}
	})

	t.Run("non-zero time returns formatted", func(t *testing.T) {
		ts := time.Date(2026, 1, 15, 12, 30, 0, 0, time.UTC)
		result := FormatTimeIfSet(ts)
		if result != "2026-01-15T12:30:00Z" {
			t.Errorf("expected '2026-01-15T12:30:00Z', got %q", result)
		}
	})
}

func TestStringFromPayload(t *testing.T) {
	t.Run("existing string", func(t *testing.T) {
		payload := map[string]interface{}{
			"key": "value",
		}
		result := StringFromPayload(payload, "key")
		if result != "value" {
			t.Errorf("expected 'value', got %q", result)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		payload := map[string]interface{}{}
		result := StringFromPayload(payload, "key")
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("non-string value", func(t *testing.T) {
		payload := map[string]interface{}{
			"key": 123,
		}
		result := StringFromPayload(payload, "key")
		if result != "" {
			t.Errorf("expected empty string for non-string, got %q", result)
		}
	})
}

func TestBuildSessionInsight(t *testing.T) {
	payload := map[string]interface{}{
		"session_id": "sess-123",
		"details": map[string]interface{}{
			"started_at": "2026-01-01T00:00:00Z",
			"ready_at":   "2026-01-01T00:01:00Z",
			"reason":     "test reason",
		},
	}
	ts := time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC)

	t.Run("succeeded", func(t *testing.T) {
		insight := buildSessionInsight(payload, "app_session_succeeded", ts)
		if insight.Status != "succeeded" {
			t.Errorf("expected status 'succeeded', got %q", insight.Status)
		}
		if insight.SessionID != "sess-123" {
			t.Errorf("expected session_id 'sess-123', got %q", insight.SessionID)
		}
	})

	t.Run("failed", func(t *testing.T) {
		insight := buildSessionInsight(payload, "app_session_failed", ts)
		if insight.Status != "failed" {
			t.Errorf("expected status 'failed', got %q", insight.Status)
		}
	})
}

func TestBuildSmokeTestInsight(t *testing.T) {
	payload := map[string]interface{}{
		"session_id": "smoke-123",
		"details": map[string]interface{}{
			"error": "test error",
		},
	}
	ts1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)

	t.Run("started", func(t *testing.T) {
		insight := buildSmokeTestInsight(payload, "smoke_test_started", ts1)
		if insight.Status != "started" {
			t.Errorf("expected status 'started', got %q", insight.Status)
		}
		if insight.StartedAt == "" {
			t.Errorf("expected StartedAt to be set")
		}
	})

	t.Run("passed", func(t *testing.T) {
		insight := buildSmokeTestInsight(payload, "smoke_test_passed", ts2)
		if insight.Status != "passed" {
			t.Errorf("expected status 'passed', got %q", insight.Status)
		}
		if insight.CompletedAt == "" {
			t.Errorf("expected CompletedAt to be set")
		}
	})

	t.Run("failed", func(t *testing.T) {
		insight := buildSmokeTestInsight(payload, "smoke_test_failed", ts2)
		if insight.Status != "failed" {
			t.Errorf("expected status 'failed', got %q", insight.Status)
		}
		if insight.Error != "test error" {
			t.Errorf("expected error 'test error', got %q", insight.Error)
		}
	})
}

func TestBuildErrorInsight(t *testing.T) {
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("with error in details", func(t *testing.T) {
		payload := map[string]interface{}{
			"details": map[string]interface{}{
				"error": "test error message",
			},
		}
		insight := buildErrorInsight(payload, "service_error", ts)
		if insight.Event != "service_error" {
			t.Errorf("expected event 'service_error', got %q", insight.Event)
		}
		if insight.Message != "test error message" {
			t.Errorf("expected message 'test error message', got %q", insight.Message)
		}
	})

	t.Run("with message in details", func(t *testing.T) {
		payload := map[string]interface{}{
			"details": map[string]interface{}{
				"message": "fallback message",
			},
		}
		insight := buildErrorInsight(payload, "service_error", ts)
		if insight.Message != "fallback message" {
			t.Errorf("expected message 'fallback message', got %q", insight.Message)
		}
	})
}

func TestIsError(t *testing.T) {
	t.Run("level error", func(t *testing.T) {
		payload := map[string]interface{}{"level": "error"}
		if !isError(payload, "any_event") {
			t.Errorf("expected isError=true for level=error")
		}
	})

	t.Run("error event", func(t *testing.T) {
		payload := map[string]interface{}{}
		if !isError(payload, "smoke_test_failed") {
			t.Errorf("expected isError=true for error event")
		}
	})

	t.Run("non-error", func(t *testing.T) {
		payload := map[string]interface{}{}
		if isError(payload, "app_start") {
			t.Errorf("expected isError=false for non-error event")
		}
	})
}
