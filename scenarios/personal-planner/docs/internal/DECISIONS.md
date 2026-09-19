# Decisions — Personal Planner

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

The product decisions D01–D09 are the operator-accepted decisions from the
"Adaptive Personal Planner — Complete Vrooli Implementation Plan" (§1.3);
they are recorded here so they are not relitigated during implementation.
The scenario decisions below them are Vrooli-shape choices made while
generating this scenario.

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-09-18 | D01 — Suggestions and previews are applied by the user, not automatically. | The product must never silently rewrite a plan or promise. | Proposals return candidate operations that write nothing; apply is a separate explicit, revalidated command; what-if has no side effects. | Opt-in automatic flexible rescheduling within declared bounds (R3, OT-P2-004). |
| 2026-09-18 | D02 — Hybrid planning model. | Fixed events, flexible timed sessions, date-assigned effort, and an unscheduled backlog must coexist without double-counting. | Allocations conserve demand on split/move (INV-07); a child allocation shares the parent's quantity rather than cloning it. | Only if the model proves insufficient for a real journey. |
| 2026-09-18 | D03 — Individual private workspaces (single-user R1). | Household/team collaboration is deferred to keep isolation simple and correct. | Every read/write/job/projection/cache is workspace-scoped (INV-01); no multi-person aggregation in R1. | Household/team collaboration expansion (R3, OT-P2-005). |
| 2026-09-18 | D04 — Optional read-only external calendars. | External obligations should inform capacity without the product owning them. | Provider access is read-only at scope and adapter; no writeback path in R1; feature degrades cleanly when absent. | Two-way provider writeback (R3, OT-P2-006). |
| 2026-09-18 | D05 — Optional timers and reviews. | Tracking friction must not drive abandonment. | Manual planning is fully usable without ever starting a timer or completing a review; both are optional and skippable. | Only if evidence shows the optional path harms the core loop. |
| 2026-09-18 | D06 — Selective read-only sharing. | Owners share chosen commitment fields, not their whole plan. | Shares are server-side field-masked DTOs to a recipient-bound viewer, revocable, read-only, and non-leaking (INV-12). | Richer collaboration/editing (R3). |
| 2026-09-18 | D07 — One shared scheduling owner. | Multiple apps (Daily, Cadence, agent work) need availability/scheduling without duplicating a scheduler. | Personal Planner is the single scheduling authority; consumers submit normalized intent and read accepted schedules through contracts; a shared DB is never the integration API. | Only if a second authoritative scheduler is ever justified (strongly discouraged). |
| 2026-09-18 | D08 — Observatory day/night UX. | Beauty and calm are explicit product requirements, not decoration; the coordinated landscape is the retention wedge. | Coordinated day/night scenes, Auto/Day/Night control, truthful time geometry, and WCAG 2.2 AA as a visual release gate; the scene is decorative and never blocks content. | Only on an approved successor visual direction. |
| 2026-09-18 | D09 — Local Vrooli implementation, deterministic no-AI core. | The core planning loop must work with no model configured. | Scheduling/capacity/forecast logic is a pure deterministic Go kernel; any assistant (R2) is optional, treated as data, and never a prerequisite. | Optional conversational assistant / calibrated statistical forecasts (R2). |
| 2026-09-18 | Scenario name `personal-planner`; "Planner"/"Observatory" are working labels, not a permanent brand. | A permanent product name/logo/slogan has not been chosen. | Working labels are replaceable configuration; branding hooks stay valid but generic until a name is selected through brand-manager. | When a permanent product name is chosen. |
| 2026-09-18 | Default `react-vite` template + `vrooli-default` design kit, intentionally re-themed to Observatory. | Observatory is a deliberate departure from the operational-console defaults ("avoid atmospheric backgrounds / compact-operational-only"). | Built on the Vrooli token plumbing and component-library shell, but the design language is Observatory; a stock console dashboard is **not** an acceptable substitute. Approved by the operator (D08). | On an approved successor visual direction, or a template change. |
| 2026-09-18 | SQLite is the default storage substrate. | Start scenario-owned and in-process; no external resource process for R1. | Per-domain schema co-located via SchemaProvider/EnsureSchemas; a relational resource (e.g. Postgres) is adopted only when measured scale or the partial-unique one-session guard justifies it, through the Scenario Dependency Analyzer. | Measured scale or the one-exclusive-session concurrency guard at scale. |
| 2026-09-18 | Deterministic no-AI core (scenario-level restatement of D09). | The product must not depend on a model for correctness. | The scheduling/capacity/forecast kernel is pure and side-effect-free; a missing model never breaks the core workflow. | R2 optional assistant. |
| 2026-09-18 | The unused legacy `calendar` scenario was removed before generation. | It was undeveloped and held no live scheduling data. | Nothing to migrate at R0; if legacy/external data appears later, follow plan §3.3 (stable ID mapping, dry-run counts, provenance, rollback) rather than treating old events as actuals. | If real legacy or external scheduling data appears. |
| 2026-09-18 | scenario-authenticator required; notification-hub optional. | Identity and private-workspace isolation are non-negotiable; notification delivery is not. | Scenario refuses to start without an identity provider; without notification-hub, notices stay in-app and the product remains fully usable. | Only if the dependency model changes. |
| 2026-09-19 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |

## Superseded Decisions

No durable decision has been replaced yet. The closest related event is
the retirement of the legacy `calendar` scenario before generation
(recorded in the Decision Log above and in
[`../concepts/DATA.md`](../concepts/DATA.md) "Migrations And
Compatibility"); it superseded no decision made in *this* scenario, since
there were none.

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision recorded above is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`../concepts/DATA.md`](../concepts/DATA.md) — storage substrate and migration decisions
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
