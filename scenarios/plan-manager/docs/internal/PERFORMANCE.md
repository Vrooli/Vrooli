# Performance — Plan Manager

This document records performance budgets, current measurements, known
constraints, and the regression procedure for plan-manager.

## Purpose Of This Document

Use this document to answer:

- What performance budgets does plan-manager target?
- What has actually been measured?
- What constraints shape its performance?
- How do we catch performance regressions?

## Budgets

- Budgets are deferred until the first vertical slice exists. Setting
  numeric latency/throughput targets now would be inventing detail for an
  unimplemented system.
- Intended budget shape: interactive plan reads/writes via the API and the
  `plan-manager` CLI should feel instant against the local SQLite store, and
  any fan-out to other scenarios (context discovery, staleness, baseline)
  must stay bounded so the UI/CLI never blocks on a slow integration.

## Current Measurements

- Focused unit, race, lint, type-check, and UI test runs have validated the
  current context-discovery implementation, but no stable latency budget has
  been captured yet. Treat live timing observations as diagnostic evidence
  until a repeatable benchmark is added.

## Known Constraints

- Fan-out dependency: staleness and baseline/diff orchestration call out to
  other scenarios (code-facts, git-control-tower, test-genie /
  scenario-validation). These calls can be slow or unavailable, so reads
  must be bounded (timeouts/limits) and degrade gracefully rather than hang
  — a missing integration must not block plan reads.
- Context discovery fan-out is capped at six in-flight probe subprocesses by
  default, with each probe still bounded by the 20s per-probe ceiling. Author
  session start also launches a best-effort background prefetch from the title;
  `context-discover` without explicit concepts can reuse that pending batch,
  while `--refresh` or explicit concepts supersede it through the normal batch
  merge path.
- Local SQLite store: reads/writes go to the shared `~/.vrooli` store; this
  keeps plans available even when the server is down, but it means
  performance is tied to local disk and single-writer SQLite semantics.
- The Plan Manager CLI/API/UI read the same store through Plan Manager-owned
  logic, so the store layout should keep common reads cheap.

## Regression Procedure

- Until budgets and measurements exist, the regression procedure is to
  capture baseline numbers when the first vertical slice ships, then guard
  them.
- Intended procedure: run the scenario test suite via
  `vrooli scenario test plan-manager`, compare per-plan velocity and read
  latency against the recorded baseline, and treat a meaningful regression
  (especially a return to unbounded fan-out latency) as a failure to
  investigate before release.

## Cross-References

- [`SECURITY.md`](SECURITY.md) — security posture
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — velocity and health signals
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system architecture
