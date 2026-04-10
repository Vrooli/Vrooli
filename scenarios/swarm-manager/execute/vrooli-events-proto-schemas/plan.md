# Implementation Plan: Draft vrooli-events Proto Schemas

## Required Reading

```bash
prompt-manager skill read implementation-plan-authoring
```

**Reference files (read before implementing):**
- `packages/proto/schemas/agent-manager/v1/domain/events.proto` — event pattern with oneof, timestamp, enums
- `packages/proto/schemas/agent-manager/v1/domain/types.proto` — enum and type patterns, go_package convention
- `packages/proto/buf.yaml` — module config (v2, lint exceptions)
- `packages/proto/buf.gen.yaml` — code generation plugins (Go, TS, Python, JS)

## Purpose

Create the proto schema definitions for the vrooli-events central event bus: event envelope, policy rules, and SSE message types. These schemas define the contract for all inter-scenario communication, policy enforcement, and real-time event subscriptions across the Vrooli ecosystem.

## Problem Statement

The vrooli-events initiative requires a shared proto contract before any runtime implementation can begin. Three sibling items (core-runtime, discovery-event-emission, analytics-ui) depend on these schemas. The schemas must follow established buf patterns from agent-manager/v1/ and generate bindings for Go, TypeScript, Python, and JavaScript.

## Scope

### In Scope
- Create `packages/proto/schemas/vrooli-events/v1/domain/envelope.proto` — event envelope with google.protobuf.Any payload
- Create `packages/proto/schemas/vrooli-events/v1/domain/policy.proto` — typed policy rule messages (PolicyMatcher, AccessRule, RateLimit, CircuitBreaker)
- Create `packages/proto/schemas/vrooli-events/v1/domain/sse.proto` — SSE subscription, notification, policy snapshot, and heartbeat messages
- Run `buf generate` and validate all generated bindings compile
- Update `buf.yaml` / `buf.gen.yaml` if needed (likely no changes needed)

### Out of Scope
- Runtime implementation (event store, SSE server, policy engine) — handled by execute/vrooli-events-core-runtime
- Discovery package integration — handled by execute/discovery-event-emission-and-policy-cache
- Policy CRUD API or gRPC service definitions — future item (will go in v1/api/)
- SQLite schema or storage layer

## Current Technical Context

### Existing Proto Infrastructure
- **buf.yaml**: v2 config at `packages/proto/buf.yaml`, module `buf.build/vrooli/schemas`, deps include `buf.build/googleapis/googleapis` and `buf.build/bufbuild/protovalidate`
- **buf.gen.yaml**: Generates Go (source_relative), TypeScript (ts + js), Python + pyi
- **Directory pattern**: `packages/proto/schemas/<scenario>/v1/domain/` for types/events, `v1/api/` for service definitions
- **Package naming**: `<scenario>_v1` with underscored convention (e.g., `agent_manager.v1`)
- **Go package**: `github.com/vrooli/vrooli/packages/proto/gen/go/<scenario>/v1/domain;domain`
- **Lint exceptions**: PACKAGE_VERSION_SUFFIX, PACKAGE_DIRECTORY_MATCH, PACKAGE_SAME_DIRECTORY, PACKAGE_SAME_GO_PACKAGE — all already configured
- **google.protobuf.Any**: Not yet used in codebase (only Struct). buf.yaml already has googleapis dep. This will be the first use.

### Reference Files
- `packages/proto/schemas/agent-manager/v1/domain/events.proto` — RunEvent with oneof, enums with UNSPECIFIED zero values, @layer/@domain annotations
- `packages/proto/schemas/agent-manager/v1/domain/types.proto` — enum patterns, go_package convention
- `packages/proto/schemas/agent-manager/v1/api/service.proto` — gRPC service definition pattern (not needed for this item)

