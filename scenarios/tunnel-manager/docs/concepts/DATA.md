# Data — Tunnel Manager

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

> Status: documentation-first (Phase 1). The tables below describe the
> **planned** schema. No product tables exist yet and no data is seeded;
> only the template scaffold + the fenced `notes` example are present.
> Column sketches are illustrative and will firm up in Phase 2. Domain
> ownership is authoritative per [`DOMAINS.md`](DOMAINS.md).

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
| `routes` (exposure manifest, SSOT) | `routes` | SQLite | `api/internal/routes/schema.sql` | Until route deleted/unexposed; CORE routes persist while in coreset. | The single source of truth other domains reconcile against. |
| `leases` (on-demand exposure) | `exposure` | SQLite | `api/internal/exposure/schema.sql` | Until expired+reaped or revoked; default TTL ≈ 1 week. | Drives LEASED-tier reaping; CORE tier needs no lease. |
| `tunnel_config` (mode + tunnel identity) | `config` | SQLite | `api/internal/config/schema.sql` | Long-lived; one active config. | Credential is referenced, not stored inline (see [`../internal/SECURITY.md`](../internal/SECURITY.md)). |
| `metrics` (cloudflared time-series) | `tunnel` | SQLite | `api/internal/tunnel/schema.sql` | Rolling window (retention TBD in Phase 2). | Scraped from Prometheus endpoint; bounded growth. |
| `probes` (liveness probe history) | `probes` | SQLite | `api/internal/probes/schema.sql` | Rolling window (retention TBD in Phase 2). | Internal + external probe results + failure class. |
| `recovery_events` (recovery audit log) | `recovery` | SQLite | `api/internal/recovery/schema.sql` | Retained for post-incident review (retention TBD). | Append-only attempt log. |
| _(audit findings)_ | `audit` | None (computed) | n/a | Not persisted. | Port-compliance is computed live from `service.json` vs the manifest. |

## Column Sketches (planned)

Illustrative shapes consistent with [`DOMAINS.md`](DOMAINS.md). Exact
types/constraints land in Phase 2.

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
- **`probes`** — `id`, `route_subdomain`, `kind` (`internal`|`external`),
  `status`, `latency_ms`, `error`, `failure_class`
  (`tunnel-down`|`scenario-down`|`cloudflare-outage`|`dns-failure`|`config-drift`),
  `observed_at`.
- **`recovery_events`** — `id`, `started_at`, `finished_at`, `trigger`,
  `action`, `outcome`, `attempt`, `backoff_ms`, `breaker_state`.

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
| `routes` | Route deleted / scenario unexposed; CORE routes removed only when dropped from coreset. | Live while exposed. | Phase 2 must wire reconcile-driven removal. |
| `leases` | Reaper removes expired leases; explicit revoke. | Default TTL ≈ 1 week; reaped after expiry unless scenario is also CORE. | Reaper + TTL not yet implemented. |
| `tunnel_config` | Overwritten on mode switch / re-config. | One active config retained. | Mode-switch migration not yet built. |
| `metrics` | Rolling-window prune. | Bounded time-series window (length TBD). | Prune policy undefined in Phase 1. |
| `probes` | Rolling-window prune. | Bounded probe history (length TBD). | Prune policy undefined in Phase 1. |
| `recovery_events` | Optional prune of old incidents. | Retained for post-incident review. | Prune policy undefined in Phase 1. |

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
