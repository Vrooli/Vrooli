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
	EntityBacklogItem  EntityType = "backlog_item"
	EntityInitiative   EntityType = "initiative"
	EntityExecution    EntityType = "execution"
	EntityQueue        EntityType = "queue"
	EntityCapture      EntityType = "capture"
	EntityAgentSession EntityType = "agent_session"
	EntityRecord       EntityType = "record"
	// EntityGoal identifies goal-scope entities. A goal is an explicit set of
	// end-state targets (backlog items and/or initiatives) whose transitive
	// prerequisite closure defines the work tracked toward it. See the goals
	// domain (internal/goals).
	EntityGoal EntityType = "goal"
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
	EventBacklogMilestoneChanged  EventType = "backlog.milestone_changed"
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
	EventInitiativeModeChanged   EventType = "initiative.mode_changed"
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

// Goal events. A goal is a first-class scope entity (see internal/goals). The
// scope-snapshot event records the closure size / progress over time so
// per-goal scope-creep is surfaced rather than hidden.
const (
	EventGoalCreated              EventType = "goal.created"
	EventGoalUpdated              EventType = "goal.updated"
	EventGoalTargetAdded          EventType = "goal.target_added"
	EventGoalTargetRemoved        EventType = "goal.target_removed"
	EventGoalPriorityChanged      EventType = "goal.priority_changed"
	EventGoalArchived             EventType = "goal.archived"
	EventGoalUnarchived           EventType = "goal.unarchived"
	EventGoalScopeSnapshot        EventType = "goal.scope_snapshot"
	EventMilestoneCreated         EventType = "milestone.created"
	EventMilestoneUpdated         EventType = "milestone.updated"
	EventMilestoneItemsAssigned   EventType = "milestone.items_assigned"
	EventMilestoneItemsUnassigned EventType = "milestone.items_unassigned"
	EventMilestoneArchived        EventType = "milestone.archived"
)

// Calibration events. A duration_sample is a coarse per-item lead-time
// observation (created → completed, in hours) tagged with the item's effort
// class. The ETA engine folds these into per-effort-class distributions;
// backfill-origin samples (emitted once from historical spec timestamps) are
// weighted lower than live-origin samples. Emitting these as first-class
// events keeps the event log the single source of truth for the ETA engine,
// mirroring how the stats engine is already event-sourced.
const (
	EventBacklogDurationSample EventType = "backlog.duration_sample"
)

