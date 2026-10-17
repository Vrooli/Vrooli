# Observability — Vrooli Memory

> **Scenario-specific signals to instrument** (designed, not yet implemented):
>
> - **Frontier size vs. target** — the primary health signal. A frontier
>   persistently over target means compaction is not keeping up, and ambient
>   recall quality degrades before anything errors.
> - **Compaction pass outcome** — collapsed / no-op / aborted-on-inference-error.
>   Aborts are expected and benign; a *rising* abort rate means ai-gateway
>   trouble.
> - **Unclassified backlog** — entries appended without a facet because
>   classification failed. Non-zero is tolerable; growing is not, because
>   unclassified entries are routed to no retention policy.
> - **Pinned-set size vs. pin budget** — the pinned set is the one structure
>   with no automatic relief valve, so this is the signal that curation is
>   overdue. Token overflow (`VMEM-P0-006`) is the *outer* limit and a real
>   operator-facing state rather than an error; the pin budget binds far earlier,
>   because the constraint that actually degrades ambient recall is attention,
>   not tokens.
> - **Pins pending review, and open merge proposals** — the response half of the
>   metric above (`VMEM-P1-010`). A growing queue against a flat pinned-set
>   size means curation is not keeping up; both flat means the loop is healthy.
> - **Summary generation depth** — how many times content has been re-encoded.
>   The observable proxy for the drift risk deferred in D-012.

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
