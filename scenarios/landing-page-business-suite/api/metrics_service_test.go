package main

import (
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"
)

func setupMetricsTestDB(t *testing.T) (*sql.DB, *MetricsService) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
		if dbURL == "" {
			t.Skip("TEST_DATABASE_URL or DATABASE_URL not set")
		}
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Clean up test data
	db.Exec("DELETE FROM metrics_events")

	service := NewMetricsService(db)
	return db, service
}

// TestTrackEvent_Valid tests successful event tracking
func TestTrackEvent_Valid(t *testing.T) {
	db, service := setupMetricsTestDB(t)
	defer db.Close()

	event := MetricEvent{
		EventType: "page_view",
		VariantID: 1,
		SessionID: "test-session-123",
		VisitorID: "visitor-456",
		EventData: map[string]interface{}{
			"page": "/",
		},
	}

	err := service.TrackEvent(event)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify event was inserted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM metrics_events WHERE session_id = $1", event.SessionID).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 event, got %d", count)
	}
}

// TestTrackEvent_Idempotency tests that duplicate events are ignored
func TestTrackEvent_Idempotency(t *testing.T) {
	db, service := setupMetricsTestDB(t)
	defer db.Close()

	event := MetricEvent{
		EventType: "page_view",
		VariantID: 1,
		SessionID: "test-session-idem",
		EventID:   "unique-event-123",
	}

	// Track event twice
	err1 := service.TrackEvent(event)
	err2 := service.TrackEvent(event)

	if err1 != nil {
		t.Errorf("First track failed: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second track failed: %v", err2)
	}

	// Verify only one event was inserted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM metrics_events WHERE session_id = $1", event.SessionID).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 event (idempotency), got %d", count)
	}
}

// TestTrackEvent_InvalidEventType tests validation of event_type
func TestTrackEvent_InvalidEventType(t *testing.T) {
	_, service := setupMetricsTestDB(t)

	event := MetricEvent{
		EventType: "invalid_type",
		VariantID: 1,
		SessionID: "test-session",
	}

	err := service.TrackEvent(event)
	var validationErr *MetricValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected MetricValidationError, got %v", err)
	}
	if validationErr.Field != "event_type" {
		t.Fatalf("expected field event_type, got %s", validationErr.Field)
	}
}

func TestTrackEvent_MissingRequiredFields(t *testing.T) {
	_, service := setupMetricsTestDB(t)

	event := MetricEvent{
		EventType: "page_view",
		VariantID: 0,
		SessionID: "",
	}

	var validationErr *MetricValidationError
	if err := service.TrackEvent(event); !errors.As(err, &validationErr) {
		t.Fatalf("expected MetricValidationError, got %v", err)
	}
}

// TestGetVariantStats tests variant statistics retrieval
func TestGetVariantStats(t *testing.T) {
	db, service := setupMetricsTestDB(t)
	defer db.Close()

	// Insert test events
	events := []MetricEvent{
		{EventType: "page_view", VariantID: 1, SessionID: "session1", EventID: "evt1"},
		{EventType: "page_view", VariantID: 1, SessionID: "session2", EventID: "evt2"},
		{EventType: "click", VariantID: 1, SessionID: "session1", EventID: "evt3", EventData: map[string]interface{}{"element_type": "cta"}},
		{EventType: "conversion", VariantID: 1, SessionID: "session1", EventID: "evt4"},
		{EventType: "download", VariantID: 1, SessionID: "session1", EventID: "evt_download", EventData: map[string]interface{}{"platform": "windows"}},
		{EventType: "page_view", VariantID: 2, SessionID: "session3", EventID: "evt5"},
	}

	for _, evt := range events {
		service.TrackEvent(evt)
	}

	// Get stats for all variants
	startDate := time.Now().AddDate(0, 0, -1)
	endDate := time.Now().AddDate(0, 0, 1)
	stats, err := service.GetVariantStats(startDate, endDate, "")

	if err != nil {
		t.Fatalf("GetVariantStats failed: %v", err)
	}

	if len(stats) < 2 {
		t.Errorf("Expected stats for at least 2 variants, got %d", len(stats))
	}

	// Find variant 1 stats
	var variant1Stats *VariantStats
	for i := range stats {
		if stats[i].VariantID == 1 {
			variant1Stats = &stats[i]
			break
		}
	}

	if variant1Stats == nil {
		t.Fatal("No stats found for variant 1")
	}

	if variant1Stats.Views != 2 {
		t.Errorf("Expected 2 views for variant 1, got %d", variant1Stats.Views)
	}
	if variant1Stats.CTAClicks != 1 {
		t.Errorf("Expected 1 CTA click for variant 1, got %d", variant1Stats.CTAClicks)
	}
	if variant1Stats.Conversions != 1 {
		t.Errorf("Expected 1 conversion for variant 1, got %d", variant1Stats.Conversions)
	}
	if variant1Stats.Downloads != 1 {
		t.Errorf("Expected 1 download for variant 1, got %d", variant1Stats.Downloads)
	}
	if variant1Stats.ConversionRate != 50.0 {
		t.Errorf("Expected 50%% conversion rate, got %.2f", variant1Stats.ConversionRate)
	}
}

