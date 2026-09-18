# Domains — Nutrition Planner

This document is the canonical map of product capabilities, bounded contexts, and ownership
for this scenario. Keep it current whenever a domain is added, renamed, split, merged, or
removed.

The map below follows specification §17.2 (module boundaries) and §12 (domain vocabulary).
The product has not been implemented yet: these are the intended bounded contexts, and the
`Source Paths` column names where each should live once built. Two scaffold artifacts are
explicitly not product scope and are marked in
[`Domain Details`](#domain-details): the `notes` worked example and the generated dashboard
placeholders.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature, CLI command, and test
  surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md). Workflow details
belong in [`FLOWS.md`](FLOWS.md). Storage details belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and database reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/nutrition-planner/v1/shared/health.proto` |
| workspace-and-profile | Own the authenticated workspace, profile, setup draft, locale, timezone, currency, and preferences. | Every user-owned record belongs to one authorized workspace; setup is resumable and applying it is explicit. | Workspace, profile, setup draft, preferences. | service | crud, validation | Workspace, Profile, SetupDraft, Preference | `api/internal/profile/`, `api/handlers/profile/`, `cli/domains/profile/`, `ui/src/features/kitchen/`, `packages/proto/schemas/nutrition-planner/v1/profile/` |
| dietary-rules | Compile diet presets, composable exclusions, and allergen evidence into eligibility rules. | Required restrictions are explicit, independently editable, and cannot be traded against cost, effort, or variety. | DietRule, presets, exclusion assertions, evidence states. | validation | classification, service | DietPreset, Exclusion, AllergenEvidence, Eligibility | `api/internal/rules/`, `api/handlers/rules/`, `cli/domains/rules/`, `packages/proto/schemas/nutrition-planner/v1/rules/` |
| kitchen-capabilities | Model the user's appliances, storage, and method feasibility. | Plan around what the user can actually cook with, including "no appliances". | Capability catalog and profile capability selections. | classification | query, validation | Capability, Appliance, PreparationMethod | `api/internal/kitchen/`, `packages/proto/schemas/nutrition-planner/v1/kitchen/` |
| food-and-product-catalog | Own food/product identities, immutable nutrient revisions, label evidence, and unit mappings. | One source of truth for what a food or branded product is, with provenance and unknowns preserved. | Food, Product, ProductRevision, NutrientDefinition, SourceArtifact. | crud | provider, validation | Food, Product, Fraction, PrepState | `api/internal/catalog/`, `api/handlers/catalog/`, `cli/domains/catalog/`, `packages/proto/schemas/nutrition-planner/v1/catalog/` |
| recipe-library | Own stable recipe identities, immutable content revisions, reusable components, and meal families. | Every plan and history entry pins an exact revision, so edits never rewrite the past. | Recipe, RecipeRevision, Component, MealFamily. | crud | service, validation | Recipe, Revision, Component, MealFamily | `api/internal/recipes/`, `api/handlers/recipes/`, `cli/domains/recipes/`, `ui/src/features/meals/`, `packages/proto/schemas/nutrition-planner/v1/recipes/` |
| recipe-method-graph | Validate and expose the acyclic preparation graph of steps, allocations, and named outputs. | One graph drives the map, reading view, and cooking mode, and ingredient nutrients are counted once. | Steps, allocations, output components, dependency edges (within a revision). | validation | mutation, query | Step, Allocation, Output, DependencyDAG | `api/internal/recipes/graph/`, `ui/src/features/meals/graph/` |
| preparation-methods-and-units | Own the unit registry, canonical conversions, and reviewed alternate preparation methods. | A quantity is meaningless without its unit, dimension, basis, and preparation state. | Unit definitions, aliases, product-specific conversions. | query | validation, infrastructure | Unit, Dimension, Conversion, Basis | `api/internal/units/`, `api/internal/recipes/methods/`, `packages/proto/schemas/nutrition-planner/v1/units/` |
| nutrition-engine | Compute unit-aware contributions, known subtotals, coverage, and target evaluations. | Totals name their basis, scope, and completeness, and unknown is never folded into zero. | No independent facts; derived assessments cached by input revision. | scoring | validation, reporting | Contribution, Target, Evaluation, Completeness | `api/internal/nutrition/`, `packages/proto/schemas/nutrition-planner/v1/nutrition/` |
| planning-engine | Filter eligible candidates, run the bounded deterministic search, evaluate plans, and produce explanations. | A reproducible draft with concrete occurrences, reasons, unresolved slots, and input references; apply is separate. | Plan drafts and run metadata; not accepted plans. | orchestration | scoring, validation | Candidate, Draft, Score, ReasonCode, Infeasibility | `api/internal/planning/`, `api/handlers/planning/`, `cli/domains/planning/`, `ui/src/features/week/`, `packages/proto/schemas/nutrition-planner/v1/planning/` |
| inventory-and-batches | Own the stock ledger, prepared batches, leftovers, and reservations. | Purchased, prepared, reserved, and consumed are distinct and idempotent; future reservations never consume on-hand stock. | InventoryEvent, assertion, PreparedBatch, reservation. | mutation | reporting, validation | StockLedger, Batch, Leftover, Reservation | `api/internal/inventory/`, `api/handlers/inventory/`, `ui/src/features/groceries/`, `packages/proto/schemas/nutrition-planner/v1/inventory/` |
| cost-and-price-book | Own price observations, packages, allocated portion cost, checkout estimate, and recorded actual spend. | Portion cost, checkout cost, and actual spend are separate and labeled; no fabricated savings. | PriceObservation, PackageOffer, assertions. | scoring | query, reporting | PriceObservation, Package, PortionCost, Checkout | `api/internal/costs/`, `api/handlers/costs/`, `ui/src/features/groceries/prices/`, `packages/proto/schemas/nutrition-planner/v1/costs/` |
| shopping-plan | Derive a shopping plan and lines from an explicit plan, inventory policy, and price choices. | Groceries are derived, overridable, and honest about unknown package counts and prices. | ShoppingPlan, ShoppingLine, overrides, check state. | aggregation | reporting, mutation | ShoppingPlan, ShoppingLine, Missing, PackagePlan | `api/internal/shopping/`, `api/handlers/shopping/`, `ui/src/features/groceries/`, `packages/proto/schemas/nutrition-planner/v1/shopping/` |
| intake-and-feedback | Record consumption, corrections, deviations, and explicit preferences. | What was eaten is what the user recorded; missing feedback stays unknown, not noncompliance. | ConsumptionEvent, correction linkage, Feedback. | mutation | validation, classification | ConsumptionEvent, Correction, Feedback, Preference | `api/internal/intake/`, `api/handlers/intake/`, `ui/src/features/today/`, `packages/proto/schemas/nutrition-planner/v1/intake/` |
| transfer-and-documents | Own versioned import/export, legacy migration, and printable documents. | Owned data is portable and printable without a paid upgrade, and import bypasses no validation. | Import/export jobs, staged envelopes, rendered documents. | orchestration | reporting, validation | Envelope, StagedImport, Restore, Document | `api/internal/transfer/`, `api/handlers/transfer/`, `cli/domains/transfer/`, `ui/src/features/data/`, `packages/proto/schemas/nutrition-planner/v1/transfer/` |
| provider-and-job-adapters | Bound external requests, structured provider proposals, and background job execution. | Providers are optional, failures degrade to manual entry, and model output never authorizes a domain write. | Job records, provider cache, proposals, budgets. | provider | orchestration, infrastructure | Job, Proposal, ProviderAdapter, Budget | `api/internal/providers/`, `api/internal/jobs/`, `api/handlers/jobs/`, `packages/proto/schemas/nutrition-planner/v1/jobs/` |
| application-services | Authorize, transact, check revisions, and orchestrate cross-domain operations. | One operation boundary enforces ownership, idempotency, and all-or-nothing effects. | Operation records and idempotency keys; no duplicated formulas. | composition-root | orchestration, validation | Operation, IdempotencyKey, Revision, Workspace | `api/internal/app/`, `api/main.go`, `api/internal/modules/` |
| ui | Compose domain APIs into the responsive Today-first console. | The next decision is one screen away, and every derived number shows its scope and honesty state. | Browser preferences and transient interaction state only. | query | reporting, mutation | Today, Week, Groceries, Meals | `ui/src/pages/`, `ui/src/layout/`, `ui/src/consts/`, `ui/src/api/` |

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live backend state.
- Primary archetype: reporting / query.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Tests: handler, module, UI feature, and accessibility tests.

### workspace-and-profile

- **Owns**: the authenticated workspace, the profile revision, the resumable setup draft,
  locale/timezone/currency settings, and the three planning priority values.
- **Key rule**: a setup draft is never active rules. Applying a reviewed draft is an atomic
  operation that previews conflicts with existing plans; an unfinished allergy change does
  not affect eligibility until applied.
- **Key rule**: fixtures and sample content can never be silently adopted as real user data
  (OT-P0-001).
- **Targets**: OT-P0-001, OT-P0-002. **Specification**: §6, §12.6.

### dietary-rules

- **Owns**: data-driven diet presets, composable exclusions, allergen evidence states, and
  the compiled rules that eligibility consumes.
- **Key rule**: a diet name is not an exhaustive policy; the expanded active rules are the
  contract, and preset versions are stored so a future preset update cannot silently change
  the user's rules.
- **Key rule**: missing allergen evidence is unknown, not absence. Restricted allergens with
  unresolved evidence produce `needs_information`, never a pass.
- **Targets**: OT-P0-006. **Specification**: §6.2, §12.1, §12.5.

### kitchen-capabilities

- **Owns**: the stable appliance capability vocabulary and each profile's selected
  capabilities, including zero.
- **Key rule**: a recipe is eligible only if at least one complete reviewed preparation
  method uses available capabilities. Oven and stove are not interchangeable.
- **Targets**: OT-P0-006. **Specification**: §6.3, §9.6.

### food-and-product-catalog

- **Owns**: food concepts, branded products, immutable food/product revisions, nutrient
  definitions and evidence, source artifacts, and unit mappings.
- **Key rule**: a generic food profile and a branded label profile are different things and
  must not be added together as independent intake.
- **Key rule**: provider or model output arrives as a proposal; only validated domain
  operations create canonical revisions (specification §16.4, DOM-07 #14).
- **Deliberately distinct**: food concept vs product vs revision — see
  [`Shared Concepts`](#shared-concepts).
- **Targets**: OT-P0-010, OT-P1-002. **Specification**: §13.1, §16.4, §12.1.

### recipe-library

- **Owns**: recipe identity (stable, mutable status) and recipe revisions (immutable
  content), plus reusable components and meal families.
- **Key rule**: a name-only draft is valid. Null yield, unknown nutrients, and missing
  method stay unknown; a draft is excluded from automatic planning until readiness
  requirements are met.
- **Key rule**: authoring status (draft/ready/archived) is independent of profile
  eligibility (eligible/ineligible/needs_information).
- **Key rule**: component expansion is acyclic and counts each ingredient once.
- **Targets**: OT-P0-003, OT-P0-004, OT-P0-005. **Specification**: §8, §20.1.

### recipe-method-graph

- **Owns**: validation and traversal of the acyclic dependency graph of steps, ingredient
  allocations, named outputs, branches, and joins.
- **Key rule**: an ingredient used in several steps is one ingredient use with several
  allocations; joining outputs never re-adds their input nutrients.
- **Key rule**: cycles, dangling references, duplicate outputs, and over-allocations are
  rejected with specific, editable error paths.
- **Targets**: OT-P0-005. **Specification**: §9.2–§9.3, §20.5.

### preparation-methods-and-units

- **Owns**: the unit registry with aliases and validated conversions, plus reviewed
  alternate preparation methods for a recipe.
- **Key rule**: changing a display unit never changes the underlying quantity. A universal
  grams-per-cup conversion is forbidden; piece weight and density require explicit,
  source-linked data.
- **Key rule**: a microwave and stovetop variant are distinct reviewed methods sharing
  ingredients where appropriate.
- **Specification**: §12.3, §9.6.

### nutrition-engine

- **Owns**: unit-aware contribution arithmetic, known subtotals, unresolved-contributor
  tracking, coverage, and target evaluation.
- **Key rule**: unknown and zero are distinct in every serializer. An upper bound cannot
  pass when unknown contributions remain; a lower-only bound can pass on the known subtotal
  with partial completeness kept visible.
- **Key rule**: store `not_applicable`, `unknown`, `fail`, and `pass` as different outcomes,
  and keep target evaluation separate from data completeness and evidence quality.
- **Targets**: OT-P0-010. **Specification**: §13, §20.4, FIX-02/FIX-04.

### planning-engine

- **Owns**: candidate eligibility, the bounded seed-reproducible search, plan-level
  evaluation, and structured explanations and infeasibility reasons.
- **Key rule**: generation is separate from transactional apply. A search timeout means "no
  feasible plan found in this search", never proven infeasibility.
- **Key rule**: locked occurrences and required exclusions are hard constraints; unknown
  prices prevent validating a required budget cap rather than making a candidate look cheap.
- **Targets**: OT-P0-007, OT-P0-009. **Specification**: §14, §20.10, FIX-10.

### inventory-and-batches

- **Owns**: the event-based stock ledger, inventory assertions, prepared batches, leftovers,
  and reservations.
- **Key rule**: purchases, preparation, and consumption are distinct idempotent operations.
  Future reservations are separate from on-hand stock, and replanning excludes its own
  replaced reservations.
- **Key rule**: cooking consumes raw ingredients once and creates prepared stock; eating a
  portion reduces batch stock, never the raw ingredients a second time.
- **Targets**: OT-P0-011. **Specification**: §15.3–§15.5, §20.3, FIX-03.

### cost-and-price-book

- **Owns**: price observations with retailer, package, conditions, date, and freshness;
  package offers; allocated portion cost; estimated checkout; and recorded actual spend.
- **Key rule**: all three money views stay separate and labeled. A missing price is unpriced,
  not zero, and an internal ranking fallback is never displayed as an observed price.
- **Targets**: OT-P0-012. **Specification**: §15.1–§15.2, §15.6.

### shopping-plan

- **Owns**: the derived shopping plan and lines, selected packages, user overrides, and
  check state.
- **Key rule**: checking a line is checklist state only; it is not a purchase, stock
  adjustment, or consumption. Manual items survive replanning, and overrides carry a
  mismatch badge when plan need changes.
- **Targets**: OT-P0-011. **Specification**: §10.1–§10.2.

### intake-and-feedback

- **Owns**: consumption events, corrections, deviations, explicit dislikes, and cooldown
  rules.
- **Key rule**: a scheduled meal is not evidence that it was eaten. Missing feedback is
  unknown; a missing-ingredients rejection is not a dislike. Logging does not require
  rating, weighing, or explaining.
- **Key rule**: historical intake keeps its pinned revision and snapshot totals; correction
  is an explicit linked event.
- **Targets**: OT-P0-008, OT-P1-006. **Specification**: §7.7, §8.5, §20.7, FIX-07.

### transfer-and-documents

- **Owns**: versioned envelopes (`daily.recipes`, `daily.workspace`), the legacy prototype
  adapter, staged imports, recoverable restore checkpoints, PDF rendering, and CSV.
- **Key rule**: parse and preview never mutate live data. Apply is atomic against an expected
  revision. Restore replaces workspace content, never identity, credentials, or entitlement.
- **Targets**: OT-P0-013. **Specification**: §11, §18.5, §20.8–§20.9, FIX-08/FIX-09.

### provider-and-job-adapters

- **Owns**: external request execution, structured provider proposals, job state, bounded
  retries, cancellation, and per-workspace budgets.
- **Key rule**: providers are optional at every release; manual entry and deterministic
  planning remain fully usable with no provider configured. Extraction success is tracked
  separately from proposal application.
- **Targets**: OT-P1-001 through OT-P1-005. **Specification**: §16.

### application-services

- **Owns**: authorization, transaction boundaries, revision checks, idempotency keys, and
  cross-domain operation orchestration.
- **Key rule**: every mutation is scoped to the authenticated workspace derived from the
  server session; a client-supplied workspace id is a request, not proof of access.
- **Key rule**: idempotency keys identify a logical mutation; the same key with a
  materially different payload is a conflict, not a duplicate.
- **Targets**: OT-P0-014. **Specification**: §17.4, §18.1–§18.2.

### ui

- **Owns**: route composition, responsive presentation, accessible editing, i18n, and honest
  loading/empty/partial/error states.
- **Does not own**: nutrition arithmetic, eligibility, planning, cost, or authorization.
- **Key rule**: an unavailable or incomplete source is rendered as an explicit state with
  its reason; it is never converted to zero.
- **Targets**: OT-P0-008, OT-P0-015. **Specification**: §4–§5. See
  [`EXPERIENCE.md`](EXPERIENCE.md).

### Scaffold example — `notes` (not product scope)

<!-- EXAMPLE-DOMAIN:notes START -->
The template ships the `notes` domain as a worked CRUD vertical slice with a binary
upload exception. It exists so a new scenario can copy the vertical-slice shape; it is
**not** a Nutrition Planner capability and `template-manager detemplate` removes it once
the first real domain is green.

| Domain | Responsibility | Primary Archetype | Source Paths |
|---|---|---|---|
| notes | Template worked example only; demonstrates the expected vertical slice and the sanctioned multipart exception (source-artifact upload is the product analog). | crud | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/nutrition-planner/v1/notes/` |

