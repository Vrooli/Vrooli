# System Invariants Documentation

This document describes the **invariants** (conditions that must always hold) and **idempotency guarantees** in the Lifestyle Dashboard scenario. Understanding these helps ensure reliable, predictable behavior under retries, failures, and concurrent access.

## Last Updated
2026-03-10 (Phase 12: Idempotency & Replay Safety Hardening)

## Replay/Idempotency Invariants

### API Operations

| Operation | Idempotent? | Mechanism | Notes |
|-----------|-------------|-----------|-------|
| POST /events | ❌ No | New UUID per call | Each call creates new event |
| POST /domains | ✅ Yes | ON CONFLICT upsert | Same name → update existing |
| PATCH /domains/{name} | ✅ Yes | Partial update | Applying same update is no-op |
| GET endpoints | ✅ Yes | Read-only | No state mutation |
| GET /domains/{name}/health | ⚠️ Side-effect | Updates domain status | Safe to call repeatedly |

### Event Creation

**NOT Idempotent**: Each POST /api/v1/events creates a new event with a server-generated UUID.

```go
// sqlite_events.go:Create
if event.ID == "" {
    event.ID = uuid.New().String()
}
```

**Client Guidance**:
- If uncertain whether a request succeeded, check recent events by domain/type/timestamp before retrying
- For truly idempotent event creation, clients should generate their own UUID and include it in the request
- Server will accept client-provided IDs (no collision check)

**Potential Improvement**: Add optional `idempotency_key` field that server uses to deduplicate within a time window.

### Domain Registration

**Idempotent**: Uses SQLite UPSERT pattern.

```sql
INSERT INTO domains (...)
VALUES (...)
ON CONFLICT(name) DO UPDATE SET
    display_name = excluded.display_name,
    ...
```

**Behavior**:
- First registration: Creates new domain, sets `registered_at`
- Subsequent: Updates mutable fields, preserves `registered_at`, updates `updated_at`

**Safe to Retry**: Yes. The final state will be identical regardless of retry count.

### Domain Status Update

**Last-Write-Wins**: Health check updates domain status unconditionally.

```go
// handlers/domains.go:GetDomainHealth
if updateErr := h.Domains.UpdateStatus(r.Context(), name, status, now); updateErr != nil {
    log.Printf("[WARN] GetDomainHealth(%s): failed to update status: %v", name, updateErr)
}
```

**Behavior**:
- Multiple concurrent health checks → last one to complete sets status
- Status update failure is logged but doesn't fail the response
- This is "fire-and-forget" - health response returns regardless of status update success

**Safe for Concurrent Access**: Yes, but status may temporarily show stale values.

### UI Bridge Initialization

**Idempotent**: Protected by window flag.

```typescript
// main.tsx
if (!window.__lifestyleDashboardBridgeInitialized) {
    initIframeBridgeChild({ ... });
    window.__lifestyleDashboardBridgeInitialized = true;
}
```

**Behavior**: Initialization runs exactly once per window, regardless of React re-renders or hot reloads.

## State Consistency Invariants

### Database Schema Invariants

| Table | Primary Key | Unique Constraints | Foreign Keys |
|-------|-------------|-------------------|--------------|
| events | id (UUID) | None | None |
| domains | name | name | None |

**Event Ordering**: Events are returned by `timestamp DESC`. The `timestamp` field is the logical event time, which may differ from `created_at` (server receipt time).

**Domain Uniqueness**: Domain name is the natural key. All operations reference domains by name.

### Timestamp Invariants

1. **Server-Generated**: All timestamps are generated server-side in UTC
2. **Format**: RFC3339 string format ("2006-01-02T15:04:05Z")
3. **Immutable**: `created_at` and `registered_at` are set once, never updated
4. **Updated Automatically**: `updated_at` is set on every mutation

```go
// Domain timestamps
if d.RegisteredAt == "" {
    d.RegisteredAt = now  // Only set on first insert
}
d.UpdatedAt = now         // Always updated
```

