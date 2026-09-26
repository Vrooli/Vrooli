# Domains — Nutrition Planner

This document is the canonical map of product capabilities, bounded contexts, and ownership
for this scenario. Keep it current whenever a domain is added, renamed, split, merged, or
removed.

The map follows specification ARCH-02 (module boundaries), DOM-01 (vocabulary), and the
redesign extension entities in R21.1. The `Source Paths` column names **real paths** where
code exists today and the **intended location** where it does not; the `Today` line in each
domain's details says which. The current-state evidence comes from the 2026-09-22 audit in
[`../internal/REDESIGN_PLAN.md` §3](../internal/REDESIGN_PLAN.md#3-current-state-inventory).

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature, CLI command, and test
  surface?
- Which domain backs each user-facing surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md). Workflow details
belong in [`FLOWS.md`](FLOWS.md). Storage details belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|
| health | Report runtime readiness and database reachability. | None. | reporting | query | HealthHandler | `api/handlers/health/` |
| diagnostics | Report scoped, actionable data health and configured capabilities. | None; reads other domains. | reporting | query | DataHealthFinding, CapabilityState | `api/internal/diagnostics/`, `api/handlers/diagnostics/` |
| workspace | Own the authenticated workspace, its locale, timezone, currency, and ownership root. | Workspace, idempotency records. | service | crud, validation | Workspace, Owner, IdempotencyKey | `api/internal/workspace/`, `api/handlers/workspace/`, `cli/domains/workspace/`, `packages/proto/schemas/nutrition-planner/v1/workspace/` |
| profile-and-preferences | Own the profile, the resumable setup draft, grouped preferences, priorities, and the appearance preference. | Profile, SetupDraft, preference groups, AppearancePreference. | service | crud, validation | Profile, SetupDraft, Preference, Appearance | `api/internal/profile/`, `api/handlers/profile/`, `ui/src/features/onboarding/`, `packages/proto/schemas/nutrition-planner/v1/profile/` |
| dietary-rules-and-eligibility | Compile diet presets and exclusions into required rules and evaluate recipe eligibility with evidence states. | DietRule, preset versions, exclusion assertions; eligibility results are derived. | validation | classification, scoring | DietPreset, Exclusion, AllergenEvidence, Eligibility | `api/internal/profile/rules.go`, `api/internal/eligibility/`, `api/handlers/eligibility/`, `packages/proto/schemas/nutrition-planner/v1/eligibility/` |
| equipment | Own the equipment catalog, the user's physical devices, their capabilities, and the equipment scene manifest. | EquipmentType, KitchenDevice, EquipmentSceneManifest. | classification | crud, query | EquipmentType, KitchenDevice, Capability, SceneSlot | `api/internal/equipment/`, `api/handlers/equipment/`, `ui/src/features/kitchen/` |
| food-and-product-catalog | Own food/product identities, immutable nutrient revisions, label evidence, nutrient definitions, and unit mappings. | Food, Product, revisions, NutrientDefinition, SourceArtifact. | crud | provider, validation | Food, Product, NutrientDefinition, Basis | `api/internal/catalog/`, `api/handlers/catalog/`, `api/internal/nutrients/`, `api/internal/units/`, `packages/proto/schemas/nutrition-planner/v1/catalog/` |
| recipe-library | Own stable recipe identities, immutable revisions, method graphs, components, families, favourites, and archive state. | Recipe, RecipeRevision, Component, MealFamily, favourite flags. | crud | validation, mutation | Recipe, Revision, Draft, Component, Favourite | `api/internal/recipe/`, `api/handlers/recipe/`, `cli/domains/recipe/`, `ui/src/features/meals/`, `ui/src/features/recipe/`, `packages/proto/schemas/nutrition-planner/v1/recipe/` |
| media-and-scenes | Own media assets, scene template versions, meal presentations, the asset manifest, and the deterministic presentation chooser. | MediaAsset, SceneTemplateVersion, MealPresentation. | classification | query, crud | MediaAsset, SceneTemplate, Treatment, SafeTextRegion | `api/internal/media/`, `api/handlers/media/`, `ui/public/` |
| generation | Quote, reserve, run, review, and activate bounded image-generation jobs through the image-tools adapter. | GenerationPolicy, GenerationJob, BudgetReservation. | orchestration | provider, validation | GenerationPolicy, Quote, Reservation, DedupKey | `api/internal/generation/`, `api/internal/jobs/`, `api/internal/entitlements/` |
| nutrition-engine | Compute unit-aware contributions, known subtotals, coverage, and target evaluations over explicit scopes. | NutritionTarget; assessments are derived. | scoring | validation, reporting | Contribution, Target, Evaluation, Scope | `api/internal/nutrition/`, `api/handlers/nutrition/`, `packages/proto/schemas/nutrition-planner/v1/nutrition/` |
| routines-and-supplements | Own recurring slot templates and fixed user-defined supplement schedules. | RoutineTemplate revisions, SupplementSchedule revisions. | crud | service | Routine, Anchor, SupplementSchedule, Dose | `api/internal/routine/`, `api/handlers/routine/`, `api/internal/supplement/`, `api/handlers/supplement/` |
| planning-engine | Filter eligible candidates, run the bounded deterministic search, persist plans as dated occurrences, and explain choices. | Plan, PlannedOccurrence, PreparationTask, drafts and run metadata. | orchestration | scoring, validation | Occurrence, Slot, Draft, Lock, ReasonCode | `api/internal/planning/`, `api/handlers/planning/`, `ui/src/features/today/`, `ui/src/features/week/`, `packages/proto/schemas/nutrition-planner/v1/planning/` |
| explore | Rank curated and user-approved recipes for an explicit context through the planner's eligibility path, with structured reasons. | Curated catalog entries, RecommendationContext. | scoring | query | RecommendationContext, Section, Reason | `api/internal/explore/`, `api/handlers/explore/`, `ui/src/features/explore/` |
| cooking-sessions | Own cooking sessions, explicit step completion, durable timers, and the finish flow. | CookingSession, CookingStepCompletion, CookingTimer. | mutation | orchestration | Session, StepCompletion, Timer, BatchConfirmation | `api/internal/cooking/`, `api/handlers/cooking/`, `ui/src/features/cooking/` |
| inventory-and-batches | Own the stock ledger, qualitative and exact assertions, storage, use-soon flags, prepared batches, leftovers, and reservations. | InventoryEvent, assertions, PreparedBatch, reservations, purchase events. | mutation | reporting, validation | StockLedger, AmountMode, Batch, Reservation | `api/internal/inventory/`, `api/handlers/inventory/`, `packages/proto/schemas/nutrition-planner/v1/inventory/` |
| cost-and-price-book | Own price observations, packages, allocated portion cost, checkout estimate, and recorded actual spend. | PriceObservation, package offers. | scoring | query, reporting | PriceObservation, PortionCost, Checkout, Coverage | `api/internal/cost/`, `api/handlers/cost/`, `api/internal/money/`, `packages/proto/schemas/nutrition-planner/v1/cost/` |
| shopping-list | Derive stable shopping rows from a plan scope and inventory policy, hold Review/Shop state, and produce plan-change diffs. | ShoppingListSnapshot, rows, manual items, overrides, picked-up state. | aggregation | mutation, reporting | ShoppingList, Row, PickedUp, HaveThis, Diff | `api/internal/shopping/`, `ui/src/features/groceries/` |
| intake-and-feedback | Record consumption, corrections, deviations, and explicit preferences. | ConsumptionEvent, corrections, Feedback. | mutation | validation, classification | ConsumptionEvent, Correction, Feedback | `api/internal/nutrition/intake_sqlite.go`, `api/internal/feedback/` |
| calendar-link | Link preparation and cooking tasks to personal-planner events through a verified adapter. | CalendarLink. | provider | orchestration | CalendarLink, ExternalKey, SyncState | `api/internal/calendarlink/` |
| transfer-and-documents | Own versioned import/export, legacy migration, restore, and printable documents. | Staged imports, restore checkpoints, rendered documents. | orchestration | reporting, validation | Envelope, StagedImport, Restore, Document | `api/internal/portability/`, `api/handlers/portability/`, `ui/src/features/transfer/`, `packages/proto/schemas/nutrition-planner/v1/portability/` |
| provider-and-job-adapters | Bound external requests, structured proposals, and background job execution. | Job records, provider cache, proposals. | provider | orchestration, infrastructure | Job, Proposal, ProviderAdapter | `api/internal/providers/`, `api/internal/assistance/`, `api/internal/jobs/`, `api/handlers/jobs/` |
| entitlements-and-budgets | Enforce per-workspace optional-compute budgets and entitlement boundaries. | Entitlements, reservations. | validation | service | Entitlement, Reservation, Allowance | `api/internal/entitlements/` |
| application-services | Authenticate, authorize, transact, check revisions, and orchestrate cross-domain operations. | Operation and idempotency records. | composition-root | orchestration, validation | Principal, Operation, Revision | `api/main.go`, `api/internal/modules/`, `api/internal/server/` |
| ui | Compose domain APIs into the five-destination, per-medium application. | Browser preferences, drafts, the offline outbox. | query | reporting, mutation | Today, Week, Meals, Groceries, Kitchen | `ui/src/app/`, `ui/src/layout/`, `ui/src/theme/`, `ui/src/api/`, `ui/src/features/` |

