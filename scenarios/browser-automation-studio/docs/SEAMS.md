# Seam Discovery and Enforcement

This document catalogs the architectural seams in browser-automation-studio for testability and maintainability.

## Overview

**Seams** are intentional architectural boundaries where behavior can be substituted for testing or extension. Well-defined seams enable:
- Unit testing without external dependencies
- Contract-based integration testing
- Easy mocking and stubbing
- Clear ownership boundaries

## Seam Categories

| Status | Meaning |
|--------|---------|
| Strong | Interface defined, test doubles exist, compile-time enforcement |
| Good | Interface defined, some test doubles, may need enforcement |
| Weak | Interface exists but not consistently used or easy to bypass |
| Missing | No interface, direct concrete dependencies |

---

## Go API Seams

### 1. AutomationEngine Seam (Strong)

**Location:** `api/automation/engine/engine.go`

**Interface:**
```go
type AutomationEngine interface {
    Name() string
    Capabilities(ctx context.Context) (contracts.EngineCapabilities, error)
    StartSession(ctx context.Context, spec SessionSpec) (EngineSession, error)
}

type EngineSession interface {
    Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error)
    Reset(ctx context.Context) error
    Close(ctx context.Context) error
}

type Factory interface {
    Resolve(ctx context.Context, name string) (AutomationEngine, error)
}
```

**Test Doubles:**
- `executor/simple_executor_test.go`: `fakeEngine`, `fakeSession`, `fakeEngineFactory`

**Status:** Strong
- Clean interface abstraction
- Factory pattern for DI
- Comprehensive test doubles
- Used consistently throughout executor

---

### 2. Executor Seam (Strong)

**Location:** `api/automation/executor/executor.go`

**Interface:**
```go
type Executor interface {
    Execute(ctx context.Context, req Request) error
}

type WorkflowResolver interface {
    GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*database.Workflow, error)
}

type PlanCompiler interface {
    Compile(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error)
}
```

**Test Doubles:**
- `executor/simple_executor_test.go`: `stubWorkflowResolver`, `subflowPlanCompiler`

**Status:** Strong
- Clean separation of orchestration from execution
- All dependencies injected via `Request` struct
- Extensive unit test coverage

---

### 3. Recorder Seam (Good)

**Location:** `api/automation/recorder/recorder.go`

**Interface:**
```go
type Recorder interface {
    RecordStepOutcome(ctx context.Context, plan contracts.ExecutionPlan, outcome contracts.StepOutcome) (RecordResult, error)
    RecordTelemetry(ctx context.Context, plan contracts.ExecutionPlan, telemetry contracts.StepTelemetry) error
    MarkCrash(ctx context.Context, executionID uuid.UUID, failure contracts.StepFailure) error
}

type ExecutionRepository interface {
    CreateExecutionStep(ctx context.Context, step *database.ExecutionStep) error
    CreateExecutionArtifact(ctx context.Context, artifact *database.ExecutionArtifact) error
}
```

**Test Doubles:**
- `executor/simple_executor_test.go`: `memoryRecorder`

**Status:** Strong
- Narrow interface (ExecutionRepository) reduces coupling
- DBRecorder uses the interface with compile-time enforcement
- MemoryRecorder keeps executor tests hermetic

---

### 4. EventSink Seam (Strong)

**Location:** `api/automation/events/sink.go`

**Interface:**
```go
type Sink interface {
    Publish(ctx context.Context, event contracts.EventEnvelope) error
    Limits() contracts.EventBufferLimits
    CloseExecution(executionID uuid.UUID)
}
```

**Test Doubles:**
- `events/memory_sink.go`: `MemorySink` (production-quality test double)

**Status:** Strong
- MemorySink is well-documented and feature-complete
- Used extensively in tests
- Compile-time enforcement exists

---

### 5. Database Repository Seam (Good)

**Location:** `api/database/repository.go`

