# Vrooli Proto Contracts

This package hosts Protocol Buffers schemas for inter-scenario contracts and their generated artifacts (Go, TypeScript, Python). Source files live under `schemas/`; generated code lives under `gen/` and is committed.

## Quickstart

- Edit `.proto` files in `schemas/`.
- Regenerate: `cd packages/proto && make generate`
- Lint: `cd packages/proto && make lint`
- Breaking check (against `master` by default): `cd packages/proto && make breaking`
- Keep `gen/` in sync with `schemas/` before committing.
- JSON serialization: BAS endpoints use snake_case; when marshaling with Go's `protojson` set `UseProtoNames: true` (and the equivalent options in TS/Python) so you don't emit lowerCamel JSON.

## Type-safety guidance

- Prefer typed fields over raw `Struct` payloads wherever available. For BAS, use `metadata_typed` / `settings_typed` instead of the deprecated maps, and only rely on map/Struct fields for provider-specific extensions that are truly dynamic.
- `JsonValue`/`JsonObject`/`JsonList` now live in `common.v1` so all scenarios can share the same typed JSON primitives; migrate imports accordingly.
- Optional scalars (`optional string ...`, `optional int64 ...`) are present throughout the BAS timeline/billing contracts to distinguish “unset” from zero values. In generated code, check presence (e.g., `HasError` in Go or truthy `error` plus `hasError()` in TS) instead of assuming defaults.
- `JsonValue` supports explicit nulls and raw bytes; use these when round-tripping JSON that contains `null` or binary blobs instead of dropping intent or coercing to zero values.
- The `@vrooli/proto-types` package is ESM-only; CJS consumers should `import()` dynamically or transpile to ESM when wiring into Jest/older Node runtimes.

## JSON casing & compatibility

- BAS APIs/WebSockets and landing-page APIs expect snake_case JSON. Use proto-name casing when marshaling/unmarshaling:
  - Go: `protojson.MarshalOptions{UseProtoNames: true}`, `protojson.UnmarshalOptions{DiscardUnknown: true}`
  - TypeScript: `{ jsonOptions: { useProtoNames: true } }` with `fromJson` / `toJsonString`
  - Python: `json_format.ParseDict(..., preserve_proto_field_name=True)`
- `Execution.execution_id` keeps `json_name = "id"` for compatibility with existing BAS responses.

## Usage examples

### TypeScript
```ts
import { fromJson, toJsonString } from '@bufbuild/protobuf';
import { ExecutionTimelineSchema } from '@vrooli/proto-types/browser-automation-studio/v1/timeline_pb';

const timeline = fromJson(ExecutionTimelineSchema, apiPayload, {
  jsonOptions: { useProtoNames: true },
});

const serialized = toJsonString(ExecutionTimelineSchema, timeline, {
  jsonOptions: { useProtoNames: true },
});
```

### TypeScript (landing-page)
```ts
import { fromJson } from '@bufbuild/protobuf';
import { GetPricingResponseSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/pricing_pb';

const pricing = fromJson(GetPricingResponseSchema, payload, {
  jsonOptions: { useProtoNames: true },
});

const monthlyPlanMetadata = pricing.pricing?.monthly[0]?.metadataTyped;
```

### Go
```go
jsonOpts := protojson.MarshalOptions{UseProtoNames: true}
payload, _ := jsonOpts.Marshal(timeline)

var parsed basv1.ExecutionTimeline
if err := protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(payload, &parsed); err != nil {
	// handle error
}
```

### Python
```py
from google.protobuf import json_format
from browser_automation_studio.v1 import timeline_pb2

parsed = json_format.ParseDict(
    data,
    timeline_pb2.ExecutionTimeline(),
    ignore_unknown_fields=True,
    preserve_proto_field_name=True,
)
```

## Workspace usage

- Go: use the repo `go.work` so scenarios can import `github.com/vrooli/vrooli/packages/proto/gen/go/...`. Go package names mirror scenario slugs with underscores, e.g. `browser_automation_studio_v1` and `landing_page_react_vite_v1`.
- TypeScript/JavaScript: `@vrooli/proto-types` is published from `gen/typescript` (ESM JS lives under `js/`) and linked via the pnpm workspace.
- Python: install `packages/proto/gen/python` in editable mode for local development.

## BAS schema notes

The BAS proto files follow a layered import hierarchy (documented in `shared.proto`):

- **Layer 0 (Base):** `shared.proto` (enums, RetryStatus), `geometry.proto` (BoundingBox, Point)
- **Layer 1:** `selectors.proto`, `telemetry.proto`
- **Layer 2:** `action.proto` (ActionDefinition), `workflow_v2.proto` (WorkflowDefinitionV2)
- **Layer 3:** `timeline.proto` (batch), `timeline_event.proto` (streaming), `recording_session.proto`
- **Layer 4:** `execution.proto`, `workflow_service.proto`
- **Layer 5:** `project.proto`

Key types:
- `ActionDefinition` (action.proto): Unified action type for recording, workflows, and execution
- `TimelineEvent` (timeline_event.proto): Streaming format for real-time events
- `TimelineFrame` (timeline.proto): Batch format for completed executions
- `WorkflowDefinitionV2` (workflow_v2.proto): Canonical workflow storage format
- `recording_session.proto`: Session management (create, start, stop, get actions)

## Landing-page schema notes

- Pricing/billing metadata maps now include `metadata_typed` so clients can avoid `google.protobuf.Value` and stay type-safe across pricing, sessions, subscription status, and wallet transactions.
- Use `common.v1.JsonValue` for flexible-yet-typed metadata instead of raw `Struct`/`Value`.
