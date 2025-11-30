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

**Status:** Good
- Narrow interface (ExecutionRepository) reduces coupling
- DBRecorder uses interface, not full Repository
- Need compile-time enforcement: `var _ Recorder = (*DBRecorder)(nil)`

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
- Comprehensive interface
- Mock exists but is verbose (all methods return nil/error)
- Consider: Interface segregation into smaller role interfaces

---

### 6. Storage Seam (Good)

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

**Status:** Good
- Clean interface for blob storage
- No dedicated test double found
- **Action Needed:** Add `MemoryStorage` test double

---

### 7. WebSocket Hub Seam (Good)

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

### 8. WorkflowService Seam (Weak)

**Location:** `api/handlers/handler.go` and `api/services/workflow/service.go`

**Interface:** Defined in `handler.go`
```go
type WorkflowService interface {
    CreateWorkflowWithProject(ctx context.Context, ...) (*database.Workflow, error)
    ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error)
    // ... 20+ methods
}
```

**Status:** Weak
- Interface is comprehensive
- Handler constructs concrete `WorkflowService` internally
- Test mock exists (`workflowServiceMock`) but requires all methods
- **Action Needed:**
  - Accept WorkflowService in Handler constructor
  - Consider interface segregation

---

### 9. AI Client Seam (Missing)

**Location:** `api/services/ai/client.go`

**Current Implementation:**
```go
type OpenRouterClient struct {
    log   *logrus.Logger
    model string
}

func (c *OpenRouterClient) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
    // Shells out to resource-openrouter command
}
```

**Status:** Missing
- No interface defined
- Shells out directly to `resource-openrouter` executable
- `WorkflowService` uses concrete type: `aiClient *ai.OpenRouterClient`
- Impossible to unit test without real executable

**Action Needed:**
- Extract `AIClient` interface
- Create `MockAIClient` for testing
- Inject via `WorkflowServiceOptions`

---

### 10. HTTP Client Seam in PlaywrightEngine (Missing)

**Location:** `api/automation/engine/playwright_engine.go`

**Current Implementation:**
```go
type PlaywrightEngine struct {
    driverURL  string
    httpClient *http.Client  // Hardcoded
    log        *logrus.Logger
}
```

**Status:** Missing
- Uses `*http.Client` directly
- Cannot mock HTTP calls for unit testing
- Integration tests require running Playwright driver

**Action Needed:**
- Create `HTTPDoer` interface: `Do(req *http.Request) (*http.Response, error)`
- Inject via constructor or use `httptest.Server` pattern

---

## TypeScript Playwright-Driver Seams

### 11. InstructionHandler Seam (Strong)

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

### 12. SessionManager Seam (Weak)

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

---

## Seam Enforcement Matrix

| Seam | Interface | Test Double | Compile Check | Priority |
|------|-----------|-------------|---------------|----------|
| AutomationEngine | Yes | Yes | Yes | - |
| Executor | Yes | Yes | Yes | - |
| Recorder | Yes | Yes | **Missing** | High |
| EventSink | Yes | Yes | Yes | - |
| Repository | Yes | Yes | **Missing** | Medium |
| Storage | Yes | **Missing** | **Missing** | High |
| WebSocket Hub | Yes | **Partial** | **Missing** | Medium |
| WorkflowService | Yes | Yes | **Missing** | High |
| AI Client | **Missing** | **Missing** | N/A | Critical |
| HTTP Client (Engine) | **Missing** | **Missing** | N/A | High |
| SessionManager (TS) | **Missing** | **Missing** | N/A | Medium |

---

## Enforcement Actions

### Critical Priority

1. **AI Client Interface**
   - Create `AIClient` interface in `api/services/ai/interface.go`
   - Add `MockAIClient` in `api/services/ai/mock.go`
   - Update `WorkflowServiceOptions` to accept interface

2. **Storage Test Double**
   - Create `MemoryStorage` in `api/storage/memory.go`
   - Implement all `StorageInterface` methods

### High Priority

3. **Compile-Time Enforcement**
   ```go
   // Add to each implementation file:
   var _ Recorder = (*DBRecorder)(nil)
   var _ Repository = (*repository)(nil)
   var _ StorageInterface = (*Storage)(nil)
   var _ WorkflowService = (*workflow.WorkflowService)(nil)
   ```

4. **HTTP Client Abstraction**
   - Create `HTTPDoer` interface
   - Update `PlaywrightEngine` constructor

5. **Handler WorkflowService Injection**
   - Modify `NewHandler` to accept optional `WorkflowService`
   - Fall back to default construction if not provided

### Medium Priority

6. **Interface Segregation for Repository**
   - Split into `ProjectRepository`, `WorkflowRepository`, `ExecutionRepository`
   - Consumers depend only on what they need

7. **WebSocket Mock Hub**
   - Create `MockHub` implementing `HubInterface`
   - Use in handler unit tests

8. **TypeScript BrowserFactory**
   - Extract browser launch into injectable factory
   - Enable session manager unit testing

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

**Recommended restructuring:**
1. Create `services/element_analysis/` for domain logic:
   - Move `DOMExtractor` logic
   - Move `ElementAnalysisHandler` logic
   - Move domain types (ElementInfo, SelectorOption, etc.)
2. Keep HTTP handlers in `handlers/ai/` that delegate to the service
3. This enables testing domain logic independently of HTTP layer

**Why this wasn't done now:**
- Per boundary-enforcement guidelines: avoid "broad, risky change"
- The current handlers/ai package is well-tested with proper seams
- Functionality works correctly; this is a maintainability improvement
- Should be tackled in a dedicated restructuring phase
