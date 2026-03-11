# Integration Boundaries (SEAMS)

## External Seams

### prompt-manager API (HTTP Client)
- **Boundary**: `api/infrastructure/promptmanager/client.go` (planned)
- **What crosses**: Skill metadata, version history, content hashes
- **Contract**: REST API at `PROMPT_MANAGER_API_URL`
- **Testability**: Interface-based client, mockable in unit tests
- **Failure mode**: If prompt-manager is down, drift detection and skill fetching fail. Connections can still be managed with cached version data.
- **Status**: Not yet implemented

### Scenario CLIs (Subprocess Execution)
- **Boundary**: `api/domain/validation/cli_executor.go` (planned)
- **What crosses**: CLI commands, JSON stdout
- **Contract**: Commands produce valid JSON on stdout, exit code 0 on success
- **Testability**: Executor interface, mockable in unit tests. Integration tests run real commands.
- **Failure modes**: Command not found, timeout, non-JSON output, non-zero exit code
- **Status**: Not yet implemented

### Reference Scenario Filesystem (os.Stat / filepath.Glob)
- **Boundary**: `api/domain/validation/structural_checker.go` (planned)
- **What crosses**: File/folder existence checks, file content reads
- **Contract**: Standard POSIX filesystem operations
- **Testability**: Use temp directories in tests with known structures
- **Failure modes**: Permission errors, race conditions with concurrent modifications
- **Status**: Not yet implemented

