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
├── handlers/       # VOLATILE (by design) - Instruction execution
├── recording/      # VOLATILE (by design) - Record mode feature
├── telemetry/      # VOLATILE EDGE - Observability data capture
├── routes/         # STABLE CORE + VOLATILE EDGE
├── session/        # STABLE CORE - Session lifecycle
├── outcome/        # STABLE CONTRACT - Wire format transformation
├── types/          # Mixed - See types/index.ts for details
├── utils/          # STABLE CORE - Infrastructure concerns
├── config.ts       # VOLATILE EDGE - Configuration options
├── constants.ts    # STABLE CORE - Hardcoded defaults
└── server.ts       # STABLE CORE - HTTP server setup
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

**Primary extension points:**
- `recording/selectors.ts` - Reference implementation (documentation)
- `recording/injector.ts` - Runtime implementation (browser-injected)

**Important:** These files must stay in sync. `selectors.ts` uses TypeScript with DOM types for documentation; `injector.ts` contains the actual stringified JavaScript injected into pages.

When adding a new selector strategy:

1. Add `SelectorType` to `recording/types.ts`
2. Implement in `selectors.ts` (for documentation/reference)
3. Mirror implementation in `injector.ts` (actual runtime code)
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

### 5. Wire Format (LOW FREQUENCY, HIGH RISK)

**Primary extension point:** `types/contracts.ts` + `outcome/outcome-builder.ts`

**CRITICAL:** Changes here affect the Go API contract.

- Adding fields is safe (Go uses `omitempty`)
- Removing/renaming fields is a **BREAKING CHANGE**
- Coordinate with Go API team before changes

## Shotgun Surgery Mitigation

The following patterns reduce multi-file changes:

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

## Architecture Decisions

### Why Handlers are Self-Describing

Handlers declare their supported types via `getSupportedTypes()` rather than external configuration. This:
- Keeps handler logic co-located with type declaration
- Enables handlers to support multiple related types
- Simplifies registration (just instantiate and register)

### Why Selectors Have Dual Implementation

`selectors.ts` (TypeScript) and `injector.ts` (stringified JS) exist because:
- Browser context requires plain JavaScript injection
- TypeScript version provides documentation and type safety
- Algorithm changes should be made in both files

Consider generating `injector.ts` from `selectors.ts` in the future.

### Why Outcomes Have Two Formats

- `StepOutcome`: Canonical format with nested objects (internal use, storage)
- `DriverOutcome`: Flattened format for Go parsing (wire format)

The Go API expects flattened fields (`screenshot_base64` not `screenshot.base64`) for simpler struct parsing.

## Future Improvements

1. **Auto-generate injector.ts** - Compile `selectors.ts` to browser-compatible JS
2. ~~**Action executor registry**~~ - ✅ Implemented in `recording/action-executor.ts`
3. **Route grouping** - Group related routes (session/*, record/*) for cleaner registration
4. **Telemetry pipeline** - Create composable collector pipeline for future types
5. **Selector strategy verification** - Add test to verify `selectors.ts` and `injector.ts` are in sync
