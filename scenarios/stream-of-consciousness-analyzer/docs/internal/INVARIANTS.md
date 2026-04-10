# Invariants

Source of truth: the code. This document catalogs idempotency, replay safety, and state invariants.

## Replay/Idempotency Invariants

### Safe to Replay (Idempotent)

| Operation | Why Safe |
|-----------|----------|
| All GET endpoints | Read-only, no side effects |
| Schema migration (`ensureSchema`) | `CREATE TABLE IF NOT EXISTS` — no-op on second run |
| DELETE endpoints | Deleting a non-existent resource returns 404, not a corrupt state |
| PUT (update) endpoints | Applies the same state; `updated_at` refreshes but data is consistent |

### NOT Safe to Replay (Create Side Effects)

| Operation | Behavior on Replay | Mitigation |
|-----------|-------------------|------------|
| POST /schemes | Duplicate scheme created (new UUID) | No idempotency key. Clients should deduplicate. |
| POST /thoughts | Duplicate thought created (new UUID) | No idempotency key. Clients should deduplicate. |
| POST /thoughts/{id}/edges | Returns 409 Conflict (UNIQUE constraint on source_id, target_id) | DB constraint prevents duplicate edges. Partial idempotency. |
| POST /schemes/{id}/information | Duplicate information item created | No idempotency key. |

### Future Consideration: Idempotency Keys

POST endpoints do not accept `Idempotency-Key` headers. For distributed deployments with retry-prone networks, adding server-side idempotency tracking (e.g., Redis-backed key dedup) would prevent duplicate creates. Current single-instance deployment makes this low priority.

## Data Integrity Invariants

### Enforced by Database

| Invariant | Mechanism |
|-----------|-----------|
| No duplicate edges (same source→target) | `UNIQUE(source_id, target_id)` constraint |
| No orphaned edges | `ON DELETE CASCADE` on `thought_edges.source_id` and `target_id` FK |
| No orphaned thoughts | `thoughts.scheme_id` is nullable, `ON DELETE SET NULL` on FK |
| Valid timestamps | `created_at DEFAULT NOW()`, `updated_at` refreshed via `COALESCE` in UPDATE |
| UUID uniqueness | `gen_random_uuid()` at DB level |

### Enforced by Application

| Invariant | Mechanism | Location |
|-----------|-----------|----------|
| No self-loop edges | `target_id != source_id` check | `handleCreateEdge` in handlers.go |
| Non-empty target_id | Validation before DB insert | `handleCreateEdge` in handlers.go |
| Non-empty scheme name on update | Validation before DB update | `handleUpdateScheme` in handlers.go |
| Request time-bounded | `requestTimeoutMiddleware` (30s) | main.go |

## UI State Invariants

| Invariant | Mechanism |
|-----------|-----------|
| Link mode exits on mutation completion | `onSettled` callback in `linkMut` clears `linkSource` |
| Drag state cleared on scheme switch | `useEffect([schemeId])` resets drag, pan, zoom |
| Drag listeners cleaned up on unmount | `dragCleanupRef` and `panCleanupRef` called in cleanup effect |
| Edge query key stable across thought reorders | Thought IDs sorted before joining in query key |
| Only one error banner shown at a time | First non-null error from mutation chain displayed |

## Error Classification & Retry Safety

| Category | HTTP Status | Retryable | Safe to Retry? |
|----------|------------|-----------|----------------|
| Validation | 400/422 | No | Yes (no side effect on bad input) |
| NotFound | 404 | No | Yes (read-only check) |
| Conflict | 409 | No | Depends (edge creation is safe, others may not be) |
| Dependency | 503 | Yes | Yes (transient external failure) |
| Internal | 500 | No | Unknown (server error, may or may not be safe) |
