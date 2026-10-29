# Event Taxonomy

Typed-operational events that ride the `run_events` table. These events
replace the freeform `LogEventData.Message` strings the orchestration phases
used to emit for fallback walks, sandbox lifecycle outcomes, heartbeat
misses, checkpoint failures, and model/runner health transitions. The
goal: every operationally-significant signal is queryable as structured
data, not regex-over-log-strings.

The eventlog package
(`scenarios/agent-manager/api/internal/eventlog/`) owns:

- the constant + payload struct for each event,
- the `(event_type, schema_version) → Go-type` dispatch table,
- the typed read repository,
- the build/emitter helpers.

The `domain.RunEvent` struct carries `SchemaVersion int` (column on
`run_events`) and `Data EventPayload`. Typed events use
`domain.TypedEventData` as the payload shape; its `MarshalJSON`/`UnmarshalJSON`
preserve the eventlog-package payload bytes verbatim so the dispatch
table is the single decoder.

## Schema version contract

- `schema_version` is **forever-forward-compatible**. Adding an optional
  field to a payload struct is non-breaking; readers ignore unknown JSON
  fields.
- Renaming or removing a field, or narrowing a type, is breaking. Bump
  the version: define a new payload struct, register a new entry in
  `dispatch.go` at the higher `schema_version`, and emit at the new
  version going forward. The old entry stays — old rows keep decoding.
- Schema versions are never reused. An entry, once registered, is
  permanent.
- `run_events.schema_version` defaults to 1 for legacy rows.

## Phase 1 events

Every entry below is registered at `schema_version = 1`. The "Replaces"
column shows the freeform string that the operationally-significant
emission used before — those `EmitSystemEvent` calls have been deleted.

| Event type                     | Payload struct                       | Replaces                                                       |
|--------------------------------|--------------------------------------|----------------------------------------------------------------|
| `runner.fallback.attempted`    | `RunnerFallbackAttemptedPayload`     | `acquire.go` "runner fallback: %s -> %s"                       |
| `runner.fallback.exhausted`    | `RunnerFallbackExhaustedPayload`     | (no prior emission — new in Phase 1)                           |
| `model.fallback.attempted`     | `ModelFallbackAttemptedPayload`      | `execute.go` "model fallback: %s -> %s" + "model attempt N/M"  |
| `model.fallback.exhausted`     | `ModelFallbackExhaustedPayload`      | `execute.go` "model fallback exhausted ..."                    |
| `model.health.transition`      | `ModelHealthTransitionPayload`       | (in-memory only today — Phase 2 emits)                         |
| `runner.health.transition`     | `RunnerHealthTransitionPayload`      | (no prior emission — Phase 2 emits)                            |
| `sandbox.operation`            | `SandboxOperationPayload`            | `finalize.go` "sandbox deleted/stopped/failed-to-..."          |
| `heartbeat.miss`               | `HeartbeatMissPayload`               | `heartbeat.go` "heartbeat update failed", "heartbeat checkpoint failed" |
| `checkpoint.failure`           | `CheckpointFailurePayload`           | `checkpoint.go` "failed to persist phase update", "failed to save checkpoint" |
| `retry.attempt`                | `RetryAttemptPayload`                | (reserved — Phase 2 emits)                                     |

Imported conversation messages with `role: "user"` are retained as the
typed `operator_turn` evidence kind. Live runs keep prompt suppression; the
operator-turn kind is reserved for historical user/operator input so
intervention and awaiting-human measures can distinguish it from assistant
messages.

## Usage events and reconciliation

`UsageEventData` is a metric payload, not a charge payload. Runner codecs may
emit one usage event for each provider turn with `turnIndex` and the observed
input/output/cache fields. Claude's terminal `result` usage snapshot is marked
`reconciliationAuthority: true`; when present, the invocation read model uses
that snapshot for run totals and does not add the per-turn events again. A
terminal snapshot remains the provider's reported total, so its token values
are not rewritten to fit a local estimate. Codex and OpenCode usage events are
per-turn and carry the current turn index. Replays remain idempotent by
deduplicating identical snapshots within the same turn.

Compaction events carry adjacent assistant-turn input counts when the stream
provides them. `TokensAfter / TokensBefore` is the measured whole-context
attenuation ratio used for later residency segments; it is an approximation
for individual tool-result survival, not a per-item provider measurement.
Compaction itself is represented by the `compaction` token-accounting bucket
so its summarisation read is not silently assigned to a neighboring tool call.

