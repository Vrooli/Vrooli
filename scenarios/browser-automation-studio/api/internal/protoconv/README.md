# protoconv - Proto Conversion Utilities

This package converts between Go types and proto-generated types for the browser-automation-studio API.

## Package Architecture

```
internal/
├── enums/           ← NEW: Standalone enum conversions (no dependencies)
│   └── enums.go     ← All string↔proto enum functions
├── protoconv/       ← Proto object conversions (depends on workflow, export, etc.)
│   ├── enum_convert.go   ← Re-exports from internal/enums for backward compat
│   ├── convert.go        ← Execution, timeline, project conversions
│   ├── driver_convert.go ← StepOutcome, Screenshot, DOM conversions
│   ├── workflows.go      ← Workflow definition conversions
│   └── README.md         ← This file
└── typeconv/        ← Primitive type conversions (depends on enums)
    ├── primitives.go     ← Any→int, Any→bool, JsonValue helpers
    └── builders.go       ← Proto parameter builders
```

## Enum Conversions

**IMPORTANT**: Enum conversions are now in `internal/enums` to avoid import cycles.

```go
// Preferred: Import enums directly
import "github.com/vrooli/browser-automation-studio/internal/enums"

actionType := enums.StringToActionType("click")
status := enums.StringToExecutionStatus("COMPLETED")

// Deprecated: These still work but delegate to enums internally
import "github.com/vrooli/browser-automation-studio/internal/protoconv"

actionType := protoconv.StringToActionType("click")  // → enums.StringToActionType
```

### Available Enum Converters

| Converter | Enum Type | Example Input |
|-----------|-----------|---------------|
| `StringToActionType` | `basactions.ActionType` | "click", "navigate", "type" |
| `StringToSelectorType` | `basbase.SelectorType` | "css", "xpath", "data-testid" |
| `StringToExecutionStatus` | `basbase.ExecutionStatus` | "PENDING", "COMPLETED" |
| `StringToAssertionMode` | `basbase.AssertionMode` | "exists", "visible" |
| `StringToLogLevel` | `basbase.LogLevel` | "DEBUG", "INFO", "ERROR" |
| `StringToMouseButton` | `basactions.MouseButton` | "left", "right", "middle" |
| `StringToKeyboardModifier` | `basactions.KeyboardModifier` | "ctrl", "shift", "meta" |
| `StringToNetworkEventType` | `basbase.NetworkEventType` | "request", "response" |
| `StringToTriggerType` | `basbase.TriggerType` | "manual", "scheduled" |
| `StringToChangeSource` | `basbase.ChangeSource` | "manual", "recording" |

## Proto Schema Overview

The browser-automation-studio uses proto definitions from `packages/proto/browser-automation-studio/v1/`:

```
browser-automation-studio/v1/
├── actions.proto       ← Action types and parameters
│   ├── ActionType enum
│   ├── ActionDefinition (typed params via oneof)
│   ├── ActionMetadata
│   └── *Params messages (ClickParams, NavigateParams, etc.)
│
├── base.proto          ← Core types and enums
│   ├── BoundingBox, Point (geometry)
│   ├── EventContext (unified recording/execution context)
│   ├── ExecutionStatus, StepStatus enums
│   ├── AssertionMode, AssertionResult
│   └── SelectorType, TriggerType, etc.
│
├── domain.proto        ← Domain-specific types
│   ├── ElementMeta (DOM element metadata)
│   ├── SelectorCandidate (selector with confidence)
│   ├── ActionTelemetry (screenshots, DOM, network)
│   ├── ConsoleLogEntry, NetworkEvent
│   └── HighlightRegion, MaskRegion
│
├── timeline.proto      ← Timeline/recording types
│   ├── TimelineEntry (unified action+telemetry+context)
│   ├── ExecutionTimeline (container)
│   └── TimelineScreenshot
│
├── execution.proto     ← Execution/driver types
│   ├── Execution (execution metadata)
│   ├── CompiledInstruction (execution plan unit)
│   ├── ExecutionPlan (compiled workflow)
│   ├── StepOutcome (proto version)
│   └── StepFailure, FailureKind, FailureSource
│
└── api.proto           ← API request/response messages
    └── Various *Request/*Response messages
```

## Key Data Flows

### Recording Path
```
Human Browser Action
    ↓
livecapture.RecordedAction (Go struct with live-capture types)
    ↓ [events/recording_convert.go]
bastimeline.TimelineEntry (unified proto format)
    ↓
WebSocket → UI Timeline
```

### Execution Path
```
Workflow Definition (V2 flow JSON)
    ↓ [compiler/plan_builder.go]
basexecution.ExecutionPlan (compiled instructions)
    ↓ [executor/simple_executor.go]
contracts.StepOutcome (native Go struct with time.Time)
    ↓ [events/unified_convert.go]
bastimeline.TimelineEntry (unified proto format)
    ↓
WebSocket → UI Timeline
```

### Unified TimelineEntry

Both paths converge on `bastimeline.TimelineEntry`:

```protobuf
message TimelineEntry {
  string id = 1;
  int32 sequence_num = 2;
  google.protobuf.Timestamp timestamp = 4;

  ActionDefinition action = 10;      // What action was performed
  ActionTelemetry telemetry = 11;    // Screenshots, DOM, network data
  EventContext context = 12;         // Origin (session/execution), success, errors
}
```

## Time Conversion

For timestamp conversions, use helpers from `automation/contracts/proto_time_helpers.go`:

```go
import "github.com/vrooli/browser-automation-studio/automation/contracts"

// Proto timestamp → Go time.Time
t := contracts.TimestampToTime(pb.StartedAt)
tPtr := contracts.TimestampToTimePtr(pb.CompletedAt)

// Go time.Time → Proto timestamp
pb.StartedAt = contracts.TimeToTimestamp(t)
pb.CompletedAt = contracts.TimePtrToTimestamp(tPtr)
```

## When to Use Each Package

| Package | Use For | Dependencies |
|---------|---------|--------------|
| `internal/enums` | String↔enum conversions | Proto types only |
| `internal/protoconv` | Complex proto object conversions | workflow, export, database |
| `internal/typeconv` | Primitive conversions, JsonValue | enums |
| `automation/events` | Timeline building, event streaming | enums, contracts |

## Migration Notes

This package works alongside native Go types in `automation/contracts/contracts.go`. The native types (with `time.Time` fields) are being gradually deprecated in favor of proto-native types. New code should:

1. Prefer proto types (`contracts.ProtoStepOutcome`) over native types (`contracts.StepOutcome`)
2. Use the time helpers from `contracts/proto_time_helpers.go`
3. Import `internal/enums` directly for enum conversions
4. Use this package for all proto object conversions rather than inline conversion code