// Decision/workshop events.
const (
	EventWorkshopRoundCompleted EventType = "decision.workshop_round_completed"
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

// Record events. Records are narrative artifacts of completed work; the
// event-log bridge is intentionally thin (created + superseded) so the stats
// engine can fold throughput and regression-rate without re-walking the
// records store.
const (
	EventRecordCreated    EventType = "record.created"
	EventRecordSuperseded EventType = "record.superseded"
)

// System events for one-time migrations.
const (
	EventSystemMigrationApplied EventType = "system.migration_applied"
)

// Operating mode events.
const (
	EventOperatingModePhaseStarted   EventType = "operating_mode.phase_started"
	EventOperatingModePhaseCompleted EventType = "operating_mode.phase_completed"
	EventOperatingModePhaseFailed    EventType = "operating_mode.phase_failed"
	EventOperatingModePhaseCanceled  EventType = "operating_mode.phase_canceled"
	EventOperatingModeReplanNeeded   EventType = "operating_mode.replan_needed"
	EventOperatingModeBacklogSynced  EventType = "operating_mode.backlog_synced"
)

// Agent session events.
const (
	EventAgentSessionCreated         EventType = "agent_session.created"
	EventAgentSessionStarted         EventType = "agent_session.started"
	EventAgentSessionContinued       EventType = "agent_session.continued"
	EventAgentSessionCompleted       EventType = "agent_session.completed"
	EventAgentSessionFailed          EventType = "agent_session.failed"
	EventAgentSessionCanceled        EventType = "agent_session.canceled"
	EventAgentSessionDeleted         EventType = "agent_session.deleted"
	EventAgentSessionProposalCreated EventType = "agent_session.proposal_created"
	EventAgentSessionProposalApplied EventType = "agent_session.proposal_applied"
	EventAgentSessionArtifactLinked  EventType = "agent_session.artifact_linked"
)

// EntitySystem is used for events that are not tied to a domain entity.
const EntitySystem EntityType = "system"

// Event represents a single state change recorded in the event log.
type Event struct {
	ID                 int64           `json:"id"`
	Timestamp          time.Time       `json:"timestamp"`
	EntityType         EntityType      `json:"entity_type"`
	EntityID           string          `json:"entity_id"`
	EventType          EventType       `json:"event_type"`
	ActorType          string          `json:"actor_type"`
	ActorID            string          `json:"actor_id"`
	RunID              string          `json:"run_id,omitempty"`
	VerificationStatus string          `json:"verification_status"`
	HarnessSessionID   string          `json:"harness_session_id,omitempty"`
	HarnessKind        string          `json:"harness_kind,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
}

// Typed metadata payloads for each event category.

// StatusChangePayload records a from/to status transition.
type StatusChangePayload struct {
	From     string                        `json:"from"`
	To       string                        `json:"to"`
	Source   *BacklogMutationSourcePayload `json:"source,omitempty"`
	ItemRefs []string                      `json:"item_refs,omitempty"`
}

// BacklogMutationSourcePayload records the causality chain for backlog
// mutations triggered by higher-level workflows such as operating modes.
type BacklogMutationSourcePayload struct {
	Entrypoint     string `json:"entrypoint,omitempty"`
	InitiativeName string `json:"initiative_name,omitempty"`
	Mode           string `json:"mode,omitempty"`
	Phase          string `json:"phase,omitempty"`
	Round          int    `json:"round,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	RequestedBy    string `json:"requested_by,omitempty"`
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

// MilestoneChangePayload records a goal-owned milestone assignment change.
type MilestoneChangePayload struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// InitiativeModeChangePayload records an initiative operating-mode transition.
type InitiativeModeChangePayload struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// OperatingModePhasePayload records an initiative operating-mode phase
// lifecycle event. It is intentionally shared by started/completed/failed/
// canceled so stats can aggregate phase usage without phase-specific parsing.
type OperatingModePhasePayload struct {
	Mode           string `json:"mode"`
	ScopeKind      string `json:"scope_kind"`
	ScopeID        string `json:"scope_id"`
	InitiativeName string `json:"initiative_name,omitempty"`
	Phase          string `json:"phase"`
	// PhaseKind is the lane classification (investigate / execute /
	// review / reconcile) attached at emit-time so stats aggregation does
	// not have to round-trip through the registry. Resolved by
	// historical phase payloads via their stored phase kind.
	PhaseKind       string   `json:"phase_kind,omitempty"`
	RunStrategy     string   `json:"run_strategy"`
	AgentProfileKey string   `json:"agent_profile_key"`
	RoundNumber     int      `json:"round_number,omitempty"`
	RunID           string   `json:"run_id,omitempty"`
	DurationSeconds float64  `json:"duration_seconds,omitempty"`
	Status          string   `json:"status,omitempty"`
	Verdict         string   `json:"verdict,omitempty"`
	ReplanNeeded    bool     `json:"replan_needed,omitempty"`
	ArtifactPaths   []string `json:"artifact_paths,omitempty"`
}

// OperatingModeBacklogSyncPayload records the audited backlog reconciliation
// summary for a mode phase. Counts are recorded even for no-op syncs so the
// absence of mutations remains observable.
type OperatingModeBacklogSyncPayload struct {
	Mode                  string                        `json:"mode"`
	ScopeKind             string                        `json:"scope_kind"`
	ScopeID               string                        `json:"scope_id"`
	InitiativeName        string                        `json:"initiative_name,omitempty"`
	Phase                 string                        `json:"phase"`
	RunStrategy           string                        `json:"run_strategy,omitempty"`
	AgentProfileKey       string                        `json:"agent_profile_key,omitempty"`
	RoundNumber           int                           `json:"round_number,omitempty"`
	RunID                 string                        `json:"run_id,omitempty"`
	Status                string                        `json:"status,omitempty"`
	BacklogItemsCompleted int                           `json:"backlog_items_completed,omitempty"`
	BacklogItemsCreated   int                           `json:"backlog_items_created,omitempty"`
	BacklogItemsUpdated   int                           `json:"backlog_items_updated,omitempty"`
	ItemRefs              []string                      `json:"item_refs,omitempty"`
	Source                *BacklogMutationSourcePayload `json:"source,omitempty"`
	ArtifactPaths         []string                      `json:"artifact_paths,omitempty"`
}

// BlockPayload records a block/unblock reason.
type BlockPayload struct {
	Reason string `json:"reason"`
}

// BacklogCreatedPayload records initial backlog item state.
type BacklogCreatedPayload struct {
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Priority  int    `json:"priority"`
	Milestone string `json:"milestone,omitempty"`
	Effort    string `json:"effort,omitempty"`
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

// GoalCreatedPayload records the initial state of a goal.
type GoalCreatedPayload struct {
	Title    string   `json:"title,omitempty"`
	Priority int      `json:"priority,omitempty"`
	Targets  []string `json:"targets,omitempty"`
	Seeded   bool     `json:"seeded,omitempty"`
}

// GoalTargetPayload records a single target add/remove. A target ref is
// "<kind>/<name>" for a backlog item or "initiative/<name>" for an initiative.
type GoalTargetPayload struct {
	Target string `json:"target"`
}

// GoalScopeSnapshotPayload records a point-in-time snapshot of a goal's
// transitive closure so scope growth (creep) is observable over time.
type GoalScopeSnapshotPayload struct {
	TargetCount    int `json:"target_count"`
	ClosureSize    int `json:"closure_size"`
	CompletedCount int `json:"completed_count"`
	BlockedCount   int `json:"blocked_count,omitempty"`
}

// MilestonePayload describes a change to a milestone owned by a goal.
type MilestonePayload struct {
	GoalName      string   `json:"goal_name"`
	MilestoneName string   `json:"milestone_name"`
	Items         []string `json:"items,omitempty"`
}

// DurationSamplePayload records one coarse lead-time observation for a
// completed backlog item, tagged with its effort class. Origin is "backfill"
// (derived once from historical spec timestamps) or "live" (a completion
// observed after instrumentation landed). The ETA engine weights backfill
// samples lower. EffortClass is empty for unsized items (the ETA engine folds
// those into the global distribution).
type DurationSamplePayload struct {
	EffortClass   string  `json:"effort_class,omitempty"`
	DurationHours float64 `json:"duration_hours"`
	Origin        string  `json:"origin"`
	Kind          string  `json:"kind,omitempty"`
	Milestone     string  `json:"milestone,omitempty"`
}

// DurationSampleOrigin values for DurationSamplePayload.Origin.
const (
	DurationOriginBackfill = "backfill"
	DurationOriginLive     = "live"
)

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
	Mode            string `json:"mode,omitempty"`
	Phase           string `json:"phase,omitempty"`
	FeedbackRoundID string `json:"feedback_round_id,omitempty"`
	ReviewRoundID   string `json:"review_round_id,omitempty"`
	RoundNumber     int    `json:"round_number,omitempty"`
	RoundSlug       string `json:"round_slug,omitempty"`
	RunID           string `json:"run_id,omitempty"`
	SessionID       string `json:"session_id,omitempty"`
	Entrypoint      string `json:"entrypoint,omitempty"`
	DecidedBy       string `json:"decided_by,omitempty"`
	MutationID      string `json:"mutation_id"`
	Op              string `json:"op"`
	Target          string `json:"target,omitempty"`
	// Sources lists the source refs collapsed into Target for merge_items
	// mutations. Empty for every other op. Lets per-source history queries
	// surface "this item was merged into Target" without re-deriving from
	// archive timestamps.
	Sources []string `json:"sources,omitempty"`
}

// RecordCreatedPayload records a records.Create or records.CreateStub call.
// Stub is true when the record was auto-created on backlog completion with
// empty narrative fields; stats consumers use this to compute stub-fill rate.
type RecordCreatedPayload struct {
	Kind       string `json:"kind"`
	Scenario   string `json:"scenario"`
	BacklogRef string `json:"backlog_ref,omitempty"`
	Stub       bool   `json:"stub"`
}

// RecordSupersededPayload records that a record was superseded by a successor.
// SupersededID is the predecessor; Event.EntityID is the successor record id
// so per-record history surfaces the link from both directions.
type RecordSupersededPayload struct {
	SupersededID string `json:"superseded_id"`
	Reason       string `json:"reason,omitempty"`
}

// MigrationAppliedPayload records that a one-time migration has completed.
// The Name field is the sentinel key used to gate re-runs.
type MigrationAppliedPayload struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	AffectedIDs int    `json:"affected_ids,omitempty"`
}
