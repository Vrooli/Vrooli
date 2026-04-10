# Implementation Plan: Resolve standards warnings in swarm-manager

## Greenfield Declaration

This is a greenfield fix — no backward-compatibility shims, migration bridges, or deprecated API preservation. All changes target the current codebase state directly.

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
- Upgrade `cli/go.mod` Go version from 1.22 to 1.24

**Out of scope:**
- Refactoring the packages themselves (only adding tests)
- Changing the event log schema or migration logic
- Adding tests beyond what's needed to satisfy standards
- Creating a `go.work` workspace file (deferred — replace directives are sufficient)

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
- `cli/go.mod` upgraded to Go 1.24; existing replace directives retained
- GCT review shows 0 high-severity violations and ≤5 total warnings

## Contract Decisions

### D1: database.Connect migration strategy — Pass existing DSN directly

**Decision:** Keep `resolveEventDBPath()` as-is. Pass its output as `Config.DSN` to `database.Connect`. This preserves all existing path resolution and WAL pragma logic while satisfying the standards requirement to use the shared connection helper. Lowest-risk change with minimal behavioral differences.

### D2: Test depth for dispatch and logging — Compile-check + functional split

**Decision:** `dispatch/` only defines interfaces with no concrete implementation — tests will verify that mock types satisfy the interfaces (compile-time guarantee). `logging/` has real behavior worth testing — tests will cover context round-trip via `Init()`, `FromContext()`, `WithLogger()`, and `With()` functions. This matches the actual testable surface of each package.

### D3: cli/go.mod workspace configuration — Version upgrade only

**Decision:** Upgrade `cli/go.mod` from Go 1.22 to Go 1.24 and keep existing replace directives. A `go.work` file is not being added at this time — the replace directives are sufficient and a workspace file would add complexity without clear benefit for this two-module scenario.

## Implementation Strategy

### Phase 1: Replace sql.Open in eventlog_setup.go
1. Replace `sql.Open("sqlite", dsn)` with `database.Connect(ctx, cfg)` using explicit DSN passthrough
2. Remove manual `SetMaxOpenConns`/`SetMaxIdleConns` calls (handled by Connect config)
3. Pass a context with timeout to Connect
4. Verify the event log still initializes correctly

### Phase 2: Add missing test files
1. `api/internal/dispatch/dispatch_test.go` — Compile-time interface satisfaction checks (mock types implement `Invalidator` and `NodeDispatcher`)
2. `api/internal/logging/logging_test.go` — Functional tests: `Init()` configures slog, `FromContext()`/`WithLogger()` context round-trip, `With()` returns child logger
3. `cli/cmd/migrate_workshop_test.go` — Table-driven tests for key conversion helpers (`clarifyToRound`, `suggestToRound`, `computeReadiness`, `mapSuggestionStatus`, `answerString`, file helpers)

### Phase 3: Fix cli/go.mod
1. Update Go version from 1.22 to 1.24
2. Run `go mod tidy` to reconcile dependencies
3. Verify `go build ./...` succeeds with the updated version

### Phase 4: Validate and restart
1. Run `go test ./...` from both `api/` and `cli/` directories
2. Run `gofumpt -w .` on all new/modified files
3. Run `golangci-lint run` to catch any remaining issues
4. Run `vrooli scenario restart swarm-manager` to validate the scenario starts cleanly with changes applied

## Testing Plan

- Run `go test ./...` from both `api/` and `cli/` directories
- Run `git-control-tower review-run swarm-manager --json` to verify standards score
- Verify high-severity violations = 0, total warnings ≤ 5

## Rollout/Validation Checklist

- [ ] `eventlog_setup.go` uses `database.Connect` instead of `sql.Open`
- [ ] `dispatch_test.go` exists and passes
- [ ] `logging_test.go` exists and passes
- [ ] `migrate_workshop_test.go` exists and passes
- [ ] `cli/go.mod` Go version updated to 1.24
- [ ] `go test ./...` passes in both api/ and cli/
- [ ] `gofumpt` and `golangci-lint` pass on all new/modified files
- [ ] GCT review shows 0 high violations, ≤5 warnings
- [ ] `vrooli scenario restart swarm-manager` completes successfully

## Risks + Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `database.Connect` retry logic adds latency on SQLite init | Low | Use single-attempt retry config or pass DSN directly (no env lookup needed) |
| CLI go.mod upgrade breaks existing replace directives | Medium | Test `go build ./...` after change; run `go mod tidy` |
| Missing test coverage for cli/cmd may require exported helpers | Medium | Test exported functions only; use table-driven tests |

## Non-goals / Prohibited Patterns

- Do not refactor existing package APIs
- Do not add unnecessary abstractions
- Do not change the SQLite schema or migration logic
- Do not add backward-compatibility shims or migration bridges
- Do not create a `go.work` file (deferred per D3 decision)

## Definition of Done

1. `git-control-tower review-run swarm-manager --json` shows 0 high-severity standards violations
2. Total standards warnings ≤ 5
3. `go test ./...` passes in both `api/` and `cli/` directories
4. No regressions in existing functionality
5. `vrooli scenario restart swarm-manager` completes successfully
