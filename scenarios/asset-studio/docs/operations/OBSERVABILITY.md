# Observability — Asset Studio

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

| Log | Source | How To Read | Details |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |

## Metrics

| Metric | Status | Details |
|---|---|---|
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| **Generation spend, rolling** | planned (`ASSET-P0-008`) | Actual cost by day, by spec, by identity, by campaign reference. The single most important operational number here: it is the only metric that can move fast enough to matter before a human notices. |
| **Spend per released artifact** | planned | Total cost divided by artifacts that passed the gate. Rising means renders are being discarded — either the specs are wrong or conformance is too strict, and the ratio does not say which. |
| **Conformance pass rate on first attempt** | planned (`ASSET-P0-010`) | The calibration signal for the whole scenario. Near 100% suggests the gate is not discriminating; near 0% suggests specs cannot express the identity. Neither extreme is good news. |
| Conformance queue age | planned | How long frames sit unresolved. Operator attention is the binding constraint here as it is in content-desk; a growing queue is the early symptom. |
| Render failure rate by backend | planned | Distinguishes gateway problems from spec problems. |
| Job duration percentiles by kind | planned | Image and video differ by orders of magnitude; a single aggregate would hide both. |
| Identity version churn | planned | Versions created per identity per month. High churn means the canon import is fighting hand edits, or an identity was never really settled. |
| Import items aborted on schema failure | planned (`ASSET-P0-003`) | Must be visible, not silent. A rising count means catalogue shape is drifting. |
| Blob storage footprint | planned | No pruning policy exists (`PROBLEMS.md`); the metric is the early warning that one is needed. |
| Performance budgets | deferred | Define in `../internal/PERFORMANCE.md` once real render durations exist. |

## Alerts / Health

The generated scenario has lifecycle health checks for API and UI. Add
deployment-specific alerts only when deployment target and operator
expectations are known.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| No baseline for any metric above | Every threshold would be a guess. Conformance pass rate in particular cannot be judged without knowing what a healthy rate looks like for this pipeline. | The P0 slice. Record the numbers before setting a single alert. |
| Cost estimates are unvalidated against actuals | An estimate that is systematically low makes the P1 budget useless — it would authorise spend it did not predict. | `ASSET-P0-008` records both from the first job; compare them before `ASSET-P1-006` sets a budget. |
| No drift signal across generations | Per-frame conformance cannot see the slow walk where every frame passes and the tenth is nothing like the first. | `ASSET-P2-003`, once several accepted generations exist for one identity. |
| Product usage telemetry | Cannot validate adoption or the monetization hypothesis. | Add before any external launch. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