## Domain Details

Each entry names what the domain owns, its load-bearing rules, the operational targets it
serves, and what exists today.

### health and diagnostics

- **Owns**: readiness, database reachability, scoped data-health findings (failed jobs,
  stale prices, unknown targets, unresolved recipes), and the capability states other
  surfaces read to decide whether an action exists (R21.3, OPS-02).
- **Key rule**: an unconfigured provider is a supported state, not an unhealthy app.
- **Targets**: OT-P0-014, OT-P0-017. **Today**: present; the UI calls the wrong diagnostics
  path (B6). The template `capabilities` module advertising `audio-tools` is a remnant to
  remove.

### workspace

- **Owns**: the ownership root for every record; the actor is derived from the
  authenticated principal (SYS-03).
- **Key rule**: the local runtime resolves a principal through the platform authentication
  profile (D-032); handlers never skip ownership checks.
- **Targets**: OT-P0-001. **Today**: present, but unreachable — every RPC returns 401 (B1);
  the UI can create duplicate workspaces (B11).

### profile-and-preferences

- **Owns**: the applied profile and its revision, the resumable setup draft (Your food, Your
  kitchen, Your rhythm, Ready), the Preferences groups (R15.3: food rules, household and
  servings, time and effort, variety and favourites, planning routine, nutrition targets and
  supplements, shopping), priority weights, and the appearance preference (Light, Evening,
  Follow device; presentation Immersive, Editorial, Minimal; generation permission).
