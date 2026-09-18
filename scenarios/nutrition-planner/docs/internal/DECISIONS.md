# Decisions — Nutrition Planner

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

The working product name is **Daily**. Scenario id is `nutrition-planner`.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

The table below consolidates the initial decision log from the canonical
specification (§25.2, `D-001`…`D-006`) with the architecture decisions
derived from the specification's domain, nutrition, planning, and
persistence requirements. Dates are the 2026-09-18 handoff unless a
decision is later superseded.

## Decision Log

| ID | Date | Status | Decision | Rationale |
|---|---|---|---|---|
| D-001 | 2026-09-18 | Confirmed user request | Produce one extremely detailed, self-contained document for a local implementation agent. | The handoff must be actionable without the hosted prototype, prior conversations, or source from that prototype (spec §1.4, `C10`, `C11`). |
| D-002 | 2026-09-18 | Confirmed product direction | Preserve low-effort nutrition planning while supporting variety, cost awareness, multiple diets, flexible entry, portability, and richer recipe views. | These are the confirmed needs `C01`–`C09`; the product fails if any is traded away for the others (spec §2.2). |
| D-003 | 2026-09-18 | Recommended default | Deliver R1 through staged R0 milestones; defer provider automation and billing unless commissioned. | R1 is the first personally useful release; R2/R3 add maintenance and commerce without blocking the core loop (spec §3.1, §22.2). |
| D-004 | 2026-09-18 | Recommended default | Use deterministic domain calculations and validated AI proposals, with explicit unknowns and immutable historical references. | Arithmetic, authorization, validation, and state transitions must be reproducible; model output never authorizes a rule, price, target, or stock change (`AI-01`, `NUT-02`). |
| D-005 | 2026-09-18 | Open pending repository inspection | Select the actual implementation stack and integration details using the target repository's conventions. | The handoff does not assert that a particular framework, persistence layer, or Vrooli service exists; the implementing agent confirms it locally (`ARCH-01`, `DEC-03`). |
| D-006 | 2026-09-18 | Open pending in-app personalization | Collect real meals, products, targets, supplements, prices, and preferences progressively; do not treat examples as personal data. | Personal targets and private purchases are unknown; a fresh real workspace must not silently inherit fixtures or the founder's diet (`DEC-01`, `DEC-02`). |
| D-007 | 2026-09-18 | Recommended default — architecture | The nutrition, cost, scaling, and planning core is a pure deterministic module callable without a browser, database, network, or model. | Makes the critical behavior testable and reproducible, and makes framework or storage migrations less expensive (`ARCH-02`, `PLN-03`). |
| D-008 | 2026-09-18 | Recommended default — architecture | Use decimal or rational arithmetic internally for quantities and nutrients; serialize canonical decimals as strings; store money as integer minor units with a currency exponent. | Avoid unintended floating-point drift and preserve precision; UI formatting is a display concern only (`DOM-03`, `NUT-02`). |
| D-009 | 2026-09-18 | Recommended default — architecture | Stable logical identities and immutable content revisions are separate; plans and history pin the exact revision they use, with a current projection. | Editing a recipe must not rewrite past intake or past plans; corrections are new records with a link (`DOM-02`, `DOM-07` #5). |
| D-010 | 2026-09-18 | Recommended default — architecture | Required constraints (exclusions, viable equipment, explicit caps, locks, permitted portions) always outrank preference weights. | Required rules cannot be traded against cost, effort, or variety; a candidate that violates one is ineligible, not merely penalized (`PLN-02`, `DOM-07` #3, `ACT-020`). |
| D-011 | 2026-09-18 | Recommended default — architecture | Adapters are manual-first and optional; AI and providers sit behind interfaces and can be absent. The product is fully useful with manual entry and deterministic planning. | Avoids making every interaction depend on a model or network call, and keeps the manual fallback prominent (`AI-01`, `DATA-01`, `DEC-01`). |
| D-012 | 2026-09-18 | Recommended default — architecture | No automated checkout or order placement; purchasing is a shopping plan and export only. | A financial action requires a separately designed, authorized flow; an export or shopping list is not payment authority (`BIZ-02`, spec §3.2). |
| D-013 | 2026-09-18 | Recommended default — architecture | One private SQLite-backed workspace per authenticated user; all ownership is server-derived and validated per aggregate. | Supports later commercialization without building sharing or hosting prematurely, and keeps the local runtime simple (`SYS-03`, `DEC-01`). |
| D-014 | 2026-09-18 | Recommended default — architecture | Native portable formats are namespaced `daily.recipes` and `daily.workspace` starting at `schemaVersion: 2`, distinct from the prototype's unnamespaced version 1. | The format string is authoritative and a version number alone is not sufficient identification; a legacy adapter handles prototype files (`UX-DAT-05`, spec §20.8/§20.9). |
| D-015 | 2026-09-18 | Recommended default — architecture | Missing values are not zero; unknown, partial, and zero are distinct in every serializer, form, and total. | Prevents false nutrition precision — the scenario's signature failure — and keeps lower-bound passes visibly partial (`NUT-03`, `DOM-07` #1). |
| D-016 | 2026-09-18 | Recommended default — architecture | Mutations are idempotent and revision-checked; profile apply, plan apply, import/restore, purchase, preparation, and intake correction are all-or-nothing. | Cancellation and retry must never apply the same purchase, consumption, import, or replan twice (`SYS-02`, `DOM-07` #12). |
| D-017 | 2026-09-18 | Deferred | Household multi-profile optimization and public recipe sharing are out of the initial releases. | Multiplying serving counts is supported sooner, but treating several people as one nutritional profile, and public catalogs, need deliberate design (`OT-P2-003`, `OT-P2-004`, spec §3.2). |
| D-018 | 2026-09-18 | Accepted | Author the PRD, requirements registry, docs, and experience contract directly from the canonical product specification rather than through the `business-health` wizard interview. | The specification already contains the confirmed needs, operational outcomes, and contracts, so a deterministic interview would add a lossy transcription step. `business-health validate`, `vrooli scenario requirements validate`, and `experience-manager spec validate` all pass; the divergence and its reconciliation option are recorded in [`PROBLEMS.md`](PROBLEMS.md). |
| D-019 | 2026-09-18 | Accepted (tracked debt) | Treat the `docs` phase reference-integrity findings (`broken_command_snippet`, `broken_marked_ref`) as inherited template debt for this milestone, not as a product defect. | The findings sit in template scaffold files (`bas/README.md`, `docs/QUICKSTART.md`, and the template guides/reference docs), and the same `path:`-marker pattern appears in mature scenarios. None point at the authored planning docs; the documentation contract is validated directly with `knowledge-observatory docs audit`. Tracked in [`PROBLEMS.md`](PROBLEMS.md). |
| D-020 | 2026-09-18 | Accepted | All experience-contract claims stay `aspirational` until the UI exists, and the `dashboard`/`notes` pages are retained as `deprecated` tombstones. | Machine-tier claims would gate CI against a UI that has not been built. The tombstones exist only because scaffold BAS cases pin their spec ids, and they are retired by `template-manager detemplate` at Gate 7 (`experience/index.json`, `PROBLEMS.md`). |
| D-021 | 2026-09-18 | Confirmed | The first durable domain is `workspace`, with ownership derived from the verified request principal and idempotency recorded beside the aggregate. | The workspace is the authorization root for every later nutrition record; deriving owner identity server-side and recording retry keys at the persistence seam prevents client-supplied scope and duplicate bootstrap records. |
| D-022 | 2026-09-18 | Confirmed | R3 billing and household sharing remain deferred at explicit adapter seams; activating either requires a separately authorized product and security decision. | The current implementation provides server-enforced optional-compute entitlements, verified-event boundaries, private workspace scope, and portable food-domain export without billing identifiers. Payment-provider credentials, webhook endpoints, shared roles, attribution/redaction, and public distribution require operator authorization beyond building Daily. |
| D-023 | 2026-09-18 | Confirmed | Operational data health is reported as scoped actionable findings, not a single account score. | Failed optional jobs, stale prices, unknown targets, unresolved recipe data, and incomplete current plans have different remedies and should be prioritized near the current decision without frightening users with an opaque global grade. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced; link the new row so the lineage is visible. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`../concepts/DATA.md`](../concepts/DATA.md) — storage and revision ownership
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — deferred billing and packaging
- [`SECURITY.md`](SECURITY.md) — server-derived workspace scope
- [`../reference/product-specification.md`](../reference/product-specification.md) — sections 12, 14, 18, 19, 23, and 25.2
