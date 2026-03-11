# Invariants & Replay Safety

This document catalogs system invariants, idempotency guarantees, and replay safety patterns for the development-toolchain-validator scenario.

## Replay/Idempotency Invariants

### State-Mutating Operations

| Operation | Idempotent? | Key | Double-Apply Behavior |
|-----------|-------------|-----|----------------------|
| Create Reference | No | slug | Second attempt returns 409 Conflict |
| Update Reference | Yes* | id + fields | Same result on replay |
| Delete Reference | Yes* | id | Second attempt returns 404 Not Found |
| List References | Yes | N/A | Read-only |
| Get Reference | Yes | id or slug | Read-only |
| Dry-run Create | Yes | N/A | No side effects |
| Dry-run Update | Yes | N/A | No side effects |
| Dry-run Delete | Yes | N/A | No side effects |

*Update and Delete are **naturally idempotent** for identical inputs, but not **exactly-once** - there's no mechanism to distinguish "already applied" from "applying now".

### Idempotency Analysis by Operation

#### Create Reference

**Current Behavior**:
```
Request 1: POST /references {slug: "foo"} → 201 Created
Request 2: POST /references {slug: "foo"} → 409 Conflict (ErrSlugExists)
```

**Status**: ✅ Safe - Database uniqueness constraint prevents duplicates.

**Improvement Opportunity**: Add idempotency key header (`Idempotency-Key`) for exactly-once semantics, allowing clients to safely retry without knowing if the first request succeeded.

#### Update Reference

**Current Behavior**:
```
Request 1: PATCH /references/{id} {name: "New"} → 200 OK
Request 2: PATCH /references/{id} {name: "New"} → 200 OK (same result)
```

**Status**: ✅ Naturally idempotent - applying same update twice produces same state.

**Edge Case**: If concurrent updates from different clients, last-write-wins (no optimistic locking).

#### Delete Reference

**Current Behavior**:
```
Request 1: DELETE /references/{id} → 204 No Content
Request 2: DELETE /references/{id} → 404 Not Found
```

**Status**: ⚠️ Idempotent by outcome (reference is deleted) but response differs on replay.

**Recommendation**: For strict idempotency, could return 204 even if already deleted. Current behavior is acceptable and common.

### Idempotency Keys

Currently **not implemented**. Operations that would benefit:

| Operation | Priority | Rationale |
|-----------|----------|-----------|
| Create Reference | Medium | Allows safe retry without duplicate detection logic |
| Batch operations | High | When implemented, prevent partial double-apply |

**Implementation Pattern** (for future):
```go
// Header: Idempotency-Key: <uuid>
// Store: idempotency_keys table with key, response, created_at
// TTL: 24 hours
```

## Safe vs Risky Retry Patterns

### Safe to Retry

| Operation | Why Safe |
|-----------|----------|
| GET any endpoint | Read-only |
| Dry-run any operation | No side effects |
| Update with same data | Naturally idempotent |
| Delete (ignore 404) | Outcome is "deleted" either way |

### Risky to Retry

| Operation | Risk | Mitigation |
|-----------|------|------------|
| Create without idempotency key | Duplicate creation | Check for 409 response |
| Update without version check | Lost concurrent changes | Accept last-write-wins |
| Any mutating op on network timeout | Unknown state | Query current state before retry |

### Retry Decision Matrix

```
┌─────────────────────────────────────────────────────────────┐
│ Response Received?                                          │
├──────────────┬──────────────────────────────────────────────┤
│ Yes          │ Check status code:                          │
│              │ - 2xx: Success, no retry needed              │
│              │ - 4xx: Client error, don't retry             │
│              │ - 503: Transient, safe to retry              │
│              │ - 500: Maybe transient, retry with caution   │
├──────────────┼──────────────────────────────────────────────┤
│ No (timeout) │ Operation may or may not have applied.       │
│              │ - Reads: Safe to retry                       │
│              │ - Create: Query first, then retry if needed  │
│              │ - Update: Query first, compare state         │
│              │ - Delete: Query first, retry if still exists │
└──────────────┴──────────────────────────────────────────────┘
```

## Commit Boundaries

### Atomic Operations

All CRUD operations are **atomic at the database level**:

| Operation | Transaction | Atomicity |
|-----------|-------------|-----------|
| Create | Single INSERT | Atomic |
| Update | Single UPDATE | Atomic |
| Delete | Single DELETE | Atomic |
| List | Single SELECT | Snapshot |

