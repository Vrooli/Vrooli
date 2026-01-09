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
**Files affected:** 1-2 (handler + registration)
**Risk:** Low - isolated changes

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

### 2. Anti-Detection Patches (MEDIUM-HIGH FREQUENCY)

**Primary extension point:** `browser-profile/patches.ts`
**Files affected:** 1 (patches.ts only)
**Risk:** Low - patches are isolated

When adding a new anti-detection technique:

1. Create patch generator function in `patches.ts`:
   ```typescript
   export function generateMyPatch(): string {
     return `// JavaScript to inject into browser context`;
   }
   ```
2. Add entry to `PATCH_REGISTRY` with enable condition
3. Test the patch in `tests/unit/browser-profile/patches.test.ts`

**Benefits of this pattern:**
- Each patch is independently testable
- Easy to enable/disable patches per-site
- No shotgun surgery to anti-detection.ts

### 3. Recording Action Types (MEDIUM FREQUENCY)

**Primary extension points:**
- `recording/action-types.ts` - Type normalization, kind mapping, confidence calculation
- `recording/action-executor.ts` - Replay execution for each action type
**Files affected:** 2-3
**Risk:** Low - registry pattern

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

### 4. Selector Strategies (MEDIUM FREQUENCY)

**Primary extension point:** `recording/injector.ts` - Runtime implementation (browser-injected)

The selector generation code lives in `injector.ts` as stringified JavaScript that gets injected
into browser pages via `page.evaluate()`. Browser context requires plain JS, so the code is
stored as a string template.

When adding a new selector strategy:

1. Add `SelectorType` to `recording/types.ts`
2. Implement in `injector.ts` (the runtime code)
3. Update `DEFAULT_SELECTOR_OPTIONS` if needed
4. Add configuration to `selector-config.ts` if the strategy has tunable parameters

### 5. Telemetry Types (LOW-MEDIUM FREQUENCY)

**Primary extension point:** `telemetry/`

When adding new observability data:

1. Create collector file: `telemetry/my-collector.ts`
2. Define types in `types/contracts.ts` (coordinate with Go API)
3. Add to `StepOutcome` interface if needed
4. Update `outcome/outcome-builder.ts` to include in output
5. Add config options in `config.ts`
6. Use collector in `routes/session-run.ts`

### 6. Frame Streaming (LOW-MEDIUM FREQUENCY)

**Primary extension point:** `frame-streaming/`

The frame streaming module handles live video streaming to the UI during recording.
Uses a **strategy pattern** for different streaming implementations:

- **CDP Screencast** (preferred): Push-based streaming from Chrome compositor (30-60 FPS)
- **Polling Fallback**: Pull-based screenshot capture with adaptive FPS

**Architecture:**
```
frame-streaming/
├── manager.ts              # Thin orchestrator (~270 lines)
├── types.ts                # Shared types and constants
├── strategies/
│   ├── interface.ts        # FrameStreamingStrategy interface
│   ├── cdp-screencast.ts   # CDP screencast implementation
│   └── polling.ts          # Polling fallback implementation
└── websocket/
    └── connection.ts       # WebSocket lifecycle management
