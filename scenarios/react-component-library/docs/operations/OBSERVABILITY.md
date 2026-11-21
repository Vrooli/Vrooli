# Observability — React Component Library

Behavioral claims are current only within the specific checks in the register. Other guidance and unverified descriptions below are **design intent**, not claims of current implementation. See the [behavior claim register](../internal/TESTING.md#behavior-claim-register).

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.


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

## Alerts / Health

The generated scenario has lifecycle health checks for API and UI. Add
deployment-specific alerts only when deployment target and operator
expectations are known.

## Workbench redesign signals (planned)

The D · Nocturne pages need signals before they can be called production-ready:

| Signal | Type | Purpose | Failure meaning |
|---|---|---|---|
| Findings projection latency/errors | metric + log | Review and Home | A slow or failed queue is not an empty queue |
| Coverage report unavailable count | metric | Coverage | A service outage must be visible, not a zero distribution |
| Activity projection errors | metric + log | Activity | A missing trail is not “no changes” |
| Evidence freshness by asset | metric | Home/Component/Coverage | Stale evidence must stay visible |
| Workbench route error rate | metric | All pages | A route that fails is not a page with no data |
| Empty vs unavailable outcome | structured log | All projections | The two states must never collapse |

These are design intent until the projections and their instrumentation exist. See [WORKBENCH-REDESIGN.md](../concepts/WORKBENCH-REDESIGN.md#data-and-contracts).

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
