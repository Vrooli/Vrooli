# Implementation Plan: Concurrency Controls, Rate Limiting, and Cost Awareness

## 1. Purpose

Add execution governance to swarm-manager that prevents runaway agent spawning, enforces configurable concurrency limits, provides circuit-breaker protection for repeatedly failing items, and offers basic cost awareness before execution.

## 2. Required Reading

```bash
prompt-manager skill read api-steer cli-steer test implementation-plan-authoring
```

**Key source files:**
- `scenarios/swarm-manager/api/internal/settings/handler.go` — Settings struct, normalization, proto mapping, HTTP handlers
- `scenarios/swarm-manager/api/internal/execution/service.go` — `QueueBacklog()`, `startLocked()`, `ProcessScheduledStarts()`, `refreshRunningLocked()`
- `scenarios/swarm-manager/api/internal/execution/model.go` — Status/Mode types, Record struct, Policy struct
- `scenarios/swarm-manager/api/internal/execution/handler.go` — HTTP handlers, `StartScheduler()`
- `scenarios/swarm-manager/api/internal/overview/service.go` — `GetOverview()`, `OverviewResponse`
- `packages/proto/schemas/swarm-manager/v1/domain/settings.proto` — Settings message
- `packages/proto/schemas/swarm-manager/v1/api/settings.proto` — UpdateSettingsRequest
- `scenarios/swarm-manager/ui/src/surfaces/graph/components/GraphWorkspace.tsx` — HUD bar layout
- `scenarios/swarm-manager/cli/cmd_execution.go` — Execution CLI commands

## 3. Problem Statement

Swarm-manager has zero concurrency controls. In yolo mode, every ready backlog item can spawn an agent simultaneously, potentially:
- Exhausting API credits (each agent uses Claude Code, which costs real money)
- Overwhelming system resources (CPU, memory, file I/O)
- Creating agent thrashing loops where a broken item keeps getting retried
- Growing an unbounded queue when batch operations queue many items at once

The system needs configurable guardrails that are consistent with the existing settings architecture (proto → Go struct → normalization → HTTP API → UI).

## 4. Scope

### In Scope
- 6 new settings: `maxConcurrentExecutions`, `maxQueueDepth`, `circuitBreakerThreshold`, `circuitBreakerCooldownMinutes`, `executionCostCapPerRun`, `costPerTurnEstimate`
- Concurrency enforcement in execution service (gate `startLocked()`)
- Queue depth enforcement in `QueueBacklog()`
- Circuit breaker state tracking per backlog item (dedicated state file)
- Explicit circuit breaker reset via dedicated API endpoint and CLI command
- Governance status in overview endpoint
- Settings UI additions (Execution tab)
- Concurrency count badge in AgentsDropdown trigger ('N/M' format)
- Circuit-broken visual indicators in sidebar/graph
- CLI parity for new status visibility

### Out of Scope
- Hard cost kill (actual token metering) — only soft estimates
- Per-agent rate limiting with automatic lockout (app-issue-tracker pattern) — future enhancement
- Priority-based queue ordering — items start FIFO
- Multi-server coordination — single server only

## 5. Current Technical Context

### Settings Pipeline
1. **Proto** (`settings.proto`): Defines `Settings` message with buf.validate constraints. Currently 24 fields (field numbers 1-24, with 2-4 reserved).
2. **Go struct** (`handler.go:Settings`): Mirrors proto with JSON tags. 24 fields covering execution defaults, workshop automation, agent behavior, UI preferences, and review thresholds. `normalizeSettings()` clamps values to valid ranges. `validateSettings()` checks enums.
3. **Patch struct** (`handler.go:SettingsPatch`): All pointer-to-type fields for null-aware sparse updates.
4. **Proto mapping**: `settingsToProto()` / `settingsPatchFromProto()` convert between Go and proto types.
5. **HTTP**: `GET/PUT /api/v1/settings` with proto-JSON encoding.
6. **API proto** (`api/settings.proto`): `UpdateSettingsRequest` with all optional fields.