Target (Usage Intelligence T-2, being implemented by W1): usage rows carry the
provider's raw cumulative counters, and one accounting function derives a
session's usage as the last cumulative value minus the value before the run's
first turn. The Codex receipt-versus-delta authority selection described above
is then deleted. See [USAGE-INTELLIGENCE.md](../concepts/USAGE-INTELLIGENCE.md).

## Quota events

A provider quota reading is its own stream event type, never an error:

| Event type | Payload struct | Proto |
|------------|----------------|-------|
| `quota` (`domain.EventTypeQuota`) | `domain.RateLimitEventData` | `RUN_EVENT_TYPE_QUOTA`, `RunEvent.rate_limit` |

- `quota` sits next to `metric` and `compaction` because codecs emit it from
  the provider stream. It is not an orchestration-emitted typed-operational
  event, so it does not use the dotted names of the table below.
- Sources: Codex `token_count.rate_limits` (percent of window), Claude
  transcript `quotaLimits` (`status`, `rateLimitType`, `resetsAt`,
  `overageStatus`: a standing without a percentage), and Claude rate-limit
  result text. Claude pools are `five_hour` (300m) and `seven_day` (10080m).
- The payload keeps provider, pool, `usedPercent`, `windowMinutes`, reset
  time, the provider's verbatim `status`/`overageStatus` and provenance. A
  missing `usedPercent` means unknown, never zero.
- `error` events decode only as `ErrorEventData`. No consumer infers a quota
  frame from an error payload's shape; event counts, the `type=` filter and
  run reports count `error` and `quota` separately.
- Every run-event sink is built by `newRunEventSink`, which always applies the
  orchestrator's post-persist hook set (`afterRunEventPersisted`: quota
  observation, workflow usage nudge, token cap) on the headless executor,
  interactive, continuation and recovery paths. Imported (read-only) runs record
  quota observations only; token caps and nudges never act on them.
- Each quota event becomes one `quota_observations` row through
  `pricing.ObservationFromEvent`. The row id (`pricing.QuotaObservationID`) is
  the provider reading (provider, pool, window, observed time, provenance,
  standing, percent, reset), not the run or event that carried it, so the same
  rollout frame seen live and through an import is one row.
- Start-up migration (`database.migrateQuotaEvents`) rewrites legacy quota
  frames stored as `error` (tagged `payloadKind: "quota"`, or the untagged
  shape with a nonempty `limitType` and no generic error field) to `quota`
  and back-fills their observations in the same transaction. It is
  idempotent.
- The Claude Code statusline feed (`resource-claude-code statusline run`,
  samples in `~/.vrooli/state/vrooli/claude-code/quota-samples.ndjson`) is
  ingested by `pricing.ClaudeStatuslineReader` as provenance
  `claude-code:statusline`; it produces observations, not run events.

## Payload schemas (v1)

### `runner.fallback.attempted`

Historical payload retained so previously written events remain decodable.
New policy-backed runs emit `policy.candidate.attempt` with catalog digest
and candidate index instead.

```json
{
  "from": "claude-code",
  "to": "codex",
  "reason": "claude binary missing",
  "attempt_no": 1
}
```

`reason` is a freeform string in Phase 1 and becomes a `fallback.Reason`
enum value in Phase 2. The JSON shape is unchanged.

### `runner.fallback.exhausted`

Historical payload retained for previously written events. New runs report
terminal policy-candidate exhaustion through the digest-qualified policy
event.

```json
{
  "primary": "claude-code",
  "candidates_tried": ["codex", "opencode"],
  "last_reason": "all unavailable"
}
```

### `model.fallback.attempted`

Historical payload retained for old event rows. The active execution contract
uses immutable policy candidates rather than preset chains.

```json
{
  "from": "sonnet-4",
  "to": "haiku",
  "reason": "rate_limit",
  "attempt_no": 2,
  "chain_position": 2,
  "chain_length": 3
}
```

### `model.fallback.exhausted`

Historical payload retained for old event rows.

```json
{
  "preset": "CHEAP",
  "chain": ["a", "b", "c"],
  "last_reason": "all unavailable"
}
```

### `model.health.transition`

Records that a `(runner, model)` pair has flipped status. Phase 2
emitter writes.

```json
{
  "runner": "claude-code",
  "model": "sonnet-4",
  "from_status": "ok",
  "to_status": "failed",
  "reason": "rate_limit",
  "message": "429"
}
```

Status enum: `ok | unknown | failed` (see `eventlog.HealthStatus*`).

### `runner.health.transition`

```json
{
  "runner": "codex",
  "from_status": "unknown",
  "to_status": "ok",
  "reason": "probe_pass"
}
```

