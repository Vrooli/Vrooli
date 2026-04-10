# Temporal Flows Documentation

> Current State (2026-02-14): Timing behavior centers on API request handling, React Query refresh policy, and execution mode scheduling (`manual`, `scheduled`, `yolo`).
>
> Historical note: Any recommendation-era timing references are preserved only as legacy context.

This document captures how the Swarm Manager handles time-based behavior—async operations, retry logic, initialization sequences, and state transitions. Understanding these flows is critical for debugging timing-related issues and making changes safely.

## Code References

- **API Client**: `ui/src/lib/api-client.ts` - HTTP infrastructure with timeout and error handling
- **Query Utilities**: `ui/src/lib/query-utils.ts` - React Query configuration
- **Configuration**: `ui/src/config/index.ts` - All timing-related constants
- **Error Utilities**: `ui/src/lib/error-utils.ts` - Error categorization and recovery paths

## Async Operation Inventory

### 1. API Requests (HTTP Client)

**Location**: `ui/src/lib/api-client.ts`

| Aspect | Value | Source |
|--------|-------|--------|
| Default timeout | 30,000ms | `apiConfig.requestTimeoutMs` |
| Timeout mechanism | `AbortController` | Line 145-146 |
| Cleanup on success | `clearTimeout(timeoutId)` | Line 157 |
| Cleanup on error | `clearTimeout(timeoutId)` | Line 180 |

**Flow**:
```
Request initiated
  → AbortController created
  → setTimeout schedules abort
  → fetch() starts
  → Response received OR abort triggered
  → clearTimeout() called
  → Return data OR throw ApiError
```

**Guarantees**:
- Timeout is always cleared (no leaked timers)
- Aborted requests throw `ApiError("timeout", ...)`
- Network failures throw `ApiError("network", ...)`

### 2. React Query Data Fetching

**Location**: `ui/src/lib/query-utils.ts`, `ui/src/pages/*.tsx`

| Setting | Value | Impact |
|---------|-------|--------|
| `retry` | 2 | Retry twice before showing error |
| `retryDelay` | Exponential (1s, 2s, 4s) | `retryDelayMs * 2^attempt` |
| `staleTime` | 30,000ms | Data considered fresh for 30s |
| `gcTime` | 300,000ms | Keep unused data 5 minutes |
| `refetchOnWindowFocus` | true | Refresh when tab regains focus |

**Flow**:
```
Component mounts
  → useQuery() checks cache
  → If stale/missing: fetch starts
    → isLoading = true (initial only)
    → On success: data populated
    → On failure: retry with backoff
      → After max retries: error set
  → Component renders based on state
```

## State Transition Ordering

### Page Load Sequence

```
1. main.tsx executes
   → QueryClient created (synchronous)
   → iframe bridge initialized (if iframe context)
   → React root created
   → App component renders

2. Router resolves
   → MainLayout renders
   → Page component renders
   → useQuery() triggers fetch

3. Data loading
   → isLoading = true, data = undefined, error = null
   → API request initiated
   → Response received
   → isLoading = false, data = [...], error = null
```

### State Consistency Matrix

React Query v5 state combinations:

| State | isLoading | data | error | Description |
|-------|-----------|------|-------|-------------|
| Initial | true | undefined | null | First fetch in progress |
| Success | false | [...] | null | Data loaded successfully |
| Empty | false | [] | null | Loaded, but no items |
| Error (initial) | false | undefined | Error | First fetch failed |
| Refetch error | false | [...] | Error | Refetch failed, stale data shown |

**Current UI behavior**: Shows error state AND data when both are truthy. This is intentional—users see stale data with an error banner, allowing them to continue working while acknowledging the issue.

## Race Conditions Assessment

### Identified (Mitigated)

1. **Request cancellation on unmount**
   - **Status**: MITIGATED by React Query
   - React Query automatically cancels in-flight requests when components unmount
   - AbortController propagates cancellation to fetch()

