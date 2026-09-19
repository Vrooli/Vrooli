# Data — Nutrition Planner

This document is the canonical data ownership and storage map for the scenario. Update it
when domains add tables, files, blobs, external records, retention rules, migrations,
imports, or exports.

Nutrition Planner stores a person's food and eating life: food logs, dietary restrictions,
allergen evidence, nutrition targets, supplement schedules, receipts, and label photos.
That is private personal data by default, and the storage design treats it that way.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes kept independent of export formats and calculation versions?

## Storage Overview

The scenario uses embedded SQLite through `modernc.org/sqlite`, matching the
specification's single private-workspace model. The database path is resolved from the
scenario id by `api-core/storage`, and the API applies schemas on startup through
`api-core/database`. A per-domain schema file lives beside the code that interprets it;
the `system schema` is the only cross-cutting, non-domain table set.

No external storage resource is required at any release through R2. External storage is
introduced only when a real domain needs it; document that decision in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing `.vrooli/service.json`. Opaque bytes
(label photos, imported files, source attachments) stay outside proto payloads and behind
a blob seam; see [`../internal/SEAMS.md`](../internal/SEAMS.md).

## Data Ownership

Each domain owns its tables and is the source of truth for its data. The `health` domain
owns no product data — it only probes configured database reachability. Ownership follows
the bounded contexts in [`DOMAINS.md`](DOMAINS.md).

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Workspace, profile, setup draft, preferences | workspace-and-profile | SQLite | `api/internal/profile/schema.sql` | Until the workspace is deleted. | Applying a setup draft updates the active profile revision. |
| Diet rules, presets, exclusions, allergen evidence | dietary-rules | SQLite | `api/internal/rules/schema.sql` | Deactivated rules keep their effective window. | Preset version is stored so updates cannot silently change active rules. |
| Nutrient definitions and unit mappings | preparation-methods-and-units | SQLite | `api/internal/units/schema.sql` | Historical definitions retained. | Form and unit semantics are stable identity, not display labels. |
| Foods, products, and their revisions | food-and-product-catalog | SQLite | `api/internal/catalog/schema.sql` | Revisions are immutable once referenced. | A refresh creates a proposal; it never mutates a used revision. |
| Source artifacts (text, URL, label) | food-and-product-catalog | SQLite metadata + BlobStore bytes | `api/internal/catalog/schema.sql` | Until the owning draft/revision is deleted. | Bytes are referenced by metadata; a text-only export states the omission. |
| Recipes and recipe revisions | recipe-library | SQLite | `api/internal/recipes/schema.sql` | Identity archives; revisions that plans reference are kept. | Stable logical id plus immutable revision number. |
| Meal families and components | recipe-library | SQLite | `api/internal/recipes/schema.sql` | Kept while any revision references them. | Component relationships must be acyclic. |
| Nutrition targets | nutrition-engine | SQLite | `api/internal/nutrition/schema.sql` | Deactivation retains history and ends enforcement at `effectiveThrough`. | At least one bound is required for an active target. |
| Supplement schedules | intake-and-feedback | SQLite | `api/internal/intake/schema.sql` | Pause retains history; delete removes the schedule. | The planner never changes a dose. |
| Plans and plan revisions | planning-engine | SQLite | `api/internal/planning/schema.sql` | Draft plans are replaceable; accepted plans are kept. | Records solver/scoring version, seed, and input references. |
| Planned occurrences and preparation tasks | planning-engine | SQLite | `api/internal/planning/schema.sql` | Kept with the plan; history is never edited. | Pin an exact recipe revision and chosen method. |
| Prepared batches and leftovers | inventory-and-batches | SQLite | `api/internal/inventory/schema.sql` | Kept until explicitly wasted or consumed. | Records the actual recipe revision and yield. |
| Consumption events and corrections | intake-and-feedback | SQLite | `api/internal/intake/schema.sql` | **Never deleted in place.** Corrected by a linked event. | Snapshot totals use the pinned revision at record time. |
| Price observations | cost-and-price-book | SQLite | `api/internal/costs/schema.sql` | Freshness policy by source/category; older observations remain usable with a label. | Editing creates a correction or new observation. |
| Inventory events and assertions | inventory-and-batches | SQLite | `api/internal/inventory/schema.sql` | Append-only event history; projection is derived. | "Have some" stays qualitative; it cannot fabricate a precise amount. |
| Shopping plans and lines | shopping-plan | SQLite | `api/internal/shopping/schema.sql` | Kept with the plan; overrides retained. | A check state is not a purchase. |
| Purchase events | inventory-and-batches | SQLite | `api/internal/inventory/schema.sql` | Kept as recorded acquisition history. | Idempotent on a durable source transaction identity. |
| Feedback and preferences | intake-and-feedback | SQLite | `api/internal/intake/schema.sql` | Kept until reset; learned values are inspectable and resettable. | Explicit dislikes outrank weak inference. |
| Import, research, and export jobs | provider-and-job-adapters | SQLite | `api/internal/jobs/schema.sql` | Windowed retention of results and errors. | Staged results are retained until applied, canceled, or expired. |
| Change and operation records | application-services | SQLite | `api/internal/app/schema.sql` | Long enough for normal client and job retries. | Carries idempotency key, expected/applied revision, and correction links. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The `system schema` is
the only cross-cutting, non-domain table set. This table is the specification §12.6
persistence entity inventory; `Defined In` names the intended schema file.

