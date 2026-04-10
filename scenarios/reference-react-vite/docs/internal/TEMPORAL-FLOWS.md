# Temporal Flows Documentation

This document maps the time-based and asynchronous behavior patterns in the reference-react-vite scenario, following the temporal-flow-audit skill patterns.

## Overview

The reference-react-vite scenario is a task management application with a Go API backend and React+Vite frontend. All data operations are request-response based with no background jobs or long-running processes.

---

## Async Operation Inventory

### API Layer (Go)

| Operation | Trigger | Duration | Coordination |
|-----------|---------|----------|--------------|
| Health check | HTTP GET /health | ~1ms | Synchronous DB ping |
| Task CRUD | HTTP request | ~5-50ms | Single DB transaction |
| Project CRUD | HTTP request | ~5-50ms | Single DB transaction |
| Note CRUD | HTTP request | ~5-50ms | Single DB transaction |
| List queries | HTTP request | ~10-100ms | Single DB query + count |
| DB connection | Server startup | ~1-5s | Retry with backoff |
| Graceful shutdown | SIGTERM | ~5s max | Context cancellation |

**Key Observations:**
- All handlers are synchronous request-response
- No background workers or job queues
- Database operations use Go's `context.Context` for timeout/cancellation
- Connection pooling managed by `database/sql`

### UI Layer (React)

| Operation | Trigger | Duration | Coordination |
|-----------|---------|----------|--------------|
| Initial data load | Route mount | ~100-500ms | React Query |
| Health polling | Timer | Every 30s | React Query refetchInterval |
| Task/Project create | Form submit | ~100-300ms | useMutation |
| Task/Project update | Status toggle | ~100-300ms | useMutation + optimistic |
| Task/Project delete | Delete button | ~100-300ms | useMutation |
| Query invalidation | Mutation success | Immediate | queryClient.invalidateQueries |

**Key Observations:**
- React Query manages all server state
- No WebSocket or Server-Sent Events (SSE)
- Mutations use `onMutate` for tracking in-flight operations
- No local storage persistence (server is source of truth)

---

## Ordering Assumptions

### Stable Orderings

| Assumption | Location | Status |
|------------|----------|--------|
| Tasks listed by created_at DESC | [CODE: api/repository/tasks_postgres.go] | ✅ Enforced |
| Projects listed by created_at DESC | [CODE: api/repository/projects_postgres.go] | ✅ Enforced |
| Notes listed by created_at DESC | [CODE: api/repository/notes_postgres.go] | ✅ Enforced |
| Middleware order: CORS → Logging → RequestID | [CODE: api/main.go] | ✅ Explicit |

### Potentially Fragile Orderings

| Assumption | Location | Risk | Status |
|------------|----------|------|--------|
| Bridge init before React mount | [CODE: ui/src/main.tsx] | Low | ✅ Explicit guard |
| QueryClient exists before Provider | [CODE: ui/src/main.tsx] | Low | ✅ Module-level init |
| Root element exists | [CODE: ui/src/main.tsx] | Low | ✅ Throws if missing |

---

## Race Conditions Analysis

### Identified Race Conditions

| Risk | Severity | Location | Status |
|------|----------|----------|--------|
| Rapid status toggle | Low | [CODE: ui/src/pages/Tasks.tsx] | ✅ Mitigated |
| Concurrent creates | Low | API layer | ✅ No conflicts (UUIDs) |
| Double delete clicks | Medium | Delete buttons | ⚠️ Needs guard |
| Stale read before update | Low | PATCH handlers | ✅ Acceptable |

### Mitigation Details

**Rapid Status Toggle:**
- `updatingIds` Set tracks in-flight mutations ([CODE: ui/src/pages/Tasks.tsx])
- Button disabled while updating (`isUpdating` prop)
- `onSettled` cleans up tracking regardless of success/failure

**Double Delete:**
- Currently no UI guard beyond `isUpdating` check
- Server DELETE is idempotent-ish (returns 404 on second call)
- Recommendation: Add delete confirmation or debounce

---

## Initialization Patterns

### Server Initialization

```
main() starts
├── preflight.Run()           # Check for stale binary, may re-exec
├── config.LoadFromEnv()      # Parse environment variables
├── database.Connect()        # DB connection with retry/backoff
│   └── Returns db or fatal
├── NewServer(db, cfg)        # Wire dependencies
│   └── setupRoutes()         # Register all handlers
└── server.Run()              # Start HTTP server, graceful shutdown
```

**Status:** ✅ Well-defined, explicit ordering

### UI Initialization

```
Module load (main.tsx)
├── QueryClient creation      # Synchronous
├── Bridge check              # if (window.parent !== window && !initialized)
│   └── initIframeBridgeChild()
│   └── Set flag
├── Root element check        # Throws if missing
└── ReactDOM.createRoot().render()
    └── StrictMode
        └── QueryClientProvider
            └── App
                └── BrowserRouter
                    └── Routes
```

**Status:** ✅ Guard prevents double-init, clear error on missing root

---

## Teardown Patterns

### Server Teardown

- `server.Run()` handles SIGTERM/SIGINT
- Calls `Cleanup` function with context (DB close)
- Gorilla recovery middleware prevents panics from crashing server

**Status:** ✅ Graceful shutdown implemented via api-core

### UI Teardown

