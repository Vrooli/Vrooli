# Monetization — Nutrition Planner

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story.

The working product name is **Daily**. Scenario id is `nutrition-planner`.
This is a self-hosted, single-user personal nutrition and meal-planning
product, vegan-first but multi-diet by design. The originating reason to
build it is personal usefulness, not revenue; monetization is an option
the architecture keeps open, not a requirement the product is built on.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

Pricing, bundle membership, and whether to monetize at all are
**operator-curated canon**. This document states a hypothesis and the
evidence behind it; it does not set strategy. Read, do not write:
[project monetization canon](/docs/monetization/README.md) and the project catalog.

## Role In Vrooli

| Role | Status | Detail |
|---|---|---|
| Direct product (self-hosted personal app) | **candidate** | The product is the point: a deterministic planning core plus an optional-AI console that turns a boring or expensive default into affordable variety. It has an identifiable user outside Vrooli. Unvalidated. |
| Internal capability | **limited** | The deterministic, decimal-safe planning and nutrition core is reusable by any future scenario that has to plan under hard constraints with incomplete data. That reuse is a by-product, not the reason this scenario exists, and no current scenario depends on it. |
| SKU/bundle candidate | **candidate** | It plausibly belongs to a **lifestyle bundle** alongside health and habit products. Membership is *proposed, not committed*; Offer Desk owns the catalog, not this document. Being multi-diet and private makes it a reasonable bundle citizen, not a bundle obligation. |
| Revenue line | **deferred** | No billing, no paywall, no checkout is in scope before R3 (`OT-P2-001`, `OT-P2-002`). The product must be fully useful self-hosted and free of a paywall at R0–R2. |

The personal role does not depend on the product role being validated. If
the commercial hypothesis fails, the scenario is still correct and still
earns its keep for the originating user and any other self-hoster. This is
reaffirmed in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

**Self-hosted first.** The runtime is the user's own box: a Go API, a
built UI, and a SQLite file (see [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md)).
There is no per-user hosting cost and no vendor custody of meal, health, or
spending data. That posture is a deliberate product property (`SYS-03`,
`SYS-04`), not a temporary packaging choice.

## Customer / Buyer

- **Originating user:** a vegan who wants substantial protein with little
  preparation or cooking, whose carefully planned but repetitive routine
  becomes boring and drifts toward more expensive or less suitable
  alternatives (spec §2.1). The exact products, doses, targets, and prices
  are personalization data the product collects in-app (`DEC-02`), not
  facts this document may invent.
- **Buyer = user.** This is a self-serve, single-person purchase. There is
  no procurement, no seat expansion, and no administrator persona.
- **Broader audience:** anyone who wants to be told what to eat rather than
  research it — a person who will accept a concrete recommendation it can
  explain, and who values not doing nutrition bookkeeping.
- **Pain:** deciding what to eat (and buying for it) is recurring, boring,
  and easy to get wrong; existing tools either demand continuous research
  or hide invented precision behind a confident number.
