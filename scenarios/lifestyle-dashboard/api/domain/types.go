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
	TotalEvents     int             `json:"total_events"`
	ActiveDomains   int             `json:"active_domains"`
	EventsByDomain  []DomainCount   `json:"events_by_domain"`
	LastEventAt     string          `json:"last_event_at"`
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
