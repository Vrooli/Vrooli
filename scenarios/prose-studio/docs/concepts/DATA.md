# Data — Prose Studio

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

The template default is embedded SQLite through `modernc.org/sqlite`.
The database path is resolved from the scenario id by `api-core/storage`, and
the API applies schemas on startup through `api-core/database`.

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

Each domain owns its own tables and is the source of truth for its data.
The `health` domain owns no product data — it only probes configured
database reachability. As you build real domains, add a row per data
shape they persist: name it, name the owning domain, the storage backend,
the schema file that is the source of truth, the retention rule, and any
remarks. Keep blob/opaque bytes outside proto payloads, behind a seam
such as BlobStore.

**One rule dominates this scenario's data model: for a consumer-declared record,
the file is the source of truth and the row is a projection.** The row carries
`authority: file`, checked on every write path. An API write to it is refused
naming the file. Its version is the content hash of the file, not a counter.
Deleting the file marks the record `unregistered` rather than deleting it, so a
candidate generated six months ago still resolves its provenance. There is no
round-trip sync, and promotion from operator-authored to declared is one-way.

The second rule: **the convergence graph is append-only.** No round, candidate,
or selection event is ever mutated. Rerolling adds; rejecting adds; committing
adds. This is what makes "what did we not choose, and what did it measure?"
answerable later, and it is why cost attributes across a whole set rather than
to the survivor.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| `style`, `style_version` | styles | SQLite | `api/internal/styles/schema.sql`, **or the declaration file when `authority: file`** | Version frozen on first reference by a committed output; never deleted while referenced. | Exemplars, directives, anti-patterns, lexicon, targets, axis defaults. Single-parent `extends`. |
| `profile`, `profile_version` | profiles | SQLite | `api/internal/profiles/schema.sql`, or the declaration file | Same freeze-on-reference rule. | Sampler kind + params, constraints, selection policy, measurement tiers, context policy, budget, gateway role. |
| `axis_space` | profiles (P1) | SQLite | `api/internal/profiles/schema.sql` | Versioned; frozen on reference. | Generalises the axis/variant/weight/constraint shape already proven in `landing-page-business-suite/config/variant_space.json`. |
| `declaration` | declarations | SQLite | The file on disk — always | Row persists as `unregistered` after file deletion. | Registration state only: path, namespaced key, content hash, status, parse error, registered-at. |
| `round` | generation | SQLite | `api/internal/generation/schema.sql` | Retained with its session. | Strategy, params, effective sampling key, resolved model, declared context window, whole-set cost. |
| `candidate` | generation | SQLite | `api/internal/generation/schema.sql` | Append-only; ineligible candidates retained, never dropped. | Text, provenance, measurements at birth, eligibility + named reason, machine-generation disclosure constant. |
| `verbalized_hint` | generation | SQLite | `api/internal/generation/schema.sql` | With its candidate. | Ordinal within its own round, stored with a constant `calibrated: false`. **Structurally barred from selection by static check.** |
| `selection_event` | selection | SQLite | `api/internal/selection/schema.sql` | Permanent; written from day one and read by nothing in v1. | Chosen candidate, considered candidate ids, measurements snapshotted at choice time, reserved `outcome_ref`. |
| `session` | sessions | SQLite | `api/internal/sessions/schema.sql` | Retained; abandoned sessions kept for provenance. | Append-only DAG root; per-session budget ceiling. |
| `document`, `section` | documents | SQLite | `api/internal/documents/schema.sql` | Retained. | Outline is an ordinary candidate. Section profile resolves by inheritance. |
| `context_snapshot` | documents | SQLite | `api/internal/documents/schema.sql` | With its section. | Prior refs carried, following intents declared, what was summarized. Makes a reroll's blast radius answerable. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `styles`, `style_versions` | styles | `api/internal/styles/schema.sql` | styles, profiles, generation, measurement (targets) |
| `profiles`, `profile_versions`, `axis_spaces` | profiles | `api/internal/profiles/schema.sql` | profiles, generation, sessions, documents |
| `declarations` | declarations | `api/internal/declarations/schema.sql` | declarations only; drives the styles/profiles write paths |
| `rounds`, `candidates`, `verbalized_hints` | generation | `api/internal/generation/schema.sql` | generation, measurement, selection, sessions |
| `selection_events` | selection | `api/internal/selection/schema.sql` | selection only |
| `sessions` | sessions | `api/internal/sessions/schema.sql` | sessions, documents |
| `documents`, `sections`, `context_snapshots` | documents | `api/internal/documents/schema.sql` | documents |
| `.vrooli/prose-studio/*.json` (in a **consuming** scenario) | that consumer | The consumer's own repository | declarations, read-only; never written by this scenario |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

### Schema decisions that are not retrofittable

Three are deliberately P0 even though the behaviour they support lands later,
because adding them afterwards is a migration rather than a feature:

1. **`authority` on a projected record.** Retrofitting file-authority onto rows
   that were API-authored means guessing which is correct per row.
2. **Cost attributed across a set.** If cost is first written against the
   committed candidate alone, the discarded candidates' cost is gone and no
   later query can reconstruct what a round actually cost.
3. **`calibrated: false` on the verbalized hint, and `outcome_ref` on the
   selection event.** The first prevents a future reader from mistaking an
   ordinal for a probability; the second is the single field that keeps
   outcome-driven scoring possible without touching history.

## Migrations And Compatibility

The generated template uses idempotent schema bootstrap. Domain schema
files should use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| None yet. | n/a | n/a | Add when product requirements include import/export. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| _(your data)_ | What removes it. | How long it is kept. | Real scenarios must define product-specific deletion semantics. |

## Privacy Notes

Generated template data is local development data. If a scenario stores
personal, regulated, customer, financial, or sensitive business data,
update this document and [`../internal/SECURITY.md`](../internal/SECURITY.md)
before implementation expands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