### Score Calculation Invariants

1. **Deterministic**: Same events → same score (no randomness)
2. **Date-Bounded**: Score is calculated for a specific date, not affected by future events
3. **Domain-Scoped**: Only "active" domains contribute to composite score
4. **Capped**: Individual domain scores are capped at 100

```go
// Scoring formula
domainScore := eventCount * 20  // Each event adds 20 points
if domainScore > 100 {
    domainScore = 100           // Cap at 100
}
compositeScore = sum(domainScores) / domainCount  // Simple average
```

## Safe/Unsafe Retry Patterns

### Safe to Retry (Idempotent)

| Operation | Notes |
|-----------|-------|
| GET any endpoint | Read-only |
| POST /domains | Upsert pattern |
| PATCH /domains/{name} | Partial update converges |
| UI page refresh | Queries are read-only |
| UI "Retry" button | Refetches all data |

### Unsafe to Retry (Non-Idempotent)

| Operation | Risk | Mitigation |
|-----------|------|------------|
| POST /events | Duplicate events | Check recent events first, or use client-side UUID |
| Domain health check | Redundant work | Acceptable, self-limiting by interval |

### Retry Behavior in UI

**React Query Defaults**:
- No automatic retry on error by default
- Manual retry via `refetch()` or "Retry" button
- Stale data shown while refetching

**Error Recovery Flow**:
```
1. Query fails
2. ErrorAlert shows with category-appropriate message
3. User clicks "Retry"
4. handleRetry() calls refetch() on all queries
5. Loading state shown
6. Success → data updates / Failure → error persists
```

## Concurrency Safety

### SQLite Single-Writer Model

The database is configured with `MaxOpenConns=1`, enforcing serial writes.

**Guarantees**:
- No concurrent write conflicts
- No dirty reads (SQLite isolation)
- WAL mode allows concurrent reads during writes

**Limitations**:
- Write throughput limited to single connection
- Long queries block other writes

### UI State Isolation

**React Query Scope**: Each query has its own cache entry and loading/error state.

**Page Component Isolation**: Each page manages its own queries independently. Navigating away cancels pending queries.

**No Global State**: No zustand/redux stores that could have race conditions.

## Recovery from Partial Execution

### API Layer

**Atomic Operations**: All repository methods complete fully or fail with error. No partial state.

```go
// Example: Event creation
_, err := r.db.ExecContext(ctx, `INSERT INTO events ...`, ...)
return err  // Either fully inserted or error
```

**No Transactions Needed**: Single-statement operations don't require explicit transactions.

### UI Layer

**Optimistic Updates**: Not currently used. UI shows server state only.

**Error Display**: Failed queries show error immediately, user can retry or navigate away.

**No Local Persistence**: UI state is ephemeral. Refresh starts fresh.

## Idempotency Key Pattern (Future)

For scenarios requiring guaranteed exactly-once event creation, consider:

```go
type CreateEventRequest struct {
    IdempotencyKey string  // Optional client-provided key
    Domain         string
    EventType      string
    // ...
}

// In handler:
if req.IdempotencyKey != "" {
    existing, err := repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
    if err == nil {
        return existing  // Already processed
    }
}
event := &domain.Event{
    IdempotencyKey: req.IdempotencyKey,
    // ...
}
```

**Implementation Notes**:
- Add `idempotency_key` column with unique index
- Keys should expire after reasonable window (e.g., 24 hours)
- Return original response for duplicate requests

## Related Documentation

- [TEMPORAL-FLOWS.md](TEMPORAL-FLOWS.md) - Async operations and timing
- [ERROR_SEMANTICS.md](ERROR_SEMANTICS.md) - Error categories and recovery
- [SEAMS.md](SEAMS.md) - Testing boundaries
- [STORAGE_AUDIT.md](STORAGE_AUDIT.md) - Database architecture decisions
