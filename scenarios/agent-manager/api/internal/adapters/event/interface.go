// Package event provides the event streaming and storage interface.
//
// This package defines the SEAM for event handling. Events are the
// append-only record of everything that happens during agent execution.
// The interface allows for different storage backends and streaming
// mechanisms without changing the orchestration code.
package event

import (
	"context"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Store Interface - The primary seam for event persistence
// -----------------------------------------------------------------------------

// Store provides persistence for run events.
type Store interface {
	// Append adds events to a run's event stream.
	// Events are assigned sequence numbers automatically.
	Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error

	// Get retrieves events for a run with optional filtering.
	Get(ctx context.Context, runID uuid.UUID, opts GetOptions) ([]*domain.RunEvent, error)

	// Stream returns a channel that receives events in real-time.
	// The channel is closed when the context is cancelled or the run ends.
	Stream(ctx context.Context, runID uuid.UUID, opts StreamOptions) (<-chan *domain.RunEvent, error)

	// Count returns the number of events for a run.
	Count(ctx context.Context, runID uuid.UUID) (int64, error)

	// Delete removes all events for a run (for cleanup).
	Delete(ctx context.Context, runID uuid.UUID) error
}

// GetOptions specifies criteria for retrieving events.
type GetOptions struct {
	// AfterSequence returns events with sequence > this value.
	AfterSequence int64

	// EventTypes filters to specific event types.
	EventTypes []domain.RunEventType

	// Limit caps the number of events returned.
	Limit int

	// Offset skips this many events.
	Offset int

	// Since returns events after this timestamp.
	Since *time.Time
}

// StreamOptions configures real-time event streaming.
type StreamOptions struct {
	// FromSequence starts streaming from this sequence number.
	// Use 0 to start from the beginning.
	FromSequence int64

	// EventTypes filters to specific event types.
	EventTypes []domain.RunEventType

	// BufferSize is the channel buffer size.
	BufferSize int
}

// -----------------------------------------------------------------------------
// Publisher Interface - For broadcasting events
// -----------------------------------------------------------------------------

// Publisher broadcasts events to multiple subscribers.
type Publisher interface {
	// Publish sends an event to all subscribers for a run.
	Publish(ctx context.Context, runID uuid.UUID, event *domain.RunEvent) error

	// Subscribe creates a subscription for a run's events.
	Subscribe(ctx context.Context, runID uuid.UUID) (Subscription, error)

	// Unsubscribe removes a subscription.
	Unsubscribe(subID uuid.UUID) error
}

// Subscription represents an active event subscription.
type Subscription interface {
	// ID returns the subscription identifier.
	ID() uuid.UUID

	// Events returns the channel receiving events.
	Events() <-chan *domain.RunEvent

	// Close terminates the subscription.
	Close() error
}

// -----------------------------------------------------------------------------
// Collector Interface - For capturing events from runners
// -----------------------------------------------------------------------------

// Collector aggregates events from an agent execution.
// This is passed to runners to capture events during execution.
type Collector interface {
	// Log emits a log event.
	Log(level, message string)

	// Message emits a message event (agent conversation).
	Message(role, content string)

	// ToolCall emits a tool invocation event.
	ToolCall(toolName string, input map[string]interface{})

	// ToolResult emits a tool completion event.
	ToolResult(toolName string, output string, err error)

	// StatusChange emits a status transition event.
	StatusChange(oldStatus, newStatus string)

	// Metric emits a metric event.
	Metric(name string, value float64)

	// Artifact emits an artifact reference event.
	Artifact(artifactType, path string)

	// Error emits an error event.
	Error(code, message string)

	// Flush ensures all buffered events are persisted.
	Flush(ctx context.Context) error
}

// -----------------------------------------------------------------------------
// Factory Functions
// -----------------------------------------------------------------------------

// CollectorFactory creates event collectors for runs.
type CollectorFactory interface {
	// Create creates a new collector for a run.
	Create(runID uuid.UUID, store Store) Collector
}
