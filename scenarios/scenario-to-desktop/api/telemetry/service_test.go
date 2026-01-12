package telemetry

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceCreation(t *testing.T) {
	service := NewService("/tmp/test-vrooli")
	if service == nil {
		t.Fatalf("expected service to be created")
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
}
