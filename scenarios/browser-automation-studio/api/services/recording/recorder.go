// Package recording provides the unified recording service.
//
// DOC: docs/architecture/recording.md
package recording

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/domain"
)

// ActionRecorder defines the interface for unified action recording.
// This interface consolidates the dual-write pattern (persistence + WebSocket broadcast)
// into a single atomic operation with full observability.
//
// DOC: docs/architecture/recording.md#unified-recording
type ActionRecorder interface {
	// RecordActionUnified records an action with unified persistence and broadcast.
	// It handles both database persistence and WebSocket broadcast atomically,
	// returning comprehensive results for observability.
	RecordActionUnified(ctx context.Context, req RecordActionRequest) (*ActionRecordResult, error)

	// RecordPageEventUnified records a page event with unified persistence and broadcast.
	RecordPageEventUnified(ctx context.Context, req RecordPageEventRequest) (*ActionRecordResult, error)
}

// RecordActionRequest contains all parameters needed to record an action.
type RecordActionRequest struct {
	// SessionID identifies the recording session.
	SessionID string

	// Action is the recorded action from the driver.
	Action *driver.RecordedAction

	// PageID identifies which page the action occurred on.
	PageID uuid.UUID

	// Source indicates how the action was initiated (auto, manual, ai).
	Source ActionSource

	// CorrelationID enables tracing the action through the pipeline.
	// If empty, one will be generated.
	CorrelationID string
}

// RecordPageEventRequest contains parameters for recording a page event.
type RecordPageEventRequest struct {
	// SessionID identifies the recording session.
	SessionID string

	// Event is the page lifecycle event.
	Event *domain.PageEvent

	// CorrelationID enables tracing the event through the pipeline.
	CorrelationID string
}

// ActionRecordResult contains the outcome of a record operation.
// This provides full observability into what happened during recording.
type ActionRecordResult struct {
	// ActionID is the ID assigned to the recorded action/event.
	ActionID uuid.UUID

	// CorrelationID links this result to its request for tracing.
	CorrelationID string

	// SequenceNum is the sequence number in the session timeline.
	SequenceNum int

	// Persisted indicates whether the action was saved to the database.
	Persisted bool

	// BroadcastSent indicates whether at least one client received the message.
	BroadcastSent bool

	// SubscriberCount is the number of clients subscribed to this session.
	SubscriberCount int

	// SentCount is the number of clients that received the broadcast.
	SentCount int

	// DroppedCount is the number of clients whose messages were dropped (buffer full).
	DroppedCount int

	// Errors contains any errors that occurred during recording.
	// The operation may partially succeed (e.g., persisted but not broadcast).
	Errors []ActionRecordError
}

// ActionRecordError represents an error at a specific stage of recording.
type ActionRecordError struct {
	// Stage identifies where the error occurred ("persistence", "broadcast", "validation").
	Stage string

	// Err is the underlying error.
	Err error

	// Message provides additional context.
	Message string
}

// Error implements the error interface.
func (e ActionRecordError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Stage, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Stage, e.Message)
}

// HasErrors returns true if any errors occurred during recording.
func (r *ActionRecordResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// WasFullySuccessful returns true if the action was both persisted and broadcast.
func (r *ActionRecordResult) WasFullySuccessful() bool {
	return r.Persisted && r.BroadcastSent && !r.HasErrors()
}

// CorrelationIDGenerator creates correlation IDs for tracing.
type CorrelationIDGenerator struct {
	counter uint64
}

// Generate creates a new correlation ID for tracing.
// It uses a monotonic counter combined with session ID for uniqueness.
func (g *CorrelationIDGenerator) Generate(sessionID string) string {
	g.counter++
	shortSession := sessionID
	if len(shortSession) > 8 {
		shortSession = shortSession[:8]
	}
	return fmt.Sprintf("rec-%s-%d", shortSession, g.counter)
}

// GenerateCorrelationID creates a new correlation ID for tracing.
// Deprecated: Use CorrelationIDGenerator for deterministic tests.
func GenerateCorrelationID(sessionID string) string {
	// Use short session ID prefix + timestamp for uniqueness and traceability
	shortSession := sessionID
	if len(shortSession) > 8 {
		shortSession = shortSession[:8]
	}
	return fmt.Sprintf("rec-%s-%d", shortSession, time.Now().UnixNano())
}
