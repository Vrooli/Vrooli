package main

import (
	"net/http"
	"sync/atomic"
	"time"
)

// DOC: docs/concepts/ARCHITECTURE.md#observability
// [REQ:P1-004b] Operational Metrics Collection

// Metrics tracks operational counters for the web-console API.
// All fields use atomic operations for lock-free concurrent access.
type Metrics struct {
	// Session lifecycle counters
	SessionsCreated atomic.Int64
	SessionsDeleted atomic.Int64
	ActiveSessions  atomic.Int64

	// WebSocket connection counters
	ConnectionsTotal  atomic.Int64
	ActiveConnections atomic.Int64

	// WebSocket message counters
	WSMessagesSent     atomic.Int64
	WSMessagesReceived atomic.Int64

	// Resize operations
	ResizeCount atomic.Int64

	// AI generation counter
	AIGenerations atomic.Int64

	// StartTime records when the server started for uptime calculation.
	StartTime time.Time
}

// NewMetrics creates a new metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		StartTime: time.Now(),
	}
}

// MetricsResponse is the JSON shape returned by the /api/v1/metrics endpoint.
type MetricsResponse struct {
	Sessions      SessionMetrics    `json:"sessions"`
	Connections   ConnectionMetrics `json:"connections"`
	Messages      MessageMetrics    `json:"messages"`
	AIGenerations int64             `json:"ai_generations"`
	Uptime        string            `json:"uptime"`
}

// SessionMetrics tracks session lifecycle counts.
type SessionMetrics struct {
	Created int64 `json:"created"`
	Deleted int64 `json:"deleted"`
	Active  int64 `json:"active"`
	Resizes int64 `json:"resizes"`
}

// ConnectionMetrics tracks WebSocket connection counts.
type ConnectionMetrics struct {
	Total  int64 `json:"total"`
	Active int64 `json:"active"`
}

// MessageMetrics tracks WebSocket message throughput.
type MessageMetrics struct {
	Sent     int64 `json:"sent"`
	Received int64 `json:"received"`
}

// Snapshot returns a point-in-time view of all metrics.
func (m *Metrics) Snapshot() MetricsResponse {
	return MetricsResponse{
		Sessions: SessionMetrics{
			Created: m.SessionsCreated.Load(),
			Deleted: m.SessionsDeleted.Load(),
			Active:  m.ActiveSessions.Load(),
			Resizes: m.ResizeCount.Load(),
		},
		Connections: ConnectionMetrics{
			Total:  m.ConnectionsTotal.Load(),
			Active: m.ActiveConnections.Load(),
		},
		Messages: MessageMetrics{
			Sent:     m.WSMessagesSent.Load(),
			Received: m.WSMessagesReceived.Load(),
		},
		AIGenerations: m.AIGenerations.Load(),
		Uptime:        time.Since(m.StartTime).Truncate(time.Second).String(),
	}
}

// handleMetrics returns a JSON snapshot of all operational metrics.
// GET /api/v1/metrics
// [REQ:P1-004b] Operational Metrics Collection
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.metrics.Snapshot())
}
