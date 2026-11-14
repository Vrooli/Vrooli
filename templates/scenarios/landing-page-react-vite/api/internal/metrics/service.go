// Package metrics is the application layer for landing-page analytics:
// idempotent event ingestion and admin funnel/summary rollups. The Connect
// handler in handlers/metrics is a thin adapter over this Service.
package metrics

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Service owns analytics event ingestion and rollups.
type Service struct {
	db *sql.DB
}

// NewService constructs the metrics Service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

var validEventTypes = map[string]struct{}{
	"page_view":    {},
	"scroll_depth": {},
	"click":        {},
	"form_submit":  {},
	"conversion":   {},
	"download":     {},
}

// Event is a single analytics event to ingest.
type Event struct {
	EventType string
	VariantID int
	EventData map[string]interface{}
	SessionID string
	VisitorID string
	// EventID is a client-supplied idempotency key; when empty the server
	// derives one deterministically.
	EventID string
}

// ValidationError is a bad-request-level failure for event ingestion.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// VariantStats is the funnel rollup for one variant over a window.
type VariantStats struct {
	VariantID      int
	VariantSlug    string
	VariantName    string
	Views          int64
	CTAClicks      int64
	Conversions    int64
	Downloads      int64
	ConversionRate float64
}

// Summary is the aggregate rollup across all variants for a window.
type Summary struct {
	TotalVisitors  int64
	TotalDownloads int64
	VariantStats   []VariantStats
	TopCTA         string
	TopCTACTR      float64
}

// TrackEvent records an event, collapsing duplicates by idempotency id.
func (s *Service) TrackEvent(event Event) error {
	if err := validateEvent(event); err != nil {
		return err
	}

	eventID := event.EventID
	if eventID == "" {
		eventID = GenerateEventID(event)
	}

	var exists bool
	if err := s.db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM metrics_events WHERE event_data->>'event_id' = $1)`, eventID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("idempotency check failed: %w", err)
	}
	if exists {
		return nil
	}

	if event.EventData == nil {
		event.EventData = make(map[string]interface{})
	}
	event.EventData["event_id"] = eventID
	eventDataJSON, err := json.Marshal(event.EventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event_data: %w", err)
	}

	if _, err := s.db.Exec(`
		INSERT INTO metrics_events (variant_id, event_type, event_data, session_id, visitor_id)
		VALUES ($1, $2, $3, $4, $5)`,
		event.VariantID, event.EventType, eventDataJSON, event.SessionID, event.VisitorID); err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}
	return nil
}

func validateEvent(event Event) error {
	if strings.TrimSpace(event.EventType) == "" {
		return &ValidationError{Field: "event_type", Reason: "event_type is required"}
	}
	if _, ok := validEventTypes[event.EventType]; !ok {
		return &ValidationError{Field: "event_type", Reason: fmt.Sprintf("invalid event_type: %s", event.EventType)}
	}
	if event.VariantID <= 0 {
		return &ValidationError{Field: "variant_id", Reason: "variant_id is required"}
	}
	if strings.TrimSpace(event.SessionID) == "" {
		return &ValidationError{Field: "session_id", Reason: "session_id is required"}
	}
	return nil
}

// GenerateEventID derives a deterministic idempotency id from the event's
// (session, type, variant, unix-second) tuple.
func GenerateEventID(event Event) string {
	timestamp := time.Now().Unix()
	input := fmt.Sprintf("%s:%s:%d:%d", event.SessionID, event.EventType, event.VariantID, timestamp)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:16])
}

// GetVariantStats returns per-variant funnel stats for a window, optionally
// filtered to a single slug.
func (s *Service) GetVariantStats(startDate, endDate time.Time, variantSlug string) ([]VariantStats, error) {
	query := `
		SELECT v.id, v.slug, v.name,
			COALESCE(SUM(CASE WHEN m.event_type = 'page_view' THEN 1 ELSE 0 END), 0) AS views,
			COALESCE(SUM(CASE WHEN m.event_type = 'click' AND m.event_data->>'element_type' = 'cta' THEN 1 ELSE 0 END), 0) AS cta_clicks,
			COALESCE(SUM(CASE WHEN m.event_type = 'conversion' THEN 1 ELSE 0 END), 0) AS conversions,
			COALESCE(SUM(CASE WHEN m.event_type = 'download' THEN 1 ELSE 0 END), 0) AS downloads
		FROM variants v
		LEFT JOIN metrics_events m ON v.id = m.variant_id AND m.created_at >= $1 AND m.created_at <= $2
		WHERE v.status = 'active'`
	args := []interface{}{startDate, endDate}
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
		if err := rows.Scan(&stat.VariantID, &stat.VariantSlug, &stat.VariantName,
			&stat.Views, &stat.CTAClicks, &stat.Conversions, &stat.Downloads); err != nil {
			return nil, fmt.Errorf("failed to scan variant stats: %w", err)
		}
		if stat.Views > 0 {
			stat.ConversionRate = float64(stat.Conversions) / float64(stat.Views) * 100
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

// GetAnalyticsSummary returns the aggregate rollup for a window.
func (s *Service) GetAnalyticsSummary(startDate, endDate time.Time) (*Summary, error) {
	stats, err := s.GetVariantStats(startDate, endDate, "")
	if err != nil {
		return nil, err
	}

	var totalVisitors int64
	if err := s.db.QueryRow(`
		SELECT COUNT(DISTINCT session_id) FROM metrics_events
		WHERE event_type = 'page_view' AND created_at >= $1 AND created_at <= $2`,
		startDate, endDate).Scan(&totalVisitors); err != nil {
		return nil, fmt.Errorf("failed to count visitors: %w", err)
	}

	var totalDownloads int64
	if err := s.db.QueryRow(`
		SELECT COUNT(*) FROM metrics_events
		WHERE event_type = 'download' AND created_at >= $1 AND created_at <= $2`,
		startDate, endDate).Scan(&totalDownloads); err != nil {
		return nil, fmt.Errorf("failed to count downloads: %w", err)
	}

	var topCTA string
	var topCTAClicks, topCTAViews int64
	err = s.db.QueryRow(`
		SELECT m.event_data->>'element_id' AS cta_id, COUNT(*) AS clicks,
			(SELECT COUNT(DISTINCT session_id) FROM metrics_events WHERE event_type = 'page_view' AND created_at >= $1 AND created_at <= $2) AS views
		FROM metrics_events m
		WHERE m.event_type = 'click' AND m.event_data->>'element_type' = 'cta'
			AND m.created_at >= $1 AND m.created_at <= $2
		GROUP BY m.event_data->>'element_id'
		ORDER BY clicks DESC
		LIMIT 1`, startDate, endDate).Scan(&topCTA, &topCTAClicks, &topCTAViews)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query top CTA: %w", err)
	}

	var topCTACTR float64
	if topCTAViews > 0 {
		topCTACTR = float64(topCTAClicks) / float64(topCTAViews) * 100
	}

	return &Summary{
		TotalVisitors:  totalVisitors,
		TotalDownloads: totalDownloads,
		VariantStats:   stats,
		TopCTA:         topCTA,
		TopCTACTR:      topCTACTR,
	}, nil
}
