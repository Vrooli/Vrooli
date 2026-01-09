# AutomationEngine Architecture Analysis

## Overview

The AutomationEngine is an abstraction layer that decouples workflow execution from browser automation backends. It enables the system to support multiple automation providers (currently Playwright only, previously Browserless) without changing the core execution logic.

**Location**: `api/automation/engine/`

## Architecture

### Core Interfaces

#### AutomationEngine (`engine.go:32-36`)
```go
type AutomationEngine interface {
    Name() string
    Capabilities(ctx context.Context) (contracts.EngineCapabilities, error)
    StartSession(ctx context.Context, spec SessionSpec) (EngineSession, error)
}
```

**Purpose**: Factory for creating browser automation sessions

**Responsibilities**:
- Advertise capabilities (HAR, video, tracing, iframes, etc.)
- Create configured browser sessions
- Health check the underlying driver

#### EngineSession (`engine.go:39-43`)
```go
type EngineSession interface {
    Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error)
    Reset(ctx context.Context) error
    Close(ctx context.Context) error
}
```

**Purpose**: Execute instructions in a browser context

**Responsibilities**:
- Execute single instruction → return normalized outcome
- Reset state (cookies, storage) for 'clean' reuse mode
- Clean up resources

### Contract Types

All located in `api/automation/contracts/`:

#### CompiledInstruction (`contracts/plan.go:32-40`)
Engine-agnostic input format:
```go
type CompiledInstruction struct {
    Index       int
    NodeID      string
    Type        string                // "click", "navigate", etc.
    Params      map[string]any        // Generic parameters
    PreloadHTML string
    Context     map[string]any
    Metadata    map[string]string
}
```

**Key Constraint**: Params must be engine-agnostic. Engines cannot add vendor-specific fields.

#### StepOutcome (`contracts/contracts.go:52-84`)
Normalized output format with comprehensive telemetry:

```go
type StepOutcome struct {
    SchemaVersion      string                // Versioning for backward compatibility
    PayloadVersion     string
    ExecutionID        uuid.UUID
    CorrelationID      string               // For deduplication
    StepIndex          int
    Attempt            int
    NodeID             string
    StepType           string
    Success            bool
    StartedAt          time.Time
    CompletedAt        *time.Time
    DurationMs         int

    // Telemetry
    Screenshot         *Screenshot          // PNG bytes, hash, truncation flags
    DOMSnapshot        *DOMSnapshot         // HTML, size limits, truncation
    ConsoleLogs        []ConsoleLogEntry    // Console events
    Network            []NetworkEvent       // Network activity

    // Results
    ExtractedData      map[string]any       // Extraction results
    Assertion          *AssertionOutcome    // Assert step results
    Condition          *ConditionOutcome    // Branch condition results

    // UI Feedback
    ElementBoundingBox *BoundingBox         // Target element position
    ClickPosition      *Point               // Actual click coordinates
    FocusedElement     *ElementFocus        // Focus metadata
    CursorTrail        []CursorPosition     // Mouse movement trail

    // Error Handling
    Failure            *StepFailure         // Taxonomized failure
}
```

**Key Features**:
- Schema versioning prevents silent drift
- Comprehensive telemetry for replay/debugging
- Taxonomized failures for retry logic
- Size limits enforced by recorder (not engine)

#### StepFailure (`contracts/contracts.go:88-97`)
Failure taxonomy for retry logic:
```go
type StepFailure struct {
    Kind       FailureKind    // engine|infra|orchestration|user|timeout|cancelled
    Code       string         // Machine-readable error code
    Message    string         // Human-readable description
    Fatal      bool           // Must abort workflow
    Retryable  bool           // May retry based on policy
    OccurredAt *time.Time
    Details    map[string]any
    Source     FailureSource  // engine|executor|recorder
}
```

**Purpose**: Enable intelligent retry/abort decisions by executor

#### EngineCapabilities (`contracts/capabilities.go:10-27`)
Advertisement of engine features:
```go
type EngineCapabilities struct {
    SchemaVersion         string
    Engine                string  // e.g., "playwright", "browserless"
    Version               string
    RequiresDocker        bool
    RequiresXvfb          bool
    MaxConcurrentSessions int
    AllowsParallelTabs    bool
    SupportsHAR           bool
    SupportsVideo         bool
    SupportsIframes       bool    // ⚠️ Playwright reports true but not implemented!
    SupportsFileUploads   bool
    SupportsDownloads     bool
    SupportsTracing       bool
    MaxViewportWidth      int
    MaxViewportHeight     int
}
```

**Purpose**: Enable preflight validation and engine selection

### Component Interactions