**Interface:**
```go
type Repository interface {
    // 20+ methods for Project, Workflow, Execution, etc.
    CreateProject(ctx context.Context, project *Project) error
    GetWorkflow(ctx context.Context, id uuid.UUID) (*Workflow, error)
    // ... etc
}
```

**Test Doubles:**
- `handlers/handlers_test.go`: `mockRepository`

**Status:** Good
- Comprehensive interface with compile-time enforcement on the concrete repository
- Mock exists but is verbose (all methods return nil/error)
- Consider: Interface segregation into smaller role interfaces to cut setup noise

---

### 6. Database Backend Dialect Seam (Good)

**Location:** `api/database/connection.go`, `api/database/dialect.go`, `api/automation/executor/integration_test.go`

**Controls:**
- `BAS_DB_BACKEND` selects runtime backend (`postgres` default, `sqlite` opt-in)
- `BAS_TEST_BACKEND` toggles test backend (Postgres testcontainer vs SQLite temp file)
- `database.Dialect` and `DialectProvider` drive placeholder rebinding and value encoding

**Status:** Good
- `NewConnection` provisions Postgres or SQLite and sets the dialect provider for value types
- Executor and repository tests resolve the backend from `BAS_TEST_BACKEND` (preferred) or `BAS_DB_BACKEND`, so a sqlite runtime config no longer spins up a Postgres container by surprise
- Dialect-aware truncate/reset logic keeps integration tests portable instead of relying on Postgres-only TRUNCATE/`$1` SQL
- SQLite paths mirror resource defaults via `sqliteDSN`; Postgres path sticks to the container DSN
- Test harness forces `BAS_DB_BACKEND` to the resolved backend and restores BAS/POSTGRES/SQLITE envs after each run to avoid leakage between suites

---

### 7. Storage Seam (Good)

**Location:** `api/storage/interface.go`

**Interface:**
```go
type StorageInterface interface {
    GetScreenshot(ctx context.Context, objectName string) (io.ReadCloser, *minio.ObjectInfo, error)
    StoreScreenshot(ctx context.Context, executionID uuid.UUID, stepName string, data []byte, contentType string) (*ScreenshotInfo, error)
    HealthCheck(ctx context.Context) error
    // ...
}
```

**Test Doubles:**
- `storage/memory.go`: `MemoryStorage`

**Status:** Strong
- Clean interface with compile-time checks on all implementations
- In-memory double enables handler/service tests without filesystem or MinIO
- Selector helpers keep naming/prefix decisions encapsulated

---

### 8. WebSocket Hub Seam (Good)

**Location:** `api/websocket/interface.go`

**Interface:**
```go
type HubInterface interface {
    ServeWS(conn *websocket.Conn, executionID *uuid.UUID)
    BroadcastEnvelope(event any)
    GetClientCount() int
    Run()
    CloseExecution(executionID uuid.UUID)
}
```

**Status:** Good
- Interface exists
- Tests use real Hub in some cases
- **Action Needed:** Add `MockHub` for isolated testing

---

### 9. WorkflowService Seam (Weak)

**Location:** `api/handlers/handler.go` and `api/services/workflow/service.go`

**Interface:** `api/services/workflow/interfaces.go` (CatalogService, ExecutionService, ExportService)

**Status:** Good
- Interfaces live in `api/services/workflow/interfaces.go` with compile-time checks on `WorkflowService`
- Handler uses `NewHandlerWithDeps` to accept injected services, keeping transport thin
- Event sink wiring now flows through an injected factory + `wsHub.HubInterface` rather than a concrete hub
- Remaining risk: interface breadth (20+ methods) still high; consider narrowing by role

---

### 10. AI Client Seam (Good)

**Location:** `api/services/ai/client.go`

**Interface:** `api/services/ai/interface.go`
- `AIClient` with `ExecutePrompt` and `Model`
- Compile-time enforcement on `OpenRouterClient` and `MockAIClient`

**Status:** Good
- Shell-out integration is confined to `OpenRouterClient`
- `MockAIClient` enables workflow tests without the executable
- Injected via `WorkflowServiceOptions`

---

