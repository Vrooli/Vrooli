# Protocol Buffer Style Guide

This guide defines the source-of-truth structure for Vrooli proto schemas in
`packages/proto/schemas/`. Protos are fleet contracts: they must be easy for
agents to place, inspect, regenerate, and consume without guessing which
transport or ownership pattern applies.

The current standard is domain organization. The old numeric layer taxonomy is
retired; do not add new layer metadata.

## Table of Contents

- [Directory Standard](#directory-standard)
- [Where Does This Proto Go?](#where-does-this-proto-go)
- [Import Rules](#import-rules)
- [File Header](#file-header)
- [Documentation Annotations](#documentation-annotations)
- [Version Directories](#version-directories)
- [Naming Conventions](#naming-conventions)
- [Validation](#validation)
- [Field Documentation](#field-documentation)
- [Enum Documentation](#enum-documentation)
- [Deprecation Patterns](#deprecation-patterns)
- [Generation Workflow](#generation-workflow)
- [Validation Checklist](#validation-checklist)

---

## Directory Standard

Scenario-owned schemas live under:

```text
packages/proto/schemas/<scenario-slug>/v<n>/<domain>/<file>.proto
```

The domain folder is the architecture signal. It should mirror the product or
implementation domain an engineer would expect to find in the scenario code:
`validation`, `health`, `search`, `scoring`, `settings`, `runs`, `inventory`,
and similar names.

Use this shape for new and migrated schema files:

```text
schemas/proto-health/
`-- v1/
    |-- validation/
    |   `-- validation.proto
    |-- protosurface/
    |   `-- protosurface.proto
    |-- shared/
    |   |-- errors.proto
    |   `-- health.proto
    `-- health/
        `-- health.proto
```

### Scenario Shared Types

Use `v<n>/shared/` for types shared by multiple domains inside one scenario.
Examples:

- shared request/response fragments.
- cross-domain enums used by the same scenario.
- reusable validation error types.
- scenario-local `health.proto` or `errors.proto` when those messages are
  reused beyond the health/errors service file.

If a type is only used by one domain, keep it in that domain. Do not create
shared buckets preemptively.

### Fleet Shared Types

Cross-scenario shared contracts do not belong in a scenario's `shared/`
directory. They belong in the top-level shared packages such as:

- `schemas/common/`
- `schemas/measures/`
- `schemas/architecture/`

Adding or changing a fleet shared package is a broader interoperability
decision. Keep scenario-local convenience types local.

### Legacy Layouts

Some existing schemas still use legacy folders such as `api/`, `domain/`,
`base/`, or numeric-layer annotations. Treat those as migration targets, not
patterns for new work. The validator should warn on legacy metadata and
cross-domain drift without turning the whole fleet red on day one.

---

## Where Does This Proto Go?

Use this decision tree before creating or moving a proto file:

```text
Is the contract shared across scenarios?
|-- Yes -> place it in the appropriate top-level shared package
|          (`common`, `measures`, `architecture`, or another reviewed shared
|          package).
`-- No
   Is the type used by more than one domain inside the same scenario?
   |-- Yes -> place it under schemas/<scenario>/v<n>/shared/.
   `-- No
      Is it the public RPC surface for a domain?
      |-- Yes -> place service and request/response messages in that domain
      |          folder, unless a message is reused elsewhere and belongs in
      |          shared/.
      `-- No -> place it in the domain that owns the concept.
```

If two domain folders need to import each other, that is a design smell. Move
the shared concept to `shared/`, or split the responsibility so each domain owns
its own contract.

---

## Import Rules

These rules are intra-scenario rules. Fleet-wide dependency graphs,
declared-vs-actual `service.json` drift, and cross-scenario encapsulation are
owned by scenario dependency analysis, not this style guide.

1. Import cycles are forbidden.
2. Domain folders must not import other domain folders from the same scenario.
3. Domain folders may import their scenario's `shared/` package.
4. Domain folders may import top-level cross-scenario shared packages.
5. Cross-scenario imports are allowed when the scenario intentionally consumes
   another contract; record them as facts for downstream dependency analysis.
6. Do not encode import policy in comments. The file path and import graph are
   the machine-readable source of truth.

Examples:

```protobuf
// OK: validation domain imports scenario-local shared types.
import "proto-health/v1/shared/findings.proto";

// OK: a scenario imports a reviewed fleet shared package.
import "architecture/v1/findings.proto";

// Not OK: validation imports another local product domain directly.
import "proto-health/v1/protosurface/protosurface.proto";
```

When a direct domain import feels unavoidable, prefer making the dependency
explicit through `shared/` and documenting why that shared concept exists.

---

## File Header

Every proto file must start with a concise header block:

```protobuf
syntax = "proto3";

package proto_health.v1.validation;

import "buf/validate/validate.proto";

option go_package = "github.com/matthalloran/Vrooli/packages/proto/gen/go/proto-health/v1/validation;validationv1";

// =============================================================================
// VALIDATION
// =============================================================================
//
// @stability experimental
//
// Validates one scenario's proto contract structure and reports actionable
// findings for the quality loop.
//
// USAGE CONTEXTS:
//   - test-genie: runs this validator as the proto phase.
//   - ecosystem-manager: consumes proto findings as an R2 maturity signal.
//
// =============================================================================
```

The header must explain ownership and usage. It must not duplicate facts that
the path or imports already expose.

---

## Documentation Annotations

Annotations are machine-parseable metadata that appear at the start of a comment
line. Keep the registry small and stable.

### Supported File-Level Annotation

Every file must include `@stability`:

| Value | Meaning | Compatibility Guarantee |
|-------|---------|-------------------------|
| `stable` | Production-ready, relied on by consumers | Breaking changes require a deprecation cycle |
| `beta` | Feature-complete but still hardening | Breaking changes require notice |
| `experimental` | Early or intentionally volatile | No compatibility guarantee |

Use exactly these values. Avoid synonyms such as `unstable`.

### Supported Documentation Annotations

| Annotation | Scope | Purpose | Example |
|------------|-------|---------|---------|
| `@stability` | file | API maturity | `@stability beta` |
| `@template` | file | Marks schema copied from a scenario template example; keep until intentionally adopted or replaced | `@template react-vite/example` |
| `@deprecated` | message, field, enum, value | Human-readable migration note paired with proto deprecation options where available | `@deprecated Use TimelineEntry instead` |
| `@example` | field, message | Concrete value or payload sketch | `@example "https://example.com"` |
| `@see` | message, field, enum | Related contract or documentation | `@see WorkflowNode.id` |
| `@format` | field | Parseable scalar format when not expressed by type | `@format uuid` |
| `@unit` | field | Unit for numeric values | `@unit milliseconds` |
| `@default` | field | Non-obvious default behavior | `@default 30000` |
| `@constraint` | message, field | Legacy prose constraint that cannot yet be expressed in Protovalidate | `@constraint mutually exclusive with ...` |

Validation constraints belong in Protovalidate rules whenever they are
enforceable. Use prose for semantics, not as the only source of truth for a
machine-checkable constraint.

`@template` is a maturity marker, not ownership. It lets validators distinguish
legitimate scenario-owned concepts from scaffold examples with generic names
such as `notes`. Remove it only when the contract is deliberately adopted as
scenario-owned surface or replaced by the scenario's real domain contract.
Proto-health compares the generated file with its template source after
placeholder substitution: byte-identical scaffold remains a finding, while a
diverged contract is treated as adopted and the marker no longer blocks clean.

`@see` may carry typed retention declarations for messages that are intentionally
kept even when no local RPC reaches them:

- `@see consumer:<scenario>` means the named in-fleet scenario is the consumer.
  Proto-health accepts the retention only when that scenario actually references
  the message through its reachable proto surface.
- `@see external:<name>` means the consumer is outside the local fleet. Use this
  only on stable published messages; experimental or beta messages should stay
  staged until a real consumer exists.

Do not use retention annotations as a mute. If the named in-fleet consumer
drifts away, proto-health reports the message as `proto.possibly_unused` again.

### Deprecated Annotations

Do not add these annotations to new files:

- `@layer`
- `@domain`
- `@imports`

They are deprecated because the same facts are now derived from directory
structure and imports. Existing uses may remain until the owning scenario is
migrated; validators should report them as maturity warnings, not correctness
errors.

Unsupported annotations should be treated as warnings so authors can either
remove them or deliberately propose expanding this registry.

---

## Version Directories

Version directories must match:

```text
^v[1-9][0-9]*$
```

Rules:

1. A scenario's schema versions start at `v1`.
2. Versions are contiguous: `v1`, `v2`, `v3`, with no gaps.
3. New scenarios start with `v1`.
4. Do not create a new version directory for routine additive changes.
5. A new version directory means a deliberate wire-contract versioning decision.

Package names must include the version:

```protobuf
package proto_health.v1.validation;
```

The package should correspond to the path:

```text
schemas/proto-health/v1/validation/validation.proto
        -> proto_health.v1.validation
```

Use underscores in proto packages for scenario slugs that contain hyphens.

---

## Naming Conventions

### Package Names

- Use `snake_case`.
- Include the version segment.
- Include the domain segment for scenario-owned domain folders.
- Match the scenario slug after converting hyphens to underscores.

### File Names

- Use `snake_case`.
- Prefer one primary file per domain named after the domain.
- Split files when a domain has distinct subcontracts that are easier to own and
  review separately.

### Service Names

- Use `PascalCase`.
- Suffix scenario services with `Service`.
- Prefer one scenario service per cohesive API surface.

```protobuf
service ProtoHealthService {
  rpc ValidateScenario(ValidateScenarioRequest) returns (ValidateScenarioResponse);
  rpc DescribeScenarioProtos(DescribeScenarioProtosRequest) returns (DescribeScenarioProtosResponse);
}
```

### Message Names

- Use `PascalCase`.
- Be descriptive but concise.
- Suffix RPC envelopes with `Request` and `Response`.
- Suffix configuration and parameter types with `Config`, `Options`, or
  `Params` only when that describes their role.

### Field Names

- Use `snake_case`.
- Use consistent suffixes:

| Suffix | Meaning | Example |
|--------|---------|---------|
| `_id` | Identifier field | `scenario_id` |
| `_at` | Timestamp field | `created_at` |
| `_ms` | Duration in milliseconds | `timeout_ms` |
| `_count` | Count or quantity | `finding_count` |
| `_url` | URL string | `callback_url` |
| `_path` | Filesystem or repo path | `proto_path` |

### Enum Names

- Use `PascalCase` for enum type names.
- Use `SCREAMING_SNAKE_CASE` for enum values.
- Prefix all values with the enum type name.
- First value must be `*_UNSPECIFIED = 0`.

```protobuf
enum ProtoFindingSeverity {
  PROTO_FINDING_SEVERITY_UNSPECIFIED = 0;
  PROTO_FINDING_SEVERITY_ERROR = 1;
  PROTO_FINDING_SEVERITY_WARNING = 2;
  PROTO_FINDING_SEVERITY_INFO = 3;
}
```

### Timestamp Fields

Use these standard suffixes:

| Suffix | When to Use |
|--------|-------------|
| `created_at` | Resource creation time |
| `updated_at` | Last modification time |
| `started_at` | Operation start time |
| `completed_at` | Operation completion time |
| `timestamp` | Generic event occurrence time |

### UUID Fields

Fields containing UUIDs must include `@format uuid`:

```protobuf
// Unique run identifier.
// @format uuid
string run_id = 1;
```

Do not use inline prose such as `(UUID format)` when an annotation is available.

---

## Validation

Use Protovalidate as the canonical, machine-readable source of truth for
boundary constraints.

- Prefer `buf/validate/validate.proto` rules for required fields, bounds,
  formats, and CEL expressions.
- Keep comments focused on semantics, units, defaults, and user-visible behavior.
- Validate at API ingress and cross-scenario ingress/egress.
- Do not rely on comments alone for constraints that can be mechanically
  enforced.

Example:

```protobuf
message ValidateScenarioRequest {
  // Scenario slug to validate.
  string scenario = 1 [(buf.validate.field).string = {
    pattern: "^[a-z0-9][a-z0-9-]*[a-z0-9]$"
  }];
}
```

---

## Field Documentation

Every field must have a comment that includes:

1. What the field contains.
2. Non-obvious constraints or defaults.
3. Unit of measurement for numeric fields.
4. Ordering semantics for repeated fields.
5. Key/value semantics for map fields.

```protobuf
message ProtoSurface {
  // Scenario slug this surface describes.
  string scenario = 1;

  // Proto files owned by the scenario, ordered by repo path.
  repeated ProtoFile files = 2;

  // Counts by finding severity.
  // Key: severity token such as "error" or "warning".
  // Value: number of findings at that severity.
  map<string, int32> finding_counts = 3;
}
```

Use `optional` only when distinguishing unset from zero or empty matters, and
document that distinction.

---

## Enum Documentation

Every enum and enum value must be documented. Lifecycle enums should include a
short state machine when valid transitions matter.

```protobuf
// Status for a validation run.
//
// State machine:
//   QUEUED -> RUNNING -> PASSED|FAILED
//                    \-> CANCELLED
enum ValidationRunStatus {
  // Default value. Should not appear in valid persisted runs.
  VALIDATION_RUN_STATUS_UNSPECIFIED = 0;

  // Run is queued and waiting for execution.
  VALIDATION_RUN_STATUS_QUEUED = 1;

  // Run is actively executing.
  VALIDATION_RUN_STATUS_RUNNING = 2;

  // Terminal state: no error findings were produced.
  VALIDATION_RUN_STATUS_PASSED = 3;
}
```

---

## Deprecation Patterns

Deprecation must be explicit and migration-oriented.

### Deprecating Messages

```protobuf
// DEPRECATED: Use NewMessage instead.
//
// @deprecated Use NewMessage from new_file.proto.
// @see NewMessage
message OldMessage {
  option deprecated = true;

  // Replacement payload for the migration period.
  NewMessage delegate = 1;
}
```

### Deprecating Fields

```protobuf
message Example {
  // DEPRECATED: Use new_field instead.
  // @deprecated Replaced by new_field in v2.
  string old_field = 1 [deprecated = true];

  // Replacement for old_field with improved semantics.
  string new_field = 2;
}
```

### Reserved Fields

When removing fields, reserve both the number and name and explain why:

```protobuf
message Example {
  // Reserved fields from migrations:
  // - 3: old_field_name (removed in v2, replaced by new_field)
  // - 5: abandoned_flag (removed in v2, no longer used)
  reserved 3, 5;
  reserved "old_field_name", "abandoned_flag";
}
```

Never reuse field numbers or enum values.

### Deprecated File Organization

If compatibility requires keeping deprecated types, place them in a
`_deprecated/` subdirectory under the owning domain:

```text
validation/
|-- validation.proto
`-- _deprecated/
    `-- old_validation.proto
```

Greenfield schemas should not create compatibility shims.

---

## Generation Workflow

After any schema change:

```bash
cd packages/proto
make generate
make lint
make breaking
make verify-committed-gen
```

`gen/`, `gen/descriptor/image.binpb`, and `gen/manifests/*.lock.json` are
committed artifacts. They must stay in sync with `schemas/`. Do not hand-edit
generated code or manifests.

`make verify-committed-gen` is the CI-style sync gate: it runs the same
prune-clean generation path as `make generate` and fails if committed generated
artifacts, descriptors, or manifests drift from schema sources.

---

## Endpoint Declaration Rules

Deliberate REST exceptions must declare payload intent in endpoint metadata,
even when a role has no proto payload. Declare request, response, and error
roles with full proto message names where proto JSON is used, and use explicit
non-proto conformance modes such as `none`, `transport_only`, or
`external_shape` where no proto payload exists.

Stable public services may not depend on less-stable transitive message
payloads. If a stable RPC accepts or returns a message, every scenario-local
message reachable through its fields must also be stable.

Reusable scenario-local support messages belong in `v<n>/shared/`. A message
used by multiple domains, including REST exception error envelopes shared by
health and product endpoints, is misplaced if it lives in a domain-specific
folder such as `errors/`.

---

## Validation Checklist

Before committing proto changes:

- [ ] File path follows `schemas/<scenario>/v<n>/<domain>/<file>.proto`, or the
      file is in a reviewed top-level shared package.
- [ ] Version directory matches `^v[1-9][0-9]*$`; scenario versions start at
      `v1` and are contiguous.
- [ ] Package name matches scenario, version, and domain.
- [ ] File header includes `@stability` with `stable`, `beta`, or
      `experimental`.
- [ ] New files do not include deprecated `@layer`, `@domain`, or `@imports`
      annotations.
- [ ] Domain files do not import other local domain folders directly.
- [ ] Shared local types live in `v<n>/shared/`; fleet shared types live in
      reviewed top-level shared packages.
- [ ] REST exception endpoint metadata declares request, response, and error
      payload intent explicitly.
- [ ] Stable public services depend only on stable scenario-local transitive
      payload messages.
- [ ] All messages, fields, enums, and enum values have useful comments.
- [ ] UUID fields use `@format uuid`.
- [ ] Timestamp and duration fields use standard suffixes and units.
- [ ] Enforceable constraints use Protovalidate.
- [ ] Deprecated items include `@deprecated` and proto deprecation options where
      available.
- [ ] Reserved fields explain what was removed and why.
- [ ] `make lint` passes.
- [ ] `make generate` and `make verify-committed-gen` leave generated artifacts
      in sync.
