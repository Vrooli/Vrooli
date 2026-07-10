# Error Semantics & Recovery Paths

This document defines the error categories, recovery paths, and handling patterns
for the Swarm Manager scenario. It serves as the source of truth for understanding
how errors should be categorized, displayed, and recovered from.

**Implementation References:**
- Error types: [CODE: ui/src/lib/api-client.ts#ApiError]
- Error categories: [CODE: ui/src/lib/error-utils.ts#ErrorCategory]
- Recovery paths: [CODE: ui/src/lib/error-utils.ts#RECOVERY_PATHS]
- Error boundary: [CODE: ui/src/components/ui/error-boundary.tsx]
- Error state display: [CODE: ui/src/components/ui/error-state.tsx]

## Error Categories

The scenario uses 8 error categories, each with a distinct recovery path:

| Category | When Used | Recovery Path | Retryable |
|----------|-----------|---------------|-----------|
| **NETWORK** | Connection failures, DNS errors, offline | Retry with backoff | Yes |
| **TIMEOUT** | Request exceeded timeout | Retry with backoff | Yes |
| **AUTH** | 401 Unauthorized, 403 Forbidden | Re-authenticate / refresh | No |
| **NOT_FOUND** | 404 Not Found | Navigate away | No |
| **SERVER** | 5xx errors | Retry later | Yes |
| **VALIDATION** | 400, 422 - bad input | Fix input and resubmit | No |
| **PARSE** | Invalid JSON response | Report bug | No |
| **RUNTIME** | Unexpected exceptions | Refresh page | No |

### Category Design Principles

1. **Mutually exclusive**: Each error maps to exactly one category
2. **Recovery-distinct**: Each category has a unique recovery action
3. **User-recognizable**: Categories make sense to users, not just developers
4. **Stable across codebase**: Same category means same thing everywhere

---

## Error Handling Layers

The scenario has 4 layers of error handling:

```
┌─────────────────────────────────────────────────────────┐
│ Layer 1: ErrorBoundary (React runtime errors)           │
│   └─ Catches: render errors, lifecycle errors           │
│   └─ Recovery: Full page refresh                        │
│   └─ [CODE: ui/src/components/ui/error-boundary.tsx]    │
├─────────────────────────────────────────────────────────┤
│ Layer 2: ApiClient (HTTP/network errors)                │
│   └─ Catches: network, timeout, http, parse errors      │
│   └─ Returns: ApiError with type, status, userMessage   │
│   └─ [CODE: ui/src/lib/api-client.ts]                   │
├─────────────────────────────────────────────────────────┤
│ Layer 3: ErrorState Component (user-facing errors)      │
│   └─ Displays: friendly message with recovery action    │
│   └─ Actions: retry button, navigation, refresh         │
│   └─ [CODE: ui/src/components/ui/error-state.tsx]       │
├─────────────────────────────────────────────────────────┤
│ Layer 4: NotFoundPage (invalid routes)                  │
│   └─ Catches: unknown routes via React Router           │
│   └─ Recovery: Navigate to home                         │
│   └─ [CODE: ui/src/pages/NotFoundPage.tsx]              │
└─────────────────────────────────────────────────────────┘
```

---

## API Error Type System

### ApiError Class

Located in `ui/src/lib/api-client.ts`:

```typescript
type ApiErrorType = "network" | "timeout" | "http" | "parse";

class ApiError extends Error {
  readonly type: ApiErrorType;
  readonly status?: number;
  readonly isClientError: boolean;  // 4xx
  readonly isServerError: boolean;  // 5xx
  readonly isRetryable: boolean;

  get userMessage(): string;  // User-friendly message
}
```

### Type-to-Category Mapping

| ApiErrorType | HTTP Status | ErrorCategory |
|--------------|-------------|---------------|
| network | - | NETWORK |
| timeout | - | TIMEOUT |
| http | 401, 403 | AUTH |
| http | 404 | NOT_FOUND |
| http | 400, 422 | VALIDATION |
| http | 5xx | SERVER |
| parse | - | PARSE |

---

## Critical Flows & Dependencies

### 1. Backlog Flow

**Path:** User → BacklogPage → backlogService → ApiClient → HTTP → API → Filesystem

**Primary Inputs:**
- Page navigation (route change to `/backlog`)
- Search queries (future: debounced input)
- Filter selections (future: status, tags, priority)

**External Dependencies:**
| Dependency | Criticality | Failure Impact |
|------------|-------------|----------------|
| Network | Critical | Total failure - ErrorState shown |
| API Server | Critical | Total failure - ErrorState shown |
| Filesystem | Critical | API returns 500 - SERVER error |

### 2. Scenarios Flow

Same pattern as Backlog Flow.

### 3. Settings Flow

**Path:** User → SettingsPage → localStorage (future)

**Current State:** UI-only, no persistence. Settings reset on refresh.

### 4. CLI Health Check

**Path:** User → `swarm-manager status` → HTTP → API /health

| Failure | Current Behavior | Category |
|---------|------------------|----------|
| API unreachable | Returns raw Go error | NETWORK |
| Invalid JSON | Prints raw bytes | PARSE |

---

## Failure Modes by Flow

### Backlog/Scenarios Flow Failures

| Failure Mode | Category | Behavior | Recovery |
|--------------|----------|----------|----------|
| Network offline | NETWORK | ErrorState with "Connection problem" | Retry button |
| API timeout | TIMEOUT | ErrorState with "Request timed out" | Retry button |
| API returns 401/403 | AUTH | ErrorState with "Session expired" | Refresh page |
| API returns 404 | NOT_FOUND | ErrorState with "Not found" | Navigate back |
| API returns 5xx | SERVER | ErrorState with "Server error" | Retry button |
| Invalid JSON | PARSE | ErrorState with "Invalid response" | Report issue |
| React render error | RUNTIME | ErrorBoundary fallback | Refresh page |
| Unknown route | NOT_FOUND | NotFoundPage | Go to home |

### CLI Health Check Failures

| Failure Mode | Category | Current Behavior |
|--------------|----------|------------------|
| API unreachable | NETWORK | Returns error from api-core |
| Invalid JSON | PARSE | Prints raw bytes |

### Operating Mode Execution Failures

| Failure Mode | Scope / Impact | Behavior | Recovery |
|--------------|----------------|----------|----------|
| Multiple nonterminal manifests for one target/mode | Local / high integrity risk | Start and resume fail with `ErrExecutionAmbiguous`; no precedence rule selects one | Inspect manifests and repair the duplicate ownership state before retrying |
| One Agent Manager run indexed to two execution rounds | Local / high attribution risk | Second registration fails with `ErrRunOwnerAmbiguous`; the first mapping is retained | Investigate the conflicting dispatch; never overwrite the owner index |
| Round definition digest differs from its manifest | Local / high replay risk | Save/interpretation fails closed before routing or side effects | Restore the matching manifest/round pair from durable history |
| Live registry changes during an execution | Expected evolution / no impact | Existing rounds resolve, classify, and delegate through the pinned bundle; only a new execution sees the edit | No recovery required; start a new execution to adopt the new definition |
| Legacy flat round has no execution manifest | Local / compatibility | An unambiguous history is staged and validated into a deterministic pinned execution while its original bytes are retained under `legacy-rounds/`; ambiguous history remains readable and is excluded from continuation | Repair ambiguity deliberately or start a fresh execution; never infer precedence |
| Caller input is missing, unknown, mistyped, out of bounds, oversized, sensitive, or uses non-replayable retention | Local / request error with no side effects | Execution preflight rejects the request before creating a manifest, round, lock, or run | Correct the request using the catalog/workspace compiled input contract; sensitive caller values require a future secure runtime store rather than bypassing retention policy |
| Caller inputs on a retry differ from the active execution snapshot | Local / high replay risk | Resume fails closed; the existing manifest and snapshot remain unchanged | Omit inputs when continuing, repeat the identical normalized map, or finish/cancel and start a new execution |
| Persisted input contract or input snapshot digest does not match canonical JSON content | Local / high replay risk | Manifest load fails closed before prompt rendering, routing, or spawn | Restore the matching manifest bytes from durable history; never recompute the stored digest to hide an unexplained mutation |

---

## Graceful Degradation Patterns

### Pattern 1: Error Type Differentiation

All API errors are wrapped in `ApiError` with:
- Machine-readable `type` property
- HTTP `status` when applicable
- `isRetryable` flag for retry logic
- `userMessage` for display

### Pattern 2: User-Friendly Error States

The `ErrorState` component provides:
- Clear title explaining what went wrong
- Helpful message with guidance
- Retry button for retryable errors
- Visual distinction from empty states (red vs neutral)

### Pattern 3: Error Boundary Fallback

The `ErrorBoundary` component catches React runtime errors:
- Full-screen fallback UI
- Refresh button for recovery
- Structured error logging with correlation ID
- Does NOT expose stack traces

### Pattern 4: 404 Route Handling

The `NotFoundPage` catches unknown routes:
- Friendly "Page not found" message
- Navigation back to home (Backlog)
- Maintains layout consistency

### Pattern 5: Retry with Exponential Backoff

Failed data fetches retry with:
- Configurable retry count (default: 2)
- Base delay (default: 1000ms)
- Exponential backoff: `delay * 2^attempt`

---

## Observability

### Structured Error Logging

Located in `ui/src/lib/error-utils.ts`:

```typescript
interface ErrorLogEntry {
  timestamp: string;      // ISO timestamp
  category: ErrorCategory;
  message: string;        // Sanitized
  correlationId: string;  // For tracing
  status?: number;
  retryable: boolean;
  source: string;         // Component name
  context?: Record<string, any>;
}
```

### Log Format

Errors are logged as structured JSON:
```
[CATEGORY] {"timestamp":"...","category":"NETWORK","message":"...","correlationId":"err_123_abc",...}
```

### Sanitization

Error messages are automatically sanitized:
- URLs replaced with `[URL]`
- File paths replaced with `[PATH]`
- Long messages truncated to 200 chars

### NOT Exposed to Users

- Stack traces
- Internal URLs or paths
- Database error details
- Session/auth tokens
- User input data

---

## Recovery Guidance

Each error category has defined recovery guidance in `ui/src/lib/error-utils.ts`:

| Category | Action | Button | Can Retry |
|----------|--------|--------|-----------|
| NETWORK | Check your internet connection... | Try Again | Yes |
| TIMEOUT | The server is taking too long... | Try Again | Yes |
| AUTH | Your session has expired... | Refresh | No |
| NOT_FOUND | This resource no longer exists | Go Back | No |
| SERVER | The server encountered an error... | Try Again | Yes |
| VALIDATION | Please check your input... | (none) | No |
| PARSE | Received an invalid response... | Report Issue | No |
| RUNTIME | Something went wrong... | Refresh Page | No |

---

## Testing Error Paths

### Unit Tests

- `api-client.test.ts`: Error type differentiation (23 tests)
- `error-utils.test.ts`: Error categorization, sanitization, recovery (27 tests)
- `BacklogPage.test.tsx`: ErrorState rendering, retry behavior (8 tests)

### Test Pattern

```typescript
// Mock specific error
vi.mocked(backlogService.list).mockRejectedValue(
  new ApiError('network', 'Network error')
);

// Verify error state renders
await waitFor(() => {
  expect(screen.getByTestId('error-state')).toBeInTheDocument();
});

// Verify retry works
fireEvent.click(screen.getByTestId('error-retry'));
expect(backlogService.list).toHaveBeenCalledTimes(2);
```

---

## State Integrity & Recovery

### Idempotent Operations

All read operations (list, get) are naturally idempotent. Future write operations
will be designed with idempotency in mind:

- **Create:** Return existing entity if duplicate name
- **Update:** Use optimistic locking (version field)
- **Delete:** Succeed silently if already deleted

### Recovery Paths

| Scenario | Recovery Action |
|----------|-----------------|
| Network restored | User clicks retry, or refetchOnWindowFocus triggers |
| API server restored | Automatic retry on next interaction |
| Stale cache | staleTime triggers background refetch |
| Corrupted state | Full page refresh clears all React Query state |
| Runtime error | ErrorBoundary shows refresh button |
| Unknown route | NotFoundPage shows navigation to home |

---

## Error Semantics Change Log

| Date | Author | Changes |
|------|--------|---------|
| 2026-01-28 | Claude Opus 4.5 | Initial failure topography mapping (Phase 5) |
| 2026-01-28 | Claude Opus 4.5 | Error categories, recovery paths, observability (Phase 6) |
