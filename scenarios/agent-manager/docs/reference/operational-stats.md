# Operational Stats — API + CLI Reference

The Phase 3 operational-stats surface is the read-side of the typed-event
log. Every metric on this page is derived from `run_events` rows whose
`event_type` is in the typed-operational set (see
[Event Taxonomy](../internal/EVENT_TAXONOMY.md)) — none of it reads from
the live SQL run tables, so the same query answers "what just happened"
and "what happened last week" the same way.

The HTTP endpoints, CLI commands, and the UI's `/stats` and `/health`
pages all sit on top of the same `internal/stats` engine and
`internal/health` audit store.

## HTTP endpoints

All endpoints live under the existing API base. JSON in, JSON out.

### `GET /api/v1/stats/operational?category=<cat>`

Returns the per-category aggregate. `category` is required and **typed**:
unknown values return HTTP 400 with the list of accepted values.

| Category | Response type (Go) | Notes |
|---|---|---|
| `summary` | `stats.Summary` | Bundles all categories below; preferred for dashboards. |
| `fallback` | `stats.FallbackInsights` | Runner+model fallback frequency, chain depth, by reason. |
| `health` | `stats.HealthSummary` | Per-(runner, model) and per-runner snapshots, plus `failing_last_hour`. |
| `sandbox` | `stats.SandboxSummary` | sandbox.operation success/failure + duration. |
| `heartbeat` | `stats.HeartbeatSummary` | heartbeat.miss totals, by target. |
| `checkpoint` | `stats.CheckpointSummary` | checkpoint.failure by phase / step. |
| `retry` | `stats.RetrySummary` | retry.attempt by operation / reason. |

Every response carries a `history` field of type `stats.HistoryWindow`
(see `api/internal/stats/types.go`); the UI renders an "InsufficientData"
card when `history.history_days < 30` or sample size <
`history.min_sample_meaningful` (5).

### `GET /api/v1/stats/fallback`

Convenience alias for `/operational?category=fallback`. Returns
`stats.FallbackInsights` directly.

### `GET /api/v1/health/models`

Flat snapshot of every (runner, model) pair the system has observed.
Source of truth: `internal/health.Store.Snapshot`.

```json
{
  "models": [
    {
      "runner": "codex",
      "model": "gpt-5.2-codex",
      "status": "ok" | "unknown" | "failed",
      "last_checked": "2026-05-07T17:32:00Z",
      "reason": "rate_limit",
      "message": "anthropic 429"
    }
  ]
}
```

### `GET /api/v1/health/runners`

Same shape but at the runner level (no `model` field).

### `GET /api/v1/health/audit`

Paginated history. Filters: `scope=model|runner` (default `model`),
`runner`, `model`, `status`, `since` / `until` (RFC3339), `limit`
(default 100). Rows are `health.AuditRow` with `id`, `timestamp`,
`runnerType`, `modelId`, `status`, `reason`, `message`, `triggeredBy`.

### `GET /api/v1/events`

Read the typed-operational event log. Filters: `run` (UUID), `type`
(must be a typed-operational event_type — others return 400),
`since` (RFC3339), `limit` (max 1000, default 100). Each row exposes
the decoded `payload` keyed by `(event_type, schema_version)`.

## CLI

The CLI groups mirror the endpoint surface. Defaults are human-readable;
add `--json` for scripting.

```
agent-manager events ...    # query the typed event log
agent-manager health ...    # current snapshots and audit history
agent-manager ops ...       # operational stats (fallback/health/sandbox/...)
```

Run `agent-manager <group> --help` for the full subcommand list. The
canonical workflow:

| Question | Command |
|---|---|
| Are any models currently failing? | `agent-manager health models` |
| Show last 7 days of failures for a model | `agent-manager health audit --runner=codex --model=gpt-5.2-codex --since=7d` |
| How often does CHEAP fall through? | `agent-manager ops fallback` |
| Show every fallback event for a run | `agent-manager events list --run=<id>` |
| Tail the typed event log for a specific type | `agent-manager events list --type=runner.fallback.attempted --since=1h` |

The CLI never bypasses the API — every command exercises the same HTTP
surface above, so anything the CLI can answer is something a downstream
script or skill can also fetch from the API.

## Honesty contract

Every response that an operator could mistake for a stable trend
includes a `history_window`:

```json
{
  "history": {
    "earliest_event_at": "2026-04-30T00:00:00Z",
    "history_days": 7.04,
    "has_history": true,
    "min_sample_meaningful": 5
  }
}
```

The UI:

- Renders `HistoryBanner` above the Stats page when `history_days < 30`.
- Renders `InsufficientDataCard` in place of any card whose underlying
  sample size is below `min_sample_meaningful`.

The same constants are used across endpoints — the threshold is a
product judgment, not a per-card setting.

## See also

## Cross-scenario receipt ledger

`agent-manager run ledger <run-id>` exposes project-owned calls correlated to
one agent run. The ledger preserves the generic receipt envelope target,
operation, outcome, status, duration, verifier state, and policy-governed
projection only; it does not interpret target-specific projection keys.

Projection values are bounded to sixteen keys and two KiB per receipt. The
availability state is `available`, `empty`, `oversized`, `policy_absent`, or
`unavailable`, so missing receipt policy is never presented as a zero result.

- [Event Taxonomy](../internal/EVENT_TAXONOMY.md) — every typed event_type, its payload, and the schema_version contract.

- [Event Taxonomy](../internal/EVENT_TAXONOMY.md) — every typed event_type, its payload, and the schema_version contract.
- [Error Semantics](../internal/ERROR_SEMANTICS.md) — the `fallback.Reason` enum that drives `*.fallback.*` events.
- [Seams](../internal/SEAMS.md) — testability boundaries (stats engine, health store, eventlog).
