# lifestyle-dashboard Storage Architecture Audit

## Last Updated
2026-03-10

## Executive Summary

This scenario uses **SQLite** (embedded) as its primary storage backend. This is an **intentional architectural decision** to enable:
- Single-file portability for mobile/desktop deployments
- No external dependencies (no PostgreSQL, no Redis)
- Simplified deployment (file-based, not networked)

**Recent Update (2026-03-10):**
1. Migrated to `api-core/database` package which supports SQLite via `DriverSQLite`
2. Added automatic retry with minimal backoff (3 attempts, 100ms base) for robustness
3. Repository pattern fully implemented with interfaces for all storage operations
4. All auditor violations resolved (HIGH severity `database_backoff` → PASS)

## Resource Configuration Status

| Item | Status | Notes |
|------|--------|-------|
| postgres declared in service.json | N/A | SQLite used instead (by design) |
| schema field uses scenario slug | N/A | SQLite uses SQLITE_PATH env var |
| initialization files referenced | Partial | Schema in api/domain/schema.go |
| redis/qdrant properly configured | N/A | Not used (P0 scope) |

## Connection Pattern Status

| Item | Status | Notes |
|------|--------|-------|
| Environment variables used | **Yes** | `SQLITE_PATH`, `SCENARIO_DATA_DIR` |
| Uses api-core/database | **Yes** | `database.Connect()` with `DriverSQLite` |
| Connection retry with backoff | **Yes** | 3 attempts, 100ms base, 500ms max |
| Connection pool configured | **Yes** | `MaxOpenConns=1`, `MaxIdleConns=1` (SQLite single-writer) |
| Health check implemented | **Yes** | `sqliteChecker` in api/main.go |

## Schema Status

| Item | Status | Notes |
|------|--------|-------|
| schema.sql exists and is idempotent | Yes | Schema in api/domain/schema.go `InitSchema()` |
| Tables use proper constraints and indexes | Yes | Indexes on domain+timestamp, timestamp, event_type, hypothesis_id |
| Greenfield default applied | Yes | No migration shims needed |
| Brownfield migrations documented | N/A | New scenario |

## Abstraction Status

| Item | Status | Notes |
|------|--------|-------|
| Repository interfaces defined | **Yes** | `api/repository/interfaces.go` - EventRepository, DomainRepository, StatsRepository |
| Business logic uses interfaces | **Yes** | Handler struct uses repository interfaces, not direct *sql.DB |
| SQLite implementations | **Yes** | `sqlite_events.go`, `sqlite_domains.go`, `sqlite_stats.go` |
| Repository unit tests | **Yes** | 15 tests in `repository_test.go` covering all operations |
| Multiple storage backends abstracted | Partial | Interface-ready for future PostgreSQL/in-memory implementations |

## Filesystem Status

| Item | Status | Notes |
|------|--------|-------|
| Runtime filesystem writes | N/A | No direct filesystem writes (SQLite handles all persistence) |
| Deploy directory treated as disposable | **Yes** | DB path from `SCENARIO_DATA_DIR` env var |
| Atomic writes used for persisted files | N/A | SQLite handles atomicity |

## Repository Architecture

```
api/repository/
├── interfaces.go         # Repository interfaces (EventRepository, DomainRepository, StatsRepository)
├── sqlite_events.go      # SQLite EventRepository implementation
├── sqlite_domains.go     # SQLite DomainRepository implementation
├── sqlite_stats.go       # SQLite StatsRepository implementation
└── repository_test.go    # 15 unit tests for repository layer
```

### Interface Design

```go
// EventRepository abstracts event storage operations
type EventRepository interface {
    Create(ctx context.Context, event *domain.Event) error
    GetByID(ctx context.Context, id string) (*domain.Event, error)
    List(ctx context.Context, filter EventFilter) ([]domain.Event, error)
}

// DomainRepository abstracts domain storage operations
type DomainRepository interface {
    Upsert(ctx context.Context, d *domain.Domain) error
    GetByName(ctx context.Context, name string) (*domain.Domain, error)
    List(ctx context.Context) ([]domain.Domain, error)
    UpdateStatus(ctx context.Context, name, status, lastHealthAt string) error
    Update(ctx context.Context, name string, updates map[string]interface{}) error
}

// StatsRepository abstracts statistics queries
type StatsRepository interface {
    GetTimeline(ctx context.Context, days int) ([]domain.TimelineEntry, error)
    GetSummary(ctx context.Context) (*domain.SummaryResponse, error)
}
```

