Warning: truncated output (original token count: 21819)
Total output lines: 1926

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

## Test Infrastructure Boundaries

### 1. Go API Test Utilities (New / Enforced)

**Location:** `api/internal/testutil`

**Contract:**
- Shared API test utilities live under `internal/testutil`.
- Production Go files must not import `github.com/vrooli/browser-automation-studio/internal/testutil`.
- New Go files must not import the legacy top-level `github.com/vrooli/browser-automation-studio/testutil` package.
- The boundary is enforced by `api/internal/testutil/no_prod_import_test.go`.

**Status:** Good
- Package documentation now defines the intended subpackages: `fixtures`, `mocks`, `db`, `httpx`, and `assertx`.
- `api/internal/testutil/fixtures` now contains canonical recording/session fixtures for `domain.RecordingSession`, recording timeline entries, recording actions, page events, and session profiles. Service-level recording tests should use these builders instead of repeating full domain structs. Tests inside `services/recording/persistence` remain package-local because importing fixtures there would create a Go import cycle through the persistence package.
- `api/internal/testutil/mocks` now contains canonical fakes for recurring seams: `shared.DirectoryScanner`, `shared.WorkflowIndexer`, `shared.ProjectIndexer`, plus function-backed `workflow.CatalogService` and `workflow.ExecutionService` fakes for tool-execution and future handler-style tests that need only a narrow service slice.
- `api/internal/testutil/integration` contains shared optional-integration gates for short mode, required environment variables, required local commands, and health-checkable HTTP services. Playwright, Ollama, MinIO, and FFmpeg tests should use these helpers instead of hand-rolled `t.Skip` strings.
- The old `api/testutil` package still exists and should be retired incrementally; a boundary test now prevents new imports from depending on it while useful recording/session factory behavior moves into `internal/testutil/fixtures`.

### 2. Driver Test Helpers (Enforced)

**Location:** `playwright-driver/tests/helpers`

**Contract:**
- Driver production code under `playwright-driver/src` must not import helper modules from `tests/helpers`.
- The boundary is enforced by `playwright-driver/tests/unit/boundaries/no-prod-testutil-imports.test.ts`.

**Status:** Good
- The import boundary is now covered by the driver unit suite.
- `tests/helpers/README.md` and `playwright-driver/docs/internal/SEAMS.md` document the current helper responsibilities: Playwright object fakes, HTTP mocks, fetch/API response helpers, typed instruction builders, config fixtures, and the compatibility barrel.
- `tests/helpers/fetch-mocks.ts` now centralizes global fetch installation, JSON/text response builders, and request inspection for vision-client and record-mode callback route tests. New unit tests should use it instead of assigning `global.fetch` or constructing ad hoc `Response` objects.
- Record-mode route tests now use the shared HTTP route harness to cover validation and lifecycle contracts without launching a browser. `recording-validation.test.ts` locks selector and replay-preview response mapping; `recording-lifecycle.test.ts` locks status and stop idempotency/cleanup behavior.
- Session, recording, and telemetry builders should be added only when those seams recur across tests.

### 3. UI Test Utilities (Enforced)

**Location:** `ui/src/test-utils`

**Contract:**
- UI production modules under `ui/src` must not import from `src/test-utils`.
- The boundary is enforced by `ui/vitest/boundaries/test-utils-imports.test.ts`.
- `ui/scripts/run-vitest.sh` now has an explicit smoke/full split: default `pnpm test` runs stable smoke projects, and `pnpm test:full` runs all configured Vitest projects.

**Status:** Good
- The boundary check runs in the default smoke suite through the `boundaries` Vitest project.
- `src/test-utils` now has focused entry points for `render`, `hooks`, `mocks`, `fixtures`, and `stores`, with compatibility exports for older `testHelpers` and `renderHook` imports.
- Fetch-based API adapter, store, component, and recording-state tests should use `installFetchMock` plus `fetchJsonResponse`, `fetchTextResponse`, or `fetchEmptyResponse` from `@/test-utils` instead of hand-rolled `global.fetch` response objects.
- The shared fetch seam now covers representative API adapter tests; the `projectStore`, `scenarioStore`, `workflowStore`, and `entitlementStore` suites; `ProjectDetail`; and recording viewport sync tests. UI tests should not assign `global.fetch` or `window.fetch` directly.
- Workflow builder tests should use the canonical ReactFlow, Monaco, drag-event, workflow-node, workflow-edge, viewport, validation-response, and workflow-store fixtures from `@/test-utils` instead of redefining canvas/editor shims locally.
- Assertion helpers still need to be added as recurring domain expectations emerge.

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

### 6. Database Storage Seam (Good)

**Location:** `api/database/connection.go`, `api/database/repository_test.go`

**Controls:**
- `BAS_SQLITE_PATH` overrides the resolved SQLite file path
- `DATABASE_URL=file:/abs/path.db` is honored as a fallback override

**Status:** Good
- `NewConnection` opens a single SQLite file (driver: `modernc.org/sqlite`, pure Go) and applies the canonical schema from `api/internal/<domain>/storage/sqlite/schemas/` idempotently
- The SQLite file path is resolved through `api-core/storage` (`ProfileAuto`), keeping mutable runtime data outside the deploy tree on every OS
- `setupTestDB` in `repository_test.go` opens a temp SQLite file and runs the same schema bootstrap — production and test paths share schema

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

### 9. WorkflowService Seam (Good)

**Location:** `api/handlers/handler.go` and `api/services/workflow/service.go`

**Interface:** `api/services/workflow/interfaces.go`

```go
// CatalogService manages workflow/project CRUD, versioning, and file synchronization
type CatalogService interface {
    // Health checks
    CheckHealth() string
    CheckAutomationHealth(ctx context.Context) (bool, error)

    // Project management (8 methods)
    CreateProject(ctx context.Context, project *database.ProjectIndex, description string) error
    GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error)
    // ... more project methods

    // Workflow CRUD (5 methods)
    CreateWorkflow(ctx context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error)
    // ... more workflow methods

    // Versioning (3 methods)
    // File sync (1 method)
    // AI modification (1 method)
}

// ExecutionService manages workflow execution lifecycle
type ExecutionService interface {
    // Execution control
    ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error)
    ExecuteWorkflowAPI(ctx context.Context, req *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error)
    ExecuteAdhocWorkflowAPI(ctx context.Context, req *basexecution.ExecuteAdhocRequest) (*basexecution.ExecuteAdhocResponse, error)
    StopExecution(ctx context.Context, executionID uuid.UUID) error
    ResumeExecution(ctx context.Context, executionID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error)

    // Execution queries (5 methods)
    // Timeline and export (4 methods)
}

// WorkflowResolver - minimal interface for execution-time workflow lookup
type WorkflowResolver interface {
    GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error)
    GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*basapi.WorkflowSummary, error)
    GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string) (*basapi.WorkflowSummary, error)
}
```

**Test Doubles:**
- Handler tests use interface types directly, enabling mock injection

**Status:** Good
- Clear separation between catalog (CRUD/versioning) and execution (lifecycle/timeline) responsibilities
- Handler uses interface types `CatalogService` and `ExecutionService` instead of concrete `*WorkflowService`
- Compile-time enforcement via `var _ CatalogService = (*WorkflowService)(nil)`
- `NewHandlerWithDeps` accepts injected services via `HandlerDeps` struct
- `WorkflowResolver` provides minimal interface for execution-time workflow lookup
- Remaining opportunity: Further interface segregation as package split progresses

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

### 10.5 Workflow ingress and execution manifest seams (Strong)

**Locations:** `api/services/workflow/v2_flow_builder.go`,
`api/services/workflow/converter.go`, and
`api/automation/execution-writer/result_manifest.go`

