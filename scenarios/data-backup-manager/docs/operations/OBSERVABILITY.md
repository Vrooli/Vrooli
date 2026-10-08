# Observability — Data Backup Manager

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

The API emits request logs and exposes backup posture through `/health`.
The first production-grade validation loop is to configure a real
destination and plan, trigger a run, then verify a restore; until that
exists, health only proves the service and catalog are reachable.

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API and dependency reachability | healthy for local development |
| `/health` backup status | health | API | Flags overdue and failed backups across plans | unhealthy when any target is overdue past its plan cadence or its last run failed |
| Last-success per target | product/health | run history | Confirms every registered target has a recent good backup | alert if a target has no success within its plan window |
| Last-verified per target | product/health | restore-verify runs | Confirms backups are actually recoverable, not just written | alert if verify is stale or last verify failed |
| Storage usage vs cap (per destination) | health | destination stats | Detects approaching/at the alert+block limit | warn near cap; block writes at cap (never silent eviction) |
| Backup outcome events | telemetry | run/restore lifecycle | Emitted for infra-health / system-monitor to consume | every run and restore emits success/failure |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| test-genie result | validation | `make test` | scenario correctness evidence | all required phases pass |

## Event Emission

Backup and restore outcomes are emitted as events so platform monitoring
(infra-health, system-monitor) can alert on them rather than polling
this scenario. Each run and each restore emits a success/failure event
carrying target, destination, plan, and outcome. Overdue/failed state is
also reflected synchronously in `/health` so a single health probe
surfaces recoverability risk.

## Logs

| Log | Source | How To Read | Notes |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |

## Metrics

| Metric | Status | Notes |
|---|---|---|
| Last-success / last-verified per target | active | Primary recoverability metrics; watch these first during an incident. |
| Storage usage vs cap per destination | active | Capacity-planning and alert+block trigger. |
| Run + restore success rate | active | Reliability of the backup loop over time. |
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Performance budgets | active | Lighthouse and build budgets are defined and validated in `../internal/PERFORMANCE.md`. |

## Alerts / Health

Beyond the lifecycle API/UI health checks, `/health` flags overdue and
failed backups, and run/restore outcome events are emitted for platform
monitoring. The operator's first-look signals during an incident are
last-success-per-target, last-verified-per-target, and storage usage vs
cap. A target that is overdue, failing, or failing verification is a
recoverability risk and should alert.

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
