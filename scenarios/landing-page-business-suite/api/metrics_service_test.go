package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	domainmetrics "landing-page-business-suite-api/internal/metrics"
)

func setupMetricsTestDB(t *testing.T) (*sql.DB, *domainmetrics.Service) {
	t.Helper()
	db := setupTestDB(t)

	// Clean up test data
	if _, err := db.Exec("DELETE FROM metrics_events"); err != nil {
		t.Fatalf("failed to clean metrics_events: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Ping(); err == nil {
			if _, err := db.Exec("DELETE FROM metrics_events"); err != nil {
				t.Fatalf("failed to clean metrics_events: %v", err)
			}
		}
		_ = db.Close()
	})

	service := NewMetricsService(db)
	return db, service
}

// [REQ:METRIC-TAG] Tracked events preserve the variant identity supplied by the client.
// TestTrackEvent_Valid tests successful event tracking
func TestTrackEvent_Valid(t *testing.T) {
	db, service := setupMetricsTestDB(t)

	event := domainmetrics.Event{
		EventType:   "page_view",
		VariantSlug: "control",
		SessionID:   "test-session-123",
		VisitorID:   "visitor-456",
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
	var storedVariant string
	if err := db.QueryRow("SELECT COUNT(*), MIN(variant_slug) FROM metrics_events WHERE session_id = $1", event.SessionID).Scan(&count, &storedVariant); err != nil {
		t.Fatalf("failed to load metrics event: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 event, got %d", count)
	}
	if storedVariant != event.VariantSlug {
		t.Errorf("Expected variant %q, got %q", event.VariantSlug, storedVariant)
	}
}

// [REQ:METRIC-IDEMPOTENT] Duplicate event IDs do not create additional metric rows.
// TestTrackEvent_Idempotency tests that duplicate events are ignored
func TestTrackEvent_Idempotency(t *testing.T) {
	db, service := setupMetricsTestDB(t)

	event := domainmetrics.Event{
		EventType:   "page_view",
		VariantSlug: "control",
		SessionID:   "test-session-idem",
		EventID:     "unique-event-123",
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
	if err := db.QueryRow("SELECT COUNT(*) FROM metrics_events WHERE session_id = $1", event.SessionID).Scan(&count); err != nil {
		t.Fatalf("failed to count metrics events: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 event (idempotency), got %d", count)
	}
}

func TestTrackEvent_AppendsGeneratedEventID(t *testing.T) {
	db, service := setupMetricsTestDB(t)

	event := domainmetrics.Event{
		EventType:   "page_view",
		VariantSlug: "control",
		SessionID:   "test-session-generated",
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
	db.Close()

	err := service.TrackEvent(domainmetrics.Event{
		EventType:   "page_view",
		VariantSlug: "control",
		SessionID:   "session-closed-db",
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

	event := domainmetrics.Event{
		EventType:   "invalid_type",
		VariantSlug: "control",
		SessionID:   "test-session",
	}

	err := service.TrackEvent(event)
	var validationErr *domainmetrics.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected MetricValidationError, got %v", err)
	}
	if validationErr.Field != "event_type" {
		t.Fatalf("expected field event_type, got %s", validationErr.Field)
	}
}

func TestTrackEvent_MissingRequiredFields(t *testing.T) {
	_, service := setupMetricsTestDB(t)

	event := domainmetrics.Event{
		EventType:   "page_view",
		VariantSlug: "",
		SessionID:   "",
	}

	var validationErr *domainmetrics.ValidationError
	if err := service.TrackEvent(event); !errors.As(err, &validationErr) {
		t.Fatalf("expected MetricValidationError, got %v", err)
	}
}

func TestTrackEvent_MissingVariantSlug(t *testing.T) {
	_, service := setupMetricsTestDB(t)

	event := domainmetrics.Event{
		EventType: "page_view",
		SessionID: "test-session",
		// VariantSlug is missing
	}

	err := service.TrackEvent(event)
	var validationErr *domainmetrics.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected MetricValidationError, got %v", err)
	}
	if validationErr.Field != "variant_slug" {
		t.Fatalf("expected field variant_slug, got %s", validationErr.Field)
	}
}

// TestGetVariantStats tests variant statistics retrieval
func TestGetVariantStats(t *testing.T) {
	db, service := setupMetricsTestDB(t)

	// Insert test events
	events := []domainmetrics.Event{
		{EventType: "page_view", VariantSlug: "control", SessionID: "session1", EventID: "evt1"},
		{EventType: "page_view", VariantSlug: "control", SessionID: "session2", EventID: "evt2"},
		{EventType: "click", VariantSlug: "control", SessionID: "session1", EventID: "evt3", EventData: map[string]interface{}{"element_type": "cta"}},
		{EventType: "conversion", VariantSlug: "control", SessionID: "session1", EventID: "evt4"},
		{EventType: "download", VariantSlug: "control", SessionID: "session1", EventID: "evt_download", EventData: map[string]interface{}{"platform": "windows"}},
		{EventType: "page_view", VariantSlug: "variant-a", SessionID: "session3", EventID: "evt5"},
	}

	for _, evt := range events {
		if err := service.TrackEvent(evt); err != nil {
			t.Fatalf("failed to track event: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO experiment_exposures (visitor_id, variant_slug, weight_fingerprint) VALUES
		('visitor-1', 'control', 'weights-v1'), ('visitor-2', 'control', 'weights-v1')`); err != nil {
		t.Fatalf("failed to seed exposures: %v", err)
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

	// Find control stats
	var controlStats *domainmetrics.VariantStats
	for i := range stats {
		if stats[i].VariantSlug == "control" {
			controlStats = &stats[i]
			break
		}
	}

	if controlStats == nil {
		t.Fatal("No stats found for control variant")
	}

	if controlStats.Views != 2 {
		t.Errorf("Expected 2 views for control, got %d", controlStats.Views)
	}
	if controlStats.CTAClicks != 1 {
		t.Errorf("Expected 1 CTA click for control, got %d", controlStats.CTAClicks)
	}
	if controlStats.Conversions != 1 {
		t.Errorf("Expected 1 conversion for control, got %d", controlStats.Conversions)
	}
	if controlStats.Downloads != 1 {
		t.Errorf("Expected 1 download for control, got %d", controlStats.Downloads)
	}
	if controlStats.Exposures != 2 {
		t.Errorf("Expected 2 exposures, got %d", controlStats.Exposures)
	}
	if controlStats.ConversionRate != 50.0 {
		t.Errorf("Expected 50%% exposure conversion rate, got %.2f", controlStats.ConversionRate)
	}
}

// [REQ:METRIC-FILTER] Analytics can be narrowed to one landing variant.
// TestGetVariantStats_FilterBySlug tests filtering stats by variant slug
func TestGetVariantStats_FilterBySlug(t *testing.T) {
	_, service := setupMetricsTestDB(t)

	// Insert test events for multiple variants
	if err := service.TrackEvent(domainmetrics.Event{
		EventType:   "page_view",
		VariantSlug: "control",
		SessionID:   "session1",
		EventID:     "evt-filter-1",
	}); err != nil {
		t.Fatalf("failed to track event: %v", err)
	}

	if err := service.TrackEvent(domainmetrics.Event{
		EventType:   "page_view",
		VariantSlug: "variant-a",
		SessionID:   "session2",
		EventID:     "evt-filter-2",
	}); err != nil {
		t.Fatalf("failed to track event: %v", err)
	}

	startDate := time.Now().AddDate(0, 0, -1)
	endDate := time.Now().AddDate(0, 0, 1)

	// Filter by "control" slug
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

// [REQ:METRIC-SUMMARY] Analytics summary includes visitors, variant conversion data, and CTA CTR.
// TestGetAnalyticsSummary tests the analytics summary aggregation
func TestGetAnalyticsSummary(t *testing.T) {
	_, service := setupMetricsTestDB(t)

	// Insert test events
	events := []domainmetrics.Event{
		{EventType: "page_view", VariantSlug: "control", SessionID: "session1", EventID: "sum1"},
		{EventType: "page_view", VariantSlug: "control", SessionID: "session2", EventID: "sum2"},
		{EventType: "click", VariantSlug: "control", SessionID: "session1", EventID: "sum3", EventData: map[string]interface{}{"element_id": "hero-cta", "element_type": "cta"}},
		{EventType: "click", VariantSlug: "control", SessionID: "session2", EventID: "sum4", EventData: map[string]interface{}{"element_id": "hero-cta", "element_type": "cta"}},
		{EventType: "conversion", VariantSlug: "control", SessionID: "session1", EventID: "sum5"},
		{EventType: "download", VariantSlug: "control", SessionID: "session1", EventID: "sum6"},
	}

	for _, evt := range events {
		if err := service.TrackEvent(evt); err != nil {
			t.Fatalf("failed to track event: %v", err)
		}
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
	if summary.ObservedAt == nil || summary.ObservedAt.After(time.Now().UTC()) {
		t.Errorf("Expected producer observation time at or before now, got %v", summary.ObservedAt)
	}
}

func TestGetAnalyticsSummary_NoCTAEvents(t *testing.T) {
	_, service := setupMetricsTestDB(t)

	if err := service.TrackEvent(domainmetrics.Event{
		EventType:   "page_view",
		VariantSlug: "control",
		SessionID:   "lonely-session",
		EventID:     "summary-no-cta",
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

func TestGetAnalyticsSummaryCountsStableVisitorsAcrossSessions(t *testing.T) {
	_, service := setupMetricsTestDB(t)
	for _, event := range []domainmetrics.Event{
		{EventType: "page_view", VariantSlug: "control", SessionID: "session-a", VisitorID: "visitor-stable", EventID: "visitor-session-a"},
		{EventType: "page_view", VariantSlug: "control", SessionID: "session-b", VisitorID: "visitor-stable", EventID: "visitor-session-b"},
	} {
		if err := service.TrackEvent(event); err != nil {
			t.Fatalf("failed to track event: %v", err)
		}
	}

	summary, err := service.GetAnalyticsSummary(time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("GetAnalyticsSummary failed: %v", err)
	}
	if summary.TotalVisitors != 1 {
		t.Fatalf("expected one stable visitor across two sessions, got %d", summary.TotalVisitors)
	}
}

func TestGetTrafficBreakdownKeepsFullShareDenominatorWhenLimited(t *testing.T) {
	_, service := setupMetricsTestDB(t)
	for i, campaign := range []string{"alpha", "beta", "gamma"} {
		if err := service.TrackEvent(domainmetrics.Event{
			EventType: "page_view", VariantSlug: "control", SessionID: fmt.Sprintf("breakdown-session-%d", i),
			VisitorID: fmt.Sprintf("breakdown-visitor-%d", i), EventID: fmt.Sprintf("breakdown-event-%d", i), UTMCampaign: campaign,
		}); err != nil {
			t.Fatalf("failed to track breakdown event: %v", err)
		}
	}

	breakdown, err := service.GetTrafficBreakdown("utm_campaign", time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("GetTrafficBreakdown failed: %v", err)
	}
	if len(breakdown.Rows) != 2 || breakdown.Exhaustive {
		t.Fatalf("expected two ranked rows and a truncated result, got rows=%d exhaustive=%v", len(breakdown.Rows), breakdown.Exhaustive)
	}
	if breakdown.TotalSessions != 3 {
		t.Fatalf("expected full session denominator of 3, got %d", breakdown.TotalSessions)
	}
	if breakdown.Rows[0].Share != 1.0/3.0 {
		t.Fatalf("expected top-row share to use full denominator, got %v", breakdown.Rows[0].Share)
	}
}

// TestGenerateEventID tests the event ID generation for idempotency
func TestGenerateEventID(t *testing.T) {
	event1 := domainmetrics.Event{
		EventType:   "page_view",
		VariantSlug: "control",
		SessionID:   "session1",
	}
	event2 := domainmetrics.Event{
		EventType:   "page_view",
		VariantSlug: "control",
		SessionID:   "session1",
	}

	// Same event attributes should generate same ID (within same second)
	id1 := generateEventID(event1)
	id2 := generateEventID(event2)

	if id1 != id2 {
		t.Errorf("Expected same event ID for identical events, got %s and %s", id1, id2)
	}

	// Different session should generate different ID
	event3 := domainmetrics.Event{
		EventType:   "page_view",
		VariantSlug: "control",
		SessionID:   "session2",
	}
	id3 := generateEventID(event3)

	if id1 == id3 {
		t.Error("Expected different event IDs for different sessions")
	}
}
