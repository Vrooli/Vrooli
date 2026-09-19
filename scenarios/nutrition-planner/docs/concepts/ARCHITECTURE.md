# Architecture — Nutrition Planner

Nutrition Planner (product name **Daily**) is a personal nutrition and meal-planning
application. It plans varied, affordable, nutritionally adequate meals around a person's
diet, restrictions, kitchen, preferences, budget, and willingness to cook; it represents
the whole day's food and supplements; it explains every number it shows; and it treats
unknown, zero, and partial data as three different facts.

This document is the scenario's system map. It explains the invariant shape inherited
from the `react-vite` template, then points to the specialized documents that own product
domains, workflows, data, integrations, deployment, operations, and commercial strategy.
Keep it high-signal. If a concern has a dedicated document below, update that document
and link it here.

## Purpose Of This Document

This document owns:

- the scenario's system shape and the role of each surface,
- how contracts and data flow between surfaces,
- the shared infrastructure boundary and the module boundaries,
- state ownership and the invalidation rules for derived data,
- extension rules for future code and the generated-file rule,
- architecture maturity, intentional deviations, and documentation architecture.

This document does not own:

- product capability inventory: [`DOMAINS.md`](DOMAINS.md),
- temporal and user/system workflows: [`FLOWS.md`](FLOWS.md),
- storage ownership, retention, and migrations: [`DATA.md`](DATA.md),
- resource, scenario, and third-party dependencies: [`INTEGRATIONS.md`](INTEGRATIONS.md),
- test seams and fakes: [`../internal/SEAMS.md`](../internal/SEAMS.md),
- test strategy: [`../internal/TESTING.md`](../internal/TESTING.md),
- deployment and operations: [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md),
- commercial strategy: [`../business/MONETIZATION.md`](../business/MONETIZATION.md).

## Scenario Shape

A scenario is one product expressed through three coordinated surfaces and one canonical
contract layer.

```
                        ┌──────────────────────────────────┐
                        │  Generated proto types            │
                        │  packages/proto/schemas/          │
                        │  nutrition-planner/v1/...          │
                        └───────────────┬──────────────────┘
                                        │ canonical wire shape
              ┌─────────────────────────┼─────────────────────────┐
              │                         │                         │
              ▼                         ▼                         ▼
        ┌──────────┐             ┌──────────┐             ┌──────────┐
        │   ui/    │ Connect-JSON│  api/    │ Connect-JSON│  cli/    │
        │ React    │ ◀─────────▶ │   Go     │ ◀─────────▶ │   Go     │
        │ + Vite   │             │ HTTP     │             │ thin CLI │
        └────┬─────┘             └────┬─────┘             └────┬─────┘
             │                        │                        │
             │                        ▼                        │
             │                 ┌────────────────┐              │
             │                 │ pure domain    │              │
             │                 │ core (Go)      │              │
             │                 │ no browser/db/ │              │
             │                 │ net/model      │              │
             │                 └───────┬────────┘              │
             │                         │                        │
             ▼                         ▼                        ▼
        browser state            ┌──────────┐             report-shaped
                                 │ SQLite   │             stdout
                                 │ (local)  │
                                 └──────────┘
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Authorization, transactions, revisions, business rules, persistence, integrations, transport edge | Browser state, CLI formatting |
| Domain core (`api/internal/<domain>/…` pure packages) | Deterministic nutrition, planning, cost, and eligibility calculation | Pure functions and policy objects only | Browser, database, network, model, or clock access |
| UI (`ui/`) | Browser presentation | Interaction state, presentation, accessible editing, i18n | A second planner, cost engine, or authorization decision |
| CLI (`cli/`) | Operator/agent wrapper | Argument parsing, output formatting, API invocation | Business rules, duplicated validation, a parallel planner |
| Contracts (`packages/proto/schemas/nutrition-planner/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principles:

