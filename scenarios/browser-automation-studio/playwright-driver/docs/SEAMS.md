# Playwright Driver - Architectural Seams

This document defines the **architectural boundaries** (seams) within the playwright-driver package. A "seam" is a place where you can alter behavior without editing existing code - understanding these boundaries helps you know where to make changes.

> **Last Updated**: 2026-01-03 (Seam discovery and boundary enforcement review)

## Module Boundaries

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          PLAYWRIGHT-DRIVER                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────────┐    ┌────────────────┐    ┌────────────────┐            │
│  │     PROTO      │    │    HANDLERS    │    │    OUTCOME     │            │
│  │  (Wire Format) │───▶│  (Execution)   │───▶│  (Results)     │            │
│  └────────┬───────┘    └───────┬────────┘    └────────────────┘            │
│           │                    │                     ▲                      │
│           │                    │                     │                      │
│           ▼                    ▼                     │                      │
│  ┌────────────────┐    ┌────────────────┐    ┌──────┴─────────┐            │
│  │   RECORDING    │    │    SESSION     │    │   TELEMETRY    │            │
│  │ (Capture/Play) │    │  (Lifecycle)   │    │  (Observability)│            │
│  └────────────────┘    └───────┬────────┘    └────────────────┘            │
│                                │                     ▲                      │
│           ┌────────────────────┴─────────────────────┤                      │
│           │                                          │                      │
│           ▼                                          │                      │
│  ┌────────────────┐    ┌────────────────┐    ┌──────┴─────────┐            │
│  │   EXECUTION    │    │     INFRA      │    │     UTILS      │            │
│  │  (Pipeline)    │    │(Infrastructure)│    │ (Cross-cutting)│            │
│  └────────────────┘    └────────────────┘    └────────────────┘            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Seam 1: Infrastructure Module (NEW)

**Location**: `src/infra/`

**Responsibility**: Cross-cutting infrastructure concerns that don't depend on domain logic

### Files and Their Boundaries

| File | Boundary | Can Change | Cannot Change |
|------|----------|------------|---------------|
| `index.ts` | Central exports | Re-exports | Public API surface |
| `idempotency-cache.ts` | Request deduplication | Cache implementation | `IdempotencyCache` interface |
| `session-cleanup-registry.ts` | Session cleanup coordination | Registration logic | `SessionCleanupFn` signature |
| `operation-tracker.ts` | Handler operation deduplication | Tracker implementation | `OperationTracker` interface |
| `circuit-breaker.ts` | Failure isolation | Breaker logic | `CircuitBreaker` interface |

### Key Patterns

#### Session Cleanup Registry

The session cleanup registry creates a seam between session management and handler-specific cleanup:

```typescript
// In handlers - register cleanup at module load
registerSessionCleanup('my-handler-cache', (sessionId) => {
  myCache.clearSession(sessionId);
});

// In session manager - call cleanup without knowing handler details
await cleanupSession(sessionId);
```

**Benefits**:
- Session manager doesn't import handler modules directly
- Handlers own their cleanup logic
- Easy to add new cleanup operations

#### Operation Tracker

The operation tracker extracts the common pattern of tracking concurrent operations:

```typescript
// Create a tracker for your handler
const myTracker = createOperationTracker<HandlerResult>({
  name: 'my-handler',
  cacheResults: true,  // Optional: cache successful results
});

// In handler execute():
const inFlight = myTracker.getInFlight(sessionId, operationKey);
if (inFlight) return inFlight;  // Await pending operation

const cached = myTracker.getCached(sessionId, operationKey);
if (cached) return cached;  // Return cached result

const promise = doWork();
const cleanup = myTracker.trackInFlight(sessionId, operationKey, promise);
try {
  const result = await promise;
  myTracker.cacheResult(sessionId, operationKey, result);
  return result;
} finally {
  cleanup();
}
```

**Benefits**:
- Eliminates module-level global Maps in handlers
- Automatic session cleanup via registry integration
- Testable (create isolated instances for testing)

