# Data — Nutrition Planner

This document is the canonical data ownership and storage map for the scenario. Update it
when domains add tables, files, media, external records, retention rules, migrations,
imports, or exports.

Nutrition Planner stores a person's food and eating life: food logs, dietary restrictions,
allergen evidence, nutrition targets, supplement schedules, kitchen inventory, receipts,
personal food photos, and cooking history. That is private personal data by default, and the
storage design treats it that way. This document states the **intended** model (Appendix A
§12, R21–R22, R25) and, separately, **what exists today** (2026-09-22 audit).

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist, and where?
- Which domain owns each data shape, and what invariant protects it?
- Where do media assets live, and how are they identified?
- How do schema migrations, export formats, and calculation versions evolve independently?
- What is the retention, deletion, and privacy story?

## Storage Overview

| Store | Intended use | Today |
|---|---|---|
| SQLite (`modernc.org/sqlite` through `api-core/database`) | Every domain's records, per-domain schema beside its code, versioned migrations, foreign keys to `workspaces`. | `data/nutrition-planner.db`; 27 tables from 15 `api/internal/*/schema.sql` files plus the stale template `notes` and `attachments` tables. **Every table has zero rows.** Schemas are applied with `CREATE TABLE IF NOT EXISTS`; there are no migrations and `PRAGMA user_version` is never set. |
| Curated asset files (UI public assets root) | Bundled scene templates, composed scenes, editorial photos, equipment layers, ingredient icons, and their manifest. | None; `ui/public/` holds only PWA icons and a placeholder logo. |
| Private media storage (behind a blob seam) | User photos and source attachments, access-controlled per workspace. | None. |
| Browser storage (per account and workspace) | Appearance for first paint, editor drafts, the bounded offline outbox, cached shopping list and loaded recipe/session. | One `localStorage` timer key and the theme key. |

No external storage resource is required. Opaque bytes stay out of proto payloads and
behind a blob seam; see [`../internal/SEAMS.md`](../internal/SEAMS.md).

## Data Ownership

Each domain owns its tables and is the source of truth for its data. Domains follow
[`DOMAINS.md`](DOMAINS.md). `Today` names the existing tables; "—" means not built.