- **Key rule**: a draft is never active; applying is atomic and previews conflicts. Theme
  has no food or planning side effects (R05.2).
- **Targets**: OT-P0-002, OT-P0-016, OT-P0-020. **Today**: partial; `Get` returns raw
  no-rows (B2), a scan bug collapses lists (B9), and onboarding is unreachable.

### dietary-rules-and-eligibility

- **Owns**: data-driven presets, composable exclusions, allergen evidence states, compiled
  rules, and the stateless eligibility evaluator.
- **Key rule**: missing evidence is `needs_information`, never a pass; no preference weight
  overrides a required rule (ACT-020). Planning, Explore, and swap build eligibility input
  through one shared builder (B7).
- **Targets**: OT-P0-006, OT-P0-018. **Today**: the evaluator exists; planning ignores
  `ExcludedGroups` (B7).

### equipment

- **Owns**: the initial catalog (R15.2), physical devices distinct from capabilities (a
  range provides cooktop and oven; a multicooker provides only its selected functions),
  counts and capacities, and the scene manifest (slots, layers, markers).
- **Key rule**: tiles are the complete selection interface; the scene reflects selection
  from data; zero equipment is valid; removing a device previews affected future methods
  (R15.2, R20).
- **Targets**: OT-P0-020, OT-P0-021. **Today**: absent; eight appliance strings on the
  profile.

