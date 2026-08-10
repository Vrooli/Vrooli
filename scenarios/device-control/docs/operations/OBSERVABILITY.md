# Observability — Device Control

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

### Scenario-specific signals

The signals that matter here are about honesty and exclusivity, not volume.

| Signal | Why it matters | Threshold |
|---|---|---|
| Capability snapshot age per device | A stale snapshot presented as current is the failure mode this scenario exists to prevent. | Age must be visible wherever a capability is asserted. |
| Lease contention rate and refusal count | Rising refusals mean consumers are competing for a device. Rising *grants without releases* means a leak. | Refusals: informational. Unreleased grants: investigate. |
| Resolution rung distribution per flow | A flow drifting from `semantic` toward `vision` is getting slower, costlier, and less deterministic — usually because a UI changed underneath it. | Drift toward `vision` is a regression signal, not a cost signal alone. |
| Redaction verification failures | A security event, not a quality metric. | **Zero.** Any non-zero value is an incident. |
| `unavailable` with no next action | Violates the error contract in [`../internal/ERROR-HANDLING.md`](../internal/ERROR-HANDLING.md). | **Zero by contract.** |

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |

## Metrics

| Metric | Status | Details |
|---|---|---|
| Product activation | deferred | Define after PRD users and workflows are real. |
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Performance budgets | deferred | Define in `../internal/PERFORMANCE.md`. |

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
- [`../internal/ERROR-HANDLING.md`](../internal/ERROR-HANDLING.md) — the disposition contract these signals measure
