# Temporal Flows & Async Operations

This document maps time-based behavior, async operations, and lifecycle patterns in the development-toolchain-validator scenario.

## Async Operation Inventory

### Backend (Go API)

| Operation | Type | Trigger | Duration | Recovery |
|-----------|------|---------|----------|----------|
| Database connect | Blocking startup | `database.Connect()` | Retry with backoff | Fatal on exhaustion |
| Health check | Sync request | `GET /health` | <100ms | Returns degraded status |
| Reference CRUD | Sync request | HTTP endpoints | <50ms typical | Returns error response |
| Path validation | Sync I/O | `os.Stat()` | <10ms | Returns validation error |
| Graceful shutdown | Signal handling | SIGTERM/SIGINT | Up to configured timeout | Force-kills after timeout |

**Key Observation**: All API operations are **synchronous request-response**. No background jobs, polling, or async processing exists yet.

### Frontend (React UI)

| Operation | Type | Trigger | Interval/Duration | Recovery |
|-----------|------|---------|-------------------|----------|
| Health check | React Query | Mount + interval | 30s refetch | Shows "Disconnected" |
| Fetch references | React Query | Health success | On demand | Shows error state |
| Iframe bridge init | Sync startup | Before React mount | Once | Idempotency guard |

**Key Observation**: React Query handles caching, deduplication, and refetching. The health check has a 30-second polling interval (`refetchInterval: 30000`).

### CLI

| Operation | Type | Trigger | Duration | Recovery |
|-----------|------|---------|----------|----------|
| Port detection | Startup probe | CLI init | <500ms | Falls back to env/default |
| API requests | Sync HTTP | Command execution | Up to timeout | Prints error, exits 1 |
| Stale binary detection | On init | CLI startup | <100ms | Warns user to rebuild |

## Ordering Assumptions

### Stable Assumptions (Safe)

1. **Database init before server start**: `database.Connect()` completes before `server.Run()` accepts connections
2. **Health check before data fetch**: UI waits for `healthQuery.isSuccess` before fetching references (`enabled: healthQuery.isSuccess`)
3. **Bridge init before React mount**: Iframe bridge sets up storage shims before any component renders
4. **Validation before persistence**: `ValidateCreate()`/`ValidateUpdate()` runs before database write

### Potentially Fragile Assumptions

1. **Single-request operations**: All CRUD operations assume single concurrent request. No optimistic locking or conflict detection.
2. **Path existence at create-time**: Path is validated once at create/update. Later filesystem changes go undetected.
3. **Health polling vs data freshness**: 30-second health interval means UI could show "Connected" while API is actually down for up to 30s.

## Race Conditions Identified

### API Layer

| Location | Risk | Severity | Mitigation Status |
|----------|------|----------|-------------------|
| Slug uniqueness | Time-of-check-to-time-of-use (TOCTOU) | Low | Database constraint catches at write |
| Path existence | TOCTOU - path could be deleted after validation | Low | Acceptable trade-off (documented) |
| Reference update | Lost update if two clients update same reference | Medium | **Not mitigated** |
| Reference delete | Delete during update | Low | One will fail with ErrNotFound |

**Recommendation**: For lost update scenario, consider adding version/ETag-based optimistic concurrency in a future phase.

### UI Layer

| Location | Risk | Severity | Mitigation Status |
|----------|------|----------|-------------------|
| Double-click create | Multiple create requests | Low | UI doesn't have create form yet |
| Rapid refresh | Multiple concurrent fetches | None | React Query dedupes |
| Stale read | Data changed since last fetch | Low | Manual refresh button available |

## Initialization Sequence

### API Server Startup

```
1. preflight.Run() → Check if rebuild needed, re-exec if so
   ↓
2. database.Connect() → Retry with backoff
   ↓
3. NewServer() → Wire services
   ↓
4. setupRoutes() → Register middleware + handlers
   ↓
5. server.Run() → Start HTTP listener
   ↓
6. → Ready to serve requests
```

