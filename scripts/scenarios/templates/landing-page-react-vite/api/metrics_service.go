package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// MetricsService handles event tracking and analytics
type MetricsService struct {
	db *sql.DB
}

// NewMetricsService creates a new metrics service
func NewMetricsService(db *sql.DB) *MetricsService {
	return &MetricsService{db: db}
}

// MetricEvent represents an analytics event
type MetricEvent struct {
	EventType string                 `json:"event_type"`
	VariantID int                    `json:"variant_id"`
	EventData map[string]interface{} `json:"event_data,omitempty"`
	SessionID string                 `json:"session_id"`
	VisitorID string                 `json:"visitor_id,omitempty"`
	EventID   string                 `json:"event_id,omitempty"` // Client-generated unique ID for idempotency
}

// VariantStats represents aggregated stats for a variant
type VariantStats struct {
	VariantID      int     `json:"variant_id"`
	VariantSlug    string  `json:"variant_slug"`
	VariantName    string  `json:"variant_name"`
	Views          int64   `json:"views"`
	CTAClicks      int64   `json:"cta_clicks"`
	Conversions    int64   `json:"conversions"`
	Downloads      int64   `json:"downloads"`
	ConversionRate float64 `json:"conversion_rate"`
	Trend          string  `json:"trend,omitempty"`
	AvgScrollDepth float64 `json:"avg_scroll_depth,omitempty"`
}

// AnalyticsSummary represents the analytics dashboard summary
type AnalyticsSummary struct {
	TotalVisitors  int64          `json:"total_visitors"`
	TotalDownloads int64          `json:"total_downloads"`
	VariantStats   []VariantStats `json:"variant_stats"`
	TopCTA         string         `json:"top_cta,omitempty"`
	TopCTACTR      float64        `json:"top_cta_ctr,omitempty"`
}

