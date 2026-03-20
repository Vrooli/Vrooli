# Temporal Flows

Source of truth: the code. This document summarizes time-based and async behavior patterns.

## Server Lifecycle

| Phase | Implementation | Async? |
|-------|---------------|--------|
| Preflight | `preflight.Run()` — sync, re-execs if rebuild needed | No |
| DB Connect | `database.Connect(ctx)` — retry with backoff | No (blocks) |
| Schema Migration | `ensureSchema(db)` — idempotent `CREATE TABLE IF NOT EXISTS` | No |
| Service Init | 5 services created sequentially, no errors possible | No |
| HTTP Serve | `server.Run()` — blocks until signal | Goroutine-per-request (net/http) |
| Shutdown | Signal → `db.Close()` → server stops | No background workers to drain |

### Request Timeout

All requests are bounded by `RequestTimeout` (30s) via `requestTimeoutMiddleware`. The middleware attaches a context deadline so slow DB queries or future LLM calls cannot run indefinitely.

## Request-Response Flow

All API endpoints are synchronous request-response. No WebSockets, SSE, or streaming.

```
Client → HTTP Request → requestTimeoutMiddleware → loggingMiddleware → Handler → Service → DB → Response
```

### Concurrency Model

- Each request runs in its own goroutine (standard `net/http`)
- Services are stateless (hold only `*sql.DB` pointer) — safe for concurrent access
- `*sql.DB` connection pool handles internal synchronization
- No shared in-memory state, no mutexes, no channels in user code
- PostgreSQL Read Committed isolation handles transaction concurrency

## UI Async Patterns

### React Query Polling

| Component | Query Key | Interval | Degraded Interval |
|-----------|-----------|----------|-------------------|
| ConnectionStatus | `["health"]` | 15s | 5s |
| ProviderStatus | `["providers"]` | 30s | N/A |

### Drag Operations (CanvasView)

- **Item drag**: Uses window-level `mousemove`/`mouseup` listeners attached on `mouseDown`
- **Canvas pan**: Uses window-level `mousemove`/`mouseup` listeners attached on canvas `mouseDown`
- Both clean up via refs on unmount
- Scheme change resets drag, pan, and zoom state
- Zoom uses current value via ref (`zoomRef`) to avoid stale closures

### Mutation Ordering (GraphView)

- 4 independent mutations: `createMut`, `deleteMut`, `linkMut`, `deleteEdgeMut`
- Each invalidates relevant query keys on success
- Edge query key includes sorted thought IDs to prevent unnecessary refetches on reorder
- Link mode (`linkSource`) is cleared on mutation settlement (success or error) via `onSettled`

### Error Aggregation

- GraphView: First non-null error from `createMut || deleteMut || linkMut || deleteEdgeMut` is shown
- CanvasView: First non-null error from `updateMut || deleteMut`
- Reset clears all mutations, not just the displayed one

## Checkpoint Flows

### Canvas Position Persistence

Canvas position (pan/zoom) is **not persisted** — resets on scheme change and page reload. Item positions are persisted to the server via `updateInformation` on drag end.

### Scheme Selection

- Scheme selection is stored in React state (`activeScheme` in App.tsx)
- Not persisted across page reloads
- Switching schemes resets canvas view state (drag, pan, zoom)

### Text Capture

- Input text is cleared on successful submission
- Input is refocused after submission and on scheme change
- No draft persistence (text lost on reload or scheme switch)

## Known Temporal Concerns

1. **Canvas drag snap-back**: If position update fails, React Query refetch returns old position. User sees position revert. Mitigation: error banner shown.
2. **Edge query N+1**: `listEdges` is called per-thought via `Promise.all`. For N thoughts, N API calls fire. Not a race condition but a performance concern.
3. **No progress persistence**: View mode, canvas position, and draft text are ephemeral. See PROBLEMS.md for tracking.