### food-and-product-catalog

- **Owns**: food concepts, products, immutable revisions, nutrient definitions and evidence,
  source artifacts, unit mappings.
- **Key rule**: provider or model output arrives as a proposal; a generic profile and a
  label profile are never added together.
- **Targets**: OT-P0-010, OT-P1-002. **Today**: CRUD exists; nutrients never reach recipes
  or plans because the contribution path is test-only.

### recipe-library

- **Owns**: recipe identity and status, immutable revisions (yield, ingredients, methods,
  evidence, source), components, families, favourites, archive.
- **Key rule**: a name-only draft is valid; readiness is separate from eligibility; edits
  create revisions and never rewrite pinned history (FIX-01, FIX-07).
- **Targets**: OT-P0-003, OT-P0-004, OT-P0-005. **Today**: revisions and the graph validator
  exist; create/update is not transactional (B12); the UI cannot edit a recipe.

### media-and-scenes

- **Owns**: media assets (content hash, dimensions, format, rights, provenance, approval),
  scene template versions (geometry, food region, safe regions, compatibility), meal
  presentations (recipe revision, asset, appearance, composition, crop, focal point), and
  the deterministic chooser: honor Minimal; Editorial uses an eligible photo; Immersive tries
  an approved matching scene, then an approved cutout, then an ordinary photo, then Minimal
  (R17.1).
- **Key rule**: choosing a presentation never enqueues generation; an image failure falls to
  the next treatment without shifting titles or controls; rendering eligibility is not
  recipe eligibility.
- **Targets**: OT-P0-017, OT-P0-021. **Today**: absent; no media anywhere.

### generation

- **Owns**: the generation policy (Off, Ask each time, Automatic within budget), quotes,
  atomic budget reservations and settlement, dedup keys, job states, review, and activation
  (R18).
- **Key rule**: default Off; no generation from views, theme changes, search, hover, or
  resize; a stale result is never activated against a newer revision; execution goes through
  the image-tools adapter only (D-029).
- **Targets**: OT-P1-007, OT-P1-004, OT-P2-006. **Today**: generic jobs and entitlement
  reservations exist with no worker.

### nutrition-engine

- **Owns**: contribution arithmetic, subtotals, unresolved contributors, coverage, target
  evaluation over Selected meal, Planned day, Recorded so far, Expected day, and Selected
  week (NUT-04).
- **Key rule**: unknown is not zero; an upper bound cannot pass with unbounded unknowns; a
  dinner's protein never appears as the day's (R08.3).
- **Targets**: OT-P0-010. **Today**: evaluates only client-supplied intake; plans contribute
  nothing.

### routines-and-supplements

- **Owns**: recurring slot templates with anchors, and supplement schedules with fixed
  user-entered doses, pause, and resume.
