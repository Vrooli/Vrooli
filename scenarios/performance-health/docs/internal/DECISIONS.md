# Decisions — Performance Health

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-21 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-06-21 | Execution-mode validate runs benchmark + Lighthouse-if-UI but NOT startup. | `include_execution=true` (test-genie's Performance delegate) needs to measure-and-gate, but startup measurement restarts the target scenario. During a `vrooli scenario test <target>` run the target IS the process under test, so an in-phase restart collides with the test harness lifecycle (the `internal/startup` package doc-contract already states it is "never invoked by a test-genie phase"). | The `ExecutionOrchestrator` keeps an optional startup seam (nil in production wiring) so the build/bundle/Lighthouse axes gate per-run, while the `startup` axis stays standalone-fed: `startup.Service` cross-writes `perf_samples.startup_ms` on each `startup measure`, so the startup budget gates whenever a recent measurement exists (capture-fed, like the analysis LCP/component axes). | Revisit if a safe "measure a non-self, not-under-test target's startup" capability lands, or if the harness exposes a restart-safe window. |
| 2026-06-21 | Remove the `p95` budget/trend axis outright (vs. mark experimental). | `p95_ms` had no honest producer and could never trip — a dead axis under the greenfield no-dead-code rule. | Removed end-to-end: proto field 9/8 reserved + regenerated, Go core, SQLite schema, CLI flags, UI, and the `PERF_BUDGET_BREACH_P95` maturity finding. Existing local trend DBs keep a harmless unused `p95_ms` column (the EnsureSchemas drift check only fails on *missing* declared columns). | Revisit if a real latency producer (e.g. a request-percentile capture) lands; re-add as a new field number. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
