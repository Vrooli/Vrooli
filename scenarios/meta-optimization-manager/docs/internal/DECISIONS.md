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
| 2026-06-24 | Empirical trials run via **agent-manager's real sandboxed primitive** (`profile ensure` → `task create` → `run create --run-mode sandboxed` → poll `run get` → `run diff`, runner=opencode + local model); metrics are success + tokens + wall-time as a historical trend. | Readiness is ultimately empirical, not declarable from coverage. agent-manager already owns the sandboxed-agent-spawn primitive; it has no dedicated dispatch verb for trials. | `trials` is gated behind explicit invocation (expensive), always sandboxed; the Runner consumes agent-manager's `profile/task/run` CLI through the `CommandRunner` seam. | Revisit when trial cost/latency justifies a cheaper proxy. |
| 2026-06-25 | **agent-manager is the spawner ONLY; the verdict is MoM's.** The Runner returns EVIDENCE (sandbox diff + token/time metrics); a new MoM-owned **Evaluator** seam decides PASS/FAIL/ERROR. | Deciding whether a SWE task was solved is a trials-domain concern, not the spawner's. The earlier stub read a `verdict` field off the spawner's output — backwards. | `internal/trials/evaluator.go` (deterministic-first oracle, else agent-judge); the verdict is never fabricated — any uncertainty/failure is `VerdictError`, still recorded. | Revisit if a family needs a richer judging protocol than oracle+single-judge. |
| 2026-06-25 | **Success is defined by a committed fixture corpus**, one per family (`trials/fixtures/<family>/`: spec + deterministic oracle + minimal `target/`). The Guide space defines *which families exist*; fixtures define *what "solved" means*. | A deterministic substrate is what makes the verdict reproducible and the trend comparable over time. | Evaluation copies `target/`, applies the agent's diff, runs the oracle (exit 0 = solved); negatives pass on correct abstention (no substantive diff). A fixture content-rev participates in idempotency. Authoring contract: `docs/internal/TRIALS-FIXTURES.md`. | Revisit if real-scenario substrates become preferable to synthetic fixtures. |
| 2026-06-25 | **`trials run` defaults to a single task; `--all` opts into the full suite. Idempotency per (task, model, fixture-rev)** within a recency window reuses a recent identical run. | Trials are expensive and operator-invoked; cost guardrails prevent accidental double-spend. | CLI requires `--task`/`--suite`/`--all`; the service reuses a recent non-error matching run rather than re-dispatching. A storage-only `fixture_rev` column was added (no wire change). | Revisit if the reuse window or default scope proves wrong in practice. |
| 2026-06-25 | **Dropped the `workspace-sandbox` dependency from trials.** | agent-manager owns sandboxing internally; MoM never talks to workspace-sandbox directly — the earlier soft dependency was misleading. | `.vrooli/service.json` lists only `agent-manager` for trials; isolation/attribution come from `run create --run-mode sandboxed`. | Revisit only if MoM ever needs a sandbox outside an agent-manager run (not foreseen). |
| 2026-06-24 | **MoM does NOT self-register a search-hub provider** (OT-P2-002 / MOM-SEARCH-001 descoped). | The user need — readiness answerable via search ("how ready are we / where should I work") — is already met: cli-health federates every scenario's CLI into search-hub, so a query surfaces MoM's `status`/`focus`/`validate-docs` commands (verified live 2026-06-24). A dedicated provider emitting the inline `AttestedAnswer` envelope adds a separate code path + the protojson casing trap (search-hub's `decodeAttestation` wants snake_case keys + lowercase enum tokens, which a Connect/protojson endpoint would silently drop) for marginal extra value over command discovery. | No `api/internal/search`, no `.vrooli/search.json`, no `searchregister` wiring in MoM. The `AttestedAnswer` carrier added to search-hub in Phase 0 is kept (additive, optional) for any future provider. | Revisit if command discovery proves insufficient and an inline confidence+citations search answer becomes genuinely needed — then build the plain-HTTP snake_case provider (not protojson). |
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
