# Fix Applied: Greenfield Storage Migration — PostgreSQL to SQLite

## Root Cause
DTV depended on PostgreSQL for persistence, but only had implementations for 2 of 4 repository interfaces (reference, skill). The expectation domain had interfaces, service, and handlers but no persistence layer. PostgreSQL is heavyweight for DTV's single-user local workload.

## Changes Made

### Deleted
- `api/infrastructure/postgres/` — all files (reference_repo.go, skill_repo.go, and test files)
- `initialization/postgres/` — schema.sql and seed.sql
- `github.com/lib/pq` from go.mod

### File: `.vrooli/service.json`
**Change**: Removed postgres resource dependency
**Reason**: SQLite is embedded, no external database resource needed

### File: `api/go.mod`
**Change**: Removed `lib/pq`, added `modernc.org/sqlite v1.34.5`, promoted `github.com/google/uuid` to direct dependency
**Reason**: SQLite driver and UUID generation in Go layer

### File: `api/infrastructure/sqlite/schema.sql`
**Added**: SQLite-compatible schema for all 7 tables
**Details**: TEXT with CHECK constraints instead of ENUMs, TEXT for UUIDs, INTEGER for booleans, no triggers (updated_at in Go layer)

### File: `api/infrastructure/sqlite/sqlite.go`
**Added**: Connection setup with go:embed schema, storage resolver for DB path, WAL/FK/busy_timeout pragmas, MaxOpenConns(1)

### File: `api/infrastructure/sqlite/reference_repo.go`
**Added**: reference.Repository implementation (6 methods) with ? placeholders, Go UUID generation, manual updated_at

### File: `api/infrastructure/sqlite/skill_repo.go`
**Added**: skill.Repository implementation (7 methods)

### File: `api/infrastructure/sqlite/structural_expectations_repo.go`
**Added**: expectation.StructuralRepository implementation (5 methods) — NEW, never existed for Postgres

### File: `api/infrastructure/sqlite/cli_assertions_repo.go`
**Added**: expectation.CLIRepository implementation (5 methods) — NEW, never existed for Postgres. JSON marshal/unmarshal for expected_value field.

### File: `api/main.go`
**Change**: Switched from Postgres to SQLite wiring, wired all 4 repos including expectation handlers (previously commented out)

### Test Files Added
- `infrastructure/sqlite/test_helpers_test.go` — setupTestDB using t.TempDir()
- `infrastructure/sqlite/reference_repo_test.go` — CRUD, not found, unique slug, pagination
- `infrastructure/sqlite/skill_repo_test.go` — CRUD, disconnect by ref+skill, unique constraint, cascade delete
- `infrastructure/sqlite/structural_expectations_repo_test.go` — CRUD, content snippet, delete by connection
- `infrastructure/sqlite/cli_assertions_repo_test.go` — CRUD, nil expected value, delete by connection, not found

## Verification

### Automated Tests
- [x] All existing tests pass (domain, handlers, config, validation — 12 packages)
- [x] New SQLite integration tests pass (infrastructure/sqlite — 14 test functions)
- [x] `go build ./...` passes clean

### Verification Checks
- [x] `grep -r "lib/pq" api/` returns nothing
- [x] `grep -r "postgres" api/infrastructure/` returns nothing
- [x] All 4 repo interfaces have SQLite implementations
- [x] Expectation handlers wired in main.go
- [x] Schema creates all 7 tables with CREATE TABLE IF NOT EXISTS

## Follow-up
- None required — migration is complete and self-contained