**Contracts:**

- `BuildFlowDefinitionV2ForWrite` is the only map-to-workflow path for newly
  authored project, recording, and AI writes. It accepts strict V2 protojson
  only and validates every node and edge before persistence.
- `ConvertExternalWorkflow` is the explicitly named migration boundary. It is
  the only workflow service path that may use the legacy V1-to-V2 adapter when
  ingesting a supported external workflow.
- `buildResultManifestPayload` owns the durable `result.json` projection. The
  `FileWriter` retains synchronization and filesystem placement, while the
  manifest contract remains a pure, independently tested concern.

### Execution writer ownership modules

`api/automation/execution-writer/` now separates the durable execution record
without changing its public `ExecutionWriter` interface:

- `artifact_config.go`: writer-wide and per-execution collection profiles.
- `result_manifest.go`: pure `result.json` projection.
- `timeline.go`: timeline creation and durable protobuf-JSON projection.
- `telemetry.go`: optional telemetry collection and timeline-log projection.
- `external_artifacts.go`: video/trace/HAR/custom file policy and storage.

`FileWriter` remains the composition root for synchronization, step outcomes,
checkpointing, and storage wiring. Domain modules do not import test helpers.

### Recording handler ownership modules

`api/handlers/record_mode_*.go` separates public routes by recording concern:

- `lifecycle`: live recording start/stop/status;
- `navigation`: navigation commands and read-side state/history;
- `frames`: HTTP frames and deterministic driver packet decoding;
- `actions`: action ingest, page attribution, timeline persistence, and typed
  WebSocket projection;
- `persistence`: session-profile resolution and durable browser state;
- `validation`: selector validation and replay preview.

The remaining `record_mode.go` is the composition surface for browser-session
creation/closure, debug proxying, generated-workflow ingress, and a small set
of transport endpoints that have not yet formed an independent domain.

**Enforcement:**

- `v2_flow_builder_test.go` proves a V1 `type`/`data` node is rejected for a
  normal write and a typed V2 action is accepted.
- `result_manifest_test.go` protects the stable fallback manifest when a
  timeline has not yet been produced.
- `cli/import_boundary_test.go`, `ui/vitest/boundaries/test-utils-imports.test.ts`,
  `playwright-driver/tests/unit/boundaries/no-prod-testutil-imports.test.ts`,
  and `api/internal/testutil/no_prod_import_test.go` prevent each delivery
  surface from depending on server implementation or test-only code.

---

### 11. HTTP Client Seam in PlaywrightEngine (Good)

**Location:** `api/automation/engine/playwright_engine.go`

**Status:** Good
- `HTTPDoer` interface defined in `automation/engine/http.go` with compile check on `http.Client`
- `NewPlaywrightEngineWithHTTPClient` accepts injected client for tests (can use `httptest.Server`)
- Default constructor still wires a long-timeout client for real driver runs

---

### 24. Credit Service Seam (Strong)

**Location:** `api/services/credits/`

**Interface:** `api/services/credits/interface.go`
```go
type CreditService interface {
    CanCharge(ctx context.Context, userIdentity string, op OperationType) (bool, int, error)
    Charge(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
    ChargeIfAllowed(ctx context.Context, req ChargeRequest) (*ChargeResult, error)
    GetUsage(ctx context.Context, userIdentity string) (*UsageSummary, error)
    GetOperationCost(op OperationType) int
    LogFailedOperation(ctx context.Context, req ChargeRequest, opErr error) error
    GetUsageHistory(ctx context.Context, userIdentity string, months, offset int) ([]UsageSummary, bool, error)
    GetOperationLog(ctx context.Context, userIdentity, month, category string, limit, offset int) (*OperationLogPage, error)
    CanPerformAIOperation(ctx context.Context, userIdentity string, op OperationType, hasBYOK bool) (bool, string, string, int, error)
}
```

**Test Doubles:**
- `services/credits/mock.go`: `MockService` - Full interface mock with configurable responses
- `services/credits/entitlement_provider.go`: `MockEntitlementProvider` - Controls tier/limit behavior

**Status:** Strong
- Clean interface separating credit checking from charging
- Multiple testing seams: `EntitlementProvider`, `LPBSReporter`, `Dialect` flag
- Comprehensive test coverage with SQLite in-memory database
- Compile-time enforcement via `var _ CreditService = (*Service)(nil)`

**Architecture:**

```
┌──────────────────────────────────────────────────────────────────┐
│                      CreditService                               │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐    ┌─────────────────────────────────┐   │
│  │ EntitlementProvider │───▶ GetAICreditsLimit()             │   │
│  │ (testing seam)      │    CanUseAIWithEntitlement()        │   │
│  └──────────────────┘    GetEntitlement()                    │   │
│                          └─────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────┐    ┌─────────────────────────────────┐   │
│  │ LPBSReporter     │───▶ ReportUsage()                      │   │
│  │ (testing seam)   │    (async to central LPBS)             │   │
│  └──────────────────┘    └─────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────┐    ┌─────────────────────────────────┐   │
│  │ Database Dialect │───▶ PostgreSQL (production)            │   │
│  │ (testing seam)   │    SQLite (unit tests)                 │   │
│  └──────────────────┘    └─────────────────────────────────┘   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**Testing Example:**
```go
// Unit test setup with all seams mocked
svc := credits.NewService(credits.ServiceOptions{
    DB:      sqliteInMemoryDB,
    Logger:  logrus.New(),
    Dialect: "sqlite",
    EntitlementProvider: &credits.MockEntitlementProvider{
        Entitlement: &entitlement.Entitlement{Tier: entitlement.TierPro},
        AICreditsLimit: 500,
        CanUseAI: true,
    },
    LPBSReporter: &mockLPBSReporter{}, // Captures reports for verification
})
```

---

### 25. Entitlement Service Seam (Strong)

**Location:** `api/services/entitlement/`

**Key Types:** `api/services/entitlement/types.go`
```go
type Entitlement struct {
    UserIdentity      string    `json:"user_identity"`
    Status            Status    `json:"status"`
    Tier              Tier      `json:"tier"`
    Features          []string  `json:"features,omitempty"`
    BillingCycleStart int       `json:"billing_cycle_start,omitempty"`
    // ...
}

type Tier string // TierFree, TierSolo, TierPro, TierStudio, TierBusiness
```

**Interface:** `api/services/credits/entitlement_provider.go`
```go
type EntitlementProvider interface {
    GetEntitlement(ctx context.Context, userIdentity string) (*entitlement.Entitlement, error)
    GetAICreditsLimit(tier entitlement.Tier) int
    CanUseAIWithEntitlement(ent *entitlement.Entitlement) bool
}
```

**Test Doubles:**
- `services/credits/entitlement_provider.go`: `MockEntitlementProvider`
- `services/entitlement/context.go`: Context injection for tier overrides

**Status:** Strong
- `EntitlementProvider` interface enables testing credit logic without real entitlement service
- Context-based override allows middleware tier injection for testing
- Cache with configurable TTL (5 min default)
- Offline grace period (5 hours) for resilience

**Testing Patterns:**

1. **Mock Provider (preferred for unit tests):**
```go
mock := &credits.MockEntitlementProvider{
    Entitlement: &entitlement.Entitlement{
        Tier: entitlement.TierPro,
        Status: entitlement.StatusActive,
    },
    AICreditsLimit: 500,
    CanUseAI: true,
}
```

2. **Context Override (for integration/handler tests):**
```go
ctx := entitlement.WithEntitlement(ctx, &entitlement.Entitlement{
    Tier: entitlement.TierBusiness,
})
// Credits service will use this entitlement instead of fetching
```

---

### 26. LPBS Reporter Seam (Strong)

**Location:** `api/services/credits/service.go`

**Interface:**
```go
type LPBSReporter interface {
    ReportUsage(ctx context.Context, report LPBSUsageReport) error
}

