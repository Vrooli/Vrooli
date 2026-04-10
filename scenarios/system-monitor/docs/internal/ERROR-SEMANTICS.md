# Error Semantics

How errors are created, propagated, and consumed in the system-monitor API.

## Error Categories
[CODE: api/internal/apierrors/apierrors.go]

| Category       | HTTP | When to use                                      | Recovery hint     |
|----------------|------|--------------------------------------------------|-------------------|
| `validation`   | 400  | Malformed input, missing required fields         | `fix_input`       |
| `unauthorized` | 401  | Missing or invalid credentials                   | `authenticate`    |
| `forbidden`    | 403  | Valid credentials but insufficient permissions   | `contact_admin`   |
| `not_found`    | 404  | Entity does not exist                            | `none`            |
| `conflict`     | 409  | Duplicate or state conflict                      | `fix_input`       |
| `cooldown`     | 429  | Action rate-limited by cooldown timer            | `wait`            |
| `unavailable`  | 503  | Upstream dependency unreachable                  | `wait`            |
| `internal`     | 500  | Unexpected server-side failure                   | `none`            |

The frontend adds a ninth code, **`network`**, for fetch failures that never reached the server.

## Recovery Action Contract

| Recovery | UI Behavior |
|---|---|
| `fix_input` | Highlight the offending `field` if present; prompt user to correct input |
| `authenticate` | Prompt sign-in and retry |
| `wait` | Show retry timer if `retry_after_seconds` present, otherwise "try again shortly" |
| `check_config` | Suggest reviewing configuration settings |
| `contact_admin` | Suggest contacting an administrator |
| `none` | Generic "if this persists, contact support" |

## Repository Sentinel Error Pattern
[CODE: api/internal/repository/errors.go]

Repository implementations return **wrapped sentinel errors** instead of
category-specific API errors. This keeps the repository layer free of HTTP
semantics.

```go
// repository/errors.go
var ErrNotFound = errors.New("not found")

// In any repo method:
return nil, fmt.Errorf("investigation %s: %w", id, repository.ErrNotFound)
```

Service methods check with `errors.Is`:

```go
if errors.Is(err, repository.ErrNotFound) {
    return nil, apierrors.NotFound("investigation", id)
}
return nil, apierrors.Internal("Unable to retrieve investigation", err)
```

### Why not return `apierrors` from the repo?

1. Repositories should not know about HTTP status codes.
2. Sentinel errors can be matched structurally (`errors.Is`) rather than by
   fragile string comparison (`strings.Contains`).
3. Different services may want to handle the same repo error differently
   (e.g. return a default object vs. 404).

## Per-Subsystem Error Model

### Backend

Each API subsystem (investigations, reports, metrics, settings) translates
repository errors into `apierrors.APIError` at the service layer. Handlers
then call `httputil.WriteError` which serialises the structured error to JSON
with the correct HTTP status code.

```
Repository  -->  Service  -->  Handler  -->  HTTP response
(sentinel)      (apierrors)    (WriteError)  (JSON + status)
```

### Frontend

The `useSystemMonitor` hook tracks errors per subsystem rather than a single
shared error state:

```typescript
type Subsystem = 'metrics' | 'detailedMetrics' | 'processes' | 'infrastructure' | 'investigations';
subsystemErrors: Partial<Record<Subsystem, APIError>>
```

- Each fetch function sets/clears only its own subsystem key.
- A backward-compatible `error` field is derived (first non-null entry).
- Toast notifications fire only for `metrics` errors (primary data source).
- `isStale` tracks consecutive `metrics` failures (>= 3 = stale).

This prevents a transient failure in one subsystem from masking errors in
another or clearing legitimate warnings.

## Graceful Shutdown

`InvestigationService` holds a `shutdownCtx` / `shutdownCancel` pair:

- `runInvestigation` derives its context from `shutdownCtx`.
- On server shutdown, `Shutdown()` is called before `srv.Close()`, canceling
  in-flight investigations.
- The panic-recovery defer in the background goroutine handles cleanup even
  if cancellation races with execution.

## Context Cancellation Handling

When a request context is canceled (client disconnect) or exceeds its deadline,
`HandleError` maps the error to HTTP 503 with `code: "unavailable"`,
`retryable: true`, and `recovery: "wait"`. This avoids noisy 500 errors in logs
for what are typically transient conditions.

```go
// In HandleError:
if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
    // → 503, retryable, recovery=wait
}
```

Wrapped context errors (e.g., `fmt.Errorf("query: %w", context.Canceled)`) are
also caught via `errors.Is`.

## Retry-After Header

`WriteAPIError` sets the `Retry-After` HTTP header (RFC 7231 §7.1.3) when both
conditions hold:

1. `RetryAfterSecs > 0` on the `APIError`
2. The mapped HTTP status is 429 or 503

This enables automated clients and proxies to respect server-specified backoff
intervals. The `Cooldown` constructor sets `RetryAfterSecs` automatically.

## WriteError Recovery Fields
[CODE: api/internal/httputil/response.go]

`WriteError` (used by panic recovery, SafeProtoJSON fallback, and generic 500
paths) now populates `retryable` and `recovery` fields based on the HTTP status
code. This ensures that even non-`APIError` error responses carry actionable
recovery information for the UI.

## Guidelines

- **Never** match errors by string content (`strings.Contains`). Use
  `errors.Is` or `errors.As`.
- **Always** wrap sentinel errors with `%w` so callers can unwrap them.
- **Keep** user-facing messages vague ("Unable to retrieve report") and put
  detail in the internal `cause` field for logging.
- **Log** internal errors at the handler or middleware level; do not log AND
  return the same error.