### Architecture Decisions (from research/vrooli-events-architecture)
- Event envelope: event_id, source_scenario, target_scenario, event_type (structured as `{scenario}.{domain}.{action}.{version}`), timestamp, correlation_id, payload (Any), metadata (map)
- Policy rules: Typed per-kind (AccessRule, RateLimit, CircuitBreaker) — strong typing, no runtime parsing
- Access control: Most-specific-wins with segment-count scoring (exact=3, prefix=2, wildcard=1, max 9)
- Rate limiting: Token bucket (capacity + refill_rate)
- Circuit breaker: 3-state (CLOSED/OPEN/HALF_OPEN) with threshold + cooldown
- SSE: 30s heartbeat, Last-Event-ID resume, 64-capacity channels, drop+notify backpressure
- Policy cache invalidation: Full snapshot push via SSE

## Contract Decisions

### Settled (from workshop round 1)

1. **Directory structure**: Subdirectory pattern matching agent-manager — `v1/domain/` for all three proto files. `v1/api/` will be created later when gRPC service definitions are needed.

2. **PolicySnapshot content**: Typed repeated fields per rule kind (`repeated AccessRule access_rules`, `repeated RateLimit rate_limits`, `repeated CircuitBreaker circuit_breakers`). No wrapper/oneof indirection.

3. **SubscriptionRequest**: Structured filter with `event_type_pattern` (string glob), `source_scenario_pattern` (optional string), `target_scenario_pattern` (optional string).

4. **AccessRule specificity**: Include a pre-computed `specificity_score` int32 field. Score computed at rule creation/update time using segment-count algorithm (exact=3, prefix=2, wildcard=1, max total=9).

### Settled (from workshop round 2)

5. **EventNotification**: Embed full `EventEnvelope envelope` field plus a `stream_sequence` int64 field (monotonically increasing per-subscription, enables Last-Event-ID resume). Cleanly separates transport concerns (sequence) from domain data (envelope).

6. **PolicySnapshot versioning**: Include `version` (int64, monotonically increasing, set by server on policy change) and `generated_at` (google.protobuf.Timestamp). Clients compare version numbers — if received version <= cached version, discard. Provides robustness during SSE reconnection scenarios.

7. **Shared PolicyMatcher message**: Extract `PolicyMatcher { source_pattern, target_pattern, action_pattern }` as a shared message. Each rule type (AccessRule, RateLimit, CircuitBreaker) embeds a `PolicyMatcher matcher` field. DRY — adding a new pattern field only changes one place, and "matcher" is a natural concept in policy evaluation.

## Target End State

Three proto files in `packages/proto/schemas/vrooli-events/v1/domain/` that:
1. Define the canonical event envelope used by all inter-scenario communication
2. Define typed policy rule messages with a shared PolicyMatcher for access control, rate limiting, and circuit breaking
3. Define SSE protocol messages for subscriptions, notifications (with stream sequence), versioned policy pushes, and heartbeats
4. Generate clean bindings in Go, TypeScript, Python, and JavaScript via `buf generate`

## Implementation Strategy

### Phase 1: Directory Structure and envelope.proto
1. Create `packages/proto/schemas/vrooli-events/v1/domain/`
2. Write `envelope.proto` with:
   - `syntax = "proto3";`
   - `package vrooli_events.v1;`
   - `option go_package = "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain;domain";`
   - Import `google/protobuf/any.proto` and `google/protobuf/timestamp.proto`
   - `EventEnvelope` message: event_id (string), source_scenario (string), target_scenario (string), event_type (string), timestamp (google.protobuf.Timestamp), correlation_id (string), payload (google.protobuf.Any), metadata (map<string,string>)

### Phase 2: policy.proto
1. Write `policy.proto` with:
   - Same package/go_package as envelope.proto
   - `PolicyMatcher` message: source_pattern (string), target_pattern (string), action_pattern (string)
   - `Effect` enum: EFFECT_UNSPECIFIED=0, EFFECT_ALLOW=1, EFFECT_DENY=2
   - `CircuitBreakerState` enum: CIRCUIT_BREAKER_STATE_UNSPECIFIED=0, CIRCUIT_BREAKER_STATE_CLOSED=1, CIRCUIT_BREAKER_STATE_OPEN=2, CIRCUIT_BREAKER_STATE_HALF_OPEN=3
   - `AccessRule` message: matcher (PolicyMatcher), effect (Effect), specificity_score (int32)
   - `RateLimit` message: matcher (PolicyMatcher), capacity (int64), refill_rate (double)
   - `CircuitBreaker` message: matcher (PolicyMatcher), failure_threshold (int32), cooldown_seconds (int64), state (CircuitBreakerState)

