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
| 2026-06-22 | One budget source of truth: `performance.budgets` in `.vrooli/testing.json` (ms/bytes). | Two config files (`.vrooli/perf-budgets.json` read by the budgets domain, plus `performance.go_build_max_seconds`/`_ui_build_max_seconds` read by the benchmark runner) emitted the same `PERF_BUDGET_BREACH_*` codes and could disagree. | `perf-budgets.json` deleted with no fallback; both the benchmark runner and the budgets domain read `performance.budgets`. The **benchmark only MEASURES** (informational `OverBudget` flag); the **budgets domain is the sole emitter** of `PERF_BUDGET_BREACH_*` (Contract Decision 8.1). Units are ms/bytes (the `_seconds` axes are gone). | Revisit only if budgets must vary per test preset (would need a per-preset block). |
| 2026-06-22 | Freshly-MEASURED axes are build+bundle; lcp/startup/component-commit are measured continuously out-of-band — but their budget CHECK is still synchronous (reads the latest persisted sample). | The synchronous performance phase can only hermetically measure what it builds. lcp/component come from a captured CDP trace (analysis) and startup from a target restart — neither is safe to *measure* inside the gated phase of a `vrooli scenario test`. The *check* against an already-persisted sample is cheap and safe. | The `ExecutionOrchestrator` freshly measures build+bundle (+Lighthouse-if-UI) only. A declared budget on an ungated axis surfaces an INFO `PERF_BUDGET_AXIS_UNGATED` advisory so it can't masquerade as freshly-measured protection; the axis is still gated — the synchronous Performance phase reads the latest persisted sample and a breach fails the suite run, though the value reflects the last out-of-band measurement, not this run's code. A gated run that measures NO surface reports `SKIPPED`, never a false `PASSED`. | Revisit if a restart-safe startup window or an in-gate trace capture lands. |
| 2026-06-22 | Invert the trend write dependency: producers depend on the `perfsample` substrate DTO + a narrow consumer-owned writer; the concrete `trend.Store` is wired from the composition root (`main.go`). | Producer domains (analysis, benchmark, startup) and the budgets domain imported the `trend` domain directly — a sibling-domain import the layering rule (correctly) flagged. | The `Sample` DTO moved to `api/internal/perfsample` (shared substrate); `trend.Sample` is now an alias. Producer/handler packages no longer import the trend domain; `trend` keeps its read-model query/CLI/UI surface (it is a real product surface — not reclassified as substrate, Contract Decision 8.2). | Revisit only if the sample shape needs producer-specific variants. |
| 2026-06-22 | Per-flow (Tier-1 interaction) budgets: the browser CAPTURE runs out-of-band on the capture-sweep cadence; the per-flow budget CHECK runs synchronously in the Performance phase. | A targeted interaction capture restarts the target in profile mode and drives a browser — unsafe inside a `vrooli scenario test` (the same lifecycle-collision reasoning that keeps `startup` out of the gated phase). But per-flow journeys (scroll/drag/click on a budgeted flow) still need a regression gate, and checking an already-persisted sample is safe inside the phase. | New `SweepService` (`sweep run <scenario>`) loops `performance.budgets.flows`: audit `--workflow <slug>` → analyze → persist a **flow-tagged** `perf_samples` row (`flow` column). The validation handler then emits per-flow `PERF_BUDGET_BREACH_<FLOW>_<AXIS>` ERROR findings off those flow-tagged samples *inside* the Performance phase, so a regression fails the test-genie Performance phase — and therefore the suite run (`vrooli scenario test` exit 1) — without the browser capture ever running in the gated phase. Build/bundle/startup stay scenario-level (no per-flow build). | Revisit if an in-gate trace capture or a restart-safe window lands. |
| 2026-06-22 | The perf gate is the SUITE RUN (`vrooli scenario test` / `test-genie execute`), NOT `git-control-tower baseline diff`. | Verified during the Phase-5 sweep proof: a `VALIDATION_STATUS_FAILED` Performance phase makes `computeSuiteVerdict` return `SuiteVerdictFail` (it ignores `Optional` for *failed* phases), so the suite run exits non-zero. But baseline-diff buckets phase results into surfaces (structure/rules/tests/workflows/visuals) and the `performance` phase maps to NONE of them, so `bucketPhaseDiffs` drops it — a perf regression never moves the baseline-diff verdict. Earlier comments/docs that said "fails baseline-diff" were inaccurate and were corrected to "fails the suite run". | All perf-budget gate copy (budgets.go, validation/sweep handlers, maturity.json, CLI manifest, these docs) now says "suite run / Performance phase", not "baseline-diff". The proof: loose flow budget → `VALIDATION_STATUS_PASSED`; tight flow budget → `VALIDATION_STATUS_FAILED` with `PERF_BUDGET_BREACH_<FLOW>_*` ERROR findings, on a real persisted sample. | Revisit if a `performance` surface is ever wired into git-control-tower baseline diff (deferred — perf-phase re-measures noisy build/Lighthouse axes per tree, which would flake a clean-vs-dirty diff). |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
