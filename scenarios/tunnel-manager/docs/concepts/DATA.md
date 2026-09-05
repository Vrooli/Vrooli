# Data — Tunnel Manager

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

> Status: implemented. The product tables below are applied at API boot
> from the domain-owned `schema.sql` files and are exercised by the API
> repository/service tests. Retention gaps are called out explicitly.

## Why SQLite Only

Tunnel Manager is foundational infrastructure: it must keep working when
other resources (including Postgres, Redis, Qdrant) are down — that is
precisely when an operator needs to know the tunnel's state and recover it.
A self-contained embedded store removes a startup/availability dependency
the rest of the system might be missing. The prior tunnel-manager wrongly
required Postgres with an empty schema; this regen drops it. See the
**SQLite only** decision in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).
Revisit only if a domain truly needs a shared relational store.

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
database reachability. Keep blob/opaque bytes outside proto payloads,
behind a seam such as BlobStore if a future domain introduces them.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| `routes` (exposure manifest, SSOT) | `routes` | SQLite | `api/internal/routes/schema.sql` | Until route deleted/unexposed; CORE routes persist while in coreset. | The single source of truth other domains reconcile against. |
| `leases` (on-demand exposure) | `exposure` | SQLite | `api/internal/exposure/schema.sql` | Until expired+reaped or revoked; default TTL ≈ 1 week. | Drives LEASED-tier reaping; CORE tier needs no lease. |
| `tunnel_config` (mode + tunnel identity) | `config` | SQLite | `api/internal/config/schema.sql` | Long-lived; one active config. | Credential is referenced, not stored inline (see [`../internal/SECURITY.md`](../internal/SECURITY.md)). |
| `metrics` (cloudflared time-series) | `tunnel` | SQLite | `api/internal/tunnel/schema.sql` | Rolling 14-day window, pruned on metrics writes. | Scraped from Prometheus endpoint; bounded growth. |
| `probes` (liveness probe history) | `probes` | SQLite | `api/internal/probes/schema.sql` | Rolling 14-day window, pruned on probe writes. | Internal + external probe results + failure class. |
| `recovery_events` (recovery audit log) | `recovery` | SQLite | `api/internal/recovery/schema.sql` | Rolling 90-day incident window, pruned on event writes. | Append-only attempt log. |
| _(audit findings)_ | `audit` | None (computed) | n/a | Not persisted. | Port-compliance is computed live from `service.json` vs the manifest. |

## Column Sketches

These sketches summarize the implemented schema shape. The exact
constraints live in the linked `schema.sql` files.

- **`routes`** — `subdomain` (DNS label, unique), `scenario`, `domain`
  (field, default `itsagitime.com`), `local_port` (fixed UI port), `tier`
  (`core`|`leased`), `lease_id` (nullable FK → `leases`), `enabled`,
  `health_path`, `created_at`, `updated_at`. `public_url` is derived
  (`https://<subdomain>.<domain>`), not stored. One route per subdomain.
- **`leases`** — `id`, `scenario`, `requested_by`, `created_at`,
  `expires_at`, `extended_count`, `status` (`active`|`expired`|`revoked`).
- **`tunnel_config`** — `mode` (`remote`|`local`), `tunnel_id`,
  `account_id`, `credential_ref`, `prometheus_endpoint` (default
  `127.0.0.1:20241`), `updated_at`.
- **`metrics`** — `id`, `observed_at`, `ha_connections`, `request_errors`,
  `rtt_ms`, `active_streams`.
- **`probes`** — `id`, `subdomain`, `kind` (`internal`|`external`),
  `status`, `latency_ms`, `status_code`, `error_msg`, `created_at`.
  Failure classification is derived from latest internal/external probe
  pairs rather than stored as a per-row enum.
- **`recovery_events`** — `id`, `trigger`, `action`, `outcome`,
  `details`, `attempt`, `created_at`.

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `routes` | `routes` | `api/internal/routes/schema.sql` | routes repository/service/handlers; read by `exposure`, `config`, `audit`, `probes` |
| `leases` | `exposure` | `api/internal/exposure/schema.sql` | exposure repository/service/handlers (request/extend/revoke/reap) |
| `tunnel_config` | `config` | `api/internal/config/schema.sql` | config repository/service/handlers |
| `metrics` | `tunnel` | `api/internal/tunnel/schema.sql` | tunnel repository/service/handlers; read by `recovery` |
| `probes` | `probes` | `api/internal/probes/schema.sql` | probes repository/service/handlers; read by `recovery` |
| `recovery_events` | `recovery` | `api/internal/recovery/schema.sql` | recovery repository/service/handlers |
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
| `routes` | Route deleted / scenario unexposed; CORE routes removed only when dropped from coreset. | Live while exposed. | No automated historical archive; manifest is current state. |
| `leases` | Explicit revoke or reconcile-driven expiry. | Default TTL ≈ 1 week; reaped after expiry unless scenario is also CORE. | Hostname-budget/LRU eviction is P2. |
| `tunnel_config` | Overwritten on mode switch / re-config. | One active config retained. | Vault/provider credential references are deferred. |
| `metrics` | Repository write prunes rows older than `tunnel.MetricsRetentionWindow`. | Rolling 14-day time-series window. | None. |
| `probes` | Repository write prunes rows older than `probes.HistoryRetentionWindow`. | Rolling 14-day probe-history window. | None. |
| `recovery_events` | Repository write prunes rows older than `recovery.EventRetentionWindow`. | Rolling 90-day incident-review window. | None. |

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