### Change Axes

1. **Adding a new infrastructure service**
   - Create new file in `infra/`
   - Export from `index.ts`
   - Register with cleanup registry if session-scoped

2. **Adding handler cleanup**
   - Call `registerSessionCleanup()` in handler module
   - No changes to session manager needed

## Seam 2: Execution Module

**Location**: `src/execution/`

**Responsibility**: Core domain logic for executing browser automation instructions

### Files and Their Boundaries

| File | Boundary | Can Change | Cannot Change |
|------|----------|------------|---------------|
| `index.ts` | Central exports | Re-exports | Public API surface |
| `instruction-executor.ts` | Execution pipeline | Internal logic | `ExecutionResult` interface |

### Execution Pipeline

The execution module provides a clean seam between HTTP routing and domain execution:

```
┌─────────────────────────────────────────────────────────────────────────┐
│ EXECUTION PIPELINE (instruction-executor.ts):                           │
│                                                                         │
│   validateInstruction() ──▶ Parse and validate instruction structure    │
│          │                                                              │
│          ▼                                                              │
│   executeInstruction() ──▶ Core execution logic                         │
│          │                                                              │
│          ├──▶ Handler dispatch (via registry)                           │
│          ├──▶ Telemetry orchestration                                   │
│          └──▶ Outcome building                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Key Patterns

#### Unified Context Type

The `ExecutionContext` is an alias for `HandlerContext`, ensuring a single context type across the execution pipeline:

```typescript
// In instruction-executor.ts
export type ExecutionContext = HandlerContext;

// Usage: both executor and handlers use the same type
const result = await handler.execute(instruction, context);
```

#### Stateless Execution

The executor is stateless - all context flows through parameters:

```typescript
const result = await executeInstruction(
  instruction,    // What to execute
  context,        // Execution context (page, config, etc.)
  handlerRegistry // Where to find handlers
);
```

### Change Axes

1. **Adding execution stages**
   - Modify `executeInstruction()` in instruction-executor.ts
   - Keep stages clearly separated (validation, dispatch, telemetry, outcome)

2. **Changing validation rules**
   - Modify `validateInstruction()` and `validateInstructionStructure()`
   - Update `ValidationError` type if needed

## Seam 3: Telemetry Module

**Location**: `src/telemetry/`

**Responsibility**: Observability data collection (screenshots, DOM, console, network, element context)

### Files and Their Boundaries

| File | Boundary | Can Change | Cannot Change |
|------|----------|------------|---------------|
| `index.ts` | Central exports | Re-exports | Public API surface |
| `orchestrator.ts` | **Unified collection** | Collection logic | `StepTelemetry` interface |
| `screenshot.ts` | Screenshot capture | Capture implementation | `Screenshot` type |
| `dom.ts` | DOM snapshot | Snapshot implementation | `DOMSnapshot` type |
| `collector.ts` | Console/Network | Collector implementations | Collector interfaces |
| `element-context.ts` | Element metadata | Context capture | `ElementContext` type |

### Telemetry Orchestrator Pattern

The TelemetryOrchestrator is the single entry point for all telemetry operations:

```typescript
// Create and start
const orchestrator = new TelemetryOrchestrator(page, config);
orchestrator.start();

// Capture element context BEFORE action (optional)
const elementContext = await orchestrator.captureElementContextForAction(selector);

// Collect telemetry AFTER action
const telemetry = await orchestrator.collectForStep(handlerResult);

// Cleanup
orchestrator.dispose();
```

### Key Patterns

#### Pre-Action Element Context

Element context capture provides recording-quality telemetry for execution:

```typescript
// In handler before action:
const elementContext = await orchestrator.captureElementContextForAction(selector);

