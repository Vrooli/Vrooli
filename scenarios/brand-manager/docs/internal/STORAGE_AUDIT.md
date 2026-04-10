# Brand Manager Storage Architecture Audit

## Last Updated
2026-03-26

## Resource Configuration Status
- [x] sqlite declared in service.json (type: sqlite, embedded — no resource daemon needed)
- [x] Environment variable BM_SQLITE_PATH used for path resolution
- [x] Schema embedded in Go binary (api/database/schema.sql) — auto-init on startup

## Connection Pattern Status
- [x] Environment variables used (BM_SQLITE_PATH, SQLITE_PATH, SQLITE_DB fallbacks)
- [x] Connection via api-core/database.Connect with retry and backoff
- [x] Connection pool configured (MaxOpenConns=1 for SQLite single-writer)
- [x] Health check implemented via api-core/health.DB

## Schema Status
- [x] schema.sql exists and is idempotent (CREATE TABLE IF NOT EXISTS)
- [x] Tables use proper constraints (FOREIGN KEY, UNIQUE, NOT NULL)
- [x] Indexes on common query paths (brand_id, scenario_name)
- [x] Greenfield default applied (no migration logic)
- [x] WAL mode + performance pragmas set via DSN

## Abstraction Status
- [x] Repository interfaces defined (BrandRepository, VersionRepository, AssignmentRepository)
- [x] Business logic (handlers) uses interfaces, not direct DB
- [x] SQLite implementations separated (sqlite_brands.go, sqlite_versions.go, sqlite_assignments.go)

## Filesystem Status
- [ ] Runtime filesystem writes for assets not yet implemented (planned: OT-P0-002)
- [ ] Should use api-core/storage when asset management is added
- [x] SQLite DB path resolved via environment, not scenario-local

## Issues Found
None critical — storage layer is well-structured.

## Follow-up Tasks
1. Asset file storage (OT-P0-002 BM-REQ-STORE-ASSETS) — implement atomic writes to `~/.vrooli/brand-manager/assets/{brand_id}/` using api-core/storage
2. Consider adding `api-core/storage.NewResolver` for DB path resolution to use standard XDG paths