**Stability**: Well-sequenced. `database.Connect` blocks until DB is ready or retries exhausted.

### UI Startup

```
1. Check iframe context → window.parent !== window
   ↓
2. Init iframe bridge (if iframe) → idempotency guard
   ↓
3. ReactDOM.createRoot() → Mount React tree
   ↓
4. QueryClientProvider → React Query context
   ↓
5. App mount → Health query starts
   ↓
6. Health success → References query enabled
   ↓
7. → UI ready
```

**Stability**: Good sequencing. Idempotency guard prevents double bridge init on fast remounts.

### CLI Startup

```
1. cliapp.NewScenarioApp() → Load env, detect ports
   ↓
2. Build info check → Stale binary warning
   ↓
3. Command parse → Route to handler
   ↓
4. Execute → API call or local operation
```

## Teardown Patterns

### API Server Shutdown

```
1. Signal received (SIGTERM/SIGINT)
   ↓
2. server.Run stops accepting new connections
   ↓
3. In-flight requests drain (up to timeout)
   ↓
4. Cleanup callback → db.Close()
   ↓
5. Process exits
```

**Assessment**: Graceful shutdown is handled by `api-core/server`. No explicit teardown of other resources needed since there are no background goroutines or long-lived connections beyond the DB pool.

### UI Teardown

```
1. Browser navigates away / iframe unloads
   ↓
2. React unmounts
   ↓
3. React Query cancels pending requests
```

**Assessment**: No explicit cleanup needed. React Query handles request cancellation. No timers or subscriptions to clean up.

## Polling & Retry Behavior

### Current Polling

| Component | Interval | Purpose |
|-----------|----------|---------|
| Health check (UI) | 30s | Connectivity monitoring |

### Retry Behavior

| Component | Retry Strategy | Limits |
|-----------|---------------|--------|
| Database connect | Exponential backoff | api-core default |
| React Query | Default (3 retries) | Built-in |
| CLI HTTP | No retry | Single attempt |

**Recommendation**: CLI could benefit from retry on transient errors (503), but current single-attempt is acceptable for dev tool usage.

## Checkpoint Flows

*For Progress Continuity & Interruption Resilience skill*

### Progress-Based Journeys

Currently, the scenario has **no multi-step workflows** requiring checkpointing. All operations are atomic:

| Operation | Steps | Checkpoint Needed? |
|-----------|-------|-------------------|
| Create reference | 1 (atomic) | No |
| Update reference | 1 (atomic) | No |
| Delete reference | 1 (atomic) | No |
| List references | 1 (atomic) | No |

### Future Checkpoint Opportunities

When the following features are implemented, they may need checkpointing:

1. **Batch validation runs** - Running validation across multiple reference scenarios
2. **Report generation** - Aggregating results from multiple validation runs
3. **Skill connection setup** - Multi-step wizard for connecting skills to references

### Resume Entrypoints

Currently not needed. When multi-step flows are added:
- Store progress in database with explicit phase/step markers
- Use job/task IDs for resumption lookups
- Provide clear "resume from step X" capability

## Known Issues & Hardening Needs

### Timing Issues

1. **No request timeout on API calls**: API handlers rely on client timeout. Consider adding server-side timeouts via context.
2. **Path validation race**: Documented in SEAMS.md, acceptable trade-off.
3. **No retry for CLI**: Single attempt; acceptable for dev tooling.

### Concurrency Issues

1. **Lost update scenario**: Two concurrent updates to same reference could lose one's changes. Consider ETag/version-based optimistic locking for future work.

## Relationship to Other Documents

| Document | What It Covers |
|----------|----------------|
| SEAMS.md | Integration boundaries, decision points |
| INVARIANTS.md | Replay safety, idempotency guarantees |
| COHERENCE-NOTES.md | React state architecture |

---

*Last updated: 2026-03-11 by Ecosystem Manager*
*Code is source of truth - verify claims against actual implementation*