type LPBSUsageReport struct {
    UserIdentity string                  `json:"user_identity"`
    LimitKey     string                  `json:"limit_key"`
    UsageAmount  int64                   `json:"usage_amount"`
    AppBundleKey string                  `json:"app_bundle_key"`
    Metadata     LPBSUsageReportMetadata `json:"metadata,omitempty"`
}
```

**Status:** Strong
- Enables capturing and verifying usage reports in tests
- Async reporting (goroutine with retry) doesn't block local operations
- Reports include operation type, model, tokens, and BYOK flag

**Test Example:**
```go
type mockLPBSReporter struct {
    reports []credits.LPBSUsageReport
    mu      sync.Mutex
}

func (m *mockLPBSReporter) ReportUsage(ctx context.Context, report credits.LPBSUsageReport) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.reports = append(m.reports, report)
    return nil
}

// In test
reporter := &mockLPBSReporter{}
svc := credits.NewService(credits.ServiceOptions{
    LPBSReporter: reporter,
    // ...
})
// After operations, verify: reporter.reports
```

---

### 27. Recording Bounded Context Seam (Strong)

**Location:** `api/recording/`

**Purpose:** Persistent action recording with dual-write strategy (hot cache + database).

**DOC Reference:** [DOC: docs/architecture/recording.md#action-persistence]

**Interfaces:**

```go
// api/recording/persistence/repository.go
type ActionRepository interface {
    // Session lifecycle
    CreateSession(ctx context.Context, session *domain.RecordingSession) error
    GetSession(ctx context.Context, sessionID string) (*domain.RecordingSession, error)
    CloseSession(ctx context.Context, sessionID string, closedAt time.Time) error
    ListSessions(ctx context.Context, profileID *string, limit, offset int) ([]*domain.RecordingSession, error)
    DeleteSession(ctx context.Context, sessionID string) error

    // Action persistence
    SaveAction(ctx context.Context, action *domain.RecordingAction) error
    SaveActions(ctx context.Context, actions []*domain.RecordingAction) error
    GetAction(ctx context.Context, actionID uuid.UUID) (*domain.RecordingAction, error)

    // Queries
    ListActions(ctx context.Context, query ActionQuery) ([]*domain.RecordingAction, error)
    CountActions(ctx context.Context, sessionID string) (int, error)

    // Cleanup
    DeleteSessionActions(ctx context.Context, sessionID string) error
    PruneOldSessions(ctx context.Context, olderThan time.Time) (int, error)
}
```

**Test Doubles:**
- `recording/persistence/mock_repository.go`: `MockRepository` - Full in-memory implementation
- Test helpers: `Reset()`, `SessionCount()`, `ActionCount()`

**Status:** Strong
- Clean interface separation (ActionRepository)
- In-memory mock enables hermetic testing
- Compile-time enforcement via `var _ ActionRepository = (*MockRepository)(nil)`
- Driver-agnostic normalization in capture layer
- Hot cache + DB dual-write for performance and durability

**Architecture:**

```
┌──────────────────────────────────────────────────────────────────┐
│                    Recording Bounded Context                      │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐    ┌─────────────────────────────────┐   │
│  │ Capture Layer    │───▶ normalizer.go (driver → domain)   │   │
│  │ (testing seam)   │    deduplicator.go (500ms window)     │   │
│  └──────────────────┘    └─────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────┐    ┌─────────────────────────────────┐   │
│  │ Service Layer    │───▶ service.go (orchestrator)         │   │
│  │ (testing seam)   │    Hot cache + WebSocket broadcast    │   │
│  └──────────────────┘    └─────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────┐    ┌─────────────────────────────────┐   │
│  │ Persistence Layer│───▶ SQLiteRepository (production)     │   │
│  │ (testing seam)   │    MockRepository (unit tests)        │   │
│  └──────────────────┘    └─────────────────────────────────┘   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**Integration Seams:**
- **ActionRepository**: Substitutable persistence layer for testing
  - Location: `api/recording/persistence/repository.go`
  - Testability: `MockRepository` provided in same package

- **Normalizer**: Driver-agnostic action conversion
  - Location: `api/recording/capture/normalizer.go`
  - Testability: Pure function, no external dependencies

- **Deduplicator**: Navigate action deduplication
  - Location: `api/recording/capture/deduplicator.go`
  - Testability: In-memory state, `ClearSession()` for test isolation

**Responsibility Zones:**
- **Capture Layer** (`api/recording/capture/`):
  - Receives driver callbacks, normalizes, deduplicates
- **Service Layer** (`api/recording/service.go`):
  - Orchestrates persistence + WebSocket broadcast
- **Persistence Layer** (`api/recording/persistence/`):
  - Database access, query execution

**Change Axes:**
- Adding new driver: Only `normalizer.go` changes
- Adding new action type: Domain + normalizer
- Changing storage backend: Only repository implementation
- Changing deduplication rules: Only `deduplicator.go`

**Testing Example:**
```go
// Unit test setup with mock repository
repo := persistence.NewMockRepository()
svc := recording.NewService(repo, logger, recording.DefaultServiceConfig())

// Create session and record action
session, _ := svc.CreateSession(ctx, recording.SessionConfig{ProfileID: "test"})
action := &domain.RecordingAction{
    ID:         uuid.New(),
    SessionID:  session.ID,
    ActionType: "click",
    Timestamp:  time.Now(),
}
svc.RecordDomainAction(ctx, action)

// Verify persistence
assert.Equal(t, 1, repo.ActionCount())
```

---

### 28. ActionRecorder Seam (Strong)

**Location:** `api/services/recording/recorder.go`

**Purpose:** Unified recording pipeline that combines persistence and WebSocket broadcast with full observability.

**DOC Reference:** [Plan: Recording Pipeline Domain Boundaries, Testing Seams & Observability]

**Problem Solved:**
The recording pipeline previously had a **dual-write anti-pattern** where persistence and WebSocket broadcast happened independently:

```go
// BEFORE (dual-write, silent failures)
h.recordModeService.AddTimelineAction(sessionID, &action, pageIDForTimeline)  // Write 1
h.wsHub.BroadcastRecordingEntry(sessionID, h.createUnifiedEntry(&action))     // Write 2
// Either could fail silently with no visibility
```

**Interfaces:**

```go
// api/services/recording/recorder.go
type ActionRecorder interface {
    RecordActionUnified(ctx context.Context, req RecordActionRequest) (*ActionRecordResult, error)
    RecordPageEventUnified(ctx context.Context, req RecordPageEventRequest) (*ActionRecordResult, error)
}

type RecordActionRequest struct {
    SessionID     string
    Action        *driver.RecordedAction
    PageID        uuid.UUID
    Source        ActionSource      // ActionSourceManual, ActionSourceAuto, ActionSourceAI
    CorrelationID string            // For tracing through pipeline
}

type ActionRecordResult struct {
    ActionID        uuid.UUID
    CorrelationID   string
    SequenceNum     int
    Persisted       bool              // Did persistence succeed?
    BroadcastSent   bool              // Did broadcast reach any clients?
    SubscriberCount int               // How many clients were subscribed?
    SentCount       int               // How many clients received the message?
    DroppedCount    int               // How many clients had full buffers?
    Errors          []ActionRecordError
}

func (r *ActionRecordResult) HasErrors() bool
```

**WebSocket Observability:**

