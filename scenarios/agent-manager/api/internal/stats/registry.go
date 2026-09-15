// Stats processor registry.
//
// Maps (event_type, schema_version) → processor. Replaces the giant
// switch in swarm-manager's processEvent (one of the four weakness
// fixes). Two consequences fall out of this shape:
//
//  1. Adding a new typed event requires registering a processor here —
//     forgetting to do so is caught by TestAllEmittedEventsAreProcessed,
//     not by silent zero counters in production.
//  2. Schema versions never collide: a v2 payload is a separate map
//     entry, can have a different processor signature, and never
//     accidentally falls through a `default:` branch.
//
// The aggregate state lives in the engine and is mutated by processors
// via the *aggregateState pointer. Processors do not return errors;
// malformed payloads (which the dispatch table already type-checked at
// decode time) are no-ops.

package stats

import (
	"sort"
	"sync"

	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
)

// Processor mutates aggregateState for one decoded record. Implementors
// type-assert on rec.Payload to the registered struct; the dispatch
// table guarantees the type at decode time.
type Processor func(state *aggregateState, rec eventlog.Record)

// processorKey deliberately mirrors eventlog.RegisteredKey so the
// "every registered event has a processor" assertion is a direct map
// comparison.
type processorKey struct {
	EventType     domain.RunEventType
	SchemaVersion int
}

var (
	processorMu       sync.RWMutex
	processorRegistry = map[processorKey]Processor{}
)

// RegisterProcessor wires a processor into the registry. Panics on
// duplicate keys — duplicate registration is a programmer error caught
// at package init.
func RegisterProcessor(eventType domain.RunEventType, schemaVersion int, p Processor) {
	processorMu.Lock()
	defer processorMu.Unlock()
	key := processorKey{EventType: eventType, SchemaVersion: schemaVersion}
	if _, exists := processorRegistry[key]; exists {
		panic("stats: duplicate processor registration for " + string(eventType))
	}
	processorRegistry[key] = p
}

// lookupProcessor returns the registered processor for the given
// (event_type, schema_version), or nil if none. The engine logs a warn
// on nil so missing processors are visible in operator logs even when
// CI didn't catch them (e.g., a new schema version snuck through).
func lookupProcessor(eventType domain.RunEventType, schemaVersion int) Processor {
	processorMu.RLock()
	defer processorMu.RUnlock()
	return processorRegistry[processorKey{EventType: eventType, SchemaVersion: schemaVersion}]
}

// RegisteredProcessorKeys returns every (event_type, schema_version)
// pair that has a processor. Used by registry_test.go's coverage check.
// Result is sorted (event_type asc, schema_version asc) for stable
// diff output in test failures.
func RegisteredProcessorKeys() []eventlog.RegisteredKey {
	processorMu.RLock()
	defer processorMu.RUnlock()
	keys := make([]eventlog.RegisteredKey, 0, len(processorRegistry))
	for k := range processorRegistry {
		keys = append(keys, eventlog.RegisteredKey{EventType: k.EventType, SchemaVersion: k.SchemaVersion})
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].EventType != keys[j].EventType {
			return keys[i].EventType < keys[j].EventType
		}
		return keys[i].SchemaVersion < keys[j].SchemaVersion
	})
	return keys
}
