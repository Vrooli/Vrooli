# Web Console — Error Semantics

## Error Categories

All errors in the web-console scenario belong to exactly one of four categories.
Each category has a distinct recovery path — if two errors share a recovery strategy,
they belong in the same category.

| Category | Meaning | Recovery Path | Retryable |
|---|---|---|---|
| `validation` | User/caller provided bad input or referenced a missing resource | Fix input and retry; open a new terminal if session is gone | No |
| `resource_limit` | A capacity bound was hit (e.g. max sessions) | Free resources (close sessions), then retry | Yes |
| `dependency` | An external dependency failed (PTY, shell, network) | Check config or wait; system/agent may auto-retry | Depends |
| `internal` | Unexpected bug or invariant violation | Retry once; if persistent, check server logs or escalate | Yes |

### Adding a New Error Code

1. Assign it to exactly one category above
2. Define its `appError` entry in the `errorCatalog` map in [CODE: api/session_handlers.go#errorCatalog]
3. Update this document
4. Add a test in [CODE: api/session_handlers_test.go]

---

## Error Response Contract

All API error responses use this shape:

```json
{
  "error": "Human-readable description of what went wrong",
  "code": "machine_readable_slug",
  "category": "validation | resource_limit | dependency | internal",
  "recovery": "What the user or agent should do next",
  "retry": true
}
```

The `retry` field (boolean, omitted when false) tells clients whether the same
request is worth retrying. The `recovery` field provides guidance for both
human users and automated agents.

### Error Code Catalog

| Code | HTTP | Category | Retry | Recovery |
|---|---|---|---|---|
| `invalid_body` | 400 | validation | No | Check the request body format and try again |
| `session_limit_reached` | 429 | resource_limit | Yes | Close an unused terminal session, then retry |
| `pty_spawn_failed` | 500 | dependency | No | Check the configured shell path or server logs |
| `internal_error` | 500 | internal | Yes | Retry the request; if the problem persists, check server logs |
| `session_not_found` | 404 | validation | No | The session may have ended. Open a new terminal. |
| `profile_not_found` | 404 | validation | No | The profile may have been deleted. Refresh the profile list. |
| `session_terminated` | 410 | dependency | No | The terminal process exited. Open a new terminal. |

---

## Request Correlation

Every HTTP request receives a unique `X-Request-ID` header (format: `req-{ms}-{counter}`).
This ID appears in server log lines, enabling correlation between:
- What the user sees (error message + code)
- What the logs record (internal error details + request ID)

Clients can read the `X-Request-ID` response header for diagnostic purposes.

---

## WebSocket Error Protocol

Errors during an active WebSocket connection are signaled via the existing message format:

```json
{"type": "error", "data": "Human-readable error description"}
```

Errors sent:
- `"Invalid message format"` — client sent non-JSON
- `"Terminal process is not accepting input"` — PTY write failed

### Client-Side WS Error Recovery

The UI provides contextual recovery hints for known WS error messages:

| WS Error | Recovery Hint (shown in terminal) |
|---|---|
| `Invalid message format` | "A malformed message was sent. This is usually harmless." |
| `Terminal process is not accepting input` | "The terminal process has stopped. Close this pane and open a new terminal." |

### Sync Warning (Data-Loss Notification)

When a client's output channel falls behind (e.g. slow WebSocket consumer, network congestion), the server coalesces frames into a pending buffer instead of dropping them. After the configured threshold (`WC_COALESCE_NOTIFY_THRESHOLD`, default 5) of coalesced frames, a `sync_warning` message is sent:

```json
{"type": "sync_warning", "coalesced_frames": 7}
```

The client renders a yellow warning in the terminal: `[Warning: 7 output frames coalesced — terminal may lag]`. Coalesced data is automatically delivered when the consumer catches up. If the pending buffer grows beyond the configured cap (`OfflineBufferMax`), the oldest data is trimmed at an ANSI-clean boundary to prevent unbounded memory growth. This is informational, not an error — the session continues normally.

### Resize Info (Informational)

After a resize, the server sends the effective PTY dimensions back to the requesting client:

```json
{"type": "resize_info", "cols": 120, "rows": 40}
```

The PTY dimensions follow a last-writer-wins model — whichever client sends a resize message last sets the size. xterm.js handles reflow for smaller viewports automatically.

### WS Close Code Handling

| Close Code | User Sees |
|---|---|
| 1000, 1001 (normal) | `[Disconnected]` (gray) |
| Any other code (visible tab) | `[Connection lost, reconnecting...]` (gray) with auto-reconnect (exponential backoff, max 5 attempts) |
| Any other code (hidden tab) | `[Connection lost while backgrounded — will reconnect when tab is active]` (gray) with deferred reconnect on `visibilitychange` |
| Exhausted reconnect attempts | `[Connection lost]` (red) + "Open a new terminal if this persists" |

---

## Client-Side Failure Handling

| Component | Failure | User Experience | Recovery Action |
|---|---|---|---|
| [CODE: ui/src/App.tsx] (health check) | API unreachable | Error screen | "Retry Connection" button |
| [CODE: ui/src/components/Workspace.tsx] (create session) | Any API error | Dismissible error banner with message + recovery hint | "Try again" button if error is retryable |
| [CODE: ui/src/components/Workspace.tsx] (create session) | Session limit (429) | Banner: "Close an existing session and try again" | Retry button shown |
| [CODE: ui/src/hooks/useTerminalSocket.ts] | WS disconnect (abnormal, visible) | Gray `[Connection lost, reconnecting...]` + auto-reconnect | Automatic (exponential backoff, max 5 attempts) |
| [CODE: ui/src/hooks/useTerminalSocket.ts] | WS disconnect (abnormal, hidden) | Gray `[Connection lost while backgrounded]` + deferred reconnect | Automatic on tab visibility return |
| [CODE: ui/src/hooks/useTerminalSocket.ts] | WS reconnect exhausted | Red `[Connection lost]` + guidance | Close pane, open new terminal |
| [CODE: ui/src/hooks/useTerminalSocket.ts] | WS disconnect (normal) | Gray `[Disconnected]` | — |
| [CODE: ui/src/hooks/useTerminalSocket.ts] | Sync warning (dropped frames) | Yellow `[Warning: N output frames dropped]` | Reconnect to resync from history buffer |
| [CODE: ui/src/hooks/useTerminalSocket.ts] | Server error msg | Red `[Error: ...]` + contextual recovery hint | Depends on error |
| [CODE: ui/src/hooks/useTerminalSocket.ts] | Malformed WS JSON | Logged to console | Handler continues |
| [CODE: ui/src/lib/api.ts] (all endpoints) | Structured JSON error | Throws `APIError` with code, category, recovery, retry | Caller inspects fields |
| [CODE: ui/src/lib/api.ts] (all endpoints) | Non-JSON error | Throws `APIError` with fallback fields | Fallback retry=true |

---

## Structured Error Type (TypeScript)

Implementation: [CODE: ui/src/lib/api.ts#APIError]

The `APIError` class extends `Error` with structured fields:

```typescript
class APIError extends Error {
  readonly code: string;      // Machine-readable slug
  readonly category: string;  // validation | resource_limit | dependency | internal
  readonly recovery: string;  // What to do next
  readonly retry: boolean;    // Safe to retry?
  readonly status: number;    // HTTP status code
}
```

UI components use `instanceof APIError` to extract recovery metadata
and show contextual actions (retry buttons, guidance text).

---

## Remaining Gaps (Future Work)

1. ~~**WebSocket reconnection**~~ — **Resolved**: Auto-reconnect with exponential backoff (max 5 attempts) + visibility-aware deferral for backgrounded tabs.
2. ~~**Offline buffer overflow notification**~~ — **Resolved**: Per-client drop counting with `sync_warning` WebSocket message at configurable threshold (`WC_DROP_NOTIFY_THRESHOLD`).
3. **Session expiration** — Sessions never expire. Long-lived abandoned sessions leak memory.
4. **TOCTOU in max sessions** — The RLock count check and subsequent Lock for insert have a small race window under extreme concurrency. Acceptable for single-user bounds.
5. **WS handler test coverage** — `handleTerminalWS` has zero test coverage; error paths exist in code but have no automated verification. Requires a WebSocket test harness.
