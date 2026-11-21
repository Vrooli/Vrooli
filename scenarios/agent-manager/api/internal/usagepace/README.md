# Operator pace API

`GET /api/v1/usage/pace` returns the currently selected pace, persisted target
deadline, effective admission ceiling, and a forecast for each schedule tier.
`PUT /api/v1/usage/pace` accepts `{"selected_tier":"<tier>"}`. Supported
selections are `at_reset`, `use_by_thursday`, `use_by_wednesday`, and
`baseline`. The extra-allowance tier is always returned, but selection is
refused until a provider reading identifies a separate extra-allowance budget.

The weekday deadlines use `usageforecast.DigestLocation()`. The `at_reset`
tier targets the latest reset among all current quota windows, so a short
five-hour reset cannot silently stand in for the weekly horizon.
`target_run_out_at` is an absolute RFC3339 instant encoded in UTC. A selected
tier and its exact target are stored in
`OrchestrationSettings.UsagePace`; `RunExecution.MaxConcurrentRuns` stores the
effective ceiling that Agent Manager's run-admission check enforces. Selecting
`baseline` restores the retained configured baseline.

Agent Manager records one durable physical inventory sample every minute.
Each sample retains its source timestamp, completeness and unknown reasons,
provider-mapped active run IDs, and unmapped physical run IDs. History is kept
for 70 days. Terminal attached runs are not inventory candidates (their
harness process was started outside Agent Manager and was observed gone or
detached), and re-attaches of one harness session share the earliest run's
physical verdict.

The estimator measures each current quota window (provider, pool, window,
account scope, reset) on its own. Burn is the running maximum of the used
percent over the trailing 24 hours of the current reset chain (concurrent
sessions re-render older cached percents); it needs at least ten minutes of
history and a reading no older than ten minutes. Burn is divided by the
run-seconds of the runs that reported the quota (their source run IDs), counted
only between adjacent samples at most three minutes apart that both positively
observed the run. Every bias lowers the ceiling: account burn includes usage no
run reported, and incomplete or partial samples can hide a run but never invent
one. A window with no positive burn, no source run IDs, a stale reading, or a
source run reported under two account scopes is listed in `unavailable_scopes`
with its reason and does not hide another window's measurement. A tier is
available when at least one window is measured; its ceiling is the smallest
measured window ceiling.

Available tiers report their quota provider/pool/window/account scope, quota
interval, utilization sample interval/count, actual contributing run IDs,
measured run-seconds, and observed quota burn. When the provider reports no
account scope (Codex), membership rests only on the source run IDs, and the
scope evidence says so. The configured baseline is the normal setting, not a
hard cap: measured tiers can raise or lower the actual admission ceiling, while
the existing ceiling validation still limits it to 9999. `PUT` returns HTTP 409
and leaves settings unchanged for every unavailable tier.

When no fresh quota reading exists, `GET` returns this exact response shape
(configured concurrency values are from the persisted settings):

```json
{
  "generated_at": "2026-11-16T12:00:00Z",
  "baseline_max_concurrent_runs": 10,
  "effective_max_concurrent_runs": 10,
  "tiers": [
    {"id":"at_reset","label":"Last through the final active quota reset","status":"unavailable","reason":"no fresh provider quota forecast is available to estimate slots"},
    {"id":"use_by_thursday","label":"Use by Thursday","status":"unavailable","reason":"no fresh provider quota forecast is available to estimate slots"},
    {"id":"use_by_wednesday","label":"Use by Wednesday","status":"unavailable","reason":"no fresh provider quota forecast is available to estimate slots"},
    {"id":"extra_allowance","label":"Use extra allowance","status":"unavailable","reason":"provider quota readings do not identify a separate extra-allowance budget"}
  ]
}
```

Selecting an unavailable tier returns HTTP 409 and leaves settings unchanged.
Selecting `baseline` returns HTTP 200 and restores the configured admission
ceiling.

The authenticated Agent Manager CLI exposes the same boundary for steward
consumers:

```sh
agent-manager usage pace --json
agent-manager usage pace set --tier use_by_thursday --json
agent-manager usage pace set --tier baseline --json
```

Human output shows the selected tier, effective concurrency, each available
slot count, and explicit unknown reasons. The CLI does not write orchestration
settings directly.
