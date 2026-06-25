# Data — Plan Manager

## Purpose Of This Document

The data ownership and storage map for plan-manager. The *shape* of the records
lives in [`PLAN-MODEL.md`](PLAN-MODEL.md); this document covers where they live,
who owns them, retention, migration, and privacy. The defining decision: storage
is rooted at the scenario-**independent** `~/.vrooli` home store, not a
scenario-private database.

## Storage Overview

- **Engine:** SQLite via `api-core/storage`.
- **Root:** the shared `~/.vrooli` home store (the same durable location the
  existing `vrooli plans` CLI uses), **not** a per-scenario DB directory.
- **Why:** plans must stay readable when the plan-manager server is down, and the
  `vrooli plans` CLI must be able to read them as a thin client. plan-manager owns
  the schema, validation, and logic; the home store is the persistence substrate.
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
home store:

- `plans` — plan records (id, slug, title, status, content_hash, timestamps, global sections).
- `phases` — phase records FK→plan (order, intent, acceptance, status, baseline_scope).
- `plan_references` — `[CODE:]`/`[REQ:]` locators FK→plan/phase, with resolution + staleness state.
- `plan_edges` — supersession/dependency edges between plans.
- `executions` — run↔plan linkage and execution state.
- `handoffs` — canonical structured handoff records FK→execution.
- `findings` — candidate (unvalidated) findings FK→execution, with triage state.
- `velocity_points` — per-plan/run time/tokens/iterations series.
- `validation_results` — validation/baseline outcomes FK→plan/phase.

Exact columns are defined alongside the domain code when implemented; this map is
the ownership contract.

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
