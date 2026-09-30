# Architecture — Nutrition Planner

Nutrition Planner (display name **Nooch**) is a personal food-planning application: decide
what to eat, plan a realistic week, shop, use the kitchen, and cook, while balancing the
person's own nutrition goals, dietary rules, effort, variety, and cost across the whole
day's food and supplements. It explains every number it shows and treats unknown, zero,
and partial data as three different facts.

This document is the scenario's system map for the v2.0 redesign. It states the
**intended** shape (specification R21–R25 and Appendix A §12, §17–§18) and, separately and
explicitly, **what exists today** (the 2026-09-22 audit). If a concern has a dedicated
document below, update that document and link it here.

## Purpose Of This Document

This document owns:

- the scenario's system shape and the role of each surface,
- module boundaries, including the redesign modules,
- how contracts and data flow between surfaces,
- persistence and authentication discipline at the architectural level,
- the shared infrastructure boundary, extension rules, and the generated-file rule,
- architecture maturity, intentional deviations, and documentation architecture.

This document does not own:

- product capability inventory and surface backing: [`DOMAINS.md`](DOMAINS.md),
- journeys and state machines: [`FLOWS.md`](FLOWS.md),
- entities, schemas, migrations, retention: [`DATA.md`](DATA.md),
- resources, scenarios, and third-party services: [`INTEGRATIONS.md`](INTEGRATIONS.md),
- the UI source tree, breakpoint composition, and component layering:
  [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md) and [`EXPERIENCE.md`](EXPERIENCE.md),