### Phase 3: sse.proto
1. Write `sse.proto` with:
   - Import `vrooli-events/v1/domain/envelope.proto` and `vrooli-events/v1/domain/policy.proto`
   - `SubscriptionRequest`: event_type_pattern (string), source_scenario_pattern (optional string), target_scenario_pattern (optional string)
   - `EventNotification`: stream_sequence (int64), envelope (EventEnvelope)
   - `PolicySnapshot`: version (int64), generated_at (google.protobuf.Timestamp), repeated AccessRule access_rules, repeated RateLimit rate_limits, repeated CircuitBreaker circuit_breakers
   - `HeartbeatMessage`: timestamp (google.protobuf.Timestamp), dropped_count (int64)

### Phase 4: Generate and Validate
1. Run `buf lint` to catch issues
2. Run `buf generate` to produce all bindings
3. Verify Go bindings compile: `cd packages/proto && go build ./gen/go/vrooli-events/...`
4. Spot-check TypeScript/Python output exists

## Testing Plan

1. **buf lint** passes with no errors on all 3 proto files
2. **buf generate** succeeds for all configured plugins (Go, TS, Python, JS)
3. **Go compilation**: `cd packages/proto && go build ./gen/go/vrooli-events/...` exits 0
4. **TypeScript output**: Files exist under `packages/proto/gen/typescript/vrooli-events/`
5. **Python output**: Files exist under `packages/proto/gen/python/vrooli_events/`
6. **Field spot-check**: Verify generated Go structs have expected fields:
   - EventEnvelope.Payload is `*anypb.Any`
   - AccessRule.Matcher is `*PolicyMatcher` with source_pattern, target_pattern, action_pattern
   - AccessRule.SpecificityScore is int32
   - EventNotification.StreamSequence is int64, EventNotification.Envelope is `*EventEnvelope`
   - PolicySnapshot.Version is int64, PolicySnapshot.GeneratedAt is `*timestamppb.Timestamp`
   - PolicySnapshot has typed repeated fields: AccessRules, RateLimits, CircuitBreakers

## Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| google.protobuf.Any generates unexpected code in some language bindings | Medium | Medium | Test all 4 language outputs; fallback to bytes + type_url string fields if Any causes issues |
| Cross-file imports within vrooli-events/v1/domain/ cause buf lint issues | Low | Low | Lint exceptions for PACKAGE_SAME_DIRECTORY already configured; test import paths early |
| Package naming conflicts with existing schemas | Low | Low | Follow established convention: `vrooli_events.v1` package |
| buf lint rules reject new patterns (e.g., Any usage) | Low | Medium | Check lint exceptions in buf.yaml; add exceptions if needed |
| PolicyMatcher extraction changes field access patterns for downstream consumers | Low | Low | This is the first schema version — no existing consumers to break |

## Non-goals / Prohibited Patterns

- Do not implement any runtime logic — this is schema-only
- Do not modify existing proto schemas in other scenarios
- Do not add new buf plugins or dependencies beyond what's already configured
- Do not create gRPC service definitions — those belong in a future v1/api/ item

## Definition of Done

- [ ] Three proto files exist in `packages/proto/schemas/vrooli-events/v1/domain/`
- [ ] `buf lint` passes
- [ ] `buf generate` produces bindings in Go, TypeScript, Python, and JavaScript
- [ ] Go bindings compile without errors
- [ ] All field names, types, and enums match the research conclusion and workshop decisions
- [ ] Enum zero values follow UNSPECIFIED convention
- [ ] go_package follows established pattern
- [ ] PolicyMatcher is used by all three rule types (AccessRule, RateLimit, CircuitBreaker)
- [ ] EventNotification has stream_sequence + embedded EventEnvelope
- [ ] PolicySnapshot has version + generated_at fields