1. **The API is the only surface that contains mutable business logic.** UI and CLI
   translate intent into API calls; neither re-derives nutrition, cost, eligibility, or
   planning.
2. **The deterministic domain core is callable without a browser, database, network, or
   model.** It receives snapshots and policies and returns values, provenance, and
   structured reasons. This makes the critical behavior fast to test and cheap to migrate.
3. **Proto types flow from one source of truth**, so wire-shape drift between surfaces is
   impossible.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas under `packages/proto/schemas/nutrition-planner/`.

The scenario does not own:

- shared package implementation under `packages/`,
- Vrooli resource implementation,
- scenario dependencies it calls,
- generated proto outputs under `packages/proto/gen/`.

Document dependency and resource decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md), not here.

### Module Boundaries

Specification §17.2 defines the module boundaries independent of folder names. They map to
the bounded contexts in [`DOMAINS.md`](DOMAINS.md); the table below states what each
module owns and, more importantly, what it must never own.

| Module | Owns | Must Not Own |
|---|---|---|
| Profile and rules | Diet, restrictions, kitchen capabilities, targets, preferences. | Billing truth or recipe parsing. |
| Food catalog | Food/product identities, revisions, nutrient evidence, unit mappings. | Unreviewed model text as canonical truth. |
| Recipe library | Drafts, immutable revisions, method graphs, components/families. | Current inventory balance. |
| Nutrition engine | Unit-aware totals, coverage, target evaluation. | Network fetches or UI state. |
| Planning engine | Candidate eligibility, search, plan assessment, explanations. | Direct persistence side effects during search. |
| Inventory and costs | Stock events, packages, observations, purchase/batch projections. | Assuming planned actions actually happened. |
| Intake and feedback | Consumption, corrections, explicit feedback. | Mutating historical recipe revisions. |
| Transfer and documents | Versioned import/export, migration, printable layout. | Bypassing domain validation on import. |
| Provider/job adapters | External requests, structured proposals, bounded execution. | Deciding permissions from model output. |
| Application services | Authorization, transactions, revisions, operation orchestration. | Duplicated nutrient formulas in handlers. |
| UI | Interaction state, presentation, accessible editing. | A second incompatible planner or cost engine. |

A module that starts using another module's vocabulary has crossed a boundary; split the
shared word into an owning domain instead of growing a generic bucket.

## Contracts And Data Flow

Wire shapes do not live in TypeScript interfaces, Go structs, or hand-written JSON
schemas. They live in `.proto` files. For proto-typed API calls, the `.proto` file also
declares the service block that generates Connect handlers and clients.

```
packages/proto/schemas/nutrition-planner/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/nutrition-planner/v1/...            (api, cli)
       ├──▶ packages/proto/gen/go/nutrition-planner/v1/...connect     (Connect-Go)
       ├──▶ packages/proto/gen/typescript/nutrition-planner/v1/...    (ui)
       └──▶ packages/proto/gen/python/nutrition_planner/v1/...        (future tools)
```

**Proto is authored first.** A new capability starts as proto messages and a service
method, then flows outward to API, CLI, and UI. Use Connect-RPC by default for UI→API and
CLI→API proto-typed payloads, and for inter-scenario calls with Vrooli-owned protos.

REST is allowed only for four enumerated reasons, defined as `RESTReason` constants in
`api/internal/module/module.go`:

| Reason | When it applies |
|---|---|
| `RESTReasonMultipartUpload` | Opaque file bytes via `multipart/form-data`. The sanctioned exception for this scenario is bounded source-artifact upload (an image, label photo, or import file). |
| `RESTReasonWebhookReceiver` | Endpoint shape is dictated by a third-party system we do not own. Deferred: no P0–R2 integration receives webhooks. |
| `RESTReasonThirdPartyShape` | Request or response is an externally-defined contract. Deferred until a named provider requires it. |
| `RESTReasonOpsProbe` | Lifecycle systems, load balancers, and `curl` must reach the endpoint without a generated client (plain `GET /health`, static assets). |

