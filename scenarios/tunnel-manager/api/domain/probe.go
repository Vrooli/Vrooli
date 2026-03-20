package domain

import "time"

// ProbeResult holds the outcome of a single liveness probe.
type ProbeResult struct {
	RouteID    int    `json:"route_id"`
	Subdomain  string `json:"subdomain"`
	ProbeType  string `json:"probe_type"` // "internal" or "external"
	Status     string `json:"status"`     // "up", "down", "timeout", "error"
	LatencyMs  int    `json:"latency_ms"`
	StatusCode int    `json:"status_code,omitempty"`
	ErrorMsg   string `json:"error_msg,omitempty"`
}

// RouteClassification describes the combined internal+external probe status for a route.
type RouteClassification struct {
	RouteID    int    `json:"route_id"`
	Subdomain  string `json:"subdomain"`
	Status     string `json:"status"`     // "up", "tunnel-issue", "scenario-down", "unknown"
	Internal   string `json:"internal"`   // probe status
	External   string `json:"external"`   // probe status
	Assessment string `json:"assessment"` // human-readable description
}

// StoredProbeResult is a probe result read from the database. [REQ:OBS-002]
type StoredProbeResult struct {
	ID         int       `json:"id"`
	RouteID    int       `json:"route_id"`
	ProbeType  string    `json:"probe_type"`
	Status     string    `json:"status"`
	LatencyMs  *int      `json:"latency_ms"`
	StatusCode *int      `json:"status_code,omitempty"`
	ErrorMsg   *string   `json:"error_msg,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