2. **Multiple rapid refetch triggers**
   - **Status**: MITIGATED by React Query
   - React Query deduplicates concurrent requests for the same query key
   - Window focus refetch doesn't stack with manual refetch

3. **Timeout cleanup on success**
   - **Status**: MITIGATED in api-client.ts
   - `clearTimeout()` called in both success and error paths
   - No leaked timers

### Not Applicable (Single-User Scope)

1. **Concurrent write conflicts**
   - The UI is single-user; no concurrent writes to the same idea
   - API uses file-based storage; OS-level file locking prevents corruption

2. **Stale read after write**
   - Not an issue currently; mutations would invalidate queries via React Query

## Initialization Patterns

### UI Initialization Order

```typescript
// main.tsx - order matters
const queryClient = new QueryClient();           // 1. Cache setup

if (window.top !== window.self) {
  initIframeBridgeChild();                       // 2. Bridge setup (conditional)
}

const rootElement = document.getElementById("root");  // 3. DOM access
ReactDOM.createRoot(rootElement).render(...);    // 4. React tree mount
```

**Stability**: This order is stable because:
- QueryClient has no async dependencies
- iframe bridge is fire-and-forget (doesn't block render)
- DOM element check throws clear error if missing

### Error Boundary Initialization

Error boundaries use `getDerivedStateFromError` which is synchronous—no timing issues.

```
Child throws during render
  → React calls getDerivedStateFromError (sync)
  → State set to { hasError: true, errorId: "..." }
  → componentDidCatch logs error (async-safe)
  → Fallback UI renders
```

## Polling & Retry Behavior

### Current Implementation

| Feature | Status | Configuration |
|---------|--------|---------------|
| Automatic refetch | ✅ Enabled | On window focus |
| Polling | ❌ Not implemented | N/A |
| Manual retry | ✅ Available | Via `refetch()` |
| Exponential backoff | ✅ Enabled | `1s * 2^attempt` |

### Retry Delay Calculation

```typescript
retryDelay: (attemptIndex: number) =>
  dataFetchingConfig.retryDelayMs * Math.pow(2, attemptIndex)

// Attempt 0: 1000ms
// Attempt 1: 2000ms
// Attempt 2: 4000ms (max, then error shown)
```

**Note**: No explicit cap on delay beyond retry count. With `retryCount: 2`, max total wait is ~7 seconds (1s + 2s + 4s) before error.

### Recommended Polling Pattern (Future)

For features requiring real-time updates, use:

```typescript
useQuery({
  queryKey: ["resource"],
  queryFn: fetchResource,
  refetchInterval: 30_000,  // Poll every 30s
  refetchIntervalInBackground: false,  // Pause when tab hidden
});
```

## Time-Sensitive Error Handling

### Error Categories and Recovery

From `ui/src/lib/error-utils.ts`:

| Category | Retryable | Recovery Path |
|----------|-----------|---------------|
| NETWORK | Yes | Auto-retry with backoff |
| TIMEOUT | Yes | Auto-retry with backoff |
| SERVER | Yes | Auto-retry with backoff |
| AUTH | No | Prompt re-authentication |
| NOT_FOUND | No | Navigate away |
| VALIDATION | No | User fixes input |
| PARSE | No | Report bug |
| RUNTIME | No | Refresh page |

### Current Flow

```
API Error occurs
  → ApiError created with type
  → React Query catches error
  → If retryable: schedule retry with backoff
  → If max retries reached: set error state
  → ErrorState component renders based on category
  → User sees appropriate message + recovery action
```

## Teardown Patterns

### Component Unmount

React Query handles cancellation automatically:
- In-flight requests are aborted
- Timers associated with retries are cleared
- Cache entries remain for `gcTime` duration

### Application Shutdown (API)

```go
// api/main.go - graceful shutdown via api-core/server
server.Run(server.Config{
  Handler: srv.Handler(),
  Cleanup: func(ctx context.Context) error { return db.Close() },
})
```

The `api-core/server` package handles:
- SIGINT/SIGTERM signal listening
- In-flight request completion (with timeout)
- Database connection cleanup

## Timing Configuration Reference

### Timing Constants

| Config | Value | Location |
|--------|-------|----------|
| `requestTimeoutMs` | 30,000 | `config/index.ts:apiConfig` |
| `retryCount` | 2 | `config/index.ts:dataFetchingConfig` |
| `retryDelayMs` | 1,000 | `config/index.ts:dataFetchingConfig` |
| `staleTimeMs` | 30,000 | `config/index.ts:dataFetchingConfig` |
| `cacheTimeMs` | 300,000 | `config/index.ts:dataFetchingConfig` |
| `searchDebounceMs` | 300 | `config/index.ts:uiBehaviorConfig` |
| `toastDurationMs` | 5,000 | `config/index.ts:uiBehaviorConfig` |
| `defaultDelaySeconds` | API-managed policy | `.vrooli/execution-policy.json` via execution policy endpoints |

### Adjusting Timing Behavior

All timing values are centralized in `ui/src/config/index.ts`. To modify:

1. Find the relevant config group
2. Update the value with documented range
3. Run tests to verify no regressions
4. Update this document if behavior changes

## Audit Summary

### Phase 18 Findings (2026-01-28)

| Area | Status | Notes |
|------|--------|-------|
| Request timeouts | ✅ Stable | AbortController with proper cleanup |
| Retry logic | ✅ Well-configured | Exponential backoff, bounded retries |
| State transitions | ✅ Consistent | React Query v5 state model |
| Initialization order | ✅ Deterministic | No race conditions |
| Error recovery | ✅ Categorized | Clear paths per error type |
| Teardown | ✅ Handled | React Query + Go graceful shutdown |

### Post-Execution Review & Follow-Up Flow

**Location**: `api/internal/execution/service.go` (`refreshRunningLocked`, `FollowUp`)

When an execution completes on a scenario-scoped backlog item:

```
Agent completes → StatusCompleted
    ↓
shouldTriggerReview() — checks: acceptance_allow references scenarios/, not archive, reviewClient available
    ↓ YES
TriggerReview() → POST git-control-tower /api/v1/review/run
    ↓
StatusValidating (stores ReviewJobID)
    ↓ polling loop (2s tick)
PollReview() → GET /api/v1/review/run/{jobId}
    ↓ completed
mapJobToResult() — green→ready, yellow→ready_with_notes, red→needs_work
    ↓
  ┌─ ready/ready_with_notes → StatusCompleted, backlog → "completed"
  └─ needs_work:
       ├─ AutoFixup ON && attempts < max → spawnFixupRun() (auto, background)
       └─ AutoFixup OFF || max reached → StatusNeedsFixup (user action needed)
```

**User-initiated follow-up** (`POST /api/v1/execution/{id}/follow-up`):

```
User clicks "Follow Up" on completed/failed/needs_fixup execution
    ↓
FollowUpDialog — selects type (fixup/followup/custom) + run mode (continue/new)
    ↓
  ┌─ continue → ContinueRun(runID, message) — replies to existing agent session
  └─ new → SpawnBacklog() — fresh agent with follow-up prompt
    ↓
New execution record created with ParentExecutionID linking to original
```

**Timing**: Review polling runs on the same 2s tick as the main `refreshRunningLocked` loop. Follow-up is synchronous from the user's perspective (request returns the new execution record).

### Future Work

1. **Add request cancellation test** - Verify unmount cancels in-flight requests
2. **Document retry delay cap** - Consider adding explicit max delay (e.g., 30s)
3. **Add isFetching indicator** - Show subtle refetch indicator when `isFetching && data`
4. **Test window focus refetch** - Add test for tab visibility behavior