### 11. HTTP Client Seam in PlaywrightEngine (Good)

**Location:** `api/automation/engine/playwright_engine.go`

**Status:** Good
- `HTTPDoer` interface defined in `automation/engine/http.go` with compile check on `http.Client`
- `NewPlaywrightEngineWithHTTPClient` accepts injected client for tests (can use `httptest.Server`)
- Default constructor still wires a long-timeout client for real driver runs

---

## TypeScript Playwright-Driver Seams

### 12. InstructionHandler Seam (Strong)

**Location:** `playwright-driver/src/handlers/base.ts`

**Interface:**
```typescript
interface InstructionHandler {
    getSupportedTypes(): string[];
    execute(instruction: CompiledInstruction, context: HandlerContext): Promise<HandlerResult>;
}
```

**Status:** Strong
- Clean strategy pattern
- Handler registry for dispatch
- Base class provides common utilities

---

### 13. SessionManager Seam (Weak)

**Location:** `playwright-driver/src/session/manager.ts`

**Current Implementation:**
```typescript
class SessionManager {
    private browser: Browser | null = null;  // Direct Playwright dependency

    private async getBrowser(): Promise<Browser> {
        this.browser = await chromium.launch(...);  // Hardcoded
    }
}
```

**Status:** Weak
- No interface for browser launch
- Direct `chromium.launch()` call
- Hard to test session management logic independently

**Action Needed:**
- Extract `BrowserFactory` interface
- Inject browser creation dependency

### 14. RecordingBuffer Seam (Strong)

**Location:** `playwright-driver/src/recording/buffer.ts`

**Interface:**
```typescript
// Pure functions for action buffer management
initRecordingBuffer(sessionId: string): void
bufferRecordedAction(sessionId: string, action: RecordedAction): void
getRecordedActions(sessionId: string): RecordedAction[]
getRecordedActionCount(sessionId: string): number
clearRecordedActions(sessionId: string): void
removeRecordedActions(sessionId: string): void
```

**Status:** Strong
- Module-level state encapsulated in dedicated file
- Clean functional interface
- No dependencies on presentation layer (routes)
- Session layer imports from recording, not routes

**History:**
- Previously, action buffering state lived in `routes/record-mode.ts` as a module-level Map
- This caused `session/manager.ts` to import from `routes/` (presentation layer importing from route layer)
- Refactored to move state to `recording/buffer.ts` where it belongs (domain/coordination layer)

---

### 15. AI Element Analyzer Seam (Good)

**Location:** `api/handlers/ai/ai_analysis.go`

**Interface:**
```go
type ElementAnalyzer interface {
    Analyze(ctx context.Context, url, intent string) ([]ElementInfo, error)
}
```

**Status:** Good
- HTTP handler now delegates to the analyzer, keeping transport separate from DOM extraction and Ollama prompting
- Default implementation (`AIElementAnalyzer`) wires `DOMExtractor` and `OllamaClient`, but tests inject `mockElementAnalyzer` for isolated domain coverage
- `WithElementAnalyzer` option allows swapping implementations without changing handler wiring

---

### 16. Playwright-Driver Router Seam (Strong)

**Location:** `playwright-driver/src/routes/router.ts`

**Interface:**
```typescript
type RouteHandler = (
  req: IncomingMessage,
  res: ServerResponse,
  params: Record<string, string>
) => Promise<void>;

class Router {
  get(path: string, handler: RouteHandler): void
  post(path: string, handler: RouteHandler): void
  handle(req: IncomingMessage, res: ServerResponse): Promise<boolean>
}
```

**Status:** Strong
- Clean declarative route registration
- Path parameter extraction (`:id` → `params.id`)
- Automatic 404/405 handling
- Separates routing coordination from HTTP server setup

---

### 17. Playwright-Driver Outcome Builder Seam (Strong)

**Location:** `playwright-driver/src/domain/outcome-builder.ts`

