// DOC: docs/concepts/ARCHITECTURE.md#Key-Domain-Concepts
// DOC: PRD.md#OT-P0-001
// DOC: PRD.md#OT-P0-002
//
// Package domain contains the core business entities for the lifestyle dashboard.
// These types represent the shared event schema (P0-001) and domain registration (P0-002).
package domain

import "encoding/json"

// Event represents a cross-domain event with JSON payload.
// This is the core envelope for all lifestyle data points (P0-001).
type Event struct {
	ID             string          `json:"id"`
	Timestamp      string          `json:"timestamp"`
	Domain         string          `json:"domain"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	IsIntervention bool            `json:"is_intervention"`
	HypothesisID   *string         `json:"hypothesis_id,omitempty"`
	CreatedAt      string          `json:"created_at"`
}

// CreateEventRequest is the request body for creating an event.
type CreateEventRequest struct {
	Domain         string          `json:"domain"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	Timestamp      *string         `json:"timestamp,omitempty"`
	IsIntervention bool            `json:"is_intervention"`
	HypothesisID   *string         `json:"hypothesis_id,omitempty"`
}

// Domain represents a registered domain scenario (P0-002).
// Domains are health/wellness data sources that integrate with the dashboard.
type Domain struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Status       string   `json:"status"` // active, inactive, unhealthy
	HealthURL    string   `json:"health_url,omitempty"`
	LastHealthAt *string  `json:"last_health_at,omitempty"`
	RegisteredAt string   `json:"registered_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// RegisterDomainRequest is the request body for registering a domain.
type RegisterDomainRequest struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	HealthURL    string   `json:"health_url,omitempty"`
}

// EventsResponse wraps a list of events for API responses.
type EventsResponse struct {
	Events []Event `json:"events"`
	Count  int     `json:"count"`
}

// DomainsResponse wraps a list of domains for API responses.
type DomainsResponse struct {
	Domains []Domain `json:"domains"`
	Count   int      `json:"count"`
}

// TimelineEntry represents a single data point in the timeline view.
type TimelineEntry struct {
	Day    string `json:"day"`
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

// TimelineResponse wraps timeline data for API responses.
type TimelineResponse struct {
	Timeline []TimelineEntry `json:"timeline"`
	Days     string          `json:"days"`
}

// SummaryResponse contains aggregated statistics across all domains.
type SummaryResponse struct {
	TotalEvents    int           `json:"total_events"`
	ActiveDomains  int           `json:"active_domains"`
	EventsByDomain []DomainCount `json:"events_by_domain"`
	LastEventAt    string        `json:"last_event_at"`
}

// DomainCount represents event count for a specific domain.
type DomainCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

// HealthCheckResponse represents the result of a domain health check.
type HealthCheckResponse struct {
	Domain    string `json:"domain"`
	Status    string `json:"status"`
	LastCheck string `json:"last_check"`
	Message   string `json:"message,omitempty"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// LifestyleScore represents the daily composite score (P0-004, P1-003).
// The score is a 0-100 value combining activity across all active domains.
// [REQ:LD-UI-SCORE] Score structure for dashboard display.
type LifestyleScore struct {
	Score               int           `json:"score"`                 // 0-100 composite score
	Date                string        `json:"date"`                  // ISO date (YYYY-MM-DD)
	DomainScores        []DomainScore `json:"domain_scores"`         // Per-domain breakdown
	Trend               string        `json:"trend"`                 // "up", "down", "stable"
	ChangeFromYesterday int           `json:"change_from_yesterday"` // Delta from previous day
	DataQuality         string        `json:"data_quality"`          // "good", "limited", "insufficient"
	Message             string        `json:"message"`               // Human-readable summary
}

// DomainScore represents a single domain's contribution to the lifestyle score.
type DomainScore struct {
	Domain      string  `json:"domain"`       // Domain name
	DisplayName string  `json:"display_name"` // Human-readable name
	Score       int     `json:"score"`        // 0-100 domain score
	Weight      float64 `json:"weight"`       // Weight in composite (0-1)
	EventCount  int     `json:"event_count"`  // Events in scoring window
}