// Include in result:
return {
  success: true,
  elementContext,  // Added to StepOutcome via outcome-builder
};
```

#### Unified StepTelemetry

All telemetry flows through a single interface:

```typescript
interface StepTelemetry {
  screenshot?: Screenshot;
  domSnapshot?: DOMSnapshot;
  consoleLogs?: ConsoleLogEntry[];
  networkEvents?: NetworkEvent[];
  elementContext?: ElementContext;  // Pre-action element state
}
```

### Change Axes

1. **Adding a new telemetry type**
   - Create collector file (e.g., `telemetry/performance.ts`)
   - Define type in proto schema and regenerate
   - Add to `StepTelemetry` interface
   - Add collection method to `TelemetryOrchestrator`
   - Update `outcome/outcome-builder.ts` to include in output
   - Add config options in `config.ts`

2. **Modifying element context capture**
   - Edit `element-context.ts`
   - Access via `orchestrator.captureElementContextForAction()`

## Seam 4: Proto Module (Wire Format)

**Location**: `src/proto/`

**Responsibility**: Wire format conversion between Go API and TypeScript driver

### Files and Their Boundaries

| File | Boundary | Can Change | Cannot Change |
|------|----------|------------|---------------|
| `index.ts` | Central hub | Re-exports | Proto type definitions |
| `params.ts` | Param extraction | Extractor implementations | Function signatures |
| `utils.ts` | JSON utilities | Helper functions | Parse/serialize contracts |
| `recording.ts` | Timeline conversion | Browser event handling | TimelineEntry schema |

### Change Axes

1. **Adding a new action type param extractor**
   - Add to `params.ts`
   - Export from `index.ts`
   - No other files need changes

2. **Changing wire format**
   - Requires proto schema change (Go API)
   - Then update `utils.ts` parse options

## Seam 5: Handlers Module

**Location**: `src/handlers/`

**Responsibility**: Execute browser automation instructions

### Files and Their Boundaries

| File | Boundary | Can Change | Cannot Change |
|------|----------|------------|---------------|
| `base.ts` | Handler contract | Utility methods | `InstructionHandler` interface |
| `registry.ts` | Handler dispatch | Lookup logic | Handler registration pattern |
| `behavior-utils.ts` | Human-like behavior | Behavior helpers | Behavior interface |
| `*.ts` (handlers) | Individual execution | Implementation | Return type (`HandlerResult`) |

### Handler Architecture Pattern

Handlers follow a consistent pattern for operation deduplication:

```typescript
export class MyHandler extends BaseHandler {
  async execute(instruction, context): Promise<HandlerResult> {
    const { sessionId } = context;
    const operationKey = generateKey(instruction);

    // 1. Check for in-flight operation
    const inFlight = myTracker.getInFlight(sessionId, operationKey);
    if (inFlight) return inFlight;

    // 2. Check for cached result (if applicable)
    const cached = myTracker.getCached(sessionId, operationKey);
    if (cached) return cached;

    // 3. Execute and track
    const promise = this.doWork(instruction, context);
    const cleanup = myTracker.trackInFlight(sessionId, operationKey, promise);

    try {
      const result = await promise;
      if (result.success) {
        myTracker.cacheResult(sessionId, operationKey, result);
      }
      return result;
    } finally {
      cleanup();
    }
  }
}
```

### Change Axes

1. **Adding a new instruction type**
   - Create new handler file
   - Implement `InstructionHandler`
   - Register in `server.ts`
   - Add param extractor in `proto/params.ts`
   - Use operation tracker for deduplication

2. **Modifying handler behavior**
   - Edit specific handler file
   - No other files affected

## Seam 6: Recording Module

**Location**: `src/recording/`

**Responsibility**: Capture and replay browser interactions

### Files and Their Boundaries

| File | Boundary | Can Change | Cannot Change |
|------|----------|------------|---------------|
| `action-types.ts` | **Single source of truth** for action type mapping | Mappings | Export names |
| `action-executor.ts` | Replay execution | Executor implementations | `TimelineExecutor` signature |
| `handler-adapter.ts` | **Bridge to handlers** | Conversion logic | Delegation pattern |
| `controller.ts` | Recording orchestration | Internal logic | `RecordModeController` public API |
| `injector.ts` | Browser-side code | Selector generation | Communication protocol |
| `buffer.ts` | Timeline entry storage | Buffer implementation | Public API |
| `replay-service.ts` | Replay preview | Replay logic | `ReplayPreviewService` API |

### Key Patterns

#### Session Cleanup Integration

The recording buffer registers with the session cleanup registry for automatic cleanup:

```typescript
// In buffer.ts
registerSessionCleanup('recording-buffer', (sessionId: string): void => {
  removeRecordingBuffer(sessionId);
});
```

This ensures buffer cleanup happens automatically when sessions close, without the session layer needing to know about recording internals.

#### Separation of Capture and Replay

Recording concerns are split between:
- **RecordModeController**: Event capture, conversion to TimelineEntry
- **ReplayPreviewService**: Replay execution of TimelineEntry actions

```typescript
// Controller handles capture
const controller = new RecordModeController(page, sessionId);
await controller.startRecording({ onEntry: handleEntry });

