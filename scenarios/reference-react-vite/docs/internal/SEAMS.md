# Seams Documentation

This document identifies the architectural seams in the reference-react-vite scenario, following the seam-discovery-and-enforcement steer skill patterns.

## Overview

The reference-react-vite scenario demonstrates a task/project management domain with clear architectural boundaries between presentation, coordination, domain rules, and data access.

## Identified Seams

### 1. HTTP Handler Seam

**Location**: [CODE: api/handlers/]
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

**Location**: [CODE: api/domain/]
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

**Location**: [CODE: api/repository/]
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
| Presentation | [CODE: api/handlers/] | HTTP orchestration |
| Domain | [CODE: api/domain/] | Business rules |
| Data Access | [CODE: api/repository/] | Database operations |
| Infrastructure | [CODE: api/main.go] | Server setup, cross-cutting |

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
| Logging | [CODE: api/main.go:133-139] | Simple request logging |
| CORS | [CODE: api/main.go:82-108] | Configurable via env var |
| Request ID | [CODE: api/handlers/errors.go#getRequestID] | Generated per-request |
| Error format | [CODE: api/handlers/errors.go#APIError] | Consistent API error shape |
| Recovery | [CODE: api/main.go#Handler] | Gorilla handlers recovery |

## Seam Testing Strategy

1. **Domain seam**: Unit tests with no dependencies
   - Location: [CODE: api/domain/tasks/task_test.go], [CODE: api/domain/projects/project_test.go], [CODE: api/domain/notes/note_test.go]
   - Pattern: Table-driven tests, boundary value analysis, equivalence partitioning
   - Status: ✅ Implemented

2. **Repository seam**: Integration tests with testcontainers
   - Location: To be added in `api/repository/*_test.go`
   - Pattern: Real database in Docker, transaction rollback
   - Status: Pending

3. **Handler seam**: HTTP tests with mock repositories
   - Location: [CODE: api/handlers/tasks_test.go]
   - Pattern: External test package (`handlers_test`), mock injection, httptest
   - Mocks: [CODE: api/internal/mocks/repository.go]
   - Status: ✅ Implemented

4. **UI seam**: Component tests with mocked API
   - Location: [CODE: ui/src/App.test.tsx]
   - Pattern: Vitest + React Testing Library, vi.mock for API
   - Utilities: [CODE: ui/src/test-utils/]
   - Status: ✅ Implemented

5. **Full stack**: E2E tests through HTTP endpoints
   - Location: To be added in `bas/` directory
   - Pattern: Playwright/Cypress with test-genie
   - Status: Pending


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

## Related Documentation

- [DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md] - Test architecture details
- [DOC: docs/concepts/ARCHITECTURE.md] - Domain mental model
- [DOC: docs/reference/api-endpoints.md] - REST API reference
- [DOC: docs/reference/data-model.md] - Database schema
- [DOC: docs/internal/STORAGE_AUDIT.md] - Storage compliance

## Last Updated

2026-03-11 - Added test infrastructure seams and mock patterns