// ScoreHistoryEntry represents a historical score data point.
type ScoreHistoryEntry struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
}

// ScoreResponse wraps the lifestyle score for API responses.
type ScoreResponse struct {
	Current LifestyleScore      `json:"current"`
	History []ScoreHistoryEntry `json:"history"`
}

// =============================================================================
// Storage Management Types (P0-006)
// =============================================================================

// StorageInfo provides database storage information for the settings page.
// [REQ:LD-UI-STORAGE] Storage overview data.
type StorageInfo struct {
	DatabaseSizeBytes int64               `json:"database_size_bytes"`
	TotalEvents       int                 `json:"total_events"`
	TotalDomains      int                 `json:"total_domains"`
	EventsByDomain    []DomainStorageInfo `json:"events_by_domain"`
	OldestEvent       string              `json:"oldest_event,omitempty"`
	NewestEvent       string              `json:"newest_event,omitempty"`
}

// DomainStorageInfo represents storage usage for a specific domain.
type DomainStorageInfo struct {
	Domain      string `json:"domain"`
	DisplayName string `json:"display_name"`
	EventCount  int    `json:"event_count"`
}

// CleanupRequest specifies what data to clean.
// [REQ:LD-UI-STORAGE] Data cleanup request.
type CleanupRequest struct {
	Domains []string `json:"domains,omitempty"` // Empty = all domains
	Before  string   `json:"before,omitempty"`  // ISO timestamp - delete events before this
}

// CleanupResponse reports the result of a cleanup operation.
type CleanupResponse struct {
	DeletedEvents  int      `json:"deleted_events"`
	DomainsCleared []string `json:"domains_cleared"`
	Message        string   `json:"message"`
}

// =============================================================================
// Daily Brief System Types (P0-005)
// =============================================================================

// Brief represents a morning or evening brief.
// [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING] Brief structure.
type Brief struct {
	Type        string         `json:"type"`                  // "morning" or "evening"
	GeneratedAt string         `json:"generated_at"`          // ISO timestamp
	Date        string         `json:"date"`                  // Target date (YYYY-MM-DD)
	Summary     string         `json:"summary"`               // Human-readable summary
	Sections    []BriefSection `json:"sections"`              // Consolidated domain content
	Score       *int           `json:"score,omitempty"`       // Current lifestyle score (if available)
	ScoreTrend  string         `json:"score_trend,omitempty"` // "up", "down", "stable"
}

// BriefSection represents a domain's contribution to the brief.
// [REQ:LD-BRIEF-CONSOLIDATE] Cross-domain consolidation.
type BriefSection struct {
	Domain      string   `json:"domain"`
	DisplayName string   `json:"display_name"`
	Priority    int      `json:"priority"`    // 1=high, 2=medium, 3=low
	Items       []string `json:"items"`       // Bullet points for this domain
	EventCount  int      `json:"event_count"` // Events in the period
}

// BriefConfig holds configuration for brief generation timing.
type BriefConfig struct {
	MorningHour int `json:"morning_hour"` // Default: 7
	EveningHour int `json:"evening_hour"` // Default: 21
}

// BriefResponse wraps a brief for API responses.
type BriefResponse struct {
	Brief  Brief       `json:"brief"`
	Config BriefConfig `json:"config"`
}

// =============================================================================
// Score Configuration Types (P1-003)
// =============================================================================

// DomainWeightConfig holds user-configurable weight for a domain.
// [REQ:LD-SCORE-CALC] Configurable domain weights for lifestyle score.
type DomainWeightConfig struct {
	Domain      string  `json:"domain"`       // Domain name
	DisplayName string  `json:"display_name"` // Human-readable name
	Weight      string  `json:"weight"`       // "high", "medium", "low", "none"
	Multiplier  float64 `json:"multiplier"`   // Numeric weight (high=3, medium=2, low=1, none=0)
}

