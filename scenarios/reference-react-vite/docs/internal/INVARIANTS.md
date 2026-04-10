# Invariants Documentation

This document captures the system invariants, replay safety patterns, and idempotency guarantees in the reference-react-vite scenario, following the idempotency-replay-safety-hardening skill patterns.

## Overview

The reference-react-vite scenario implements a task management system with CRUD operations. This document identifies which operations are safe to replay and which require guards.

---

## Replay/Idempotency Invariants

### Operation Classification

| Operation | Idempotent | Replay-Safe | Notes |
|-----------|------------|-------------|-------|
| GET /health | ✅ Yes | ✅ Yes | Read-only |
| GET /tasks | ✅ Yes | ✅ Yes | Read-only |
| GET /tasks/{id} | ✅ Yes | ✅ Yes | Read-only |
| POST /tasks | ❌ No | ⚠️ Creates duplicates | New UUID each time |
| PATCH /tasks/{id} | ✅ Yes | ✅ Yes | Same input = same state |
| DELETE /tasks/{id} | ✅ Yes | ✅ Yes | Returns 404 on replay |
| GET /projects | ✅ Yes | ✅ Yes | Read-only |
| POST /projects | ❌ No | ⚠️ Creates duplicates | New UUID each time |
| PATCH /projects/{id} | ✅ Yes | ✅ Yes | Same input = same state |
| DELETE /projects/{id} | ✅ Yes | ✅ Yes | Returns 404 on replay |
| GET /notes | ✅ Yes | ✅ Yes | Read-only |
| POST /notes | ❌ No | ⚠️ Creates duplicates | New UUID each time |
| PATCH /notes/{id} | ✅ Yes | ✅ Yes | Same input = same state |
| DELETE /notes/{id} | ✅ Yes | ✅ Yes | Returns 404 on replay |

---

## Idempotency Keys

### Current State

**No idempotency keys are implemented.** This is acceptable for the current single-user, direct-manipulation use case.

### Where Keys Would Help

If the scenario evolves to support:
- CLI scripting with retry logic
- Multi-device sync
- Webhook delivery
- Agent automation

Then idempotency keys should be added:

```
POST /tasks
X-Idempotency-Key: <client-generated-uuid>

# Server behavior:
# 1. Check if key exists in cache/DB
# 2. If exists, return cached response
# 3. If not, execute operation, store key + response
# 4. Keys expire after TTL (e.g., 24h)
```

---

## State-Mutating Operations

### Create Operations

| Operation | Side Effects | Idempotency Status |
|-----------|--------------|-------------------|
| POST /tasks | Insert DB row | ❌ Not idempotent |
| POST /projects | Insert DB row | ❌ Not idempotent |
| POST /notes | Insert DB row, verify task exists | ❌ Not idempotent |

**Current Behavior:**
- Server generates UUID for new entities
- Same request body = different IDs
- No duplicate detection

**Mitigation:**
- UI clears form on success (prevents accidental re-submit)
- Button disabled during mutation (`isPending`)
- No auto-retry on POST failures

### Update Operations

| Operation | Side Effects | Idempotency Status |
|-----------|--------------|-------------------|
| PATCH /tasks/{id} | Update DB row, set updated_at | ✅ Idempotent |
| PATCH /projects/{id} | Update DB row, set updated_at | ✅ Idempotent |
| PATCH /notes/{id} | Update DB row, set updated_at | ✅ Idempotent |

**Current Behavior:**
- Same input produces same state
- `updated_at` changes on each call (side effect but harmless)
- No optimistic locking (last write wins)

### Delete Operations

| Operation | Side Effects | Idempotency Status |
|-----------|--------------|-------------------|
| DELETE /tasks/{id} | Remove DB row + cascade notes | ✅ Idempotent |
| DELETE /projects/{id} | Remove DB row | ✅ Idempotent |
| DELETE /notes/{id} | Remove DB row | ✅ Idempotent |

**Current Behavior:**
- First call: 204 No Content
- Replay: 404 Not Found
- Safe to retry (no duplicate side effects)

---

## Commit Boundaries

### Database Transactions

**Current State:** Single-statement operations, no explicit transactions.

| Operation | Transaction Scope | Atomicity |
|-----------|-------------------|-----------|
| Create task | Single INSERT | ✅ Atomic |
| Update task | Single UPDATE | ✅ Atomic |
| Delete task | Single DELETE | ✅ Atomic |
| Delete task + notes | **Not transactional** | ⚠️ Cascade |

**Note:** Task deletion cascades to notes via FK constraint (`ON DELETE CASCADE`), which is atomic at the DB level.

### Partial Failure States