**Interface:**
```typescript
function buildStepOutcome(params: BuildOutcomeParams): StepOutcome
function toDriverOutcome(
  outcome: StepOutcome,
  screenshotData?: { base64?: string; ... },
  domSnapshot?: DOMSnapshot
): DriverOutcome
```

**Status:** Strong
- Centralizes outcome construction logic
- Separates domain transformation from route handling
- Clean interface with explicit parameters
- `routes/session-run.ts` delegates to these functions instead of inline building

---

### 18. Playwright-Driver Metrics Server Seam (Strong)

**Location:** `playwright-driver/src/utils/metrics-server.ts`

**Interface:**
```typescript
function createMetricsServer(port: number): Promise<Server>
function closeMetricsServer(server: Server): Promise<void>
```

**Status:** Strong
- Separates metrics endpoint from main HTTP server
- Clean lifecycle functions for startup and shutdown
- Infrastructure concern isolated from business logic

---

## Seam Enforcement Matrix

| Seam | Interface | Test Double | Compile Check | Priority |
|------|-----------|-------------|---------------|----------|
| AutomationEngine | Yes | Yes | Yes | - |
| Executor | Yes | Yes | Yes | - |
| Recorder | Yes | Yes | Yes | - |
| EventSink | Yes | Yes | Yes | - |
| Repository | Yes | Yes | Yes | Medium |
| Database Backend | Env flags `BAS_DB_BACKEND`/`BAS_TEST_BACKEND` + `database.Dialect` (tests default to `BAS_TEST_BACKEND` else `BAS_DB_BACKEND`) | Postgres testcontainer handle, SQLite temp DB in tests | N/A | Medium |
| Storage | Yes | Yes (MemoryStorage) | Yes | - |
| WebSocket Hub | Yes | **Partial** | **Missing** | Medium |
| WorkflowService | Yes | Yes | Yes | Medium |
| AI Client | Yes | Yes | Yes | - |
| HTTP Client (Engine) | Yes | Injectable HTTPDoer | Yes | Medium |
| SessionManager (TS) | **Missing** | **Missing** | N/A | Medium |
| RecordingBuffer (TS) | Yes | N/A (pure functions) | N/A | - |
| AI Element Analyzer | Yes | Yes | Yes (interface injection) | Medium |
| Router (TS) | Yes | N/A (coordination) | N/A | - |
| OutcomeBuilder (TS) | Yes | N/A (pure functions) | N/A | - |
| MetricsServer (TS) | Yes | N/A (infrastructure) | N/A | - |

---

## Enforcement Actions

### Critical Priority

*None currently marked critical.*

### High Priority

1. **WebSocket Mock Hub**
   - Create `MockHub` implementing `HubInterface`
   - Use in handler/service tests to avoid goroutines and sockets

### Medium Priority

2. **Interface Segregation for Repository**
   - Split into `ProjectRepository`, `WorkflowRepository`, `ExecutionRepository`
   - Consumers depend only on what they need

3. **WorkflowService Surface Trim**
   - Consider narrowing Catalog/Execution/Export into smaller interfaces for handler deps and mocks

4. **TypeScript BrowserFactory**
   - Extract browser launch into injectable factory
   - Enable session manager unit testing

5. **HTTP Client Test Double**
   - Add a lightweight `HTTPDoer` stub for Playwright engine unit tests when `httptest.Server` is overkill

---

## Testing Guidelines

### Unit Tests (Mock Seams)
- Use test doubles at every seam boundary
- No I/O, no network, no file system
- Fast execution (<100ms per test)

### Integration Tests (Real Seams)
- Use `testcontainers` for database
- Use `httptest.Server` for HTTP
- Test seam contracts

### E2E Tests (Full Stack)
- Real browser via Playwright driver
- Real database
- Validate complete flows

---

## Adding New Seams

When adding new dependencies:

1. **Define Interface First**
   ```go
   // In component/interface.go
   type MyDependency interface {
       DoThing(ctx context.Context, input Input) (Output, error)
   }
   ```

2. **Add Compile-Time Check**
   ```go
   var _ MyDependency = (*ConcreteImpl)(nil)
   ```

