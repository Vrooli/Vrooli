# Data — Device Control

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
| Device records | devices | SQLite | `api/internal/devices/schema.sql` | Until the device leaves the bridge fleet. | Mirrors bridge fleet identity; bridge remains the source of truth for identity and trust. |
| Capability snapshots | devices | SQLite | `api/internal/devices/schema.sql` | Rolling window per device; latest snapshot always retained. | Records what was *probed*, never what the device kind usually has. |
| Authentication profiles | auth | SQLite | `api/internal/auth/auth.go` | Metadata retained for audit; authority-held values are deleted or rotated separately. | Stores device binding, method, policy, status, and credential-authority reference only; never a secret value. |
| Strategy registrations | strategies | SQLite | `api/internal/strategies/schema.sql` | Until the strategy is unregistered. | Declaration only; the verified tier comes from a conformance result. |
| Conformance results | strategies | SQLite | `api/internal/strategies/schema.sql` | Latest per strategy plus history for trend. | The authority for what a strategy can actually do. |
| Lease records | sessions | SQLite | `api/internal/sessions/schema.sql` | Retained after expiry for audit. | Expired leases are never deleted eagerly — they are the audit trail for who held a device when. |
| Verb audit | sessions | SQLite | `api/internal/sessions/schema.sql` | Long-lived; governed retention. | Actor, device, lease, verb, outcome. Append-only. |
| Flow definitions | flows | SQLite | `api/internal/flows/schema.sql` | Until deleted by the owner; versioned. | Includes declared capability requirements so a flow can be checked before a run. |
| Run records and chapters | flows | SQLite | `api/internal/flows/schema.sql` | Governed retention; release-relevant runs pinned. | Chapters carry purpose, bounded policies, expected/observed, and capture references. |
| Capture artifacts | flows | Filesystem, producer-owned | Referenced by checksum from the run record | Same lifecycle as the run, subject to retention. | **Bytes never enter the database or a consumer payload.** Consumers receive `common/v1` `EvidenceRef` only. |
| Visual anchors | flows | Filesystem + SQLite metadata | `api/internal/flows/schema.sql` | Until the owning flow or anchor is deleted. | Reference images with stable identity for the deterministic middle rung. |
| Agent run records | agent | Service memory for the current process | `api/internal/control/agent.go` | Until the scenario restarts; promoted flow runs use the existing flow-run store. | Includes typed chapters and executable planned steps; promotion copies them into the deterministic flow export path. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| Device records, capability snapshots | devices | `api/internal/devices/schema.sql` | devices repository/service/handlers; inventory probe |
| Strategy registrations, conformance results | strategies | `api/internal/strategies/schema.sql` | strategies repository/service/handlers; `strategy verify` |
| Lease records, verb audit | sessions | `api/internal/sessions/schema.sql` | sessions repository/service/handlers; lease enforcement, kill switch |
| Flow definitions, runs, chapters, anchor metadata | flows | `api/internal/flows/schema.sql` | flows repository/service/handlers; executor, gap report, evidence sink |
| Agent run records, promotion provenance | agent | `api/internal/control/agent.go` | agent service/handlers; run-to-flow promotion |
| Capture artifacts, anchor images | flows | Filesystem, producer-owned; referenced by checksum | Never a table. Consumers receive `common/v1` `EvidenceRef` only. |
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
| Capture artifacts (screen frames, recordings) | Run retention expiry, or explicit owner deletion. | Governed retention; runs referenced by a delivery-ramp release are pinned and exempt until the release is retired. | **Highest-risk data in this scenario.** A screen frame from a personal phone can contain messages, tokens, banking detail, or 2FA codes. Redaction status must be verified before a capture leaves the producer, and the redaction policy itself needs review before the first physical-device strategy ships. |
| Verb audit | Never deleted by product behavior. | Long-lived, governed retention. | Audit is the accountability record for a capability that can drive a personal device; truncating it silently would defeat its purpose. |
| Lease records | Retained after expiry. | Governed retention alongside audit. | None; expiry releases the device but keeps the record. |
| Capability snapshots | Superseded by the next probe; history trimmed to a rolling window. | Latest snapshot per device always retained. | None. |
| Authentication profiles | Profile revocation; delete or rotate the authority-held credential separately. | Metadata remains for audit and troubleshooting. | Credential values are never owned by device-control and must not be copied into SQLite. |
| Device records | Device leaves the bridge fleet. | Removed with the fleet member. | Bridge owns the lifecycle; this scenario must not resurrect a removed device from a stale snapshot. |
| Flow definitions and run records | Owner deletion, or run retention expiry. | Definitions kept until deleted; runs under governed retention. | None. |

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
