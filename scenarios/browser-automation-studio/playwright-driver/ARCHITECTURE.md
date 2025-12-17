# Playwright Driver Architecture

This document describes the architecture of the Playwright Driver, focusing on change axes, stability boundaries, and extension points for future evolution.

## Stability Classification

Each module is classified by its stability:

| Classification | Meaning | Change Frequency |
|---------------|---------|------------------|
| **STABLE CONTRACT** | Wire format with external systems (Go API) | Very Low |
| **STABLE CORE** | Internal abstractions unlikely to change | Low |
| **VOLATILE EDGE** | Expected to evolve frequently | High |
| **VOLATILE (by design)** | Primary extension points | Medium-High |

## Module Overview

```
src/
├── handlers/        # VOLATILE (by design) - Instruction execution
├── recording/       # VOLATILE (by design) - Record mode feature
├── telemetry/       # VOLATILE EDGE - Observability data capture
├── frame-streaming/ # VOLATILE EDGE - Live frame streaming to UI
├── infra/           # STABLE CORE - Cross-cutting infrastructure patterns
├── routes/          # STABLE CORE + VOLATILE EDGE
├── session/         # STABLE CORE - Session lifecycle
│   ├── manager.ts        # Session CRUD operations
│   └── browser-manager.ts # Browser lifecycle (launch, verify, shutdown)
├── outcome/         # STABLE CONTRACT - Wire format transformation
├── types/           # Mixed - See types/index.ts for details
├── fps/             # STABLE CORE - Adaptive FPS control
├── performance/     # VOLATILE EDGE - Performance metrics collection
├── utils/           # STABLE CORE - Infrastructure concerns
├── config.ts        # VOLATILE EDGE - Configuration options
├── constants.ts     # STABLE CORE - Hardcoded defaults
└── server.ts        # STABLE CORE - HTTP server setup
```

## Primary Change Axes

### 1. New Instruction Types (HIGH FREQUENCY)

**Primary extension point:** `handlers/`

When adding a new browser automation capability:

1. Create handler file: `handlers/my-action.ts`
2. Implement `InstructionHandler` interface from `handlers/base.ts`
3. Add Zod schema to `types/instruction.ts`
4. Export from `handlers/index.ts`
5. Register in `server.ts:registerHandlers()`

**Example:**
```typescript
// handlers/my-action.ts
export class MyActionHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['my-action'];
  }

  async execute(instruction: CompiledInstruction, context: HandlerContext): Promise<HandlerResult> {
    const params = MyActionParamsSchema.parse(instruction.params);
    // ... implementation
    return { success: true };
  }
}
```

**Tests:** `tests/unit/handlers/` - one test file per handler

### 2. Recording Action Types (MEDIUM FREQUENCY)

**Primary extension points:**
- `recording/action-types.ts` - Type normalization, kind mapping, confidence calculation
- `recording/action-executor.ts` - Replay execution for each action type

When adding a new recorded action type:

1. Add to `ACTION_TYPES` constant in `action-types.ts`
2. Add alias mapping if needed (e.g., 'input' -> 'type')
3. Add proto kind mapping in `ACTION_KIND_MAP`
4. Add to `SELECTOR_OPTIONAL_ACTIONS` if no selector required
5. Update `buildTypedActionPayload()` for typed payloads
6. Register executor in `action-executor.ts` using `registerActionExecutor()`
7. Add tests in `tests/unit/recording/action-executor.test.ts`

**Note:** The action executor registry pattern eliminates the need to modify
`controller.ts` when adding new action types - just register your executor.

### 3. Selector Strategies (MEDIUM FREQUENCY)

**Primary extension point:**
- `recording/injector.ts` - Runtime implementation (browser-injected) **SOURCE OF TRUTH**
- `recording/selectors.reference.ts` - TypeScript reference (documentation only, not imported)

**Note:** `selectors.reference.ts` is documentation-only and may drift from `injector.ts`. The injector is the source of truth. Update the reference file opportunistically when making algorithm changes.

When adding a new selector strategy:

1. Add `SelectorType` to `recording/types.ts`
2. Implement in `injector.ts` (the actual runtime code)
3. Optionally update `selectors.reference.ts` for documentation
4. Update `DEFAULT_SELECTOR_OPTIONS` if needed

### 4. Telemetry Types (LOW-MEDIUM FREQUENCY)

**Primary extension point:** `telemetry/`

When adding new observability data:

1. Create collector file: `telemetry/my-collector.ts`
2. Define types in `types/contracts.ts` (coordinate with Go API)
3. Add to `StepOutcome` interface if needed
4. Update `outcome/outcome-builder.ts` to include in output
5. Add config options in `config.ts`
6. Use collector in `routes/session-run.ts`

### 5. Frame Streaming (LOW-MEDIUM FREQUENCY)

**Primary extension point:** `frame-streaming/`

The frame streaming module handles live video streaming to the UI during recording:

- **CDP Screencast** (preferred): Push-based streaming from Chrome compositor (30-60 FPS)
- **Polling Fallback**: Pull-based screenshot capture with adaptive FPS