```
┌─────────────────────────────────────────────────────────────────┐
│ WorkflowService (services/workflow_service_automation.go)       │
│  - Compiles workflow → ExecutionPlan                            │
│  - Resolves engine via Factory                                  │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│ SimpleExecutor (executor/simple_executor.go)                    │
│  - Orchestrates execution (loops, retries, variables)           │
│  - Emits telemetry/heartbeats via EventSink                     │
│  - Validates capabilities before starting                       │
└───────┬───────────────────────────────┬─────────────────────────┘
        │                               │
        ▼                               ▼
┌─────────────────────┐    ┌────────────────────────────────────┐
│ AutomationEngine    │    │ Recorder (recorder/db_recorder.go) │
│  (Interface)        │    │  - Persists StepOutcome to DB      │
│                     │    │  - Generates IDs/dedupe keys       │
│ ┌─────────────────┐ │    │  - Applies size limits             │
│ │ PlaywrightEngine│ │    │  - Uploads screenshots to MinIO   │
│ │  (Go adapter)   │ │    └────────────────────────────────────┘
│ └────────┬────────┘ │
│          │          │                ┌──────────────────────┐
│          ▼          │                │ EventSink            │
│  ┌──────────────┐   │                │  - WebSocket events  │
│  │ Node.js      │   │                │  - Ordering          │
│  │ Playwright   │◄──┼────────────────┤  - Backpressure      │
│  │ Driver       │   │   HTTP         └──────────────────────┘
│  │ (server.js)  │   │
│  └──────────────┘   │
└─────────────────────┘
```

## Current Implementation: Playwright Engine

### Go Adapter: `PlaywrightEngine` (`engine/playwright_engine.go`)

**Design**: Thin HTTP client (~ 342 lines)

**Key Methods**:
```go
func (e *PlaywrightEngine) Capabilities(ctx context.Context) (contracts.EngineCapabilities, error)
// Returns capability descriptor after health check

func (e *PlaywrightEngine) StartSession(ctx context.Context, spec SessionSpec) (EngineSession, error)
// POSTs to /session/start, returns playwrightSession

func (s *playwrightSession) Run(ctx context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error)
// POSTs to /session/:id/run, decodes StepOutcome from driver
```

**Translation Layer**:
- Input: `CompiledInstruction` → HTTP JSON
- Output: HTTP JSON → `StepOutcome` (decodes base64 screenshots, normalizes timestamps)

**Configuration**:
- `PLAYWRIGHT_DRIVER_URL` (default: `http://127.0.0.1:39400`)

### Node.js Driver: `playwright-driver/server.js`

**Design**: Monolithic HTTP server (503 lines)

**Endpoints**:
- `POST /session/start` → `{ session_id }`
- `GET /health` → `{ status, sessions }`
- `POST /session/:id/run` → `StepOutcome` JSON
- `POST /session/:id/reset` → `{ status: "ok" }`
- `POST /session/:id/close` → `{ status: "closed" }`

**Responsibilities**:
- Launch Chromium via Playwright
- Execute instructions
- Collect console/network events
- Capture screenshots and DOM
- Return normalized `StepOutcome` JSON

**Supported Instructions** (13/28):
✅ navigate, click, hover, type, scroll, wait, evaluate, extract, assert, uploadfile, download, screenshot

**Missing Instructions** (15/28):
❌ **frame-switch** (CRITICAL - marked as supported!), blur, focus, cookie-storage, drag-drop, gesture, keyboard, shortcut, network-mock, rotate, select, tab-switch

## Browserless Engine: Removed

### Historical Context

The system previously supported a `BrowserlessEngine` that wrapped the CDP-based `browserless/client.go`.

**Removal Date**: November 2024 (per `automation/README.md:58`)

**Reason**: Migration to Playwright for:
- Improved stability
- Better feature support (HAR, video, tracing)
- Simpler desktop/Electron integration

**Current State**:
- All browserless-specific code removed from scenario
- Legacy references remain only in:
  - Plan compiler registry (alias "browserless" → default compiler)
  - Default engine name in executor (should be updated to "playwright")
  - Documentation describing the migration

**No Direct Usage**: Zero browserless imports or direct usage in API code

## Engine Selection

### Factory Pattern (`engine/static_factory.go`)

```go
type Factory interface {
    Resolve(ctx context.Context, name string) (AutomationEngine, error)
}

type StaticFactory struct {
    engines map[string]AutomationEngine
}
```

**Registration**:
```go
func DefaultFactory(log *logrus.Logger) (Factory, error) {
    pw, err := NewPlaywrightEngineWithDefault(log)
    if err != nil {
        return nil, fmt.Errorf("playwright engine required: %w", err)
    }
    return NewStaticFactory(pw), nil
}
```

**Selection Logic** (`engine/selection.go`):
```go
type SelectionConfig struct {
    DefaultEngine string  // From ENGINE env var
    Override      string  // From ENGINE_OVERRIDE env var
}

func (c SelectionConfig) Resolve(requested string) string {
    // Override > Requested > Default
}
```

