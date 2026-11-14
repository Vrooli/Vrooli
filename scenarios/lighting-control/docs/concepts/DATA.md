# Data — Lighting Control

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
The same owned schema bootstrap and controller identity migration initialize
each admitted routed test pool through `RoutedDB.SetTestPoolInitializer`.
An empty test lease is initialized before requests can reach it; initialization
failure rejects admission through the shared owner rather than using primary.

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
| Enrolled controllers and stable zones | control | SQLite | `api/internal/control/schema.sql` | Until explicit unenrollment. | Serial or reported device location forms durable identity. SDK handles and zone indexes are discovery-time addresses only. Controllers without unique stable identity remain read-only. |
| Desired lighting state and reconciliation generation | control | SQLite | `api/internal/control/schema.sql` | Retained across restart until superseded or reset explicitly. | Manual Off is durable policy. Device firmware persistence is not assumed. |
| Zone request ownership | control | SQLite | `lighting_zone_ownership` in `api/internal/control/schema.sql` | Follows the current desired row. | New explicit zone writes are manual; evaluated policy writes are automatic. Missing legacy provenance requires explicit adoption. |
| Previous manual or legacy zone settings | control | SQLite | `lighting_previous_zone_state` in `api/internal/control/schema.sql` | Latest superseded manual/legacy state and last-apply bytes per zone, retained for recovery. | Archived atomically before explicit global preset selection releases manual ownership. Further policy selections preserve that archive. |
| Completed policy selection receipt | control | SQLite | `lighting_policy_selection` in `api/internal/control/schema.sql` | Latest serialized selection identity and original request time; replaced by the next explicit selection. | Commits atomically with ownership, archive and Off mutation. A retried acknowledged transition cannot adopt a newer manual request. |
| Causal override revision | control | SQLite | `lighting_override_order` in `api/internal/control/schema.sql` | Monotonic numeric revision and causing selection identity for the latest Off publication/release. | Advances in the same transaction as the override mutation. Pending policy requests persist the captured revision; wall-clock changes cannot reverse precedence. Missing legacy revision is unknown ordering, not proof that Off is older. |
| Preset definitions and active request/expiry | policy | SQLite | `api/internal/policy/schema.sql` | Definitions persist; expired requests are cleared by deterministic reconciliation. | Telemetry samples are consumed from source APIs, not copied into a local history. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| controller, zone, desired state | control | `api/internal/control/schema.sql` | control repository/service/handlers |
| preset, active request | policy | `api/internal/policy/schema.sql` | policy repository/service/handlers |
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
| Enrollment and desired state | control | Explicit unenrollment or reset. | Retained until then. | Removing hardware does not erase desired state; reconnect reapplies it. |
| Preset configuration | policy | Explicit preset deletion/reset. | Retained until then. | No external telemetry samples are retained here. |

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
