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
| Federated query telemetry | telemetry | `query_telemetry` / `search-hub insights insights` | Query volume, zero-result rate, degraded-query count, rerank count, windowed p50/p95 latency, rolling recent-10 p50/p95 latency, explicit RFC3339 bounds, sample count/sufficiency, routing mode, selected/withheld/queued fan-out, stage latency, rerank candidates, and closed response-degradation cause | Treat latency as provisional until the response's `sample_sufficient` is true; compare the bounded window with recent-10 to distinguish recovery from historical outage data |
| Per-provider fan-out telemetry | telemetry | `query_telemetry_provider` / `search-hub insights insights` | Provider routed count, hit count, per-leg p50/p95 latency, active reranker leg, degraded count/rate, and top degradation reasons, all bounded by the requested window | slow or frequently degraded providers should be identifiable without ad-hoc probing |

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
| Per-provider latency | active | `search-hub insights insights --window <duration>` accepts `15m`, `2h`, or a bare day count and reports p50/p95 per provider from persisted fan-out legs; sparse providers return zero percentiles rather than implying stable evidence. |
| Provider degradation rate | active | `search-hub insights insights --window <days>` reports degraded percentage and reason buckets (`timeout`, `unreachable`, `http_error`, `reranker_unavailable`, `other`). |
| Declared query telemetry measures | active | `/measures/declarations` and `MeasuresService` serve `query_telemetry.federated-latency`, `query_telemetry.degraded-query-rate`, `query_telemetry.provider-degradation-rate`, and `federation.stuck-provider-count`; the first three use the same `InsightsRange` compute path and the last reads persisted recovery state. |
| Fleet routing envelope | active | The metrics store buckets telemetry by routing mode and selected fan-out (`1`, `2-6`, `7-12`, `13-24`, `25+`) with p50/p95 total and classifier/resolver/fan-out/rerank stage costs. Query text is never persisted. |

The classifier and embedding models are expected to remain resident through
host configuration. Search Hub only applies bounded, leg-specific rerank
budgets: 500 ms for the measured TEI cross-encoder path and 8 s for the LLM
fallback. A cold `qwen3:1.7b` load was measured at approximately 16.41 s,
which is outside the interactive budget and must be addressed by residency,
not by adding model-management code to the router.

## Alerts / Health

The generated scenario has lifecycle health checks for API and UI. Add
deployment-specific alerts only when deployment target and operator
expectations are known.

## Provider Performance & Reliability Budgets

Each provider's `.vrooli/search.json` declares a latency/reliability budget the
search maturity contract enforces. The provider's operability **class** is the
latency class and supplies a conservative default p95 budget when the descriptor
declares none — modeled by class, never by scenario name:

| Class | Default p95 budget |
|---|---|
| `local_index`, `local_live` | 1500 ms |
| `external` | 4000 ms |
| `async` | 15000 ms |

The optional `performance` block tightens the default or opts into telemetry:

| Field | Meaning | Finding when violated |
|---|---|---|
| `p95_ms` | p95 query-latency budget (overrides the class default) | `SEARCH_PERF_BUDGET_BREACH` (advisory) — latest run p95 exceeds the budget |
| `degraded_rate_max` | max fraction of real queries returning no result | `SEARCH_PERF_DEGRADED` (advisory) — latest run's empty-result rate exceeds it |
| `telemetry_required` | require measurable latency evidence to exist | `SEARCH_PERF_BUDGET_UNPROVEN` (**required**) — no run latency to measure |
| `minimum_samples` | smallest eval-run sample that may substantiate the provider SLO | `SEARCH_PERF_SAMPLES_UNPROVEN` (**required**) — latest run contains too few cases |

Latency and degradation are read from the **latest eval run's** aggregate
(`latency_p95_ms`, per-case empty results). Because an eval run is a small, noisy
sample, budget/degradation breaches are **advisory** — they surface operability
debt without gating certification. A provider that sets `telemetry_required` opts
into a **hard** gate: if no latency evidence exists it fails certification. This is
how Search Hub avoids "silently passing" a provider whose performance it cannot
prove, without claiming a guarantee from descriptor shape alone.

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