```

When adding a new streaming strategy:
1. Implement `FrameStreamingStrategy` interface from `strategies/interface.ts`
2. Add strategy selection logic in `manager.ts:startWithStrategy()`
3. Export from `strategies/index.ts`

When modifying frame streaming:
1. Configuration options are in `config.ts` under `frameStreaming`
2. FPS control is handled by `fps/controller.ts`
3. Performance metrics are collected by `performance/collector.ts`

### 7. Session Lifecycle (LOW FREQUENCY)

**Primary extension point:** `session/session-decisions.ts`, `session/state-machine.ts`
**Files affected:** 2-3
**Risk:** Medium - affects session behavior

Session lifecycle changes are now well-isolated:

1. **Decision logic**: Add/modify functions in `session-decisions.ts`
2. **Phase transitions**: Modify `state-machine.ts` for new phases
3. **Manager orchestration**: `manager.ts` calls decision functions

This separation means:
- Reuse logic changes don't touch state machine
- Phase changes don't affect reuse decisions
- Both are independently testable

### 8. Artifact Configuration (LOW FREQUENCY)

**Primary extension point:** `session/artifact-paths.ts`
**Files affected:** 1
**Risk:** Low - isolated path resolution

When adding a new artifact type (like a new recording format):

1. Add resolver function in `artifact-paths.ts` following existing patterns
2. Add capability check function (e.g., `shouldRecordNewArtifact()`)
3. Update `resolveArtifactPaths()` to include new artifact
4. Use in `context-builder.ts`

### 9. Wire Format (LOW FREQUENCY, HIGH RISK)

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

### Why Selectors are Stringified JavaScript

`injector.ts` contains stringified JavaScript for selector generation because:
- Browser context requires plain JavaScript injection (no TypeScript)
- The code must be injected via `page.evaluate()` which requires a serializable function
- Configuration values from `selector-config.ts` are shared between the injector and other modules

### Why Outcomes Have Two Formats

- `StepOutcome`: Canonical format with nested objects (internal use, storage)
- `DriverOutcome`: Flattened format for Go parsing (wire format)

The Go API expects flattened fields (`screenshot_base64` not `screenshot.base64`) for simpler struct parsing.

## Decision Boundaries

Explicit decision points that were previously embedded in complex conditionals have been
extracted into named, testable functions. This follows the Decision Boundary Extraction pattern.

### Session Decisions (`session/session-decisions.ts`)

| Decision | Function | Description |
|----------|----------|-------------|
| Session reuse | `shouldAttemptReuse()` | Whether to look for reusable sessions |
| Reuse behavior | `makeReuseDecision()` | Whether to reset, recover phase, etc. |
| Session lookup | `findByExecutionId()`, `findByLabels()` | Find sessions by different criteria |
| Idle cleanup | `findIdleSessions()` | Which sessions should be cleaned up |

### Artifact Path Decisions (`session/artifact-paths.ts`)

| Decision | Function | Description |
|----------|----------|-------------|
| HAR path | `resolveHarPath()` | 3-tier fallback: explicit > root-derived > temp |
| Video dir | `resolveVideoDir()` | 3-tier fallback with directory semantics |
| Trace path | `resolveTracePath()` | 3-tier fallback for trace files |
| Recording enabled | `shouldRecordHar/Video/Tracing()` | Check capabilities |

### Anti-Detection Patches (`browser-profile/patches.ts`)

| Decision | Registry Entry | Description |
|----------|----------------|-------------|
| Patch selection | `PATCH_REGISTRY` | Which patches to apply based on settings |
| Patch composition | `composePatches()` | Combine enabled patches into script |

Benefits of this pattern:
- **Testable**: Each decision function can be unit tested
- **Discoverable**: Find "where we decide X" by searching function names
- **Documentable**: Decisions have JSDoc explaining the "why"
- **Auditable**: Easy to review all decision points in one place

## Future Improvements

1. ~~**Auto-generate injector.ts**~~ - Not needed: injector.ts is self-contained and readable
2. ~~**Action executor registry**~~ - ✅ Implemented in `recording/action-executor.ts`
3. **Route grouping** - Group related routes (session/*, record/*) for cleaner registration
4. **Telemetry pipeline** - Create composable collector pipeline for future types
5. ~~**Selector strategy verification**~~ - Not needed: single source of truth in injector.ts
6. ~~**Extract circuit breaker**~~ - ✅ Implemented in `infra/circuit-breaker.ts`
7. ~~**Extract frame streaming**~~ - ✅ Implemented in `frame-streaming/`
8. ~~**Extract browser manager**~~ - ✅ Implemented in `session/browser-manager.ts`
9. ~~**Unify result types**~~ - ✅ All result types now extend BaseExecutionResult (outcome/types.ts)
10. ~~**Extract idempotency cache**~~ - ✅ Implemented in `infra/idempotency-cache.ts`
11. ~~**Extract anti-detection patches**~~ - ✅ Implemented in `browser-profile/patches.ts`
12. ~~**Extract artifact path decisions**~~ - ✅ Implemented in `session/artifact-paths.ts`
13. ~~**Extract session decisions**~~ - ✅ Implemented in `session/session-decisions.ts`
