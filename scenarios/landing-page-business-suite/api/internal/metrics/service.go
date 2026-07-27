// Package metrics owns event validation, idempotent ingestion, and analytics
// aggregation. HTTP handlers remain at the transport edge and depend on this
// package rather than holding metrics rules themselves.
package metrics

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/clock"
)

type Service struct {
	db    Store
	clock clock.Clock
}

// Store is the persistence boundary for metrics ingestion and reporting.
type Store interface {
	QueryRow(string, ...any) *sql.Row
	Query(string, ...any) (*sql.Rows, error)
	Exec(string, ...any) (sql.Result, error)
}

var validEventTypes = map[string]struct{}{
	"page_view": {}, "scroll_depth": {}, "click": {}, "form_submit": {},
	"conversion": {}, "download": {},
}

// ValidationError identifies a request field that violates the metrics contract.
type ValidationError struct{ Field, Reason string }

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

func NewService(db Store) *Service { return NewServiceWithClock(db, clock.System{}) }

func NewServiceWithClock(db Store, serviceClock clock.Clock) *Service {
	if serviceClock == nil {
		serviceClock = clock.System{}
	}
	return &Service{db: db, clock: serviceClock}
}

type Event struct {
	EventType   string                 `json:"event_type"`
	VariantSlug string                 `json:"variant_slug"`
	EventData   map[string]interface{} `json:"event_data,omitempty"`
	SessionID   string                 `json:"session_id"`
	VisitorID   string                 `json:"visitor_id,omitempty"`
	EventID     string                 `json:"event_id,omitempty"`
}

type VariantStats struct {
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

type AnalyticsSummary struct {
	TotalVisitors  int64          `json:"total_visitors"`
	TotalDownloads int64          `json:"total_downloads"`
	VariantStats   []VariantStats `json:"variant_stats"`
	TopCTA         string         `json:"top_cta,omitempty"`
	TopCTACTR      float64        `json:"top_cta_ctr,omitempty"`
}

func (s *Service) TrackEvent(event Event) error {
	if err := ValidateEvent(event); err != nil {
		return err
	}
	eventID := event.EventID
	if eventID == "" {
		eventID = GenerateEventIDAt(event, s.clock.Now())
	}

	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM metrics_events WHERE event_data->>'event_id' = $1)`, eventID).Scan(&exists)
	if err != nil {
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
	if _, err = s.db.Exec(`INSERT INTO metrics_events (variant_slug, event_type, event_data, session_id, visitor_id) VALUES ($1, $2, $3, $4, $5)`, event.VariantSlug, event.EventType, eventDataJSON, event.SessionID, event.VisitorID); err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}
	return nil
}

func ValidateEvent(event Event) error {
	if strings.TrimSpace(event.EventType) == "" {
		return &ValidationError{Field: "event_type", Reason: "event_type is required"}
	}
	if _, ok := validEventTypes[event.EventType]; !ok {
		return &ValidationError{Field: "event_type", Reason: fmt.Sprintf("invalid event_type: %s", event.EventType)}
	}
	if strings.TrimSpace(event.VariantSlug) == "" {
		return &ValidationError{Field: "variant_slug", Reason: "variant_slug is required"}
	}
	if strings.TrimSpace(event.SessionID) == "" {
		return &ValidationError{Field: "session_id", Reason: "session_id is required"}
	}
	return nil
}

func GenerateEventIDAt(event Event, now time.Time) string {
	input := fmt.Sprintf("%s:%s:%s:%d", event.SessionID, event.EventType, event.VariantSlug, now.Unix())
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:16])
}

func (s *Service) GetVariantStats(startDate, endDate time.Time, variantSlug string) ([]VariantStats, error) {
	query := `SELECT variant_slug,
		COALESCE(SUM(CASE WHEN event_type = 'page_view' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN event_type = 'click' AND event_data->>'element_type' = 'cta' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN event_type = 'conversion' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN event_type = 'download' THEN 1 ELSE 0 END), 0)
		FROM metrics_events WHERE created_at >= $1 AND created_at <= $2`
	args := []interface{}{startDate, endDate}
	if variantSlug != "" {
		query += " AND variant_slug = $3"
		args = append(args, variantSlug)
	}
	query += " GROUP BY variant_slug ORDER BY variant_slug"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query variant stats: %w", err)
	}
	defer rows.Close()
	var stats []VariantStats
	for rows.Next() {
		var stat VariantStats
		if err := rows.Scan(&stat.VariantSlug, &stat.Views, &stat.CTAClicks, &stat.Conversions, &stat.Downloads); err != nil {
			return nil, fmt.Errorf("failed to scan variant stats: %w", err)
		}
		stat.VariantName = stat.VariantSlug
		if stat.Views > 0 {
			stat.ConversionRate = float64(stat.Conversions) / float64(stat.Views) * 100
		}
		stats = append(stats, stat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate variant stats: %w", err)
	}
	return stats, nil
}

func (s *Service) GetAnalyticsSummary(startDate, endDate time.Time) (*AnalyticsSummary, error) {
	stats, err := s.GetVariantStats(startDate, endDate, "")
	if err != nil {
		return nil, err
	}
	var totalVisitors int64
	if err = s.db.QueryRow(`SELECT COUNT(DISTINCT session_id) FROM metrics_events WHERE event_type = 'page_view' AND created_at >= $1 AND created_at <= $2`, startDate, endDate).Scan(&totalVisitors); err != nil {
		return nil, fmt.Errorf("failed to count visitors: %w", err)
	}
	var totalDownloads int64
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM metrics_events WHERE event_type = 'download' AND created_at >= $1 AND created_at <= $2`, startDate, endDate).Scan(&totalDownloads); err != nil {
		return nil, fmt.Errorf("failed to count downloads: %w", err)
	}
	var topCTA string
	var topCTAClicks, topCTAViews int64
	err = s.db.QueryRow(`SELECT m.event_data->>'element_id', COUNT(*),
		(SELECT COUNT(DISTINCT session_id) FROM metrics_events WHERE event_type = 'page_view' AND created_at >= $1 AND created_at <= $2)
		FROM metrics_events m WHERE m.event_type = 'click' AND m.event_data->>'element_type' = 'cta' AND m.created_at >= $1 AND m.created_at <= $2
		GROUP BY m.event_data->>'element_id' ORDER BY COUNT(*) DESC LIMIT 1`, startDate, endDate).Scan(&topCTA, &topCTAClicks, &topCTAViews)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query top CTA: %w", err)
	}
	var topCTACTR float64
	if topCTAViews > 0 {
		topCTACTR = float64(topCTAClicks) / float64(topCTAViews) * 100
	}
	return &AnalyticsSummary{TotalVisitors: totalVisitors, TotalDownloads: totalDownloads, VariantStats: stats, TopCTA: topCTA, TopCTACTR: topCTACTR}, nil
}
