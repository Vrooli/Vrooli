# Implementation Plan: Resolve standards warnings in swarm-manager

## Purpose

Eliminate all high-severity standards violations and reduce total warnings to ≤5 in the swarm-manager scenario. This unblocks readiness gates and prevents standards-related test failures.

## Required Reading

```bash
prompt-manager skill read scientific-debugging
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read test platform-package-consumption-audit
```

## Problem Statement

The GCT review (job `14ca47e3-0748-4480-931e-cf7970a3f481`) reports 10 standards warnings including high-severity findings:

1. **`api/eventlog_setup.go`** — Uses `sql.Open("sqlite", dsn)` directly instead of the shared `database.Connect()` from `api-core`. This bypasses retry logic, environment-based configuration, and pool management that the standard provides.
2. **`api/internal/dispatch/`** — No `*_test.go` file. Package defines `Invalidator` and `NodeDispatcher` interfaces.
3. **`api/internal/logging/`** — No `*_test.go` file. Package provides `Init()`, `FromContext()`, `WithLogger()`, `With()`.
4. **`cli/cmd/`** — No `*_test.go` file. Contains `migrate_workshop.go` (626 lines) with complex migration logic.
5. **`cli/go.mod`** — Uses Go 1.22 while `api/go.mod` uses Go 1.24; no workspace mode configured.

## Scope

**In scope:**
- Replace `sql.Open` with `database.Connect` in `eventlog_setup.go`
- Add test files for `dispatch`, `logging`, and `cli/cmd` packages
- Fix `cli/go.mod` Go version and workspace configuration

**Out of scope:**
- Refactoring the packages themselves (only adding tests)
- Changing the event log schema or migration logic
- Adding tests beyond what's needed to satisfy standards

## Current Technical Context

### Key Files
| File | Role | Issue |
|------|------|-------|
| `api/eventlog_setup.go` | Initializes SQLite event log DB | Direct `sql.Open` (line 50) |
| `api/internal/dispatch/dispatch.go` | Interface definitions for graph event dispatching | No test file |
| `api/internal/logging/logging.go` | Structured logging with context propagation | No test file |
| `cli/cmd/migrate_workshop.go` | One-time migration from old to new workshop format | No test file |
| `cli/go.mod` | CLI module definition | Go 1.22, no workspace mode |
| `packages/api-core/database/connect.go` | Shared DB connection with retry + env config | Supports `DriverSQLite` with `SQLITE_PATH` env var |

### api-core database.Connect capabilities
- Supports `sqlite` driver natively (line 53 of connect.go)
- Reads `SQLITE_PATH` or `SQLITE_DB` from environment
- Accepts explicit `DSN` to bypass env-based config
- Configures pool settings (`MaxOpenConns`, `MaxIdleConns`)
- Provides retry with exponential backoff

## Target End State

- `eventlog_setup.go` uses `database.Connect(ctx, database.Config{Driver: "sqlite", DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1})` instead of `sql.Open`
- Each of `dispatch`, `logging`, and `cli/cmd` has a `*_test.go` file with meaningful tests
- `cli/go.mod` upgraded to Go 1.24 and workspace configuration resolved
- GCT review shows 0 high-severity violations and ≤5 total warnings

## Implementation Strategy

### Phase 1: Replace sql.Open in eventlog_setup.go
1. Replace `sql.Open("sqlite", dsn)` with `database.Connect(ctx, cfg)` using explicit DSN
2. Remove manual `SetMaxOpenConns`/`SetMaxIdleConns` calls (handled by Connect config)
3. Pass a context with timeout to Connect
4. Verify the event log still initializes correctly

### Phase 2: Add missing test files
1. `api/internal/dispatch/dispatch_test.go` — Test that types satisfy interfaces (compile-time checks)
2. `api/internal/logging/logging_test.go` — Test `Init()`, `FromContext()`, `WithLogger()`, `With()` round-trip
3. `cli/cmd/migrate_workshop_test.go` — Test key conversion helpers (`clarifyToRound`, `suggestToRound`, `computeReadiness`, `mapSuggestionStatus`, `answerString`, file helpers)

### Phase 3: Fix cli/go.mod
1. Update Go version from 1.22 to 1.24
2. Investigate and resolve workspace mode configuration

## Contract Decisions

<!-- TBD — pending workshop decisions on database.Connect migration approach and cli/go.mod workspace strategy -->

## Testing Plan

- Run `go test ./...` from both `api/` and `cli/` directories
- Run `git-control-tower review-run swarm-manager --json` to verify standards score
- Verify high-severity violations = 0, total warnings ≤ 5

## Rollout/Validation Checklist

- [ ] `eventlog_setup.go` uses `database.Connect` instead of `sql.Open`
- [ ] `dispatch_test.go` exists and passes
- [ ] `logging_test.go` exists and passes
- [ ] `migrate_workshop_test.go` exists and passes
- [ ] `cli/go.mod` Go version updated
- [ ] `go test ./...` passes in both api/ and cli/
- [ ] GCT review shows 0 high violations, ≤5 warnings

## Risks + Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `database.Connect` retry logic adds latency on SQLite init | Low | Use single-attempt retry config or pass DSN directly (no env lookup needed) |
| CLI go.mod upgrade breaks existing replace directives | Medium | Test `go build ./...` after change |
| Missing test coverage for cli/cmd may require exported helpers | Medium | Test exported functions only; use table-driven tests |

## Non-goals / Prohibited Patterns

- Do not refactor existing package APIs
- Do not add unnecessary abstractions
- Do not change the SQLite schema or migration logic

## Definition of Done

1. `git-control-tower review-run swarm-manager --json` shows 0 high-severity standards violations
2. Total standards warnings ≤ 5
3. `go test ./...` passes in both `api/` and `cli/` directories
4. No regressions in existing functionality
