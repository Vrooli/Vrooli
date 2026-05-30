# Error Handling — Ecosystem Manager

## Purpose Of This Document

Ecosystem-manager is a Go REST/JSON API over `gorilla/mux`
([CODE: api/pkg/server/server.go]). It does **not** use proto +
Connect-RPC, so its error contract is an HTTP status code plus a JSON
error body — not the Connect error envelope used by newer Vrooli
scenarios. This document defines that contract, maps the common failure
categories to it, describes how the UI turns those responses into
user-facing messages, and records the deviation from the platform
standard.

Read this before adding an endpoint or surfacing a new error in the UI —
the goal is one consistent error shape across every handler.

## Transport error shape (REST/JSON)

Every handler failure goes through one helper,
[CODE: api/pkg/autosteer/handlers.go]::`writeError(w, statusCode, message)`,
which sets `Content-Type: application/json`, writes the HTTP status, and
encodes the `ErrorResponse` envelope:

```go
// api/pkg/autosteer/handlers.go
type ErrorResponse struct {
    Error   string `json:"error"`             // http.StatusText(statusCode)
    Message string `json:"message,omitempty"` // human-readable detail
    Code    int    `json:"code"`              // numeric HTTP status (mirror)
}
```

So a 404 looks like:

```json
{ "error": "Not Found", "message": "No execution state found", "code": 404 }
```

The HTTP status line is the primary signal; `error` is its text form,
`message` carries the specific detail, and `code` mirrors the status for
clients that read the body only. Success responses go through the sibling
`writeJSON(w, statusCode, data)` helper. There is no error envelope
beyond this struct — clients should branch on the HTTP status first and
read `message` for display.

## Error categories

| Category | HTTP status | Source / trigger | `message` example |
|---|---|---|---|
| Bad request — malformed body | `400` | `json.Decode` of the request body fails ([CODE: api/pkg/autosteer/handlers.go]) | `Invalid request body: <decode err>` |
| Bad request — missing fields | `400` | Required field validation (e.g. `task_id`, `profile_id`, `scenario_name`) | `task_id, profile_id, and scenario_name are required` |
| Not found — profile | `404` | `ProfileRepository.GetProfile` returns not-found | profile lookup error text |
| Not found — execution state | `404` | No `ProfileExecutionState` for the task | `No execution state found` |
| Conflict | `409` | A mutation conflicts with current state (e.g. duplicate create) | conflict detail |
| Engine unavailable | `503` | Auto-steer execution engine not wired/available (reset path) | `Auto Steer execution engine unavailable` |
| Internal | `500` | Persistence/IO failure, profile create/update/delete, history load | `Failed to <op>: <err>` |
| Upstream — agent-manager | `500` (surfaced) | Outbound run start/stop fails via `AgentServiceAPI` ([CODE: api/pkg/agentmanager/api.go]); errors wrap with context (`failed to evaluate iteration: …`) | `Failed to start execution: <upstream err>` |

### `MetricUnavailableError` is non-fatal (not an HTTP error)

A special case worth calling out: when a stop condition references a
metric that is unsupported or not yet collected, the evaluator returns a
typed `*MetricUnavailableError`
([CODE: api/pkg/autosteer/metrics_registry.go], surfaced from
`ConditionEvaluator.GetMetricValue` in
[CODE: api/pkg/autosteer/evaluator.go (19-78)]). The phase coordinator
**does not** turn this into an HTTP error — it `errors.As`-detects it,
logs, and continues evaluating the next condition
([CODE: api/pkg/autosteer/phase_coordinator.go (32-71)]). The loop is
intentionally tolerant of a missing sensor reading: a single
uncollected metric must not halt or fail the whole iteration. So this
error class shapes control-flow, not the HTTP response.

## UI error mapping

The UI talks to these endpoints through a typed client that raises an
`ApiError` carrying the HTTP status code, so components branch on status
rather than parsing message strings:

- **`404` on `/auto-steer/execution/{taskId}`** → "no execution state
  yet", rendered as the empty/uninitialized state (the
  `useAutoSteerExecutionState` hook returns undefined for 404 while
  surfacing other statuses as real errors). This is the canonical
  "missing is not an error" mapping.
- **`400`** → inline validation feedback on the form/field that produced
  the bad request; the `message` is shown as the detail.
- **`409`** → conflict notice prompting the user to refresh/retry.
- **`503`** → "Auto Steer is temporarily unavailable" — a transient,
  retryable banner, distinct from a hard failure.
- **`500` / upstream agent-manager failures** → generic error toast/state
  with the `message` for context; treated as retryable where the action
  is idempotent.

Centralizing the status code on `ApiError` keeps UI components from
inferring meaning from message text, so copy changes don't break error
handling.

## Deviation from the proto/Connect standard

The Vrooli platform standard (see the memory feedback "Proto + Connect
always") is that every scenario API domain uses proto + Connect-RPC, with
errors travelling as a `connect.Error` (typed `Code` → i18n key). Newer
scenarios map domain sentinels to Connect codes in a
`service_error_mapping.go` and let the UI translate `ConnectError.code`.

**Ecosystem-manager deviates: it is REST/JSON over `gorilla/mux`, with
the `ErrorResponse` envelope above instead of a Connect error.** This is
a deliberate, documented exception (the scenario predates the proto
mandate), not drift to fix opportunistically. The rationale and the
position on a future migration are recorded in
[../internal/DECISIONS.md](../internal/DECISIONS.md). Until a deliberate
migration happens:

- New endpoints stay REST/JSON and route failures through `writeError`
  so the error shape stays uniform within the scenario.
- Do **not** introduce a second, parallel error format (e.g. a bespoke
  per-handler JSON shape) — extend the `ErrorResponse` contract instead.
- When/if the scenario migrates to Connect, this document and the
  DECISIONS entry are the starting point for mapping each HTTP status to
  its Connect code.

## Cross-references

- [../internal/DECISIONS.md](../internal/DECISIONS.md) — why REST/JSON instead of proto/Connect
- [SEAMS.md](SEAMS.md) — `AgentServiceAPI` (upstream-failure source) and the repository seams behind 404/500
- [TESTING.md](TESTING.md) — asserting `ErrorResponse` (`code`/`message`) in handler tests
- [../concepts/CONTROL-MODEL.md](../concepts/CONTROL-MODEL.md) — why `MetricUnavailableError` is a non-fatal sensor-gap, not an HTTP error
