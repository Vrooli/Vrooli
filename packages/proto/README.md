# Vrooli Proto Contracts

This package hosts Protocol Buffers schemas for inter-scenario contracts and their generated artifacts (Go, TypeScript, Python). Source files live under `schemas/`; generated code lives under `gen/` and is committed.

> **Style Guide**: For detailed naming conventions, file headers, layer architecture, and documentation requirements, see [STYLE_GUIDE.md](./STYLE_GUIDE.md).

## Quickstart

- Edit `.proto` files in `schemas/`.
- Regenerate one schema change: `cd packages/proto && make generate SCENARIO=<scenario>`
- Deliberate full-fleet rebuild: `cd packages/proto && make generate`
- Lint: `cd packages/proto && make lint`
- Breaking check: `cd packages/proto && make breaking SCENARIO=<scenario>`
- Keep `gen/` in sync with `schemas/` before committing.
- JSON serialization: Vrooli HTTP JSON endpoints use proto field names (`snake_case`) on the wire. Go writers use `protojson.MarshalOptions{UseProtoNames: true}`; TypeScript writers use `toJsonString(..., { useProtoFieldName: true })`. TypeScript `fromJson` accepts both proto field names and JSON/lowerCamel names, so UI readers should parse through generated descriptors instead of manually reshaping payloads.

## Type-safety guidance

- Prefer typed fields over raw `Struct` payloads wherever available. For BAS, use `metadata_typed` / `settings_typed` instead of the deprecated maps, and only rely on map/Struct fields for provider-specific extensions that are truly dynamic.
- `JsonValue`/`JsonObject`/`JsonList` now live in `common.v1` so all scenarios can share the same typed JSON primitives; migrate imports accordingly.
- Optional scalars (`optional string ...`, `optional int64 ...`) are present throughout the BAS timeline/billing contracts to distinguish “unset” from zero values. In generated code, check presence (e.g., `HasError` in Go or truthy `error` plus `hasError()` in TS) instead of assuming defaults.
- `JsonValue` supports explicit nulls and raw bytes; use these when round-tripping JSON that contains `null` or binary blobs instead of dropping intent or coercing to zero values.
- The `@vrooli/proto-types` package is ESM-only; CJS consumers should `import()` dynamically or transpile to ESM when wiring into Jest/older Node runtimes.

## JSON casing & compatibility

- Vrooli APIs/WebSockets expect proto-name JSON (`snake_case`) for structured proto messages. Use generated descriptors and explicit writer options:
  - Go: `protojson.MarshalOptions{UseProtoNames: true}`, `protojson.UnmarshalOptions{DiscardUnknown: false}` for ingress that must reject unknown fields.
  - TypeScript read: `fromJson(MessageSchema, payload, { ignoreUnknownFields: true })`
  - TypeScript write: `toJsonString(MessageSchema, message, { useProtoFieldName: true })`
  - Python: `json_format.ParseDict(..., preserve_proto_field_name=True)`
- Multipart/form-data and raw file uploads are transport exceptions: keep file bytes in multipart parts or binary streams, and encode any structured metadata part with proto JSON.
- `Execution.execution_id` keeps `json_name = "id"` for compatibility with existing BAS responses.

## Usage examples

### TypeScript
```ts
import { fromJson, toJsonString } from '@bufbuild/protobuf';
import { ExecutionTimelineSchema } from '@vrooli/proto-types/browser-automation-studio/v1/timeline_pb';

const timeline = fromJson(ExecutionTimelineSchema, apiPayload, {
  ignoreUnknownFields: true,
});

const serialized = toJsonString(ExecutionTimelineSchema, timeline, {
  useProtoFieldName: true,
});
```

