# Decisions — Meta-Optimization Manager

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
| 2026-06-24 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs + maturity metadata in `docs/manifest.json`. | Revisit when the scenario adopts a different template or doc contract. |
| 2026-06-24 | Scope is a **thin, read-mostly aggregator** — measure/surface/route only. | This scenario replaces the meta-optimization team's *measurement* surface, not its judgment. | Never re-implements an owner's measurement, never does the improvement, never owns the denominators, never makes the substrate/tiering/nomination/root-cause call. | Revisit if a needed measurement has no owner scenario to delegate to. |
| 2026-06-24 | Four domains: `coverage`, `convergence`, `focus`, `trials`. Stewardship / intake / contrarian / team-audit themes are **excluded**. | Those themes are irreducibly judgment; programmatizing them would weaken not strengthen them. | The meta-optimization team keeps those as agentic work; this scenario does not absorb them. | Revisit if an excluded theme develops a genuinely metric-driven core. |
| 2026-06-24 | Denominators are **distributed** — each space doc lives with its owner (Answer→search-hub, Validate→test-genie, Guide→prompt-manager), read via a `space --projection <p> --json` verb. | Co-locating the denominator with the numerator owner minimizes drift. | This scenario depends on that shared verb existing on each owner; it never copies a denominator locally. | Revisit if an owner cannot host its own denominator. |
| 2026-06-24 | Attestation rides as an optional `AttestedAnswer` sub-message on search-hub `SearchHit` (mirrors the existing `MeasureHit` carrier) — **not** a new result class. | `MeasureHit` is the proven precedent; providers stay decoupled via `ResultMapping`. | Trust travels inside the result, separate from relevance score; lives in search-hub `routing.proto`. | Revisit if a richer/standalone attested-answer result kind becomes necessary. |
| 2026-06-24 | **Denominator-confidence is mandatory and recursive.** | A scoreboard must never imply false completeness. | Every coverage number is paired with `AUTHORITATIVE \| PARTIAL \| SKETCH` + rationale ("X% complete against a Y-confidence denominator"). | Load-bearing honesty invariant — do not relax. |
| 2026-06-24 | Convergence is **included** despite the judgment-exclusion rule. | Its core is genuinely metric-driven (frozen-metrics, date-compare staleness, clean-scan eligibility, coordinated-edit counts). | A `convergence` domain mechanizes the reference-pattern-fitness counts + reference health; tiering/substrate/nomination stay agentic. | Revisit if the coordinated-edit walkthrough proves unmechanizable. |
| 2026-06-24 | Empirical trials run via **agent-manager** (runner=opencode + local model) inside **workspace-sandbox**; metrics are success + tokens + wall-time as a historical trend. | Readiness is ultimately empirical, not declarable from coverage. | `trials` is gated behind explicit invocation (expensive), sandboxed, with diff attribution. | Revisit when trial cost/latency justifies a cheaper proxy. |
| 2026-06-24 | Documentation-first: PRD + requirements authored before any domain code. PRD was **hand-authored to the canonical template + deterministically validated** because the `prd-control-tower` LLM-generation backend returned HTTP 500. | The guide prefers `prd-control-tower prd generate`, but AI generation was unavailable; the deterministic validator (`prd validate`) was healthy. | PRD validates clean (0 violations) and drives the requirements registry. | Optionally regenerate via `prd-control-tower` once its LLM backend is healthy, if a different shape is wanted. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`../concepts/COVERAGE-MODEL.md`](../concepts/COVERAGE-MODEL.md) — the attestation contract + model these decisions assume
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