### `sandbox.operation`

Records the outcome of a sandbox lifecycle action issued from finalize.

```json
{
  "operation": "delete",
  "success": false,
  "duration_ms": 42,
  "reason": "finalize",
  "message": "workspace-sandbox unreachable"
}
```

Operation enum: `delete | stop` (see `eventlog.SandboxOp*`).

### `heartbeat.miss`

```json
{
  "target": "run",
  "attempt_no": 1,
  "last_success_at": "2026-05-07T12:34:56.789Z",
  "error_code": "",
  "message": "db locked"
}
```

Target enum: `run | checkpoint`. AttemptNo is currently always 1; future
work may aggregate consecutive misses into one event.

### `checkpoint.failure`

```json
{
  "phase": "running",
  "step": "save_phase",
  "error_code": "",
  "message": "io error"
}
```

Step enum: `save_phase | save_step`.

### `retry.attempt`

Reserved for Phase 2 retry classification.

```json
{
  "operation": "execute",
  "attempt_no": 1,
  "max_attempts": 3,
  "reason": "network_transient"
}
```

## Adding a new event

1. Add the constant to `domain/types.go` (next to the other typed
   `EventType*` constants) and extend `RunEventType.IsTypedOperationalEvent`.
2. Define the payload struct in `eventlog/types.go`. Implement the
   private `payloadMarker()` method (compile-time check by adding it to
   the `Payload` interface set in the same file).
3. Extend `eventlog.EventTypeOf` to return your constant for both the
   value and pointer shapes of the struct.
4. Register the entry in `eventlog/dispatch.go::init()` at
   `SchemaVersionDefault` (currently 1).
5. Add a typed-event helper to `phases/emitters.go`
   (`EmitYourEvent(ctx, deps, runID, payload)`).
6. Update `eventlog/repository.go::typedEventTypes` so `SinceForRun` /
   `SinceID` return rows of your new type.
7. Add the entry to the table in this document.
8. Add a round-trip case in `eventlog/eventlog_test.go::TestRoundTrip_PerEventType`.

## Adding a new schema version for an existing event

1. Define a new payload struct in `eventlog/types.go`. Do **not** delete
   the old struct — old rows still need to decode through it.
2. Register the new struct in `dispatch.go::init()` at the higher
   `schema_version`. The old registration stays.
3. Update `eventlog.EventTypeOf` to map the new struct value/pointer
   shapes to the same `RunEventType` constant.
4. Update emit helpers in `phases/emitters.go` to take the new payload
   shape; emitters always write at the latest registered version (via
   `LatestSchemaVersion`).
5. Document the change in this file with the version number, the
   reason, and the old/new JSON shapes.

## Reading typed events

Stats, health, and the UI read events through
`eventlog.SQLiteRepository`. The repository:

- filters to typed-operational event types only (see
  `RunEventType.IsTypedOperationalEvent` and
  `eventlog.typedEventTypes`),
- decodes the row's `data` JSON through the dispatch table,
- returns `[]eventlog.Record` with a typed `Payload` field consumers
  type-assert against the concrete struct.

Consumers must not parse log strings to recover operationally-significant
signals. If you find yourself wanting to grep `LogEventData.Message`,
the right answer is to add a typed event.

## DOC references

- `docs/internal/SEAMS.md` — eventlog seam entry; fallback insights seam entry.
- `docs/internal/ERROR_SEMANTICS.md` — Phase 2 extends with the
  `fallback.Reason` enum referenced by event payloads.
- `internal/eventlog/dispatch.go` — authoritative registry.
- `internal/eventlog/types.go` — payload struct definitions.
- `internal/adapters/database/repository_fallback_insights.go` — the fallback
  and policy-candidate events the `FallbackInsights` measure counts.

## Phase 3 read surfaces (added 2026-05-07)

Three HTTP/CLI surfaces consume the typed event log:

- `MeasuresService.FallbackInsights` — fallback frequency and policy-candidate
  outcomes counted from the retained events of a window.
- `GET /api/v1/health/{models,runners,audit}` — current snapshot and
  paginated audit history out of `model_health_audit` and
  `runner_health_audit` (Phase 2 tables).
- `GET /api/v1/events` — direct typed-event reads via
  `eventlog.Repository` with optional `?run=&type=&since=&limit=` filters.

CLI parity:

- `agent-manager measures fallback-insights`
- `agent-manager health {models|runners|audit}`
- `agent-manager events list [--run --type --since --limit]`

All commands follow the project default of human output with `--json`
opt-in for scripting (per `feedback_cli_default_human_output`).
