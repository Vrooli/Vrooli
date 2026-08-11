# Observability — Program Runtime

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
| Typed inference usage | usage | ai-gateway `Usage` | enforce and explain per-session inference budgets | `cost_micros`, input tokens, and output tokens remain within configured ceilings |
| Delegated-run spend | usage | agent-manager run receipts | enforce and explain the separate delegated-run budget | delegated usage remains within its configured ceiling |
| Binding reachability | diagnostic | `program-runtime bindings doctor` | distinguish absent manifests from a stopped owner | unreachable count is investigated, not treated as unbound |
| Friction evidence | empirical | `programs mine`, `mine-refusals`, `mine-unresolved` | feed recurring runtime failures to meta-optimization-manager | durable shapes have a recent locator |

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

## Runtime Configuration And Counters

| Setting/counter | Source | Meaning |
|---|---|---|
| `PROGRAM_RUNTIME_INFERENCE_CEILING_MICROS` | API environment | Default per-session inference monetary ceiling; zero disables the default ceiling. |
| `PROGRAM_RUNTIME_DELEGATION_CEILING_MICROS` | API environment | Default per-session delegation ceiling; unmeasured agent-manager charges are never fabricated as zero. |
| `PROGRAM_RUNTIME_SUBMISSION_DEADLINE` | runner configuration | Supervisor deadline for one program submission. |
| session idle reclamation interval | API supervisor | One-minute scan for expired sessions. |
| telemetry outbox drain interval | telemetry store | Retry cadence for pending event delivery. |
| dead-letter window | telemetry schema/retention | Seven-day retention for undeliverable events. |
| program/refusal/unresolved retention | retention worker | 90 days for program evidence; 30 days for refusal and unresolved-name evidence. |
| skipped manifests | `bindings doctor` | Malformed or unreadable manifests kept out of the callable registry. |
| unreachable scenarios | `bindings doctor` | Bound scenarios whose discovery probe cannot resolve a live owner. |
| dormant bindings | binding condition response | Callable bindings with no recent invocation evidence. |
| dead-letter events | telemetry health | Events that exhausted retry policy and require operator inspection. |

## Alerts / Health

The generated scenario has lifecycle health checks for API and UI. Add
deployment-specific alerts only when deployment target and operator
expectations are known.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Product usage telemetry | Cannot validate monetization or adoption. | Add before public launch or monetization review. |
| Delegation cost receipt | agent-manager currently omits a per-run charge receipt. | Revisit PRT-P1-011 when agent-manager publishes an explicit charge contract. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
