// Package metrics owns the in-process operational counter struct for the
// web-console API. Production wires a single *Metrics into server.Deps;
// observers (readLoop, session lifecycle, AI/voice paths) increment counters
// in place. The HTTP/Connect surface lives in handlers/metrics, which depends
// on this package through a narrow Service interface — see
// handlers/metrics/module.go.
//
// DOC: docs/internal/SEAMS.md#observability-surface
// [REQ:P1-004b] Operational Metrics Collection
package metrics

import (
	"sync/atomic"
	"time"
)

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

	// VoiceSkipVerificationTotal counts /voice/transcribe requests that
	// explicitly bypassed speaker verification via the
	// `skip_speaker_verification=true` query parameter. User-initiated
	// "Transcribe anyway" retries drive this counter; non-zero values are
	// expected during normal operation when users override false rejections.
	VoiceSkipVerificationTotal atomic.Int64

	// StartTime records when the server started for uptime calculation.
	StartTime time.Time
}

// New creates a new metrics collector.
func New() *Metrics {
	return &Metrics{
		StartTime: time.Now(),
	}
}

// Response is the JSON-shaped point-in-time snapshot returned by Snapshot.
// Mirrors the proto MetricsService.Get response, but lives here so server-side
// adapters can produce it without crossing the package main boundary.
type Response struct {
	Sessions                   SessionMetrics    `json:"sessions"`
	Connections                ConnectionMetrics `json:"connections"`
	Messages                   MessageMetrics    `json:"messages"`
	Reattach                   ReattachMetrics   `json:"reattach"`
	Recovery                   RecoveryMetrics   `json:"recovery"`
	AIGenerations              int64             `json:"ai_generations"`
	AISuggestions              int64             `json:"ai_suggestions"`
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
func (m *Metrics) Snapshot() Response {
	return Response{
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
		VoiceSkipVerificationTotal: m.VoiceSkipVerificationTotal.Load(),
		Uptime:                     time.Since(m.StartTime).Truncate(time.Second).String(),
	}
}
