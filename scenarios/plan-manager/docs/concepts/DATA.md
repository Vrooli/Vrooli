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
  raw directory — is how plan-manager coexists with the legacy `vrooli plans`
  data and how the future consumer-inversion (OT-P2-002) delegates plan-location
  logic here.
- **Why:** plans must stay readable when the plan-manager server is down, and the
  thin `vrooli plans` CLI must be able to read them. plan-manager owns the schema,
  validation, and logic; the home store is the persistence substrate.
- **Consequence:** reads do not require the API process; writes go through
  plan-manager (the schema/lifecycle authority), but the on-disk store is
  process-independent and concurrency-safe for multiple readers.

## Data Ownership

| Data | Owning domain | Notes |
|---|---|---|
| Plans, phases, references, supersession edges, content hashes | `plans` | The structured-record SSOT. |
| Authoring-session progression + validation findings | `authoring` | Transient; the produced plan is owned by `plans`. |
| Run↔plan linkage, captured decisions/findings, candidate findings, canonical handoff records, velocity series | `execution` | Candidate findings are explicitly unvalidated. |
| Reference resolutions, staleness factors, validation results | `validation` | Derived; never the source of truth for code itself. |
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
  `document` JSON column carrying the rest of the structured record — purpose/
  scope/constraints/non_goals/definition_of_done plus the ordered **phases[]**,
  **references[]**, and the regression anchor. Phases and references persist
  inside the document (they always load with their plan and are never queried
  across plans), which keeps round-trips deterministic and avoids the SQLite
  pool=1 nested-query deadlock. The ownership contract is unchanged.
- `plan_edges` (owned by `plans`) — supersession/dependency edges between plans
  (queried across plans by `GetGraph`, so a first-class table).
- `authoring_sessions` (owned by `authoring`) — transient guided-wizard session
  state (sections[] + current pointer + finalized + produced plan_id), as a JSON
  document; the produced plan is owned by `plans`.
- `executions`, `handoffs`, `findings`, `velocity_points` (owned by `execution`)
  — run↔plan linkage, canonical handoff records, candidate (unvalidated)
  findings with triage state, and the per-plan/run velocity series.
- `validation` persists **no** tables in v1: reference resolution, staleness, and
  baseline outcomes are computed on demand and returned (execution calls
  `RunValidation`/`ComputeStaleness` when it needs the just-in-time signal).
  A `validation_results` table is a future caching concern, not v1.

Exact columns live alongside the domain code (`api/internal/<domain>/schema.sql`);
this map is the ownership contract.

## Migrations And Compatibility

- Schema is created idempotently from the embedded `schema.sql` per domain on boot.
- Because the store is shared with the legacy `vrooli plans` data, the first
  migration concern is **adopting / coexisting with** the existing `~/.vrooli/plans`
  file store: plan-manager reads the existing plans and writes the structured
  superset. No destructive migration of legacy plans without an explicit,
  reversible step.
- Greenfield within plan-manager: no compatibility shims between plan-manager's
  own schema versions during initial development — fail forward, no alias tables.

## Import / Export

- **Import:** adopt existing markdown plans (the current `vrooli plans import`
  path) into the structured model; references are parsed from `[CODE:]`/`[REQ:]`.
- **Export:** render any plan to a markdown view (the human-readable projection);
  optionally export to a repo path (the existing `vrooli plans export` behavior).
- The structured record is canonical; markdown is always a derived projection.

## Retention And Deletion

- Plans are retained until archived or deleted; archival is a soft state (kept,
  hidden by default) mirroring today's `vrooli plans archive`.
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