| Scenario | Current Behavior | Risk |
|----------|------------------|------|
| Network timeout mid-create | Task may or may not exist | Low |
| Network timeout mid-update | Update may or may not apply | Low |
| Network timeout mid-delete | Resource may or may not exist | Low |
| Server crash mid-operation | DB transaction rollback | ✅ Safe |

**Recommendation:** For network failures, client should:
1. Check if resource exists (for creates)
2. Retry (for updates, deletes)
3. Surface ambiguity to user

---

## UI Replay Safety

### Form Submission

| Component | Guard | Status |
|-----------|-------|--------|
| TaskForm | `disabled={isSubmitting}` | ✅ Protected |
| ProjectForm | `disabled={isSubmitting}` | ✅ Protected |
| Status toggle | `disabled={isUpdating}` | ✅ Protected |
| Delete button | `disabled={isUpdating}` | ✅ Protected |

### Mutation Tracking

The `updatingIds` Set pattern prevents concurrent mutations on the same entity:

```tsx
// [CODE: ui/src/pages/Tasks.tsx]
const [updatingIds, setUpdatingIds] = useState<Set<string>>(new Set());

// Before mutation
onMutate: ({ id }) => {
  setUpdatingIds((prev) => new Set(prev).add(id));
}

// After mutation (success or failure)
onSettled: (_, __, { id }) => {
  setUpdatingIds((prev) => {
    const next = new Set(prev);
    next.delete(id);
    return next;
  });
}
```

---

## Safe Retry Patterns

### API Error Retryability

From [CODE: api/handlers/errors.go]:

| Error Code | HTTP Status | Retryable |
|------------|-------------|-----------|
| `BAD_REQUEST` | 400 | ❌ No |
| `VALIDATION_ERROR` | 422 | ❌ No |
| `NOT_FOUND` | 404 | ❌ No |
| `INTERNAL_ERROR` | 500 | ✅ Yes |
| `CONFLICT` | 409 | ❌ No |
| `UNAUTHORIZED` | 401 | ❌ No |

### React Query Retry Behavior

Default: 3 retries with exponential backoff for failed requests.

**Safe for:**
- GET requests (read-only)
- Network failures on POST (will create duplicates but user initiated)

---

## Invariants to Maintain

### Data Integrity Invariants

| Invariant | Enforced By | Status |
|-----------|-------------|--------|
| Task ID is unique | UUID + DB primary key | ✅ Guaranteed |
| Project ID is unique | UUID + DB primary key | ✅ Guaranteed |
| Note ID is unique | UUID + DB primary key | ✅ Guaranteed |
| Note belongs to valid task | FK constraint | ✅ DB enforced |
| Status is valid enum | Domain validation | ✅ Code enforced |
| Priority in range [1-5] | Domain validation | ✅ Code enforced |

### Temporal Invariants

| Invariant | Enforced By | Status |
|-----------|-------------|--------|
| created_at ≤ updated_at | DB trigger | ✅ DB enforced |
| created_at never changes | No UPDATE on column | ✅ By design |
| updated_at updates on change | DB trigger | ✅ DB enforced |

### Ordering Invariants

| Invariant | Enforced By | Status |
|-----------|-------------|--------|
| List order is deterministic | ORDER BY created_at DESC | ✅ Query enforced |

---

## Risky Operations

### Operations That Need Extra Care

1. **Bulk operations** (not yet implemented)
   - Would need transactional boundaries
   - Consider partial success handling

2. **Cascade deletes**
   - Task deletion removes all notes
   - Currently atomic via FK cascade
   - If batch delete added, consider explicit transaction

3. **Status transitions** (future)
   - If workflow rules added (e.g., pending→in_progress only)
   - Would need optimistic locking for concurrent edits

---

## Recommendations

### Short-Term

1. ✅ Current state is acceptable for single-user CRUD
2. Consider adding delete confirmation modal
3. Consider debouncing rapid create clicks

### Medium-Term (Multi-User/CLI Automation)

1. Add `X-Idempotency-Key` header support for POST
2. Add ETag/version field for optimistic locking
3. Surface `retryable` hint in UI error messages

### Long-Term (Distributed/Webhook)

1. Implement at-least-once delivery with deduplication
2. Add event log for audit trail
3. Consider saga pattern for multi-step operations

---

## Related Documentation

- [DOC: docs/internal/TEMPORAL-FLOWS.md] - Async operation patterns
- [DOC: docs/internal/ERROR_SEMANTICS.md] - Error handling
- [DOC: docs/internal/SEAMS.md] - Architectural boundaries

---

## Last Updated

2026-03-11 - Initial invariants audit (Phase 12)