Key files:
- `frame-streaming/manager.ts` - Main streaming logic, WebSocket management
- `frame-streaming/types.ts` - Types and constants
- `routes/record-mode/screencast-streaming.ts` - CDP screencast implementation

When modifying frame streaming:
1. Configuration options are in `config.ts` under `frameStreaming`
2. FPS control is handled by `fps/controller.ts`
3. Performance metrics are collected by `performance/collector.ts`

### 6. Wire Format (LOW FREQUENCY, HIGH RISK)

**Primary extension point:** `types/contracts.ts` + `outcome/outcome-builder.ts`

**CRITICAL:** Changes here affect the Go API contract.

- Adding fields is safe (Go uses `omitempty`)
- Removing/renaming fields is a **BREAKING CHANGE**
- Coordinate with Go API team before changes

## Shotgun Surgery Mitigation

The following patterns reduce multi-file changes:

### Circuit Breaker Pattern

`infra/circuit-breaker.ts` provides a reusable circuit breaker for failure-prone operations:

```typescript
import { createCircuitBreaker } from './infra';

const breaker = createCircuitBreaker<string>({
  maxFailures: 5,
  resetTimeoutMs: 30_000,
  name: 'my-service',
});

// Usage
if (breaker.isOpen(key) && !breaker.tryEnterHalfOpen(key)) {
  return; // Skip operation, circuit is open
}
try {
  await riskyOperation();
  breaker.recordSuccess(key);
} catch (err) {
  breaker.recordFailure(key);
}
```

Currently used for callback streaming in record mode. Can be reused for any operation
that may fail repeatedly (external APIs, network calls, etc.).

### Handler Registration

Handlers self-register via `getSupportedTypes()`. Adding a handler only requires:
- Create handler file
- Register in `server.ts` (single line)

### Action Type Registry

`action-types.ts` centralizes action type handling:
- Type normalization
- Proto kind mapping
- Selector requirements
- Confidence calculation

### Action Executor Registry

`action-executor.ts` centralizes replay execution:
- Each action type registers its executor via `registerActionExecutor()`
- New action types don't require modifying `controller.ts`
- Executors receive context with page, timeout, and validator

### Error Type Registry

`utils/errors.ts` centralizes error definitions:
- Error codes are stable contracts
- New errors extend `PlaywrightDriverError`
- Each error has semantic `FailureKind`

## Test Coverage for Change Axes

| Change Axis | Test File |
|-------------|-----------|
| Handler Registration | `tests/unit/handlers/registry.test.ts` |
| Recording Actions | `tests/unit/recording/action-types.test.ts` |
| Recording Controller | `tests/unit/recording/controller.test.ts` |
| Error Normalization | `tests/unit/utils/errors.test.ts` |
| Session Lifecycle | `tests/unit/session/manager.test.ts` |
| Circuit Breaker | `src/infra/circuit-breaker.test.ts` |
| Browser Manager | `src/session/browser-manager.test.ts` |

## Architecture Decisions

### Why Handlers are Self-Describing

Handlers declare their supported types via `getSupportedTypes()` rather than external configuration. This:
- Keeps handler logic co-located with type declaration
- Enables handlers to support multiple related types
- Simplifies registration (just instantiate and register)

### Why Selectors Have Reference Implementation

`selectors.reference.ts` (TypeScript) and `injector.ts` (stringified JS) exist because:
- Browser context requires plain JavaScript injection
- TypeScript version is easier to read and understand than stringified JS
- `injector.ts` is the SOURCE OF TRUTH - it runs at runtime
- `selectors.reference.ts` is documentation-only and NOT imported anywhere
- The reference file may drift over time - that's acceptable

### Why Outcomes Have Two Formats

- `StepOutcome`: Canonical format with nested objects (internal use, storage)
- `DriverOutcome`: Flattened format for Go parsing (wire format)

The Go API expects flattened fields (`screenshot_base64` not `screenshot.base64`) for simpler struct parsing.

## Future Improvements

1. ~~**Auto-generate injector.ts**~~ - Decided against: `injector.ts` is source of truth, `selectors.reference.ts` is documentation
2. ~~**Action executor registry**~~ - ✅ Implemented in `recording/action-executor.ts`
3. **Route grouping** - Group related routes (session/*, record/*) for cleaner registration
4. **Telemetry pipeline** - Create composable collector pipeline for future types
5. ~~**Selector strategy verification**~~ - No longer needed: reference file drift is acceptable
6. ~~**Extract circuit breaker**~~ - ✅ Implemented in `infra/circuit-breaker.ts`
7. ~~**Extract frame streaming**~~ - ✅ Implemented in `frame-streaming/`
8. ~~**Extract browser manager**~~ - ✅ Implemented in `session/browser-manager.ts`
9. **Unify result types** - HandlerResult, ActionReplayResult, and HandlerAdapterResult have similar shapes
10. **Extract idempotency cache** - Currently inline in `routes/session-run.ts`
