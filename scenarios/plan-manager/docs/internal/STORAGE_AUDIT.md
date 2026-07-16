# Plan Manager Storage Architecture Audit

## Last Updated

2026-07-16

## Current Pattern

- Per-domain, embedded `internal/<domain>/schema.sql` providers registered by `internal/modules`.
- SQLite through `api-core/database` and the variant-aware `api-core/storage` resolver.
- Canonical local database: `~/.vrooli/data/vrooli/plan-manager/plan-manager.db`.
- Startup runs a small, domain-owned migration before schema drift verification when a compatible additive SQLite change is needed. It is idempotent, runs before listeners open, and never rewrites evidence during reads.

## Architecture Status

- Each SQL table is owned by one domain; `internal/database/system.sql` remains empty for SQLite.
- Repositories are domain-local and handlers do not issue SQL directly. `storage-health`'s three `DIRECT_SQL_IN_HANDLERS` findings are false positives on endpoint-descriptor strings, not executable SQL.
- SQLite is isolated by the scenario namespace and uses WAL, foreign keys, a busy timeout, and a single connection to avoid nested-query deadlocks.

## Migration Hygiene

- `EnsureSchemas` applies only idempotent desired-state schemas and detects SQLite column drift at boot.
- `validation.EnsureMigrations` adds missing terminal-result receipt columns before `EnsureSchemas` performs its SQLite drift check. Old result rows receive safe zero values and cannot satisfy an execution gate; they require fresh validation.
- Validation-operation payloads are forward-only schema V2. Older command-only payloads are rejected with an actionable migration error rather than silently rewritten on read.

## Cross-References

- `storage-health validate scenario plan-manager`
- `packages/api-core/database/schemas.go`
- `docs/concepts/DATA.md`
