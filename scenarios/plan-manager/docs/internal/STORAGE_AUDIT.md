# Plan Manager Storage Architecture Audit

## Last Updated

2026-07-13

## Current Pattern

- Per-domain, embedded `internal/<domain>/schema.sql` providers registered by `internal/modules`.
- SQLite through `api-core/database` and the variant-aware `api-core/storage` resolver.
- Canonical local database: `~/.vrooli/data/vrooli/plan-manager/plan-manager.db`.
- The database is development/personal-runtime data, so schema evolution uses a stopped-service, one-shot script in `/tmp/plan-manager/`; Plan Manager ships no versioned migration runner or read-time schema upgrader.

## Architecture Status

- Each SQL table is owned by one domain; `internal/database/system.sql` remains empty for SQLite.
- Repositories are domain-local and handlers do not issue SQL directly. `storage-health`'s three `DIRECT_SQL_IN_HANDLERS` findings are false positives on endpoint-descriptor strings, not executable SQL.
- SQLite is isolated by the scenario namespace and uses WAL, foreign keys, a busy timeout, and a single connection to avoid nested-query deadlocks.

## Migration Hygiene

- `EnsureSchemas` applies only idempotent desired-state schemas and detects SQLite column drift at boot.
- Local data is upgraded before boot by a temporary, idempotent migration script; the script is deleted after post-migration integrity and drift checks pass.
- Validation-operation payloads are forward-only schema V2. Older command-only payloads are rejected with an actionable migration error rather than silently rewritten on read.

## Cross-References

- `storage-health validate scenario plan-manager`
- `packages/api-core/database/schemas.go`
- `docs/concepts/DATA.md`