| Resource | Cleanup | Status |
|----------|---------|--------|
| React Query cache | Automatic GC | ✅ Default behavior |
| Health polling | Query unmount | ✅ Automatic |
| Event listeners | React handles | ✅ Via useEffect cleanup |
| Bridge messages | iframe-bridge | ✅ Handles internally |

**Status:** ✅ No manual cleanup needed for current feature set

---

## Polling and Scheduled Work

### Active Polling

| Poller | Interval | Location | Purpose |
|--------|----------|----------|---------|
| Health check | 30s | [CODE: ui/src/components/Layout.tsx] | API status indicator |

**Configuration:**
```tsx
refetchInterval: 30000 // 30 seconds
```

**Behavior:**
- Continues in background (no `refetchIntervalInBackground: false`)
- Shows "checking" → "healthy"/"offline" transition
- Non-blocking (health failure doesn't break app)

### Recommendations

1. ✅ Interval is reasonable (not aggressive)
2. Consider `refetchOnWindowFocus: false` for health if needed
3. No exponential backoff on health failure (acceptable for UX)

---

## Error Recovery Patterns

### Transient vs Permanent Errors

| Error Type | API Code | Retryable | Recovery |
|------------|----------|-----------|----------|
| Validation | 422 | No | Fix input |
| Not Found | 404 | No | Verify ID |
| Bad Request | 400 | No | Fix format |
| Internal | 500 | Yes | Retry with backoff |
| Network | N/A | Yes | Retry |

### UI Error States

| Component | Error Display | Recovery Action |
|-----------|---------------|-----------------|
| Tasks list | Error box with icon | "Make sure API is running" |
| Projects list | Error box with icon | "Make sure API is running" |
| Dashboard stats | Graceful (shows "--") | Auto-retry via React Query |
| Recent tasks | Error box | Same |
| Create/Update | Inline error message | "Please try again" |

**Strengths:**
- Consistent error UI pattern
- React Query handles retry logic
- Error boundaries prevent cascading failures

**Gaps:**
- No explicit retry button on list errors
- No toast notifications for mutation failures
- Recovery hints from API not surfaced in UI

---

## Checkpoint Flows

This section documents progress-based journeys and their natural checkpoints.

### Task Creation Flow

```
1. User enters title in form
   └── Checkpoint: None (ephemeral)
2. User clicks "Add Task"
   └── Checkpoint: POST request initiated
3. Server creates task
   └── Checkpoint: DB commit (durable)
4. UI invalidates queries
   └── Checkpoint: UI refresh
```

**Progress State:** Binary (pending or complete)
**Interruption Recovery:** Form cleared on success, preserved on failure

### Status Cycling Flow

```
1. User clicks status icon
   └── Checkpoint: updatingIds.add(id)
2. PATCH request initiated
   └── In-flight (visible via opacity change)
3. Server updates status
   └── Checkpoint: DB commit
4. UI invalidates + removes from updatingIds
   └── Checkpoint: UI refresh
```

**Progress State:** Tracked via `updatingIds` Set
**Interruption Recovery:** Server state is truth; UI will sync on refresh

### Multi-Step Workflows

Currently no multi-step workflows exist (e.g., wizard, batch operations).

**Future considerations:**
- Batch task creation could use staged commits
- Project templates might need progress tracking

---

## Concurrent Operation Handling

### Safe Concurrency

| Operation | Concurrent Safety | Notes |
|-----------|-------------------|-------|
| Parallel list fetches | ✅ Safe | Read-only, independent |
| Create during list | ✅ Safe | UUIDs, invalidation refresh |
| Update same task | ⚠️ Last-write-wins | No optimistic locking |
| Delete + update race | ✅ Safe | Delete wins, 404 on update |

### Recommendations

1. **Optimistic updates:** Currently mutations invalidate after settlement. Consider optimistic UI for better UX.

2. **Conflict detection:** No ETag/version checking. Acceptable for single-user but may cause surprises with multiple tabs.

3. **Debouncing:** Form submission has no debounce. Very rapid clicks could create duplicates (unlikely but possible).

---

## React Query Configuration

### Default Settings

The QueryClient uses default settings:
- `staleTime`: 0 (immediate refetch on mount)
- `gcTime`: 5 minutes
- `retry`: 3 times with exponential backoff
- `refetchOnWindowFocus`: true

### Query Keys

| Key Pattern | Usage |
|-------------|-------|
| `["health"]` | Health polling |
| `["tasks", {}]` | Full task list |
| `["tasks", { limit: 5 }]` | Recent tasks (dashboard) |
| `["projects", {}]` | Full project list |

**Note:** Empty object `{}` in key prevents conflicts with `{ limit: 5 }` queries.

---

## Signal Propagation

### Request Context Flow

```
HTTP Request
└── requestIDMiddleware
    └── loggingMiddleware
        └── corsMiddleware
            └── Handler
                └── Repository (ctx passed through)
                    └── DB operation
```

**Cancellation:** Context cancellation propagates to DB operations.

### Error Propagation

```
Repository error
└── Wrapped with context (fmt.Errorf)
    └── Handler catches
        └── Maps to APIError
            └── JSON response with recovery hint
```

---

## Related Documentation

- [DOC: docs/internal/SEAMS.md] - Architectural seams and boundaries
- [DOC: docs/internal/ERROR_SEMANTICS.md] - Error handling patterns
- [DOC: docs/reference/api-endpoints.md] - API reference

---

## Last Updated

2026-03-11 - Initial temporal flow audit (Phase 12)