| Entity | Owner | Minimum Persisted Information | Defined In |
|---|---|---|---|
| Workspace / profile | workspace-and-profile | Owner, locale/timezone/currency, settings revision, setup progress, active rules, appliances, preferences. | `api/internal/profile/schema.sql` |
| DietRule / preset | dietary-rules | Canonical rule definition, required/preferred status, source/preset version, effective dates. | `api/internal/rules/schema.sql` |
| NutrientDefinition | preparation-methods-and-units | Stable identity, canonical unit, form semantics, allowed conversions, source mappings. | `api/internal/units/schema.sql` |
| Food / Product + revision | food-and-product-catalog | Identity, preparation state, serving mappings, nutrient profile, ingredient/allergen evidence, source provenance. | `api/internal/catalog/schema.sql` |
| SourceArtifact | food-and-product-catalog | Type, source URL/provider ID/text or attachment reference, fetched/entered date, hash, parser version, privacy scope. | `api/internal/catalog/schema.sql` |
| Recipe + revision | recipe-library | Stable identity/status and immutable yield, ingredients, methods, source references. | `api/internal/recipes/schema.sql` |
| MealFamily | recipe-library | Typed slots, compatible options, portion bounds, reviewed generation rules. | `api/internal/recipes/schema.sql` |
| NutritionTarget | nutrition-engine | Bounds, period, contribution scope, enforcement, provenance, effective dates. | `api/internal/nutrition/schema.sql` |
| SupplementSchedule | intake-and-feedback | Product revision, dose quantity, local recurrence, start/end, user confirmation, pause state. | `api/internal/intake/schema.sql` |
| Plan + revision | planning-engine | Date range/timezone, scope, profile reference, solver inputs/version/seed, status. | `api/internal/planning/schema.sql` |
| PlannedOccurrence | planning-engine | Local date, slot, recipe revision or explicit open item, servings, method, lock, batch/source relation. | `api/internal/planning/schema.sql` |
| PreparationTask | planning-engine | Scheduled recipe/batch output, resources, active/elapsed effort, dependent occurrences. | `api/internal/planning/schema.sql` |
| PreparedBatch | inventory-and-batches | Actual recipe revision/yield, remaining amount, preparation time, storage assertions. | `api/internal/inventory/schema.sql` |
| ConsumptionEvent | intake-and-feedback | Actual local/UTC time, pinned content/quantity, source occurrence/batch, snapshot totals, provenance, correction linkage. | `api/internal/intake/schema.sql` |
| PriceObservation | cost-and-price-book | Product/package, amount/currency, retailer/conditions, observed date, validity/freshness, source. | `api/internal/costs/schema.sql` |
| InventoryEvent / assertion | inventory-and-batches | Item/batch, amount/unit, event type, evidence/source, effective time, superseded/correction reference. | `api/internal/inventory/schema.sql` |
| ShoppingPlan / line | shopping-plan | Plan/input revisions, derived quantities, selected packages, overrides, check state, purchase links. | `api/internal/shopping/schema.sql` |
| PurchaseEvent | inventory-and-batches | Bought quantity/packages, actual spend/currency, retailer/date, receipt source, inventory application key. | `api/internal/inventory/schema.sql` |
| Feedback / preference | intake-and-feedback | Explicit reason/action, scope, timestamp, source occurrence, inferred/explicit distinction. | `api/internal/intake/schema.sql` |
| Import / research / export job | provider-and-job-adapters | Input reference, status, dedup key, budget, progress, errors, staged results, applied operation ID. | `api/internal/jobs/schema.sql` |
| Change / operation record | application-services | Actor, request/idempotency key, expected/applied revision, affected IDs, undo/correction linkage. | `api/internal/app/schema.sql` |
| system schema | infrastructure | Migration version and cross-cutting DB bookkeeping. | `api/internal/database/system.sql` |