### PostgreSQL (Database)
- **Boundary**: [CODE: api/infrastructure/postgres/] (repository implementations)
- **What crosses**: CRUD for all entities, validation history
- **Contract**: PostgreSQL 15+ with schema from [CODE: initialization/postgres/schema.sql]
- **Testability**: Testcontainers with postgres:15-alpine
- **Failure modes**: Connection refused, schema mismatch, constraint violations
- **Status**: ✅ Implemented for reference domain
- **Implementation**: [CODE: api/infrastructure/postgres/reference_repo.go#ReferenceRepository]

## Internal Seams

### Registry → Validation Engine
- Registry provides connection and expectation data
- Validation engine operates on this data without modifying it
- Clear separation: registry is CRUD, validation is read-only execution

### Validation Engine → Report Generator
- Validation produces per-expectation results
- Report generator aggregates into overlaps, conflicts, summaries
- Overlap/conflict detection is a separate analysis pass, not inline with validation

### API Handlers → Domain Services
- Handlers parse HTTP, call domain services, serialize responses
- Domain services contain all business logic
- Handlers do not access the database directly
- **Status**: ✅ Implemented for reference domain
- **Handler**: [CODE: api/handlers/reference.go#ReferenceHandler]
- **Service**: [CODE: api/domain/reference/service.go#Service]

## Architecture Alignment Update (2026-03-11)

### Implemented Structure

| Path | Description | Status |
|------|-------------|--------|
| [CODE: api/main.go] | Server wiring and startup | ✅ |
| [CODE: api/domain/reference/model.go] | Reference domain entities | ✅ |
| [CODE: api/domain/reference/repository.go] | Repository interface | ✅ |
| [CODE: api/domain/reference/service.go] | Business logic | ✅ |
| [CODE: api/handlers/reference.go] | Reference CRUD endpoints | ✅ |
| [CODE: api/infrastructure/postgres/reference_repo.go] | PostgreSQL repository | ✅ |
| `api/domain/skill/` | Skill connection domain | Placeholder |
| `api/domain/validation/` | Validation engine domain | Placeholder |
| `api/domain/report/` | Report generation domain | Placeholder |
| `api/infrastructure/promptmanager/` | prompt-manager client | Placeholder |

### Responsibility Flow
```
HTTP Request
    ↓
handlers/reference.go       [Parse HTTP, validate input, serialize output]
    ↓
domain/reference/service.go [Business rules, validation, coordination]
    ↓
domain/reference/repository.go [Interface - storage contract]
    ↓
infrastructure/postgres/reference_repo.go [PostgreSQL implementation]
    ↓
PostgreSQL Database
```

### Key Design Decisions
1. **Domain-driven structure**: Code organized by domain concept (reference, skill, validation) not by technical layer
2. **Repository pattern**: Storage abstracted behind interfaces for testability and future backend swaps
3. **Service layer**: Business logic centralized in services, not scattered in handlers or repositories
4. **Handler isolation**: HTTP handlers only parse requests and serialize responses, no business logic

## Testing Seams (2026-03-11)

### Mock Repository Seam
- **Boundary**: [CODE: api/internal/mocks/repository.go#MockRepository]
- **Purpose**: Enables unit testing of service and handler layers without database
- **Pattern**: Builder pattern with `WithReference()`, `WithCreateError()`, etc.
- **Call Tracking**: `CreateCallCount()`, `DeleteCallCount()` for assertions
- **Thread Safety**: Uses sync.RWMutex for parallel test execution

### Test Data Factory Seam
- **Boundary**: [CODE: api/internal/testutil/fixtures.go]
- **Purpose**: Consistent test data creation across all tests
- **Factories**: `ReferenceFactory`, `CreateInputFactory`
- **Pattern**: Builder with `Build()` returning configured instances

### HTTP Test Helpers Seam
- **Boundary**: [CODE: api/internal/testutil/helpers.go]
- **Purpose**: Standardize HTTP test assertions and request creation
- **Functions**: `AssertStatus`, `AssertJSON`, `MakeJSONRequest`, etc.
- **Benefits**: Reduces test boilerplate, consistent error messages

### External Test Package Pattern
- **Boundary**: `domain/reference/service_test.go` uses `package reference_test`
- **Purpose**: Avoids import cycles when tests import mocks from `internal/mocks`
- **Tradeoff**: Cannot test unexported functions directly (use interfaces instead)

## Error Semantics (2026-03-11)

This section documents the error handling architecture and recovery paths.

### Error Categories

Errors are classified into distinct categories with specific recovery strategies:

| Category | HTTP Status | Recovery Strategy | Example |
|----------|-------------|-------------------|---------|
| `validation` | 400 | User corrects input and retries | Invalid slug format |
| `not_found` | 404 | Check identifier, resource may be deleted | Reference not found |
| `conflict` | 409 | Use different identifier, check existing | Slug already exists |
| `database` | 500/503 | Retry (if transient) or contact support | Connection failed |
| `internal` | 500 | Retry may help, log for debugging | Unexpected error |
| `dependency` | 502/503 | Retry with backoff | External service down |

### Error Flow

```
Domain Layer (service.go)
    ↓ Sentinel errors (ErrNotFound, ErrInvalidSlug, etc.)
Handler Layer (reference.go, errors.go)
    ↓ MapDomainError() converts to structured API errors
    ↓ logError() logs with appropriate severity
HTTP Response
    ↓ Status code from ToHTTPStatus()
    ↓ JSON body with message, code, recovery guidance
```

### Implementation Files

| File | Purpose |
|------|---------|
| [CODE: api/internal/errors/errors.go] | Structured error types with categories |
| [CODE: api/handlers/errors.go] | Domain-to-API error mapping |
| [CODE: api/domain/reference/service.go] | Domain sentinel errors |

### Error Response Format

```json
{
  "error": "Human-readable message",
  "code": "machine_readable_code",
  "category": "validation",
  "details": {"field": "slug", "provided": "Bad-Slug"},
  "recovery": "Check your input and try again"
}
```

### Logging Strategy

- **Critical**: System health affected, requires immediate attention
- **High**: Significant errors (database failures), requires investigation
- **Medium**: Notable issues, warn level
- **Low**: Expected validation errors, not logged by default

### Recovery Path Design Principles

1. **User-actionable guidance**: Each error tells the user what to do next
2. **Machine-readable codes**: Agents can programmatically handle errors
3. **Transient vs permanent**: Errors indicate whether retry is likely to help
4. **No internal leakage**: Stack traces and internal details stay in logs only

### Failure Modes by Component

| Component | Failure Mode | Category | Transient? |
|-----------|--------------|----------|------------|
| JSON decode | Invalid body | validation | No |
| Slug validation | Format/length | validation | No |
| Path validation | Not exists | validation | No |
| Slug uniqueness | Conflict | conflict | No |
| Repository lookup | Not found | not_found | No |
| Database query | Connection | database | Usually |
| Database write | Constraint | database | No |

## Change Axes (2026-03-11)

This section identifies the primary ways this scenario is likely to evolve, enabling future agents to make changes in predictable, localized places.

### Primary Axes of Change

| Axis | Volatility | Extension Point | Files Affected |
|------|------------|-----------------|----------------|
| New domain modules | High | `api/domain/<module>/` | New folder per module |
| CLI tool assertions | High | Operators in handler/validator | Assertion evaluator (planned) |
| External integrations | Medium | `api/infrastructure/<client>/` | One client per service |
| Validation rules | Medium | `domain/<module>/service.go` | Service layer only |
| Error categories | Low | `internal/errors/errors.go` | Single file + handler mapping |
| Storage backends | Low | `infrastructure/<backend>/` | Repository implementations |

### Axis: New Domain Modules (HIGH VOLATILITY)

The scenario will expand from Reference-only to include Skill, Validation, and Report domains per the PRD.

**Where to add new domains:**
```
api/domain/<module>/
├── model.go      # Entity definitions
├── repository.go # Storage interface
├── service.go    # Business logic
└── <module>_test.go
```

**Propagation requirements:**
1. Add handler in `api/handlers/<module>.go`
2. Add PostgreSQL repo in `api/infrastructure/postgres/<module>_repo.go`
3. Register routes in `api/main.go`
4. Add CLI commands if needed

**Design guardrails:**
- Each domain owns its entities; no cross-domain entity imports
- Services depend on repository interfaces, not implementations
- Handlers only parse HTTP and delegate to services

### Axis: CLI Tool Assertions (HIGH VOLATILITY)

Adding new assertion operators (eq, neq, contains, etc.) for CLI tool validation.

**Current extension point:**
- Operators defined in PRD Appendix
- Implementation will be in `api/domain/validation/assertion_evaluator.go` (planned)

**To add a new operator:**
1. Define operator semantics in PRD Appendix
2. Add case in assertion evaluator switch statement
3. Add test cases covering edge cases

**Design notes:**
- Operators must work on JSON primitive types (string, number, bool)
- Complex operators (between, matches) take structured values
- Unknown operators should fail validation, not evaluation

### Axis: External Integrations (MEDIUM VOLATILITY)

New external service integrations (prompt-manager API, scenario CLIs).

**Extension point:** `api/infrastructure/<client>/client.go`

**Pattern:**
```go
type PromptManagerClient interface {
    GetSkill(ctx context.Context, id string) (*Skill, error)
    GetSkillVersion(ctx context.Context, id, version string) (*SkillVersion, error)
}
```

**Design guardrails:**
- Clients defined as interfaces in domain layer
- Implementations in infrastructure layer
- Mock implementations in `internal/mocks/`
- HTTP clients should handle retries, timeouts, circuit breaking

### Axis: Validation Rules (MEDIUM VOLATILITY)

Business rules for validating inputs (slug format, path existence, etc.).

**Extension point:** Service layer methods with `isValid*` or `validate*` prefixes.

**Current validators:**
- [CODE: api/domain/reference/service.go#isValidSlug] - Slug format/length
- Path existence check in Create/Update

**To add new validation:**
1. Add `isValid*` or `validate*` method to service
2. Call in Create/Update as appropriate
3. Return domain-specific sentinel error
4. Map to API error in `handlers/errors.go`

### Axis: Error Categories (LOW VOLATILITY)

Adding new error categories for different recovery strategies.

**Extension point:** [CODE: api/internal/errors/errors.go]

**To add a new category:**
1. Add constant to `Category` enum
2. Document recovery strategy in comment
3. Add HTTP status mapping in `ToHTTPStatus()`
4. Add constructor function
5. Update `handlers/errors.go` if domain-specific

**Stability notes:**
- Existing categories are well-defined and stable
- New categories should have distinct recovery strategies
- Changing category semantics affects client retry logic

### Stable Cores vs Volatile Edges

**Stable (change rarely):**
- Repository interface patterns
- Error category definitions
- HTTP handler structure
- Configuration loading patterns

**Volatile (expect changes):**
- Domain-specific business rules
- Assertion operators and their logic
- External API client implementations
- UI components and routes

## Decision Points (2026-03-11)

This section catalogs where the system makes decisions, making them easy to find, understand, and test.

### Decision: Slug Format Validation

**Location:** [CODE: api/domain/reference/service.go#isValidSlug]

**What it decides:** Whether a slug is acceptable for reference identification.

**Criteria:**
- Length: Between SlugMinLength and SlugMaxLength (configurable)
- Format: Regex `^[a-z0-9][a-z0-9-]*[a-z0-9]$` (not configurable)

**Rationale:**
- Lowercase-only: URL-friendliness, case-insensitive matching
- Hyphens allowed: Human readability for multi-word slugs
- Numbers allowed: Version suffixes (e.g., "react-app-v2")
- Must start/end with alphanumeric: Prevents edge cases like "--slug--"

**Recovery path:** CategoryValidation → 400 → User corrects input

### Decision: Path Existence Check

**Location:** [CODE: api/domain/reference/service.go#Create] lines 97-103

**What it decides:** Whether a reference scenario path points to a real directory.

**Criteria:**
- `os.Stat()` returns no error, or error is not `IsNotExist`

**Rationale:**
- References must point to real scenarios for validation to work
- Absolute path normalization ensures consistent lookups
- Check happens at create/update time, not at validation time

**Edge cases:**
- Symlinks: Followed by `os.Stat()`, considered valid
- Race condition: Path could be deleted after create; acceptable trade-off
- Permission errors: Currently treated as "exists"; could be improved

### Decision: Error Category Mapping

**Location:** [CODE: api/handlers/errors.go#MapDomainError]

**What it decides:** How domain-layer errors become API responses.

**Mapping rules:**
| Domain Error | API Category | HTTP Status | Recovery |
|--------------|--------------|-------------|----------|
| ErrNotFound | not_found | 404 | Check identifier |
| ErrInvalidSlug | validation | 400 | Fix format/length |
| ErrSlugExists | conflict | 409 | Use different slug |
| ErrPathNotExists | validation | 400 | Fix path |
| (unknown) | internal | 500 | Retry/contact support |

**Design notes:**
- All known domain errors have explicit mappings
- Unknown errors become internal errors (safe default)
- Underlying error is wrapped for logging but not exposed to clients

### Decision: Pagination Limit Application

**Location:** [CODE: api/internal/config/config.go#ApplyPaginationLimit]

**What it decides:** The effective page size for list operations.

**Criteria:**
- If limit <= 0: Use DefaultLimit (20)
- If limit > MaxLimit: Use DefaultLimit (20)
- Otherwise: Use requested limit

**Rationale:**
- Protects against resource exhaustion from unlimited queries
- DefaultLimit is conservative for typical use
- MaxLimit (100) allows batch operations without excessive memory

**Notable choice:** Invalid limits (negative, excessive) fall back to default rather than max. This prevents accidental large queries while allowing intentional batching with explicit limits.

### Decision: CORS Origin Validation

**Location:** [CODE: api/internal/config/config.go#IsOriginAllowed]

**What it decides:** Whether a cross-origin request should be permitted.

**Criteria:**
- Origin must exactly match an entry in AllowedOrigins list (after trimming)

**Default origins:**
- localhost:3000, localhost:5173 (dev servers)
- 127.0.0.1:3000, 127.0.0.1:5173 (alternate localhost)

**Production notes:**
- Set CORS_ALLOWED_ORIGINS to explicit list in production
- No wildcard support (intentional security constraint)

### Decision: HTTP Status from Error Category

**Location:** [CODE: api/internal/errors/errors.go#ToHTTPStatus]

**What it decides:** The HTTP status code for API error responses.

**Mapping:**
| Category | Transient? | HTTP Status |
|----------|------------|-------------|
| validation | - | 400 |
| not_found | - | 404 |
| conflict | - | 409 |
| database | true | 503 |
| database | false | 500 |
| dependency | true | 503 |
| dependency | false | 502 |
| internal | - | 500 |

**Rationale:**
- 503 for transient errors signals "retry later"
- 502 for dependency errors distinguishes from server bugs
- 500 as catch-all for unexpected failures

### Decision Groups by Domain

**Reference Domain:**
- Slug validation (format, length, uniqueness)
- Path validation (existence)
- Pagination (limit enforcement)

**Error Handling:**
- Category assignment
- Severity assignment
- HTTP status mapping
- Recovery message selection

**API Layer:**
- CORS origin validation
- Request body parsing
- Route matching (delegated to gorilla/mux)

### Future Decision Points (Planned)

These decisions will be needed as domains are implemented:

**Skill Connection Domain:**
- Version compatibility checking
- Hash mismatch detection (drift)
- Skill-reference affinity validation

**Validation Domain:**
- Assertion operator selection
- Pass/fail threshold determination
- Overlap detection algorithm

**Report Domain:**
- Health score calculation
- Severity weighting for violations
- Trend detection windows

## CLI-API Parity (2026-03-11)

This section documents the alignment between CLI commands and API endpoints per CLI Steer skill requirements.

### Coverage Map

| API Endpoint | HTTP Method | CLI Command | Flags |
|--------------|-------------|-------------|-------|
| /health | GET | `status` | - |
| /api/v1/references | GET | `reference list` | `--template`, `--json` |
| /api/v1/references | POST | `reference create` | `--slug`, `--name`, `--template`, `--path`, `--description`, `--json` |
| /api/v1/references/{id} | GET | `reference get <id>` | `--json` |
| /api/v1/references/{id} | PATCH | `reference update <id>` | `--name`, `--template`, `--path`, `--description`, `--json` |
| /api/v1/references/{id} | DELETE | `reference delete <id>` | `--json` |
| /api/v1/references/by-slug/{slug} | GET | `reference get <slug>` | `--json` |

### CLI Architecture

The CLI follows these patterns from CLI Steer skill:

1. **Thin Wrapper**: CLI parses args, calls API, formats output. No business logic.
2. **cli-core**: Uses `ScenarioApp` for scaffolding, env var handling, stale detection.
3. **HTTP Helpers**: Centralized `get()`, `post()`, `patch()`, `delete()` methods.
4. **Two Output Modes**: Human-friendly default, `--json` for machine-readable.
5. **ParseInterspersed**: Uses `cliutil.ParseInterspersed()` for mixed args/flags.

### Implementation Files

| File | Purpose |
|------|---------|
| [CODE: cli/app.go] | CLI application with command registration |
| [CODE: cli/app_test.go] | CLI unit tests |
| [CODE: cli/main.go] | Entry point |

## Dry-Run Support (2026-03-11)

Mutating API endpoints support dry-run validation via `X-Dry-Run: true` header.

### Supported Endpoints

| Endpoint | Dry-Run Behavior |
|----------|------------------|
| POST /api/v1/references | Validates input (slug format, uniqueness, path exists) without creating |
| PATCH /api/v1/references/{id} | Validates update (reference exists, new path if provided) without persisting |
| DELETE /api/v1/references/{id} | Validates reference exists without deleting |

### Response Format

Dry-run responses include `"dry_run": true` and realistic response data:

```json
{
  "success": true,
  "dry_run": true,
  "data": {
    "id": "generated-uuid",
    "slug": "requested-slug",
    ...
  }
}
```

### Implementation

Service layer provides validation-only methods:
- [CODE: api/domain/reference/service.go#ValidateCreate] - Validates without persistence
- [CODE: api/domain/reference/service.go#ValidateUpdate] - Validates without persistence

Handler layer checks `isDryRun(r)` and uses validation methods:
- [CODE: api/handlers/reference.go#isDryRun] - Header check helper

### Testing

Dry-run validation is tested at the service layer:
- [CODE: api/domain/reference/service_test.go#TestService_ValidateCreate]
- [CODE: api/domain/reference/service_test.go#TestService_ValidateUpdate]