| Data | Owning Domain | Intended Source Of Truth | Today | Invariant |
|---|---|---|---|---|
| Workspace, idempotency | workspace | `api/internal/workspace/schema.sql` | `workspaces`, `workspace_idempotency` | Ownership is server-derived from the principal. |
| Profile, setup draft, preferences, appearance | profile-and-preferences | `api/internal/profile/schema.sql` | `profiles` (draft and applied JSON; no expected-revision check) | A draft is never active; theme has no domain side effects. |
| Diet rules, presets, allergen evidence | dietary-rules-and-eligibility | `api/internal/profile/schema.sql` | inside `profiles` | Preset versions stored; expanded rules are the contract. |
| Equipment devices, capabilities, scene manifest | equipment | `api/internal/equipment/schema.sql` | — (`profiles.appliances_json` strings) | Devices are distinct from capabilities; display strings are not ids. |
| Foods, products, revisions, nutrient definitions | food-and-product-catalog | `api/internal/catalog/schema.sql` | `catalog_concepts`, `catalog_revisions` | Revisions immutable once referenced. |
| Recipes, revisions, components, families, favourites | recipe-library | `api/internal/recipe/schema.sql` | `recipes`, `recipe_revisions`, `recipe_idempotency` | Stable id plus immutable revision; archive preserves pinned uses. |
| Media assets, scene templates, meal presentations | media-and-scenes | `api/internal/media/schema.sql` plus the asset manifest | — | Content hash immutable; approval and compatibility explicit. |
| Generation policy, jobs, budget reservations | generation | `api/internal/generation/schema.sql` | generic `jobs`, `entitlement_reservations` | Atomic reservation; exactly-once settlement. |
| Nutrition targets | nutrition-engine | `api/internal/nutrition/schema.sql` | `nutrition_targets` | At least one bound; history kept after deactivation. |
| Routines and supplement schedules | routines-and-supplements | `api/internal/routine/schema.sql`, `api/internal/supplement/schema.sql` | `routine_templates`, `routine_template_revisions`, `supplement_schedules`, `supplement_schedule_revisions` | Doses are user-defined and never changed by the planner. |
| Plans, occurrences, preparation tasks | planning-engine | `api/internal/planning/schema.sql` | `plans` — **one whole-plan JSON row per workspace** (B4) | Occurrences have stable identities and revisions (D-033). |
| Recommendation context | explore | `api/internal/explore/` (transient plus revalidation inputs) | — | Revalidated at apply; no stale rule bypass. |
| Cooking sessions, step completions, timers | cooking-sessions | `api/internal/cooking/schema.sql` | — | View, completion, timers, batches, intake are separate records. |
| Inventory events, assertions, batches, purchases | inventory-and-batches | `api/internal/inventory/schema.sql` | `inventory_events` (decimal amounts only), `inventory_batches`, `inventory_receipt_proposals` | Append-only events; reservations never decrement on-hand. |
| Price observations | cost-and-price-book | `api/internal/cost/schema.sql` | `price_observations` | Editing creates a new observation. |
| Shopping list snapshots, rows, manual items | shopping-list | `api/internal/shopping/schema.sql` | `shopping_checks` (string-keyed checks) | Plan diffs preserve manual and fulfilled quantities. |
| Consumption events, corrections | intake-and-feedback | `api/internal/nutrition/schema.sql` | `nutrition_intake_events` | Never edited in place; correction is a linked event. |
| Feedback | intake-and-feedback | `api/internal/feedback/schema.sql` | `meal_feedback` (keyed per date, not per occurrence) | Missing feedback stays unknown. |
| Calendar links | calendar-link | `api/internal/calendarlink/schema.sql` | — | One stable external key per source task and purpose. |
| Restore checkpoints, import operations | transfer-and-documents | `api/internal/portability/schema.sql` | `portability_restore_checkpoints`, `portability_restore_operations`, `portability_recipe_import_operations` | Staging never mutates live data. |
| Jobs | provider-and-job-adapters | `api/internal/jobs/schema.sql` | `jobs` | Extraction success separate from application. |
| Entitlements | entitlements-and-budgets | `api/internal/entitlements/schema.sql` | `workspace_entitlements`, `entitlement_reservations` | Own-data export never gated. |
| Stale template data | none | migrate away | `notes`, `attachments` | Remove through a migration (D-034). |

## Schema Map

The entity inventory is Appendix A DOM-06 plus the R21.1 extension entities. Use relational
constraints, indexes for real access paths, and foreign keys to `workspaces`; an append-only
correction history plus current projections is sufficient — full event sourcing is not
required, and no endlessly growing JSON blob may be the only queryable store.

### Baseline entities (DOM-06)

