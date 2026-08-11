# Data — Program Runtime

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
The lifecycle sets `SQLITE_PATH` through `.vrooli/service.json`, and
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

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Binding-resolution snapshots | bindings | SQLite | `api/internal/bindings/schema.sql` | Superseded on regeneration; keep the newest per descriptor digest. | Cache and provenance, never authority. The descriptor image and manifests are the truth; this records what resolved and what was refused. |
| Refusal reasons | bindings | SQLite | `api/internal/bindings/schema.sql` | 30 days. | Why a callable was withheld — needed to explain an absence without re-deriving it. |
| Sessions and grants | sessions | SQLite | `api/internal/sessions/schema.sql` | Deleted on reclamation; reason retained per the row below. | Grants are the authorization record for destructive-effect calls and must outlive the kernel process. |
| Reclamation reasons | sessions | SQLite | `api/internal/sessions/schema.sql` | 30 days. | A session vanishing without a stated reason is indistinguishable from a crash; keep the reason past the session. |
| Program submissions and source | programs | SQLite | `api/internal/programs/schema.sql` | 90 days, then source-only. | The corpus that makes recurring failure shapes mechanically derivable (`PRT-P1-006`). |
| Program results and failure detail | programs | SQLite | `api/internal/programs/schema.sql` | 90 days. | Failure detail is the friction evidence; results are bounded at write, never full materialization. |
| Kernel variable state | sessions | **process memory — not persisted** | n/a | Dies with the kernel process. | Deliberate. See "Why kernel state is not persisted" below. |
| Event outbox | telemetry | SQLite | `api/internal/telemetry/schema.sql` | Deleted on successful emit; 7-day dead-letter. | An outbox, not a store. Analysis lives in agent-manager and meta-optimization-manager. |

### Why kernel state is not persisted

A handle is a **live reference into a running interpreter**, not a value.
Persisting it would mean serializing what it points at — which is exactly
the materialization this scenario exists to avoid, paid on every
submission instead of never. A session that outlives its kernel therefore
loses its variables by design, and `sessions` reports that plainly rather
than silently rehydrating a stale copy.

The consequence is intentional and load-bearing: **handles are cheap
because they are ephemeral.** A durable equivalent would be a different
capability with a different cost model, and it is not this one.

`actspace` owns no data at all: the denominator is a document under
`docs/spaces/`, and the numerator is computed live and never stored —
matching the rule every sibling projection already follows.

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `refusals` | bindings | `api/internal/bindings/schema.sql` | binding bridge; retention worker |
| `sessions`, `session_grants`, `reclamation_reasons` | sessions | `api/internal/sessions/schema.sql` | sessions repository/service; `programs` reads session identity at dispatch |
| `programs` | programs | `api/internal/programs/schema.sql` | programs repository/service; `telemetry` reads to build failure events |
| `event_outbox` | telemetry | `api/internal/telemetry/schema.sql` | telemetry emitter only |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

Cross-domain reads above are **read-only and one-directional**, matching
the acyclic read graph in [`DOMAINS.md`](DOMAINS.md). No domain writes
another domain's tables.

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships the `notes` domain as a worked CRUD slice with a
binary attachment-upload exception, showing how a real domain owns its
tables, metadata, and opaque blob bytes. Copy its shape, then remove it.

Its Data Ownership rows:

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Notes | notes | SQLite | `api/internal/notes/schema.sql` | Until deleted by future product behavior | Template reference data; remove with notes domain. |
| Attachment metadata | notes | SQLite | `api/internal/notes/schema.sql` | Until parent note or attachment is deleted by future product behavior | Metadata only; bytes are stored through BlobStore. |
| Attachment bytes | notes | Filesystem BlobStore by default | BlobStore implementation in notes handler module | Same lifecycle as metadata | Opaque bytes stay outside proto payloads. |

Its Schema Map row:

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| notes tables | notes | `api/internal/notes/schema.sql` | notes repository/service/handlers |

Its Retention And Deletion row:

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Template notes data | Domain removal or future product delete behavior | Local development data only | Real scenarios must define product-specific deletion semantics. |
<!-- EXAMPLE-DOMAIN:notes END -->

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
| Kernel variable state | Kernel process exit (explicit close, reclamation, or crash). | None — never persisted. | None. This is the designed behavior, not a gap. |
| Sessions and grants | Session close or idle reclamation. | Deleted with the session. | Reclamation ceilings are `PRT-P1-005`; until it lands, sessions are closed explicitly only. |
| Reclamation reasons | Age. | 30 days. | None. |
| Program submissions, source, results, failure detail | Age. | 90 days. | `api/internal/retention/worker.go` prunes old rows on a dedicated SQLite handle; source and bounded results are retained together within the declared window. |
| Binding refusals | Age. | 30 days. | `api/internal/retention/worker.go` prunes old refusal evidence. |
| Event outbox rows | Successful emit. | Deleted on emit; 7-day dead-letter. | None. |

**Program source is user-authored content, not a regenerable artifact.**
Deleting it destroys the friction corpus that `PRT-P1-006` and
`PRT-P2-002` depend on, and it cannot be reconstructed. Declare it
accordingly before the corpus has value worth losing.

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
