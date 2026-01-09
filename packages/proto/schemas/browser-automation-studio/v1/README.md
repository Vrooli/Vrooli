# Browser Automation Studio Proto Schemas

This directory contains Protocol Buffer schemas for Browser Automation Studio (BAS).
The schemas follow a **layered architecture** organized by domain, where each layer
can only import from lower layers.

## Directory Structure

```
v1/
├── base/           Layer 0: Fundamental types (no BAS imports)
│   ├── shared.proto        Enums, RetryStatus, AssertionResult, EventContext
│   └── geometry.proto      BoundingBox, Point, NodePosition
│
├── domain/         Layer 1: Domain primitives (imports base/)
│   ├── selectors.proto     SelectorCandidate, ElementMeta, HighlightRegion
│   └── telemetry.proto     ConsoleLogEntry, NetworkEvent, ActionTelemetry
│
├── actions/        Layer 2: Action definitions (imports base/, domain/)
│   └── action.proto        ActionType, *Params, ActionDefinition
│
├── workflows/      Layer 2: Workflow definitions (imports base/, actions/)
│   └── definition.proto    WorkflowDefinitionV2, WorkflowNodeV2, WorkflowEdgeV2
│
├── timeline/       Layers 3-4: Execution history (imports layers 0-2)
│   ├── entry.proto         TimelineEntry (unified streaming + batch format)
│   └── container.proto     ExecutionTimeline (batch container)
│
├── recording/      Layer 3: Recording sessions (imports actions/, timeline/)
│   └── session.proto       RecordingState, session lifecycle messages
│
├── execution/      Layer 4: Execution runtime (imports workflows/, domain/)
│   └── execution.proto     Execution, ExecutionResult, ExecutionParameters
│
├── api/            Layer 5: Service definitions (imports all layers)
│   └── service.proto       WorkflowService gRPC, CRUD request/response types
│
└── projects/       Layer 5: Project organization (no BAS imports)
    └── project.proto       Project, ProjectStats
```

## Layer Dependency Graph

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Layer 5: api/, projects/                                                     │
│   ├── api/service.proto (WorkflowService, CRUD operations)                   │
│   └── projects/project.proto (Project, ProjectStats)                         │
├─────────────────────────────────────────────────────────────────────────────┤
│ Layer 4: execution/                                                          │
│   └── execution/execution.proto (Execution, ExecutionResult)                 │
├─────────────────────────────────────────────────────────────────────────────┤
│ Layer 3: timeline/, recording/                                               │
│   ├── timeline/entry.proto (TimelineEntry - unified format)                  │
│   ├── timeline/container.proto (ExecutionTimeline)                           │
│   └── recording/session.proto (RecordingState, session lifecycle)            │
├─────────────────────────────────────────────────────────────────────────────┤
│ Layer 2: actions/, workflows/                                                │
│   ├── actions/action.proto (ActionDefinition, ActionType, *Params)           │
│   └── workflows/definition.proto (WorkflowDefinitionV2, nodes, edges)        │
├─────────────────────────────────────────────────────────────────────────────┤
│ Layer 1: domain/                                                             │
│   ├── domain/selectors.proto (SelectorCandidate, ElementMeta)                │
│   └── domain/telemetry.proto (ActionTelemetry, TimelineScreenshot)           │
├─────────────────────────────────────────────────────────────────────────────┤
│ Layer 0: base/                                                               │
│   ├── base/shared.proto (enums, RetryStatus, EventContext)                   │
│   └── base/geometry.proto (BoundingBox, Point, NodePosition)                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Types by Domain

| Domain | Key Types | Purpose |
|--------|-----------|---------|
| `base/` | ExecutionStatus, StepStatus, EventContext, RetryStatus | Shared enums and status tracking |
| `domain/` | SelectorCandidate, ActionTelemetry, TimelineScreenshot | Element targeting and observability |
| `actions/` | ActionDefinition, ActionType, *Params | Unified action representation |
| `workflows/` | WorkflowDefinitionV2, WorkflowNodeV2, WorkflowEdgeV2 | Workflow storage format |
| `timeline/` | TimelineEntry, ExecutionTimeline | Execution history (streaming + batch) |
| `recording/` | RecordingState, *Request/*Response | Recording session lifecycle |
| `execution/` | Execution, ExecutionResult, ExecutionParameters | Runtime execution state |
| `api/` | WorkflowService, WorkflowSummary, CRUD messages | gRPC service definitions |
| `projects/` | Project, ProjectStats | Project organization |

## Unified Recording/Execution Model

BAS treats recording and execution as the **same operation** with different controllers.
Both modes capture identical data via `TimelineEntry`:

- **Recording**: User-driven actions captured live → `TimelineEntry` stream
- **Execution**: Workflow-driven actions replayed → `TimelineEntry` stream

This unified model enables:
- Live preview using the same UI during both recording and playback
- Fallback selectors captured during recording, used during execution
- Full debugging visibility regardless of how actions were triggered

See `base/shared.proto` for the detailed design rationale.

## Go Package Mapping

| Proto Directory | Go Package |
|-----------------|------------|
| `base/` | `github.com/.../v1/base` (alias: `basbase`) |
| `domain/` | `github.com/.../v1/domain` (alias: `basdomain`) |
| `actions/` | `github.com/.../v1/actions` (alias: `basactions`) |
| `workflows/` | `github.com/.../v1/workflows` (alias: `basworkflows`) |
| `timeline/` | `github.com/.../v1/timeline` (alias: `bastimeline`) |
| `recording/` | `github.com/.../v1/recording` (alias: `basrecording`) |
| `execution/` | `github.com/.../v1/execution` (alias: `basexecution`) |
| `api/` | `github.com/.../v1/api` (alias: `basapi`) |
| `projects/` | `github.com/.../v1/projects` (alias: `basprojects`) |

## TypeScript Import Paths

```typescript
import { ActionType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { ExecutionStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { TimelineEntry } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import { WorkflowDefinitionV2 } from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
```

## Adding New Types

When adding new types:

1. **Identify the appropriate layer** based on dependencies
2. **Place in the correct directory** matching that layer
3. **Update imports** to only reference lower layers
4. **Run `make generate`** from `packages/proto/`
5. **Run `make lint`** to verify

Example: A new "assertion" type that uses `ActionDefinition` would go in Layer 3+
since it depends on Layer 2 (`actions/`).
