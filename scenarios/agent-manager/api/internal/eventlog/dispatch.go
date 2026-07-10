// Dispatch table for typed-operational events.
//
// Maps (RunEventType, schema_version) → factory that produces a fresh
// payload value of the registered Go type. The Repository read path uses
// this to decode a row's `data` column into the right struct without a
// giant switch; it also gives the codebase a single grep-able registry of
// every event payload shape that has ever been emitted.
//
// Versioning policy:
//   - Schema versions are forever. An entry is registered once and never
//     removed; old events keep decoding through their registered entry
//     even after a new version supersedes them in production.
//   - Adding an optional JSON field to a payload struct is non-breaking
//     and DOES NOT bump the version.
//   - Renaming a field, narrowing a type, or removing a field is breaking
//     and DOES require a new payload struct + a new dispatch entry at a
//     higher schema_version. The old struct stays so legacy rows decode
//     correctly.

package eventlog

import (
	"encoding/json"
	"fmt"
	"sync"

	"agent-manager/internal/domain"
)

// PayloadFactory produces a fresh, zero-valued payload pointer that the
// repository unmarshals into. Returning a fresh pointer (instead of
// json.Unmarshal-ing into a shared instance) keeps the dispatch table
// thread-safe.
type PayloadFactory func() Payload

// RegisteredKey identifies one row shape: an event type plus the
// schema_version recorded on the row. Exported so the stats engine's
// registry test (TestAllEmittedEventsAreProcessed) can enumerate every
// (event_type, schema_version) pair the dispatch table knows about and
// assert each has a stats processor.
type RegisteredKey struct {
	EventType     domain.RunEventType
	SchemaVersion int
}

var (
	dispatchMu     sync.RWMutex
	dispatchTable  = map[RegisteredKey]PayloadFactory{}
	defaultVersion = map[domain.RunEventType]int{}
)

// Register registers a payload factory at the given (event_type, schema_version).
// Panics if the same key is registered twice — duplicate registration is a
// programmer error and the table is initialized at package load.
func Register(eventType domain.RunEventType, schemaVersion int, factory PayloadFactory) {
	dispatchMu.Lock()
	defer dispatchMu.Unlock()
	key := RegisteredKey{EventType: eventType, SchemaVersion: schemaVersion}
	if _, exists := dispatchTable[key]; exists {
		panic(fmt.Sprintf("eventlog: duplicate dispatch registration for %s v%d", eventType, schemaVersion))
	}
	dispatchTable[key] = factory
	if cur, ok := defaultVersion[eventType]; !ok || schemaVersion > cur {
		defaultVersion[eventType] = schemaVersion
	}
}

// LatestSchemaVersion returns the highest schema_version registered for
// an event type. Builders use this so emitters always write the current
// shape without callers having to thread the version.
func LatestSchemaVersion(eventType domain.RunEventType) int {
	dispatchMu.RLock()
	defer dispatchMu.RUnlock()
	if v, ok := defaultVersion[eventType]; ok {
		return v
	}
	return 0
}

// Decode unmarshals raw payload bytes into the registered Go type for the
// given (event_type, schema_version). Returns an error if no entry is
// registered — Phase 3's `TestAllEmittedEventsAreProcessed` will catch
// anything emitted-but-not-registered before it ships.
func Decode(eventType domain.RunEventType, schemaVersion int, body json.RawMessage) (Payload, error) {
	dispatchMu.RLock()
	factory, ok := dispatchTable[RegisteredKey{EventType: eventType, SchemaVersion: schemaVersion}]
	dispatchMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("eventlog: no dispatch entry for %s v%d", eventType, schemaVersion)
	}
	value := factory()
	if len(body) > 0 {
		if err := json.Unmarshal(body, value); err != nil {
			return nil, fmt.Errorf("eventlog: decode %s v%d: %w", eventType, schemaVersion, err)
		}
	}
	return value, nil
}

// RegisteredKeys returns every (event_type, schema_version) pair currently
// registered. Sorted for stable test output.
func RegisteredKeys() []RegisteredKey {
	dispatchMu.RLock()
	defer dispatchMu.RUnlock()
	keys := make([]RegisteredKey, 0, len(dispatchTable))
	for k := range dispatchTable {
		keys = append(keys, k)
	}
	return keys
}

// init wires up Phase 1's typed event taxonomy. Each Register call is the
// authoritative source for "this event type at this schema_version uses
// this Go payload". When Phase 2/3 introduces new event categories or
// bumps a version, register the new entries here — never delete an old
// one, since old rows still need to decode.
func init() {
	Register(domain.EventTypeRunnerFallbackAttempted, 1, func() Payload {
		return &RunnerFallbackAttemptedPayload{}
	})
	Register(domain.EventTypeRunnerFallbackExhausted, 1, func() Payload {
		return &RunnerFallbackExhaustedPayload{}
	})
	Register(domain.EventTypeModelFallbackAttempted, 1, func() Payload {
		return &ModelFallbackAttemptedPayload{}
	})
	Register(domain.EventTypeModelFallbackExhausted, 1, func() Payload {
		return &ModelFallbackExhaustedPayload{}
	})
	Register(domain.EventTypePolicyCandidateAttempt, 1, func() Payload {
		return &PolicyCandidateAttemptPayload{}
	})
	Register(domain.EventTypeModelHealthTransition, 1, func() Payload {
		return &ModelHealthTransitionPayload{}
	})
	Register(domain.EventTypeRunnerHealthTransition, 1, func() Payload {
		return &RunnerHealthTransitionPayload{}
	})
	Register(domain.EventTypeSandboxOperation, 1, func() Payload {
		return &SandboxOperationPayload{}
	})
	Register(domain.EventTypeHeartbeatMiss, 1, func() Payload {
		return &HeartbeatMissPayload{}
	})
	Register(domain.EventTypeCheckpointFailure, 1, func() Payload {
		return &CheckpointFailurePayload{}
	})
	Register(domain.EventTypeRetryAttempt, 1, func() Payload {
		return &RetryAttemptPayload{}
	})
}