**Environment Variables**:
- `ENGINE` - Default engine (currently defaults to "browserless", should be "playwright")
- `ENGINE_OVERRIDE` - Force all executions to specific engine
- `PLAYWRIGHT_DRIVER_URL` - URL to Playwright driver

## Orchestration Layer

### SimpleExecutor (`executor/simple_executor.go`)

**Responsibilities**:
1. Compile workflow → ExecutionPlan
2. Validate engine capabilities against requirements
3. Create engine session with SessionSpec
4. Execute instructions (linear or graph-based)
5. Handle retries, loops, variables
6. Emit telemetry and heartbeats
7. Record outcomes via Recorder
8. Emit events via EventSink

**Key Features**:
- **Capability Preflight**: Analyzes plan, derives requirements, validates against engine
- **Session Reuse Modes**:
  - `fresh` - New session per step
  - `clean` - Reuse process, reset storage
  - `reuse` - Keep state intact
- **Control Flow**: Supports loops (repeat, forEach, while), conditionals, subflows
- **Variable Interpolation**: `${var}` replacement in params
- **Failure Handling**: Maps FailureKind to retry/abort decisions

### Recorder (`recorder/db_recorder.go`)

**Responsibilities**:
- Persist `StepOutcome` to database
- Generate correlation IDs for deduplication
- Apply size limits to DOM/console/network
- Upload screenshots to MinIO (or local storage)
- Annotate truncation flags

**Key Constraint**: Engines must not generate IDs or apply truncation—recorder owns this

### EventSink (`events/ws_sink.go`)

**Responsibilities**:
- Emit WebSocket events for UI updates
- Enforce ordering per execution
- Handle backpressure (queue limits, drop counters)
- Never drop completion/failure events

## Critical Findings

### 🚨 Contract Violation

**Issue**: PlaywrightEngine reports `SupportsIframes: true` but frame-switch instruction is not implemented in the driver.

**Location**: `playwright_engine.go:74`

**Impact**: Workflows using iframe navigation will fail with "unsupported step type: frame-switch" error, despite preflight capability check passing.

**Fix**: Implement frame-switch handler in `server.js` (see Phase 2 of completion plan)

### Missing Features Summary

| Category | Implemented | Missing |
|----------|------------|---------|
| Navigation | navigate | frame-switch (iframe navigation) |
| Interaction | click, hover, type | blur, focus, drag-drop, gesture |
| Input | type, uploadfile | select (dropdown) |
| Keyboard | - | keyboard, shortcut |
| State | - | cookie-storage (cookies/localStorage) |
| Network | - | network-mock (request interception) |
| Device | - | rotate (orientation) |
| Multi-context | - | tab-switch (multi-tab) |
| **Total** | **13/28** | **15/28** |

### Architectural Strengths

✅ **Clean Separation of Concerns**:
- Engine handles only execution
- Executor handles orchestration
- Recorder handles persistence
- EventSink handles streaming

✅ **Type Safety via Contracts**:
- Explicit schema versioning
- Taxonomized failures
- Capability-based preflight

✅ **Engine Agnostic**:
- CompiledInstruction has no vendor-specific fields
- Multiple engines can share same executor/recorder
- Easy to add new engines (e.g., Puppeteer, Selenium)

✅ **Observability**:
- Comprehensive telemetry
- Failure taxonomy for debugging
- Cursor trails for replay

### Architectural Weaknesses

❌ **Monolithic Driver**: 503-line JavaScript file is unmaintainable

❌ **No Type Safety**: JavaScript lacks compile-time checking

❌ **Incomplete Implementation**: 15/28 features missing, including one contract violation

❌ **No Resource Management**: No session timeouts, cleanup, or limits

❌ **Minimal Observability**: No structured logging or metrics in driver

## Recommendations

See detailed completion plan in `docs/plans/playwright-driver-completion.md`:

1. **Phase 1 (Week 1)**: Migrate to TypeScript with modular architecture
2. **Phase 2 (Week 2)**: Implement missing critical features (frame-switch, focus, cookies, etc.)
3. **Phase 3 (Week 3)**: Implement advanced features (drag-drop, multi-tab, network mock)
4. **Phase 4 (Week 4)**: Production hardening (performance, monitoring, documentation)

**Total Timeline**: 4 weeks

**Priority 1**: Fix frame-switch contract violation

---

## References

- **Engine Interface**: `api/automation/engine/engine.go`
- **Contracts**: `api/automation/contracts/*.go`
- **Playwright Engine**: `api/automation/engine/playwright_engine.go`
- **Playwright Driver**: `playwright-driver/src/server.ts` (TypeScript v2.0)
- **Executor**: `api/automation/executor/simple_executor.go`
- **Recorder**: `api/automation/recorder/db_recorder.go`
- **Architecture Docs**: `api/automation/README.md`, `api/automation/engine/README.md`
- **Completion Plan**: `docs/plans/playwright-driver-completion.md`
