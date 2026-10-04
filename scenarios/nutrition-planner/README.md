# Nutrition Planner

## Current delivery contract

The owner-approved 2026-10-03 selected product, exact visuals and commissioning authority are integrated into the canonical specification, PRD, requirements and experience. Start with docs/reference/product-specification.md §Operative selection and docs/internal/goal/GOAL.md. The goal home is the current sole ledger; historical redesign files are retained. No implementation or validation claim was promoted by this documentation change.


> Working product name **Nooch** — decide what to eat, plan a realistic week, shop, use your kitchen, and cook, around your own nutrition goals, dietary rules, effort, variety, and budget. The whole day's food and supplements, not just dinner.

Nutrition Planner exists so a person can consistently eat in a way that fits
their nutritional goals and dietary needs with near-zero ongoing research or
decision-making. The ordinary interaction is choosing whether the next meal
sounds good and pressing **Start cooking** — not maintaining spreadsheets,
re-researching food, or filling in long forms.

The capability in one sentence: capture food, recipes, supplements, prices, and
stock once (allowing incomplete information), then produce an explainable plan
whose immediate action is obvious, shop and cook from it, and improve future
suggestions from optional lightweight feedback — never by inventing personal
targets, allergies, prices, or a savings claim.

The experience is a calm cookbook in a warm kitchen: five destinations (Today,
Week, Meals, Groceries, Kitchen), photorealistic food, an editorial serif, and
Light and Evening appearances composed separately for desktop and phone. The
approved concept mockups are in [`docs/reference/mockups/`](docs/reference/mockups/README.md).

> **Status (2026-10-03): selected design and finite commission approved; implementation unproven.** The v2.0 design and ND-007–025 selected amendments are documented. The code is a non-working prototype of the
> earlier direction: the 2026-10-03 read-only ListWorkspaces probe failed with
> `401 unauthenticated` and the inspected active database had 29 empty tables at that snapshot. The redesign goal
> starts by repairing that foundation. Read
> [`docs/internal/REDESIGN_PLAN.md`](docs/internal/REDESIGN_PLAN.md) §3 for that
> historical audit. Current delivery is governed by
> [GOAL](docs/internal/goal/GOAL.md), [QUEUE](docs/internal/goal/QUEUE.md),
> [ACCEPTANCE](docs/internal/goal/ACCEPTANCE.md) and the admitted epochs.
> Historical observations here are not current runtime verification.

## What You Get

- **A canonical specification.** [`docs/reference/product-specification.md`](docs/reference/product-specification.md)
  holds the v2.0 redesign (R01–R30), the retained domain specification
  with its calculation fixtures (Appendix A), and repository notes (Appendix C).
- **An approved visual target.** Thirty exact registered candidate0.3 families plus accepted Today Sunroom v02 for
  Today, Week, Meals, Explore, recipe detail, cooking, Groceries, and Kitchen, in
  Light and Evening, with a reading guide that resolves their inconsistencies.
- **A binding design language.** `DESIGN.md` (Warm Kitchen) defines tokens,
  typography, spacing, the appearance model, media treatments, and the component
  grammar.
- **A traceable contract.** `PRD.md` (P0/P1/P2 operational targets) and a
  `requirements/` registry of 29 modules and 161 requirements, every target
  linked; all statuses stay `planned` until tagged tests earn them.
- **An experience contract.** `experience/` defines the redesigned pages and
  journeys with states, elements, claims, and test-id bindings.
- **A native finite delivery target.** The build order, engineering bar, surface
  specifications, production-artwork plan, verification commands, and the
  convergence protocol retained as history; current admission, status, feedback
  and acceptance live in `docs/internal/goal/` under the existing orchestrator.
- **A deterministic core by design.** Business rules, nutrition arithmetic,
  cost, eligibility, and planning are pure domain logic callable without a
  browser, database, network, or model; image generation, calendar links, and
  assisted import are optional adapters with honest fallbacks.
- **Honest data semantics.** Unknown is never displayed as zero; required
  dietary rules cannot be traded against cost, effort, or variety; history uses
  immutable revisions unless explicitly corrected.

## Documentation Map