```go
// api/websocket/hub.go
type BroadcastResult struct {
    SubscriberCount int   // Number of subscribed clients
    SentCount       int   // Successfully sent
    DroppedCount    int   // Dropped due to full buffers
}

func (h *Hub) BroadcastRecordingEntry(sessionID string, entry *UnifiedTimelineEntry) BroadcastResult
```

**Test Doubles:**
- `handlers/record_mode_integration_test.go`: `TestRecordingHub` - Full HubInterface mock with broadcast tracking
- `handlers/testutil_mock_services.go`: `MockHub` - General-purpose mock
- `services/recording/persistence/mock_repository.go`: `MockRepository` - In-memory persistence

**Status:** Strong
- Unified interface eliminates dual-write anti-pattern
- Full observability: correlation IDs, persistence status, broadcast metrics
- Compile-time enforcement via `var _ ActionRecorder = (*Service)(nil)`
- Comprehensive integration tests

**Architecture:**

```
┌──────────────────────────────────────────────────────────────────┐
│                    ActionRecorder Pipeline                        │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐                                           │
│  │ HTTP Entry Point │───▶ generateCorrelationID()               │
│  │ (record_mode.go) │    Format: rec-{session[:8]}-{timestamp}  │
│  └──────────────────┘                                           │
│           │                                                      │
│           ▼                                                      │
│  ┌──────────────────┐    ┌─────────────────────────────────┐   │
│  │ RecordActionUnified │───▶ 1. Validate request              │   │
│  │ (Service method)    │    2. Normalize action               │   │
│  └──────────────────┘    3. Check deduplication              │   │
│           │              4. Persist to hot cache + DB        │   │
│           │              5. Broadcast to WebSocket           │   │
│           ▼              6. Return unified result            │   │
│  ┌──────────────────┐    └─────────────────────────────────┘   │
│  │ ActionRecordResult │                                         │
│  │ - Persisted: bool  │                                         │
│  │ - BroadcastSent    │                                         │
│  │ - SubscriberCount  │                                         │
│  │ - Errors[]         │                                         │
│  └──────────────────┘                                           │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**Observability Benefits:**

| Pipeline Step | Old Visibility | New Visibility |
|---------------|----------------|----------------|
| HTTP entry | Debug log | Debug log + correlation ID |
| Validation | None | Error in result.Errors |
| DB persistence | Warn on error | result.Persisted + error details |
| WebSocket broadcast | **None** | result.BroadcastSent, SubscriberCount |
| Client delivery | **None** | result.SentCount, DroppedCount |

**Testing Example:**

```go
// Integration test with full pipeline visibility
func TestRecordingPipeline_EndToEnd(t *testing.T) {
    repo := persistence.NewMockRepository()
    hub := NewTestRecordingHub(logger)
    recordingSvc := recording.NewService(repo, hub, logger, recording.ServiceConfig{})

    // Subscribe test client
    clientCh := hub.Subscribe(sessionID)
    defer hub.Unsubscribe(sessionID)

    // Record action
    result, err := recordingSvc.RecordActionUnified(ctx, recording.RecordActionRequest{
        SessionID:     sessionID,
        Action:        action,
        PageID:        pageID,
        Source:        recording.ActionSourceManual,
        CorrelationID: "test-corr-123",
    })

    // Verify full observability
    assert.True(t, result.Persisted)
    assert.True(t, result.BroadcastSent)
    assert.Equal(t, 1, result.SubscriberCount)
    assert.Equal(t, 1, result.SentCount)
    assert.False(t, result.HasErrors())

    // Verify WebSocket delivery
    select {
    case entry := <-clientCh:
        assert.Equal(t, "click", entry.Action.ActionType)
    case <-time.After(time.Second):
        t.Fatal("Action did not appear in WebSocket")
    }
}
```

**Integration with Existing Seams:**

| Seam | Integration |
|------|-------------|
| Recording Bounded Context (#27) | ActionRecorder uses ActionRepository for persistence |
| WebSocket Hub (#8) | BroadcastRecordingEntry returns BroadcastResult |
| EventBroadcaster (#22) | ActionRecorder unifies the broadcast path |

---

### 29. Execution Artifact Retention Seam (Strong)

**Location:** `api/services/retention/retention.go`

**Purpose:** Select terminal executions for artifact cleanup and, in apply mode, delete their artifact directories and DB index rows together. Owns the retention business logic so the Connect handler (`handlers/executions/retention.go`) and CLI stay thin.

**Interfaces:**

```go
// api/services/retention/retention.go
type FileSystem interface {
    DirSize(dir string) (sizeBytes int64, exists bool, err error) // missing dir -> (0,false,nil)
    RemoveAll(dir string) error
}

