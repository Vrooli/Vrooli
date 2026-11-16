# Operator pace API

`GET /api/v1/usage/pace` returns the currently selected pace, persisted target
deadline, effective admission ceiling, and a forecast for each schedule tier.
`PUT /api/v1/usage/pace` accepts `{"selected_tier":"<tier>"}`. Supported
selections are `at_reset`, `use_by_thursday`, `use_by_wednesday`, and
`baseline`. The extra-allowance tier is always returned, but selection is
refused until a provider reading identifies a separate extra-allowance budget.

The weekday deadlines use `usageforecast.DigestLocation()` and are capped by
each quota window's reset. `target_run_out_at` is an absolute RFC3339 instant
encoded in UTC. A selected tier and its exact target are stored in
`OrchestrationSettings.UsagePace`; `RunExecution.MaxConcurrentRuns` stores the
effective ceiling that Agent Manager's run-admission check enforces. Selecting
`baseline` restores the retained configured baseline.

Available slot counts are estimates. Each tier scales the provider-reported
projected run-out interval by the target interval and assumes quota burn scales
linearly with admitted concurrent runs. Results expose the source quota scope,
projected run-out, reset, observation time, and that assumption. No fresh or
projected quota data means no slot count. These are not per-run quota
measurements; provider quota percentage is not attributed to individual runs.

When no fresh quota reading exists, `GET` returns this exact response shape
(configured concurrency values are from the persisted settings):

```json
{
  "generated_at": "2026-11-16T12:00:00Z",
  "baseline_max_concurrent_runs": 10,
  "effective_max_concurrent_runs": 10,
  "tiers": [
    {"id":"at_reset","label":"Last until quota resets","status":"unavailable","reason":"no fresh provider quota forecast is available to estimate slots"},
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
