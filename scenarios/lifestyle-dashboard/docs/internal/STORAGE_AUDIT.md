# lifestyle-dashboard Storage Architecture Audit

## Last Updated
2026-03-10

## Executive Summary

This scenario uses **SQLite** (embedded) as its primary storage backend. This is an **intentional architectural decision** to enable:
- Single-file portability for mobile/desktop deployments
- No external dependencies (no PostgreSQL, no Redis)
- Simplified deployment (file-based, not networked)

The `sql.Open()` usage flagged by scenario-auditor is a **false positive** - api-core's PostgreSQL retry logic is not applicable to embedded SQLite.

## Resource Configuration Status

| Item | Status | Notes |
|------|--------|-------|
| postgres declared in service.json | N/A | SQLite used instead (by design) |
| schema field uses scenario slug | N/A | SQLite uses SQLITE_PATH env var |
| initialization files referenced | Partial | Schema in api/main.go, reference in initialization/postgres/ |
| redis/qdrant properly configured | N/A | Not used (P0 scope) |

## Connection Pattern Status

| Item | Status | Notes |
|------|--------|-------|
| Environment variables used | Yes | `SQLITE_PATH`, `SCENARIO_DATA_DIR` |
| Connection retry with exponential backoff | N/A | SQLite is local file, no retry needed |
| Connection pool configured | Yes | `SetMaxOpenConns(1)` for single-writer constraint |
| Health check implemented | Yes | `sqliteChecker` in api/main.go |

## Schema Status

| Item | Status | Notes |
|------|--------|-------|
| schema.sql exists and is idempotent | Partial | Schema in api/main.go `initSchema()`, reference SQL in initialization/postgres/ |
| Tables use proper constraints and indexes | Yes | Indexes on domain+timestamp, timestamp, event_type, hypothesis_id |
| Greenfield default applied | Yes | No migration shims needed |
| Brownfield migrations documented | N/A | New scenario |

## Abstraction Status

| Item | Status | Notes |
|------|--------|-------|
| Repository interfaces defined | No | Direct SQL in handlers (acceptable for P0 MVP) |
| Business logic uses interfaces | No | Server struct owns *sql.DB directly |
| Multiple storage backends abstracted | No | SQLite only (by design for portability) |

## Filesystem Status

| Item | Status | Notes |
|------|--------|-------|
| Runtime filesystem writes go through api-core/storage | No | Uses `SCENARIO_DATA_DIR` env var |
| Deploy directory treated as disposable | Partial | DB file location configurable |
| Atomic writes used for persisted files | N/A | SQLite handles atomicity |

## SQLite-Specific Configuration

```go
// api/main.go:748 - Intentional SQLite configuration
db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")

// SQLite single-writer constraint (by design)
db.SetMaxOpenConns(1)
db.SetMaxIdleConns(1)
```

**Why this is NOT a violation:**
- `api-core/database` is designed for PostgreSQL with network retry logic
- SQLite is a local file - there's no network to retry
- WAL mode and busy_timeout handle concurrent access appropriately
- Single-writer constraint is documented in PROBLEMS.md as PROB-002

## Known Limitations

### PROB-002: Single-Writer SQLite Constraint
- **Status:** By design
- **Impact:** All writes must go through API; no direct UI→DB writes
- **Notes:** Enables WAL mode and concurrent readers, matches intended architecture

## Issues Found

1. **api/main.go:748** - `sql.Open()` flagged as direct DB access
   - **Resolution:** This is a false positive. SQLite requires direct `sql.Open()` - api-core's database package is for PostgreSQL only.
   - **Recommendation:** Add scenario-specific exception in auditor or document as accepted deviation.

## Priority Fixes (None Required)

The current SQLite architecture is correct for the scenario's requirements:
- Portability (single file)
- No external dependencies
- Mobile/desktop deployment target

## Architecture Decision Record

**Decision:** Use SQLite instead of PostgreSQL
**Date:** 2026-03-09
**Status:** Accepted

**Context:**
- Scenario is designed for single-user personal health dashboard
- Future deployment targets include mobile apps and Electron desktop
- PostgreSQL would add unnecessary operational complexity

**Decision:**
- Use SQLite with JSON text columns for event payloads
- Store in `${SCENARIO_DATA_DIR}/lifestyle.db`
- Enable WAL mode for concurrent readers

**Consequences:**
- Single-writer constraint (acceptable for personal dashboard)
- File-based backup/restore is trivial
- No network configuration needed
- api-core/database not applicable (designed for PostgreSQL)