| Entity | Owner | Minimum Persisted Information | Today |
|---|---|---|---|
| Workspace / profile | workspace, profile-and-preferences | Owner, locale/timezone/currency, settings revision, setup progress, active rules, preferences. | Partial |
| DietRule / preset | dietary-rules-and-eligibility | Rule definition, required/preferred, preset version, effective dates. | Partial (JSON on profile) |
| NutrientDefinition | food-and-product-catalog | Identity, canonical unit, form semantics, conversions, source mappings. | Code registry only |
| Food / Product + revision | food-and-product-catalog | Preparation state, serving mappings, nutrient profile, allergen evidence, provenance. | Present |
| SourceArtifact | food-and-product-catalog | Type, URL/provider id/text or attachment ref, date, hash, parser version, privacy scope. | — |
| Recipe + revision | recipe-library | Identity/status; immutable yield, ingredients, methods, sources. | Present |
| MealFamily | recipe-library | Typed slots, options, portion bounds. | — |
| NutritionTarget | nutrition-engine | Bounds, period, scope, enforcement, provenance, effective dates. | Present |
| SupplementSchedule | routines-and-supplements | Product revision, dose, recurrence, dates, pause. | Present |
| Plan + revision | planning-engine | Range/timezone, scope, profile ref, solver inputs/version/seed, status. | Blob |
| PlannedOccurrence | planning-engine | Local date, slot, recipe revision or open item, servings, method, lock, batch relation. | Inside blob |
| PreparationTask | planning-engine | Output, resources, active/elapsed effort, dependent occurrences. | — |
| PreparedBatch | inventory-and-batches | Recipe revision, yield, remaining, storage. | Present |
| ConsumptionEvent | intake-and-feedback | Local/UTC time, pinned content and quantity, source, snapshot totals, correction link. | Present |
| PriceObservation | cost-and-price-book | Package, amount/currency, retailer, conditions, date, freshness, source. | Present |
| InventoryEvent / assertion | inventory-and-batches | Item/batch, amount/unit or qualitative mode, event type, evidence, time, correction ref. | Decimal events only |
| ShoppingPlan / line | shopping-list | Input revisions, derived quantities, packages, overrides, check state, purchase links. | Checks only |
| PurchaseEvent | inventory-and-batches | Packages, spend/currency, retailer/date, receipt source, application key. | Via events and receipt proposals |
| Feedback / preference | intake-and-feedback | Reason/action, scope, time, source occurrence, inferred vs explicit. | Per date |
| Job | provider-and-job-adapters | Input ref, status, dedup key, budget, progress, errors, results, applied operation. | Present, no worker |
| Change / operation record | application-services | Actor, idempotency key, expected/applied revision, affected ids, undo links. | Per-domain idempotency tables |

### Redesign extension entities (R21.1)

| Entity | Owner | Key Fields | Invariant |
|---|---|---|---|
| AppearancePreference | profile-and-preferences | owner, appearance (Light/Evening/Follow device), presentation (Immersive/Editorial/Minimal), revision | No food or domain side effects; never triggers generation. |
| MediaAsset | media-and-scenes | owner/scope, content hash, storage ref, media type, dimensions, source, rights, state | Access-controlled; immutable original; no unvalidated remote URLs. |
| SceneTemplateVersion | media-and-scenes | family, version, geometry, variant refs, food region, safe regions, compatibility, status | Immutable once referenced by a job or result; retire blocks new generation only. |
| MealPresentation | media-and-scenes | recipe ref, asset, scene version, appearance, viewport class, crop, focal point, approval | Rendering eligibility differs from recipe eligibility. |
| GenerationPolicy | generation | owner, mode, cap/unit/period, allowed triggers and variants | Automatic mode needs an explicit bounded authorization. |
| GenerationJob | generation | inputs, dedup key, state, provider-role version, attempt limit, usage, result | A stale completion cannot overwrite newer user choices. |
| BudgetReservation | generation | policy/period, job/attempt, reserved and settled amounts, status | Atomic cap enforcement; exactly-once settlement; unknown usage stays pending. |
| EquipmentType | equipment | stable id, categories, capability definitions, icon | Display strings are not capability ids. |
| KitchenDevice | equipment | owner, type, selected functions, capacity/count, revision | Physical resources distinct from capabilities. |
| EquipmentSceneManifest | equipment | version, theme variants, slots, layers, markers | Bounded deterministic placement; no effect on eligibility. |
| CookingSession | cooking-sessions | recipe and method refs, scale, occurrence, state, selected step, revision | View, completion, timers, batches, intake are separate. |
| CookingStepCompletion | cooking-sessions | session/step, assertion time, actor, correction | Never inferred from visiting a step. |
| CookingTimer | cooking-sessions | session/step, duration, start/target timestamps, paused remainder, state, revision | Expiry does not complete a step. |
| ShoppingListSnapshot | shopping-list | plan scope, requirement revision, mode/state, rows | Plan diffs preserve manual and fulfilled quantities. |
| RecommendationContext | explore | target slot, inputs, policy version, reason codes | Revalidated at apply. |
| CalendarLink | calendar-link | source task, external event id and revision, sync state, operation identity | Unique stable external identity; no duplicate events on retry. |

### Decimal, Unit, And Money Rules (DOM-03)

- Decimal or rational arithmetic in the domain; canonical decimal strings on the wire and in
  exports; only display rounds.
