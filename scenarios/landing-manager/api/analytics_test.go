package main

import (
	"os"
	"testing"
	"time"
)

// [REQ:TMPL-GENERATION-ANALYTICS] Unit tests for analytics service

func TestAnalyticsService_RecordGeneration(t *testing.T) {
	// Create temp directory for test data
	tmpDir, err := os.MkdirTemp("", "analytics-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("ANALYTICS_DATA_DIR", tmpDir)
	defer os.Unsetenv("ANALYTICS_DATA_DIR")

	as := NewAnalyticsService()

	// Record a successful generation
	as.RecordGeneration("saas-landing-page", "my-landing", false, true, "", 100)

	// Wait for async save
	time.Sleep(100 * time.Millisecond)

	events := as.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if events[0].TemplateID != "saas-landing-page" {
		t.Errorf("Expected template_id 'saas-landing-page', got '%s'", events[0].TemplateID)
	}

	if events[0].ScenarioID != "my-landing" {
		t.Errorf("Expected scenario_id 'my-landing', got '%s'", events[0].ScenarioID)
	}

	if !events[0].Success {
		t.Error("Expected success to be true")
	}

	if events[0].Duration != 100 {
		t.Errorf("Expected duration 100, got %d", events[0].Duration)
	}
}

func TestAnalyticsService_RecordDryRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "analytics-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("ANALYTICS_DATA_DIR", tmpDir)
	defer os.Unsetenv("ANALYTICS_DATA_DIR")

	as := NewAnalyticsService()

	// Record a dry-run generation
	as.RecordGeneration("saas-landing-page", "test-dry", true, true, "", 50)

	time.Sleep(100 * time.Millisecond)

	events := as.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if !events[0].IsDryRun {
		t.Error("Expected is_dry_run to be true")
	}
}

func TestAnalyticsService_RecordFailedGeneration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "analytics-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("ANALYTICS_DATA_DIR", tmpDir)
	defer os.Unsetenv("ANALYTICS_DATA_DIR")

	as := NewAnalyticsService()

	// Record a failed generation
	as.RecordGeneration("invalid-template", "failed-landing", false, false, "template not found", 10)

	time.Sleep(100 * time.Millisecond)

	events := as.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if events[0].Success {
		t.Error("Expected success to be false")
	}

	if events[0].ErrorReason != "template not found" {
		t.Errorf("Expected error_reason 'template not found', got '%s'", events[0].ErrorReason)
	}
}

func TestAnalyticsService_GetSummary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "analytics-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("ANALYTICS_DATA_DIR", tmpDir)
	defer os.Unsetenv("ANALYTICS_DATA_DIR")

	as := NewAnalyticsService()

	// Record multiple generations
	as.RecordGeneration("saas-landing-page", "landing-1", false, true, "", 100)
	as.RecordGeneration("saas-landing-page", "landing-2", false, true, "", 150)
	as.RecordGeneration("lead-magnet", "magnet-1", false, true, "", 120)
	as.RecordGeneration("saas-landing-page", "dry-test", true, true, "", 50)
	as.RecordGeneration("invalid", "failed", false, false, "error", 10)

	time.Sleep(100 * time.Millisecond)

	summary := as.GetSummary()

	if summary.TotalGenerations != 5 {
		t.Errorf("Expected 5 total generations, got %d", summary.TotalGenerations)
	}

	if summary.SuccessfulCount != 4 {
		t.Errorf("Expected 4 successful, got %d", summary.SuccessfulCount)
	}

	if summary.FailedCount != 1 {
		t.Errorf("Expected 1 failed, got %d", summary.FailedCount)
	}

	if summary.DryRunCount != 1 {
		t.Errorf("Expected 1 dry-run, got %d", summary.DryRunCount)
	}

	if summary.SuccessRate != 80.0 {
		t.Errorf("Expected 80%% success rate, got %.1f%%", summary.SuccessRate)
	}

	if summary.ByTemplate["saas-landing-page"] != 3 {
		t.Errorf("Expected 3 saas-landing-page generations, got %d", summary.ByTemplate["saas-landing-page"])
	}

	if summary.ByTemplate["lead-magnet"] != 1 {
		t.Errorf("Expected 1 lead-magnet generation, got %d", summary.ByTemplate["lead-magnet"])
	}
}

func TestAnalyticsService_EmptySummary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "analytics-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("ANALYTICS_DATA_DIR", tmpDir)
	defer os.Unsetenv("ANALYTICS_DATA_DIR")

	as := NewAnalyticsService()

	summary := as.GetSummary()

	if summary.TotalGenerations != 0 {
		t.Errorf("Expected 0 total generations, got %d", summary.TotalGenerations)
	}

	if summary.SuccessRate != 0 {
		t.Errorf("Expected 0%% success rate, got %.1f%%", summary.SuccessRate)
	}
}

func TestAnalyticsService_RecentEventsLimit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "analytics-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("ANALYTICS_DATA_DIR", tmpDir)
	defer os.Unsetenv("ANALYTICS_DATA_DIR")

	as := NewAnalyticsService()

	// Record 15 generations
	for i := 0; i < 15; i++ {
		as.RecordGeneration("saas-landing-page", "test", false, true, "", 100)
	}

	time.Sleep(100 * time.Millisecond)

	summary := as.GetSummary()

	// RecentEvents should be limited to last 10
	if len(summary.RecentEvents) != 10 {
		t.Errorf("Expected 10 recent events, got %d", len(summary.RecentEvents))
	}

	// Total should still be 15
	if summary.TotalGenerations != 15 {
		t.Errorf("Expected 15 total generations, got %d", summary.TotalGenerations)
	}
}

func TestAnalyticsService_Persistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "analytics-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("ANALYTICS_DATA_DIR", tmpDir)
	defer os.Unsetenv("ANALYTICS_DATA_DIR")

	// Create first service and record events
	as1 := NewAnalyticsService()
	as1.RecordGeneration("saas-landing-page", "persisted-1", false, true, "", 100)
	as1.RecordGeneration("saas-landing-page", "persisted-2", false, true, "", 150)

	// Wait for async save
	time.Sleep(200 * time.Millisecond)

	// Create new service instance - should load persisted events
	as2 := NewAnalyticsService()

	events := as2.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 persisted events, got %d", len(events))
	}

	if events[0].ScenarioID != "persisted-1" {
		t.Errorf("Expected first event to be 'persisted-1', got '%s'", events[0].ScenarioID)
	}
}

func TestAnalyticsService_AverageDuration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "analytics-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("ANALYTICS_DATA_DIR", tmpDir)
	defer os.Unsetenv("ANALYTICS_DATA_DIR")

	as := NewAnalyticsService()

	// Record with known durations: 100, 200, 300 -> average = 200
	as.RecordGeneration("test", "t1", false, true, "", 100)
	as.RecordGeneration("test", "t2", false, true, "", 200)
	as.RecordGeneration("test", "t3", false, true, "", 300)

	time.Sleep(100 * time.Millisecond)

	summary := as.GetSummary()

	if summary.AverageDurationMs != 200 {
		t.Errorf("Expected average duration 200ms, got %d ms", summary.AverageDurationMs)
	}
}
