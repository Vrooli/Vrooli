# Temporal Flows

Documents async operations, ordering assumptions, and time-dependent behavior in the brand-manager scenario.

## Async Operation Inventory

### API Request/Response (synchronous)
All API operations are synchronous request-response. There are no background jobs, queues, or streaming endpoints.

| Operation | Endpoint | Side Effects | Ordering Notes |
|-----------|----------|-------------|----------------|
| CreateBrand | POST /brands | Insert brand row + version snapshot | Version snapshot is best-effort (logged warning on failure) |
| UpdateBrand | PUT /brands/{id} | Read-modify-write brand + version snapshot | Supports optimistic locking via `If-Match` header |
| DeleteBrand | DELETE /brands/{id} | Delete brand (cascades to versions/assignments) | Idempotent: returns 204 even if already deleted |
| CreateAssignment | POST /assignments | Upsert assignment (INSERT OR REPLACE) | Idempotent at DB level via scenario_name UNIQUE constraint |
| DeleteAssignment | DELETE /assignments/{id} | Delete assignment | Idempotent: returns 204 even if already deleted |

### React Query (client-side async)
- **Health polling**: Refetches every 30s via `refetchInterval` (configurable in constants)
- **Query invalidation**: Mutations invalidate related query keys on success (`["brands"]`, `["brand", id]`)
- **Conditional queries**: Version list query `enabled: !!brand` waits for brand data before fetching

### Iframe Bridge Initialization
- Runs before React mount (in `main.tsx`)
- Idempotency guard: `window.__brandManagerBridgeInitialized` prevents double-init on hot reload
- Gracefully handles missing `document.referrer`

## Ordering Assumptions & Stability

### Brand Create + Version Snapshot
- **Assumption**: Version snapshot creation follows brand creation in the same request handler
- **Stability**: Partially stable. If snapshot fails, brand exists without v1 snapshot (degraded but not broken)
- **Mitigation**: Warning logged; brand is usable; snapshot gap visible in version list

### Read-Modify-Write in UpdateBrand
- **Assumption**: Between GetByID and Update, no concurrent modification occurs
- **Stability**: Protected by `If-Match` optimistic locking when clients supply the header
- **Without If-Match**: Last-write-wins (acceptable for single-user or CLI workflows)
- **SQLite guarantee**: WAL mode serializes writes; busy_timeout (10s default) prevents immediate SQLITE_BUSY

### Version Ordering
- Versions are ordered by `version` column (integer, incremented on each update)
- `ListByBrandID` returns `ORDER BY version DESC` (newest first)
- Version numbers are strictly monotonic per brand (incremented in repository.Update)

## Race Conditions & Mitigation

| Race | Severity | Mitigation | Status |
|------|----------|------------|--------|
| Concurrent brand updates | Medium | `If-Match` optimistic locking (409 Conflict on version mismatch) | Mitigated |
| Double-create on retry | Medium | `Idempotency-Key` header deduplicates identical creates | Mitigated |
| Double-delete | Low | DELETE returns 204 regardless (idempotent) | Mitigated |
| Assignment re-assign same scenario | Low | `INSERT OR REPLACE` on scenario_name UNIQUE | Mitigated |
| Version snapshot fails after brand write | Low | Logged warning, brand still usable | Accepted (degraded) |

## Initialization & Teardown

### API Server
- **Init sequence**: Preflight checks -> Config load -> DB connect (WAL mode, pragmas) -> Schema init (IF NOT EXISTS) -> Wire repos -> Wire handlers -> Register routes -> Start HTTP server
- **Teardown**: Graceful shutdown via `server.Run` with cleanup callback that closes DB

### UI App
- **Init sequence**: Iframe bridge init (idempotent guard) -> React mount -> QueryClient setup -> App renders health check query
- **Teardown**: React unmount cleans up query client; hashchange listener removed by useRouter cleanup

### SQLite Connection
- **Pragmas applied on connect**: `foreign_keys=ON`, `journal_mode=WAL`, `busy_timeout=10000`, `cache_size`, `synchronous=NORMAL`, `temp_store=MEMORY`
- **Schema**: Idempotent (`CREATE TABLE IF NOT EXISTS`) — safe to re-apply on restart

## Polling & Retry Behavior

### Health Check Polling
- **Interval**: 30s (React Query `refetchInterval`)
- **Retry**: Default React Query retry (3 attempts with exponential backoff)
- **Impact**: Low — health check is lightweight and rate-limited

### No Server-Side Polling
- No background jobs, cron tasks, or polling loops in the API
- All state changes are initiated by client requests

## Checkpoint Flows

### Brand Edit Form (UI)
- **Progress state**: Local React `useState` (form fields)
- **Natural checkpoints**: Form submit (mutation) is the only checkpoint
- **Interruption behavior**: Unsaved form data is lost on navigation away
- **Resume**: Edit mode re-fetches brand from API; create mode starts fresh
- **Future improvement**: Could add localStorage draft persistence

### Version History
- Each brand update creates an immutable version snapshot
- Snapshots serve as progress checkpoints — you can always see the brand's state at any version
- No rollback mechanism yet (future: restore from snapshot)
