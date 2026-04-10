# Implementation Plan: Greenfield Storage Migration — PostgreSQL to SQLite

## 1. Purpose

Hard-cut greenfield migration of the development-toolchain-validator (DTV) storage layer from PostgreSQL to SQLite. Delete all Postgres-specific code and rebuild clean with SQLite implementations for all three domains: reference, skill, and expectation.

## 2. Required Reading

```bash
prompt-manager skill read storage-steer
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read cli-steer api-steer
```

**Key reference files:**
- `scenarios/development-toolchain-validator/api/domain/reference/repository.go` — Reference repository interface (6 methods)
- `scenarios/development-toolchain-validator/api/domain/skill/repository.go` — Skill repository interface (7 methods)
- `scenarios/development-toolchain-validator/api/domain/expectation/repository.go` — Expectation interfaces (StructuralRepository: 5 methods, CLIRepository: 5 methods)
- `scenarios/development-toolchain-validator/api/infrastructure/postgres/` — Existing Postgres implementations (DELETE target)
- `scenarios/development-toolchain-validator/api/main.go` — Server bootstrap and repo wiring
- `packages/api-core/storage/resolver.go` — Storage path resolver (`ClassData` for DB files)
- `scenarios/agent-manager/api/internal/database/connection.go` — Reference SQLite pattern with pragmas

## 3. Problem Statement

DTV currently depends on PostgreSQL for persistence, but only the reference and skill repos have Postgres implementations. The expectation domain has interfaces, service, and handlers but **no persistence layer at all**. PostgreSQL is heavyweight for DTV's single-user local workload. SQLite eliminates the external dependency while completing the missing expectation repos.

## 4. Scope

**Acceptance patterns (settled, round 1):**
- `acceptance_allow`: `scenarios/development-toolchain-validator/**`
- `acceptance_deny`: TBD (round 2, decision d1)

**In scope:**
- Delete `infrastructure/postgres/` directory entirely (reference_repo.go, skill_repo.go, tests)
- Delete `initialization/postgres/` directory (schema.sql, seed.sql)
- Remove `github.com/lib/pq` from go.mod
- Remove postgres resource dependency from `.vrooli/service.json`
- Create `infrastructure/sqlite/` with implementations for ALL 4 interfaces:
  - `reference_repo.go` — reference.Repository
  - `skill_repo.go` — skill.Repository
  - `structural_expectations_repo.go` — expectation.StructuralRepository
  - `cli_assertions_repo.go` — expectation.CLIRepository
- Create `infrastructure/sqlite/schema.sql` — SQLite-compatible schema (all 7 tables — settled, round 1)
- Create `infrastructure/sqlite/sqlite.go` — connection setup, schema init, pragmas
- Update `main.go` — switch from Postgres to SQLite wiring, wire all repos including expectations
- Update tests to use SQLite (temp file per test)

**Out of scope:**
- Domain interface changes (reference.Repository, skill.Repository, expectation.*Repository stay as-is)
- Service layer changes
- Handler changes (except wiring expectation handlers in main.go)
- Data migration (greenfield — no existing data to preserve)
- CLI changes

## 5. Current Technical Context

### Existing Implementations
| Domain | Interface | Postgres Impl | Lines |
|--------|-----------|--------------|-------|
| Reference | reference.Repository (6 methods) | ✅ reference_repo.go | ~250 |
| Skill | skill.Repository (7 methods) | ✅ skill_repo.go | ~280 |
| Expectation | expectation.StructuralRepository (5 methods) | ❌ Missing | — |
| Expectation | expectation.CLIRepository (5 methods) | ❌ Missing | — |

