package main

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

func setupMetricsTestDB(t *testing.T) (*sql.DB, *MetricsService) {
	t.Helper()
	db := setupTestDB(t)

	// Clean up test data
	if _, err := db.Exec("DELETE FROM metrics_events"); err != nil {
		t.Fatalf("failed to clean metrics_events: %v", err)
	}

	t.Cleanup(func() {
		db.Exec("DELETE FROM metrics_events")
		db.Close()
	})

	service := NewMetricsService(db)
	return db, service
}

func lookupVariantID(t *testing.T, db *sql.DB, slug string) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`SELECT id FROM variants WHERE slug = $1`, slug).Scan(&id); err != nil {
		t.Fatalf("failed to lookup variant %s: %v", slug, err)
	}
	return id
}

// TestTrackEvent_Valid tests successful event tracking
func TestTrackEvent_Valid(t *testing.T) {
	db, service := setupMetricsTestDB(t)
	controlID := lookupVariantID(t, db, "control")

	event := MetricEvent{
		EventType: "page_view",
		VariantID: controlID,
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
	controlID := lookupVariantID(t, db, "control")

	event := MetricEvent{
		EventType: "page_view",
		VariantID: controlID,
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

	var storedEventID string
	if err := db.QueryRow("SELECT event_data->>'event_id' FROM metrics_events WHERE session_id = $1", event.SessionID).Scan(&storedEventID); err != nil {
		t.Fatalf("failed to load stored event id: %v", err)
	}
	if storedEventID != event.EventID {
		t.Fatalf("expected stored event_id to match provided id, got %q", storedEventID)
	}
}

func TestTrackEvent_AppendsGeneratedEventID(t *testing.T) {
	db, service := setupMetricsTestDB(t)
	controlID := lookupVariantID(t, db, "control")

	event := MetricEvent{
		EventType: "page_view",
		VariantID: controlID,
		SessionID: "test-session-generated",
		EventData: map[string]interface{}{
			"page": "/",
		},
	}

	if err := service.TrackEvent(event); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var storedEventID, page string
	if err := db.QueryRow(`
		SELECT event_data->>'event_id', event_data->>'page'
		FROM metrics_events
		WHERE session_id = $1
	`, event.SessionID).Scan(&storedEventID, &page); err != nil {
		t.Fatalf("failed to load stored event data: %v", err)
	}
	if storedEventID == "" {
		t.Fatal("expected generated event_id to be persisted")
	}
	if page != "/" {
		t.Fatalf("expected page metadata preserved, got %q", page)
	}
}

func TestTrackEvent_IdempotencyCheckError(t *testing.T) {
	db, service := setupMetricsTestDB(t)
	controlID := lookupVariantID(t, db, "control")
	db.Close()

	err := service.TrackEvent(MetricEvent{
		EventType: "page_view",
		VariantID: controlID,
		SessionID: "session-closed-db",
	})
	if err == nil {
		t.Fatalf("expected error when idempotency check fails")
	}
	if !strings.Contains(err.Error(), "idempotency check failed") {
		t.Fatalf("expected idempotency failure in error, got %v", err)
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
	controlID := lookupVariantID(t, db, "control")
	variantAID := lookupVariantID(t, db, "variant-a")

	// Insert test events
	events := []MetricEvent{
		{EventType: "page_view", VariantID: controlID, SessionID: "session1", EventID: "evt1"},
		{EventType: "page_view", VariantID: controlID, SessionID: "session2", EventID: "evt2"},
		{EventType: "click", VariantID: controlID, SessionID: "session1", EventID: "evt3", EventData: map[string]interface{}{"element_type": "cta"}},
		{EventType: "conversion", VariantID: controlID, SessionID: "session1", EventID: "evt4"},
		{EventType: "download", VariantID: controlID, SessionID: "session1", EventID: "evt_download", EventData: map[string]interface{}{"platform": "windows"}},
		{EventType: "page_view", VariantID: variantAID, SessionID: "session3", EventID: "evt5"},
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
		if stats[i].VariantID == controlID {
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
	controlID := lookupVariantID(t, db, "control")

	// Insert test events for variant 1 (control)
	service.TrackEvent(MetricEvent{
		EventType: "page_view",
		VariantID: controlID,
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
	controlID := lookupVariantID(t, db, "control")

	// Insert test events
	events := []MetricEvent{
		{EventType: "page_view", VariantID: controlID, SessionID: "session1", EventID: "sum1"},
		{EventType: "page_view", VariantID: controlID, SessionID: "session2", EventID: "sum2"},
		{EventType: "click", VariantID: controlID, SessionID: "session1", EventID: "sum3", EventData: map[string]interface{}{"element_id": "hero-cta", "element_type": "cta"}},
		{EventType: "click", VariantID: controlID, SessionID: "session2", EventID: "sum4", EventData: map[string]interface{}{"element_id": "hero-cta", "element_type": "cta"}},
		{EventType: "conversion", VariantID: controlID, SessionID: "session1", EventID: "sum5"},
		{EventType: "download", VariantID: controlID, SessionID: "session1", EventID: "sum6"},
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

func TestGetAnalyticsSummary_NoCTAEvents(t *testing.T) {
	db, service := setupMetricsTestDB(t)
	controlID := lookupVariantID(t, db, "control")

	if err := service.TrackEvent(MetricEvent{
		EventType: "page_view",
		VariantID: controlID,
		SessionID: "lonely-session",
		EventID:   "summary-no-cta",
	}); err != nil {
		t.Fatalf("failed to seed page view: %v", err)
	}

	startDate := time.Now().AddDate(0, 0, -1)
	endDate := time.Now().AddDate(0, 0, 1)

	summary, err := service.GetAnalyticsSummary(startDate, endDate)
	if err != nil {
		t.Fatalf("GetAnalyticsSummary failed: %v", err)
	}

	if summary.TotalVisitors != 1 {
		t.Fatalf("expected one visitor, got %d", summary.TotalVisitors)
	}
	if summary.TopCTA != "" || summary.TopCTACTR != 0 {
		t.Fatalf("expected no CTA leader when none clicked, got id=%q ctr=%.2f", summary.TopCTA, summary.TopCTACTR)
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
