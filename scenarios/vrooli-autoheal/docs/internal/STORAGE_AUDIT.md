# vrooli-autoheal Storage Audit

## Last Updated
2026-02-18

## Storage Posture
- Primary and only runtime persistence backend: SQLite (file-backed, scenario-scoped)
- No Postgres compatibility, fallback, or migration path in runtime startup

## Runtime Backend
- Fixed backend: SQLite
- Connection uses `modernc.org/sqlite` via `api-core/database`

## SQLite Path Resolution
- If `SQLITE_PATH` or `SQLITE_DB` is set, that path is used directly.
- Otherwise path is resolved with `api-core/storage`:
  - profile: `auto`
  - app: `vrooli`
  - scenario: `vrooli-autoheal`
  - file: `autoheal.sqlite` under scenario `data` class

## Schema Layout
- Active schema file: `initialization/sqlite/schema.sql`
- API startup initializes SQLite schema idempotently on boot.

## Notes
- This is a greenfield storage posture: sqlite-only, no legacy cutover scaffolding.