// Replay service handles replay (delegated from controller)
const response = await controller.replayPreview({ entries, stopOnFailure: true });
```

### Change Axes

1. **Adding a new recording action type**
   - Add to `ACTION_TYPE_MAP` in `action-types.ts`
   - **Option A**: Implement handler in `handlers/` (recommended)
     - Handler is automatically available for replay via handler-adapter
   - **Option B**: Add executor in `action-executor.ts` via `registerTimelineExecutor()`
   - Update `injector.ts` if browser capture needed

2. **Modifying selector generation**
   - Edit `injector.ts` (browser code)
   - Edit `selector-config.ts` (scoring)

3. **Modifying buffer behavior**
   - Edit `buffer.ts`
   - Cleanup is automatic via session cleanup registry

## Seam 7: Session Module

**Location**: `src/session/`

**Responsibility**: Browser session lifecycle management

### Files and Their Boundaries

| File | Boundary | Can Change | Cannot Change |
|------|----------|------------|---------------|
| `manager.ts` | Session lifecycle | Internal logic | Public API |
| `cleanup.ts` | Background cleanup task | Cleanup intervals | Integration with manager |
| `context-builder.ts` | Browser context creation | Context options | Return type |
| `state-machine.ts` | Phase transitions | Transition rules | Phase definitions |
| `browser-manager.ts` | Browser instance management | Launch options | Browser lifecycle |

### Session Cleanup Integration

The session manager uses the cleanup registry for handler cleanup:

```typescript
// In manager.ts - resetSession()
await cleanupSession(sessionId);  // Clears all handler state
```

This eliminates direct imports from session layer into handler modules.

## Seam 8: Outcome Module

**Location**: `src/outcome/`

**Responsibility**: Transform execution results to wire format

### Files and Their Boundaries

| File | Boundary | Can Change | Cannot Change |
|------|----------|------------|---------------|
| `types.ts` | Shared error codes/utilities | Error code mappings | Core interfaces |
| `outcome-builder.ts` | StepOutcome construction | Building logic | `HandlerResult` interface |

### Change Axes

1. **Adding new error codes**
   - Add to `HandlerErrorCode` or `ActionErrorCode` in `types.ts`
   - Map to `FailureKind` in `ERROR_CODE_TO_FAILURE_KIND`

2. **Modifying outcome structure**
   - Requires proto schema change first
   - Then update `outcome-builder.ts`

## Key Architectural Principles

### 1. Single Source of Truth

| Concept | Single Source | Location |
|---------|---------------|----------|
| Action type mapping | `ACTION_TYPE_MAP` | `recording/action-types.ts` |
| Param extractors | `get*Params()` functions | `proto/params.ts` |
| Error code classification | `ERROR_CODE_TO_FAILURE_KIND` | `outcome/types.ts` |
| Wire format conversion | `toHandlerInstruction()` | `proto/index.ts` |
| **Action execution** | `handlers/*.ts` | `handlers/` (canonical) |
| **Session cleanup** | `registerSessionCleanup()` | `infra/session-cleanup-registry.ts` |
| **Operation tracking** | `OperationTracker` | `infra/operation-tracker.ts` |
| **Execution pipeline** | `executeInstruction()` | `execution/instruction-executor.ts` |
| **Telemetry collection** | `TelemetryOrchestrator` | `telemetry/orchestrator.ts` |
| **Handler context** | `HandlerContext` | `handlers/base.ts` |

### 2. Import Direction

```
infra/ ◄───────────────────────────────────────────┐
   │                                               │
   ▼                                               │
proto/ ◄───────────────────────────────────────────┤
   │                                               │
   ▼                                               │
execution/ ──▶ handlers/ ──────────────────▶ outcome/
   │               │                               ▲
   │               │                               │
   ▼               ▼                               │
telemetry/ ◄── recording/ ─────────────────────────┘
   │               │
   │               └──▶ handler-adapter.ts ──▶ handlers/
   │
   └──▶ element-context.ts (pre-action telemetry)
```

- `infra/` is the foundation (imported by all, imports only utils)
- `execution/` orchestrates the pipeline, imports handlers and telemetry
- `telemetry/` provides unified telemetry collection (screenshot, DOM, element-context)
- `proto/` is imported by handlers and recording
- `outcome/` can be imported by handlers and recording
- `handler-adapter.ts` bridges recording to handlers for replay execution
- `recording/` registers with `infra/` for session cleanup

### 3. Circular Import Prevention

- Infrastructure modules import only from `utils/`
- Handlers import from `infra/` for tracking, not vice versa
- Session cleanup uses registry pattern to avoid importing handlers

### 4. Responsibility Boundaries

| Layer | Owns | Does Not Own |
|-------|------|--------------|
| `infra/` | Caching, tracking, circuit breaking | Domain logic |
| `execution/` | Execution pipeline orchestration | HTTP routing, session state |
| `telemetry/` | Observability data collection | Outcome building |
| `handlers/` | Instruction execution | Session lifecycle |
| `session/` | Browser lifecycle | Handler internals |
| `recording/` | Capture and replay | Direct instruction execution |
| `outcome/` | Result transformation | Telemetry collection |

## Testing Boundaries

| Module | Test Type | Location |
|--------|-----------|----------|
| Infra | Unit | `tests/unit/infra/` |
| Execution | Unit | `tests/unit/execution/` |
| Telemetry | Unit | `tests/unit/telemetry/` |
| Proto | Unit | `tests/unit/proto/` |
| Handlers | Integration | `tests/integration/handlers/` |
| Recording | Unit + Integration | `tests/unit/recording/` |
| Outcome | Unit | `tests/unit/outcome/` |
| Session | Integration | `tests/integration/session/` |

## Evolution Guidelines

### Safe Changes (Won't Break Other Code)

- Adding new handler files
- Adding new param extractors
- Adding new action type mappings
- Adding new error codes
- Internal refactoring within a module
- Adding new cleanup registrations
- Adding new operation trackers
- Adding new telemetry collectors (via orchestrator)
- Adding optional fields to `HandlerContext`

### Dangerous Changes (May Break Other Code)

- Modifying `HandlerResult` interface
- Modifying `TimelineEntry` structure
- Renaming exported functions
- Changing handler registration pattern
- Modifying the `ACTION_TYPE_MAP` key format
- Changing `OperationTracker` interface
- Changing `SessionCleanupFn` signature
- Modifying `StepTelemetry` interface (affects execution pipeline)
- Changing required fields in `HandlerContext`

### Requires Coordination

- Wire format changes (Go API must change first)
- New instruction types (proto schema + param extractor + handler)
- Recording format changes (affects UI and API)
- Execution pipeline changes (affects all instruction execution)