### Execution Service
- `QueueBacklog()` (line ~238): Validates inputs, checks preflight, creates `Record` with `StatusPending`. In yolo mode, immediately calls `startLocked()` and rolls back on failure.
- `startLocked()` (line ~461): Loads record, checks agent availability, builds prompt, spawns agent. Transitions to `StatusStarting`.
- `ProcessScheduledStarts()` (line ~770): 2-second ticker, iterates records for `StatusScheduled`, calls `startLocked()` when due. Fire-and-forget (returns nil regardless).
- `refreshRunningLocked()`: Polls agent-manager for status updates on running executions.
- All methods operate under `s.mu.Lock()` (single mutex).

### Key Integration Points
- `startLocked()` is the choke point — all execution starts flow through it (yolo, manual, scheduled, retry, fixup).
- `QueueBacklog()` is where queue depth enforcement should go.
- `ProcessScheduledStarts()` is the existing poller that will be renamed to `ProcessPendingStarts()` and expanded to drain pending items after processing scheduled starts.
- The overview service aggregates backlog items into `OverviewResponse` with items, initiatives, dependency graph, and summary. Has no execution awareness currently.

### UI HUD Bar
- Right section: Settings gear → Help button → AgentsDropdown (shows running activities)
- AgentsDropdown already owns the "running agents" concept — confirmed as the integration point for concurrency count badge
- StatusBadge.tsx: colored dots (cyan=running, gray=pending, amber=review, red=failed)

## 6. Target End State

### Settings
| Setting | Type | Range | Default | Proto Field # |
|---------|------|-------|---------|---------------|
| `maxConcurrentExecutions` | int32 | 1-20 | 3 | 25 |
| `maxQueueDepth` | int32 | 0-100 (0=unlimited) | 50 | 26 |
| `circuitBreakerThreshold` | int32 | 1-10 | 3 | 27 |
| `circuitBreakerCooldownMinutes` | int32 | 5-1440 | 60 | 28 |
| `executionCostCapPerRun` | double | 0.0+ (0=unlimited) | 0.0 | 29 |
| `costPerTurnEstimate` | double | 0.00-5.00 | 0.10 | 30 |

### Enforcement Behavior

**Concurrency**: `startLocked()` counts active executions where active = `StatusStarting` + `StatusRunning` only (tightest definition — slots open as soon as agent work finishes, even if pending review). If active count >= `maxConcurrentExecutions`, returns a sentinel error. The caller handles gracefully — yolo mode leaves the item as pending, manual mode returns error to user.

**Queue depth**: `QueueBacklog()` counts pending + scheduled records. Rejects with HTTP 409 and clear error when limit exceeded.

**Circuit breaker**: Dedicated state file at `.vrooli/circuit-breaker.json`:
```json
{
  "items": {
    "execute/broken-thing": {
      "consecutive_failures": 3,
      "last_failure": "2026-03-30T12:00:00Z",
      "broken_at": "2026-03-30T12:00:00Z"
    }
  }
}
```
On execution failure: increment counter. On threshold breach: set `broken_at`. On queue/retry of broken item: check cooldown, block if broken and cooldown not expired. On explicit reset (via dedicated API/CLI): clear the breaker entry for that item regardless of cooldown. On cooldown expiry: auto-reset (checked in poller).

**Cost cap**: Before starting, estimate = `costPerTurnEstimate * agentMaxTurns`. With default values: $0.10 * 60 = $6.00. If `executionCostCapPerRun > 0` and estimate > cap: require `force: true` or return HTTP 409 with cost warning.

### Governance Status
Extend overview response with:
```json
{
  "governance": {
    "active_executions": 2,
    "max_concurrent": 3,
    "queue_depth": 5,
    "max_queue_depth": 50,
    "circuit_broken_items": ["execute/broken-thing"],
    "estimated_queued_cost": 12.50
  }
}
```

## 7. Implementation Strategy

