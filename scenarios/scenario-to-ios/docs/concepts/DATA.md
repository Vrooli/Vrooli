# Data — Scenario to iOS

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
| Generated-project records | projects | SQLite | `api/internal/projects/schema.sql` | Until the project is regenerated or deleted. | Records source scenario, revision, shell template, and derived bundle identifier. |
| Capability snapshots | targets | SQLite | `api/internal/targets/schema.sql` | Rolling window per target; latest always retained. | Records what was *probed*. A Linux host's `unsupported` rows are recorded as terminal, not as a failed probe. |
| Build-host role state | targets | SQLite | `api/internal/targets/schema.sql` | Latest plus history. | Which node currently fills the role, its macOS and Xcode versions, and whether it can still produce submittable builds. |
| Build records | builds | SQLite | `api/internal/builds/schema.sql` | Governed retention; builds referenced by a release are pinned. | Includes the SDK-floor assertion result, the observed toolchain version, and the executing node identity. |
| Build artifacts (`.app`, IPA) | builds | Filesystem on the producing node, producer-owned | Referenced by checksum from the build record | Same lifecycle as the build, subject to retention. | **Bytes never enter the database or a consumer payload.** For a remote build the bytes stay on the macOS node; identity crosses, the artifact does not. |
| Journey plans | journeys | SQLite | `api/internal/journeys/schema.sql` | Until superseded; versioned. | The registered conformance capability plan and its chapter definitions. |
| Run records and chapters | journeys | SQLite | `api/internal/journeys/schema.sql` | Governed retention; release-relevant runs pinned. | Chapters carry purpose, bounded policies, expected/observed, capture references, **and the executing strategy with its capability tier**. |
| Capture artifacts (frames, recordings) | journeys | Filesystem, producer-owned | Referenced by checksum from the run record | Same lifecycle as the run. | **Highest-risk data in this scenario** — see Retention And Deletion. |
| Matrix run selections | releases | SQLite | `api/internal/releases/schema.sql` | Immutable once started; retained for compare. | A rerun creates a new run; prior evidence is never overwritten. |
| Gate verdicts and promotability | releases | SQLite | `api/internal/releases/schema.sql` | Retained alongside the run. | Records whether contributing evidence was semantic or pixel-grade, because mirror-derived evidence is non-promotable. |
| Channel availability and upload receipts | distribution | SQLite | `api/internal/distribution/schema.sql` | Receipts retained for audit; availability is a rolling probe. | Receipts record upload identity, never credentials. |
| Rung state and probe results | readiness | SQLite | `api/internal/readiness/schema.sql` | Latest per rung plus history for trend. | Enrollment state is probed, never asserted from a stored flag — an expired membership must not read as ready. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| Generated-project records | projects | `api/internal/projects/schema.sql` | projects repository/service/handlers; the Builder |
| Capability snapshots, build-host role state | targets | `api/internal/targets/schema.sql` | targets repository/service/handlers; the spine `Prober` |
| Build records, assertion results | builds | `api/internal/builds/schema.sql` | builds repository/service/handlers; the spine `Builder` |
| Journey plans, runs, chapters, strategy attribution | journeys | `api/internal/journeys/schema.sql` | journeys repository/service/handlers; the spine `Driver` |
| Matrix run selections, gate verdicts, promotability | releases | `api/internal/releases/schema.sql` | releases repository/service/handlers; verdict emission |
| Channel availability, upload receipts | distribution | `api/internal/distribution/schema.sql` | distribution repository/service/handlers; the spine `Distributor` |
| Rung state, probe results | readiness | `api/internal/readiness/schema.sql` | readiness repository/service/handlers; CLI and UI walkthrough |
| Build artifacts, capture artifacts | builds, journeys | Filesystem, producer-owned; referenced by checksum | Never a table. Consumers receive `common/v1` `EvidenceRef` only. |
| Certificates, provisioning profiles, Connect API keys | — | **Not stored here.** Held by `secrets-manager`, referenced by identity. | builds and distribution, at invocation time only |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

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
| Capture artifacts (screen frames, recordings) | Run retention expiry, or explicit owner deletion. | Governed retention; runs referenced by a release gate are pinned and exempt until the release is retired. | **Highest-risk data in this scenario.** A frame captured from a physical iPhone can contain messages, tokens, banking detail, or 2FA codes — and the `ios-mirror` strategy captures whatever the mirrored screen shows, including content outside the app under test. `device-control` owns the redaction policy; this ramp must verify redaction status before a capture is referenced in a verdict, and must not ship a physical-device journey before that policy exists. |
| Build artifacts (`.app`, IPA) | Build retention expiry, or explicit owner deletion. | Retained while any release gate references the build; pinned for released versions so an update journey can install version N. | Artifacts produced on a remote node are retained on that node. Retention must be enforced where the bytes live, not only where the record lives. |
| Run records and chapters | Run retention expiry. | Governed retention; release-relevant runs pinned. | None. |
| Matrix run selections and gate verdicts | Retained with the run. | Immutable; a rerun creates a new run rather than mutating a prior one. | None — immutability is a spine invariant, not a local choice. |
| Build-host role state | Superseded by the next probe; history retained. | Latest plus history. | History matters here: knowing when a host stopped being able to produce submittable builds is the difference between a planned migration and a surprise at upload. |
| Upload receipts | Never deleted by product behavior. | Long-lived, governed retention. | Receipts are the record of what was published and by whom; truncating them silently would defeat their purpose. |
| Capability snapshots | Superseded by the next probe; history trimmed to a rolling window. | Latest snapshot per target always retained. | None. |
| Generated-project records | Regeneration, or explicit owner deletion. | Kept until superseded. | None. |
| Rung state | Superseded by the next probe. | Latest per rung plus history. | Enrollment must always be re-probed rather than trusted from storage, so a lapsed membership is never reported as ready. |
| Signing material | **Not stored here.** | Held only by `secrets-manager`. | None by construction — material that never enters this scenario cannot leak from it. |

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