- **Key rule**: the optimizer never changes a dose (R23 #13); templates generate dated
  occurrences, never history.
- **Targets**: OT-P0-010. **Today**: revisioned CRUD exists; the UI asks for raw IDs.

### planning-engine

- **Owns**: candidate filtering, bounded seed-reproducible search, persisted plans as dated
  occurrences with stable identities, locks, open and social slots, leftovers links,
  preparation tasks, swap quotes, and explanations.
- **Key rule**: generation is separate from apply; a timeout is not infeasibility; Today and
  Week read the persisted plan (D-033); Next week never replans unrelated days silently.
- **Targets**: OT-P0-007, OT-P0-008, OT-P0-009. **Today**: greedy picker with `Cost: 0`;
  one plan JSON row per workspace (B4); no read RPC (B5).

### explore

- **Owns**: the curated catalog query, sections (For your week, Use what you have, Quick
  meals, Something different, Cook once eat twice, Affordable additions), structured reasons,
  Save (idempotent), Add to or Replace an occurrence with revalidation.
- **Key rule**: hard constraints filter first through the planner's eligibility path; a
  reason never claims more than its evaluated inputs (R11.2).
- **Targets**: OT-P0-018. **Today**: absent.

### cooking-sessions

- **Owns**: sessions pinned to recipe revision, method, scale, and occurrence; step
  completion events; timers with absolute timestamps and revisions; the finish flow.
- **Key rule**: viewing a step never completes it; timer expiry never completes a step or
  meal; finishing never records intake by itself (R13).
- **Targets**: OT-P0-019. **Today**: absent; one fixed `localStorage` timer in the viewer.

### inventory-and-batches

- **Owns**: the event ledger, amount modes (exact, estimated, qualitative, out, unknown) with
  the original expression, storage location, use-soon flags with source, last-checked
  evidence, prepared batches, reservations, purchase events.
- **Key rule**: reservations never decrement on-hand; batches consume raw ingredients once;
  "Have some" never subtracts invented grams (R23 #8–#10).
- **Targets**: OT-P0-011, OT-P0-020. **Today**: decimal-only event log; no storage or
  qualitative model.

### cost-and-price-book

- **Owns**: price observations with conditions and freshness, package offers, allocated
  portion cost, checkout estimate, recorded actual spend.
- **Key rule**: the three money views stay separate; unknown price is never free.
- **Targets**: OT-P0-012. **Today**: observation CRUD exists; the calculator is test-only.

### shopping-list

- **Owns**: stable rows derived from canonical requirement grouping with source
  contributions, Review/Shop mode, picked-up state, manual items, overrides, and plan-change
  diffs (Added, Changed, No longer needed).
- **Key rule**: checking is picked-up only; Have this is a stock assertion, not a check;
  Confirm purchases is the only path to purchase events (R14.2).
- **Targets**: OT-P0-011, OT-P0-014. **Today**: rows come from step inputs with every
  quantity "unknown"; checks are keyed by strings.

### intake-and-feedback

- **Owns**: consumption events with pinned snapshots, corrections, deviation reasons,
  explicit dislikes, cooldowns.
- **Key rule**: a scheduled or cooked meal is not evidence of eating; missing feedback stays
  unknown.
- **Targets**: OT-P0-008, OT-P1-006. **Today**: manual intake events; feedback keyed per date,
  not per occurrence.

### calendar-link

- **Owns**: links between a nutrition task (prep or cooking) and an external calendar event:
  source task and revision, external event id and revision, sync state, operation identity,
  and a stable idempotent external key.
- **Key rule**: deleting a calendar block unschedules time only; completing a calendar task
  never records eating; a cross-day move previews plan implications (R24.3).
- **Targets**: OT-P1-008. **Today**: absent.

#### Ownership Boundary With personal-planner

| Concern | Owner |
|---|---|
| Meal choices, recipes, servings, ingredient requirements, prep and cooking tasks, actual food records | nutrition-planner |
| Calendar event timing, calendar semantics, availability | personal-planner (through its verified API) |
| The link between the two | nutrition-planner `calendar-link` |

Meal slot dates alone do not require timed events. Without the adapter, prep tasks stay
untimed inside nutrition-planner; nutrition-planner never builds a competing calendar
(R24.2–R24.3).

### transfer-and-documents

- **Owns**: `daily.recipes` and `daily.workspace` envelopes, the legacy adapter, staged
  imports, restore checkpoints, recipe/week/grocery PDFs, grocery CSV.
- **Key rule**: parse and preview never mutate; restore never carries authority; exports
  include new media, equipment, and preference metadata and state omissions (R25.2).
- **Targets**: OT-P0-013. **Today**: present but limited — the backup omits eleven record
  kinds and PDFs need a host font (B14).

### provider-and-job-adapters

- **Owns**: external request execution, structured proposals, job state, retries,
  cancellation, and safe URL fetch.
- **Key rule**: every provider is optional; extraction success is tracked separately from
  proposal application; fetched content is untrusted data.
- **Targets**: OT-P1-001, OT-P1-002, OT-P1-003, OT-P1-004, OT-P1-005. **Today**: libraries
  with tests; nothing executes jobs.

### entitlements-and-budgets

- **Owns**: per-workspace optional-compute allowances and reservations shared with the
  generation budget.
- **Key rule**: required restrictions, nutrition checks, and own-data export are never gated.
- **Targets**: OT-P2-001, OT-P2-002. **Today**: reservations exist; billing is a disabled seam.

### application-services

- **Owns**: authentication context, authorization, transaction boundaries, revision checks,
  idempotency, orchestration.
- **Key rule**: the same idempotency key with a materially different payload is a conflict.
- **Targets**: OT-P0-001, OT-P0-014. **Today**: idempotency tables exist; only restore uses a
  transaction.

### ui

- **Owns**: routes, per-medium composition, accessible editing, i18n, honest lifecycle
  states, the offline outbox for shopping and timers.
- **Does not own**: arithmetic, eligibility, planning, cost, presentation choice, or
  authorization.
- **Targets**: OT-P0-015, OT-P0-016, OT-P0-021. **Today**: prototype; see
  [`EXPERIENCE.md`](EXPERIENCE.md) and [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md).

### Which Domain Backs Each Surface

| Surface | Route (R03.1) | Backing domains |
|---|---|---|
| Today | `/today` | planning-engine, media-and-scenes, nutrition-engine, intake-and-feedback, routines-and-supplements |
| Week | `/week` | planning-engine, nutrition-engine, cost-and-price-book, inventory-and-batches (leftovers), calendar-link (Schedule prep) |
| Meals — Your meals | `/meals` | recipe-library, dietary-rules-and-eligibility, media-and-scenes |
| Explore | `/meals/explore` | explore, planning-engine, dietary-rules-and-eligibility, inventory-and-batches, media-and-scenes |
| Recipe detail and map | `/meals/:id` | recipe-library, nutrition-engine, media-and-scenes, transfer-and-documents (print) |
| Cooking | `/cook/:sessionId` | cooking-sessions, recipe-library, inventory-and-batches, intake-and-feedback |
| Groceries | `/groceries` | shopping-list, inventory-and-batches, cost-and-price-book |
| Kitchen — On hand | `/kitchen` | inventory-and-batches, explore (Find meals) |
| Kitchen — Equipment | `/kitchen` | equipment, dietary-rules-and-eligibility (affected methods) |
| Kitchen — Preferences | `/kitchen` | profile-and-preferences, dietary-rules-and-eligibility, nutrition-engine (targets), routines-and-supplements |
| Onboarding | setup flow | profile-and-preferences, equipment, dietary-rules-and-eligibility |
| Settings | `/settings` | profile-and-preferences (appearance, units), generation, calendar-link, transfer-and-documents (Data & exports), diagnostics |

Nutrition and Data are no longer primary destinations (D-031): nutrition analysis lives in
Week › Nutrition, the Today overview, and the recipe Nutrition tab; transfer lives in
Settings › Data & exports.

## Shared Concepts

These terms are used the same way across every domain document.

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | This document defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same capability. | [`ARCHITECTURE.md`](ARCHITECTURE.md). |
| Seam | Test-substitutable boundary wired once in production. | [`../internal/SEAMS.md`](../internal/SEAMS.md). |
| Requirement | Implementation-facing measurement tied to the PRD. | `requirements/`. |
| Derived assessment | A value recomputed from input revisions and a policy version, never stored as independent truth. | The computing engine. |
| Capability state | What an adapter or optional feature can actually do right now. | diagnostics. |

### Vocabulary Separation (DOM-01)

These concepts stay separate even where storage co-locates them. Collapsing any two of them
is the root defect the domain model exists to prevent.

| Concept | Meaning | Must not be confused with |
|---|---|---|
| Food concept | Generic edible item or state, such as cooked lentils. | A particular package or store offer. |
| Product | Specific branded or packaged formulation. | Every product with a similar name. |
| Food/product revision | Snapshot of composition, serving definitions, and evidence. | Mutable catalog metadata. |
| Ingredient use | Quantity of a food, product, or component in one recipe. | An extra purchase each time a step references it. |
| Recipe | Stable meal or component identity. | One immutable version of it. |
| Recipe revision | Immutable composition, yield, method, and assertions. | The current recipe pointer. |
| Planned occurrence | Date and slot, pinned recipe, servings, status. | Evidence of consumption. |
| Prepared batch | Actual output of a cooking event. | A scheduled cooking task. |
| Consumption event | What the user recorded eating or taking. | Plan acceptance or a finished cooking session. |
| Step viewed | The step on screen in cooking mode. | A step completion event. |
| Timer elapsed | A timer reached zero. | A completed step or meal. |
| Picked up | A shopping row checked while shopping. | A purchase, stock change, or intake. |
| Have this | A stock assertion with quantity or qualitative evidence. | A checked shopping row. |
| Available (inventory) | The user asserts some stock exists. | A measured quantity or food-safety claim. |
| Physical device | A piece of equipment the user owns. | A capability it provides. |
| Media asset | Bytes plus provenance and rights. | A presentation decision or recipe evidence. |
| Serving inspiration | A generated image of the meal. | Proof of ingredients, portions, or nutrition. |
| Appearance | Light, Evening, or Follow device. | Meal type, recipe, or plan. |
| Price observation | Dated package offer or user assertion. | A universally current price. |
| Target | A configured bound with scope. | A universal dietary recommendation. |
| Assessment | Evaluation against a versioned policy and input revisions. | A permanent property of a meal. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| scene-studio | User-created scene templates need budgets, region marking, and review (R18.5). | OT-P2-006. |
| household-collaboration | Person-specific targets and shared roles need deliberate design. | OT-P2-003. |
| preference-learning | Transparent explicit rules come first; learned ranking needs data and calibration. | OT-P1-006 evidence plus a measured outcome. |
| price-research-agents | Needs authorized sources, identity, freshness, and a value budget. | OT-P2-005. |
| public-recipe-sharing | Needs attribution, permission, redaction, and moderation. | OT-P2-004, if commissioned. |
| billing | Architectural readiness only; no provider selected. | OT-P2-002. |

## Non-Domains

These are important but must not become product domains:

- `api/internal/server/`, `api/internal/module/`, `api/internal/modules/` — composition and
  registry substrate.
- `api/internal/database/`, `api/internal/middleware/`, `api/internal/httpx/`,
  `api/internal/httpc/` — cross-cutting infrastructure.
- `api/internal/decimalx/`, `api/internal/money/`, `api/internal/units/` — arithmetic
  substrate.
- `api/internal/testutil/` — cross-domain test harnesses.
- `api/internal/capabilities/` — template remnant advertising `audio-tools`; remove.
- `packages/proto/` — generated wire contracts.
- `ui/src/components/`, `ui/src/test-utils/` — shared UI primitives and test support.

If one of these starts using product vocabulary, split the product piece into an owning
domain.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and module boundaries
- [`FLOWS.md`](FLOWS.md) — journeys and state machines
- [`DATA.md`](DATA.md) — entities and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`EXPERIENCE.md`](EXPERIENCE.md) — surfaces and composition
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) — current-state inventory and build order
- [`../reference/product-specification.md`](../reference/product-specification.md) — R03, R21, R24, Appendix A §12, §17