### Phase 1: Settings Foundation
1. Add 6 new fields to `settings.proto` (field numbers 25-30)
2. Add fields to Go `Settings` struct, `SettingsPatch`, defaults, normalization, proto mapping
3. Add fields to `api/settings.proto` `UpdateSettingsRequest`
4. Regenerate proto
5. Add settings UI controls to Execution tab

### Phase 2: Concurrency Enforcement
1. Add `countActiveExecutions()` helper to execution service — counts records with status `starting` or `running` only
2. Gate `startLocked()` — if active count >= `maxConcurrentExecutions`, return a sentinel error (e.g., `errAtCapacity`)
3. Modify `QueueBacklog()` yolo path: on `errAtCapacity`, leave record as `StatusPending` instead of rolling back
4. Rename `ProcessScheduledStarts()` to `ProcessPendingStarts()`. After the existing scheduled-start loop, add a second loop over pending records sorted by `CreatedAt`. Check concurrency limit before each start. Scheduled starts have priority over pending items.
5. Add concurrency tests

### Phase 3: Queue Depth Enforcement
1. Add `countQueuedExecutions()` helper (counts pending + scheduled)
2. Add queue depth check at top of `QueueBacklog()`, before record creation
3. Return clear error: "queue depth limit exceeded (N/M)"
4. Add queue depth tests

### Phase 4: Circuit Breaker
1. Add circuit breaker service managing `.vrooli/circuit-breaker.json` with atomic writes
2. On execution failure: increment counter. On threshold breach: set `broken_at`.
3. On `QueueBacklog()`: check circuit breaker state. Block if broken and cooldown not expired.
4. Add explicit reset endpoint: `POST /api/v1/execution/circuit-breaker/reset` with `{"item": "kind/name"}`. Clears the breaker entry for that item regardless of cooldown.
5. Add CLI command: `swarm-manager execution circuit-breaker-reset <kind/name>`.
6. On cooldown expiry: auto-reset (checked in poller).
7. Add circuit breaker tests

### Phase 5: Cost Awareness
1. `costPerTurnEstimate` setting: range 0.00-5.00 $/turn, default 0.10
2. Before starting: estimate = `costPerTurnEstimate * agentMaxTurns`
3. If `executionCostCapPerRun > 0` and estimate > cap: require `force: true` or return error
4. Track estimated cost in governance status for queued items
5. Add cost estimation tests

### Phase 6: Status Visibility
1. Extend `OverviewResponse` with `Governance` field
2. Populate from execution service + circuit breaker state + settings
3. Update CLI overview command to display governance info
4. Add concurrency count badge to AgentsDropdown trigger — display 'N/M' format (e.g., '2/3') with colored indicator matching current status semantics
5. Add circuit-broken visual indicator to sidebar items and graph nodes

## 8. Contract Decisions

### API Changes
- `GET /api/v1/settings` — returns 6 new fields
- `PUT /api/v1/settings` — accepts 6 new optional fields
- `GET /api/v1/overview` — adds `governance` object to response
- `POST /api/v1/execution` — returns HTTP 409 when queue depth exceeded, circuit-broken, or cost cap exceeded
- `POST /api/v1/execution/circuit-breaker/reset` — new endpoint to explicitly reset circuit breaker for a specific item. Body: `{"item": "kind/name"}`. Returns 200 on success, 404 if item has no breaker state.

### CLI Changes
- `swarm-manager execution create` — shows governance errors clearly
- `swarm-manager overview` — displays governance section
- `swarm-manager execution circuit-breaker-reset <kind/name>` — new command to explicitly reset circuit breaker for an item

### Error Codes
- Queue depth exceeded: HTTP 409 Conflict with structured error
- Circuit broken: HTTP 409 Conflict with structured error including cooldown remaining
- Cost cap exceeded: HTTP 409 Conflict with message suggesting `--force`
- At capacity (internal): sentinel error handled by caller, not returned to user

## 9. Testing Plan