- test seams and fakes: [`../internal/SEAMS.md`](../internal/SEAMS.md),
- test strategy: [`../internal/TESTING.md`](../internal/TESTING.md),
- the build order and current-state inventory:
  [`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md).

> **Current state (2026-09-22 audit).** The scenario has never worked end to end in the
> local runtime. The fourteen blocking defects are catalogued as **B1–B14** in
> [`../internal/REDESIGN_PLAN.md` §3.2](../internal/REDESIGN_PLAN.md#32-blocking-defects-fix-first-in-d0).
> The architecture-level ones:
>
> | ID | Defect | Intended fix |
> |---|---|---|
> | B1 | No principal in the local runtime; every workspace RPC returns `401 unauthenticated`. | Declare the platform authentication profile (D-032). |
> | B4 / B5 | Plans are one JSON row per workspace with no read RPC; Today and Week overwrite each other and regenerate on every load. | Plans as persisted dated occurrences with a read model (D-033). |
> | B7 | Planning builds its eligibility input differently from the profile handler. | One eligibility-input builder shared by planning, Explore, and swap. |
> | B12 | Multi-statement recipe writes run without a transaction. | Transactions wherever R22 and SYS-02 name all-or-nothing effects (D-034). |
> | B13 | No migrations; every schema is `CREATE TABLE IF NOT EXISTS`. | Versioned per-domain migrations (D-034). |
> | B14 | PDF export depends on a host font file. | Embed licensed print fonts (R25.2). |
>
> About 1.1k lines of domain logic (cost, catalog contribution, scaling, units and nutrient
> registries, providers, AI proposal, recurring jobs, billing, legacy import) are reachable
> only from tests. Treat every module below as **intended** unless the maturity table says
> otherwise.

## Scenario Shape

A scenario is one product expressed through three coordinated surfaces and one canonical
contract layer.

```
                        ┌──────────────────────────────────┐
                        │  Proto contracts (authored first) │
                        │  packages/proto/schemas/          │
                        │  nutrition-planner/v1/<domain>/   │
                        └───────────────┬──────────────────┘
                                        │ canonical wire shape
              ┌─────────────────────────┼─────────────────────────┐
              ▼                         ▼                         ▼
        ┌──────────┐             ┌──────────┐             ┌──────────┐
        │   ui/    │ Connect-JSON│  api/    │ Connect-JSON│  cli/    │
        │ React    │ ◀─────────▶ │   Go     │ ◀─────────▶ │   Go     │
        │ + Vite   │             │ HTTP     │             │ thin CLI │
        └────┬─────┘             └────┬─────┘             └──────────┘
             │                        │
             │  outbox (shopping,     ├──▶ pure domain core (Go): nutrition, cost,
             │  timers) + cache       │    eligibility, scaling, planning, presentation
             │                        │    chooser — no browser/db/net/model/clock
             ▼                        ├──▶ SQLite (per-domain schemas, migrations)
        browser state                 └──▶ adapters: image-tools (→ ai-gateway),
                                           personal-planner, notification-hub, USDA
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Authorization, transactions, revisions, idempotency, business rules, persistence, adapters, transport edge | Browser state, CLI formatting |
| Domain core (pure packages under `api/internal/`) | Deterministic nutrition, cost, eligibility, scaling, planning, media-presentation choice | Pure functions and versioned policy objects | Browser, database, network, model, or clock access |
| UI (`ui/`) | Browser presentation composed per medium | Interaction state, per-medium composition, accessible editing, i18n, the offline outbox for shopping and timers | A second planner, cost engine, eligibility check, or authorization decision |
| CLI (`cli/`) | Operator and agent wrapper | Argument parsing, output formatting, API invocation | Business rules, duplicated validation, a parallel planner |
| Contracts (`packages/proto/schemas/nutrition-planner/v1/`) | Wire shape | Proto messages and services, generated clients | Hand-written route or type mirrors |

The load-bearing principles:

1. **The API is the only surface with mutable business logic.** The UI renders the totals,
   eligibility, costs, and reasons the API returns (R23, ARCH-02).
2. **The domain core is callable without a browser, database, network, model, or clock.**
   It receives snapshots and policies and returns values, provenance, and structured
   reasons.
3. **Proto types flow from one source of truth.** Opaque JSON string fields (`draft_json`,
   `plan_json`, `preview_json` today) are replaced by typed messages as each area is rebuilt.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- scenario docs under `docs/`, lifecycle metadata under `.vrooli/`, `requirements/`, and
  `experience/`,
- proto schemas under `packages/proto/schemas/nutrition-planner/`,
- curated artwork and its manifest under the UI public assets root (see [`DATA.md`](DATA.md)).

The scenario does not own:

- shared packages under `packages/` (including `@vrooli/react-component-library`),
- Vrooli resource implementations,
- the scenarios it calls — image-tools, ai-gateway, personal-planner, notification-hub,
- generated proto outputs under `packages/proto/gen/`,
- calendar event timing and semantics, which personal-planner owns (R24.2).

Document dependency and resource decisions in [`INTEGRATIONS.md`](INTEGRATIONS.md).

### Module Boundaries

Specification ARCH-02 defines the baseline modules; R21 adds the redesign modules. Each
maps to the bounded contexts in [`DOMAINS.md`](DOMAINS.md). The last column records what
exists today.

| Module | Owns | Must Not Own | Today |
|---|---|---|---|
| Profile and rules | Diet, restrictions, targets, preferences, appearance preference | Billing truth, recipe parsing | Partial (`internal/profile`, `internal/eligibility`) |
| Food catalog | Food/product identities, revisions, nutrient evidence, unit mappings | Unreviewed model text as canonical truth | Partial (`internal/catalog`; never reaches plans) |
| Recipe library | Drafts, immutable revisions, method graphs, components, families, favourites | Current inventory balance | Partial (`internal/recipe`; no UI editing) |
| Nutrition engine | Unit-aware totals, coverage, target evaluation | Network fetches, UI state | Partial (evaluates only client-supplied intake) |
| Planning engine | Candidate eligibility, bounded search, plan assessment, explanations, recommendation context | Persistence side effects during search | Partial (greedy picker; cost hard-coded to 0) |
| Explore | Curated catalog query, sections, structured reasons, save and plan-from-context | A separate recommendation path with weaker restrictions | Absent |
| Media and scenes | MediaAsset, SceneTemplateVersion, MealPresentation, the deterministic presentation chooser, asset manifest | Recipe composition, eligibility, generation requests | Absent |
| Generation | GenerationPolicy, GenerationJob, BudgetReservation, quotes, review and activation, through an image-tools adapter | Direct provider calls or credentials (ai-gateway routing lives behind image-tools) | Absent (generic `internal/jobs` and `internal/entitlements` plumbing only) |
| Equipment | EquipmentType catalog, KitchenDevice with capabilities, EquipmentSceneManifest | Eligibility decisions (it supplies capabilities to them) | Absent (eight appliance strings on the profile) |
| Cooking sessions | CookingSession, CookingStepCompletion, CookingTimer, finish flow | Intake records (it hands off to intake) | Absent (one `localStorage` timer in the UI) |
| Inventory and costs | Stock events and qualitative assertions, storage, packages, observations, purchase and batch projections | Assuming planned actions happened | Partial (event log with decimal amounts only) |
| Shopping | ShoppingListSnapshot, stable rows, Review/Shop state, plan-change diffs | Purchases (it hands off to inventory) | Partial (string-keyed checks; all values "unknown") |
| Intake and feedback | Consumption, corrections, explicit feedback | Mutating historical revisions | Partial (per-date feedback, manual intake) |
| Calendar link | CalendarLink records and the personal-planner adapter | Calendar timing semantics, meal or intake changes | Absent |
| Transfer and documents | Versioned import/export, migration, legacy adapter, printable layout | Bypassing domain validation on import | Present, limited (backup omits eleven record kinds) |
| Provider and job adapters | External requests, structured proposals, bounded execution | Permissions derived from model output | Libraries only; no worker executes jobs |
| Application services | Authentication context, authorization, transactions, revisions, idempotency, orchestration | Duplicated formulas in handlers | Partial (idempotency tables; one transaction) |
| UI | Interaction state, per-medium composition, accessible editing | A second planner or cost engine | Prototype (see [`EXPERIENCE.md`](EXPERIENCE.md)) |

A module that starts using another module's vocabulary has crossed a boundary; split the
shared word into an owning domain instead of growing a generic bucket.

### Local Authentication Profile

Every workspace-scoped RPC derives its actor from the authenticated request principal; a
client-supplied workspace id is a request, never proof of access (SYS-03). The local
runtime must therefore resolve a principal without a hosted sign-in. The intended shape is
the platform's declared authentication profile — `hybrid` with default mode
`personal_local`, as `scenarios/git-control-tower/.vrooli/service.json` declares — per the
repository's `docs/concepts/IDENTITY-AND-AUTHENTICATION.md` (decision D-032). `personal_local`
is not an authorization bypass: handlers keep their ownership checks, and no handler grows
a "no principal" branch.

### Persistence Discipline

- **Plans are persisted dated occurrences** with stable identities and revisions, read
  through a plan/occurrence read model; generation runs only on Plan my week, Swap, or an
  explicit replan, and ApplyPlan validates occurrences server-side (D-033).
- **Versioned per-domain migrations** replace idempotent bootstrap before any column is
  added; foreign keys reference `workspaces`; stale template tables are migrated away
  (D-034, R25.1, SYS-05).
- **Transactions** for plan application, catalog-save-plus-plan, import and restore,
  purchase plus stock, preparation plus batch, intake plus correction, and budget
  reservation (R22, SYS-02). External side effects (calendar, generation) go through a
  durable outbox, never inside an open database transaction.
- **Read models over blobs.** No whole-aggregate JSON column is the only queryable store
  (DOM-06).

## Contracts And Data Flow

Wire shapes live in `.proto` files, not TypeScript interfaces, Go structs, or hand-written
JSON schemas.

```
packages/proto/schemas/nutrition-planner/v1/<domain>/<file>.proto
       │
       ▼
       make -C packages/proto generate
       │
       ├──▶ packages/proto/gen/go/nutrition-planner/v1/...            (api, cli)
       ├──▶ packages/proto/gen/go/nutrition-planner/v1/...connect     (Connect-Go)
       └──▶ packages/proto/gen/typescript/nutrition-planner/v1/...    (ui)
```

**Proto is authored first.** Today there are thirteen services and 58 RPCs (workspace,
recipe, catalog, cost, inventory, nutrition, supplement, routine, portability, profile,
eligibility, jobs, planning). The redesign adds contracts for media and presentation,
favourites, equipment devices, cooking sessions and timers, shopping lists and diffs,
inventory assertions, Explore, calendar links, generation, and a plan read model (R21.3
operation catalog). Capability endpoints describe what is actually configured, so an
unavailable adapter produces an honest disabled state instead of a dead button (R21.3).

REST is allowed only for the four `RESTReason` constants in `api/internal/module/module.go`:

| Reason | When it applies |
|---|---|
| `RESTReasonMultipartUpload` | Opaque file bytes: user photo upload, import files, label photos. |
| `RESTReasonWebhookReceiver` | A third-party-dictated shape. None planned. |
| `RESTReasonThirdPartyShape` | An externally defined contract. None planned. |
| `RESTReasonOpsProbe` | `GET /health`, static assets, the diagnostics probe. |

`cmd/gen-endpoints` rejects any endpoint path that is not a generated Connect procedure
unless it carries a `RESTException`. Even for a REST exception, the payload stays
proto-typed wherever possible.

### Generated-File Rule

Everything under a `generated/` directory is codegen output. Regenerate it with the owning
tool (`make -C packages/proto generate` for protos, `make temporal-models`, or the domain's flow command); **never
hand-edit** a generated file.

### State Ownership And Invalidation

Specification §17.3 and R22 separate these kinds of state:

| State | Owner | Rule |
|---|---|---|
| Server facts | API + SQLite | Authoritative profiles, recipes, plans, stock, sessions, timers, intake. |
| Derived assessments | Nutrition, planning, cost, eligibility, presentation engines | Keyed by input revisions and evaluator or policy version; recomputed, never mutated. |
| Editor drafts | UI | Dirty fields plus a base revision; blank numeric input stays null. |
| Pending outbox operations | UI (bounded, per workspace) | Operation id, base revision, desired state, status; replayed idempotently on reconnect (R22). |
| Local timer display | UI | Derived from server timer timestamps; never the only truth (R13.3). |
| Transient interface state | UI | Tabs, filters, scroll, selected day; persisted where R03 asks, never mutating domain data. |

The UI reads server state through react-query with keys that include the relevant
revisions (D-036). An explicit dependency map governs invalidation: ingredient composition
affects nutrition and eligibility; methods affect equipment and effort; prices affect cost;
stock affects shopping; targets affect assessments; plan and portion changes affect every
plan summary; equipment changes affect method eligibility; appearance affects presentation
only. Do not rely on page reloads for correctness.

## Shared Infrastructure

Shared infrastructure is allowed only when the code is business-vocabulary-free and used by
unrelated domains or surfaces.

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers | Today |
|---|---|---|---|---|
| `api/internal/server/` | Compose modules and middleware into one HTTP server. | Server lifecycle is not a product capability. | API entrypoint, handler modules. | Present |
| `api/internal/module/` | Module and endpoint descriptor types. | Common shape returned by every domain. | Handlers, server, endpoint codegen. | Present |
| `api/internal/modules/` | Thin registry for schemas and endpoints. | Boot and codegen need central lists. | `main.go`, `gen-endpoints`. | Present |
| `api/internal/database/` | System schema and database reachability seam. | Cross-cutting database infrastructure. | API boot, health, migrations. | Present (system schema empty) |
| `api/internal/middleware/`, `httpx/`, `httpc/` | Logging, security headers, error mapping, outbound HTTP seam. | Transport concerns. | Handlers, adapters. | Present |
| `api/internal/decimalx/`, `money/`, `units/` | Exact decimals, minor-unit money, unit registry. | Arithmetic substrate used by nutrition, cost, and shopping. | Domain core. | Present; `units` test-only today |
| Clock seam | Deterministic time (UTC instants, IANA zones) for timers, schedules, tests. | Time is cross-cutting and substitutable. | Timers, jobs, planning, repositories. | Absent — add before cooking timers |
| `api/internal/testutil/` | Cross-domain test harnesses and fakes. | Used by unrelated domains. | API tests. | Present |
| `ui/src/components/` | Shared presentation primitives (R05.3 list). | Used by unrelated features. | UI features and shell. | Two components today |
| `ui/src/test-utils/` | Render, a11y, and model test helpers. | Used by unrelated features. | UI tests. | Present |

If shared infrastructure starts using product vocabulary, move that piece back into the
owning domain.

## Extension Rules

Add product behavior by adding or updating the owning domain, not by growing generic
buckets. For a proto-backed domain:

1. Author proto messages and service methods under
   `packages/proto/schemas/nutrition-planner/v1/<domain>/`, then `make -C packages/proto generate` from the repository root.
2. Add pure domain code under `api/internal/<domain>/`, free of database, network, clock,
   and model access; add a migration for any schema change.
3. Add transport code under `api/handlers/<domain>/` with handler tests for authorization,
   revisions, idempotency, and transactions.
4. Register schemas and endpoints in `api/internal/modules/registry.go` and mount the module
   in `api/main.go`.
5. Add CLI commands under `cli/domains/<domain>/` as a thin translation layer.
6. Add UI API wrappers under `ui/src/api/` and feature code under `ui/src/features/<surface>/`,
   composed per medium as [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md) describes.
7. Update selectors, strings, endpoints, `[REQ:ID]`-tagged tests, the experience contract
   bindings, and `docs/manifest.json`.

Wire an existing test-only library into a production path or delete it; do not leave
production code reachable only from tests. For ownership update [`DOMAINS.md`](DOMAINS.md);
for persistence [`DATA.md`](DATA.md); for lifecycle [`FLOWS.md`](FLOWS.md).

## Architecture Maturity

Grounded in the 2026-09-22 audit. The PROGRESS log's 2026-09-18 "implemented" entries are
unverified claims (D-035); this table is the honest baseline.

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Wide and shallow | 13 services, 58 RPCs, ~10.2k production lines; handlers are real, not canned. | No local principal (B1); plan blob (B4/B5); no migrations (B13); one transaction; 0 % handler coverage; redesign modules absent. |
| Domain core | Partial | Decimal, money, recipe graph validator, eligibility evaluator exist with unit tests. | Cost, scaling, units, nutrient contribution are test-only; planner ignores cost and real effort. |
| UI | Prototype | Nine routes, RCL `AppShell/2`, typed Connect clients, i18n scaffolding. | ~85 KB of TSX in ~570 lines, 261 raw palette classes, hard-coded English, UTC dates, no breakpoint hook, no media, no redesign surfaces. |
| CLI | Scaffold | `workspace list`, `recipe list`, `recipe create`. | 55 of 58 RPCs omitted; 401 at runtime. |
| Docs | Redesign-aligned (2026-09-22) | v2.0 specification canonical; PRD regenerated; plan and mockup guide written. | Keep aligned with code as surfaces land. |

Use `docs/manifest.json` as the documentation contract.

## Intentional Deviations

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-09-18 | None from Vrooli platform standards. | Standard three-surface shape and Connect-RPC transport. | A platform rule must bend for a documented product constraint. |
| 2026-09-22 | The UI uses the scenario's Warm Kitchen design language instead of the default Operational Console (D-030). | Beauty and reference fidelity are product requirements (OT-P0-021). | A platform-wide design change. |
| 2026-09-22 | Curated artwork is produced through image-tools during development, not generated at runtime by default (D-029). | Bounded, reviewed, provenance-tracked assets; runtime generation is Off by default (R18). | OT-P1-007 opt-in generation lands. |
| 2026-09-22 | No Plan Manager for this effort (D-026). | Operator direction; a convergence loop tracked in scenario files. | Operator changes direction. |

## Documentation Architecture

One durable question, one canonical home.

| Concern | Canonical Document |
|---|---|
| Intended product behavior | `docs/reference/product-specification.md` |
| Visual target | `docs/reference/mockups/README.md` |
| System map, boundaries, extension rules | `docs/concepts/ARCHITECTURE.md` |
| Product capabilities and bounded contexts | `docs/concepts/DOMAINS.md` |
| Workflows and state transitions | `docs/concepts/FLOWS.md` |
| Data ownership, retention, migrations | `docs/concepts/DATA.md` |
| Resources, scenarios, external services | `docs/concepts/INTEGRATIONS.md` |
| Information architecture and surface composition | `docs/concepts/EXPERIENCE.md` |
| UI source tree and component layering | `docs/concepts/UI-ARCHITECTURE.md` |
| Binding design language | `DESIGN.md` |
| Build order, current-state inventory | `docs/internal/REDESIGN_PLAN.md` |
| Convergence findings and evidence | `docs/internal/REDESIGN_LEDGER.md` |
| Seams and test doubles | `docs/internal/SEAMS.md` |
| Testing strategy | `docs/internal/TESTING.md` |
| Decisions | `docs/internal/DECISIONS.md` |
| Known drift and deferred work | `docs/internal/PROBLEMS.md` |
| Change history | `docs/internal/PROGRESS.md` |

Every durable scenario document is registered in `docs/manifest.json`.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — bounded contexts and surface backing
- [`FLOWS.md`](FLOWS.md) — journeys and state machines
- [`DATA.md`](DATA.md) — entities, schemas, migrations
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`EXPERIENCE.md`](EXPERIENCE.md) — information architecture
- [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md) — UI source tree
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — seam registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
- [`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) — build order and inventory
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-025 onward for the redesign
- [`../reference/product-specification.md`](../reference/product-specification.md) — R21–R25, Appendix A §12, §17–§18