// TrackEvent records an analytics event with idempotency support
func (s *MetricsService) TrackEvent(event MetricEvent) error {
	// Validate event type
	validTypes := map[string]bool{
		"page_view": true, "scroll_depth": true, "click": true,
		"form_submit": true, "conversion": true, "download": true,
	}
	if !validTypes[event.EventType] {
		return fmt.Errorf("invalid event_type: %s", event.EventType)
	}

	// Generate event_id if not provided (for idempotency)
	eventID := event.EventID
	if eventID == "" {
		eventID = generateEventID(event)
	}

	// Check if event already exists (idempotency check)
	var exists bool
	checkQuery := `
		SELECT EXISTS(
			SELECT 1 FROM metrics_events
			WHERE event_data->>'event_id' = $1
		)
	`
	err := s.db.QueryRow(checkQuery, eventID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		// Event already recorded, skip insertion
		return nil
	}

	// Add event_id to event_data
	if event.EventData == nil {
		event.EventData = make(map[string]interface{})
	}
	event.EventData["event_id"] = eventID

	// Marshal event_data to JSON
	eventDataJSON, err := json.Marshal(event.EventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event_data: %w", err)
	}

	// Insert event
	query := `
		INSERT INTO metrics_events (variant_id, event_type, event_data, session_id, visitor_id)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = s.db.Exec(query, event.VariantID, event.EventType, eventDataJSON, event.SessionID, event.VisitorID)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

// generateEventID creates a unique event ID based on event attributes
func generateEventID(event MetricEvent) string {
	// Create deterministic hash from session_id + event_type + timestamp (rounded to second)
	timestamp := time.Now().Unix()
	input := fmt.Sprintf("%s:%s:%d:%d", event.SessionID, event.EventType, event.VariantID, timestamp)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:16]) // 32-char hex string
}

// GetVariantStats retrieves aggregated stats for variants within a time range
func (s *MetricsService) GetVariantStats(startDate, endDate time.Time, variantSlug string) ([]VariantStats, error) {
	query := `
		SELECT
			v.id as variant_id,
			v.slug as variant_slug,
			v.name as variant_name,
		COALESCE(SUM(CASE WHEN m.event_type = 'page_view' THEN 1 ELSE 0 END), 0) as views,
		COALESCE(SUM(CASE WHEN m.event_type = 'click' AND m.event_data->>'element_type' = 'cta' THEN 1 ELSE 0 END), 0) as cta_clicks,
		COALESCE(SUM(CASE WHEN m.event_type = 'conversion' THEN 1 ELSE 0 END), 0) as conversions,
		COALESCE(SUM(CASE WHEN m.event_type = 'download' THEN 1 ELSE 0 END), 0) as downloads
		FROM variants v
		LEFT JOIN metrics_events m ON v.id = m.variant_id
			AND m.created_at >= $1
			AND m.created_at <= $2
		WHERE v.status = 'active'
	`

	args := []interface{}{startDate, endDate}

	// Filter by variant slug if provided
	if variantSlug != "" {
		query += " AND v.slug = $3"
		args = append(args, variantSlug)
	}

	query += " GROUP BY v.id, v.slug, v.name ORDER BY v.id"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query variant stats: %w", err)
	}
	defer rows.Close()

	var stats []VariantStats
	for rows.Next() {
		var stat VariantStats
		err := rows.Scan(
			&stat.VariantID,
			&stat.VariantSlug,
			&stat.VariantName,
			&stat.Views,
			&stat.CTAClicks,
			&stat.Conversions,
			&stat.Downloads,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan variant stats: %w", err)
		}

		// Calculate conversion rate
		if stat.Views > 0 {
			stat.ConversionRate = float64(stat.Conversions) / float64(stat.Views) * 100
		}

		stats = append(stats, stat)
	}

	return stats, nil
}

// GetAnalyticsSummary retrieves the analytics dashboard summary
func (s *MetricsService) GetAnalyticsSummary(startDate, endDate time.Time) (*AnalyticsSummary, error) {
	// Get variant stats
	stats, err := s.GetVariantStats(startDate, endDate, "")
	if err != nil {
		return nil, err
	}

	// Calculate total unique visitors (distinct session_ids with page_view events)
	var totalVisitors int64
	visitorQuery := `
		SELECT COUNT(DISTINCT session_id)
		FROM metrics_events
		WHERE event_type = 'page_view'
			AND created_at >= $1
			AND created_at <= $2
	`
	err = s.db.QueryRow(visitorQuery, startDate, endDate).Scan(&totalVisitors)
	if err != nil {
		return nil, fmt.Errorf("failed to count visitors: %w", err)
	}

	var totalDownloads int64
	downloadQuery := `
		SELECT COUNT(*) FROM metrics_events
		WHERE event_type = 'download'
			AND created_at >= $1
			AND created_at <= $2
	`
	if err := s.db.QueryRow(downloadQuery, startDate, endDate).Scan(&totalDownloads); err != nil {
		return nil, fmt.Errorf("failed to count downloads: %w", err)
	}

	// Find top CTA by CTR
	topCTAQuery := `
		SELECT
			m.event_data->>'element_id' as cta_id,
			COUNT(*) as clicks,
			(SELECT COUNT(DISTINCT session_id) FROM metrics_events WHERE event_type = 'page_view' AND created_at >= $1 AND created_at <= $2) as views
		FROM metrics_events m
		WHERE m.event_type = 'click'
			AND m.event_data->>'element_type' = 'cta'
			AND m.created_at >= $1
			AND m.created_at <= $2
		GROUP BY m.event_data->>'element_id'
		ORDER BY clicks DESC
		LIMIT 1
	`

	var topCTA string
	var topCTAClicks, topCTAViews int64
	err = s.db.QueryRow(topCTAQuery, startDate, endDate).Scan(&topCTA, &topCTAClicks, &topCTAViews)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query top CTA: %w", err)
	}

	var topCTACTR float64
	if topCTAViews > 0 {
		topCTACTR = float64(topCTAClicks) / float64(topCTAViews) * 100
	}

	summary := &AnalyticsSummary{
		TotalVisitors:  totalVisitors,
		TotalDownloads: totalDownloads,
		VariantStats:   stats,
		TopCTA:         topCTA,
		TopCTACTR:      topCTACTR,
	}

	return summary, nil
}