- Does not own: any product capability, table, or requirement.
- Storage: `api/internal/notes/schema.sql`; removed with the domain.
<!-- EXAMPLE-DOMAIN:notes END -->

## Shared Concepts

These terms are used the same way across every domain document.

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | [`ARCHITECTURE.md`](ARCHITECTURE.md). |
| Seam | Test-substitutable boundary wired once in production. | [`../internal/SEAMS.md`](../internal/SEAMS.md). |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Derived assessment | A value recomputed from input revisions and an evaluator version, never stored as independent truth. | The computing engine. |

### Vocabulary Separation (DOM-01)

Specification §12.1 requires these concepts to stay separate even while the first database
co-locates them. Collapsing any two of them is the root defect the domain model exists to
prevent.

| Concept | Meaning | Must not be confused with |
|---|---|---|
| Food concept | Generic edible item/state, such as cooked lentils. | A particular package or store offer. |
| Product | Specific branded/packaged formulation. | Every product with a similar display name. |
| Food/product revision | Snapshot of composition, serving definitions, and evidence. | Mutable current catalog metadata. |
| Ingredient use | Quantity of a food/product/component in one recipe. | An additional food purchase each time a step references it. |
| Recipe | Stable meal/component identity. | One immutable version of its quantities and method. |
| Recipe revision | Immutable composition, yield, method, and assertions. | Current recipe pointer. |
| Meal component | A reusable prepared or ready-to-use part. | A free substitute with identical nutrition. |
| Meal family | Reviewed template of compatible slots/options. | A fully quantified meal. |
| Planned occurrence | Date/slot, pinned recipe, selected servings, status. | Evidence of consumption. |
| Prepared batch | Actual output quantity from a cooking event. | A scheduled future cooking task. |
| Consumption event | What the user recorded eating/taking. | Automatic plan acceptance. |
| Price observation | Dated package offer or user assertion. | A universally current market price. |
| Purchase event | Recorded acquisition/spend. | A checkbox on a shopping list. |
| Inventory assertion/event | Evidence about stock or a stock change. | A precise measurement when only inferred. |
| Target | A configured bound/preference with scope. | A universally appropriate dietary recommendation. |
| Source artifact | Original text/label/provider record. | Approved structured truth. |
| Assessment | Evaluation against a versioned policy and input revisions. | A permanent property of a meal for all users. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they are real enough to
affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| household-collaboration | The initial releases model one profile with adjustable serving counts. Person-specific targets and shared roles need deliberate design. | OT-P2-003, when a second person's targets must coexist. |
| preference-learning | Transparent explicit dislikes and cooldown rules come first; learned ranking needs enough voluntary data and a calibrated evaluation. | OT-P1-006 evidence plus a measured outcome. |
| price-research-agents | Needs authorized sources, package identity, freshness, and a value/cost budget before any automation. | OT-P2-005 with a bounded research job. |
| public-recipe-sharing | Requires attribution, permission, redaction, moderation, and copied-versus-linked revision policy. | OT-P2-004, if commissioned. |
| billing-and-entitlements | Architectural readiness now; actual billing is a later commercial decision with no confirmed provider. | OT-P2-002, when a provider is selected. |

## Non-Domains

These are important but must not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/clock/` — deterministic time seam.
- `api/internal/testutil/` — cross-domain test harnesses.
- `packages/proto/` — generated wire contracts, not a bounded context.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.
- The `notes` scaffold example — template reference only, removed by detemplate.

If one of these starts using product vocabulary, split the product piece into an owning
domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`EXPERIENCE.md`](EXPERIENCE.md) — the UI decision
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
- [`../reference/product-specification.md`](../reference/product-specification.md) — canonical product specification
