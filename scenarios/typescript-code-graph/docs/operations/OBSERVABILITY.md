# Observability — TypeScript Code Graph

This document records logs, metrics, telemetry, health checks, and business/product signals for the scenario.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us consumers are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment?

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API + sidecar + (optional) SQLite reachability | healthy locally |
| Sidecar status | health | sidecar supervisor | Sidecar process is `ready` | required for all `graph` / `rewrite` calls |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| Determinism gate result | validation | `make test` (`bas/fixtures/`) | Extracted graphs match `expected-graph.json` byte-for-byte | required to ship a change |
| Leading-comment fidelity gate | validation | `bas/fixtures/ts-jsdoc-tags/` test | Leading-comment metadata is preserved verbatim | required — load-bearing for rcl migration |
| Performance regression result | validation | CI perf suite | Extract latency within SLA (≤200 files <5s, ≤2000 files <30s) | required to ship a change |
| Sidecar chaos test result | validation | CI chaos suite | Kill-and-restart recovers within budget | required to ship a change |
| `extract_duration_seconds` | metric (planned) | API per-call | Distribution of Extract latency by project size bucket | p95 within SLA |
| `sidecar_restarts_total` | metric (planned) | sidecar supervisor | Count of supervisor-driven sidecar restarts | non-zero is investigation-worthy; spike is an alert |
| `sidecar_unavailable_responses_total` | metric (planned) | API per-call | Count of `ErrSidecarUnavailable` returns | spike indicates sidecar instability |
| `rewrite_apply_partial_failures_total` | metric (planned) | API per-call | Count of `ErrApplyPartial` responses | non-zero is a bug signal |
| `extract_warnings_total` | metric (planned) | API per-call | Count of partial-graph warnings by kind | informational; tracks input-quality trends |

## Logs

| Log | Source | How To Read | Notes |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Per-call entries include `request_id`, `scenario_path`, `op_kind` (extract/rewrite_plan/rewrite_apply), `duration_ms`, `warning_count`, `error_kind`. |
| Sidecar logs | API process (inherits sidecar stderr) | `make logs` | Sidecar logs are tagged with `[sidecar]` prefix for grep-ability. Captures `ts-morph` errors and IPC framing errors. |
| Supervisor logs | API process | `make logs` | Spawn/restart events tagged `[supervisor]`: `spawn_attempted`, `handshake_succeeded`, `sidecar_exited`, `restart_scheduled`, `permanently_unhealthy`. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Recent-calls telemetry | API in-memory ring buffer | Visible in the explorer UI's diagnostics page | Bounded at 256 entries; lost on restart. UI-only convenience surface. |

## Metrics

| Metric | Status | Notes |
|---|---|---|
| Extract latency (per call) | planned | Bucketed by project size: small (≤200), medium (≤2000), large (>2000). |
| Extract success rate | planned | Catastrophic-error rate; warnings are not failures. |
| Sidecar uptime | planned | Time since last `ready` transition. |
| Sidecar restart frequency | planned | Restarts per hour. |
| Rewrite apply success rate | planned | Fully-applied vs. partial-failure vs. plan-mismatch. |
| Concurrent calls in flight | planned | Per-path mutex queue depth. |
| Consumer identification | deferred | Connect-RPC client metadata (cartographer / rcl / cli / explorer) for cross-cutting analysis. |

## Alerts / Health

The scenario has lifecycle health checks for API + UI + sidecar. Deployment-specific alerts (PagerDuty, Slack, etc.) are not in scope for v1 since the scenario runs locally.

If/when remote-exposure happens (deferred — see [`DEPLOYMENT.md`](DEPLOYMENT.md)), candidate alerts:

- API `/health` failing for >2 minutes → page operator.
- Sidecar `permanently_unhealthy` → immediate page.
- `sidecar_restarts_total` rate spike → investigate.
- `rewrite_apply_partial_failures_total` rate spike → investigate.
- p95 `extract_duration_seconds` over SLA → investigate `ts-morph` regression or input growth.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Per-call metrics (counter/histogram) | Cannot detect performance regressions in production; only in CI fixtures. | Add when first remote consumer goes live or when CI fixtures stop being representative. |
| Sidecar internal metrics (Project init time, `ts-morph` walk time, IPC serialization time) | Cannot tell where extraction latency is coming from without ad-hoc profiling. | Add alongside the first major sidecar perf regression. |
| Consumer attribution | Cannot tell which consumer caused a regression. | Add when more than one consumer is in production (rcl will be the second). |
| Operation Log query telemetry (P1) | Cannot tell how often the audit trail is read. | Add alongside REQ-P1-002 if the audit trail is exposed via UI. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business signals (n/a — infrastructure scenario)
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
