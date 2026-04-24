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

	// tmux re-attach counters (readLoop resilience)
	ReattachAttempts  atomic.Int64
	ReattachSuccesses atomic.Int64
	ReattachFailures  atomic.Int64

	// Recovery counters (startup session restoration)
	RecoveryRecovered       atomic.Int64
	RecoveryOrphanedMeta    atomic.Int64
	RecoveryOrphanedTmux    atomic.Int64
	RecoveryAttachRetries   atomic.Int64
	RecoveryPreservedForNow atomic.Int64 // sessions kept for future recovery

	// AI generation counter
	AIGenerations atomic.Int64
	// AI suggestion counter
	AISuggestions atomic.Int64

	// StdinBeforeReadyTotal counts stdin messages arriving before the server
	// has emitted session_ready for the connection. Expected to be 0 in
	// steady state; any increment indicates a sequencing regression.
	StdinBeforeReadyTotal atomic.Int64

	// VoiceSkipVerificationTotal counts /voice/transcribe requests that
	// explicitly bypassed speaker verification via the
	// `skip_speaker_verification=true` query parameter. User-initiated
	// "Transcribe anyway" retries drive this counter; non-zero values are
	// expected during normal operation when users override false rejections.
	// DOC: docs/plans/stt-voice-filter-retry-implementation-plan.md §9.4
	VoiceSkipVerificationTotal atomic.Int64

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
	Sessions                   SessionMetrics    `json:"sessions"`
	Connections                ConnectionMetrics `json:"connections"`
	Messages                   MessageMetrics    `json:"messages"`
	Reattach                   ReattachMetrics   `json:"reattach"`
	Recovery                   RecoveryMetrics   `json:"recovery"`
	AIGenerations              int64             `json:"ai_generations"`
	AISuggestions              int64             `json:"ai_suggestions"`
	StdinBeforeReadyTotal      int64             `json:"stdin_before_ready_total"`
	VoiceSkipVerificationTotal int64             `json:"voice_skip_verification_total"`
	Uptime                     string            `json:"uptime"`
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

// ReattachMetrics tracks tmux re-attach resilience during normal operation.
type ReattachMetrics struct {
	Attempts  int64 `json:"attempts"`
	Successes int64 `json:"successes"`
	Failures  int64 `json:"failures"`
}

// RecoveryMetrics tracks session recovery at server startup.
type RecoveryMetrics struct {
	Recovered       int64 `json:"recovered"`
	OrphanedMeta    int64 `json:"orphaned_metadata"`
	OrphanedTmux    int64 `json:"orphaned_tmux"`
	AttachRetries   int64 `json:"attach_retries"`
	PreservedForNow int64 `json:"preserved_for_future_recovery"`
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
		Reattach: ReattachMetrics{
			Attempts:  m.ReattachAttempts.Load(),
			Successes: m.ReattachSuccesses.Load(),
			Failures:  m.ReattachFailures.Load(),
		},
		Recovery: RecoveryMetrics{
			Recovered:       m.RecoveryRecovered.Load(),
			OrphanedMeta:    m.RecoveryOrphanedMeta.Load(),
			OrphanedTmux:    m.RecoveryOrphanedTmux.Load(),
			AttachRetries:   m.RecoveryAttachRetries.Load(),
			PreservedForNow: m.RecoveryPreservedForNow.Load(),
		},
		AIGenerations:              m.AIGenerations.Load(),
		AISuggestions:              m.AISuggestions.Load(),
		StdinBeforeReadyTotal:      m.StdinBeforeReadyTotal.Load(),
		VoiceSkipVerificationTotal: m.VoiceSkipVerificationTotal.Load(),
		Uptime:                     time.Since(m.StartTime).Truncate(time.Second).String(),
	}
}

// handleMetrics returns a JSON snapshot of all operational metrics.
// GET /api/v1/metrics
// [REQ:P1-004b] Operational Metrics Collection
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.metrics.Snapshot())
}