Mechanical enforcement: `cmd/gen-endpoints` rejects any `EndpointDescriptor.Path` that is
not a generated Connect procedure constant (i.e. does not start with `/vrooli.`) unless the
descriptor carries a `RESTException` with one of the four reasons. A REST endpoint without
that tag fails `make endpoints`, which fails `make test`, which fails CI. The fix is either
to author a proto service method (the preferred path) or to tag the exception explicitly.
Even for a REST exception, the **payload shape** stays proto-typed wherever possible.

### Generated-File Rule

Everything under a `generated/` directory is codegen output. Regenerate it with the owning
tool (`make generate`, `make temporal-models`, or the domain's flow command); **never
hand-edit** a generated file. Generated artifacts are checked in for reviewability, and
`flow-verifier verify check` byte-compares every generated file, so a manual edit fails the
build rather than silently drifting.

### State Ownership And Invalidation

Specification §17.3 separates four kinds of state:

| State | Owner | Rule |
|---|---|---|
| Server facts | API + SQLite | The authoritative record of profiles, recipes, plans, stock, and intake. |
| Derived assessments | Nutrition/planning/cost engines | Keyed by the relevant input revisions and the evaluator version; recomputed rather than mutated. |
| Editor drafts | UI | Dirty fields and a base revision. Blank numeric input stays blank/null and never coerces to zero. |
| Transient interface state | UI | Open tabs, panel sizes, filters. Opening a tab does not mutate a recipe. |

The server is authoritative for domain operations. The UI may optimistically present simple
reversible changes but must reconcile the returned revision or surface failure. An explicit
dependency map governs derived data: ingredient composition affects nutrition and
eligibility; method changes affect equipment and effort; price changes affect cost; stock
changes affect shopping; target changes affect assessments; plan/portion changes affect all
plan summaries. Do not rely on arbitrary page reloads for correctness.

## Shared Infrastructure

Shared infrastructure is allowed only when the code is business-vocabulary-free and used by
unrelated domains or surfaces.

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| `api/internal/server/` | Compose modules and middleware into one HTTP server. | Server lifecycle is not a product capability. | API entrypoint and handler modules. |
| `api/internal/module/` | Shared module and endpoint descriptor types. | Domain modules return this common shape. | Handler packages, server, endpoint codegen. |
| `api/internal/modules/` | Thin registry for schemas and endpoints. | Boot/codegen need central lists; logic stays domain-owned. | `main.go`, `gen-endpoints`. |
| `api/internal/database/` | System schema and DB reachability seam. | Cross-cutting DB infrastructure, not one domain's data. | API boot, health. |
| `api/internal/clock/` | Deterministic time seam (UTC instants, IANA timezones). | Time is cross-cutting and test-substitutable. | Middleware, repositories, jobs. |
| `api/internal/testutil/` | Cross-domain test harnesses and fakes. | Used by unrelated domains; domain fakes stay domain-local. | API tests. |
| `ui/src/components/` | Shared presentation primitives. | Used by unrelated UI features. | UI features and shell. |
| `ui/src/test-utils/` | Cross-feature render helpers, a11y helpers, and model tests. | Used by unrelated UI features. | UI tests. |

If shared infrastructure starts using product vocabulary, move that piece back into the
owning domain or split a new domain first.

## Extension Rules

Add product behavior by adding or updating the owning domain, not by growing generic
buckets.

For a normal proto-backed domain:

1. Add proto messages and service methods under
   `packages/proto/schemas/nutrition-planner/v1/<domain>/`.
2. Add pure domain code under `api/internal/<domain>/`; keep it free of database, network,
   clock, and model access.
3. Add transport code under `api/handlers/<domain>/`.
4. Register schemas/endpoints in `api/internal/modules/registry.go` and mount the module in
   `api/main.go`.
5. Add CLI commands under `cli/domains/<domain>/`; keep them a thin translation layer.
6. Add UI API wrappers under `ui/src/api/<domain>.ts` and UI feature code under
   `ui/src/features/<domain>/`.
7. Update selectors, strings, endpoints, tests, and the documentation contract in
   `docs/manifest.json`.

For detailed product ownership, update [`DOMAINS.md`](DOMAINS.md). For persistence and
retention, update [`DATA.md`](DATA.md). For temporal behavior, update [`FLOWS.md`](FLOWS.md).

## Architecture Maturity

The scenario currently carries the mature template shape plus product planning documents;
the product vertical slice has not been implemented. This table must be updated as M0–M4
land (specification §22.2).

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Scaffold | Domain-owned vertical-slice stack, module registry, per-domain schema, documented seams. | The `notes` worked example must be removed by detemplate; real domains from [`DOMAINS.md`](DOMAINS.md) are not implemented. |
| Domain core | Planned | Specification §12–§15 define the contracts, arithmetic, and invariants; module boundaries defined in §17.2. | No pure calculation package exists yet; FIX-02…FIX-07 are not executable. |
| UI | Scaffold | Feature folders, typed API clients, selector/i18n registries, modeltest helpers. | The generated dashboard and notes feature are placeholders; Today is not built. See [`EXPERIENCE.md`](EXPERIENCE.md). |
| CLI | Scaffold | Domain command groups wrap API calls and render reports. | No nutrition/planning commands exist. |
| Docs | In progress | Manifest v2 registers docs, maturity, stages, and validation hints; these concept documents are being authored. | Requirements, experience contracts, and operations docs must be filled to match. |

Use `docs/manifest.json` as the documentation contract. Declared `maturity` values are
maintained by agents and later grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards when they are
deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-09-18 | None from Vrooli platform standards. | The scenario uses the standard three-surface shape and Connect-RPC transport. | Update when a platform rule must be bent for a documented product constraint. |
| 2026-09-18 | The scaffold `notes` domain and dashboard placeholders are not product scope. | They are the template's worked vertical slice, kept only until real domains land. | `template-manager detemplate` removes them once the first real domain is green. |

## Documentation Architecture

Scenario docs follow the same ownership rule as code: one durable question, one canonical
home.

| Concern | Canonical Document |
|---|---|
| System map, boundaries, extension rules | `docs/concepts/ARCHITECTURE.md` |
| Product capabilities and bounded contexts | `docs/concepts/DOMAINS.md` |
| Workflows and state transitions | `docs/concepts/FLOWS.md` |
| Data ownership, retention, migrations | `docs/concepts/DATA.md` |
| Resources, scenarios, external services | `docs/concepts/INTEGRATIONS.md` |
| UI decision and primary surface | `docs/concepts/EXPERIENCE.md` |
| Source-tree layout and slot taxonomy | `docs/concepts/UI-ARCHITECTURE.md` |
| Monetization and packaging | `docs/business/MONETIZATION.md` |
| Deployment tiers and readiness | `docs/operations/DEPLOYMENT.md` |
| Telemetry, metrics, and alerts | `docs/operations/OBSERVABILITY.md` |
| Seams and test doubles | `docs/internal/SEAMS.md` |
| Testing strategy | `docs/internal/TESTING.md` |
| Known drift and deferred work | `docs/internal/PROBLEMS.md` |
| Change history | `docs/internal/PROGRESS.md` |

Every durable scenario document should be registered in `docs/manifest.json`. Put deep
domain-specific documentation under `docs/domains/<domain>/` when `DOMAINS.md` would become
noisy.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — bounded contexts and ownership
- [`FLOWS.md`](FLOWS.md) — workflow and state-transition map
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`EXPERIENCE.md`](EXPERIENCE.md) — the UI decision
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — seam registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
- [`../reference/product-specification.md`](../reference/product-specification.md) — canonical product specification
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — commercial story
