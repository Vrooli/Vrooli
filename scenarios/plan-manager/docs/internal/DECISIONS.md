# Decisions — Plan Manager

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

These were converged in the founding idea-workshop (2026-06-25) before any code.
They are the load-bearing choices; do not relitigate without a stated trigger.

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-25 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-06-25 | plan-manager is the **plan-logic SSOT**, exposed as a guided wizard runtime — not just a plan store. | Plan logic is scattered/trapped today (swarm-manager `phased-plan-drain`, the prose authoring skill, project hygiene, the `vrooli plans` CLI). | Other surfaces become thin consumers that delegate here (inversion of control). | If a second authority for plan logic is proposed, reconcile rather than fork. |
| 2026-06-25 | The deterministic **wizard + validators** are the primary local-model unlock; just-in-time context injection is the multiplier. | North star: drive the tokens + intelligence cost of plan work low enough for local models. | Mechanical work (anchor, required-reading, refs, status, baseline scope) moves into code; the model supplies only genuine prose. | If empirical trials show prose authoring still needs a large model, revisit which steps are deterministic. |
| 2026-06-25 | **Storage = scenario owns logic over a durable `~/.vrooli` home store** (not a scenario-private DB). | Plans must be readable when the server is down and by the thin `vrooli plans` CLI. | plan-manager owns schema/validation/logic; persistence is process-independent. | If multi-host/remote storage is required, revisit the substrate. |
| 2026-06-25 | **Artifact = structured phases as first-class records**, markdown is a rendered view. | Computed status + staleness require structured data, not prose. | Phase status is a typed transition; the agent never hand-edits plan markdown. | If structured authoring proves too heavy for the smallest models, revisit. |
| 2026-06-25 | **Validation autonomy = compute + run, agent in the loop.** | Agents shouldn't reason about which scenarios to baseline. | plan-manager derives the exact baseline/check set and runs it on request; agents may run baselines, never commit. | If a fully autonomous background refresh is wanted, revisit. |
| 2026-06-25 | **Sequencing = build standalone first, invert consumers later** (OT-P2-002). | swarm-manager is mid-refactor; lower blast radius to prove standalone first. | Consumer inversion is deferred to a later phase. | After P0/P1 are green and proven standalone. |
| 2026-06-25 | **Code references via today's `[CODE:]`/code-facts now; unified code identifier is a drop-in upgrade later.** | The unified identifier is still being designed in another track. | Staleness is computable today; soft-depend, do not block. | When the unified code identifier ships, swap the locator. |
| 2026-06-25 | **Handoff ownership split:** plan-manager owns the *structured* handoff; the *prose* final-message catch-all is owned by the orchestration layer (agent-manager transcript → swarm-manager operating mode). | plan-manager is a tool agents call; it never sees the chat stream and must not read transcripts. | Prose detection/extraction/attribution-crossoff live at the spawner; plan-manager links to it by reference. | Never folds transcript-reading into plan-manager. |
| 2026-06-25 | **Findings are filed as candidate (unvalidated); an operator triages** before they become real bugs. | Small/local models produce noisy/transient "bugs". | A candidate finding is never presented as a confirmed bug; promotion is a separate human step. | If a reliable auto-validation gate is built, revisit promotion. |
| 2026-06-25 | **Project-level validation is consumed, not owned.** | It belongs with test-genie / scenario-validation / MoM's Validate projection. | plan-manager reads validation results; it does not test resources/packages/project. | If no owner emerges, reconsider. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
