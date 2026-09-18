# Nutrition Planner

> Working product name **Daily** — plan varied, affordable, nutritionally adequate meals around your diet, restrictions, kitchen, preferences, budget, and willingness to cook. The whole day's food and supplements, not just dinner.

Nutrition Planner exists so a person can consistently eat in a way that fits their
nutritional goals and dietary needs with near-zero ongoing research or
decision-making. The ordinary interaction is choosing whether tonight's
suggestion sounds good — not maintaining spreadsheets, re-researching food, or
filling in long forms.

The capability in one sentence: capture food, recipes, supplements, prices, and
stock once (allowing incomplete information), then produce an explainable plan
whose immediate action is obvious, and improve future suggestions from optional
lightweight feedback — never by inventing personal targets, allergies, prices,
or a savings claim.

This scenario was generated from the `react-vite` template and packages the
standard full-stack Vrooli scenario shape (Go API over Connect-RPC, typed Go CLI,
React + TypeScript + Vite UI, SQLite storage, lifecycle wiring, requirements
registry, experience contract, and documentation contract).

> **Planning status: documented, not implemented.** `PRD.md`, `requirements/`,
> `docs/`, and `experience/` describe the intended product in full. The code is
> still the generated scaffold plus the removable `notes` example. Read
> [`docs/reference/product-specification.md`](docs/reference/product-specification.md)
> for the authoritative product and implementation specification, then
> [`docs/START-HERE.md`](docs/START-HERE.md) for the initialization protocol.

## What You Get

- **PRD and requirements registry.** A release-scoped product charter
  (`PRD.md`, P0/P1/P2 operational targets) and a scenario-specific
  `requirements/` registry: 19 modules, 84 requirements, every operational
  target linked, validating clean under `business-health`.
- **Domain, data, flow, and integration concepts.** `docs/concepts/` maps the
  bounded contexts, the DOM-06 entity inventory, the seven user journeys and
  lifecycle state machines, and the (deliberately empty) P0 dependency set.
- **A multi-page experience contract.** `experience/` defines nine product
  pages and seven journeys with priorities, elements, states, and
  intent-level claims — the design-first sibling of `requirements/`.
- **Business, operations, and internal docs.** Monetization and go-to-market
  hypotheses; deployment, runbook, and observability procedures; security,
  performance, decision log, progress, and problems registers.
- **A deterministic planning core by design.** Business rules, nutrition
  arithmetic, cost, and planning are pure domain logic callable without a
  browser, database, network, or model; AI and provider adapters are optional
  and never load-bearing.
- **Honest data semantics.** Unknown is never displayed as zero; required
  dietary rules cannot be traded against cost, effort, or variety; historical
  intake uses immutable revisions unless explicitly corrected.

## Documentation Map

| Need | Start Here |
|---|---|
| Canonical product/implementation spec | [`docs/reference/product-specification.md`](docs/reference/product-specification.md) |
| Initialize after generation | [`docs/START-HERE.md`](docs/START-HERE.md) |
| Product charter and targets | [`PRD.md`](PRD.md) |
| Requirements registry | [`requirements/README.md`](requirements/README.md) |
| UX contract (pages and journeys) | [`experience/README.md`](experience/README.md), [`docs/concepts/EXPERIENCE.md`](docs/concepts/EXPERIENCE.md) |
| Design language (binding tokens) | `DESIGN.md` at this scenario's root |
| Architecture and boundaries | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) |
| Domain map | [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Workflows and state machines | [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md) |
| Data model, storage, portability | [`docs/concepts/DATA.md`](docs/concepts/DATA.md) |
| Dependencies and provider adapters | [`docs/concepts/INTEGRATIONS.md`](docs/concepts/INTEGRATIONS.md) |
| Monetization and launch strategy | [`docs/business/MONETIZATION.md`](docs/business/MONETIZATION.md), [`docs/business/GO-TO-MARKET.md`](docs/business/GO-TO-MARKET.md) |
| Deployment and operations | [`docs/operations/DEPLOYMENT.md`](docs/operations/DEPLOYMENT.md), [`docs/operations/RUNBOOK.md`](docs/operations/RUNBOOK.md), [`docs/operations/OBSERVABILITY.md`](docs/operations/OBSERVABILITY.md) |
| Security and performance | [`docs/internal/SECURITY.md`](docs/internal/SECURITY.md), [`docs/internal/PERFORMANCE.md`](docs/internal/PERFORMANCE.md) |
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

## Customize Safely

The generated scaffold is intentionally not the product. Keep these seams:

- **Connect-RPC is the default transport.** Every domain endpoint goes through a
  proto service; author proto first and never write literal REST paths except
  the sanctioned `RESTException` cases.
- **API owns business rules.** The CLI and UI are translation layers; do not put
  a second planner or cost engine in either.
- **Pure domain core.** Nutrition, cost, and planning calculations must stay
  callable without a browser, database, network, or model.
- **Unknown is not zero.** Keep unknown, zero, partial, and complete distinct in
  every serializer, form, total, and claim.
- **Required rules are hard constraints.** Exclusions, equipment, and explicit
  caps can never be optimized away by preferences.
- **Immutable revisions.** Recipe edits create revisions; history and pinned
  occurrences are never silently rewritten.
- **i18n, accessibility, design tokens, and the feature-folder pattern** under
  `ui/src/features/<name>/` are durable seams. Prefer the library shell and
  adopt `react-component-library` components over hand-rolling.
- **Dependencies flow through Scenario Dependency Analyzer** — never raw package
  managers or hand-edited governance JSON.

When you build the first real product domain beside the `notes` example and
prove it green, remove the example with
`template-manager detemplate nutrition-planner`.

## pnpm Everywhere

This scenario assumes pnpm. Scripts use `pnpm` directly (no `npm` fallbacks) to
reduce drift.
