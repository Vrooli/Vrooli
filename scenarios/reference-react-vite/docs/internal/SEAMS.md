# Seams Documentation

This document identifies the architectural seams in the reference-react-vite scenario, following the seam-discovery-and-enforcement steer skill patterns.

## Overview

The reference-react-vite scenario demonstrates a task/project management domain with clear architectural boundaries between presentation, coordination, domain rules, and data access.

## Identified Seams

### 1. HTTP Handler Seam

**Location**: `api/handlers/`
**Responsibility**: HTTP request/response orchestration

- Parses HTTP requests into domain input types
- Delegates business logic to domain packages
- Converts domain errors to standardized API error responses [CODE: api/handlers/errors.go#APIError]
- Handles pagination metadata wrapping [CODE: api/handlers/errors.go#ListResponse]

**Does NOT**:
- Contain business validation logic (delegated to domain)
- Directly access database (uses repository interfaces)
- Know about specific database technologies

**Key Files**:
- [CODE: api/handlers/tasks.go] - Task CRUD operations
- [CODE: api/handlers/projects.go] - Project CRUD operations
- [CODE: api/handlers/notes.go] - Note CRUD operations
- [CODE: api/handlers/errors.go] - Error formatting utilities

### 2. Domain Logic Seam

**Location**: `api/domain/`
**Responsibility**: Business rules and entity definitions

- Defines domain entities (Task, Project, Note)
- Implements validation rules (title length, status values)
- Provides factory functions with business rule enforcement
- Contains update operations with invariant preservation

**Does NOT**:
- Know about HTTP or REST concepts
- Know about database schemas or SQL
- Handle persistence concerns

**Key Files**:
- [CODE: api/domain/tasks/task.go] - Task entity and validation
- [CODE: api/domain/projects/project.go] - Project entity and validation
- [CODE: api/domain/notes/note.go] - Note entity and validation

### 3. Repository Seam

**Location**: `api/repository/`
**Responsibility**: Data access abstraction

- Defines interfaces for data operations [CODE: api/repository/repository.go#TaskRepository]
- PostgreSQL implementations handle SQL translation
- Provides pagination with total count

**Does NOT**:
- Contain business logic
- Make business decisions about data validity
- Know about HTTP layer

**Key Files**:
- [CODE: api/repository/repository.go] - Interface definitions
- [CODE: api/repository/tasks_postgres.go] - PostgreSQL task implementation
- [CODE: api/repository/projects_postgres.go] - PostgreSQL project implementation
- [CODE: api/repository/notes_postgres.go] - PostgreSQL note implementation
- [CODE: api/repository/query_builder.go] - Safe SQL query construction

### 4. Database Schema Seam

**Location**: [CODE: initialization/postgres/schema.sql]
**Responsibility**: Physical data structure

- Defines tables with constraints
- Provides indexes for query performance
- Implements auto-update triggers

**Does NOT**:
- Contain application logic
- Know about Go types or repository interfaces

## Architecture Alignment

The codebase follows screaming architecture principles:

```
api/
├── domain/          # ← Business domains "scream" at this level
│   ├── tasks/       #    Each domain is self-contained
│   ├── projects/    #    with its own types and rules
│   └── notes/
├── handlers/        # ← HTTP layer, domain-organized
├── repository/      # ← Data access, interfaces + implementations
└── main.go          # ← Composition root
```

### Logical vs Physical Structure

| Logical Layer | Physical Location | Responsibility |
|--------------|-------------------|----------------|
| Presentation | `api/handlers/` | HTTP orchestration |
| Domain | `api/domain/` | Business rules |
| Data Access | `api/repository/` | Database operations |
| Infrastructure | `api/main.go` | Server setup, cross-cutting |

### Boundary Violations Fixed

1. **Before**: All code in single `main.go` - no clear boundaries
2. **After**: Domain packages with explicit interfaces

### Remaining Work

1. ✅ ~~Add domain tests co-located with domain packages~~ - Completed
2. ✅ ~~Add handler tests with mock repositories~~ - Completed
3. Add integration tests with real database (testcontainers)
4. Consider adding service layer if coordination logic grows

## Cross-Cutting Concerns

| Concern | Location | Notes |
|---------|----------|-------|
| Logging | [CODE: api/main.go] | Request logging middleware |
| Error logging | [CODE: api/handlers/errors.go] | Structured error logs |
| CORS | [CODE: api/main.go] | Configurable via env var |
| Request ID | [CODE: api/handlers/errors.go#getRequestID] | Generated per-request |
| Error format | [CODE: api/handlers/errors.go#APIError] | Consistent API error shape |
| Recovery hints | [CODE: api/handlers/errors.go#errorRecoveryHints] | User-actionable guidance |
| Recovery | [CODE: api/main.go#Handler] | Gorilla handlers recovery |

## Observability Surface

This section documents the signals available for monitoring and debugging the scenario.

### Key Observable States

| State | Signal | How to Observe |
|-------|--------|----------------|
| Request received | `[METHOD] path duration` | stdout logs |
| Error occurred | `[ERROR] request_id=X code=Y status=Z path=W message=M` | stdout logs |
| Health status | `/health` endpoint | HTTP GET |

### Signal Inventory

#### Structured Error Logs

All API errors emit structured log entries:
```
[ERROR] request_id=abc123 code=VALIDATION_ERROR status=422 path=/api/v1/tasks message="task title is required"
```

Fields:
- `request_id`: Correlation ID (from client or auto-generated UUID)
- `code`: Machine-readable error category
- `status`: HTTP status code
- `path`: Request path
- `message`: Error description

#### API Error Response Shape

```json
{
  "code": "ERROR_CODE",
  "message": "Human-readable description",
  "details": { "field": "specific_error" },
  "recovery": "Suggested action to resolve",
  "retryable": false,
  "request_id": "correlation-id"
}
```

### Error Categories by Recovery Path

| Category | User Action | Agent Action |
|----------|-------------|--------------|
| `BAD_REQUEST` | Fix request format | Parse error, check docs |
| `VALIDATION_ERROR` | Fix field values | Read `details` field |
| `NOT_FOUND` | Verify resource ID | List endpoint or retry |
| `INTERNAL_ERROR` | Wait and retry | Exponential backoff |
| `CONFLICT` | Refresh resource | Re-fetch then retry |
| `UNAUTHORIZED` | Login/refresh token | Auth flow |

### Gaps and Signal Debt

| Gap | Current State | Future Work |
|-----|---------------|-------------|
| Request tracing | Request ID in errors only | OpenTelemetry integration |
| Metrics | None | Prometheus endpoints |
| Audit logging | None | Critical operation logging |
| Health dependencies | DB only | External service checks |

### Observability-Related Files

- [CODE: api/handlers/errors.go] - Error logging and response formatting
- [CODE: api/main.go] - Request logging middleware
- [DOC: docs/internal/ERROR_SEMANTICS.md] - Error category documentation

## Seam Testing Strategy

### Testing Pyramid

```
          ╱╲
         ╱E2E╲           ← BAS workflows (pending - critical paths only)
        ╱──────╲
       ╱Integration╲      ← Testcontainers (pending - repository layer)
      ╱──────────────╲
     ╱   Unit Tests    ╲   ← 480+ tests (domain, handlers, components)
    ╱────────────────────╲
```

### Seam Coverage Matrix

| Seam | Test Layer | Test Count | Pattern | Status |
|------|------------|------------|---------|--------|
| Domain | Unit | ~60 | Table-driven, boundary analysis | ✅ Complete |
| Config | Unit | 8 | Env override testing | ✅ Complete |
| Handlers | Unit | ~70 | Mock repositories, httptest | ✅ Complete |
| Repository | Integration | 5 | Interface verification | ⚠️ Partial |
| CLI | Unit | ~36 | Command validation, response parsing | ✅ Complete |
| UI Components | Unit | 27 | React Testing Library, vi.mock | ✅ Complete |
| UI Pages | Unit | 91 | Route-level, mutation testing | ✅ Complete |
| API Client | Unit | 30 | Fetch mocking, error handling | ✅ Complete |
| E2E Flows | E2E | 0 | BAS workflows | ⏳ Pending |

### Seam Test Details

1. **Domain seam**: Unit tests with no dependencies
   - Location: [CODE: api/domain/tasks/task_test.go], [CODE: api/domain/projects/project_test.go], [CODE: api/domain/notes/note_test.go]
   - Pattern: Table-driven tests, boundary value analysis, equivalence partitioning
   - Coverage: Status validation, factory functions, update operations
   - Status: ✅ Implemented

2. **Repository seam**: Integration tests with testcontainers
   - Location: To be added in `api/repository/*_test.go`
   - Pattern: Real database in Docker, transaction rollback
   - Current: Interface verification tests only
   - Status: ⚠️ Partial (interface tests exist, real DB tests pending)

3. **Handler seam**: HTTP tests with mock repositories
   - Location: [CODE: api/handlers/tasks_test.go], [CODE: api/handlers/projects_test.go], [CODE: api/handlers/notes_test.go]
   - Pattern: External test package (`handlers_test`), mock injection, httptest
   - Mocks: [CODE: api/internal/mocks/repository.go]
   - Coverage: CRUD operations, filtering, pagination, error responses
   - Status: ✅ Implemented

4. **UI seam**: Component tests with mocked API
   - Location: [CODE: ui/src/App.test.tsx], [CODE: ui/src/pages/TaskDetail.test.tsx], [CODE: ui/src/pages/ProjectDetail.test.tsx]
   - Pattern: Vitest + React Testing Library, vi.mock for API
   - Utilities: `ui/src/test-utils/`
   - Coverage: Loading/error/empty states, CRUD flows, status cycling, delete confirmation
   - Status: ✅ Implemented

5. **Full stack**: E2E tests through browser automation
   - Location: `bas/` directory (workflows to be added)
   - Pattern: BAS workflows with selector registry
   - Selector Registry: [CODE: ui/src/consts/selectors.ts]
   - Status: ⏳ Pending (infrastructure ready, workflows not yet defined)

### Test Isolation Patterns

| Concern | Go Pattern | TypeScript Pattern |
|---------|------------|-------------------|
| **Database** | Mock repository | vi.mock for API calls |
| **HTTP** | httptest.ResponseRecorder | None (pure unit) |
| **Time** | Not needed (no time-dependent logic) | Not needed |
| **Randomness** | Not needed (no random generation) | Not needed |
| **File system** | Not needed | Not needed |
| **Cleanup** | t.Cleanup() | afterEach cleanup() |

### Test Data Management

**Go Fixtures** ([CODE: api/internal/testutil/fixtures.go]):
- Factory pattern with chainable builders
- Immutable defaults, explicit overrides
- Separate factories for Task, Project, Note

**TypeScript Factories** ([CODE: ui/src/test-utils/factories.ts]):
- Function factories with partial overrides
- Types imported from `lib/api.ts` (single source of truth)
- createMockTask, createMockProject, createMockNote, createMockListResponse


## Test Infrastructure

### Go Test Infrastructure

| Component | Location | Purpose |
|-----------|----------|---------|
| Test helpers | [CODE: api/internal/testutil/helpers.go] | HTTP assertions, JSON utilities |
| Fixtures | [CODE: api/internal/testutil/fixtures.go] | Factory functions for test data |
| Mocks | [CODE: api/internal/mocks/repository.go] | In-memory repository implementations |

### TypeScript Test Infrastructure

| Component | Location | Purpose |
|-----------|----------|---------|
| Test setup | [CODE: ui/src/test-utils/setup.ts] | Global mocks (ResizeObserver, matchMedia) |
| Render helper | [CODE: ui/src/test-utils/renderWithProviders.tsx] | QueryClient provider wrapping |
| Factories | [CODE: ui/src/test-utils/factories.ts] | Mock data creation |

### Mock Design Patterns

All mocks follow these patterns for testability:

1. **Builder pattern**: Chainable methods for configuration
   ```go
   repo := mocks.NewMockTaskRepository().
       WithTask(testutil.NewTaskFactory().Build()).
       WithCreateError(errors.New("db error"))
   ```

2. **Call tracking**: Count and inspect mock invocations
   ```go
   if repo.CreateCallCount() != 1 {
       t.Error("expected Create to be called once")
   }
   ```

3. **Error injection**: Configurable error returns for testing error paths
   ```go
   repo.WithFindError(errors.New("not found"))
   ```

## Change Axes

This section documents the primary ways the scenario is likely to evolve and how localized those changes are.

### Identified Change Axes

| Change Axis | Current Cost | Extension Point | Notes |
|-------------|--------------|-----------------|-------|
| **Add new domain entity** | Low | Create `api/domain/<name>/` package following tasks pattern | Well-localized: add domain, repository, handler |
| **Change validation rules** | Low | [CODE: api/domain/rules.go] | Single source of truth; domain packages import limits |
| **Add status values** | Medium | Domain package + schema.sql | Must sync database CHECK constraints |
| **Add new error category** | Low | [CODE: api/handlers/errors.go] | Well-documented extension points |
| **Add filter to list endpoint** | Low | Handler + ListFilter struct | Follow existing pattern |
| **Change pagination defaults** | Low | [CODE: api/config/config.go] | Environment variable override |
| **Add API version (v2)** | High | Router + handlers | No versioning strategy yet |
| **Add authentication** | High | Middleware + all handlers | No auth infrastructure |

### Stable Core vs Volatile Edges

**Stable Core** (rarely needs changing):
- Repository interfaces [CODE: api/repository/repository.go]
- Error response shape [CODE: api/handlers/errors.go#APIError]
- Health check endpoint
- Pagination utilities [CODE: api/handlers/pagination.go]
- Centralized rules [CODE: api/domain/rules.go] (structure stable, values may change)

**Volatile Edges** (expected to evolve):
- Validation limit values in rules.go (business requirements change)
- Status values (workflow evolution)
- List filters (reporting needs)
- Config defaults (deployment tuning)

### Extension Point Patterns

1. **New Entity**: Copy existing domain package structure
   ```
   api/domain/<entity>/
   ├── <entity>.go       # Entity, CreateInput, UpdateInput, ListFilter
   └── <entity>_test.go  # Table-driven tests
   ```

2. **New Filter**: Add field to ListFilter, update handler, update repository
   ```go
   // In domain package
   type ListFilter struct {
       ExistingField *string
       NewField      *string  // Add here
   }
   ```

3. **New Status**: Add constant, update Validate(), update schema CHECK

## Decision Points

This section documents where the system makes choices between alternatives.

### Major Decision Points

| Decision | Location | Criteria | Outcomes |
|----------|----------|----------|----------|
| **Status validation** | Domain packages | `Validate()` method | Allow/reject status value |
| **Priority defaulting** | [CODE: api/domain/tasks/task.go#NewTask] | `priority == 0` | Use Medium priority |
| **Color format validation** | [CODE: api/domain/rules.go#IsValidHexColor] | Regex pattern | Allow/reject color |
| **Pagination bounds** | [CODE: api/handlers/pagination.go] | Config limits | Clamp to max, apply default |
| **Error category selection** | Handler methods | Error type/message | Map to error code |
| **CORS origin matching** | [CODE: api/main.go#isOriginAllowed] | Pattern matching | Allow/block origin |
| **Retry eligibility** | [CODE: api/handlers/errors.go#retryableErrors] | Error code map | Mark retryable/not |

### Decision Helpers

Shared validation and decision helpers in [CODE: api/domain/rules.go]:

```go
// Validation limits - single source of truth
limits := domain.DefaultValidationLimits()
limits.TaskTitleMaxLength   // 255
limits.ProjectNameMaxLength // 100
limits.NoteContentMaxLength // 10000

// Priority validation
IsPriorityValid(p int) bool

// Color validation
IsValidHexColor(color string) bool

// Status constants (for switch statements)
TaskStatuses.Pending, TaskStatuses.InProgress, ...
ProjectStatuses.Active, ProjectStatuses.Paused, ...
```

Domain packages (tasks, projects, notes) import `domain.DefaultValidationLimits()` rather than
defining their own constants. This ensures a single source of truth for validation limits.

### Decision Categories

**Input Validation Decisions**:
- What input formats are valid?
- What lengths are acceptable?
- What values are allowed for enums?
- Location: Domain packages, rules.go

**Default Value Decisions**:
- What happens when optional fields are omitted?
- Location: Domain factory functions (NewTask, NewProject)

**Error Mapping Decisions**:
- What error code corresponds to each failure mode?
- What recovery hints should be shown?
- Location: [CODE: api/handlers/errors.go]

**Configuration Decisions**:
- What are the runtime-tunable parameters?
- What are sensible defaults?
- Location: [CODE: api/config/config.go]

### UI Decision Points

**Status Cycling Order** (presentation layer decision):
- Task status cycling: `pending → in_progress → completed → pending`
- Project status cycling: `active → paused → complete → active`
- Location: [CODE: ui/src/pages/TaskDetail.tsx], [CODE: ui/src/pages/ProjectDetail.tsx]
- Rationale: Matches typical workflow progression; archived is excluded from cycling

**Status Visual Mapping** (presentation layer decision):
- Each status maps to an icon, color, and label for consistent display
- Location: Component-local constants (`statusIcons`, `statusColors`, `statusLabels`)
- Design choice: Explicit mappings per component rather than shared module to keep pages copy-pasteable

**Priority Display Mapping** (presentation layer decision):
- Priority 1-5 maps to labels: Low, Medium, High, Urgent, Critical
- Priority 1-5 maps to colors: slate, yellow, orange, red, dark-red
- Location: [CODE: ui/src/pages/TaskDetail.tsx#priorityLabels]

**Delete Confirmation** (UX decision):
- All delete operations require confirmation dialog
- Location: [CODE: ui/src/components/ConfirmDialog.tsx]
- Rationale: Destructive actions should be explicit

### Repository Decision Points

**Not Found Handling** (data access decision):
- `FindByID` returns `(nil, nil)` when no rows found - handler decides semantics
- `Update/Delete` return typed `ErrNotFound` when rows affected = 0
- Location: [CODE: api/repository/tasks_postgres.go#FindByID], [CODE: api/repository/repository.go#ErrNotFound]
- Rationale: Type-safe error checking over string comparison; handler layer interprets meaning

**Default Pagination Fallback** (query building decision):
- List queries apply `limit=20, offset=0` when invalid/missing values provided
- Location: [CODE: api/repository/tasks_postgres.go#List] (lines 97-104)
- Rationale: Defense-in-depth; config-layer should clamp, but repository has safe defaults

**Sort Order** (query stability decision):
- All list operations sort by `created_at DESC` (newest first)
- Location: [CODE: api/repository/tasks_postgres.go#List], [CODE: api/repository/projects_postgres.go#List]
- Rationale: Deterministic ordering for pagination; newest items typically most relevant

**NULL Handling for Optional Fields** (type mapping decision):
- `project_id` stored as `NULL` when empty string (foreign key constraint compatibility)
- `due_date` stored as `NULL` when nil pointer
- Location: [CODE: api/repository/tasks_postgres.go#Create] (lines 27-30)
- Rationale: PostgreSQL NULL semantics for optional relationships vs empty strings

**Query Builder Parameterization** (security decision):
- All filter values pass through `$N` parameters, never string interpolation
- Column names are code constants, not user input
- Location: [CODE: api/repository/query_builder.go#addCondition]
- Rationale: SQL injection prevention by construction

### Future Decision Points (Not Yet Implemented)

| Decision | When Needed | Considerations |
|----------|-------------|----------------|
| Status transitions | Workflow enforcement | State machine vs. free transitions |
| Authorization | Multi-user support | RBAC, ownership, sharing |
| Rate limiting | Production scale | Per-user, per-endpoint limits |
| Soft delete | Data retention | Audit trails, recovery |

## CLI Seam

**Location**: `cli/`
**Responsibility**: Thin wrapper over API for command-line operations

The CLI provides full feature parity with the API, following the cli-steer pattern of being a thin wrapper that delegates all business logic to the API.

### CLI Architecture

```
cli/
├── main.go              # Entry point
├── app.go               # App struct, command registration, handlers
├── app_test.go          # CLI tests
└── install.sh           # Cross-platform installer
```

### Command Structure

| Domain | Commands | API Endpoints |
|--------|----------|---------------|
| Health | `status` | GET /health |
| Tasks | `task list`, `task get`, `task create`, `task update`, `task delete` | /api/v1/tasks |
| Projects | `project list`, `project get`, `project create`, `project update`, `project delete` | /api/v1/projects |
| Notes | `note list`, `note get`, `note create`, `note update`, `note delete` | /api/v1/tasks/{id}/notes, /api/v1/notes/{id} |
| Config | `configure` | N/A (local config) |

### CLI Design Patterns

**Uses cli-core for:**
- ScenarioApp scaffolding with global flags
- APIClient for HTTP requests
- StandardScenarioEnv for env var derivation
- Port detection via DetectPortFromVrooli
- Stale binary detection and auto-rebuild
- ParseInterspersed for flag/arg handling

**Output modes:**
- Default: Human-friendly formatted output
- `--json`: Machine-readable JSON for scripting

**Boundary enforcement:**
- CLI never implements business logic
- All operations go through API
- Validation is minimal (just required args)
- Error handling follows API error shapes

### CLI Decision Points

**Output Mode Selection** (presentation layer decision):
- Default output: Human-friendly formatted text with summaries
- `--json` flag: Raw JSON for scripting and automation
- Location: Each command handler checks `*jsonOutput` flag
- Rationale: Human-readable by default, machine-parseable on demand

**Priority Label Mapping** (presentation layer decision):
- Priority 1 → "Low", Priority 2 → "Medium", Priority 3 → "High"
- Location: [CODE: cli/app.go#cmdTaskList], [CODE: cli/app.go#cmdTaskGet]
- Design choice: Explicit map literals in each command for clarity and copy-pasteability

**List Pagination Defaults** (UX decision):
- Default limit: 20 items per page
- Default offset: 0 (start from beginning)
- Location: Each list command's flag definitions
- Rationale: Reasonable default for terminal display width

**Content Truncation** (display decision):
- Note content truncated to 50 characters in list view
- Location: [CODE: cli/app.go#cmdNoteList]
- Rationale: Prevents long notes from breaking list layout

**ID Display Truncation** (display decision):
- UUIDs displayed as first 8 characters (e.g., `abc12345`)
- Location: List command formatters
- Rationale: Full UUIDs clutter output; 8 chars sufficient for disambiguation

### Key Files

- [CODE: cli/app.go] - Command registration and handlers
- [CODE: cli/app_test.go] - Command validation tests
- [DOC: docs/reference/cli-commands.md] - CLI reference documentation

## UI Stability Seam

**Location**: [CODE: ui/src/App.tsx]
**Responsibility**: React stability patterns for crash prevention

The UI implements React Stability skill patterns to prevent runtime crashes and ensure graceful degradation.

### TypeScript Safety

- [CODE: ui/tsconfig.node.json] - `strict: true` and `noUncheckedIndexedAccess: true` enabled
- [CODE: ui/eslint.config.js] - Safety-critical ESLint rules for:
  - `react-hooks/rules-of-hooks` - Prevents React Error #310
  - `@typescript-eslint/no-non-null-assertion` - Prevents null assertion operator
  - `import/no-cycle` - Prevents circular dependencies

### Error Boundaries

- [CODE: ui/src/components/ErrorBoundary.tsx] - Reusable error boundary component
- Wraps major UI sections to isolate failures
- Provides fallback UI with retry capability

### Iframe Interop

- [CODE: ui/src/main.tsx] - Bridge initialization with idempotency guard
- [CODE: ui/vite.config.ts] - Relative base (`./`) for proxy/tunnel contexts
- [CODE: ui/src/styles.css] - `h-full` height chain for iframe compatibility
- [CODE: ui/src/hooks/useKeyboardShortcuts.ts] - Centralized keyboard handler with iframe relay

### Key Patterns

| Pattern | Location | Purpose |
|---------|----------|---------|
| Error Boundary | [CODE: ui/src/components/ErrorBoundary.tsx] | Isolate component crashes |
| Iframe Guard | [CODE: ui/src/main.tsx] | Prevent double bridge init |
| Height Chain | [CODE: ui/src/styles.css] | Correct sizing in iframe |
| Keyboard Relay | [CODE: ui/src/hooks/useKeyboardShortcuts.ts] | Host shortcut communication |

## Related Documentation

- [DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md] - Test architecture details
- [DOC: docs/internal/ERROR_SEMANTICS.md] - Error handling patterns
- [DOC: docs/internal/TEMPORAL-FLOWS.md] - Async operations and time-based behavior
- [DOC: docs/internal/INVARIANTS.md] - Replay safety and idempotency patterns
- [DOC: docs/concepts/ARCHITECTURE.md] - Domain mental model
- [DOC: docs/reference/api-endpoints.md] - REST API reference
- [DOC: docs/reference/cli-commands.md] - CLI reference
- [DOC: docs/reference/data-model.md] - Database schema
- [DOC: docs/internal/STORAGE_AUDIT.md] - Storage compliance

## React Coherence Seam

**Location**: [CODE: ui/src/App.tsx]
**Responsibility**: Architectural coherence for maintainability and theming

The UI follows React Coherence skill patterns to ensure the codebase is well-organized, consistent in patterns, and free from avoidable duplication.

### Code Organization

```
ui/src/
├── components/
│   ├── ErrorBoundary.tsx    # Shared error boundary
│   └── ui/
│       └── button.tsx        # Primitive with CVA variants
├── hooks/
│   └── useKeyboardShortcuts.ts  # Central shortcut handler
├── lib/
│   ├── api.ts               # API client (interop slot [F])
│   └── utils.ts             # Utility functions
├── consts/
│   └── selectors.ts         # Test selectors registry
├── test-utils/              # Test infrastructure
├── App.tsx                  # App shell
├── main.tsx                 # Entry point
└── styles.css               # Global styles with tokens
```

### Ownership Rules

| Location | Criteria | Examples |
|----------|----------|----------|
| `components/ui/` | Lowest-level reusable UI atoms | Button (has CVA variants) |
| `components/` | Non-design-system shared widgets | ErrorBoundary |
| `hooks/` | Domain-agnostic hooks | useKeyboardShortcuts |
| `lib/` | Services and utilities | api.ts, utils.ts |

### State Architecture

- **Current pattern**: Server state only (React Query)
- **App-wide stores**: None (not needed for current scope)
- **Recommendation**: Follow state location decision table when expanding

### Styling System

- **Token coverage**: Tailwind defaults (no custom tokens yet)
- **Primitive variant coverage**: Good (Button uses CVA)
- **Pattern**: CVA variants for primitives, Tailwind utilities for composition

### Key Files

- [CODE: ui/src/components/ui/button.tsx] - Button primitive with CVA variants
- [CODE: ui/src/lib/utils.ts] - `cn()` utility for className merging
- [CODE: ui/src/styles.css] - Global styles with height chain
- [DOC: docs/internal/COHERENCE-NOTES.md] - Full coherence audit findings

## E2E Testing Readiness

The scenario is ready for E2E test implementation with the following infrastructure in place:

### Selector Registry

Complete selector registry at [CODE: ui/src/consts/selectors.ts] with:

**Literal Selectors (static):**
- `layout.*` - App shell (header, nav, health indicator)
- `dashboard.*` - Dashboard page (stats, recent tasks, quick actions)
- `tasks.*` - Tasks page (form, list, loading/error/empty states)
- `projects.*` - Projects page (form, grid, loading/error/empty states)
- `taskDetail.*` - Task detail page (edit, delete, notes section)
- `projectDetail.*` - Project detail page (edit, delete, tasks section)
- `notes.*` - Notes section (form, list, loading/error/empty states)
- `confirmDialog.*` - Confirmation dialog (confirm, cancel buttons)

**Dynamic Selectors (parameterized):**
- `tasks.rowById(id)` - Task row in list
- `tasks.statusToggleById(id)` - Task status toggle
- `tasks.deleteById(id)` - Task delete button
- `projects.cardById(id)` - Project card
- `projects.statusToggleById(id)` - Project status toggle
- `notes.itemById(id)` - Note item
- `notes.deleteById(id)` - Note delete button

### BAS Workflow Structure

```
bas/
├── registry.json        # Auto-generated manifest (empty)
├── README.md            # Workflow authoring guide
├── actions/             # Atomic operations (login, navigate)
├── cases/               # Test cases with assertions
├── flows/               # Multi-step user journeys
└── seeds/               # Optional setup data
```

### Priority E2E Flows (Recommended)

When implementing E2E tests, prioritize these critical user journeys:

| Flow | Description | Validates |
|------|-------------|-----------|
| Task CRUD | Create → Read → Update → Delete task | REQ-P0-001a, REQ-P1-001a |
| Project CRUD | Create → Read → Update → Delete project | REQ-P0-002a, REQ-P1-002a |
| Task Status Cycling | Click status through pending → in_progress → completed | REQ-P1-002b |
| Task-Project Association | Create project → Create task with project → View project tasks | REQ-P1-003a |
| Notes Management | Open task detail → Add note → Delete note | REQ-P1-004a |

### E2E Testing Blockers

None - infrastructure is ready. Workflows need to be authored.

## Last Updated

2026-03-11 - Phase 20 testing seams enhancement (testing pyramid, seam coverage matrix, E2E readiness documentation)
2026-03-11 - Added Repository Decision Points section for error handling, pagination, sort order, NULL handling, parameterization (Phase 19.3 final)
2026-03-11 - Added CLI Decision Points section for output mode, display mappings, truncation (Phase 19.2)
2026-03-11 - Added UI Decision Points section for status cycling, display mappings (Phase 19)
2026-03-11 - Spec sync verification, confirmed architecture alignment (Phase 15.2)
2026-03-11 - Added TEMPORAL-FLOWS.md and INVARIANTS.md references (Phase 12)
2026-03-11 - Added React Coherence Seam (Phase 10)
2026-03-11 - Added UI Stability Seam with React safety patterns (Phase 9)
2026-03-11 - Added CLI seam documentation with full API parity (Phase 8)
2026-03-11 - Consolidated validation limits to use domain.DefaultValidationLimits() (Phase 7 iteration 2)
2026-03-11 - Added change axes and decision points documentation (Phase 7)