### TypeScript (landing-page)
```ts
import { fromJson } from '@bufbuild/protobuf';
import { GetPricingResponseSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/pricing_pb';

const pricing = fromJson(GetPricingResponseSchema, payload, {
  ignoreUnknownFields: true,
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

## Adoption

- Go: scenarios that import `github.com/vrooli/vrooli/packages/proto/gen/go/...` must add a local `replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto` (path adjusted per module) so proto adoption stays explicit and workspace-independent. Go package names mirror scenario slugs with underscores, e.g. `browser_automation_studio_v1` and `landing_page_react_vite_v1`.
- TypeScript/JavaScript: `@vrooli/proto-types` is published from `gen/typescript` (ESM JS lives under `js/`) and adopted by isolated consumers via governed local `file:` dependencies rather than a shared workspace.
- Python: generated under `packages/proto/gen/python`; Tier 1 import support is not guaranteed yet (Protovalidate annotations require additional Python packaging), but you can install this directory in editable mode for local experiments.

## Package Governance

`proto` is a governed schema/contract package that owns its generated outputs, including `@vrooli/proto-types`.

- Scenario-adoptable: yes
- Allowed consumer classes: `scenario_ui`, `template_ui`, `scenario_test`, `scenario_api`, `scenario_cli`, `template_api`, `template_cli`
- Supported adoption modes: `file_dependency`, `go_module_replace`, `generated_artifact`
- Refresh strategy: generate first, then apply consumer-specific refresh behavior

Canonical operator flow:

```bash
vrooli package info proto
vrooli package generate proto
vrooli package dependents proto
vrooli package refresh proto all --no-restart
```

`make generate` remains useful for deliberate full-fleet rebuilds when working
directly inside `packages/proto`, but a routine schema edit should use
`vrooli package generate --scenario <scenario>` or the equivalent
`make generate SCENARIO=<scenario>`. Generation is staged and advisory-locked;
unchanged outputs are not rewritten. Scoped runs include reverse dependents and
shared imports. See [docs/package-governance.md](/home/matthalloran8/Vrooli/docs/package-governance.md:1)
for the canonical policy.

## Language support matrix

This repo generates Protobuf code for multiple languages, but “generated” does not necessarily mean “supported”.

Definitions:
- Tier 0 (Generate): Code is generated in `packages/proto/gen/<lang>/`.
- Tier 1 (Import): Downstream projects can import the generated code without extra, undocumented dependencies.
- Tier 2 (Validate): We run Protovalidate in this language at defined boundaries (ingress/compile/runtime).

| Language | Tier 0 | Tier 1 | Tier 2 | Notes |
|---|:---:|:---:|:---:|---|
| Go | Yes | Yes | Partial/Planned | Protovalidate annotations may require adding protovalidate Go deps in consumers. |
| TypeScript | Yes | Yes | Partial/Planned | Runtime validation uses `@bufbuild/protovalidate` in validating services only. |
| Python | Yes | No (today) | No | If any schema imports `buf/validate/validate.proto`, generated Python imports `buf.validate.validate_pb2`, which is not currently shipped in `gen/python`. Python Tier 1 requires either vendoring/generating Protovalidate protos into `gen/python` or depending on a Protovalidate Python SDK that provides them. |

## BAS schema notes

The BAS proto files follow a layered import hierarchy (documented in each file's header via `@layer`, `@domain`, `@imports` annotations):

- **Layer 0 (Base):** `base/shared.proto` (enums, RetryStatus, EventContext), `base/geometry.proto` (BoundingBox, Point, NodePosition)
- **Layer 1 (Domain):** `domain/selectors.proto` (SelectorCandidate, ElementMeta), `domain/telemetry.proto` (ActionTelemetry, TimelineScreenshot)
- **Layer 2 (Actions/Workflows):** `actions/action.proto` (ActionDefinition), `workflows/definition.proto` (WorkflowDefinitionV2)
- **Layer 3 (Timeline/Recording):** `timeline/entry.proto` (TimelineEntry - unified format), `recording/session.proto`
- **Layer 4 (Execution/Timeline Container):** `execution/execution.proto`, `timeline/container.proto` (ExecutionTimeline)
- **Layer 5 (API/Projects):** `api/service.proto` (WorkflowService gRPC), `projects/project.proto`

Key types:
- `ActionDefinition` (actions/action.proto): Unified action type for recording, workflows, and execution
- `TimelineEntry` (timeline/entry.proto): **Unified format** for both streaming and batch timeline data
- `EventContext` (base/shared.proto): Unified origin/outcome context for recording and execution
- `ExecutionTimeline` (timeline/container.proto): Container for batch timeline retrieval
- `WorkflowDefinitionV2` (workflows/definition.proto): Canonical workflow storage format
- `recording/session.proto`: Session management (create, start, stop, get actions)

## Landing-page schema notes

- Pricing/billing metadata maps now include `metadata_typed` so clients can avoid `google.protobuf.Value` and stay type-safe across pricing, sessions, subscription status, and wallet transactions.
- Use `common.v1.JsonValue` for flexible-yet-typed metadata instead of raw `Struct`/`Value`.

## Adding a new scenario

When creating proto definitions for a new scenario:

### 1. Create the directory structure

```bash
mkdir -p schemas/<scenario-name>/v1
```

### 2. Create your proto files

Follow the layer hierarchy (see [STYLE_GUIDE.md](./STYLE_GUIDE.md) for details):

| Layer | Purpose | Examples |
|-------|---------|----------|
| 0 | Base types, enums | `base/shared.proto` |
| 1 | Domain models | `domain/models.proto` |
| 2 | Actions, workflows | `actions/action.proto` |
| 3 | Aggregates, sessions | `recording/session.proto` |
| 4 | Runtime state | `execution/execution.proto` |
| 5 | API services, projects | `api/service.proto` |

Each file should include a header with `@layer`, `@domain`, `@imports`, and `@stability` annotations.

### 3. Regenerate code

```bash
cd packages/proto && make generate SCENARIO=<scenario>
```

This stages and regenerates all language outputs (Go, TypeScript, Python),
creates `py.typed` markers, rebuilds the descriptor image, and writes
per-scenario generation manifests under `gen/manifests/` before publishing only
changed outputs. Use `go run -mod=mod ./cmd/protogen generate --changed` for
the routine dependency-aware scoped run, or `--scenario` for an explicit scope.

### 4. Verify your changes

```bash
# Lint your proto files
make lint