// ScoreConfigResponse wraps score configuration for API responses.
// [REQ:LD-SCORE-CALC] Score configuration endpoint.
type ScoreConfigResponse struct {
	Weights       []DomainWeightConfig `json:"weights"`
	DefaultWeight string               `json:"default_weight"` // Applied to new domains
}

// UpdateWeightRequest is the request body for updating a domain's weight.
type UpdateWeightRequest struct {
	Weight string `json:"weight"` // "high", "medium", "low", "none"
}

// WeightPresets defines recommended weight configurations.
var WeightPresets = map[string]string{
	"sleep":         "high",   // Sleep is foundational
	"exercise":      "high",   // Physical activity is core
	"diet":          "medium", // Nutrition is important
	"nootropics":    "medium", // Supplements matter
	"socialization": "low",    // Social is tracked but less weighted
}

// WeightMultipliers maps weight labels to numeric multipliers.
var WeightMultipliers = map[string]float64{
	"high":   3.0,
	"medium": 2.0,
	"low":    1.0,
	"none":   0.0,
}

// =============================================================================
// Weekly Digest Types (P1-002)
// =============================================================================

// WeeklyDigest represents the "What Changed?" weekly summary.
// Generated every Sunday 6pm, comparing current week to rolling 4-week baseline.
// [REQ:LD-DIGEST-WEEKLY] Weekly digest structure.
type WeeklyDigest struct {
	GeneratedAt   string               `json:"generated_at"`    // ISO timestamp
	WeekStartDate string               `json:"week_start_date"` // Monday of the summarized week
	WeekEndDate   string               `json:"week_end_date"`   // Sunday of the summarized week
	Summary       string               `json:"summary"`         // Human-readable overview
	ScoreTrend    DigestScoreTrend     `json:"score_trend"`     // Lifestyle score comparison
	DomainChanges []DigestDomainChange `json:"domain_changes"`  // Per-domain deltas
	Correlations  []DigestCorrelation  `json:"correlations"`    // New correlations discovered
	Highlights    []string             `json:"highlights"`      // Notable achievements or concerns
	NextWeekFocus []string             `json:"next_week_focus"` // Recommendations for next week
}

// DigestScoreTrend compares lifestyle score between current week and baseline.
type DigestScoreTrend struct {
	CurrentWeekAvg   float64 `json:"current_week_avg"`  // Average score this week
	BaselineAvg      float64 `json:"baseline_avg"`      // 4-week rolling baseline average
	PercentChange    float64 `json:"percent_change"`    // Percentage change from baseline
	Direction        string  `json:"direction"`         // "up", "down", "stable"
	ConsecutiveWeeks int     `json:"consecutive_weeks"` // Weeks in current direction
	Message          string  `json:"message"`           // Human-readable trend description
}

// DigestDomainChange summarizes a domain's activity change from baseline.
type DigestDomainChange struct {
	Domain            string  `json:"domain"`
	DisplayName       string  `json:"display_name"`
	CurrentWeekEvents int     `json:"current_week_events"` // Events this week
	BaselineAvgEvents float64 `json:"baseline_avg_events"` // Average events in baseline
	PercentChange     float64 `json:"percent_change"`      // Change from baseline
	Direction         string  `json:"direction"`           // "up", "down", "stable"
	Notable           bool    `json:"notable"`             // True if change > 20%
	Message           string  `json:"message"`             // Human-readable summary
}

// DigestCorrelation represents a newly discovered or confirmed correlation.
type DigestCorrelation struct {
	Domain1     string  `json:"domain1"`
	Domain2     string  `json:"domain2"`
	EventType1  string  `json:"event_type1"`
	EventType2  string  `json:"event_type2"`
	Correlation float64 `json:"correlation"`       // Pearson correlation coefficient
	Status      string  `json:"status"`            // "new", "confirmed", "strengthened"
	DataPoints  int     `json:"data_points"`       // Sample size
	Message     string  `json:"message,omitempty"` // Human-readable description
}

// WeeklyDigestResponse wraps a weekly digest for API responses.
type WeeklyDigestResponse struct {
	Digest WeeklyDigest `json:"digest"`
}