### Unit Tests
- Settings normalization for 6 new fields (clamping, defaults, edge cases)
- `countActiveExecutions()` with various status combinations — verify only starting+running count
- `countQueuedExecutions()` accuracy
- Circuit breaker state transitions (increment, breach, cooldown, reset, atomic write/read)
- Explicit circuit breaker reset clears entry regardless of cooldown
- Cost estimation calculation (costPerTurnEstimate * agentMaxTurns vs cap) — default: $0.10 * 60 = $6.00
- Concurrency gate in `startLocked()` — returns errAtCapacity when at limit
- Queue depth gate in `QueueBacklog()` — rejects when at max depth
- Yolo path at capacity: record stays pending, not rolled back

### Integration Tests
- Queue 4 items when maxConcurrent=3: verify 4th stays pending
- Complete 1 execution: verify pending item auto-starts on next poll cycle (via renamed `ProcessPendingStarts()`)
- Queue items to max depth: verify rejection with 409
- Fail item N times (N=threshold): verify circuit breaker trips
- Reset circuit-broken item via explicit reset endpoint: verify breaker clears and item can be re-queued
- Wait for cooldown: verify auto-reset
- Set cost cap below estimate: verify force required
- Update settings mid-flight: verify enforcement changes immediately (new limit applies on next start attempt)

## 10. Rollout / Validation Checklist

- [ ] Proto compiles cleanly (`buf lint && buf generate`)
- [ ] Go builds with no errors (`go build ./...`)
- [ ] All existing tests pass (`go test ./... -timeout 300s`)
- [ ] New tests pass for each phase
- [ ] Settings UI renders new controls correctly
- [ ] CLI displays governance info in overview
- [ ] Circuit breaker reset CLI command works
- [ ] Manual test: set maxConcurrent=1, queue 2 items, verify queuing behavior

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Mutex contention with concurrent slot counting | Medium | Already single-mutex design; counting is O(n) where n is small (max 20 active) |
| Circuit breaker state file corruption | Low | Use atomic writes (write-to-temp + rename, existing pattern in codebase) |
| Cost estimates wildly inaccurate | Low | Explicitly labeled as estimates, soft guardrail only, user-configurable per-turn rate ($0.10 default) |
| Existing tests break from new defaults | Medium | New settings have safe defaults (3 concurrent, 50 queue, no cost cap) |
| Poller frequency (2s) too slow for slot draining | Low | 2s is reasonable for backfill; event-driven drain is a future optimization |
| Active=starting+running misses review backlog | Low | User chose tight definition. If review floods become an issue, can expand later without breaking changes |
| Explicit reset adds API surface | Low | Clean separation of concerns worth the small surface area; rare operation but important for operator control |

## 12. Non-goals / Prohibited Patterns

- No hard token kill switches (can't know actual cost in advance)
- No per-agent rate limiting (future enhancement)
- No priority queue ordering
- No backward compatibility shims — new fields with safe defaults
- No new execution statuses — use existing pending/failed + circuit breaker state file
- No implicit circuit breaker reset on queue — reset is an explicit, intentional action via dedicated endpoint/CLI

## 13. Definition of Done

- All 6 settings configurable via API, CLI, and UI (costPerTurnEstimate defaults to $0.10/turn)
- Concurrency enforcement prevents exceeding maxConcurrentExecutions (active = starting + running)
- Yolo mode at capacity leaves items pending; `ProcessPendingStarts()` drains them when slots open (scheduled first, then pending by CreatedAt)
- Queue depth enforcement rejects when exceeded
- Circuit breaker trips after threshold failures, auto-resets after cooldown
- Explicit circuit breaker reset via `POST /api/v1/execution/circuit-breaker/reset` and `swarm-manager execution circuit-breaker-reset <kind/name>`
- Cost cap provides soft warning before execution ($0.10/turn * maxTurns vs cap), overridable with force
- Governance status visible in overview endpoint, CLI, and AgentsDropdown trigger ('N/M' badge)
- All new code has unit tests
- All existing tests continue to pass
- Proto changes are clean and backward-compatible (new field numbers only)
