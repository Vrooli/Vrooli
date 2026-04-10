# Error Semantics

## Last Updated
2026-04-05

## Error Categories

### API Errors (HTTP responses)

All API errors return JSON: `{"error": "<CODE>", "message": "<human-readable>"}`

| HTTP Status | Error Code | When | Recovery |
|-------------|-----------|------|----------|
| 400 | `READ_ERROR` | Request body unreadable | Fix request payload |
| 400 | `INVALID_PAYLOAD` | Proto JSON unmarshal fails | Fix event envelope format |
| 400 | `MISSING_EVENT_TYPE` | `event_type` field empty | Provide event type |
| 400 | `MISSING_SOURCE` | `source_scenario` field empty | Provide source scenario |
| 400 | `INVALID_EVENT_TYPE` | Event type doesn't match `{a}.{b}.{c}.{d}` format | Fix event type format |
| 400 | `INVALID_LIMIT` | `limit` query param not a valid integer | Use integer limit |
| 400 | `INVALID_SINCE` | `since` query param not a valid integer | Use integer since |
| 500 | `STORE_ERROR` | SQLite write/read failure | Retry; check disk space |
| 500 | `MARSHAL_ERROR` | JSON serialization of response failed | Bug — should not happen |

### SSE Errors

- **Connection drop**: Client receives no data for >60s (2 missed heartbeats). Recovery: reconnect with `Last-Event-ID`.
- **Backpressure**: Subscriber channel full. Events dropped silently; `dropped_count` reported in next heartbeat. Recovery: consume events faster or filter more narrowly.

### Store Errors

- **WAL checkpoint failure**: Logged but non-fatal. SQLite retries automatically.
- **Prune failure**: Logged by pruner goroutine. Next interval retries. No user-visible impact unless disk fills.

## Error Propagation

```
Store error → Handler catches → writeError(w, 500, code, msg) → JSON response
Broker error → Subscriber channel closed → SSE connection ends → Client reconnects
Pruner error → Logged to stderr → Retry on next tick
```

## SSE Replay Error Handling

When a client reconnects with `Last-Event-ID`, the server replays missed events. Failures during replay are now logged with context:
- Invalid `Last-Event-ID` format → logged, replay skipped
- Store `GetSince` failure → logged with last ID, replay skipped
- Individual event conversion/marshal failure → event skipped, count logged
- Summary log emitted when any events are skipped during replay

## User-Facing Error Categories (UI)

The UI categorizes errors into four recovery-distinct groups via `categorizeError()` in `lib/errors.ts`:

| Category | Trigger | User Message | Recovery Action |
|----------|---------|-------------|-----------------|
| `connection` | `Failed to fetch`, `NetworkError`, `Load failed` | "Cannot reach the server" | Check API, retry |
| `server` | HTTP 500, 503 | "Server error" / "Service temporarily unavailable" | Wait, retry, check logs |
| `validation` | HTTP 400 | "Invalid request" | Fix filters/input |
| `unknown` | Anything else | "Something went wrong" | Refresh page |

**Error surfaces:**
- `ErrorAlert` component renders categorized messages with optional retry buttons
- `ErrorBoundary` wraps the entire app to catch render crashes with "Try Again" recovery
- All pages show contextual error state instead of stale data
- SSE malformed events logged to console with `[SSE]` prefix (no longer silently swallowed)
- QueryClient configured with 2 retries + exponential backoff (1s, 2s, 4s capped at 10s)

## Error Propagation

```
Store error → Handler catches → writeError(w, 500, code, msg) → JSON response
Broker error → Subscriber channel closed → SSE connection ends → Client reconnects
Pruner error → Logged to stderr → Retry on next tick
Replay error → Logged to stderr → Events skipped, client unaware
SSE parse error → console.warn("[SSE]") → Event skipped
Render crash → ErrorBoundary catches → "Try Again" fallback UI
```
