# Decisions — Offer Desk

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
| 2026-08-13 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-13 | A **delivery tier is a `variant` node**, not a sixth node kind. | The source canon holds tiers in `strategy/TIERS.md` with their own four-state lifecycle (`active` / `candidate` / `north-star` / `retired`), declared *orthogonal* to the SKU catalog. The node kinds here are `offer`, `variant`, `channel`, `revenue-line`, `deliverable`; `tier` appears in none of them. | An offer is *sold at* a variant, so "Business Bundle at Tier 2" is one offer-to-variant edge and pricing is a property of that edge — which is exactly the orthogonality the source document asserts. Tiers therefore inherit the one shared status vocabulary instead of keeping a private one. The `north-star` state does not survive the move: it maps to `idea`, whose semantics ("captured, not planned against") are identical. | A second axis needs the same orthogonal treatment (e.g. geography or currency zone) and edge properties stop being expressive enough. |
| 2026-08-13 | An **add-on is an `offer` with a `requires` edge to its parent**, not a distinct node kind. | `catalogs/CATALOG.md` splits SKUs into base bundles and add-ons, and asserts membership must be many-to-many. | Guardrail 3 ("no add-on activation before parent bundle has paying users") becomes a transition precondition on the `requires` edge rather than prose — an add-on promotion is refused while its parent has no paying-user fact. Many-to-many survives because an offer may carry several `requires` edges. | An add-on needs pricing or entitlement semantics that differ structurally from an offer, which would mean commerce concerns have leaked in and belong upstream. |
| 2026-08-13 | **No feature candidates are recorded for this scenario**, and no standalone competitive positioning is pursued. | A 2026-08-13 competitive scan found no product joining a lifecycle-enforced catalog to a ledger of actuals — which strengthens the Money Ledger pairing but does not establish standalone demand. An unoccupied space beside mature adjacent markets is more often unwanted than unnoticed. | The roadmap is whatever makes Money Ledger's surfaces better, in the order the PRD sequences. Absence of a competitor is explicitly not treated as demand evidence; the revisit trigger still requires operators to describe the pain unprompted. | The `../business/MONETIZATION.md` revisit trigger fires — all three parts. |
| 2026-08-13 | **The funnel is not modelled here.** | `strategy/FUNNEL.md` describes stage conversion metrics. | Funnel stages are measurements over channels and revenue lines, not records with a lifecycle. Modelling them would put a metrics pipeline inside a catalog. Channels remain nodes; their conversion numbers stay in the ledger and telemetry. | Channel attribution telemetry ships and a funnel stage acquires a state an operator must disposition. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
