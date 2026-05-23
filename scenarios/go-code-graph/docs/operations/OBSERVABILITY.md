# Observability — Go Code Graph

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
| `/health` status | health | API | API and dependency reachability | healthy locally |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| Determinism gate result | validation | `make test` (`bas/fixtures/`) | Extracted graphs match `expected-graph.json` byte-for-byte | required to ship a change |
| Performance regression result | validation | CI perf suite | Extract latency within SLA (≤200 files <5s, ≤2000 files <30s) | required to ship a change |
| `extract_duration_seconds` | metric (planned) | API per-call | Distribution of Extract latency by module size bucket | p95 within SLA |
| `rewrite_apply_partial_failures_total` | metric (planned) | API per-call | Count of `ErrApplyPartial` responses | non-zero is a bug signal worth investigating |
| `extract_warnings_total` | metric (planned) | API per-call | Count of partial-graph warnings by kind | informational; tracks input-quality trends |

## Logs

| Log | Source | How To Read | Notes |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. Per-call entries include `request_id`, `scenario_path`, `op_kind` (extract/rewrite_plan/rewrite_apply), `duration_ms`, `warning_count`, `error_kind`. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Recent-calls telemetry | API in-memory ring buffer | Visible in the explorer UI's diagnostics page | Bounded at 256 entries; lost on restart. UI-only convenience surface. |

## Metrics

| Metric | Status | Notes |
|---|---|---|
| Extract latency (per call) | planned | Bucketed by module size: small (≤200), medium (≤2000), large (>2000). |
| Extract success rate | planned | Catastrophic-error rate; warnings are not failures. |
| Rewrite apply success rate | planned | Fully-applied vs. partial-failure vs. plan-mismatch. |
| Concurrent calls in flight | planned | Per-path mutex queue depth. |
| Consumer identification | deferred | Connect-RPC client metadata (cartographer / cli / explorer) for cross-cutting analysis. |

## Alerts / Health

The scenario has lifecycle health checks for API and UI. Deployment-specific alerts (PagerDuty, Slack, etc.) are not in scope for v1 since the scenario runs locally.

If/when remote-exposure happens (deferred — see [`DEPLOYMENT.md`](DEPLOYMENT.md)), candidate alerts:

- API `/health` failing for >2 minutes → page operator.
- `rewrite_apply_partial_failures_total` rate spike → investigate for implementation bug.
- p95 `extract_duration_seconds` over SLA for the medium-module bucket → investigate `go/packages` regression or input growth.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Per-call metrics (counter/histogram) | Cannot detect performance regressions in production; only in CI fixtures. | Add when first remote consumer goes live or when CI fixtures stop being representative. |
| Consumer attribution | Cannot tell which consumer caused a regression or warning spike. | Add when more than one consumer is in production. |
| Operation Log query telemetry (P1) | Cannot tell how often the audit trail is read. | Add alongside REQ-P1-002 if the audit trail is exposed via UI. |
| Cost telemetry | Cannot evaluate hosted/SaaS unit economics. | Add only if remote-exposure happens. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business signals (n/a — infrastructure scenario)
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
