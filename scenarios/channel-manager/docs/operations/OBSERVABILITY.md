# Observability — Channel Manager

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

Two kinds live here and they are easy to confuse. **Account metrics** describe how
a platform is treating an identity; they are product data owned by `signals` and
documented in `DATA.md`. **Operational metrics** describe whether this scenario is
working. Only the second kind belongs in this document.

| Metric | Status | Details |
|---|---|---|
| Queue depth and overdue count | planned | Actions past their window that were never executed. The primary sign that manual execution has fallen behind — and under manual execution, falling behind is the normal failure. |
| Cadence headroom per identity | planned | Actions used against the platform ceiling today. Approaching the ceiling means a program is over-scheduled, which is a descriptor defect rather than an operational one. |
| Execution outcome mix | planned | Succeeded / retried / failed / cancelled, split by executor. A rising manual share after the browser executor lands means dispatch is degrading. |
| **AI-navigation fraction** | planned | Share of browser-executed actions that required vision-based navigation rather than a stable selector. **This is the revenue hypothesis in `../business/MONETIZATION.md`** and its kill signal; measure it from the first week the browser executor runs. |
| Quarantine rate | planned | Identities failing their trust gate, per program. The first real evidence about whether D-002's speculative defaults are any good. |
| Flag rate and resolution latency | planned | How often decay is suspected, and how long a paused identity stays paused. A high rate with fast dismissal means the decay thresholds are too tight. |
| Descriptor seed status | planned | Whether the loaded descriptors match the files on disk. A silent divergence would apply stale cadence ceilings. |
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Performance budgets | deferred | Define in `../internal/PERFORMANCE.md`. |

## Alerts / Health

Lifecycle health checks cover API and UI. Beyond those, this scenario's meaningful
conditions are slow rather than sudden, which changes what an alert should be.

| Condition | Why it matters | Response |
|---|---|---|
| Actions overdue past their window | The window is the point. An action executed hours late is a different behavioural signature from the one the program declared. | Surface in the console's due list, not as a page. Manual execution is expected to slip. |
| An identity paused by a flag | Its queue has stopped and nothing will resume it automatically (D-004). | Operator decision. Never auto-resume. |
| A gate waiting past its interval | The program has stalled and the identity is neither progressing nor failing. | Surface it. A silently stalled warming run is the failure mode most likely to go unnoticed for weeks. |
| Vault unreachable | Every browser and API execution fails; manual execution is unaffected. | Health check reports the dependency. Actions fail terminally rather than being marked complete. |

Deliberately **not** alerts: reach drops (that is a flag, and flags are evidence for
an operator rather than pages), and failed individual actions (retry classification
handles them).

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| No platform metric ingestion | Every observation is entered by hand at P0, so baselines are only as regular as the operator. | `CHANMGR-P1-007` for published-post metrics; automated capture of per-identity reach has no requirement yet and probably needs the browser executor. |
| AI-navigation fraction unmeasurable until P1 | The revenue hypothesis cannot be evaluated at all. | `CHANMGR-P1-001`. |
| No cost telemetry | Cannot evaluate the allowance economics flagged in `../business/MONETIZATION.md` — specifically whether identity count needs to be a pricing dimension. | When BAS credit consumption becomes attributable per identity. |
| Warming efficacy unmeasurable | The capability kill signal cannot be evaluated without several identities and a control. | Several graduated identities plus at least one that posted without warming. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
