# Data — Search Hub

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

Search Hub persists **only its own metadata** — the provider registry
and per-query telemetry. It stores **no vectors and no corpus content**
(the thin-router invariant; each provider owns its corpus). The store is
the template's embedded **SQLite** (`modernc.org/sqlite`, CGO-clean),
rooted at `SQLITE_PATH` (default `${SCENARIO_DATA_DIR}/search-hub.db`).
Schemas are applied on startup through `api-core/database`'s
`EnsureSchemas` over the `modules.AllSchemas()` registry.

> **Storage decision (2026-06-03, Phase 3).** Phase 2's orientation docs
> proposed PostgreSQL (a `vrooli_search_hub` DB was provisioned). Phase 3
> **reversed that** to SQLite — the registry holds a handful of provider
> rows plus (later) telemetry, every sibling scenario on this template's
> `RoutedDB`+modules pattern uses SQLite, and staying on SQLite keeps the
> whole pure-Go test harness intact with no new test dependency
> (testcontainers). The provisioned postgres DB is left unused (harmless).
> The `providers` table is **live as of Phase 3**; `query_telemetry`
> lands in Phase 7.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Status |
|---|---|---|---|---|---|
| Provider descriptors (`providers`) | registry | SQLite | `api/internal/registry/schema.sql` | Until deregistered | **Live — Phase 3.** Full descriptor persisted as a protojson blob plus projected filter columns (`provider_group`, `bucket`, `type`, `state`, `scope`). Includes `CAPABILITY_GAP` stubs (no endpoint). |
| Per-query telemetry (`query_telemetry`) | metrics | SQLite | `api/internal/metrics/schema.sql` (planned) | Rolling window (define in Phase 7) | **Planned — Phase 7.** Classified types, providers hit, counts, latency, re-query count, zero-result flag. Query text hashed/opt-in. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By | Status |
|---|---|---|---|---|
| `providers` table | registry | `api/internal/registry/schema.sql` | registry store/handlers, routing fan-out | Live — Phase 3 |
| `query_telemetry` table | metrics | `api/internal/metrics/schema.sql` | metrics writer + insights aggregates | Planned — Phase 7 |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup | Live |

The router imports **no qdrant client** and defines **no corpus-content
tables** — only `providers` + `query_telemetry`. This is a guarded
invariant (plan §6 Validation #4): an architectural test asserts the
absence of a qdrant import and corpus tables.

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
| Provider descriptors | `DeregisterProvider` (explicit) | Held until deregistered | Re-registration is upsert; no soft-delete history in v1. |
| Per-query telemetry | Rolling-window prune (Phase 7) | Define window + prune job in Phase 7 | Window/prune not yet implemented (Phase 7). Query text hashed/opt-in. |
| Template notes data | Domain removal (Phase 3) | Local development data only | Removed with the notes scaffold in Phase 3. |

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