Use normal relational constraints and indexes for actual access paths. Full event sourcing
is not required: an append-only correction history for important facts plus current
projections is sufficient (specification §12.6). Do not keep one endlessly growing JSON
blob as the only independently editable or queryable store.

### Decimal, Unit, And Money Rules (DOM-03)

- Use decimal or rational arithmetic for domain calculations and serialize decimals as
  canonical strings on the wire and in exports. Persistence and aggregation preserve
  precision; only display rounds.
- A quantity requires unit, dimension, preparation state where relevant, and basis.
  `100 g dry rice` and `100 g cooked rice` are different nutrient bases. `1 scoop` is not a
  mass until mapped for a specific product revision.
- Unknown and explicit zero are distinct in every serializer and form. A blank numeric input
  stays blank/null and never coerces to zero.
- Use ISO currency codes. Store transaction/package money as integer minor units with a
  currency-specific exponent; maintain higher precision internally for allocated portion
  cost. Do not assume every currency has two decimal places, and do not sum across
  currencies without a dated exchange-rate policy (deferred).
- Round package counts upward where appropriate, and round money at documented
  transaction/report boundaries, not every ingredient multiplication.

### Identity And Time (DOM-02)

- Use opaque IDs generated by the chosen stack; a prefix such as `recipe_` is illustrative,
  not a required scheme.
- Every user-owned aggregate belongs to a server-authorized workspace. References from
  plans and history identify the exact revision they use.
- Record creation/update timestamps as UTC instants. Store local dates and IANA timezone
  separately for calendar semantics. A revision number is an integer used for concurrency,
  not a timestamp.
- Deletions normally archive or tombstone referenced records. A source update creates a new
  revision and never mutates a historical snapshot.

### Immutable Revisions And Corrections

Recipes, foods/products, and plans separate stable identity from immutable content
revisions. Historical intake is pinned to the revision and quantity actually recorded, so a
later recipe edit cannot silently change past totals. Corrections are new, linked events
that preserve the original; they are not in-place edits or deletes. This rule is what
specification §12.7 #5 and FIX-07 protect.

## Migrations And Compatibility

Version these independently, per specification §18.5:

| Version | Meaning | Independence Rule |
|---|---|---|
| Database migration version | The schema applied to the local SQLite database. | Upgrades tested against a populated prior database, not only an empty one. Back up before a destructive migration. |
| Export schema version (`daily.recipes`, `daily.workspace`) | The portable envelope format, currently `schemaVersion: 2`. | Changing storage does not require renumbering export formats, and vice versa. |
| Calculation/evaluator version | The algorithm version behind nutrition, planning, cost, and eligibility assessments. | Cached assessments are keyed by evaluator version; a code change invalidates them without a data migration. |

Domain schema files use idempotent bootstrap (`CREATE TABLE IF NOT EXISTS`) and live beside
the code that interprets them. For production migrations that need column drops, renames, or
backfills, add a scenario-specific migration plan here and record the tradeoff in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

