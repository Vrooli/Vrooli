# Nutrition Planner

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

> **Status (2026-09-22): redesign specified, not implemented.** The v2.0
> redesign is fully documented. The code is a non-working prototype of the
> earlier direction: in the running scenario every request fails with
> `401 unauthenticated` and the database has never held a row. The redesign goal
> starts by repairing that foundation. Read
> [`docs/internal/REDESIGN_PLAN.md`](docs/internal/REDESIGN_PLAN.md) §3 for the
> audited state and [`docs/internal/REDESIGN_GOAL.md`](docs/internal/REDESIGN_GOAL.md)
> for the goal that drives the work.

## What You Get

- **A canonical specification.** [`docs/reference/product-specification.md`](docs/reference/product-specification.md)
  holds the v2.0 redesign (R01–R30), the retained domain specification
  with its calculation fixtures (Appendix A), and repository notes (Appendix C).
- **An approved visual target.** Fifteen desktop-and-phone concept mockups for
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
- **An executable redesign plan.** The build order, engineering bar, surface
  specifications, production-artwork plan, verification commands, and the
  convergence protocol, plus the ledger and operator-feedback file the goal runs
  on.
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
| What is being built, and the goal that builds it | [`docs/internal/REDESIGN_PLAN.md`](docs/internal/REDESIGN_PLAN.md), [`docs/internal/REDESIGN_GOAL.md`](docs/internal/REDESIGN_GOAL.md) |
| Progress, findings, and operator feedback for the redesign | [`docs/internal/REDESIGN_LEDGER.md`](docs/internal/REDESIGN_LEDGER.md), [`docs/internal/OPERATOR_FEEDBACK.md`](docs/internal/OPERATOR_FEEDBACK.md) |
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

Until the redesign's first milestone lands, the running app authenticates no
one: every workspace request returns `401 unauthenticated` because the scenario
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
