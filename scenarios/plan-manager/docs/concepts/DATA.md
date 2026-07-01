# Data — Plan Manager

## Purpose Of This Document

The data ownership and storage map for plan-manager. The *shape* of the records
lives in [`PLAN-MODEL.md`](PLAN-MODEL.md); this document covers where they live,
who owns them, retention, migration, and privacy. The defining decision: storage
is rooted at the scenario-**independent** `~/.vrooli` home store, not a
scenario-private database.

## Storage Overview

- **Engine:** SQLite via `api-core/storage`.
- **Canonical root:** the scenario-scoped `api-core/storage` location under the
  `~/.vrooli` home store (`ClassData`, resolved by `storage.NewResolver` +
  `storage.ScenarioNamespace("plan-manager")` in `api/main.go`), **not** a
  per-scenario repo `data/` directory and **not** the flat `~/.vrooli/plans` dir.
  The store is process-independent (readable with the server down) and
  variant-aware (shadow engagements get their own namespaced DB for free).
- **Plan-source resolver (compatibility mechanism):** the `plans` domain owns a
  resolver (`internal/plans/resolver.go`) that treats the hygiene-blessed
  **fallback** locations — `~/.vrooli/plans`, the project `plans/` dir, and
  `docs/plans/` — as valid read/import locations. `ImportPlan` adopts a markdown
  plan from a fallback source into the structured model; `MigratePlan` ensures a
  resolved plan resides in the canonical store. The fallback sources are **never
  mutated destructively** without an explicit step. This resolver — not a shared
  raw directory — is how plan-manager coexists with legacy markdown plan data and
  how the future consumer-inversion (OT-P2-002) delegates plan-location logic here.
- **Why:** plans must remain durable outside the plan-manager server process.
  plan-manager owns the schema, validation, and logic; the home store is the
  persistence substrate.
- **Consequence:** reads do not require the API process; writes go through
  plan-manager (the schema/lifecycle authority), but the on-disk store is
  process-independent and concurrency-safe for multiple readers.
- **Rendered markdown mirror:** every plan may carry `mirror` metadata in its
  structured document. The mirror file itself is a durable projection under the
  repo-contract runtime-home `plans` entry. The mirror is not canonical:
  when it is missing or stale, Plan Manager regenerates it from SQLite and
  updates the metadata. If a mirror write fails after SQLite commits, the plan
  remains saved with `mirror.status=write_failed` and can be repaired later.

## Data Ownership

| Data | Owning domain | Notes |
|---|---|---|
| Plans, phases, references, supersession edges, content hashes, rendered-mirror metadata | `plans` | The structured-record SSOT. Mirror files are derived projections. |
| Authoring-session progression + validation findings | `authoring` | Transient; the produced plan is owned by `plans`. |
| Run↔plan linkage, canonical handoff records (carry a log summary + entries), velocity series | `execution` | Decisions/findings are no longer stored here — they live in `log`; execution reads a compact summary through the `LogLedger` seam. |
| Reference resolutions, staleness factors, validation results | `validation` | Derived; never the source of truth for code itself. |
| Execution-log entries (decisions, candidate findings, bug reports, records, notes) + downstream sync state | `log` | One `log_entries` table; findings are explicitly unvalidated/candidate; bug reports & records carry `sync_status` for internal downstream forwarding. |
| Health probe state | `health` | No persisted product data. |

Data plan-manager deliberately does **not** own: agent transcripts and the prose
final handoff (orchestration layer); confirmed bugs (operator triage / the issue
backlog); project-level validation results for resources/packages (consumed from
test-genie / scenario-validation).

## Schema Map

Tables are domain-owned (`api/internal/<domain>/schema.sql`), all rooted at the
home store. As built:

- `plans` (owned by `plans`) — plan records: first-class queryable columns
  (id, slug, title, status, content_hash, created_at, updated_at) alongside a
  `document` JSON column carrying the rest of the structured record. The document
  holds every non-queryable structured field: the Overview prose (purpose,
  problem_statement, target_outcome, scope, non_goals, assumptions), the Work
  Posture (work_posture/source/detail — autofilled, see
  [PLAN-MODEL.md](PLAN-MODEL.md#work-posture--greenfieldbrownfield)), the
  Execution Model (relevant_context[], references[], constraints,
  prohibited_approaches, technical_approach), the Validation Model
  (regression_anchor, validation_strategy, final_validation_commands[],
  definition_of_done), the ordered **phases[]** (each with affected_areas[],
  steps[], expected_outputs[], validation, acceptance, risks_hazards[],
  handoff_notes, …), and the governance fields (supersedes/superseded_by,
  import_provenance, preserved_legacy_sections[]). Because storage is a single
  JSON document, **adding a structured field flows through automatically** — the
  `planDocument` struct in `sqlite.go` is the one place a new field must be added
  to its `Save`/scan mapping. Phases and references persist inside the document
  (they always load with their plan and are never queried across plans), which
  keeps round-trips deterministic and avoids the SQLite pool=1 nested-query
  deadlock. The ownership contract is unchanged.
- `plan_edges` (owned by `plans`) — supersession/dependency edges between plans
  (queried across plans by `GetGraph`, so a first-class table).
- `authoring_sessions` (owned by `authoring`) — transient guided-wizard session
  state (sections[] + current pointer + finalized + produced plan_id), as a JSON
  document; the produced plan is owned by `plans`.
- `executions`, `handoffs`, `velocity_points` (owned by `execution`)
  — run↔plan linkage, canonical handoff records, and the per-plan/run velocity
  series. Decisions and candidate findings are **not** stored here — they live in
  the `log` domain; execution reads a compact summary through the `LogLedger` seam.
- `log_entries` (owned by `log`, `api/internal/planlog/schema.sql`) — the single
  execution-log ledger: typed entries (decisions, candidate findings, bug reports,
  records, notes) scoped to a plan/execution/phase, each carrying triage, evidence,
  downstream sync state, and idempotency/attribution metadata. Two partial UNIQUE
  indexes enforce dedup: `(plan_id, idempotency_key)` for keyed retries, and
  `(plan_id, execution_id, attribution_run_id, type, normalized title)` for keyless
  attribution dedup.
- `validation` persists **no** tables in v1: reference resolution, staleness, and
  baseline outcomes are computed on demand and returned (execution calls
  `RunValidation`/`ComputeStaleness` when it needs the just-in-time signal).
  A `validation_results` table is a future caching concern, not v1.

Exact columns live alongside the domain code (`api/internal/<domain>/schema.sql`);
this map is the ownership contract.

## Migrations And Compatibility

- Schema is created idempotently from the embedded `schema.sql` per domain on boot.
- Because the store is shared with legacy markdown plan data, the first
  migration concern is **adopting / coexisting with** the existing `~/.vrooli/plans`
  file store: plan-manager reads the existing plans and writes the structured
  superset. No destructive migration of legacy plans without an explicit,
  reversible step.
- Greenfield within plan-manager: no compatibility shims between plan-manager's
  own schema versions during initial development — fail forward, no alias tables.

## Import / Export

- **Import:** adopt existing markdown plans via `plan-manager plans import`
  into the structured model; references are parsed from `[CODE:]`/`[REQ:]`.
- **Reconcile/adopt legacy:** `ReconcilePlans` can dry-run or execute a bulk pass
  over the runtime-home `plans`, repo `docs/plans`, and repo `plans` fallback
  locations. It reports each source as imported, already canonical, duplicate,
  parse failed, conflict, or source untouched; it never deletes or overwrites the
  legacy source file.
- **Render/export:** render any plan to a markdown view (the human-readable
  projection). `RenderMarkdown` returns the mirror contents when fresh, repairs
  the mirror from SQLite when missing or stale, and returns the resolved plan
  metadata alongside the markdown/mirror metadata for caller provenance.
- The structured record is canonical; markdown is always a derived projection.
  Editing the mirror file by hand never updates SQLite.
- Canonical plan records carry `workspace_id`/`workspace_root`. Rendered mirror
  index metadata also carries workspace identity so root fallback clients can
  filter safely and fail closed for scoped reads when provenance is missing.

## Retention And Deletion

- Plans are retained until archived or deleted; archival is a soft state (kept,
  hidden by default) managed by `plan-manager plans archive`.
- Velocity points and handoff records are retained as historical signal (small,
  append-only); pruning is a future tuning concern, not v1.
- Candidate findings are retained until an operator triages (promote/dismiss).

## Privacy Notes

- Plan data is low-sensitivity: titles, prose, and **code locations / small
  snippets** referenced via `[CODE:]`. It must **not** contain credentials or
  secrets — references point at code, they do not embed secret material.
- Data is local to the Vrooli runtime under `~/.vrooli`; no external transmission
  by plan-manager. Velocity emitted to meta-optimization is local, aggregate
  signal (counts/times), not plan content.

## Cross-References

- [`PLAN-MODEL.md`](PLAN-MODEL.md) — record shapes
- [`DOMAINS.md`](DOMAINS.md) — which domain owns each table
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — the home store + composed substrate
- [`../reference/configuration.md`](../reference/configuration.md) — storage configuration
- [`../../PRD.md`](../../PRD.md) — operational targets