### PostgreSQL Schema (7 tables)
1. `reference_scenarios` — UUID PK, slug (unique), name, template, path, description, timestamps
2. `skill_connections` — UUID PK, reference_id FK, skill_id, version/hash, unique(ref_id, skill_id), timestamps
3. `structural_expectations` — UUID PK, connection_id FK, type (ENUM), pattern, required, expected_content, description
4. `cli_assertions` — UUID PK, connection_id FK, command, json_path, operator (ENUM), expected_value (JSONB), description
5. `validation_runs` — UUID PK, reference_id FK, timestamps, status (ENUM)
6. `structural_results` — UUID PK, run_id FK, expectation_id FK, status, actual_value, error_message
7. `cli_results` — UUID PK, run_id FK, assertion_id FK, status, actual_value (JSONB), execution_time_ms

### PostgreSQL Features Requiring SQLite Adaptation
- `uuid_generate_v4()` → Generate UUIDs in Go with `github.com/google/uuid` (already indirect dep at v1.6.0, promote to direct)
- `ENUM` types → `TEXT` with `CHECK` constraints
- `JSONB` → `TEXT` (store as JSON string, parse in Go)
- `$1` placeholders → `?` placeholders
- `PL/pgSQL` triggers for `updated_at` → Handle in repo layer (Go code sets timestamp on update)
- Connection pool → `MaxOpenConns(1)` for SQLite single-writer model
- `RETURNING` clause → **Fully supported** by modernc.org/sqlite v1.34.5 (bundles SQLite 3.46+). All 4 existing `INSERT/UPDATE ... RETURNING` patterns in Postgres repos can be directly ported.

### api-core Infrastructure (Already Available)
- `storage.NewResolver()` with `ClassData` — provides `~/.vrooli/data/vrooli/<scenario>/<file>.db` **(settled, round 1: use this pattern)**
- WAL mode, foreign keys, busy timeout pragmas — established patterns in agent-manager

### SQLite Driver
- `modernc.org/sqlite` v1.34.5 — pure Go, no CGO, matches agent-manager. Bundles SQLite 3.46+.

## 6. Target End State

- DTV has zero PostgreSQL dependencies (no lib/pq, no postgres resource, no postgres init files)
- All 4 repository interfaces have working SQLite implementations
- Expectation handlers are fully wired in main.go (currently commented out)
- SQLite database file stored via api-core storage resolver at `ClassData` path
- Schema initialized on first connection with idempotent `CREATE TABLE IF NOT EXISTS`
- Tests use temporary SQLite files (no Docker/testcontainers needed)
- `go build ./...` and `go test ./...` pass clean

## 7. Implementation Strategy

### Phase 1: Delete PostgreSQL (Clean Slate)
1. Delete `api/infrastructure/postgres/` directory
2. Delete `initialization/postgres/` directory
3. Remove `github.com/lib/pq` from go.mod, run `go mod tidy`
4. Remove postgres resource from `.vrooli/service.json`

### Phase 2: SQLite Foundation
1. Create `api/infrastructure/sqlite/sqlite.go`:
   - `NewDB(dbPath string) (*sql.DB, error)` — opens SQLite with pragmas (WAL, foreign_keys, busy_timeout, cache_size, synchronous=NORMAL)
   - `InitSchema(db *sql.DB) error` — runs schema SQL (loading approach TBD — round 2, decision d2)
   - Connection config: `MaxOpenConns(1)`
2. Create `api/infrastructure/sqlite/schema.sql`:
   - Convert all 7 tables from PostgreSQL to SQLite syntax
   - `TEXT` with `CHECK` constraints instead of ENUMs
   - `TEXT` for UUIDs (generated in Go)
   - `TEXT` for JSON (was JSONB)
   - `DATETIME DEFAULT CURRENT_TIMESTAMP` for timestamps
   - All `CREATE TABLE IF NOT EXISTS` for idempotency
3. Promote `github.com/google/uuid` to direct dependency
4. Add `modernc.org/sqlite` v1.34.5

