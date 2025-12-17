# Playwright Driver - Architectural Seams

This document defines the **architectural boundaries** (seams) within the playwright-driver package. A "seam" is a place where you can alter behavior without editing existing code - understanding these boundaries helps you know where to make changes.

## Module Boundaries

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          PLAYWRIGHT-DRIVER                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌────────────────┐    ┌────────────────┐    ┌────────────────┐            │
│  │     PROTO      │    │    HANDLERS    │    │    OUTCOME     │            │
│  │  (Wire Format) │───▶│  (Execution)   │───▶│  (Results)     │            │
│  └────────┬───────┘    └────────────────┘    └────────────────┘            │
│           │                    │                     ▲                      │
│           │                    │                     │                      │
│           ▼                    ▼                     │                      │
│  ┌────────────────┐    ┌────────────────┐    ┌──────┴─────────┐            │
│  │   RECORDING    │    │    SESSION     │    │     UTILS      │            │
│  │ (Capture/Play) │    │  (Lifecycle)   │    │ (Cross-cutting)│            │
│  └────────────────┘    └────────────────┘    └────────────────┘            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Seam 1: Proto Module

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

## Seam 2: Handlers Module

**Location**: `src/handlers/`

**Responsibility**: Execute browser automation instructions

### Files and Their Boundaries

| File | Boundary | Can Change | Cannot Change |
|------|----------|------------|---------------|
| `base.ts` | Handler contract | Utility methods | `InstructionHandler` interface |
| `registry.ts` | Handler dispatch | Lookup logic | Handler registration pattern |
| `*.ts` (handlers) | Individual execution | Implementation | Return type (`HandlerResult`) |

### Change Axes

1. **Adding a new instruction type**
   - Create new handler file
   - Implement `InstructionHandler`
   - Register in `server.ts`
   - Add param extractor in `proto/params.ts`

2. **Modifying handler behavior**
   - Edit specific handler file
   - No other files affected

## Seam 3: Recording Module

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

## Seam 4: Outcome Module

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

**Note**: Action execution has a single source of truth in handlers. The action-executor delegates to handlers via handler-adapter.ts, eliminating code duplication.

### 2. Import Direction

```
proto/ ◄───────────────────────────────────────────┐
   │                                               │
   ▼                                               │
handlers/ ──────────────────────────────────▶ outcome/
   │                                               ▲
   │                                               │
   ▼                                               │
recording/ ────────────────────────────────────────┘
   │
   └──▶ handler-adapter.ts ──▶ handlers/
```

- `proto/` is the foundation (imported by all)
- `recording/` may import from `proto/` via the safe path in `action-types.ts`
- `outcome/` can be imported by handlers and recording
- `handler-adapter.ts` bridges recording to handlers for replay execution

### 3. Circular Import Prevention

The `recording/action-types.ts` file imports directly from `@vrooli/proto-types` rather than `../proto` to avoid circular imports. This pattern must be maintained.

## Testing Boundaries

| Module | Test Type | Location |
|--------|-----------|----------|
| Proto | Unit | `tests/unit/proto/` |
| Handlers | Integration | `tests/integration/handlers/` |
| Recording | Unit + Integration | `tests/unit/recording/` |
| Outcome | Unit | `tests/unit/outcome/` |

## Evolution Guidelines

### Safe Changes (Won't Break Other Code)

- Adding new handler files
- Adding new param extractors
- Adding new action type mappings
- Adding new error codes
- Internal refactoring within a module

### Dangerous Changes (May Break Other Code)

- Modifying `HandlerResult` interface
- Modifying `TimelineEntry` structure
- Renaming exported functions
- Changing handler registration pattern
- Modifying the `ACTION_TYPE_MAP` key format

### Requires Coordination

- Wire format changes (Go API must change first)
- New instruction types (proto schema + param extractor + handler)
- Recording format changes (affects UI and API)
