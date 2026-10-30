# Operational Stats — API + CLI Reference

The Phase 3 operational-stats surface is the read-side of the typed-event
log. Every metric on this page is derived from `run_events` rows whose
`event_type` is in the typed-operational set (see
[Event Taxonomy](../internal/EVENT_TAXONOMY.md)) — none of it reads from
the live SQL run tables, so the same query answers "what just happened"
and "what happened last week" the same way.

The HTTP endpoints, CLI commands, and the UI's Health (`/observability`) page sit on
the `internal/health` audit store and the Measures service.

## HTTP endpoints

### Efficiency report

There is no efficiency REST endpoint. The bounded, read-only efficiency report is
the Program Runtime program `agent-manager.efficiency-report` (inputs `preset`,
`start`/`end`, `tag_prefix`, `group_by=runner|model|profile`, `limit`,
`compare=previous`); per-workload consumption is `agent-manager measures
workload-efficiency`.

The same freshness evidence is available on governed `MeasuresService` responses
under `provenance`: `projection_at`, `projection_age_ms`, `projection_stale`,
and `projection_stale_reason`. The Program Runtime program
`agent-manager.efficiency-report` uses these fields and returns
`stale_evidence` with `status: "partial"` when the projection is older than its
requested window. It composes existing read-only measure bindings and keeps its
output bounded; it does not call an inference model or change team policy.

All endpoints live under the existing API base. JSON in, JSON out.

### Fallback insights (Measures)

Runner and model fallbacks and policy-candidate outcomes are the Health
measure `health.fallback_insights` (`MeasuresService.FallbackInsights`, CLI
`agent-manager measures fallback-insights [--window TOKEN]`). It counts the
`runner.fallback.*`, `model.fallback.*` and `policy.candidate.attempt` events
still retained in the window: attempts, exhausted walks, the most frequent
from/to/reason pairs, reasons, and policy outcomes and failure classes. The
former `/api/v1/stats/operational` and `/api/v1/stats/fallback` routes, the
`internal/stats` engine and `agent-manager ops` were deleted (P2-2); sandbox,
heartbeat, checkpoint and retry counters have no consumer and no replacement.

### `GET /api/v1/health/models`

Flat snapshot of every (runner, model) pair the system has observed.
Source of truth: `internal/health.Store.Snapshot`.

```json
{
  "models": [
    {
      "runner": "codex",
      "model": "<observed-model>",
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
agent-manager measures fallback-insights   # fallbacks and policy-candidate outcomes
```

Run `agent-manager <group> --help` for the full subcommand list. The
canonical workflow:

| Question | Command |
|---|---|
| Are any models currently failing? | `agent-manager health models` |
| Which models does Codex currently offer? | `resource-codex models list --json` (the runner catalog; use the resource policy command for role selection) |
| Show last 7 days of failures for a model | `agent-manager health audit --runner=codex --model=<observed-model> --since=7d` (substitute a model returned by `health models`) |
| How often does CHEAP fall through? | `agent-manager measures fallback-insights` |
| Show every fallback event for a run | `agent-manager events list --run=<id>` |
| Tail the typed event log for a specific type | `agent-manager events list --type=runner.fallback.attempted --since=1h` |

The CLI never bypasses the API — every command exercises the same HTTP
surface above, so anything the CLI can answer is something a downstream
script or skill can also fetch from the API.

`agent-manager health models` is a persisted execution-observation snapshot. It
answers whether a model has recently succeeded or failed in managed runs; it is
not a runner inventory and must not be used to infer that a model is absent.
For Codex, runner availability comes from the resource-owned catalog
(`resource-codex models list --json`), which queries the installed Codex runner
first and uses the local compatibility cache only when that probe is
unavailable.

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
agent-manager measures cohort show imported-baseline-2026-08 --json
agent-manager measures select-cohort --cohort imported-baseline-2026-08 --json
agent-manager measures episode-cohort --cohort imported-baseline-2026-08 --json
agent-manager measures cohort-report --cohort imported-baseline-2026-08 --json
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

- Renders `HistoryBanner` above Usage › Details (`/usage?tab=details`) when `history_days < 30`.
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

Every cohort command under `agent-manager measures` (`select-cohort`,
`episode-cohort`, `cohort-report`, `episode-compare`) takes one selector:
`--cohort NAME` (a named cohort; `measures cohort define|list|show|delete` manages them),
`--run-ids ID,ID` (at most 100), or `--filter-json '<InvocationFilter>'`, each over
`--window`. `episode-compare` takes the same selector with `left-`/`right-`
prefixes. `cohort-report` returns friction signals, time accounting and goal
progression together.

New diagnosis requests use the finite typed investigation lifecycle and require
an explicit bounded run set. Resolve a cohort first, inspect its omitted count,
then pass only the selected IDs:

```bash
agent-manager measures cohort-report --run-ids run-a,run-b --json
agent-manager run investigate --run-ids run-a,run-b --depth quick
agent-manager investigation wait <investigation-id> --timeout-seconds 180 --json
```

The legacy `--filter-json` and `--goal-id` selectors are retired from
`run investigate`; they could hide an unbounded or changing subject behind a
convenient writer. Historical legacy results remain readable, and explicit
approval/apply remains a separate authorized operation.

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
- `usage run <run-id> --facts`: ownership, unclassified reason, capability,
  intent class, segment, outcome, and classifier version per invocation.
- `run ledger <run-id>` and `measures capability-usage`: verified capability
  calls, outcomes, duration, population, and receipt validity.
- `measures capability-efficacy`: separate calls, successes, fallback-after,
  and abandoned counts; no blended score is emitted.
- `measures episode-cohort` and `measures episode-compare`: recurring fingerprint cost,
  distinct-run counts, population sizes, and before/after deltas.
- `run episodes` (and `run episodes --spans`): bounded struggle vocabulary,
  operator corrections, guidance repair, wait misuse, abandonment, and
  handoff continuation.
- `usage friction`, `usage friction-report` and `usage friction-dismiss`:
  recurring friction problems (three or more episodes of one fingerprint) in
  plain words with owner, counts, cost and status; reporting files one
  scenario-qa bug per problem. They replace `run publish-recurring-friction`.
- `run import-session-corpus` and `usage reindex`: named sampling strategy,
  candidate/omitted counts, checkpoint, per-item failure, transcript time
  basis, cost basis, and pinned classifier replay (`usage reindex [<run-id>]`
  refreshes by default; `--full` rebuilds every selected run).

Every surface carries either a validity or availability contract. Missing
receipts, too-small comparison populations, unclassified calls, and absent
transcript usage are reported explicitly rather than converted to zeroes.

- [Event Taxonomy](../internal/EVENT_TAXONOMY.md) — every typed event_type, its payload, and the schema_version contract.
- [Error Semantics](../internal/ERROR_SEMANTICS.md) — the `fallback.Reason` enum that drives `*.fallback.*` events.
- [Seams](../internal/SEAMS.md) — testability boundaries (stats engine, health store, eventlog).
