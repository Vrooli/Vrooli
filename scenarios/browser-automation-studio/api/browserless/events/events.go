package events

import (
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// EventType represents the canonical event identifiers emitted over the WebSocket stream.
type EventType string

const (
	EventExecutionStarted   EventType = "execution.started"
	EventExecutionProgress  EventType = "execution.progress"
	EventExecutionCompleted EventType = "execution.completed"
	EventExecutionFailed    EventType = "execution.failed"
	EventExecutionCancelled EventType = "execution.cancelled"
	EventStepStarted        EventType = "step.started"
	EventStepCompleted      EventType = "step.completed"
	EventStepFailed         EventType = "step.failed"
	EventStepScreenshot     EventType = "step.screenshot"
	EventStepLog            EventType = "step.log"
	EventStepTelemetry      EventType = "step.telemetry"
)

// Event is the structured payload broadcast to clients.
type Event struct {
	Type        EventType      `json:"type"`
	ExecutionID uuid.UUID      `json:"execution_id"`
	WorkflowID  uuid.UUID      `json:"workflow_id"`
	StepIndex   *int           `json:"step_index,omitempty"`
	StepNodeID  string         `json:"step_node_id,omitempty"`
	StepType    string         `json:"step_type,omitempty"`
	Status      string         `json:"status,omitempty"`
	Progress    *int           `json:"progress,omitempty"`
	Message     string         `json:"message,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
}

// NewEvent constructs a generic execution event.
func NewEvent(eventType EventType, executionID, workflowID uuid.UUID, opts ...Option) Event {
	cfg := &options{timestamp: time.Now().UTC()}
	for _, opt := range opts {
		opt(cfg)
	}

	return Event{
		Type:        eventType,
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		StepIndex:   cfg.stepIndex,
		StepNodeID:  cfg.stepNodeID,
		StepType:    cfg.stepType,
		Status:      cfg.status,
		Progress:    cfg.progress,
		Message:     cfg.message,
		Payload:     cfg.payload,
		Timestamp:   cfg.timestamp,
	}
}

// Option configures optional Event fields.
type Option func(*options)

type options struct {
	stepIndex  *int
	stepNodeID string
	stepType   string
	status     string
	progress   *int
	message    string
	payload    map[string]any
	timestamp  time.Time
}

// WithStep details adds step metadata to an event.
func WithStep(index int, nodeID, stepType string) Option {
	return func(o *options) {
		o.stepIndex = &index
		o.stepNodeID = nodeID
		o.stepType = stepType
	}
}

// WithStatus sets a status string on the event.
func WithStatus(status string) Option {
	return func(o *options) {
		o.status = status
	}
}

// WithProgress sets the execution progress percentage.
func WithProgress(progress int) Option {
	return func(o *options) {
		o.progress = &progress
	}
}

// WithMessage sets a human-readable message on the event.
func WithMessage(msg string) Option {
	return func(o *options) {
		o.message = msg
	}
}

// WithPayload attaches arbitrary metadata to the event.
func WithPayload(payload map[string]any) Option {
	return func(o *options) {
		o.payload = payload
	}
}

// WithTimestamp overrides the default timestamp (mostly for testing).
func WithTimestamp(ts time.Time) Option {
	return func(o *options) {
		o.timestamp = ts
	}
}

// Emitter bridges the new event contract with the existing WebSocket hub.
type Emitter struct {
	hub *wsHub.Hub
	log *logrus.Logger
}

// NewEmitter returns an emitter that broadcasts events through the hub.
func NewEmitter(hub *wsHub.Hub, log *logrus.Logger) *Emitter {
	if hub == nil {
		return nil
	}
	return &Emitter{hub: hub, log: log}
}

// Emit broadcasts a structured event. It falls back to the legacy ExecutionUpdate envelope for compatibility.
func (e *Emitter) Emit(event Event) {
	if e == nil || e.hub == nil {
		return
	}

	update := wsHub.ExecutionUpdate{
		Type:        "event",
		ExecutionID: event.ExecutionID,
		Status:      event.Status,
		Progress:    valueOrZero(event.Progress),
		CurrentStep: event.StepNodeID,
		Message:     event.Message,
		Data:        event,
	}

	e.hub.BroadcastUpdate(update)
}

func valueOrZero(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}
