# Invariants

Documents idempotency, replay safety, and correctness invariants for the brand-manager scenario.

## Replay/Idempotency Invariants

### State-Mutating Operations

| Operation | Idempotent? | Mechanism | Notes |
|-----------|------------|-----------|-------|
| CreateBrand (no key) | No | New UUID each call | Duplicate brands on retry |
| CreateBrand (with Idempotency-Key) | Yes | In-memory cache keyed by header value | Replayed response includes `X-Idempotent-Replayed: true` |
| UpdateBrand (no If-Match) | Yes (outcome) | Last-write-wins; same input produces same state | Version increments on each call |
| UpdateBrand (with If-Match) | Yes (safe) | 409 Conflict if version mismatch | Prevents accidental overwrites |
| DeleteBrand | Yes | Returns 204 whether brand existed or not | `sql.ErrNoRows` treated as success |
| CreateAssignment | Yes (outcome) | `INSERT OR REPLACE` on scenario_name UNIQUE | Upsert semantics; same brand+scenario produces same state |
| DeleteAssignment | Yes | Returns 204 whether assignment existed or not | `sql.ErrNoRows` treated as success |

### Idempotency Keys

- **Scope**: Per-Handlers instance (in-memory `sync.Map`)
- **Lifetime**: Lives for the duration of the process; cleared on restart
- **Key format**: Arbitrary string provided by client via `Idempotency-Key` header
- **Cache content**: HTTP status code + JSON response body
- **Replay indicator**: `X-Idempotent-Replayed: true` header on cached responses
- **Limitation**: Only covers CreateBrand; other POST endpoints (CreateAssignment) use upsert semantics instead

### Safe Retry Patterns

| Pattern | When to Use | Example |
|---------|------------|---------|
| Idempotency-Key header | Creating brands from agents/scripts | `Idempotency-Key: task-123-create-brand` |
| If-Match header | Updating brands after reading current version | `If-Match: 3` (from ETag) |
| Plain DELETE | Always safe to retry | DELETE returns 204 regardless |
| Dry-run before real | Validate before committing | `X-Dry-Run: true` then real request |

### Unsafe Retry Patterns

| Pattern | Risk | Mitigation |
|---------|------|------------|
| POST /brands without Idempotency-Key | Duplicate brands | Use Idempotency-Key header |
| PUT without reading current version | Silent overwrite of concurrent changes | Use If-Match with ETag from GET |

## Optimistic Locking Protocol

### ETag / If-Match Flow

1. Client GETs brand -> response includes `ETag: <version>` header
2. Client PUTs update with `If-Match: <version>` header
3. Server compares If-Match value against current brand.Version
4. **Match**: Update proceeds normally, new ETag returned
5. **Mismatch**: 409 Conflict with `{code: "conflict", message: "...", recovery: "Re-read the resource and retry with the current version."}`

### Backwards Compatibility
- If-Match is **optional** — omitting it falls back to last-write-wins
- Existing clients without If-Match continue to work unchanged
- ETag is always returned on brand responses (GET, POST, PUT) for clients that want to opt in

## Data Integrity Invariants

### Version Counter
- `brand.Version` starts at 1 on create
- Incremented by 1 on each successful Update (in repository layer)
- Never decremented or reset
- Monotonically increasing per brand

### Version Snapshots
- Created after brand create/update as best-effort
- Snapshot failure does NOT prevent brand persistence (degraded mode)
- Snapshot `version` field matches `brand.version` at time of snapshot
- Snapshots are immutable once created

### Foreign Key Cascades
- Deleting a brand cascades to `brand_versions` and `assignments`
- No orphaned versions or assignments after brand deletion

### Assignment Uniqueness
- `assignments.scenario_name` has UNIQUE constraint
- Re-assigning a scenario replaces the previous assignment (INSERT OR REPLACE)
- At most one active assignment per scenario