- A quantity carries unit, dimension, preparation state where relevant, and basis. `1 scoop`
  is not a mass until mapped for a specific product revision.
- Inventory amounts additionally carry an amount mode — exact, estimated, qualitative, out,
  unknown — and the original expression ("About half a bag") separately from any reviewed
  quantity mapping (R15.1).
- Unknown and explicit zero are distinct in every serializer and form.
- ISO currency codes; integer minor units with a currency-specific exponent; no summing
  across currencies without a dated exchange-rate policy.

### Identity And Time (DOM-02)

- Opaque ids; every aggregate belongs to a server-authorized workspace; references pin
  revisions.
- UTC instants for events; local dates plus IANA timezone for planning. The UI must derive
  local dates in the workspace timezone — today it uses `toISOString().slice(0,10)` (UTC) in
  eight places (B10).
- Timers store absolute start and target instants plus a paused remainder; display is derived
  from the current time (R13.3).

### Asset Manifest

Curated artwork is a versioned manifest plus files (R17.4, R19.2–R19.3; production plan in
[`../internal/REDESIGN_PLAN.md` §7](../internal/REDESIGN_PLAN.md#7-production-artwork)).

| Rule | Detail |
|---|---|
| Location | Files under the UI public assets root, for example `assets/scenes/<family>/v1/<appearance>-<composition>.webp` and `assets/equipment/kitchen-v1/<appearance>/<layer>.webp`. |
| Identity | Stable manifest ids independent of filenames; no development path or conversation filename becomes a production URL. |
| Required metadata | id, type, content hash, dimensions, format, source and rights, generation provenance (prompt version, reference ids and hashes, model role, reported cost), approval state and reviewer, recipe-revision compatibility, scene version, appearance, composition, focal point, safe-text rectangles, crop bounds — normalized 0–1 coordinates in a defined source coordinate system. |
| Renditions | Generated once during processing: wide and compact sizes in a modern format plus a compatible fallback; width and height reserved in markup. |
| Private media | User photos stay access-controlled per workspace even though shared scene templates are public assets; signed URLs are never cached indefinitely or treated as identities. |

## Migrations And Compatibility

Three versions evolve independently (SYS-05):

| Version | Meaning | Rule |
|---|---|---|
| Database migration version | The schema applied to the local database. | Versioned per-domain migrations recorded in a migrations table or `PRAGMA user_version` (D-034); upgrades tested against a populated prior database. |
| Export schema version | `daily.recipes` / `daily.workspace`, `schemaVersion: 2`. | Storage changes do not renumber export formats, and vice versa. |
| Calculation / policy version | Nutrition, planning, cost, eligibility, badge, and presentation-chooser versions. | Cached assessments key on it; a code change invalidates them without a data migration. |

**Today:** idempotent bootstrap only — a column added to an existing table never reaches an
existing database (B13).

**Data preservation (R25.1).** Inspect the actual schema and data before migrating; back up
under repository policy; add theme, media, equipment, session, and shopping fields with
conservative defaults; keep recipe ids, revisions, plans, intake, grocery state, preferences,
and provenance; map existing appliance strings to stable capability ids and leave ambiguous
values visible for review; never reset anyone to demo data or seed stock. Plans without scene
metadata still open through the editorial or minimal treatment. The live database is empty
today, but the migration path must still be exercised against a populated fixture database
(AT-051).

## Import / Export

| Path | Format | Owner | Today |
|---|---|---|---|
| Native recipe collection | `daily.recipes`, `schemaVersion: 2`, records keyed by `(kind, id, revision)`. | transfer-and-documents | Present (export/import with conflict policy and remapping). |
| Native workspace backup | `daily.workspace`, `scope.kind: workspace_backup`. | transfer-and-documents | Present but covers only workspace, recipes, and plan; declares eleven omissions. |
| Legacy prototype adapter | Single recipes, arrays, `{recipes: […]}`, `schemaVersion: 1` workspaces. | transfer-and-documents | Library only; not wired. |
| Grocery CSV | Spreadsheet view of a shopping list with formula-injection handling. | shopping-list | Present (all values "unknown" today). |
| Recipe / week / grocery PDF | Real text/vector documents in the light print theme on A4 and Letter. | transfer-and-documents | Recipe and weekly PDFs exist in the old style and depend on a host font (B14). |

Rules (R25.2, UX-DAT-05):

- A complete export includes every supported record kind — profile and rules, catalog
  revisions, recipes and revisions, plans and occurrences, targets, routines and supplement
  schedules, prices, inventory and batches, shopping lists, consumption and corrections,
  feedback, **scene and media metadata, equipment devices and capabilities, appearance and
  presentation preferences, cooking sessions where chosen, and calendar links as
  non-authoritative external references**. A JSON export that omits binary originals says so;
  a full media backup uses a bounded archive with a manifest and checksums.
- Secrets, tokens, billing instruments, signed URL credentials, and tenant authority are
  never exported; restore never grants them.
- Import stages first and validates ownership remapping, schema version, references, cycles,
  and limits; no live mutation during preview; full restore requires a checkpoint.
- Generated assets are not regenerated on restore unless explicitly requested and
  budget-authorized.
- Round-trip compares semantic content after normalization.
- R0 import limits (configurable): 2 MB JSON, 200 recipes, 80 ingredient uses and 30 steps
  per recipe, 10,000 characters of notes.

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Workspace and profile | Account deletion (commercial release). | Active until deletion; derived caches dropped first. | Account deletion not built. |
| Recipe identity | Archive, or delete of an unreferenced draft. | Archived recipes stay resolvable for pinned uses. | Permanent-delete policy deferred. |
| Consumption events and corrections | Not deletable in place. | Corrections are linked events. | — |
| Inventory and purchase events | Append-only. | Projection recomputable. | — |
| Price observations | Superseded, not deleted. | Usable with an age label. | Freshness window not chosen. |
| Cooking sessions and timers | Session completed or abandoned. | Kept with completion events; timers settle to a terminal state. | Not built. |
| Shopping list snapshots | List completed or archived. | Picked-up and purchase history kept. | Not built. |
| Media assets | Owning recipe or presentation removed; asset unreferenced. | Originals kept for recrops and export while referenced; rejected generation results keep the source photo. | Not built. |
| Generation jobs and reservations | Settled, canceled, or expired. | Usage and reservation history kept for budget honesty. | Not built. |
| Calendar links | Task unscheduled or link revoked. | Link history kept; never deletes meals or intake. | Not built. |
| Offline outbox and caches | Replay success, sign-out, account switch. | Partitioned by account and workspace; cleared on sign-out per policy (R22). | Not built. |
| Stale `notes`/`attachments` tables | Migration. | Remove. | Present in the live database. |

Derived assessments and caches are never authoritative. Consumption, purchases, corrections,
settled usage, and operation records are protected and never treated as cache.

## Privacy Notes

- Food logs, restrictions, allergen evidence, targets, supplement schedules, inventory,
  receipts, private food photos, and cooking history are private to their workspace.
- Send an external provider only the task-relevant subset: a food-image generation request
  needs the reviewed visual summary and references, not target history or supplement
  schedules (R25.3). Prefer opaque local references to account identifiers.
- Private photos stay private by default; shared scene artwork does not imply shared recipes.
- Logs carry operation ids, timings, counts, error codes, and safe revision references — never
  raw food histories, full prompts containing user details, signed media URLs, or tokens
  (R25.4).
- Offline caches and service-worker storage are partitioned by account and workspace; an
  offline account switch never shows the previous person's meals (R22, AT-050).
- Export of owned data is never behind a paid upgrade.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — ownership by domain
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — persistence discipline and state ownership
- [`FLOWS.md`](FLOWS.md) — lifecycle states that govern these records
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — image-tools, personal-planner, and other adapters
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) — blocking defects and artwork plan
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — blob and clock boundaries
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — upload validation and access control
- [`../reference/product-specification.md`](../reference/product-specification.md) — R17.4, R19, R21–R22, R25, Appendix A §12, §18, §20.8–§20.9
