# Observability — Measures Health

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
| Federated measure-search p95 | active | `.vrooli/search.json` requires a 1500ms p95 and telemetry evidence. The interactive provider uses deterministic matching and canonical parameter resolution; unresolved parameters remain explicit rather than spending the request budget on an LLM extraction. |

## Interactive Search Boundary

The Search Hub provider is an interactive **measure-selection** surface. It
matches a declaration and resolves deterministic parameters such as a canonical
time window. It deliberately does not enable the optional LLM parameter
extractor by default: an unresolved enum or free-form parameter returns an
honest incomplete result instead of turning a federated search into a
long-running inference request. Call the selected measure's owning scenario
through its direct measures execute surface when a full computation is needed.

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
