# Data — Storage Manager

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

The scenario default is embedded SQLite through `modernc.org/sqlite`.
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

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Active cleanup policy | cleanup | In-memory Phase 4 store; planned SQLite | `api/internal/orchestrator/service.go` | Until replaced by a newer policy version | Profiles: conservative, balanced, aggressive. |
| Cleanup plans | cleanup | In-memory Phase 4 store; planned SQLite | `api/internal/orchestrator/service.go` | Product-defined retention, likely bounded audit history | Plan ids are stable hashes of policy/provider/preview inputs. |
| Apply attempts | cleanup | In-memory Phase 4 store; planned SQLite | `api/internal/orchestrator/service.go` | Must outlive retries for idempotency-key replay | Replays return stored results without reapplying providers. |
| Audit events | cleanup | In-memory Phase 4 store; planned SQLite | `api/internal/orchestrator/service.go` | Operator-configured retention once SQLite lands | Messages are redacted before storage. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| cleanup orchestration store | cleanup | `api/internal/orchestrator/service.go` | CleanupService handlers and CLI |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

## Migrations And Compatibility

The scenario uses idempotent schema bootstrap. Domain schema
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
| Cleanup plans/apply attempts/audit events | Future retention policy | Keep long enough to prove idempotent replay and operator auditability | SQLite schema and retention job are not implemented yet. |

## Privacy Notes

Cleanup previews and audit messages can contain host paths or command
output, so provider errors are redacted before audit storage. If future
providers collect personal, regulated, customer, financial, or sensitive
business data, update this document and
[`../internal/SECURITY.md`](../internal/SECURITY.md) before implementation
expands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