### Benefits Achieved

1. **Testability**: Handler tests use real SQLite repositories; repository tests verify storage logic independently
2. **Separation of Concerns**: Handlers focus on HTTP, repositories focus on SQL
3. **Error Handling**: Custom `ErrNotFound` type with `IsNotFound()` helper
4. **Context Propagation**: All repository methods accept context for cancellation
5. **Future Flexibility**: Can add PostgreSQL/in-memory implementations without changing handlers
6. **Auditor Compliance**: Uses api-core/database package (no violations)

## SQLite-Specific Configuration

```go
// api/main.go - Uses api-core/database with SQLite driver
db, err := database.Connect(ctx, database.Config{
    Driver:       database.DriverSQLite,
    MaxOpenConns: 1, // SQLite single-writer constraint
    MaxIdleConns: 1,
    Retry: &retry.Config{
        MaxAttempts: 3,
        BaseDelay:   100 * time.Millisecond,
        MaxDelay:    500 * time.Millisecond,
    },
})
```

**Key configuration choices:**
- `DriverSQLite`: Uses api-core's SQLite support with SQLITE_PATH env var
- `MaxOpenConns=1`: SQLite single-writer constraint (by design)
- Minimal retry: 3 attempts with 100ms base (SQLite is local, no network issues)
- WAL mode + busy_timeout: Set via SQLITE_PATH query params

## Known Limitations

### PROB-002: Single-Writer SQLite Constraint
- **Status:** By design
- **Impact:** All writes must go through API; no direct UI→DB writes
- **Notes:** Enables WAL mode and concurrent readers, matches intended architecture

## Auditor Compliance Status

| Check | Status | Notes |
|-------|--------|-------|
| Security scan | **PASS** | 0 vulnerabilities |
| database_backoff | **PASS** | Uses api-core/database.Connect() |
| setup_steps | **PASS** | Fixed `build-ui` command pattern |

**Resolved violations:**
- `api/main.go:192` - Direct `sql.Open()` → Migrated to `database.Connect()` with `DriverSQLite`
- `.vrooli/service.json:119` - `build-ui` command pattern → Fixed to use `pnpm run build --ignore-workspace`

## Architecture Decision Records

### ADR-001: Use SQLite instead of PostgreSQL

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
- Use repository pattern for storage abstraction

**Consequences:**
- Single-writer constraint (acceptable for personal dashboard)
- File-based backup/restore is trivial
- No network configuration needed
- Repository interfaces enable future backend swapping

### ADR-002: Repository Pattern Implementation

**Decision:** Implement repository interfaces for all storage operations
**Date:** 2026-03-10
**Status:** Accepted

**Context:**
- Handlers had direct `*sql.DB` dependency
- Unit testing required database setup
- Storage logic mixed with HTTP handling

**Decision:**
- Define repository interfaces in `api/repository/interfaces.go`
- Implement SQLite-specific repositories
- Inject repositories into handlers at startup
- Add custom error type `ErrNotFound` for not-found handling

**Consequences:**
- +15 repository unit tests added
- Handlers simplified (~40% fewer lines)
- Can add mock repositories for unit tests
- Can add PostgreSQL implementation for server deployments
- Total test count: 51 (22 main + 4 domain + 10 handlers + 15 repository)

### ADR-003: Migrate to api-core/database Package

**Decision:** Use api-core/database package instead of direct sql.Open()
**Date:** 2026-03-10
**Status:** Accepted

**Context:**
- scenario-auditor flagged direct `sql.Open()` as HIGH severity violation
- api-core/database package supports SQLite via `DriverSQLite`
- Package provides automatic retry with backoff (even for SQLite)

**Decision:**
- Migrate from `sql.Open("sqlite3", dsn)` to `database.Connect(ctx, cfg)`
- Use minimal retry config (3 attempts, 100ms base) for robustness
- Set SQLITE_PATH env var for api-core to read

**Consequences:**
- HIGH severity auditor violation resolved
- Adds retry robustness (handles transient SQLite busy states)
- Consistent with other Vrooli scenarios
- No behavioral changes to existing functionality
