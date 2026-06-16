# Data — Image Tools

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

image-tools has two distinct stores with a hard split:

- **Scenario metadata → embedded SQLite** through `modernc.org/sqlite`.
  The lifecycle sets `SQLITE_PATH` through `.vrooli/service.json`, and the
  API applies schemas on startup through `api-core/database`. SQLite holds
  jobs, recipes, model-registry state, watch-folder/automation config,
  measures, and usage/cost records. None of these are image bytes.
- **Image binaries → api-core storage/blobstore**, stored *outside the
  repo* by default. The save location is overridable per request, and
  outputs are user-owned (copyable, movable, deletable anywhere). The
  `storage` domain owns the consuming seam; api-core owns the
  implementation. Opaque bytes never enter SQLite or proto payloads — the
  metadata references blobs by handle.

This separation is deliberate: SQLite is the queryable proto-typed
metadata plane; blobstore is the bulk binary plane. External storage
resources beyond api-core blobstore should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Job records (id, op, status, progress, ETA, tier, error) | jobs | SQLite | `api/internal/jobs/schema.sql` | Until completion + configured TTL, then pruned | Server-owned; survive client disconnect. |
| Recipe definitions (op-stack graphs) | recipes | SQLite | `api/internal/recipes/schema.sql` | Until user deletes | Unified UI op-stack / CLI pipeline representation. |
| Model registry state (entries, enabled flag, install/local-path) | models | SQLite | `api/internal/models/schema.sql` | Until model removed | Declarative entries; weights live on disk via blobstore/local path, not in SQLite. |
| Watch-folder / automation config | automation | SQLite | `api/internal/automation/schema.sql` | Until user deletes config | Debounce, output routing, callback URLs. |
| Webhook callback delivery state | automation | SQLite | `api/internal/automation/schema.sql` | Until completion + retry window | Best-effort POST + retry; not an event bus. |
| Measure samples + aggregates | measures | SQLite | `api/internal/measures/schema.sql` | Rolling window per measures policy | Latency p50/p95, throughput, queue-wait, fallback-tier usage, VRAM headroom. |
| BYOK usage / cost records | models / backends | SQLite | usage schema (audio-tools cost-tracking pattern) | Rolling window | Cost estimate before op; keys are secrets, never stored with usage rows. |
| Image inputs and outputs (bytes) | storage | api-core blobstore (outside repo) | BlobStore implementation behind the `storage` seam | Configurable TTL / user control; outputs user-owned | Overridable save location per request; opaque bytes, never in proto or SQLite. |
| Blob references + output-ownership metadata | storage | SQLite | `api/internal/storage/schema.sql` | Same lifecycle as referenced blob | Handle/id, owner, save-location resolution. |
| Notes + attachment metadata (**template example, remove**) | notes | SQLite | `api/internal/notes/schema.sql` | Until deleted | Template reference data; remove with notes domain. |
| Attachment bytes (**template example, remove**) | notes | Filesystem BlobStore by default | BlobStore in notes handler module | Same lifecycle as metadata | Opaque bytes outside proto payloads. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| jobs tables | jobs | `api/internal/jobs/schema.sql` | jobs repository/queue/handlers, CLI wait, UI progress |
| recipes tables | recipes | `api/internal/recipes/schema.sql` | recipes repository/replay/handlers |
| model registry tables | models | `api/internal/models/schema.sql` | models repository/selector/handlers, CLI, Settings UI |
| automation tables (watch-folder, callbacks) | automation | `api/internal/automation/schema.sql` | automation repository/handlers, watcher |
| measures tables | measures | `api/internal/measures/schema.sql` | measures collector/aggregator, manifest measure blocks |
| BYOK usage/cost tables | models / backends | usage schema | cost estimator, fallback messaging |
| storage reference tables | storage | `api/internal/storage/schema.sql` | storage seam, all operation domains |
| image blobs | storage | api-core blobstore (object store, outside repo) | every operation domain via the `storage` seam |
| notes tables (**template, remove**) | notes | `api/internal/notes/schema.sql` | notes repository/service/handlers |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

## Migrations And Compatibility

The generated template uses idempotent schema bootstrap. Domain schema
files should use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them.

Two compatibility concerns are scenario-specific:

- **Model registry schema is proto-validated.** Registry entries follow a
  schema/proto contract (id/name, operations served, hardware
  requirements, capability/license labels, checksum/source). Registry
  *evolution* (new fields, new capability labels, new quant tiers per
  OT-P2-004) is an additive, backward-compatible proto change — never a
  destructive recreate. Existing-table column changes are one-shot
  migrations, never a DB recreate.
- **Blob references must outlive schema changes.** Because image bytes live
  in api-core blobstore and SQLite only holds handles, a metadata
  migration must never orphan or drop live blob references; backfills
  preserve handle→blob mappings.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Image input upload | multipart/form-data (PNG/JPEG/WebP/AVIF/HEIC/GIF/TIFF/BMP, SVG→raster import) | storage / operation domains | Planned (P0). |
| Image output download/save | original or converted raster format; user-owned, overridable save location | storage | Planned (P0). |
| Recipe export/import | proto-typed op-stack graph (portable across UI and CLI) | recipes | Planned (P1). |
| Model registry entry import | declarative entry (custom/fine-tuned local model path) | models | Planned (P0, OT-P0-007). |
| Analysis results | structured proto (OCR text, embeddings, detection boxes, hashes, QR payloads) | analysis | Planned (P0/P1). |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Image inputs (uploads) | Job completion or configured TTL | Not retained beyond configured TTL / user control | TTL policy to be finalized at implementation. |
| Image outputs | User delete / move / copy | User-owned; retained until user removes | None — outputs are user-owned by design. |
| Job records | TTL after terminal state | Pruned after retention window | Window value to be set per profile. |
| Measure samples | Rolling window | Aggregated then pruned | Window value to be set per measures policy. |
| BYOK usage/cost records | Rolling window | Retained for cost reporting; keys never stored | None — keys handled as secrets only. |
| Recipes / model registry / automation config | User delete | Retained until user removes | None. |
| Template notes data (**remove**) | Domain removal | Local development data only | Replace with the deletion semantics above. |

## Privacy Notes

image-tools handles user images, which can carry personal and location
data, and BYOK provider keys. The privacy posture:

- **EXIF/GPS stripped by default.** Metadata read is available, but the
  GPS-strip op is on by default in the deterministic metadata path; users
  opt in to retaining location/EXIF.
- **Uploads are not retained beyond configured TTL / user control.** Input
  images are transient; outputs are user-owned and live only where the
  user saves them.
- **BYOK keys are secrets.** Provider keys are handled as secrets, never
  stored alongside images or in usage rows; only derived cost/usage
  metadata is persisted.
- **NSFW/safety classification produces metadata, not a hard block.**
  Auto-scan of AI output is configurable; classification results are
  metadata and do not, by default, block delivery.
- **No face recognition / identity matching.** Anonymous face *detection*
  (count + boxes) is in scope; no identity data is derived or stored.
- **Ingestion safety.** Decompression-bomb / oversized-image guards run at
  the upload boundary before any processing.

If implementation expands what is persisted, update this document and
[`../internal/SECURITY.md`](../internal/SECURITY.md) first.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
