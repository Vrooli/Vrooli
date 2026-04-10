# development-toolchain-validator Storage Architecture Audit

## Last Updated
2026-03-11

## Resource Configuration Status
- [x] postgres declared in service.json
- [x] schema field uses scenario slug (`development-toolchain-validator`)
- [ ] initialization files referenced in service.json (schema.sql exists but not declared)
- [ ] redis/qdrant properly configured (not used)

## Connection Pattern Status
- [x] Environment variables used (via api-core/database)
- [x] Connection retry with exponential backoff (via api-core/database)
- [x] Connection pool configured (via api-core/database defaults)
- [x] Health check implemented (via api-core/health with DB check)

## Schema Status
- [x] schema.sql exists and is idempotent (IF NOT EXISTS patterns)
- [x] Tables use proper constraints and indexes
- [x] Greenfield default applied (clean schema, no migrations)
- [ ] Brownfield migrations documented (not needed)

## Abstraction Status
- [x] Repository interfaces defined (`domain/reference/repository.go`)
- [x] Business logic uses interfaces, not direct DB (`domain/reference/service.go`)
- [ ] Multiple storage backends abstracted (only PostgreSQL implemented)

## Filesystem Status
- [ ] Runtime filesystem writes go through `api-core/storage` (not yet needed)
- [x] Deploy directory treated as disposable
- [ ] Atomic writes used for persisted files (not yet needed)

## Schema Design Summary

### Tables Created
1. **reference_scenarios** - Stores registered reference scenarios with template associations
2. **skill_connections** - Links prompt-manager skills to references with version tracking
3. **structural_expectations** - Defines expected folders/files/content patterns
4. **cli_assertions** - Defines CLI tool commands and JSONPath assertions
5. **validation_runs** - Tracks validation execution history
6. **structural_results** - Stores per-expectation validation results
7. **cli_results** - Stores per-assertion validation results

### Custom Types
- `expectation_type` ENUM: folder, file, content_snippet
- `assertion_operator` ENUM: eq, neq, gt, gte, lt, lte, exists, contains, matches, between
- `validation_status` ENUM: pass, fail, error, skip

### Indexes
- All foreign keys are indexed for join performance
- `slug` has unique index for fast lookups
- `template` indexed for filtering
- `started_at DESC` for recent validation queries

## Architecture Patterns Applied

### Repository Pattern
```
Handler → Service → Repository → PostgreSQL
   ↓         ↓          ↓
 HTTP     Business   Storage
Parsing    Logic    Abstraction
```

### Domain-Driven Structure
```
api/
├── domain/           # Domain models and interfaces
│   ├── reference/    # Reference scenario domain
│   ├── skill/        # Skill connection domain (placeholder)
│   ├── validation/   # Validation engine domain (placeholder)
│   └── report/       # Report generation domain (placeholder)
├── handlers/         # HTTP request handling
└── infrastructure/   # External system implementations
    └── postgres/     # PostgreSQL repository implementations
```

## Issues Found
1. No initialization declaration in service.json (schema.sql exists but not configured)
2. Missing repository implementations for skill, validation, report domains
3. No unit tests for repository layer

## Priority Fixes
1. Add initialization declaration to service.json for schema.sql
2. Implement skill connection repository (next P0 target)
3. Add repository unit tests with testcontainers

## Notes
- Using api-core/database for connection management provides automatic retry, pooling, and environment variable handling
- Schema is designed to be idempotent for safe re-runs
- All domain entities use UUID primary keys for distributed compatibility