3. **Create Test Double**
   ```go
   // In component/mock.go or testutil/
   type MockMyDependency struct { ... }
   ```

4. **Inject via Constructor/Options**
   ```go
   func NewService(dep MyDependency) *Service { ... }
   ```

5. **Document in This File**
   - Add to appropriate section
   - Update enforcement matrix

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-12-09 | Claude | Boundary of Responsibility Enforcement pass #2: Removed unused BaseHandler.buildOutcome (consolidated in domain/outcome-builder.ts); replaced any types with Page in assertion handler; replaced console.log with injected logger in assertNotExists |
| 2025-12-09 | Claude | Boundary of Responsibility Enforcement: Added Router (#16), OutcomeBuilder (#17), MetricsServer (#18) seams; extracted domain/outcome-builder.ts, utils/metrics-server.ts, routes/router.ts; updated responsibility boundaries |
| 2025-12-09 | Claude | Added RecordingBuffer seam (#14), Playwright-Driver Responsibility Boundaries section; moved action buffer state from routes to recording/buffer.ts |
| 2025-12-09 | Assistant | Hardened test backend resolution (BAS_TEST_BACKEND -> BAS_DB_BACKEND fallback) and env reset to keep Postgres/SQLite toggles aligned |
| 2025-12-09 | Assistant | Documented DB backend seam and executor test harness for Postgres/SQLite |
| 2025-11-29 | Claude | Initial seam discovery and documentation |
| 2025-11-29 | Claude | Added Responsibility Boundaries section, apierror package |

---

## Responsibility Boundaries

### Current Architecture
The codebase follows a layered architecture with clear responsibilities:

| Layer | Location | Responsibility |
|-------|----------|----------------|
| **Entry/Presentation** | `handlers/` | HTTP routing, request/response mapping |
| **Coordination** | `services/workflow/` | Orchestration, business flow |
| **Domain Rules** | `automation/` | Execution contracts, engine abstraction |
| **Infrastructure** | `database/`, `storage/`, `websocket/` | Persistence, storage, real-time |
| **Cross-cutting** | `internal/` | Shared utilities, error handling |

### Cross-cutting Concerns Centralization
Error handling is centralized in `internal/apierror/` to avoid duplication:
- `handlers/errors.go` re-exports from `internal/apierror`
- `handlers/ai/response.go` re-exports from `internal/apierror`
- Both packages use identical error types without code duplication

### Handler Dependency Injection
The `Handler` struct supports explicit dependency injection for testing:

```go
// HandlerDeps holds all dependencies for the Handler
type HandlerDeps struct {
    WorkflowService   WorkflowService
    WorkflowValidator *workflowvalidator.Validator
    Storage           storage.StorageInterface
    RecordingService  recording.RecordingServiceInterface
    RecordingsRoot    string
    ReplayRenderer    replayRenderer
}

// Production usage (wires all dependencies)
handler := handlers.NewHandler(repo, wsHub, log, allowAll, origins)

// Testing usage (inject mocks)
deps := handlers.HandlerDeps{
    WorkflowService: mockWorkflowService,
    Storage:         mockStorage,
    // ...
}
handler := handlers.NewHandlerWithDeps(repo, wsHub, log, allowAll, origins, deps)
```

### Future Restructuring: `handlers/ai`
The `handlers/ai/` package currently contains both HTTP handling AND domain logic:

**Current state (mixed responsibilities):**
- HTTP handlers (GetDOMTree, AnalyzeElements, etc.)
- Domain logic (DOM extraction, element analysis, selector generation)
- Domain types (ElementInfo, SelectorOption, etc.)

**Current mitigation:**
- `AIAnalysisHandler` is now transport-only and defers to an injected `ElementAnalyzer` (defaulting to `AIElementAnalyzer`), so DOM extraction and Ollama prompting can be exercised without HTTP concerns.
- Handler timeouts use `constants.AIAnalysisTimeout` rather than inlined durations to keep cross-cutting configuration centralized.

**Remaining opportunity:**
- Consider moving shared domain types and analysis helpers into a dedicated service package to further decouple HTTP routing from analysis internals, following the same analyzer seam pattern.

---

## Playwright-Driver Responsibility Boundaries

### Architecture Layers

| Layer | Location | Responsibility |
|-------|----------|----------------|
| **Entry/Presentation** | `routes/`, `server.ts`, `middleware/` | HTTP routing, request parsing, response formatting |
| **Coordination** | `routes/session-run.ts`, `routes/router.ts` | Wiring handlers, routing dispatch, sequencing operations |
| **Domain Rules** | `handlers/`, `domain/`, `recording/controller.ts`, `recording/selectors.ts` | Instruction execution, outcome building, action replay, selector generation |
| **Integration** | `session/`, `telemetry/`, Playwright API | Browser management, context building, screenshot capture |
| **Cross-cutting** | `utils/`, `config.ts`, `constants.ts` | Logging, errors, metrics, metrics server, configuration |

### Key Design Decisions

1. **Handlers follow Strategy Pattern**
   - Each instruction type has a dedicated handler implementing `InstructionHandler`
   - `HandlerRegistry` dispatches by type
   - `BaseHandler` provides shared utilities

2. **Recording Buffer in Domain Layer**
   - Action buffering lives in `recording/buffer.ts` (domain/coordination)
   - NOT in routes (presentation layer)
   - Session layer can import from recording without circular dependency

3. **Types Mirror Go Contracts**
   - `types/contracts.ts` must stay in sync with Go `automation/contracts/*.go`
   - Wire format types live in contracts, internal types elsewhere

4. **Telemetry as Infrastructure**
   - Screenshot/DOM capture are integration concerns
   - Collectors live in `telemetry/` separate from route handling

5. **Outcome Building in Domain Layer** (NEW)
   - `domain/outcome-builder.ts` owns StepOutcome and DriverOutcome construction
   - Routes call `buildStepOutcome()` and `toDriverOutcome()` instead of inline building
   - Centralizes wire format transformation logic

6. **Metrics Server Extracted to Utils** (NEW)
   - `utils/metrics-server.ts` handles Prometheus metrics endpoint
   - Separates infrastructure concern from main server entry point

7. **Declarative Router** (NEW)
   - `routes/router.ts` provides lightweight routing with path parameter extraction
   - Routes registered declaratively with `router.get()` / `router.post()`
   - Automatic 404/405 handling based on route registration
   - Replaces large if/else chain in server.ts

### Remaining Opportunities

1. **Extract Route Registration**
   - Route registration could move to a dedicated `routes/register.ts` module
   - Would further slim down `server.ts` to pure startup orchestration

2. **BrowserFactory Interface**
   - `SessionManager` still has direct `chromium.launch()` dependency
   - Extract to injectable factory for unit testing session logic

3. **Handler Telemetry Abstraction**
   - Handlers could optionally receive a `TelemetryContext` for screenshot/DOM capture
   - Would allow handlers to capture telemetry without knowing infrastructure details

4. **RecordModeController Separation**
   - `recording/controller.ts` mixes domain logic (action normalization, confidence scoring) with integration (Playwright page interaction)
   - Replay preview execution (~300 lines) could be extracted to a dedicated `ReplayExecutor` or moved closer to handlers
   - Current design trades modularity for co-location of recording concerns

### Enforced Boundaries (Completed)

1. **Handler Type Safety**
   - Private assertion methods now use proper `Page` type instead of `any`
   - Type imports consolidated at top of file
   - Enables IDE autocomplete and compile-time checking

2. **Logging in Domain Layer**
   - `assertNotExists` now receives logger via parameter injection instead of using `console.log`
   - Debug output follows structured logging patterns consistent with rest of codebase
   - Production logs are clean; debug info available via log level configuration

3. **Outcome Building Consolidation**
   - Removed duplicate `BaseHandler.buildOutcome()` method
   - All outcome construction now flows through `domain/outcome-builder.ts`
   - Single source of truth for wire format transformation
