# Data — Scenario to Android

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
| Generated-project records | projects | SQLite | `api/internal/projects/schema.sql` | Until the project is regenerated or deleted. | Records source scenario, revision, shell template, and bundle identity so generation is reproducible and comparable. |
| Capability snapshots | targets | SQLite | `api/internal/targets/schema.sql` | Rolling window per target; latest always retained. | Records what was *probed*, never what a target kind usually has. |
| AVD records | targets | SQLite | `api/internal/targets/schema.sql` | Until the AVD is torn down. | Emulator instances created for a validation run; not a durable device. |
| Build records | builds | SQLite | `api/internal/builds/schema.sql` | Governed retention; builds referenced by a release are pinned. | Includes the target-API assertion result and observed value. |
| Build artifacts (APK, AAB) | builds | Filesystem, producer-owned | Referenced by checksum from the build record | Same lifecycle as the build, subject to retention. | **Bytes never enter the database or a consumer payload.** Consumers receive `common/v1` `EvidenceRef` only. |
| Journey plans | journeys | SQLite | `api/internal/journeys/schema.sql` | Until superseded; versioned. | The registered conformance capability plan and its chapter definitions. |
| Run records and chapters | journeys | SQLite | `api/internal/journeys/schema.sql` | Governed retention; release-relevant runs pinned. | Chapters carry purpose, bounded policies, expected/observed, and capture references. |
| Capture artifacts (frames, recordings) | journeys | Filesystem, producer-owned | Referenced by checksum from the run record | Same lifecycle as the run. | **Highest-risk data in this scenario** — see Retention And Deletion. |
| Matrix run selections | releases | SQLite | `api/internal/releases/schema.sql` | Immutable once started; retained for compare. | A rerun creates a new run; prior evidence is never overwritten. |
| Gate verdicts | releases | SQLite | `api/internal/releases/schema.sql` | Retained alongside the run. | The composed fail-closed decision and its contributing cells. |
| Channel availability and upload receipts | distribution | SQLite | `api/internal/distribution/schema.sql` | Receipts retained for audit; availability is a rolling probe. | Receipts record upload identity, never credentials. |
| Rung state and probe results | readiness | SQLite | `api/internal/readiness/schema.sql` | Latest per rung plus history for trend. | Account state is probed, never asserted from a stored flag. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| Generated-project records | projects | `api/internal/projects/schema.sql` | projects repository/service/handlers; the Builder |
| Capability snapshots, AVD records | targets | `api/internal/targets/schema.sql` | targets repository/service/handlers; the spine `Prober` |
| Build records, assertion results | builds | `api/internal/builds/schema.sql` | builds repository/service/handlers; the spine `Builder` |
| Journey plans, runs, chapters | journeys | `api/internal/journeys/schema.sql` | journeys repository/service/handlers; the spine `Driver` |
| Matrix run selections, gate verdicts | releases | `api/internal/releases/schema.sql` | releases repository/service/handlers; verdict emission |
| Channel availability, upload receipts | distribution | `api/internal/distribution/schema.sql` | distribution repository/service/handlers; the spine `Distributor` |
| Rung state, probe results | readiness | `api/internal/readiness/schema.sql` | readiness repository/service/handlers; CLI and UI walkthrough |
| Build artifacts, capture artifacts | builds, journeys | Filesystem, producer-owned; referenced by checksum | Never a table. Consumers receive `common/v1` `EvidenceRef` only. |
| Signing identity | — | **Not stored here.** Held by `secrets-manager`, referenced by identity. | builds, at invocation time only |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

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
| Capture artifacts (screen frames, recordings) | Run retention expiry, or explicit owner deletion. | Governed retention; runs referenced by a release gate are pinned and exempt until the release is retired. | **Highest-risk data in this scenario.** A frame captured from a physical phone can contain messages, tokens, banking detail, or 2FA codes. `device-control` owns the redaction policy; this ramp must verify redaction status before a capture is referenced in a verdict, and must not ship a physical-device journey before that policy exists. |
| Build artifacts (APK, AAB) | Build retention expiry, or explicit owner deletion. | Retained while any release gate references the build; pinned for released versions so an update journey can install version N. | None, provided the update-migration journey pins the prior artifact rather than rebuilding it. |
| Run records and chapters | Run retention expiry. | Governed retention; release-relevant runs pinned. | None. |
| Matrix run selections and gate verdicts | Retained with the run. | Immutable; a rerun creates a new run rather than mutating a prior one. | None — immutability is a spine invariant, not a local choice. |
| Upload receipts | Never deleted by product behavior. | Long-lived, governed retention. | Receipts are the record of what was published and by whom; truncating them silently would defeat their purpose. |
| Capability snapshots | Superseded by the next probe; history trimmed to a rolling window. | Latest snapshot per target always retained. | None. |
| AVD records | Torn down at the end of a validation run. | Not durable; an orphaned AVD is a leak, not a record. | Teardown must survive an aborted run. |
| Generated-project records | Regeneration, or explicit owner deletion. | Kept until superseded. | None. |
| Rung state | Superseded by the next probe. | Latest per rung plus history. | Account state must always be re-probed rather than trusted from storage, so a revoked account is never reported as ready. |
| Signing material | **Not stored here.** | Held only by `secrets-manager`. | None by construction — a key that never enters this scenario cannot leak from it. |

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
