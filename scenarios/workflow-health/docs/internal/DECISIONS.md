# Decisions — Workflow Health

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
| 2026-07-02 | Use `workflow-health` as the new scenario and `workflow` as the canonical Test Genie phase. | Test Genie still had a native `playbooks` phase whose name no longer matched the desired architecture. | New work targets workflow-health and the canonical phase name `workflow`; `playbooks` is migration-only aliasing. | Revisit only if the phase naming contract is explicitly reopened by platform owners. |
| 2026-07-02 | Keep Browser Automation Studio as runtime engine and make workflow-health the policy/intelligence owner. | BAS already owns browser execution; workflow-health needs validation, maturity, safety, search, and Test Genie provider semantics. | workflow-health calls BAS and stores artifact references instead of copying browser internals. | Revisit if BAS stops being the supported workflow execution engine. |
| 2026-07-02 | Split workflow search leaves by role. | Agents need runnable flows, while Test Genie needs validation cases and BAS actions are reusable fragments. | `bas/flows` become `workflow.flow`, `bas/cases` become `workflow.test`, and `bas/actions` become hidden `workflow.fragment` dependency context. | Revisit if Search Hub adds a stronger native workflow role model. |
| 2026-07-02 | Fail closed for mutating execution. | Workflow assets can alter scenario state; accidental primary database mutation is unacceptable. | Mutating execution requires safety metadata, explicit confirmation policy, seed/reset consistency, and routed isolation proof before any BAS call. | Revisit only with a stronger platform-wide execution sandbox. |
| 2026-07-02 | Keep Phase 3 autofix mechanical only. | Registry, metadata stubs, invalid execution mode, and legacy reset values are deterministic structural drift. Requirement linking, selector registry choices, unresolved subflows, and mutating safety declarations require product or safety judgment. | The fix registry implements only mechanical edits; safety and behavior-affecting findings remain visible manual work. | Revisit when a separate workflow-authoring policy can prove an edit is semantics-preserving. |
| 2026-07-02 | Introduce a narrow BAS execution seam before provider wiring. | workflow-health needs execution tests now, but provider protos and Test Genie migration are later phases. | `api/internal/execution` depends on a small BAS client interface and writes generic timeline/latest artifacts; generated provider/native detail can wrap this service later without changing fail-closed safety behavior. | Revisit in Phase 5 when `ScenarioValidationService` and native detail protos are mounted. |
| 2026-07-02 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start from the template maturity metadata but Phase 1 replaces starter product claims with workflow-health-specific contract docs. | Revisit when scenario adopts a different template or doc contract. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
