# Observability — Search Hub

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API and dependency reachability | healthy for local development |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| test-genie result | validation | `make test` | scenario correctness evidence | all required phases pass |
| Federated query telemetry | telemetry | `query_telemetry` / `search-hub insights insights` | Query volume, zero-result rate, degraded-query count, rerank count, and total p50/p95 latency | p95 should stay below the operator budget for the active provider set |
| Per-provider fan-out telemetry | telemetry | `query_telemetry_provider` / `search-hub insights insights` | Provider routed count, hit count, per-leg p50/p95 latency, degraded count/rate, and top degradation reasons | slow or frequently degraded providers should be identifiable without ad-hoc probing |

## Logs

| Log | Source | How To Read | Notes |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |

## Metrics

| Metric | Status | Notes |
|---|---|---|
| Product activation | deferred | Define after PRD users and workflows are real. |
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Performance budgets | deferred | Define in `../internal/PERFORMANCE.md`. |
| Per-provider latency | active | `search-hub insights insights --window <days>` reports p50/p95 per provider from persisted fan-out legs. |
| Provider degradation rate | active | `search-hub insights insights --window <days>` reports degraded percentage and reason buckets (`timeout`, `unreachable`, `http_error`, `reranker_unavailable`, `other`). |
| Declared query telemetry measures | active | `/measures/declarations` and `MeasuresService` serve `query_telemetry.federated-latency`, `query_telemetry.degraded-query-rate`, and `query_telemetry.provider-degradation-rate` from the same `InsightsRange` compute path. |

## Alerts / Health

The generated scenario has lifecycle health checks for API and UI. Add
deployment-specific alerts only when deployment target and operator
expectations are known.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Product usage telemetry | Cannot validate monetization or adoption. | Add before public launch or monetization review. |
| Cost telemetry | Cannot evaluate hosted/SaaS unit economics. | Add before managed deployment. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
