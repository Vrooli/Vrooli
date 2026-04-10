# Report Domain Implementation Plan

## Purpose

Implement the report domain for the Development Toolchain Validator (DTV), transforming granular validation results into actionable aggregate reports for the meta-optimization team.

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read implementation-plan-authoring
```

**Codebase familiarity:**
- `scenarios/development-toolchain-validator/api/domain/report/doc.go` — domain intent and boundaries
- `scenarios/development-toolchain-validator/api/domain/validation/model.go` — validation result types this domain aggregates
- `scenarios/development-toolchain-validator/api/domain/skill/model.go` — Connection and DriftStatus types
- `scenarios/development-toolchain-validator/api/domain/expectation/model.go` — StructuralExpectation and CLIAssertion types
- `scenarios/development-toolchain-validator/api/handlers/errors.go` — error mapping patterns
- `scenarios/development-toolchain-validator/api/main.go` — wiring pattern (repo → service → handler → routes)
- `scenarios/development-toolchain-validator/api/infrastructure/sqlite/schema.sql` — existing schema (7 tables)

## Problem Statement

The report domain (`domain/report/`) is a documented stub (`doc.go` only). The meta-optimization team needs four aggregate report types that transform raw per-assertion validation results into decision-quality signals:

1. **Conflicts** — Cross-skill contradictions where skills connected to the same reference have incompatible structural expectations or overlapping CLI assertions producing different expected values
2. **Drift** — Aggregation of skill content-hash drift across all connections (drift detection exists per-connection in the skill domain; this aggregates it)
3. **Maturity** — Skill maturity scoring based on expectation coverage (few/no expectations = low maturity)
4. **Tool Baselines** — Tool accuracy regression checks running CLI assertions for known tools against references and comparing to stored baselines

## Scope

**Acceptance patterns:**
- `acceptance_allow`: `["scenarios/development-toolchain-validator/api/**"]`

### In Scope
- `domain/report/model.go` — report data models for all four report types
- `domain/report/service.go` — report generation logic aggregating from existing domains
- `domain/report/repository.go` — repository interface (for baseline storage)
- `domain/report/errors.go` — domain-specific sentinel errors
- `handlers/report.go` — HTTP handlers for report endpoints
- `infrastructure/sqlite/report_repo.go` — SQLite repository for baseline storage
- `infrastructure/sqlite/schema.sql` — schema additions for tool baselines table
- `main.go` — wire report service and handler into the server
- `handlers/errors.go` — add report-domain error mappings
- `infrastructure/promptmanager/client.go` — SkillMetadataFetcher implementation

### Out of Scope
- Modifying the validation engine itself
- Report caching/invalidation (future optimization)
- UI/frontend for reports
- CLI integration for reports
- Modifying existing domain interfaces

## Current Technical Context

### Existing Domain Pattern
Each domain follows: `model.go` (types) + `service.go` (business logic) + `repository.go` (storage interface). Handlers live in `handlers/` and register routes via `RegisterRoutes(r *mux.Router)`. Services are injected into handlers, handlers into the server.

### Key Dependencies
- **validation domain**: `StructuralChecker` and `CLIExecutor` for running validations; `ConnectionValidationResult` and `ReferenceValidationReport` for result types. Note: validation has NO service layer — only standalone checker/executor types. Report service will take these as constructor dependencies.
- **skill domain**: `Connection` model (has `SkillContentHash`, `SkillVersion`), `CheckDrift` method, `SkillRepository` for listing connections
- **expectation domain**: `StructuralExpectation` and `CLIAssertion` for counting expectations per skill; `StructuralRepository` and `CLIRepository` for listing expectations by connection
- **reference domain**: `Reference` model for reference metadata

### Database
- SQLite with WAL mode, single connection (max open conns = 1)
- Schema in `infrastructure/sqlite/schema.sql`
- Existing tables: `reference_scenarios`, `skill_connections`, `structural_expectations`, `cli_assertions`, `validation_runs`, `structural_results`, `cli_results`
- New table needed: `tool_baselines` for stored baseline snapshots

### Dependency Note
The `depends_on: fix/dtv-validation-api` in spec.json references an item whose spec no longer exists. The validation domain's checkers, executors, and models DO exist in the codebase. This dependency can be considered resolved — reports build on existing validation types.

## Target End State

Four HTTP endpoints returning structured JSON reports:
- `GET /api/v1/reports/conflicts?reference_id=<optional>` — cross-skill contradictions
- `GET /api/v1/reports/drift?reference_id=<optional>` — aggregated drift status
- `GET /api/v1/reports/maturity?reference_id=<optional>` — skill maturity scores
- `GET /api/v1/reports/tool-baselines?reference_id=<optional>` — tool regression checks

Each endpoint returns a structured JSON response following existing patterns (`writeJSON`/`WriteStructuredError`).

## Implementation Strategy

### Phase 1: Models and Interfaces
1. Create `domain/report/model.go` with types for all four report types
2. Create `domain/report/repository.go` with `BaselineRepository` interface
3. Create `domain/report/errors.go` with domain-specific sentinel errors
4. Define `SkillMetadataFetcher` interface in `domain/report/service.go` (or separate file)

### Phase 2: Report Service
1. Create `domain/report/service.go` with `ReportService`
2. Service dependencies: `skill.Repository`, `expectation.StructuralRepository`, `expectation.CLIRepository`, `BaselineRepository`, `SkillMetadataFetcher`
3. Implement each report method:
   - `GenerateConflicts(ctx, opts)` — for each reference, get all connections and their expectations, pairwise compare structural expectations by pattern equality and CLI assertions by (command, json_path) equality, flag pairs with differing required/expected_value *(settled: round 1, d1 — pairwise comparison)*
   - `GenerateDrift(ctx, opts)` — list all connections, fetch current version/hash via SkillMetadataFetcher, use skill.CheckDrift for each *(settled: round 1, d3 — SkillMetadataFetcher interface)*
   - `GenerateMaturity(ctx, opts)` — count structural + CLI expectations per connection, assign tier: 0→none(0.0), 1-2→low(0.25), 3-5→medium(0.5), 6-10→high(0.75), 10+→comprehensive(1.0) *(settled: round 1, d2 — count-based tiers)*
   - `GenerateToolBaselines(ctx, opts)` — run CLI assertions for tool connections, compare to stored baselines

### Phase 3: Storage and Infrastructure
1. Add `tool_baselines` table to `schema.sql`
2. Implement `infrastructure/sqlite/report_repo.go` (BaselineRepository)
3. Implement `infrastructure/promptmanager/client.go` (SkillMetadataFetcher)

### Phase 4: HTTP Handlers and Wiring
1. Create `handlers/report.go` with `ReportHandler`
2. Add report error mappings to `handlers/errors.go`
3. Wire into `main.go`: create repos → fetcher → service → handler → register routes

### Phase 5: Testing
1. Unit tests for conflict detection logic (mock repositories)
2. Unit tests for maturity scoring
3. Unit tests for drift aggregation (mock SkillMetadataFetcher)
4. Integration tests for baseline storage round-trip
5. Integration tests for HTTP endpoints

## Contract Decisions

### API Endpoints
- URL pattern: `GET /api/v1/reports/{report-type}`
- Optional query param: `reference_id` to filter to a single reference
- Response follows existing `writeJSON` pattern — raw report body, no envelope wrapper
- Errors follow existing `WriteStructuredError` pattern with `ErrorResponse` struct

### Conflict Detection (settled, round 1)
- Pairwise comparison within each reference
- Structural conflicts: same `pattern` on two connections with different `required` or `expected_content`
- CLI conflicts: same `(command, json_path)` on two connections with different `(operator, expected_value)`

### Maturity Scoring (settled, round 1)
- Count-based tiers using total expectations (structural + CLI) per connection:
  - 0 → `none` (0.0)
  - 1-2 → `low` (0.25)
  - 3-5 → `medium` (0.5)
  - 6-10 → `high` (0.75)
  - 10+ → `comprehensive` (1.0)

### Drift Detection (settled, round 1)
- SkillMetadataFetcher interface abstracts external skill version/hash lookup
- Implementation in `infrastructure/promptmanager/` calls prompt-manager to get current metadata
- Each connection's stored hash compared against fetched current hash

### Tool Baselines
<!-- TBD — pending decisions on baseline schema, snapshot semantics, and tool list -->

## Testing Plan

- Unit tests for each report generation method (mock repositories + mock SkillMetadataFetcher)
- Unit tests for conflict detection algorithm (pairwise comparison edge cases)
- Unit tests for maturity scoring formula (boundary values: 0, 1, 2, 3, 5, 6, 10, 11)
- Integration tests for baseline repository (SQLite round-trip)
- Integration tests for HTTP endpoints (full stack with test DB)
- Test fixtures covering: no data, single reference, multiple references with conflicts, drift/no-drift, mixed maturity levels

## Rollout/Validation Checklist

- [ ] `go build ./...` succeeds
- [ ] `go test ./... -timeout 300s` passes
- [ ] `gofumpt -l .` reports no unformatted files
- [ ] `golangci-lint run` passes
- [ ] All four endpoints return valid JSON for empty database
- [ ] All four endpoints return correct results for seeded test data
- [ ] Drift report handles SkillMetadataFetcher errors gracefully

## Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Validation domain has no service layer (only checkers) | Report service needs to invoke validation directly | Take checker/executor as constructor dependencies; no new service layer needed |
| Drift check requires current skill version/hash from prompt-manager | External dependency at runtime | SkillMetadataFetcher interface; mock in tests; graceful error handling per-connection |
| Conflict detection O(n²) per reference | Could be slow for many expectations | n is expected to be small (10s per reference); start with brute-force, optimize only if measured slow |
| Tool baseline storage schema not defined | Blocks tool-baselines report | Design in round 2 |
| SQLite migration (fix/dtv-sqlite-migration) not yet complete | Schema additions depend on existing tables | Report schema additions are additive (new table only); can be added independently |

## Non-goals / Prohibited Patterns

- Do NOT modify existing domain services or models
- Do NOT add report caching in this iteration
- Do NOT add authentication/authorization (not present in existing endpoints)
- Do NOT create `lib/` folders
- Do NOT add a validation service layer — use checker/executor directly

## Definition of Done

1. All four report types implemented with HTTP endpoints
2. All tests pass (`go test ./... -timeout 300s`)
3. Code formatted (`gofumpt`) and linted (`golangci-lint run`)
4. Report domain follows existing patterns (model/service/repository/handler)
5. SkillMetadataFetcher interface defined and implemented
6. Tool baselines table added to schema
7. Wired into main.go following existing dependency injection pattern
