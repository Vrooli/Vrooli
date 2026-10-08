// Responsibility: retain types declarations within their original package.
package domain

import (
	"github.com/google/uuid"
	"time"
)

// RunEvent represents a single event in a run's event stream.
//
// SchemaVersion identifies the on-wire shape of Data for typed events. It is
// recorded as a column on run_events (not embedded in the JSON body) so the
// event-log dispatch table can route old payloads to old payload types
// indefinitely while new payloads use the current types. The default is 1;
// the eventlog package is the source of truth for which versions are
// registered for which event types.
type RunEvent struct {
	ID            uuid.UUID    `json:"id" db:"id"`
	RunID         uuid.UUID    `json:"runId" db:"run_id"`
	Sequence      int64        `json:"sequence" db:"sequence"`
	EventType     RunEventType `json:"eventType" db:"event_type"`
	Timestamp     time.Time    `json:"timestamp" db:"timestamp"`
	SchemaVersion int          `json:"schemaVersion,omitempty" db:"schema_version"`
	Data          EventPayload `json:"data" db:"data"`
}

// RunEventType categorizes the event.
type RunEventType string

const (
	EventTypeLog               RunEventType = "log"
	EventTypeMessage           RunEventType = "message"
	EventTypeMessageDeleted    RunEventType = "message_deleted"
	EventTypeToolCall          RunEventType = "tool_call"
	EventTypeToolResult        RunEventType = "tool_result"
	EventTypeStatus            RunEventType = "status"
	EventTypeMetric            RunEventType = "metric"
	EventTypeArtifact          RunEventType = "artifact"
	EventTypeError             RunEventType = "error"
	EventTypeCompaction        RunEventType = "compaction"
	EventTypeLifecycle         RunEventType = "lifecycle"
	EventTypeGoalStatusChanged RunEventType = "goal_status_changed"

	// Typed operational events.
	//
	// These replace freeform LogEventData strings for operationally-significant
	// signals (fallback walks, sandbox ops, heartbeat misses, checkpoint
	// failures, model/runner health transitions). Payload structs and emit
	// helpers live in the eventlog package; the dispatch table there is the
	// authoritative (event_type, schema_version) → payload-type registry.
	EventTypeRunnerFallbackAttempted RunEventType = "runner.fallback.attempted"
	EventTypeRunnerFallbackExhausted RunEventType = "runner.fallback.exhausted"
	EventTypeModelFallbackAttempted  RunEventType = "model.fallback.attempted"
	EventTypeModelFallbackExhausted  RunEventType = "model.fallback.exhausted"
	EventTypePolicyCandidateAttempt  RunEventType = "policy.candidate.attempt"
	EventTypeModelHealthTransition   RunEventType = "model.health.transition"
	EventTypeRunnerHealthTransition  RunEventType = "runner.health.transition"
	EventTypeSandboxOperation        RunEventType = "sandbox.operation"
	EventTypeHeartbeatMiss           RunEventType = "heartbeat.miss"
	EventTypeCheckpointFailure       RunEventType = "checkpoint.failure"
	EventTypeRetryAttempt            RunEventType = "retry.attempt"
)

// IsTypedOperationalEvent reports whether t is one of the typed-operational
// event categories whose payload shape is owned by the eventlog package.
//
// The SQLite event store consults this so it can deserialize the payload
// through the eventlog dispatch table instead of trying to decode it as a
// legacy tagged-union shape.