### Phase 3: Repository Implementations
1. `reference_repo.go` — Implement reference.Repository (6 methods), port from Postgres with `?` placeholders
2. `skill_repo.go` — Implement skill.Repository (7 methods), port from Postgres with `?` placeholders
3. `structural_expectations_repo.go` — Implement expectation.StructuralRepository (5 methods), NEW
4. `cli_assertions_repo.go` — Implement expectation.CLIRepository (5 methods), NEW

SQL library choice TBD (round 2, decision d3 — database/sql vs sqlx).

### Phase 4: Wiring
1. Update `main.go`:
   - Use storage resolver to get DB path, then open with `infrastructure/sqlite.NewDB()`
   - Create all 4 repo instances
   - Wire expectation repos into service and handlers (uncomment/fix the current gap)

### Phase 5: Tests
1. Create `api/infrastructure/sqlite/test_helpers_test.go` — shared `setupTestDB(t)` using `t.TempDir()`
2. Create integration tests for all 4 repos
3. Ensure existing handler/service tests still pass (they use mocks, should be unaffected)

## 8. Contract Decisions

- **Domain interfaces**: Unchanged — this is a pure infrastructure swap
- **API/CLI behavior**: Unchanged — same endpoints, same responses
- **Data model**: Unchanged — same fields, same types at the Go level
- **JSONB → TEXT**: `expected_value` and `actual_value` fields stored as JSON strings; repos marshal/unmarshal in Go
- **ENUMs → CHECK constraints**: Same valid values, enforced at DB level via CHECK
- **RETURNING**: Preserved — SQLite 3.46+ supports it natively

## 9. Testing Plan

- **Unit tests**: Existing mock-based handler/service tests should pass without changes
- **Integration tests**: New SQLite-backed repo tests using `t.TempDir()` temp databases
  - Test all CRUD operations for each repo
  - Test unique constraint enforcement (skill_connections)
  - Test cascade deletes (FK relationships)
  - Test concurrent access behavior (single-writer)
- **Build verification**: `go build ./...` passes
- **Full test suite**: `go test ./... -timeout 300s` passes

## 10. Rollout / Validation Checklist

- [ ] `go build ./...` succeeds with no Postgres imports
- [ ] `go test ./...` passes all tests
- [ ] `grep -r "lib/pq" api/` returns nothing
- [ ] `grep -r "postgres" api/infrastructure/` returns nothing
- [ ] All 4 repo interfaces have SQLite implementations
- [ ] Expectation handlers wired in main.go
- [ ] SQLite DB file created at storage resolver path on startup
- [ ] Schema creates all 7 tables on first run

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| SQLite single-writer bottleneck | Low — DTV is single-user local tool | WAL mode + busy_timeout pragma |
| JSONB query loss (can't query inside JSON) | Low — current code only stores/retrieves, no JSON path queries | Store as TEXT, marshal/unmarshal in Go |
| Missing expectation domain nuances | Medium — building fresh with no prior impl | Interfaces are well-defined; follow reference/skill patterns |
| CGO dependency with mattn/go-sqlite3 | Medium — complicates cross-compilation | Use `modernc.org/sqlite` (pure Go) like agent-manager |

## 12. Non-goals / Prohibited Patterns

- **No compatibility shims** — this is greenfield, not brownfield
- **No data migration** — there is no production data to migrate
- **No PostgreSQL fallback** — delete it completely, don't keep it as an option
- **No testcontainers** — SQLite tests use temp files, not Docker containers
- **No domain interface changes** — only infrastructure layer changes

## 13. Definition of Done

1. `infrastructure/postgres/` directory deleted
2. `initialization/postgres/` directory deleted
3. `lib/pq` removed from go.mod
4. postgres resource removed from service.json
5. `infrastructure/sqlite/` exists with 6 files: sqlite.go, schema.sql, and 4 repo files
6. All 4 repository interfaces fully implemented for SQLite
7. main.go wires SQLite connection and all 4 repos (including expectations)
8. `go build ./...` passes
9. `go test ./... -timeout 300s` passes
10. No references to PostgreSQL remain in api/ code
