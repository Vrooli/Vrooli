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
	EntityCapture     EntityType = "capture"
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
	EventBacklogUnarchived        EventType = "backlog.unarchived"
	EventBacklogDeleted           EventType = "backlog.deleted"
	EventBacklogProposalApplied   EventType = "backlog.proposal_applied"
)

// Initiative events.
const (
	EventInitiativeCreated       EventType = "initiative.created"
	EventInitiativeItemAdded     EventType = "initiative.item_added"
	EventInitiativeItemRemoved   EventType = "initiative.item_removed"
	EventInitiativeStatusChanged EventType = "initiative.status_changed"
	EventInitiativeArchived      EventType = "initiative.archived"
	EventInitiativeUnarchived    EventType = "initiative.unarchived"
)

// Execution events.
const (
	EventExecutionCreated          EventType = "execution.created"
	EventExecutionStatusChanged    EventType = "execution.status_changed"
	EventExecutionCompleted        EventType = "execution.completed"
	EventExecutionFailed           EventType = "execution.failed"
	EventExecutionCanceled         EventType = "execution.canceled"
	EventExecutionManuallyAccepted EventType = "execution.manually_accepted"
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

// Clarification events.
const (
	EventClarificationStarted  EventType = "backlog.clarification_started"
	EventClarificationResolved EventType = "backlog.clarification_resolved"
	EventClarificationAction   EventType = "backlog.clarification_action"
)

// Review evidence events.
const (
	EventReviewStarted          EventType = "review.started"
	EventReviewEvidenceAdded    EventType = "review.evidence_added"
	EventReviewEvidenceVerified EventType = "review.evidence_verified"
	EventReviewRequestCreated   EventType = "review.request_created"
	EventReviewRequestFulfilled EventType = "review.request_fulfilled"
	EventReviewRoundCompleted   EventType = "review.round_completed"
	EventReviewFailed           EventType = "review.failed"
)

// View events (read-only analytics).
const (
	EventBacklogViewed    EventType = "backlog.viewed"
	EventExecutionViewed  EventType = "execution.viewed"
	EventInitiativeViewed EventType = "initiative.viewed"
	EventCaptureViewed    EventType = "capture.viewed"
)

// System events for one-time migrations.
const (
	EventSystemMigrationApplied EventType = "system.migration_applied"
)

// EntitySystem is used for events that are not tied to a domain entity.
const EntitySystem EntityType = "system"

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

// ExecutionManuallyAcceptedPayload records a manual-accept event where the
// user overrode the agent's failure verdict and judged the execution
// acceptable. Emitted in addition to a regular execution.completed event so
// stats can distinguish agent-finished from human-finished work.
type ExecutionManuallyAcceptedPayload struct {
	AcceptedBy         string `json:"accepted_by"`
	Reason             string `json:"reason,omitempty"`
	PreviousExecStatus string `json:"previous_exec_status"`
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
//
// Per-item decision counters (ItemsTotal, ItemsAnswered,
// ItemsRecommendedChosen, ItemsFreeformChosen) drive the recommendation
// acceptance metric. Pre-existing events emitted before the per-item
// schema landed have only RoundNumber populated; the stats engine treats
// zero ItemsAnswered as "no signal" and contributes nothing, so old and
// new events coexist without compatibility branches.
//
// Counting rules used by workshop.SummarizeRound:
//   - Only items with Type == "decision" count toward ItemsTotal.
//   - ItemsAnswered counts items with Selected != nil.
//   - ItemsRecommendedChosen counts items where the selected option's
//     Recommended flag is true. A freeform answer (Selected == OtherKey)
//     never increments this — picking "Other" rejects the recommendation.
//   - ItemsFreeformChosen counts items where Selected == OtherKey.
type WorkshopRoundPayload struct {
	RoundNumber            int    `json:"round_number"`
	Kind                   string `json:"kind,omitempty"`
	ItemsTotal             int    `json:"items_total,omitempty"`
	ItemsAnswered          int    `json:"items_answered,omitempty"`
	ItemsRecommendedChosen int    `json:"items_recommended_chosen,omitempty"`
	ItemsFreeformChosen    int    `json:"items_freeform_chosen,omitempty"`
}

// ClarificationStartedPayload records clarification initiation.
type ClarificationStartedPayload struct {
	RoundNumber int    `json:"round_number"`
	ItemID      string `json:"item_id"`
	HasMessage  bool   `json:"has_message"`
}

// ClarificationResolvedPayload records clarification completion.
type ClarificationResolvedPayload struct {
	RoundNumber  int    `json:"round_number"`
	ItemID       string `json:"item_id"`
	MessageCount int    `json:"message_count"`
	ImpactLevel  string `json:"impact_level"`
}

// ClarificationActionPayload records which post-clarification action was taken.
type ClarificationActionPayload struct {
	RoundNumber int    `json:"round_number"`
	ItemID      string `json:"item_id"`
	Action      string `json:"action"`
}

// ReviewStartedPayload records review agent initiation.
type ReviewStartedPayload struct {
	ExecutionID string `json:"execution_id"`
	RoundNumber int    `json:"round_number"`
}

// ReviewEvidencePayload records evidence item creation.
type ReviewEvidencePayload struct {
	ExecutionID  string `json:"execution_id"`
	EvidenceID   string `json:"evidence_id"`
	EvidenceType string `json:"evidence_type"`
}

// ReviewVerifiedPayload records evidence verification by a user.
type ReviewVerifiedPayload struct {
	ExecutionID string `json:"execution_id"`
	EvidenceID  string `json:"evidence_id"`
}

// ReviewRequestPayload records an additional evidence request.
type ReviewRequestPayload struct {
	ExecutionID string `json:"execution_id"`
	RequestID   string `json:"request_id"`
	Description string `json:"description,omitempty"`
	EvidenceID  string `json:"evidence_id,omitempty"`
}

// ReviewRoundCompletedPayload records review round completion with metrics.
type ReviewRoundCompletedPayload struct {
	ExecutionID    string  `json:"execution_id"`
	RoundNumber    int     `json:"round_number"`
	EvidenceCount  int     `json:"evidence_count"`
	Classification string  `json:"classification"`
	DurationSecs   float64 `json:"duration_seconds"`
}

// ReviewFailedPayload records review agent failure.
type ReviewFailedPayload struct {
	ExecutionID  string  `json:"execution_id"`
	Reason       string  `json:"reason"`
	DurationSecs float64 `json:"duration_seconds"`
}

// ArchivePayload records an archive event with context about what was archived.
type ArchivePayload struct {
	PreviousStatus string `json:"previous_status"`
	ArchivedAt     string `json:"archived_at"`
}

// UnarchivePayload records an unarchive event.
type UnarchivePayload struct {
	ArchivedAt string `json:"archived_at"`
}

// ViewPayload records a view event. Intentionally minimal.
type ViewPayload struct {
	Kind string `json:"kind,omitempty"`
}

// ProposalAppliedPayload records a single mutation applied through the
// proposals layer (initiative feedback or review). EntityID on the parent
// Event is the affected backlog ref so per-item history surfaces these
// alongside other backlog events; the originating round lives in the
// payload so consumers can group by feedback/review round.
type ProposalAppliedPayload struct {
	InitiativeName  string `json:"initiative_name"`
	FeedbackRoundID string `json:"feedback_round_id,omitempty"`
	ReviewRoundID   string `json:"review_round_id,omitempty"`
	RoundNumber     int    `json:"round_number,omitempty"`
	RoundSlug       string `json:"round_slug,omitempty"`
	Entrypoint      string `json:"entrypoint,omitempty"`
	DecidedBy       string `json:"decided_by,omitempty"`
	MutationID      string `json:"mutation_id"`
	Op              string `json:"op"`
	Target          string `json:"target,omitempty"`
}

// MigrationAppliedPayload records that a one-time migration has completed.
// The Name field is the sentinel key used to gate re-runs.
type MigrationAppliedPayload struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	AffectedIDs int    `json:"affected_ids,omitempty"`
}
