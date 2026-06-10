# Decisions — Scenario Completeness Scoring

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-10 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-06-10 | Greenfield rewrite; legacy REST API and JSON field names die with it. | scenario-status-layer plan §8: nothing called the old API (verified; the one CLI consumer, swarm-manager, degrades gracefully — see PROBLEMS.md). | New proto `ScoreService` is the only contract; old `/api/v1/scores` shapes are gone. Reference copy: git 9926067d + /tmp/scs-old-reference-20260610. | Never (greenfield rule). |
| 2026-06-10 | Maturity/freshness semantics are imported, not owned. | `packages/maturity-go` (dimension vocabulary + R0–R4 gates) and `packages/freshness-go` (frozen treedigest spec, required-phase SSOT, verdict core) are shared with ecosystem-manager and test-genie. | This scenario adds NO local policy: no per-scenario freshness phases (anti-gaming operator decision), no custom rung gates. Rung labeled "as of digest", never EM's live state. | A shared-package major version change. |
| 2026-06-10 | Rung evidence prefers per-finding detail, approximates from phase status otherwise. | On-disk phase results only started carrying `findings` (ArchitectureFinding, proto-int enums) on 2026-06-10 via test-genie's `writePhasePointer`. | Failed phase without findings = ≥1 error in that phase's dimension, flagged `approximate`. Self-heals as suites re-run. | When fleet history is fully re-stamped. |
| 2026-06-10 | Operational-target bar for the R4 gate is 100%. | EM reads its profile's target; this scenario has no profile concept. | Strictest interpretation: R4 satisfied only when all declared targets pass (or none are declared). | If a consumer needs profile-aware rungs, plumb a target parameter through GetScore. |
| 2026-06-10 | Requirement "passing" vocabulary mirrors test-genie's NormalizeDeclaredStatus. | Fleet registries use pending/implemented/passing/complete/passed/…; old SCS counted only passed/complete/done. | `statusPasses` = passed/passing/complete/completed/done/implemented/validated; keep in lockstep with test-genie `internal/requirements/types/status.go`. | test-genie vocabulary change. |
| 2026-06-10 | Composite keeps the legacy weights (quality 50 / coverage 15 / quantity 10 / UI 25) and classification bands; two metrics re-based. | Individual test counts are not in the cached artifacts. | Test pass rate → phase pass rate (skipped excluded); test-to-requirement ratio → declared-validation coverage; phase-count quantity bands OK 5 / Good 10 / Excellent 14. Legacy validation penalties NOT ported (standards/structure findings now surface via the rung headline). | If a consumer needs old-score comparability (none does — greenfield). |
| 2026-06-10 | No persistence: scores computed per request, no cache, no history. | <1s warm is met by artifacts being small and local (RPC ~35ms observed); history is a §12 non-goal. | No SQLite tables owned by scoring domains; no cache invalidation complexity. | A measured perf regression or a demonstrated trends consumer. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
