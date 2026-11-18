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

Agent Manager now records one durable physical inventory sample every minute.
Each sample retains its source timestamp, completeness and unknown reasons,
provider-mapped active run IDs, and unmapped physical run IDs. History is kept
for 70 days. The estimator pairs quota observations with complete samples for
the same provider and current reset window. It requires at least ten minutes of
positive quota burn, fresh monotone quota points, a complete sample interval
that brackets the quota readings, and no sample gap over three minutes. Run
exposure counts only IDs present at both ends of each sample interval, so a
run's occupancy is never inferred from its logical status or configured
concurrency.

Available tiers report their quota provider/pool/window/account scope, quota
interval, utilization sample interval/count, actual contributing run IDs,
measured run-seconds, and observed quota burn. A single provider-reported
account scope is required, but the runner's credential identity is not mapped
to that opaque account scope; the evidence explicitly reports this limitation.
Multiple or unknown account scopes, unknown runners, incomplete snapshots,
stale or non-monotone readings, and unsupported allowance remain unavailable.
The configured baseline is the normal setting, not a hard cap: measured tiers
can raise or lower the actual admission ceiling, while the existing ceiling
validation still limits it to 9999. `PUT` returns HTTP 409 and leaves settings
unchanged for every unavailable tier.

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