- **Existing alternatives:** a notes app or spreadsheet, a recipe website
  plus manual grocery list, a macro-tracking app, meal-kit delivery, or
  ordering out. See [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Willingness to pay:** none validated. No price is set in this document.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone self-hosted app | candidate | The primary shape. Local runtime, no hosting cost, no external service required at R0. |
| Bundle component | candidate | Proposed for a **lifestyle bundle**, not committed. Multi-diet and private by design. Offer Desk owns the catalog. |
| Add-on | rejected | It is a base capability, not an extension of a parent SKU. Modelling it as an add-on would put a paywall in front of a standalone product. |
| Service/consulting assist | not-applicable | No service delivery is contemplated. |
| Future paid value | deferred hypothesis | Larger AI/research allowances, additional integrations, household features, and richer automation are the only candidate paid surfaces. Each is hypothesized, none is priced or committed. |

**What is never an upsell.** The core product is not gated:

- Required dietary restrictions, exclusions, and eligibility checks
  (`OT-P0-006`, `PLN-02`) are core behavior and are never behind payment.
- Nutrition checks, unknowns/partial handling, and provenance (`NUT-03`,
  `DOM-07`) are correctness properties, not a tier.
- Export and backup of owned data (`OT-P0-013`, `SYS-05`) is always
  available. A model outage or exhausted AI allowance must not block
  recipes, editing, existing plans, or exporting owned data (`BIZ-01`).
- The deterministic planning core is callable without a browser, database,
  network, or model. It cannot be made to require a paid dependency.

**Bounded provider cost.** Any future provider or AI usage is metered and
capped server-side per workspace (`JOB-01`: token/spend, runtime, retries,
items, and concurrency limits). Automation is never unbounded, and an
unconfigured provider is a supported state, not a broken product.

## Pricing Hypothesis

- **Model (hypothesis only):** a low flat subscription or a one-time
  purchase for a convenience tier. Not per-meal, not per-export, and never
  a function of grocery spend or the number or size of plans. Charging for
  the product working would create exactly the wrong incentive.
- **Cost drivers:** the local runtime has near-zero marginal cost. The only
  variable line is optional AI/provider usage (model tokens, provider
  requests) when R2 adapters are enabled, which is why those features must
  be optional, bounded, and non-load-bearing for core workflows (`16.1`,
  `19.3`).
- **Comparable products:** none captured yet. Do not treat a list price read
  from a marketing page as validation.
- **Explicitly rejected pricing:** usage-based pricing on data volume,
  plan count, or grocery spend; per-seat; and gating export, required
  rules, or nutrition correctness. See the "never an upsell" list above.
- **No number is set here.** A price written into a document becomes canon
  by accident. Validation owns producing a number.

## Validation Plan

- **Demand signal needed:** evidence that the originating user (and then
  other dieters) actually returns to the plan rather than reverting to the
  expensive default, and that the product reduces decision work. The
  sharpest available signal is behavioral: does a returning user get a
  usable plan without effort, accept and lightly adjust it, and keep using
  it past the first week.
- **What must be measured** (with consent, per [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md)):
  time-to-first-useful-plan, repeated information requests and manual edits
  per week, suggestion acceptance and voluntary swap reasons, planned
  versus recorded spend, and nutrient coverage/missing-data status.
- **Channel:** [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Success threshold:** deliberately unset. It is set at the project
  monetization review, not here.
- **Revisit trigger:** R1 is personally useful and has been used with real
  meals, products, targets, and prices through at least one real week that
  produced an accepted plan, a shopping list, and recorded intake; **and**
  R2 bounded automation has been evaluated for whether it saved work
  relative to its cost. Self-use before external claim.
- **Anti-signal:** requests are about a missing integration while the user
  does not actually run a plan. That is a signal for a different product.

**The self-referential risk, stated carefully.** The originating user is
also the builder. Two disciplines keep that honest:

1. **The first user is not the shape of the product.** Multi-diet support is
   a first-class requirement (`C03`), not a configuration afterthought, and
   the commercial default must not assign the founder's vegan diet to
   everyone (spec §3.3).
2. **Dogfooding proves the thing runs; it does not prove anyone would pay.**
   Keep those two claims separate everywhere downstream. A feature is not
   validated by our own use of it.

## Current Status

`hypothesis` / `deferred`. The personal role is real and is being built as
R1. The commercial role is unvalidated: no price, no bundle commitment, no
willingness-to-pay evidence. Billing is explicitly deferred to R3
(`OT-P2-001`, `OT-P2-002`) and the deterministic core, required rules,
nutrition checks, and data export are never monetized.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — operational targets `OT-P2-001`…`OT-P2-004`
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — audience, positioning, channels, and validation
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — self-hosted runtime and tiers
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for validation
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — why billing stays deferred
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — why data stays on the user's box
- Project-level monetization strategy: [project monetization canon](/docs/monetization/README.md).