type ExecutionStore interface {
    ListExecutions(ctx, workflowID, projectID *uuid.UUID, limit,…1819 tokens truncated…ctions, no side effects
- Comprehensive unit tests
- No external dependencies
- Mirrors backend logic in `api/handlers/record_mode.go`

**Design Rationale:**
- Client-side merging enables instant preview as users record (WYSIWYG)
- Single-pass greedy algorithm for O(n) complexity
- Tracks original action IDs in `_merged` metadata for undo capability

---

### 20. Recording Reconciliation - AI Correlation Seam (Strong)

**Location:** `ui/src/domains/recording/types/timeline-unified.ts`

**Interface:**
```typescript
interface AIReconciliationService {
    mergeActionsWithAISteps(actions: RecordedAction[], aiSteps: AIStep[]): TimelineItem[];
    recordedActionToTimelineItem(action: RecordedAction, aiMetadata?: AIMetadata): TimelineItem;
    workflowNodesToTimelineItems(nodes: WorkflowNode[], edges: Edge[]): TimelineItem[];
    updateTimelineItemStatus(items: TimelineItem[], nodeId: string, status: ExecutionStatus): TimelineItem[];
}
```

**Test File:** `ui/src/domains/recording/types/timeline-unified.test.ts`

**Status:** Strong
- Pure functions for timeline transformation
- Timestamp-based matching with 5s proximity window
- Action type normalization (e.g., "type" → "input")
- Consumes matched AI steps to prevent duplicate attribution

**Design Rationale:**
- Enables users to see both what happened AND why the AI did it
- Immutable transformations (returns new objects, never mutates)
- Supports both recording mode (live) and execution mode (workflow playback)

---

### 21. Recording Reconciliation - Workflow Sync Seam (Strong)

**Location:** `api/services/workflow/sync.go`, `api/services/workflow/sync_interfaces.go`

**Interface:**
```go
type WorkflowSyncRepository interface {
    GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error)
    ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.WorkflowIndex, error)
    CreateWorkflow(ctx context.Context, workflow *database.WorkflowIndex) error
    UpdateWorkflow(ctx context.Context, workflow *database.WorkflowIndex) error
    DeleteWorkflow(ctx context.Context, id uuid.UUID) error
    ListAssetsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.AssetIndex, error)
    CreateAsset(ctx context.Context, asset *database.AssetIndex) error
    UpdateAsset(ctx context.Context, asset *database.AssetIndex) error
    DeleteAsset(ctx context.Context, id uuid.UUID) error
}
```

**Test Doubles:**
- `api/services/workflow/sync_interfaces.go`: `MockWorkflowSyncRepository`

**Test File:** `api/services/workflow/sync_test.go`

**Status:** Strong
- Interface enables testing sync logic without real database
- DBState/SeenState intermediate types for O(1) lookups
- Decomposed into small functions (loadDBState, scanAndReconcile, garbageCollect)
- Per-project locking prevents concurrent sync corruption

**Design Rationale:**
- Filesystem is source of truth (enables git version control)
- File limits (1000 files, depth 4) prevent runaway operations
- In-place format conversion normalizes external workflows

---

### 22. EventBroadcaster Seam (Strong)

**Location:** `api/websocket/interface.go`

**Interface:**
```go
type HubInterface interface {
    // Recording-related methods (EventBroadcaster subset)
    BroadcastRecordingAction(sessionID string, action any)
    BroadcastRecordingActionWithTimeline(sessionID string, action any, timelineEntry map[string]any)
    BroadcastRecordingFrame(sessionID string, frame *RecordingFrame)
    BroadcastPageEvent(sessionID string, event any)
    HasRecordingSubscribers(sessionID string) bool
    // ... other methods
}
```

**Test Doubles:**
- `api/handlers/testutil_mock_services.go`: `MockHub`

**Status:** Strong (Updated 2026-01-20)
- HubInterface serves as the EventBroadcaster for recording features
- MockHub already implements all required methods
- Enables isolated testing of recording handlers without real WebSockets

---

### 23. RetryService Seam (Strong)

**Location:** `ui/src/domains/recording/services/RetryService.ts`

**Interface:**
```typescript
interface RetryService {
    createInitialRetryState(): RetryState;
    calculateRetryDelay(attempt: number, config: RetryConfig): number;
    getNextRetryState(currentAttempts: number, config: RetryConfig): RetryState;
    canRetry(state: RetryState): boolean;
    getRemainingCooldown(state: RetryState): number;
    createSuccessState(): RetryState;
    createManualRetryState(): RetryState;
}
```

**Test File:** `ui/src/domains/recording/services/RetryService.test.ts`

**Status:** Strong
- Pure functions, no side effects
- Comprehensive unit tests (35 test cases)
- No external dependencies
- Used by `hooks/useRecordingSession.ts` for session creation retry logic

**Design Rationale:**
- Pure functions extracted from hook for testability
- Configurable via `RetryConfig` for different retry behaviors
- State includes UI-friendly fields (`nextRetryAt`, `inCooldown`) for countdown display
- Exponential backoff with cap prevents server hammering

---

### 30. Accessibility Snapshot Capture + Contract Seam (Strong)

**Locations:**
- Driver: `playwright-driver/src/tracing/accessibility-snapshot.ts` (`AccessibilitySnapshotter`, `normalizeAccessibilityTree`, `parseDomSnapshot`)
- Handler: `api/handlers/capture/producer.go` (accessibility `fileProducer`), `api/handlers/capture/inline_accessibility.go` (inline read + 2 MiB cap)
- Proto: `CaptureType.CAPTURE_TYPE_ACCESSIBILITY = 7`, `CaptureRequest.inline_accessibility = 9`, `CaptureResponse.accessibility_json = 7`, `ArtifactType.ARTIFACT_TYPE_ACCESSIBILITY_SNAPSHOT = 8`

**What it captures:** `CAPTURE_TYPE_ACCESSIBILITY` requests a normalized JSON snapshot of the Chromium accessibility tree. The driver walks the AX tree via CDP `Accessibility.getFullAXTree` at a settled point (session close, on the final page — after `wait_for` and any `interaction_flow_json`, the same point the final screenshot fires), joins per-node geometry + `data-testid` from one `DOMSnapshot.captureSnapshot`, and writes `accessibility.json`. It rides the same capability plumbing as the perf trace (`RequiresAccessibility` → plan metadata `requiresAccessibility` → preflight `NeedsAccessibility` → driver `required_capabilities.accessibility` + `artifact_paths.accessibility_dir`). `ExportToFolder` lands the file flat in the capture out dir; the accessibility `fileProducer` surfaces it (absent → unavailable artifact, never an error).

**Frozen contract — `bas-accessibility-snapshot/v1`** (field names are stable; another scenario builds against this):
```json
{
  "contract": "bas-accessibility-snapshot/v1",
  "url": "<final page url>",
  "viewport": {"width": 1440, "height": 900, "deviceScaleFactor": 1},
  "captured_at": "<RFC3339>",
  "node_count": 0,
  "truncated": false,
  "root": {
    "role": "...", "name": "...", "description": "...", "value": "...",
    "states": ["focusable"],
    "bounds": {"x": 0, "y": 0, "width": 0, "height": 0},
    "dom": {"testid": "...", "tag": "div"},
    "children": []
  },
  "meta": {"frames": "main-only", "source": "cdp-accessibility"}
}
```
Rules: ignored AX nodes are pruned (children spliced up); `bounds`/`dom`/empty-string scalar fields are omitted rather than nulled; main frame only in v1; node count capped (`truncated` flips true past the cap). `inline_accessibility` returns the same JSON inline in `CaptureResponse.accessibility_json` (server-capped at 2 MiB, silent truncation) and independently drives the capture (a caller need not also list the capture type), mirroring `inline_dom`.

**Test Doubles:**
- Driver: `normalizeAccessibilityTree`/`parseDomSnapshot` are pure (golden-file test `tests/unit/accessibility-snapshot.test.ts`); `AccessibilitySnapshotter` takes a `cdpFactory` seam so `capture()` runs against a fake CDP with no real browser.
- Handler: `InlineAccessibilityConfig.readInlineAccessibility` + accessibility `fileProducer` covered by `handlers/capture/inline_accessibility_test.go` / `producer_test.go` via the existing `fakeExecutor` export seam.

**Status:** Strong — pure normalizer, injectable CDP factory, golden coverage, graceful degradation on every failure path.

---

## Seam Enforcement Matrix

| Seam | Interface | Test Double | Compile Check | Priority |
|------|-----------|-------------|---------------|----------|
| AutomationEngine | Yes | Yes | Yes | - |
| Executor | Yes | Yes | Yes | - |
| Recorder | Yes | Yes | Yes | - |
| EventSink | Yes | Yes | Yes | - |
| Repository | Yes | Yes | Yes | Medium |
| Database Storage | `BAS_SQLITE_PATH` / `DATABASE_URL=file:…` overrides; `api-core/storage` resolves the canonical SQLite file | Temp-file SQLite via `setupTestDB` (and in-memory `sql.Open("sqlite", ":memory:")` for credits tests) | N/A | Medium |
| Storage | Yes | Yes (MemoryStorage) | Yes | - |
| WebSocket Hub | Yes | Yes (MockHub) | Yes | - |
| WorkflowService | Yes (CatalogService, ExecutionService) | Yes | Yes | - |
| AI Client | Yes | Yes | Yes | - |
| HTTP Client (Engine) | Yes | Injectable HTTPDoer | Yes | Medium |
| SessionManager (TS) | Yes (BrowserManager extracted) | **Partial** | N/A | Low |
| RecordingBuffer (TS) | Yes | N/A (pure functions) | N/A | - |
| AI Element Analyzer | Yes | Yes | Yes (interface injection) | Medium |
| Router (TS) | Yes | N/A (coordination) | N/A | - |
| OutcomeBuilder (TS) | Yes | N/A (pure functions) | N/A | - |
| MetricsServer (TS) | Yes | N/A (infrastructure) | N/A | - |
| ActionMergeService (TS) | Yes | N/A (pure functions) | N/A | - |
| AIReconciliationService (TS) | Yes | N/A (pure functions) | N/A | - |
| WorkflowSyncRepository | Yes | Yes (Mock) | Yes | - |
| EventBroadcaster | Yes (via HubInterface) | Yes (MockHub) | Yes | - |
| RetryService (TS) | Yes | N/A (pure functions) | N/A | - |
| Accessibility Snapshot (TS + Go) | Yes (`cdpFactory`, pure normalizer, `fileProducer`) | Yes (fake CDP + golden; `fakeExecutor`) | Yes | - |
| CreditService | Yes | Yes (MockService, MockEntitlementProvider) | Yes | - |
| EntitlementProvider | Yes | Yes (MockEntitlementProvider) | Yes | - |
| LPBSReporter | Yes | Yes (mockLPBSReporter in tests) | Yes | - |
| Recording (ActionRepository) | Yes | Yes (MockRepository) | Yes | - |
| ActionRecorder | Yes | Yes (TestRecordingHub, MockHub) | Yes | - |

---

## Enforcement Actions

### Critical Priority

*None currently marked critical.*

### High Priority

1. ~~**WebSocket Mock Hub**~~ (COMPLETED 2026-01-20)
   - ~~Create `MockHub` implementing `HubInterface`~~
   - MockHub exists in `handlers/testutil_mock_services.go`
   - Used for recording handler tests and WebSocket isolation

### Medium Priority

2. **Interface Segregation for Repository**
   - Split into `ProjectRepository`, `WorkflowRepository`, `ExecutionRepository`
   - Consumers depend only on what they need

3. **WorkflowService Surface Trim**
   - Consider narrowing Catalog/Execution/Export into smaller interfaces for handler deps and mocks

4. **TypeScript BrowserFactory** (PARTIALLY ADDRESSED - 2026-01-03)
   - BrowserManager extracted from SessionManager
   - Browser lifecycle (launch, close, verify) now in dedicated class
   - Remaining: Consider interface extraction for full testability

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
| 2026-01-31 | Claude | ActionRecorder Seam: Added seam #28 (ActionRecorder interface); unified recording pipeline with observability; BroadcastResult for WebSocket metrics; correlation IDs for tracing; comprehensive integration tests; updated enforcement matrix |
| 2026-01-30 | Claude | Recording Bounded Context: Added seam #27 (ActionRepository, Normalizer, Deduplicator); documented capture/service/persistence layers; added integration seams and responsibility zones; updated enforcement matrix |
| 2026-01-21 | Claude | Credits System Seams: Added seams #24-26 (CreditService, EntitlementProvider, LPBSReporter); created EntitlementProvider interface with MockEntitlementProvider test double; documented testing patterns for credit/entitlement logic; updated enforcement matrix |
| 2026-01-20 | Claude | Recording Reconciliation Completion: Added RetryService seam (#23) with 35 test cases; completed services/index.ts exports; refactored useRecordingSession.ts to use RetryService (removed duplicate retry logic); updated enforcement matrix |
| 2026-01-20 | Claude | Recording Reconciliation Seams: Added seams #19-22 (ActionMergeService, AIReconciliationService, WorkflowSyncRepository, EventBroadcaster); created comprehensive test suites; decomposed sync.go into smaller functions; extracted frontend services to services/ directory |
| 2025-12-17 | Claude | Export package consolidation: Deleted duplicate handlers/export/presets.go (identical to services/export/presets.go); documented remaining export package duplication for future cleanup |
| 2025-12-17 | Claude | Action type mapping clarification: Completed typeconv.ActionTypeToString with all action types; documented intentional separation from flow_utils.actionTypeToString (legacy compatibility) |
| 2025-12-17 | Claude | WorkflowService decomposition Phase 1: Created CatalogService, ExecutionService, WorkflowResolver interfaces; refactored Handler to use interface types instead of concrete *WorkflowService; updated HandlerDeps for clean dependency injection |
| 2025-12-17 | Claude | Architectural consolidation: Merged duplicate async execution runners; centralized eventBufferLimits(); consolidated ExecutionPlan types via compiler/contracts_adapter.go |
| 2025-12-16 | Claude | Architecture refactoring: Added internal/wire package, services/recordmode, services/recordingimport; documented new responsibility boundaries |
| 2026-01-03 | Claude | Screaming Architecture Audit: Fixed type safety (logger: any → winston.Logger) in gesture.ts, device.ts; added logger injection to RecordModeController; replaced console.log/warn/error with structured logging in controller.ts and action-executor.ts |
| 2025-12-09 | Claude | Boundary of Responsibility Enforcement pass #2: Removed unused BaseHandler.buildOutcome (consolidated in domain/outcome-builder.ts); replaced any types with Page in assertion handler; replaced console.log with injected logger in assertNotExists |
| 2025-12-09 | Claude | Boundary of Responsibility Enforcement: Added Router (#16), OutcomeBuilder (#17), MetricsServer (#18) seams; extracted domain/outcome-builder.ts, utils/metrics-server.ts, routes/router.ts; updated responsibility boundaries |
| 2025-12-09 | Claude | Added RecordingBuffer seam (#14), Playwright-Driver Responsibility Boundaries section; moved action buffer state from routes to recording/buffer.ts |
| 2026-04-16 | Assistant | Removed Postgres backend; SQLite-only via `modernc.org/sqlite`. Database Backend seam collapsed to single-driver Database Storage seam |
| 2025-11-29 | Claude | Initial seam discovery and documentation |
| 2025-11-29 | Claude | Added Responsibility Boundaries section, apierror package |

---

## Responsibility Boundaries

### Current Architecture
The codebase follows a layered architecture with clear responsibilities:

| Layer | Location | Responsibility |
|-------|----------|----------------|
| **Entry/Presentation** | `handlers/` | HTTP routing, request/response mapping |
| **Coordination** | `services/workflow/`, `services/live-capture/` | Orchestration, business flow |
| **Domain Rules** | `automation/` | Execution contracts, engine abstraction, format conversion |
| **Infrastructure** | `database/`, `storage/`, `websocket/` | Persistence, storage, real-time |
| **Cross-cutting** | `internal/` | Shared utilities, error handling, dependency wiring |

### Dependency Wiring (NEW - 2025-12-16)

**Location:** `internal/wire/`

**Purpose:** Centralized dependency injection that separates infrastructure setup from business logic.

**Interface:**
```go
type Dependencies struct {
    WorkflowService   *workflow.WorkflowService
    RecordModeService *livecapture.Service       // services/live-capture
    RecordingService  archiveingestion.IngestionServiceInterface
    WorkflowValidator *workflowvalidator.Validator
    Storage           storage.StorageInterface
    RecordingsRoot    string
    ReplayRenderer    ReplayRenderer
    SessionProfiles   *archiveingestion.SessionProfileStore
    UXMetricsRepo     uxmetrics.Repository
}

func BuildDependencies(repo database.Repository, hub *wsHub.Hub, log *logrus.Logger, cfg Config) (*Dependencies, error)
```

**Benefits:**
- Single source of truth for production dependency construction
- Clear separation between infrastructure and business logic
- Easier testing through explicit dependency injection
- Handler construction simplified to receiving pre-wired dependencies

### Recording Services Clarification (UPDATED - 2025-12-17)

The codebase has distinct recording-related services with clear responsibilities:

| Package | Purpose | Key Operations |
|---------|---------|----------------|
| `services/archive-ingestion/` | Import Chrome extension archives, session profiles | `ImportArchive()`, `SessionProfileStore` |
| `services/live-capture/` | Live browser recording session management | `CreateSession()`, `StartRecording()`, `GenerateWorkflow()` |
| `services/replay/` | Video/screenshot rendering from execution data | `Render()` |

**`services/live-capture/`**:
- Manages live recording sessions via Playwright driver
- Encapsulates driver client HTTP communication
- Contains workflow generation business logic:
  - `WorkflowGenerator` - Converts recorded actions to workflow definitions
  - `MergeConsecutiveActions()` - Optimizes action sequences
  - `insertSmartWaits()` - Inserts wait nodes for reliability
- `handlers/record_mode.go` delegates to this service

**`services/archive-ingestion/`**:
- Handles ZIP archive ingestion from Chrome extension
- Creates projects, workflows, and execution artifacts from imported recordings
- `SessionProfileStore` - Manages persisted session profiles AND tracks active browser sessions

**Migration Status:**
- Session profile active session tracking moved from handlers to `SessionProfileStore` (2025-12-17)
- Deprecated type aliases removed from `handlers/record_mode.go` (2025-12-17)

### Compiler Package Consolidation (2025-12-17)

The `automation/workflow/` package has been **merged into `automation/compiler/`**:

**Rationale:**
- `automation/workflow/` contained V1↔V2 format conversion code
- This is fundamentally compilation: transforming workflow definitions to execution plans
- Having a separate `workflow/` folder in `automation/` was confusing (different from `services/workflow/`)

**Files Moved:**
- `v2_types.go` → `compiler/v1_types.go` - Legacy V1 type definitions
- `v2_utils.go` → `compiler/v1_utils.go` - Type conversion utilities
- `v2_param_builders.go` → `compiler/v1_param_builders.go` - Parameter builder wrappers
- `v2_convert.go` → `compiler/v1_convert.go` - V1↔V2 format conversion
- `v2_execution.go` → `compiler/v1_execution.go` - V2 workflow → execution plan

**Impact:**
- Imports changed from `automation/workflow` to `automation/compiler`
- All conversion functions now in a single, logical location
- `automation/workflow/` folder deleted

### Execution Flow Consolidation (2025-12-17)

The execution subsystem had significant technical debt with duplicate implementations. This has been addressed:

#### Problem 1: Duplicate Async Execution Runners

**Before:**
```go
// Two nearly identical functions (90% code duplication)
func (s *WorkflowService) executeWorkflowAsync(ctx, workflow, executionID, parameters)
func (s *WorkflowService) executeWorkflowAsyncWithNamespaces(ctx, workflow, executionID, store, params, env)
```

**After:**
- Single `executeWorkflowAsync(ctx, workflow, executionID, store, params, env)` implementation
- Legacy `startExecutionRunner()` normalizes flat parameters to namespaced model
- `startExecutionRunnerWithNamespaces()` calls the unified implementation

**Files Changed:** `services/workflow/executions.go`

#### Problem 2: Duplicate eventBufferLimits()

**Before:**
- `handlers/handler.go:80` - Identical function loading from config
- `services/workflow/service.go:174` - Same implementation copied

**After:**
- Single `config.EventBufferLimitsFromConfig()` in `config/config.go`
- Both call sites delegate to centralized function
- Returns `contracts.EventBufferLimits` directly

**Files Changed:** `config/config.go`, `handlers/handler.go`, `services/workflow/service.go`

#### Problem 3: Dual ExecutionPlan Types

**Before:**
- `compiler.ExecutionPlan` - Legacy type with `Steps []ExecutionStep`
- `contracts.ExecutionPlan` - Canonical type with `Instructions` + `Graph`
- `executor/contract_plan_compiler.go` - 60+ lines converting between types
- `executor/plan_graph_helpers.go` - Graph conversion helper

**After:**
- New `compiler.CompileWorkflowToContracts()` centralizes conversion
- `compiler/contracts_adapter.go` - Single location for compiler → contracts conversion
- `ContractPlanCompiler.Compile()` now delegates to centralized function (3 lines)
- Deleted redundant `executor/plan_graph_helpers.go`

**Files Changed:**
- `automation/compiler/contracts_adapter.go` (new)
- `automation/executor/contract_plan_compiler.go` (simplified)
- `automation/executor/plan_graph_helpers.go` (deleted)

#### Change Axis Improvements

These consolidations improve change axis resilience:

| Change Type | Before | After |
|-------------|--------|-------|
| New parameter namespace | Edit 2 functions | Edit 1 function |
| Event buffer config | Edit 2 files | Edit 1 file (config) |
| New compilation target | Add conversion code in executor | Extend compiler adapter |

### WorkflowService Decomposition (IN PROGRESS)

The `WorkflowService` handles multiple responsibilities. Decomposition is proceeding in phases:

**Phase 1: Interface Extraction (COMPLETED - 2025-12-17)**

Created clean service interfaces in `services/workflow/interfaces.go`:

| Interface | Responsibility | Method Count |
|-----------|----------------|--------------|
| `CatalogService` | Project/workflow CRUD, versioning, file sync, AI modification | ~20 methods |
| `ExecutionService` | Execution lifecycle, queries, timeline, export | ~12 methods |
| `WorkflowResolver` | Minimal workflow lookup for execution-time resolution | 3 methods |

**Handler Changes:**
```go
// Before: Triple alias to same concrete type
type Handler struct {
    workflowCatalog   *workflow.WorkflowService
    executionService  *workflow.WorkflowService
    exportService     *workflow.WorkflowService
}

// After: Clean interface dependencies
type Handler struct {
    catalogService   workflow.CatalogService   // CRUD, versioning, sync
    executionService workflow.ExecutionService // Execution lifecycle
}
```

**Benefits Achieved:**
- Handler no longer depends on concrete `*WorkflowService`
- Clear responsibility boundaries at interface level
- Enables future package split without handler changes
- Tests can inject mock implementations

**Phase 2: Package Split (PLANNED)**

| Package | Responsibility | Key Methods |
|---------|----------------|-------------|
| `services/workflow/catalog/` | Workflow CRUD, versioning, storage | `CreateWorkflow`, `UpdateWorkflow`, `ListWorkflows`, `GetVersionHistory` |
| `services/workflow/execution/` | Execution orchestration | `StartExecution`, `CancelExecution`, `GetExecutionStatus` |
| `services/workflow/ai/` | AI workflow generation | `GenerateWorkflowFromPrompt`, `RefineWorkflow` |
| `services/workflow/sync/` | File synchronization | `SyncToFilesystem`, `SyncFromFilesystem` |

**Change Axis Analysis:**
- Adding new node types → catalog package (validation) + execution (handlers)
- New AI models → ai package only
- New storage backends → catalog package only
- New execution modes → execution package only

**Status:** Phase 1 complete. Phase 2 planned - requires careful migration.

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
    CatalogService    workflow.CatalogService   // Workflow/project CRUD, versioning, sync
    ExecutionService  workflow.ExecutionService // Execution lifecycle, timeline, export
    WorkflowValidator *workflowvalidator.Validator
    Storage           storage.StorageInterface
    RecordingService  archiveingestion.IngestionServiceInterface
    RecordModeService *livecapture.Service
    RecordingsRoot    string
    ReplayRenderer    replayRenderer
    SessionProfiles   *archiveingestion.SessionProfileStore
    UXMetricsRepo     uxmetrics.Repository
}

// Production usage (wires all dependencies)
handler := handlers.NewHandler(repo, wsHub, log, allowAll, origins)

// Testing usage (inject mocks)
deps := handlers.HandlerDeps{
    CatalogService:   mockCatalogService,
    ExecutionService: mockExecutionService,
    Storage:          mockStorage,
    // ...
}
handler := handlers.NewHandlerWithDeps(repo, wsHub, log, allowAll, origins, deps)
```

### Export Package Boundary Enforcement (COMPLETED 2025-12-17)

The export functionality now has proper boundary enforcement per architectural principles:

| Package | Role |
|---------|------|
| `handlers/export/` | **HTTP layer only** - Request types, thin wrappers delegating to services |
| `services/export/` | **Business logic** - Spec building, preset application, validation, format strategies |

**Architecture Refactoring (Phase 2 - 2025-12-17):**

All business logic moved from `handlers/export/` to `services/export/`:

| File | Before (handlers) | After (services) |
|------|-------------------|------------------|
| `preset_builder.go` | ❌ Deleted | ✅ Owns `BuildThemeFromPreset`, `BuildCursorSpec` |
| `spec_harmonizer.go` | ❌ Deleted | ✅ Owns `BuildSpec`, `Clone`, `Harmonize`, `ErrMovieSpecUnavailable` |
| `spec_overrides.go` | ❌ Deleted | ✅ Owns `Apply`, `applyDecorOverrides`, `syncCursorFields` |
| `types_overrides.go` | ❌ Deleted | ✅ Owns `Overrides`, `ThemePreset`, `CursorPreset` |
| `overrides_test.go` | Reduced (public API) | ✅ Added (internal functions) |

**handlers/export/ now contains (thin wrappers only):**
- `types.go` - `Request` type + type aliases to services/export
- `builder.go` - Delegates to `services/export.BuildThemeFromPreset`, `BuildCursorSpec`
- `overrides.go` - Delegates to `services/export.Apply`
- `spec_builder.go` - Delegates to `services/export.BuildSpec`, `Clone`, `Harmonize`

**services/export/ now owns:**
- Types: `ReplayMovieSpec`, `ExportTheme`, `Overrides`, `ThemePreset`, `CursorPreset`, etc.
- Business logic: `BuildSpec`, `Apply`, `Clone`, `Harmonize`, `BuildThemeFromPreset`, `BuildCursorSpec`
- Presets: `ChromeThemePresets`, `BackgroundThemePresets`, `CursorThemePresets`
- Format strategies: Export service, markdown generation

### Action Type Mapping Boundary (2025-12-17)

**Two separate action type converters exist intentionally:**

| Function | Location | Returns for INPUT | Default |
|----------|----------|-------------------|---------|
| `ActionTypeToString()` | `internal/typeconv/primitives.go` | "input" | "unknown" |
| `actionTypeToString()` | `automation/executor/flow_utils.go` | "type" | "custom" |

**Why separate:**
- `typeconv.ActionTypeToString` is the **canonical form** for new code
- `flow_utils.actionTypeToString` is for **legacy workflow compatibility** when comparing against Type fields

**Usage:**
- New code should use `typeconv.ActionTypeToString()`
- The flow_utils version is private and only used internally for `IsActionType()` checks against legacy workflows

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

4. **RecordModeController Separation** (PARTIALLY ADDRESSED - 2026-01-03)
   - `recording/controller.ts` mixes domain logic (action normalization, confidence scoring) with integration (Playwright page interaction)
   - Replay preview execution has been delegated to `ReplayPreviewService` (already extracted)
   - Logger injection added to enable structured logging instead of console.log
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

4. **Handler Type Safety (Extended - 2026-01-03)**
   - Fixed `logger: any` parameters in `gesture.ts` (handleSwipe, handlePinchZoom)
   - Fixed `page: any, logger: any` parameters in `device.ts` (handleRotate)
   - All handlers now use properly typed `Page` and `winston.Logger` parameters

5. **Recording Module Structured Logging (2026-01-03)**
   - `RecordModeController` now accepts optional logger via constructor injection
   - Falls back to global logger if not provided (backward compatible)
   - Replaced 8 console.log/warn/error calls with `scopedLog()` structured logging
   - `action-executor.ts` uses structured logging for executor registration warnings

---

## Future Work

### API Client Abstraction (Medium Priority)

**Current State:** The recording domain has 33+ direct `fetch` calls scattered across 16+ files.

**Affected Files:**
- `hooks/usePages.ts`, `hooks/useRecordMode.ts`, `hooks/useTimeline.ts`
- `hooks/useBrowserNavigation.ts` (5 calls), `hooks/useSessionProfiles.ts` (5 calls)
- `components/RecordingSession.tsx` (3 calls), `ViewportSyncManager.ts`
- And 9+ other files

**Recommended Approach:**
1. Create `recording/api/client.ts` with typed methods
2. Centralize error handling and authentication
3. Migrate fetch calls incrementally

**Benefits:**
- Testable API layer with easy mocking
- Consistent error handling across all API calls
- Single point for auth token injection
- Better TypeScript types for request/response

### WebSocket Protocol Handler (Low Priority)

**Current State:** WebSocket message handling is spread across multiple components.

**Recommended Approach:**
- Extract protocol handling to dedicated service
- Centralize message type dispatching
- Enable unit testing of message handlers

### Workflow Sync Algorithm Tests (Low Priority)

**Current State:** `sync_test.go` tests detection functions only.

**Recommended Approach:**
- Core `SyncProjectWorkflows` algorithm tests require filesystem mocking infrastructure
- Consider using `afero` or similar for testable filesystem operations

---

## CaptureService Connect-RPC Seam (New, 2026-05-18)

**Location:** `api/handlers/capture/{service,module,mocks_test}.go`

Capture is the **first proto-first Connect-RPC handler in BAS**. The rest of the API is REST-only on chi. Capture mounts the generated `captureconnect.CaptureServiceHandler` next to the chi router — side-by-side, not replacing — so future domains can migrate incrementally. The list of mounted Connect services lives in `main.go` under the "Connect-RPC services (side-by-side with chi REST routes)" block; adding a second service is one line.

**Seam interface:**
- `capture.Deps.Executor` — calls `ExecuteAdhocWorkflowAPIWithOptions(ctx, *basexecution.ExecuteAdhocRequest, *workflow.ExecuteOptions)`. The capture handler builds a navigate DAG from the typed request and delegates; the executor owns artifact production.
- `capture.Deps.Resolver` — exposes `ResolveScenarioURLDefault(ctx, slug) (string, error)`. Lets capture accept the `scenario=<slug>,path=<path>` shorthand without coupling to a specific lookup implementation.
- `capture.Deps.Now` — clock injection for `duration_ms` so tests are deterministic.

**Tests:** Seven cases in `service_test.go` cover happy path, multi-capture, dimensions preset, dimensions explicit override, scenario shorthand resolution, dry-run short-circuit, and validation errors (empty URL, half-set width/height, UNSPECIFIED capture type, malformed shorthand, shorthand without resolver).

**Status:** Strong (proto-first contract, deps interface, mock-friendly).

---

## Evidence and Replay Package Seam

**Location:** `api/services/evidence`

Evidence owns the storage-independent contract for browser-captured material:

- `DefaultPolicy` defines classification, retention, access, size, and redaction defaults.
- `DescribeFile` computes SHA-256 and portable metadata without serializing a capture path.
- `SanitizeHAR` removes secret-bearing headers, query parameters, and request/response bodies before a derivative can leave protected storage.
- `BuildReplayPackage` creates the versioned `bas-evidence/v1` / `bas-replay/v1` renderer handoff from identifiers, manifests, timeline entries, and presentation metadata only.

**Disclosure boundary:** raw HAR remains `PROTECTED_STORAGE_ONLY`; neither execution artifact listings nor `/api/v1/recordings/assets/...` expose a URL or bytes for it. Recorded-HAR API and CLI commands return safe metadata (integrity, classification, retention, and access policy). Video and trace artifacts remain individually authorized through their asset URLs.

**Tests:** `services/evidence/*_test.go`, `automation/execution-writer/external_artifacts_test.go`, `services/workflow/execution_results_test.go`, and `handlers/recordings_test.go` cover policy assignment, secret redaction, path non-disclosure, storage-independent replay construction, and raw-HAR route rejection.

**Status:** Strong.

---

## Measures Ownership Boundary

Selector candidates and step telemetry are embedded evidence owned by workflows and executions. BAS does not expose standalone historical measures for either substrate: selector quality is evaluated by workflow/execution validation, and telemetry is consumed through its owning replay or execution artifact. The `session_checkpoints` table is bounded crash-recovery state, removed by completion and retention cleanup; current resumability is the relevant operational concern, not a historical trend. These three substrates are declared as explicit `measures.omitted` entries in `cli/manifest.json` so Measures Health does not imply they are unowned analytics domains.