Recommended compatibility policy: write the current native version; support the previous
supported native version through an explicit, tested migrator; reject future versions with
actionable copy. Keep the legacy prototype format working through the separate adapter in
[`Import / Export`](#import--export). A permissive JSON parser is not a migration strategy.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Native recipe collection | `daily.recipes`, `schemaVersion: 2`, namespaced records keyed by `(kind, id, revision)`. | transfer-and-documents | Specified (FIX-09); not implemented. |
| Native workspace backup | `daily.workspace`, `schemaVersion: 2`, `scope.kind: workspace_backup`. | transfer-and-documents | Specified; not implemented. |
| Legacy prototype adapter | Single recipe objects, arrays, `{recipes: […]}`; workspace with `schemaVersion: 1`, plan of seven IDs, completed/checked indexes. | transfer-and-documents | Specified (FIX-08); isolated adapter, not a generic JSON loader. |
| Grocery CSV | Human-readable spreadsheet view of the selected shopping plan. | shopping-plan | Specified; CSV is not the canonical backup. |
| Recipe / week PDF | Vector PDFs for recipes and weekly menus with grocery checklist. | transfer-and-documents | Specified; real text/vector output, not a screenshot. |

Rules:

- A complete export includes profile/rules, food/product revisions, recipes/revisions and
  components, plans/occurrences, targets, supplement schedules, prices,
  inventory/batches/events, consumption/corrections, feedback, and referenced source
  metadata supported by the release. Secrets, auth tokens, model credentials, private
  infrastructure URLs, and billing instruments are excluded.
- A source attachment omitted from a text-only export is described as omitted, with its
  metadata preserved. Never label that export a complete attachment backup.
- Round-trip acceptance compares semantic content after normalization, not JSON property
  order. Derived totals and solver caches may export for explanation but are marked derived
  and recomputed on import.
- **R0 import limits** (configurable abuse/UX limits, not permanent domain limits):
  2 MB pasted/file JSON, 200 recipes, 80 ingredient uses per recipe, 30 steps per recipe,
  and 10,000 characters of notes. Later releases support bounded larger backups through
  jobs.
- Import stages and validates before any live write; apply is atomic against an expected
  revision. Full restore creates a recoverable checkpoint and never imports membership,
  credentials, entitlement, or tenant authority.

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Workspace and profile | Workspace/account deletion (commercial release). | Active until deletion; derived caches dropped first. | Account-deletion flow is not implemented; personal use is single-workspace. |
| Recipe identity | User archives or deletes an unreferenced draft. | Archived recipes stop being offered but remain resolvable; referenced revisions are kept. | Permanent deletion policy for referenced revisions is deferred. |
| Consumption events and corrections | Not deletable in place. | Kept for history; a correction adds a linked event. | — |
| Inventory events | Projection is derived; events are append-only. | Events kept; projection can be recomputed. | — |
| Price observations | Superseded, not deleted. | Kept with age; older observations usable with a label. | Freshness window is configurable by source/category and not yet chosen. |
| Jobs and staged results | Applied, canceled, or expired. | Windowed retention; errors retained for support. | Exact window not fixed. |
| Source attachment bytes | Owning draft/revision removal. | Kept while referenced; omitted from text-only exports with a declared omission. | Attachment custody via `document-manager` is optional at P1. |

Derived assessments and search caches are not authoritative and may be dropped and
recomputed at any time. Non-regenerable records — consumption events, purchases,
corrections, and operation records — are protected and must not be treated as cache.

## Privacy Notes

- Food logs, dietary restrictions, allergen evidence, nutrition targets, supplement
  schedules, receipt details, and label images are private personal data. They are private
  to their workspace until a deliberate sharing feature exists; public recipe sharing is
  deferred (specification §18.3–§18.4).
- Send a model only the task-relevant subset, and prefer a local opaque reference over an
  account identifier in a prompt. Disclose configured provider behavior in settings.
- Logs contain operation IDs, timings, counts, error codes, and safe revision references.
  They exclude raw recipe text, receipt contents, medical notes, access tokens, and
  attachment URLs by default. Debug capture requires deliberate bounded activation and a
  retention policy.
- Exports and generated PDFs contain no secrets and no tenant authority. Import/restore
  cannot install credentials or grant permissions.
- Export of owned domain data is available without a paid upgrade. Deployment-specific
  legal wording for commercial release still requires appropriate review.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — module boundaries and state ownership
- [`FLOWS.md`](FLOWS.md) — lifecycle states that govern these records
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../reference/product-specification.md`](../reference/product-specification.md) — §12, §15, §18, §20.8–§20.9
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — blob and clock boundaries
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy and security posture