**No partial commits** are possible in the current implementation.

### Future Concerns

When multi-entity operations are added (e.g., "validate all references"), ensure:
1. Use database transactions for multi-row operations
2. Document commit points clearly
3. Provide recovery guidance for interrupted operations

## Deterministic Computations

### Currently Deterministic

| Computation | Input | Output |
|-------------|-------|--------|
| Slug validation | slug string | bool (pass/fail) |
| Path resolution | relative path | absolute path |
| Error mapping | domain error | HTTP status + message |
| Pagination limits | requested limit | applied limit |

### Currently Non-Deterministic

| Computation | Why | Impact |
|-------------|-----|--------|
| UUID generation (Create) | `uuid.New()` | Different ID each create |
| Timestamp (CreatedAt/UpdatedAt) | `time.Now()` | Different on replay |

**Assessment**: Non-determinism is acceptable here. IDs and timestamps should differ on replay.

## Global State Dependencies

### Backend

| State | Scope | Impact |
|-------|-------|--------|
| Database connection | Process | Single pool, thread-safe |
| Config | Process | Loaded once at startup, immutable |
| Slug regex | Package | Compiled once, immutable |

### Frontend

| State | Scope | Impact |
|-------|-------|--------|
| QueryClient | App | Shared cache, React Query manages |
| Bridge init flag | Window | Prevents double init |
| API_BASE | Module | Resolved once at load |

### CLI

| State | Scope | Impact |
|-------|-------|--------|
| HTTP client | App instance | Created per command run |
| Port detection | App instance | Cached for session |

## Recovery Patterns

### Partial Failure Recovery

Currently, no operations can partially fail (all atomic). When batch operations are added:

1. **Record progress** in database before each step
2. **Check progress** on resume
3. **Skip completed steps** using progress record
4. **Clean up** if needed (rollback or mark failed)

### Network Failure Recovery

| Scenario | Recovery Action |
|----------|-----------------|
| Timeout on GET | Retry immediately |
| Timeout on Create | Query by slug, create if not found |
| Timeout on Update | Query by ID, compare state, update if needed |
| Timeout on Delete | Query by ID, delete if exists |
| 503 Service Unavailable | Retry with backoff |

## Testing Replay Safety

### Test Patterns to Include

```go
// Test: Create is not idempotent
func TestCreateReference_ReplayReturnsConflict(t *testing.T) {
    // Create first
    ref1, err := service.Create(ctx, input)
    require.NoError(t, err)

    // Replay should fail
    _, err = service.Create(ctx, input)
    assert.ErrorIs(t, err, reference.ErrSlugExists)
}

// Test: Update is idempotent
func TestUpdateReference_ReplayProducesSameState(t *testing.T) {
    // Update first
    ref1, err := service.Update(ctx, id, input)
    require.NoError(t, err)

    // Replay should succeed with same result
    ref2, err := service.Update(ctx, id, input)
    require.NoError(t, err)
    assert.Equal(t, ref1.Name, ref2.Name)
}

// Test: Delete is idempotent by outcome
func TestDeleteReference_ReplayIsSafe(t *testing.T) {
    // Delete first
    err := service.Delete(ctx, id)
    require.NoError(t, err)

    // Replay returns ErrNotFound (acceptable)
    err = service.Delete(ctx, id)
    assert.ErrorIs(t, err, reference.ErrNotFound)

    // Either way, reference is deleted
    _, err = service.GetByID(ctx, id)
    assert.ErrorIs(t, err, reference.ErrNotFound)
}
```

**Status**: These patterns should be added to service_test.go for explicit replay safety coverage.

## Summary: Invariants Checklist

| Invariant | Status | Notes |
|-----------|--------|-------|
| No duplicate references by slug | ✅ Enforced | Database UNIQUE constraint |
| All operations atomic | ✅ Enforced | Single SQL statements |
| Reads are idempotent | ✅ Enforced | GET operations are pure |
| Dry-run has no side effects | ✅ Enforced | Validation only |
| Updates are naturally idempotent | ✅ Enforced | Same input → same state |
| Deletes are safe to retry | ✅ Enforced | 404 on replay is acceptable |
| Creates prevent double-apply | ✅ Enforced | 409 Conflict response |
| Optimistic concurrency | ❌ Not implemented | Future enhancement |
| Idempotency keys | ❌ Not implemented | Future enhancement |

---

*Last updated: 2026-03-11 by Ecosystem Manager*
*Code is source of truth - verify claims against actual implementation*
