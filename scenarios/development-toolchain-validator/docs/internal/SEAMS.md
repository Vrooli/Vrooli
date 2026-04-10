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

### Decision: Skill ID Format Validation

**Location:** [CODE: api/domain/skill/service.go#isValidSkillID]

**What it decides:** Whether a skill ID is acceptable for connection management.

**Criteria:**
- Length: Between 2 and 100 characters
- Format: Must start with a letter, contain only lowercase letters, numbers, and hyphens

**Rationale:**
- Must start with letter: Distinguishes skill IDs from UUIDs
- Lowercase-only: Consistency with slug conventions
- Hyphens allowed: Human readability for multi-word skill IDs

**Recovery path:** CategoryValidation → 400 → User corrects input

### Decision: Skill Connection Error Mapping

**Location:** [CODE: api/handlers/errors.go#MapSkillDomainError]

**What it decides:** How skill domain errors become API responses.

**Mapping rules:**
| Domain Error | API Category | HTTP Status | Recovery |
|--------------|--------------|-------------|----------|
| ErrNotFound | not_found | 404 | Check identifier |
| ErrInvalidSkillID | validation | 400 | Fix skill ID format |
| ErrInvalidReferenceID | validation | 400 | Provide valid reference ID |
| ErrConnectionExists | conflict | 409 | Use existing connection |
| (unknown) | internal | 500 | Retry/contact support |

**Design notes:**
- Follows same pattern as reference domain for consistency
- Centralized in `handlers/errors.go` for maintainability
- Error constructors defined in `internal/errors/errors.go`

### Decision: Drift Detection Logic

**Location:** [CODE: api/domain/skill/service.go#CheckDrift]

**What it decides:** Whether a skill connection has drifted from its stored state.

**Criteria:**
- Version changed: `stored.SkillVersion != current.SkillVersion`
- Content changed: `stored.SkillContentHash != current.ContentHash`
- Has drifted: Either version OR content changed

**Rationale:**
- Simple boolean drift detection (not magnitude)
- Caller provides current values from prompt-manager API
- Both version and hash tracked for precise drift identification

### Future Decision Points (Planned)

These decisions will be needed as domains are implemented:

**Skill Connection Domain:**
- Version compatibility checking (semantic versioning rules)
- Skill-reference affinity validation (template compatibility)

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

## React Stability (2026-03-11)

This section documents the stability infrastructure per React Stability skill requirements.

### TypeScript Safety Configuration

The scenario uses strict TypeScript configuration to catch runtime errors at compile time:

| Rule | Location | Purpose |
|------|----------|---------|
| `strict: true` | tsconfig.node.json | Enables all strict type checks |
| `noUncheckedIndexedAccess: true` | tsconfig.node.json | Forces null checks on array/object access |

These rules prevent common crashes like:
- `Cannot read property X of undefined`
- `X is not a function`
- `undefined is not iterable`

### ESLint Safety Rules

ESLint config (eslint.config.js) includes safety-critical rules:

| Rule | Level | Purpose |
|------|-------|---------|
| `react-hooks/rules-of-hooks` | error | Prevents React Error #310 (hook count changes) |
| `@typescript-eslint/no-non-null-assertion` | error | Prevents `!` operator that hides null bugs |
| `@typescript-eslint/no-explicit-any` | error | Prevents disabling type checking |
| `react-hooks/exhaustive-deps` | warn | Catches stale closure bugs |

### Error Boundary Strategy

Error boundaries are placed to isolate failures:

| Location | Component | Purpose |
|----------|-----------|---------|
| App root | [CODE: ui/src/main.tsx] | Catches top-level rendering errors |
| Data sections | Component-level | Isolates external data failures |

The ErrorBoundary component ([CODE: ui/src/components/ErrorBoundary.tsx]) provides:
- Error logging to console for debugging
- User-friendly error message (no raw stack traces)
- Recovery options: Try Again, Refresh Page
- Optional custom fallback UI via props

### Data Access Patterns

All data access follows defensive patterns:
- Optional chaining (`?.`) for nested access
- Nullish coalescing (`??`) for defaults
- Array guards before mapping: `(data?.items ?? []).map(...)`

## UI Interop Compliance (2026-03-11)

This section documents compliance with Vrooli UI Interop skill requirements.

### Interop Slot Compliance

| Slot | File | Status | Notes |
|------|------|--------|-------|
| [A] Dependencies | ui/package.json | ✅ | @vrooli/api-base, @vrooli/iframe-bridge |
| [B] Vite base | ui/vite.config.ts | ✅ | `base: './'` with INTEROP-CRITICAL comment |
| [C] Server | ui/server.js | ✅ | Uses `startScenarioServer()` |
| [D] Bridge init | ui/src/main.tsx | ✅ | Idempotency guard, parentOrigin extraction |
| [E] Router basename | N/A | N/A | No router (simple UI) |
| [F] API client | ui/src/lib/api.ts | ✅ | Uses `resolveApiBase()`, `buildApiUrl()` |
| [G] Keyboard shortcuts | N/A | N/A | No app-level shortcuts (simple UI) |

### iframe Bridge Configuration

The bridge initialization includes:
1. **Guard**: `window.parent !== window` (iframe detection)
2. **Idempotency**: `window.__developmentToolchainValidatorBridgeInitialized` flag
3. **Parent origin**: Extracted from `document.referrer` for secure postMessage
4. **App ID**: `"development-toolchain-validator"` for host identification

### API Base Resolution

API calls use centralized resolution ([CODE: ui/src/lib/api.ts]):
```typescript
const API_BASE = resolveApiBase({ appendSuffix: true });
// Resolves to correct endpoint in all contexts:
// - Localhost: http://localhost:PORT/api/v1
// - Tunnel: https://subdomain.trycloudflare.com/api/v1
// - Proxy: /apps/NAME/proxy/api/v1
```

### Self-Detection & Graceful Degradation

All interop patterns auto-detect context and degrade to no-op when not applicable:
- Bridge init: Skipped when not in iframe
- API base: Falls back to localhost when not proxied
- Router basename: Not needed (no router)

## React Coherence (2026-03-11)

This section documents architectural coherence per React Coherence skill requirements.

### State Architecture

Current pattern: **Server-state only** (appropriate for template stage)

| State Type | Mechanism | Usage |
|------------|-----------|-------|
| Server state | React Query | Health check in App.tsx |
| Local state | None | Template has no local state needs |
| App-wide stores | None | Not needed for current feature set |

**State decision guidance for future features:**
1. Local `useState` for component-local, ephemeral state
2. Feature-local hooks for feature-scoped shared state
3. App-wide stores only for truly cross-surface state
4. Server state via React Query (already in place)

### Code Organization

```
ui/src/
├── components/
│   ├── ui/             # Primitives (Button) - CVA variants
│   └── ErrorBoundary.tsx
├── consts/
│   └── selectors.ts    # Automation selectors (well-structured)
├── lib/
│   ├── api.ts          # Centralized API client
│   └── utils.ts        # CN helper
├── App.tsx             # Main app (minimal template)
├── main.tsx            # Entry with providers
└── styles.css          # Design tokens + Tailwind
```

**Ownership rules:**
1. `styles.css` owns design tokens (color, surface, border, radius, motion)
2. `components/ui/` owns base interactive primitives (Button)
3. `lib/` owns pure utilities and API client
4. `consts/` owns automation selectors

### Design Token System

Semantic tokens defined in `styles.css`:

| Category | Tokens Defined | Status |
|----------|---------------|--------|
| Color - Text | primary, secondary, muted | ✅ |
| Color - Surface | base, elevated, overlay | ✅ |
| Color - Status | danger, success, warning | ✅ |
| Border | default, subtle | ✅ |
| Radius | sm, md, lg, xl, 2xl | ✅ |
| Motion | fast, normal, slow, ease-out | ✅ |

**Theme refresh readiness:** Foundation in place. Add theme switching infrastructure when product requirements call for it.

### Iframe-Safe Layout Pattern

Per UI Interop §4.5, the app uses `h-full` instead of `h-screen`:

```tsx
// In App.tsx - uses h-full, NOT h-screen
<div className="h-full bg-slate-950 ...">
```

Height chain in styles.css:
```css
html, body, #root { height: 100%; margin: 0; }
```

This ensures correct sizing in all three deployment contexts (localhost, tunnel, proxy/iframe).

### UI-API Integration (2026-03-11)

The UI integrates with the Reference Registry API endpoints:

| UI Component | API Endpoint | Method | Purpose |
|--------------|--------------|--------|---------|
| Dashboard health indicator | `/health` | GET | API connectivity status |
| Reference list | `/references` | GET | Fetch all references |
| Reference detail (planned) | `/references/{id}` | GET | Single reference |
| Reference by slug (planned) | `/references/by-slug/{slug}` | GET | Lookup by slug |

**API Client Pattern** ([CODE: ui/src/lib/api.ts]):
- Typed request/response interfaces (`Reference`, `ReferenceListResponse`)
- Centralized error handling with `ApiError` type
- Uses `@vrooli/api-base` for URL resolution across deployment contexts

**Automation Selectors** ([CODE: ui/src/consts/selectors.ts]):
- Literal selectors for dashboard states (`loading`, `error`, `empty`)
- Dynamic selectors for reference cards (`cardBySlug`, `cardTemplate`, `cardPath`)
- All selectors use `data-testid` pattern for reliable automation

### Coherence Audit Results

Full audit documented in [docs/internal/COHERENCE-NOTES.md].

**Summary:**
- State: Server-state only via React Query (appropriate for current feature set)
- Duplication: None found
- Styling: Design tokens added, CVA in use
- Architecture: Reference Registry dashboard implemented
- Iframe safety: Uses `h-full` pattern throughout

### Relationship to Other Skills

| Skill | Boundary |
|-------|----------|
| react-stability | Stability = crash prevention (error boundaries, null guards) |
| react-coherence | Coherence = code organization, state management, styling system |
| vrooli-ui-interop | Interop = deployment-context correctness (iframe, proxy) |

These skills work together:
1. **Coherence** ensures maintainable structure
2. **Stability** prevents runtime crashes
3. **Interop** ensures correct behavior across deployment contexts

## Temporal Flow & Replay Safety (2026-03-11)

This section documents time-based behavior and idempotency guarantees.

### Documentation References

| Document | Purpose |
|----------|---------|
| [docs/internal/TEMPORAL-FLOWS.md] | Async operations, initialization sequences, polling |
| [docs/internal/INVARIANTS.md] | Idempotency, replay safety, commit boundaries |

### Key Temporal Patterns

**API Operations**: All CRUD operations are synchronous request-response with no background processing.

| Pattern | Status | Details |
|---------|--------|---------|
| Database init before server | ✅ Stable | `database.Connect()` blocks until ready |
| Health check before data fetch (UI) | ✅ Stable | React Query `enabled: healthQuery.isSuccess` |
| Iframe bridge before React mount | ✅ Stable | Idempotency guard prevents double-init |

**Polling**: Health check (UI) polls every 30 seconds.

### Idempotency Summary

| Operation | Idempotent? | Double-Apply Behavior |
|-----------|-------------|----------------------|
| Create | No | Returns 409 Conflict |
| Update | Yes | Same result on replay |
| Delete | Yes* | Returns 404 on replay (acceptable) |
| Dry-run | Yes | No side effects |

### Replay Safety Tests

Tests verifying replay behavior: [CODE: api/domain/reference/service_test.go#TestService_Create_ReplayReturnsConflict]

```go
// Replay patterns tested:
// - TestService_Create_ReplayReturnsConflict
// - TestService_Update_ReplayProducesSameState
// - TestService_Delete_ReplayIsSafe
// - TestService_ValidateCreate_NoSideEffects
```

### Future Hardening

1. **Optimistic concurrency**: Add ETag/version field for update conflict detection
2. **Idempotency keys**: Add `Idempotency-Key` header support for safe retries
3. **Request timeouts**: Add server-side context timeouts to API handlers

## Utility Consolidation (2026-03-11)

This section documents the shared utility architecture per Utils Unification skill.

### Package Structure

```
api/internal/
├── validation/    # Core: Pure validators (slug, JSONPath, command safety)
├── config/        # Core: Environment configuration
├── errors/        # Core: Structured error types
├── testutil/      # Testing: Factories and helpers
└── mocks/         # Testing: Mock implementations
```

### Consolidated Utilities

The `internal/validation` package consolidates validation patterns from domain services:

| Utility | Domains Using | Purpose |
|---------|--------------|---------|
| `IsValidSlugFormat` | reference | URL-safe slug format check |
| `IsValidSkillIDFormat` | skill | Skill ID format check |
| `IsValidJSONPath` | expectation | JSONPath expression validation |
| `IsLengthInRange` | reference, skill | Generic length bounds check |
| `IsCommandSafe` | expectation | Dangerous command pattern detection |
| `Truncate` | (CLI only) | String truncation with ellipsis |

### Design Principles

1. **Pure functions**: Validators have no side effects
2. **No domain imports**: Internal packages cannot import domain packages
3. **Config injection**: Domain-specific limits (min/max) come from config
4. **Error construction**: Validators return bool; callers construct errors

### Domain vs Shared

**Keep in domain:**
- Repository-dependent checks (slug uniqueness, path existence)
- Domain-specific enums (ExpectationType, AssertionOperator)
- Business rules requiring context

**Move to shared:**
- Format validation (regex-based)
- Range checks (length bounds)
- Safety checks (dangerous patterns)

### Documentation

Full utility architecture: [docs/internal/UTILS_UNIFICATION_NOTES.md]

## UI Boundary Enforcement (2026-03-11)

This section documents the boundary-of-responsibility patterns enforced in the React UI layer.

### Centralized Health Status Management

**Hook:** [CODE: ui/src/hooks/useHealthStatus.ts]

**Responsibility boundary:** Health check polling and status derivation are owned by a single hook, not duplicated across pages.

**Decision boundary:** "What is the health status?"
- loading: Health check is in progress
- connected: Health check succeeded
- disconnected: Health check failed

**Usage pattern:**
```tsx
const { isHealthy, healthStatus, refetch } = useHealthStatus();

// Enable data queries only when healthy
const dataQuery = useQuery({
  queryKey: ["data"],
  queryFn: fetchData,
  enabled: isHealthy
});

// Use healthStatus for the indicator
<HealthIndicator status={healthStatus} />
```

**Before:** Dashboard and ReferenceDetail each had their own health query setup, status derivation logic (8-10 lines each), and refresh handling. Same pattern repeated in two places.

**After:** Single `useHealthStatus()` hook owns health check polling, status derivation, and refetch. Pages import and destructure only what they need.

### Centralized API Request Handling

**Location:** [CODE: ui/src/lib/api.ts#apiRequest]

**Responsibility boundary:** Error extraction from failed responses is owned by a single helper function, not repeated in every API function.

**Decision boundary:** "How to handle API errors?"
- Extract error message from response body if available
- Fall back to provided error context with status code
- Throw Error with user-facing message

**Pattern:**
```typescript
async function apiRequest<T>(path: string, options: RequestOptions<T>): Promise<T>
```

**Before:** Each API function (fetchReferences, fetchReferenceById, createReference, etc.) had its own error handling block (5-6 lines each): check `!res.ok`, try to parse error body, throw with message. Pattern repeated 8+ times.

**After:** Single `apiRequest()` helper consolidates:
- URL building with correct base
- Headers and cache control
- Error message extraction
- Optional response transformation

Individual API functions are now 3-5 lines each, delegating all error handling to the shared helper.

### Responsibility Zones (UI Layer)

| Zone | Owner | Files |
|------|-------|-------|
| Health state | `useHealthStatus` hook | `hooks/useHealthStatus.ts` |
| Request/error handling | `apiRequest` helper | `lib/api.ts` |
| Keyboard shortcuts | `useKeyboardShortcut` hook | `hooks/useKeyboardShortcut.ts` |
| Date formatting | `formatDate` utility | `lib/utils.ts` |
| Automation selectors | Selector registry | `consts/selectors.ts` |
| Layout structure | `Layout` component | `components/Layout.tsx` |

### Decision Points (UI Layer)

**Decision: Health Status Derivation**

**Location:** [CODE: ui/src/hooks/useHealthStatus.ts#healthStatus]

**Criteria:**
- isLoading → "loading"
- isSuccess → "connected"
- else → "disconnected"

**Rationale:** Loading state takes precedence because we want to show loading indicator while checking, not stale disconnected status.

**Decision: API Error Message Extraction**

**Location:** [CODE: ui/src/lib/api.ts#apiRequest]

**Criteria:**
- Try to parse response body as JSON with `error` field
- If parsing fails, use "Unknown error"
- Combine with context: `errorBody.error ?? \`${errorContext}: ${res.status}\``

**Rationale:** API returns structured error responses with user-facing messages. Fallback ensures we never show undefined/null to users.

**Decision: Query Enable Cascading**

**Location:** Pages (Dashboard, ReferenceDetail)

**Criteria:**
- Reference list query: `enabled: isHealthy`
- Single reference query: `enabled: !!slug && isHealthy`
- Connections query: `enabled: !!referenceQuery.data?.id`

**Rationale:** Don't attempt data fetches until prerequisites are met:
1. API must be healthy before fetching data
2. Slug must be present before fetching by slug
3. Reference ID must exist before fetching connections

## CLI Decision Points (2026-03-11)

This section documents decision points specific to the CLI layer.

### Decision: Command Subcommand Routing

**Location:** [CODE: cli/app.go#cmdReference], [CODE: cli/app.go#cmdConnection]

**What it decides:** Which subcommand handler to invoke.

**Criteria:**
- First argument determines subcommand (list, get, create, update, delete)
- Aliases supported (e.g., "ls" → "list", "rm" → "delete", "show" → "get")
- Unknown subcommands return error with usage text

**Design notes:**
- Uses switch statement for explicit routing
- Help subcommand is first-class, not flag-based
- Consistent alias pattern across command groups

### Decision: Reference Get ID vs Slug Resolution

**Location:** [CODE: cli/app.go#cmdRefGet]

**What it decides:** How to resolve an ambiguous identifier.

**Criteria:**
- Try GET /references/{identifier} first (assumes UUID)
- If 404, retry with GET /references/by-slug/{identifier}
- Return first successful response

**Rationale:**
- UUIDs are more common in automated workflows (machine-to-machine)
- Slugs are more common in human workflows (memorable names)
- Try UUID first because it's unambiguous; slug may conflict with future UUID format

### Decision: Required Field Validation Order

**Location:** [CODE: cli/app.go#cmdRefCreate], [CODE: cli/app.go#cmdConnConnect]

**What it decides:** When to validate required fields before API call.

**Criteria:**
- Check all required fields in single condition
- Return error before any API call if validation fails
- Include usage hint in error message

**Rationale:**
- Early validation avoids wasted API roundtrip
- Combined check shows all missing fields at once
- Usage hint guides user to correct invocation

## Validation Domain Decision Points (2026-03-11)

This section documents decision points in the CLI assertion validation engine.

### Decision: Assertion Operator Selection

**Location:** [CODE: api/domain/validation/cli_executor.go#evaluateAssertion]

**What it decides:** How to compare actual vs expected values.

**Mapping:**
| Operator | Comparison Type | Value Types Supported |
|----------|-----------------|----------------------|
| `exists` | Presence check | Any |
| `eq` | Deep equality | Any (numeric normalization) |
| `neq` | Deep inequality | Any |
| `contains` | Substring/element | String, Array |
| `matches` | Regex match | String only |
| `gt/gte/lt/lte` | Numeric comparison | Numeric types |
| `between` | Range check | Numeric types |

**Design notes:**
- Unknown operators return `ErrInvalidOperator`
- Type coercion handled via `toFloat64()` for numeric comparisons
- Deep equality uses `reflect.DeepEqual` with numeric normalization

### Decision: JSONPath Expression Parsing

**Location:** [CODE: api/domain/validation/cli_executor.go#extractJSONPath]

**What it decides:** How to navigate JSON structure.

**Supported patterns:**
- `$` - Root element
- `$.field` - Object field access
- `$.field.nested` - Nested field access
- `$[0]` - Array index access
- `$.field[0].nested` - Mixed access
- `$[*]` - All array elements

**Error conditions:**
- Path must start with `$`
- Unclosed brackets
- Non-existent fields
- Array index out of bounds
- Type mismatches (accessing array on object, etc.)

**Recovery path:** Return descriptive error explaining what failed

### Decision: Command Safety Blocking

**Location:** [CODE: api/domain/validation/cli_executor.go#ValidateAssertion]

**What it decides:** Whether a command is safe to execute.

**Criteria:**
- Checks via `validation.ValidateCommandSafety()` before execution
- Blocks dangerous patterns: `rm -rf`, `sudo`, `curl|bash`, `eval`, etc.

**Rationale:**
- CLI assertions execute arbitrary commands
- Security boundary must block destructive operations
- Better to block legitimate edge cases than allow dangerous ones

**Recovery path:** Return error with detected dangerous pattern

### Decision: Command Timeout Handling

**Location:** [CODE: api/domain/validation/cli_executor.go#executeCommand]

**What it decides:** How long to wait for command completion.

**Criteria:**
- Default: 30 seconds (`DefaultCommandTimeout`)
- Configurable via `WithTimeout()` option
- Uses `context.WithTimeout()` for cancellation

**Behavior on timeout:**
- Returns `ErrCommandTimeout`
- Kills the child process via context cancellation

**Rationale:**
- Long-running commands shouldn't block validation indefinitely
- 30 seconds generous enough for most CLI tools
- Configurable for known slow commands

## E2E Testing Seams (2026-03-11)

This section documents the seams available for end-to-end testing via Browser Automation Studio (BAS) workflows.

### Selector Registry Seam

**Location:** [CODE: ui/src/consts/selectors.ts]

**Purpose:** Single source of truth for all automation selectors, enabling UI-agnostic E2E tests.

**Pattern:**
- Literal selectors for static elements (dashboard titles, buttons)
- Dynamic selectors for parameterized elements (card by slug, connection by skill ID)
- Auto-generated `selectors.manifest.json` for workflow linting

**Usage in workflows:**
```json
{
  "type": "click",
  "data": { "selector": "@selector/dashboard.refreshButton" }
}
```

**Usage for dynamic selectors:**
```json
{
  "type": "assert",
  "data": {
    "selector": "@selector/references.cardBySlug(slug=my-reference)",
    "assertMode": "exists"
  }
}
```

**Testability benefits:**
- UI refactors don't break E2E workflows
- Selector changes propagate to all workflows via registry
- Type-safe selector references catch typos at build time

### API Endpoint Seam

**Location:** [CODE: ui/src/lib/api.ts]

**Purpose:** Centralized API client enabling consistent request/response handling in tests.

**Testable endpoints:**
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | API connectivity check |
| `/api/v1/references` | GET | List all references |
| `/api/v1/references/{id}` | GET | Get reference by ID |
| `/api/v1/references/by-slug/{slug}` | GET | Get reference by slug |
| `/api/v1/connections` | GET | List skill connections |
| `/api/v1/connections/{id}` | GET | Get connection by ID |

**BAS workflow pattern:** Navigate → interact → assert API state via UI or direct API calls.

### UI Component Test IDs

All interactive UI components include `data-testid` attributes for reliable automation:

| Component | Test ID Pattern | Example |
|-----------|-----------------|---------|
| Dashboard title | `dashboard-title` | Fixed |
| Health status | `dashboard-health-status` | Fixed |
| Reference card | `reference-card-${slug}` | Dynamic |
| Reference link | `reference-card-link-${slug}` | Dynamic |
| Connection card | `connection-card-${skillId}` | Dynamic |
| Back button | `reference-detail-back-button` | Fixed |
| Refresh button | `dashboard-refresh-button` | Fixed |

### BAS Workflow Organization

**Directory structure:**
```
bas/
├── registry.json              # Auto-generated manifest
├── cases/                     # Test cases WITH assertions
│   └── 01-reference-scenario-registry/
│       └── api/
│           ├── crud-operations.json
│           └── dry-run-validation.json
├── flows/                     # Reusable journeys (NO assertions)
└── actions/                   # Atomic operations (NO assertions)
```

**Execution hierarchy:**
1. `actions/` - Atomic steps (login, navigate, click)
2. `flows/` - Multi-step journeys composed from actions
3. `cases/` - Full test cases with assertions

### iframe Bridge Seam

**Location:** [CODE: ui/src/main.tsx]

**Purpose:** Communication channel between parent Vrooli app and embedded scenario UI.

**Testable signals:**
- `ready` - UI has mounted and bridge initialized
- `shortcutIntent` - Keyboard shortcut handled within scenario

**BAS integration:**
```json
{
  "type": "wait",
  "data": {
    "selector": "@selector/app.ready",
    "state": "visible",
    "timeoutMs": 10000
  }
}
```

### Health Status Seam

**Location:** [CODE: ui/src/hooks/useHealthStatus.ts]

**Purpose:** Centralized health check polling enabling consistent status assertions.

**Test states:**
| Status | Condition | UI Indicator |
|--------|-----------|--------------|
| `loading` | Health check in progress | Spinner |
| `connected` | Health check succeeded | Green indicator |
| `disconnected` | Health check failed | Red indicator |

**BAS assertion example:**
```json
{
  "type": "assert",
  "data": {
    "selector": "@selector/dashboard.healthStatus",
    "assertMode": "contains_text",
    "expectedText": "connected"
  }
}
```

### Test Data Management Seam

**Location:** [CODE: api/internal/testutil/fixtures.go]

**Purpose:** Consistent test data creation for API and E2E tests.

**Available factories:**
- `NewReferenceFactory()` - Create reference fixtures
- `NewCreateInputFactory()` - Create reference input fixtures
- `NewConnectionFactory()` - Create skill connection fixtures
- `NewConnectInputFactory()` - Create connect input fixtures

**Pattern:** Builder pattern with chainable `With*()` methods:
```go
ref := NewReferenceFactory().
    WithSlug("e2e-test-ref").
    WithTemplate("react-vite").
    Build()
```

### Mock Repository Seam

**Location:** [CODE: api/internal/mocks/repository.go]

**Purpose:** Enable unit testing without database dependency.

**Available mocks:**
- `MockRepository` - Reference repository mock
- `MockSkillRepository` - Skill connection repository mock

**Test isolation features:**
- Error injection: `WithCreateError()`, `WithGetByIDError()`, etc.
- Call tracking: `CreateCallCount()`, `DeleteCallCount()`, etc.
- State management: `WithReference()`, `Reset()`

### Cross-Layer Testing Strategy

| Layer | Test Type | Tools | Seam Used |
|-------|-----------|-------|-----------|
| Domain | Unit | Go tests | Mock repositories |
| Handler | Integration | Go tests + httptest | Full service stack |
| API | E2E | BAS workflows | API endpoints |
| UI | E2E | BAS workflows | Selector registry |
| Full stack | Smoke | `vrooli scenario ui-smoke` | iframe bridge + health |

### BAS Workflow Best Practices

1. **Use selector registry** - Never hardcode selectors in workflows
2. **Wait for health** - Always wait for API health before data operations
3. **Atomic actions first** - Build reusable actions before complex flows
4. **Assert state, not timing** - Use `wait` with state conditions, not fixed timeouts
5. **Clean test data** - Create unique test data per workflow to avoid conflicts