| Need | Start Here |
|---|---|
| Accepted delivery outcome and closure obligations | [GOAL](docs/internal/goal/GOAL.md), [ACCEPTANCE](docs/internal/goal/ACCEPTANCE.md) |
| Current progress, findings and owner feedback | [QUEUE](docs/internal/goal/QUEUE.md), [epochs](docs/internal/goal/epochs/), [FEEDBACK](docs/internal/goal/FEEDBACK.md) |
| Historical redesign evidence | [plan](docs/internal/REDESIGN_PLAN.md), [ledger](docs/internal/REDESIGN_LEDGER.md), [feedback](docs/internal/OPERATOR_FEEDBACK.md) |
| Canonical product/implementation spec | [`docs/reference/product-specification.md`](docs/reference/product-specification.md) |
| Visual target (concept mockups) | [`docs/reference/mockups/README.md`](docs/reference/mockups/README.md) |
| Design language (binding tokens) | `DESIGN.md` at this scenario's root |
| Product charter and targets | [`PRD.md`](PRD.md) |
| Requirements registry | [`requirements/README.md`](requirements/README.md) |
| UX contract (pages and journeys) | [`experience/README.md`](experience/README.md), [`docs/concepts/EXPERIENCE.md`](docs/concepts/EXPERIENCE.md) |
| UI structure | [`docs/concepts/UI-ARCHITECTURE.md`](docs/concepts/UI-ARCHITECTURE.md) |
| Architecture and boundaries | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Domain map | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Workflows and state machines | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md) |
| Data model, storage, portability | [`docs/concepts/DATA.md`](docs/concepts/DATA.md) |
| Dependencies and adapters (image-tools, personal-planner, …) | [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| Monetization and launch strategy | [`docs/business/MONETIZATION.md`](docs/business/MONETIZATION.md), [`docs/business/GO-TO-MARKET.md`](docs/business/GO-TO-MARKET.md) |
| Deployment and operations | [`docs/operations/DEPLOYMENT.md`](docs/operations/DEPLOYMENT.md), [`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md), [`docs/operations/OBSERVABILITY.md`](docs/operations/OBSERVABILITY.md) |
| Security, performance, testing | [`docs/internal/SECURITY.md`](docs/internal/SECURITY.md), [`docs/internal/PERFORMANCE.md`](docs/internal/PERFORMANCE.md), [`docs/internal/TESTING.md`](docs/internal/TESTING.md) |
| Decisions, progress, known problems | [`docs/internal/DECISIONS.md`](docs/internal/DECISIONS.md), [`docs/internal/PROGRESS.md`](docs/internal/PROGRESS.md), [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md) |
| Run the scenario | [`docs/QUICKSTART.md`](docs/QUICKSTART.md) |

## Running The Scenario

```bash
# Build API + UI, install pnpm deps, install scenario CLI
make setup   # wraps `vrooli scenario setup`

# Start API + UI in the background
make start   # wraps `vrooli scenario start`

# Machine-readable initialization gates
make orient
```

Run tests with `make test` (which wraps `vrooli scenario test`), or use
`vrooli scenario test nutrition-planner --phases <relevant-phases>` for scoped
validation. Requirements are validated with
`vrooli scenario requirements validate nutrition-planner --json`.

Until the redesign's first milestone lands, the retained first-use probe returned `401 unauthenticated` because the scenario
declares no authentication profile (blocker B1 in the redesign plan).

## Customize Safely

Keep these seams:

- **Connect-RPC is the default transport.** Every domain endpoint goes through a
  proto service; author proto first, prefer typed messages over opaque JSON
  strings, and never write literal REST paths except the sanctioned
  `RESTException` cases.
- **API owns business rules.** The CLI and UI are translation layers; do not put
  a second planner, cost engine, or eligibility check in either.
- **Pure domain core.** Nutrition, cost, eligibility, and planning calculations
  stay callable without a browser, database, network, or model.
- **Unknown is not zero.** Keep unknown, zero, partial, and complete distinct in
  every serializer, form, total, and claim.
- **Required rules are hard constraints.** Exclusions, equipment, and explicit
  caps can never be optimized away by preferences.
- **Immutable revisions and real persistence.** Recipe edits create revisions;
  history and pinned occurrences are never silently rewritten; schema changes go
  through versioned migrations; multi-record operations are transactional.
- **One design system.** Tokens from `DESIGN.md` only, shared components built
  once, per-medium composition, i18n for every string, and the
  `react-component-library` where it fits.
- **Artwork is data.** Images are manifest-tracked assets with provenance and
  rights; generation goes through image-tools and ai-gateway under an explicit
  policy, never per view.
- **Dependencies flow through Scenario Dependency Analyzer** — never raw package
  managers or hand-edited governance JSON.

## pnpm Everywhere

This scenario assumes pnpm. Scripts use `pnpm` directly (no `npm` fallbacks) to
reduce drift.