# Check for breaking changes (against master)
make breaking SCENARIO=<scenario>

# Ensure committed generated code, descriptors, and manifests are in sync
make verify-committed-gen
```

### 5. Import in your scenario

- **Go**: Add the required local `replace` directive to point at `packages/proto`
- **TypeScript**: Import from `@vrooli/proto-types/<scenario-name>/v1/<file>_pb`
- **Python**: Import from `<scenario_name>.v1.<file>_pb2`

## Protovalidate

Protovalidate provides **declarative validation rules** directly in your proto definitions. This complements client-side validation (like Zod) by ensuring validation rules are defined once and enforced consistently across all languages.

### When to use Protovalidate

- **API ingress**: Validate incoming requests before processing
- **Critical business logic**: Enforce invariants at the proto level
- **Cross-language consistency**: Same rules in Go, TS, Python without duplication

### Current support

| Language | Status | Notes |
|----------|--------|-------|
| Go | Partial | Requires `protovalidate` Go module in consumers |
| TypeScript | Partial | Uses `@bufbuild/protovalidate` in validating services |
| Python | Not yet | Requires vendoring Protovalidate protos |

### Example annotation

```protobuf
import "buf/validate/validate.proto";

message CreatePlanRequest {
  string plan_name = 1 [(buf.validate.field).string.min_len = 1];
  int64 price_cents = 2 [(buf.validate.field).int64.gte = 0];
}
```

**Note**: For UI applications, Protovalidate is typically used server-side. Client-side validation (Zod) provides immediate feedback to users, while Protovalidate provides a backend safety net. See the `ui-health` skill in prompt-manager for client-side validation patterns.

## Troubleshooting

### Code generation fails

1. **Check proto syntax**: Run `make lint` to validate your proto files
2. **Check imports**: Ensure all imports exist and paths are correct
3. **Check buf.yaml**: Verify your scenario is included in the `inputs` section

### Breaking change detected

`make breaking SCENARIO=<scenario>` asks proto-health to inspect the selected
scenario's compatibility impact. If you intentionally made a breaking change:

1. Document the change in your PR
2. Ensure downstream consumers are updated
3. Consider using field deprecation instead of removal when possible

### Generated code not updating

```bash
# Regenerate the affected scenario through the staged pipeline
make generate SCENARIO=<scenario>

# Verify generated code is committed
make verify-committed-gen
```

### TypeScript import issues

- `@vrooli/proto-types` is ESM-only
- For Jest/CJS consumers, use dynamic `import()` or configure Jest for ESM
- Ensure `tsconfig.json` includes `"moduleResolution": "bundler"` or `"node16"`

### Python import errors

Python Tier 1 support is limited. If you see `ModuleNotFoundError` for `buf.validate`:
- Protovalidate annotations require additional Python packaging
- For local development, install `gen/python` in editable mode
- Consider vendoring Protovalidate protos if needed