// TestGetVariantStats_FilterBySlug tests filtering stats by variant slug
func TestGetVariantStats_FilterBySlug(t *testing.T) {
	db, service := setupMetricsTestDB(t)
	defer db.Close()

	// Insert test events for variant 1 (control)
	service.TrackEvent(MetricEvent{
		EventType: "page_view",
		VariantID: 1,
		SessionID: "session1",
		EventID:   "evt-filter-1",
	})

	startDate := time.Now().AddDate(0, 0, -1)
	endDate := time.Now().AddDate(0, 0, 1)

	// Filter by "control" slug (variant 1)
	stats, err := service.GetVariantStats(startDate, endDate, "control")
	if err != nil {
		t.Fatalf("GetVariantStats with filter failed: %v", err)
	}

	if len(stats) != 1 {
		t.Errorf("Expected 1 variant (control), got %d", len(stats))
	}
	if len(stats) > 0 && stats[0].VariantSlug != "control" {
		t.Errorf("Expected control variant, got %s", stats[0].VariantSlug)
	}
}

// TestGetAnalyticsSummary tests the analytics summary aggregation
func TestGetAnalyticsSummary(t *testing.T) {
	db, service := setupMetricsTestDB(t)
	defer db.Close()

	// Insert test events
	events := []MetricEvent{
		{EventType: "page_view", VariantID: 1, SessionID: "session1", EventID: "sum1"},
		{EventType: "page_view", VariantID: 1, SessionID: "session2", EventID: "sum2"},
		{EventType: "click", VariantID: 1, SessionID: "session1", EventID: "sum3", EventData: map[string]interface{}{"element_id": "hero-cta", "element_type": "cta"}},
		{EventType: "click", VariantID: 1, SessionID: "session2", EventID: "sum4", EventData: map[string]interface{}{"element_id": "hero-cta", "element_type": "cta"}},
		{EventType: "conversion", VariantID: 1, SessionID: "session1", EventID: "sum5"},
		{EventType: "download", VariantID: 1, SessionID: "session1", EventID: "sum6"},
	}

	for _, evt := range events {
		service.TrackEvent(evt)
	}

	startDate := time.Now().AddDate(0, 0, -1)
	endDate := time.Now().AddDate(0, 0, 1)

	summary, err := service.GetAnalyticsSummary(startDate, endDate)
	if err != nil {
		t.Fatalf("GetAnalyticsSummary failed: %v", err)
	}

	if summary.TotalVisitors != 2 {
		t.Errorf("Expected 2 unique visitors, got %d", summary.TotalVisitors)
	}

	if summary.TopCTA != "hero-cta" {
		t.Errorf("Expected top CTA 'hero-cta', got '%s'", summary.TopCTA)
	}

	// CTR = (2 clicks / 2 views) * 100 = 100%
	if summary.TopCTACTR != 100.0 {
		t.Errorf("Expected top CTA CTR 100%%, got %.2f", summary.TopCTACTR)
	}

	if len(summary.VariantStats) == 0 {
		t.Error("Expected variant stats in summary, got none")
	}
	if summary.TotalDownloads != 1 {
		t.Errorf("Expected 1 download, got %d", summary.TotalDownloads)
	}
}

// TestGenerateEventID tests the event ID generation for idempotency
func TestGenerateEventID(t *testing.T) {
	event1 := MetricEvent{
		EventType: "page_view",
		VariantID: 1,
		SessionID: "session1",
	}
	event2 := MetricEvent{
		EventType: "page_view",
		VariantID: 1,
		SessionID: "session1",
	}

	// Same event attributes should generate same ID (within same second)
	id1 := generateEventID(event1)
	id2 := generateEventID(event2)

	if id1 != id2 {
		t.Errorf("Expected same event ID for identical events, got %s and %s", id1, id2)
	}

	// Different session should generate different ID
	event3 := MetricEvent{
		EventType: "page_view",
		VariantID: 1,
		SessionID: "session2",
	}
	id3 := generateEventID(event3)

	if id1 == id3 {
		t.Error("Expected different event IDs for different sessions")
	}
}
