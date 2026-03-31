// Package eventlog provides an append-only event log for tracking state changes
// across all swarm-manager entities (backlog items, initiatives, executions, queue).
//
// Events are stored in SQLite and consumed by the stats engine to compute
// throughput, timing, scope, blocking, and agent efficiency metrics.
package eventlog

import (
	"encoding/json"
	"time"
)

// EntityType identifies the kind of entity an event relates to.
type EntityType string

const (
	EntityBacklogItem EntityType = "backlog_item"
	EntityInitiative  EntityType = "initiative"
	EntityExecution   EntityType = "execution"
	EntityQueue       EntityType = "queue"
)

// EventType identifies what happened to an entity.
type EventType string

// Backlog item events.
const (
	EventBacklogCreated           EventType = "backlog.created"
	EventBacklogStatusChanged     EventType = "backlog.status_changed"
	EventBacklogPriorityChanged   EventType = "backlog.priority_changed"
	EventBacklogEffortChanged     EventType = "backlog.effort_changed"
	EventBacklogDependencyAdded   EventType = "backlog.dependency_added"
	EventBacklogDependencyRemoved EventType = "backlog.dependency_removed"
	EventBacklogInitiativeChanged EventType = "backlog.initiative_changed"
	EventBacklogBlocked           EventType = "backlog.blocked"
	EventBacklogUnblocked         EventType = "backlog.unblocked"
	EventBacklogArchived          EventType = "backlog.archived"
)

// Initiative events.
const (
	EventInitiativeCreated       EventType = "initiative.created"
	EventInitiativeItemAdded     EventType = "initiative.item_added"
	EventInitiativeItemRemoved   EventType = "initiative.item_removed"
	EventInitiativeStatusChanged EventType = "initiative.status_changed"
	EventInitiativeArchived      EventType = "initiative.archived"
)

// Execution events.
const (
	EventExecutionCreated       EventType = "execution.created"
	EventExecutionStatusChanged EventType = "execution.status_changed"
	EventExecutionCompleted     EventType = "execution.completed"
	EventExecutionFailed        EventType = "execution.failed"
	EventExecutionCanceled      EventType = "execution.canceled"
)

// Queue events.
const (
	EventQueued   EventType = "queue.queued"
	EventDequeued EventType = "queue.dequeued"
)

// Decision/workshop events.
const (
	EventWorkshopRoundCompleted EventType = "decision.workshop_round_completed"
)

// Event represents a single state change recorded in the event log.
type Event struct {
	ID         int64           `json:"id"`
	Timestamp  time.Time       `json:"timestamp"`
	EntityType EntityType      `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	EventType  EventType       `json:"event_type"`
	ActorType  string          `json:"actor_type"`
	ActorID    string          `json:"actor_id"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// Typed metadata payloads for each event category.

// StatusChangePayload records a from/to status transition.
type StatusChangePayload struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// PriorityChangePayload records a priority change.
type PriorityChangePayload struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// EffortChangePayload records an effort estimate change.
type EffortChangePayload struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// DependencyPayload records a dependency target.
type DependencyPayload struct {
	Target string `json:"target"`
}

// InitiativeChangePayload records an initiative assignment change.
type InitiativeChangePayload struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// BlockPayload records a block/unblock reason.
type BlockPayload struct {
	Reason string `json:"reason"`
}

// BacklogCreatedPayload records initial backlog item state.
type BacklogCreatedPayload struct {
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	Priority   int    `json:"priority"`
	Initiative string `json:"initiative,omitempty"`
	Effort     string `json:"effort,omitempty"`
}

// ExecutionCreatedPayload records execution creation details.
type ExecutionCreatedPayload struct {
	BacklogKind string `json:"backlog_kind"`
	BacklogName string `json:"backlog_name"`
	Mode        string `json:"mode"`
}

// ExecutionCompletedPayload records execution completion details.
type ExecutionCompletedPayload struct {
	DurationSeconds float64 `json:"duration_seconds"`
	HadFixups       bool    `json:"had_fixups"`
}

// ExecutionFailedPayload records execution failure details.
type ExecutionFailedPayload struct {
	Reason          string  `json:"reason"`
	DurationSeconds float64 `json:"duration_seconds"`
}

// ExecutionCanceledPayload records execution cancellation details.
type ExecutionCanceledPayload struct {
	Reason string `json:"reason"`
}

// QueuePayload records queue position info.
type QueuePayload struct {
	BacklogKind string `json:"backlog_kind"`
	BacklogName string `json:"backlog_name"`
	Position    int    `json:"position,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

// InitiativeItemPayload records item add/remove from initiative.
type InitiativeItemPayload struct {
	Item string `json:"item"`
}

// WorkshopRoundPayload records workshop round completion.
type WorkshopRoundPayload struct {
	RoundNumber int `json:"round_number"`
}
