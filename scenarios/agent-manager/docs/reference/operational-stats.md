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

### Invocation denominators

Invocation rates never divide by all retained rows. Each response exposes a
`classifiedBase`, `unclassifiedCount`, and `unclassifiedShare`; the base is
the set of command facts the classifier could safely interpret. Paired-call
counts are the outcome evidence for failure rates, while unpaired and
unclassifiable facts remain visible. A zero classified base is `unavailable`;
coverage below the declared 90% minimum is `unreliable` with a reason.

For the Meta Optimization sweep, the reproducible command sequence is:

```bash
agent-manager run cohort show imported-baseline-2026-08 --json
agent-manager run invocation-metrics --cohort imported-baseline-2026-08 --json
agent-manager run episode-cohort --cohort imported-baseline-2026-08 --json
agent-manager run goal-cohort --cohort imported-baseline-2026-08 --json
```

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

## Measure classifier-validity gate

Every typed measure response includes `validity`. A response is `unreliable`
instead of a trustworthy analytical result when its sample is smaller than the
configured meaningful minimum, or when one classifier fingerprint accounts for
more than the configured share of the measured population. The response keeps
its counts for diagnosis and states the reason; CLI human output does not print
a bare repeated-work percentage in that state.

The product defaults are a minimum sample of `5` and a maximum fingerprint
bucket share of `0.90`. Operators may set
`AGENT_MANAGER_MEASURE_MIN_SAMPLE_MEANINGFUL` and
`AGENT_MANAGER_MEASURE_MAX_FINGERPRINT_BUCKET_SHARE` (a fraction in `(0,1]`)
at startup. These are global analytical-honesty settings, never per-measure
tuning knobs.

## Token attribution

`agent-manager measures token-attribution` reports durable token rows grouped
by `capability`, `executable`, `command_path`, or
`target_scenario_operation`. Select one of three views:

- `footprint`: estimated intrinsic result size for finding oversized command
  output;
- `residency`: footprint carried across turns, using observed compaction
  attenuation as an explicit approximation; or
- `incurred`: measured provider usage attributed to the local turn, with an
  explicit `unattributed` residual when evidence cannot explain the run total.

Every response includes estimated token share and a token basis. Footprint
also exposes p50, p95, and maximum footprint values. These views are not
alternate names for one cost: footprint is a payload estimate, residency is a
context-occupancy estimate, and incurred is turn-local accounting. The durable
bucket conservation rule and approximation bias are documented in the
[token attribution reference](token-attribution.md).

## Cohort investigations

`agent-manager run investigate` can select durable evidence without manually
listing run IDs. Supply a shared read-model filter or explicit run IDs:

```bash
# Any shared invocation-read-model predicate.
agent-manager run investigate --filter-json '{"runnerType":"codex","runStatus":"failed"}' --depth quick
```

The selection is evaluated against the same `invocationreadmodel.Filter` used
by aggregates and cohorts. Investigations are capped at 50 runs. Their context
records the predicate, matched-run count, and omitted-run count, so a bounded
selection is never presented as a complete cohort.

## See also

## Cross-scenario receipt ledger

`agent-manager run ledger <run-id>` exposes project-owned calls correlated to
one agent run. The ledger preserves the generic receipt envelope target,
operation, outcome, status, duration, verifier state, and policy-governed
projection only; it does not interpret target-specific projection keys.

Projection values are bounded to sixteen keys and two KiB per receipt. Every
evidence surface uses the closed availability vocabulary: `available`,
`unavailable`, `degraded`, `unobserved`, `unknown`, `resolved`,
`policy_absent`, `oversized`, `not_captured`, `external`, `empty`, and
`complete`. The accompanying `reason` explains non-self-evident states, so a
missing receipt policy is never presented as a zero result. In particular,
`unobserved` means collection was armed but found no verified receipt; `empty`
means a policy returned no projection fields; and `policy_absent` means no
applicable capture or projection policy exists.

### Receipt-backed capability measures

`agent-manager measures capability-usage --json` reports verified calls grouped
by target scenario and operation, with call count, success/failure outcome, and
total duration. `capability-efficacy` adds separate `fallback-after` and
`abandoned` counts; it deliberately does not collapse them into one score.
Both responses carry `validity`. No verified receipt means `unavailable` with a
reason, not a zero. Filters may constrain `target_scenario` and `operation`.

The shared API-core runtime emits receipts after an instrumented request only
when the local vrooli-events policy cache authorizes a bounded projection and
the inbound Agent Manager provenance is verified. Ledger rows are limited to
verified receipts with non-empty target and operation; empty sentinel rows are
not evidence.

### Corpus import selection

`agent-manager run import-session-corpus` supports `--strategy deterministic-per-month`,
`--strategy stratified`
(deterministic runner-month coverage), `--strategy recent` (newest governed
sessions first), and `--strategy all` (bounded by `--limit`). Every response
reports selected, imported/existing, replayed, unreplayable, failed, and named
skipped counts. Imported runs remain terminal and are replayed through pinned
classifier versions; partial import is visible in the returned coverage.

### Evidence-spine answer surfaces

The retained projection answers the plan's twelve target questions through
these bounded surfaces:

- `run report <run-id>`: time accounting, episode cost, suspected ownership,
  self-report spans, and receipt availability for one run.
- `run invocation-facts <run-id>`: ownership, unclassified reason, capability,
  intent class, segment, outcome, and classifier version per invocation.
- `run ledger <run-id>` and `measures capability-usage`: verified capability
  calls, outcomes, duration, population, and receipt validity.
- `measures capability-efficacy`: separate calls, successes, fallback-after,
  and abandoned counts; no blended score is emitted.
- `run episode-cohort` and `run cohort-compare`: recurring fingerprint cost,
  distinct-run counts, population sizes, and before/after deltas.
- `run episode-trend`: time-bucketed episode occurrence, distinct-run, and
  wall-clock cost trends with the episode classifier version.
- `run messages-friction` and `run episodes`: bounded struggle vocabulary,
  operator corrections, guidance repair, wait misuse, abandonment, and
  handoff continuation.
- `run publish-recurring-friction`: idempotent three-distinct-run routing to
  `friction-inbox/<scope>/`, with `recurring`, `auto-generated`, recurrence
  evidence, and a reported daily cap/withheld count.
- `run import-session-corpus` and `run replay-invocation-corpus`: named
  sampling strategy, candidate/omitted counts, checkpoint, per-item failure,
  transcript time basis, cost basis, and pinned classifier replay.

Every surface carries either a validity or availability contract. Missing
receipts, too-small comparison populations, unclassified calls, and absent
transcript usage are reported explicitly rather than converted to zeroes.

- [Event Taxonomy](../internal/EVENT_TAXONOMY.md) — every typed event_type, its payload, and the schema_version contract.
- [Error Semantics](../internal/ERROR_SEMANTICS.md) — the `fallback.Reason` enum that drives `*.fallback.*` events.
- [Seams](../internal/SEAMS.md) — testability boundaries (stats engine, health store, eventlog).
