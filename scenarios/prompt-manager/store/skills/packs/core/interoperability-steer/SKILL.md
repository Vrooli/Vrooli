## Steer focus: Interoperability Steer

Prioritize **hardening type-safe, proto-first contracts** in `scenarios/{{TARGET}}/` for UI↔API and API↔API boundaries. This skill steers toward contracts that are impossible to misuse, eliminating stringly-typed drift and unsafe type coercion in the target scenario.

Your goal is to ensure the target scenario **communicates reliably** with shared protos, using proper serialization options, runtime validation at boundaries, and no unsafe type assertions.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve the scenario's interoperability posture.

Required reading:
- `prompt-manager skills read visited-tracker-tools`

Optional reading:
- `prompt-manager skills read api-steer`

---

### 0. Why This Skill Exists

Inter-scenario communication is the connective tissue of Vrooli. When `{{TARGET}}`'s boundaries are weak—using ad-hoc JSON shapes, unsafe type casts, or ignored schemas—the system becomes fragile in ways that are invisible until runtime:

- **Silent data loss:** Fields renamed or removed without notice
- **Type drift:** UI expects `camelCase`, API sends `snake_case`, parsers silently drop data
- **Unsafe casts:** `as SomeType` hides missing fields until production crashes
- **Validation gaps:** Rules exist in proto but aren't enforced at boundaries
- **Documentation rot:** Types in code diverge from stated contracts

**Proto-first contracts solve this** by making the schema the single source of truth. When adopted correctly, generated types flow through every layer—eliminating hand-written interfaces, enforcing validation rules, and catching breaking changes before they ship.

But "using protos" alone isn't enough. This skill ensures protos are:
- **Properly structured** (layer architecture, domain alignment, explicit enums)
- **Correctly generated** (regeneration workflow, verification)
- **Safely consumed** (no unsafe casts, proper serialization options)
- **Runtime validated** (protovalidate at boundaries)

---

### 1. Scope Boundaries

**In scope**
- Proto schema design: structure, naming, layer architecture, validation annotations
- Proto generation workflow: when to regenerate, verification, breaking change detection
- Type-safe consumption: parsing, serialization, casing compatibility
- Runtime validation: protovalidate usage, boundary enforcement
- Inter-scenario contracts: how scenarios communicate via shared proto types
- UI↔API communication: JSON handling, normalization patterns
- Anti-patterns and enforcement: what NOT to do and how to detect violations

**Out of scope**
- API surface design (endpoints, routing, error models) → see `api-steer`
- Domain modeling and business logic → see domain architecture skills
- gRPC vs REST transport decisions → implementation choice, not interoperability
- Language-specific codegen configuration (buf.yaml details) → operational docs

---

### 2. The Proto Contract Hierarchy

Proto contracts exist at multiple levels. Understanding this hierarchy prevents confusion about what to change and where.

```
                    PROTO CONTRACT HIERARCHY
┌─────────────────────────────────────────────────────────┐
│  packages/proto/schemas/                                │
│  ──────────────────────                                 │
│  SOURCE OF TRUTH                                        │
│  • .proto files define the canonical schema             │
│  • Changes here flow to all consumers                   │
│  • Breaking changes require deprecation cycle           │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼ make generate
┌─────────────────────────────────────────────────────────┐
│  packages/proto/gen/{go,typescript,python}/             │
│  ──────────────────────────────────────────             │
│  GENERATED ARTIFACTS                                    │
│  • Never edit directly—always regenerate                │
│  • Committed to repo (enables offline builds)           │
│  • Must stay in sync with schemas                       │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼ import
┌─────────────────────────────────────────────────────────┐
│  Scenario code (UI, API, CLI)                           │
│  ───────────────────────────                            │
│  CONSUMERS                                              │
│  • Import from generated packages                       │
│  • Use fromJson/toJson with proper options              │
│  • Never define parallel types—use proto types          │
└─────────────────────────────────────────────────────────┘
```

**Steer:** Changes flow downward only. Never modify generated code. Never define hand-written types that duplicate proto types.

---

### 3. Proto Schema Design Principles

#### 3.1 Layer Architecture

**The core principle:** Proto files are organized into layers where **higher layers may import from lower layers, but never the reverse**. This prevents circular dependencies and creates a clear, one-way dependency flow.

Think of it like building construction:
- Foundation (Layer 0) supports everything but depends on nothing
- Each floor above can use everything below it
- The roof (Layer 5) sits on top, using all layers beneath

**Why this matters:**
```
WITHOUT layers:
  execution.proto imports workflow.proto
  workflow.proto imports execution.proto  ← CIRCULAR! Won't compile

WITH layers:
  Layer 4 (execution) can import Layer 2 (workflow) ✓
  Layer 2 (workflow) cannot import Layer 4 (execution) ✓
  Shared concepts live in lower layers where all can access them
```

**Layer definitions (concept, not folder names):**

| Layer | Purpose | What belongs here |
|-------|---------|-------------------|
| 0 | Foundational primitives | Enums, basic types, shared constants |
| 1 | Domain concepts | Domain models independent of execution |
| 2 | Definitions/templates | Workflow definitions, action schemas |
| 3 | Historical/audit data | Logs, recordings, timeline entries |
| 4 | Runtime state | Execution state, live session data |
| 5 | API surface | Service definitions, top-level aggregates |

**Folder names are scenario-specific.** For example, browser-automation-studio uses `base/`, `domain/`, `actions/`, `timeline/`, `execution/`, `api/`. Another scenario might use `models/`, `events/`, `services/`. The *concept* of layered imports is universal; the folder structure is not.

Each proto file declares its layer in the header: `@layer 2`. This makes the dependency rules explicit and verifiable.

**Convergence Pattern: Layer Placement Decision**

```
Where should this new type live?
│
├─ Is it a fundamental primitive (enum, geometry, status)?
│   └─ YES → Layer 0
│
├─ Is it a domain concept independent of execution?
│   └─ YES → Layer 1
│
├─ Is it a definition or template (workflows, actions)?
│   └─ YES → Layer 2
│
├─ Is it history/audit/recording data?
│   └─ YES → Layer 3
│
├─ Is it runtime execution state?
│   └─ YES → Layer 4
│
└─ Is it a service API or top-level aggregate?
    └─ YES → Layer 5
```

#### 3.2 Use Enums Over Strings for Well-Defined States

**Steer:** When a field has a finite, known set of values, use an enum instead of a string.

| Scenario | Use Enum | Use String |
|----------|----------|------------|
| Status with known states (pending/running/completed) | ✅ | |
| Type discriminator (action_type, node_type) | ✅ | |
| User-provided free text (name, description) | | ✅ |
| Dynamic/extensible values (plugin names) | | ✅ |

**Enum Requirements:**
- First value MUST be `ENUM_NAME_UNSPECIFIED = 0`
- All values prefixed with enum name: `EXECUTION_STATUS_RUNNING`
- Document valid state transitions for lifecycle enums

```protobuf
// GOOD - explicit states, type-safe
enum ExecutionStatus {
  EXECUTION_STATUS_UNSPECIFIED = 0;
  EXECUTION_STATUS_PENDING = 1;
  EXECUTION_STATUS_RUNNING = 2;
  EXECUTION_STATUS_COMPLETED = 3;
}

// BAD - string allows typos, no exhaustiveness checking
string status = 1;  // "pending" | "running" | "completed" ???
```

#### 3.3 Use Optional for Semantic Distinction

Use `optional` when distinguishing "unset" from default value matters:

```protobuf
// GOOD - can distinguish "use default timeout" vs "zero timeout"
optional int32 timeout_ms = 1;

// BAD - zero and unset are indistinguishable
int32 timeout_ms = 1;
```

#### 3.4 Typed Metadata Over Raw Struct

**Steer:** Prefer explicit typed fields over `google.protobuf.Struct` or `map<string, string>`.

```protobuf
// GOOD - type-safe, documented, validated
message PlanMetadata {
  bool is_featured = 1;
  string badge_text = 2;
  optional string promo_code = 3;
}

message PricingPlan {
  PlanMetadata metadata_typed = 10;

  // DEPRECATED - kept for migration only
  map<string, string> metadata = 5 [deprecated = true];
}

// BAD - no type safety, validation impossible
map<string, google.protobuf.Value> metadata = 1;
```

Use `common.v1.JsonValue` when truly dynamic JSON is needed, but prefer typed fields when structure is known.

#### 3.5 Protovalidate Annotations

Add validation rules directly in proto schemas using protovalidate:

```protobuf
import "buf/validate/validate.proto";

message CreateProfileRequest {
  // Profile identifier.
  // @format uuid
  string profile_id = 1 [(buf.validate.field).string.uuid = true];

  // Profile name.
  string name = 2 [(buf.validate.field).string = {
    min_len: 1
    max_len: 100
  }];

  // Priority level.
  optional int32 priority = 3 [(buf.validate.field).int32 = {
    gte: 1
    lte: 10
  }];
}
```

**Common validation patterns:**
- `string.uuid = true` — UUID format
- `string.min_len` / `string.max_len` — length bounds
- `int32.gte` / `int32.lte` — numeric bounds
- `repeated.min_items` / `repeated.max_items` — array size
- `message.required = true` — non-null nested message

---

### 4. Proto Generation Workflow

#### 4.1 Vrooli's Monorepo Context

Unlike typical proto setups where schemas live in a shared repo and consumers pull from it, Vrooli keeps everything in one monorepo:

```
packages/proto/
├── schemas/          ← Source .proto files
└── gen/              ← Generated Go, TypeScript, Python (also in repo)

scenarios/
├── agent-manager/    ← Consumes generated types
├── browser-automation-studio/
└── ...
```

**What this means:**
- Proto schemas and all consumers live together
- Generated code is checked into the repo (enables offline builds, ensures consistency)
- Changes to schemas require regeneration before consumers see the updates
- No external dependencies or version coordination—just keep `gen/` in sync with `schemas/`

#### 4.2 When to Regenerate

**ALWAYS regenerate after:**
- Adding, modifying, or removing any `.proto` file
- Changing field numbers, types, or names
- Adding or modifying validation annotations
- Changing enum values

**Regeneration command:**
```bash
cd packages/proto && make generate
```

This regenerates all language outputs (Go, TypeScript, Python) from the current schemas.

#### 4.3 Verification Steps

After regenerating, run these checks to catch problems early:

```bash
# 1. Lint proto files for style/syntax issues
make lint

# 2. Check for breaking changes against master
make breaking

# 3. Verify generated code matches schemas
make check
```

**Convergence Pattern: Proto Change Workflow**

```
Edit .proto file(s)
│
├─ make generate
│   └─ Regenerates Go, TypeScript, Python from schemas
│
├─ make lint
│   └─ Catches syntax errors, bad imports, style issues
│
├─ make breaking
│   └─ Flags breaking changes (removed fields, type changes)
│   └─ If breaking: reconsider approach or update all consumers
│
└─ make check
    └─ Verifies gen/ matches schemas/ (nothing out of sync)
```

**After this workflow:** Both `schemas/` and `gen/` reflect your changes, and all scenario code that imports from `@vrooli/proto-types` or the Go packages sees the updated types immediately.

#### 4.4 Breaking Change Handling

| Change Type | Breaking? | Safe Alternative |
|-------------|-----------|------------------|
| Add optional field | No | — |
| Add enum value | No | — |
| Remove field | YES | Deprecate first, then reserve |
| Rename field | YES | Add new field, deprecate old |
| Change field type | YES | Add new field with new type |
| Change enum value number | YES | Never reuse numbers |
| Change field number | YES | Never change numbers |

**Steer:** Unless explicitly stated that the scenario is greenfield, prefer additive changes. Breaking changes require a deprecation cycle with migration window.

---

### 5. Type-Safe Consumption Patterns

#### 5.1 The Golden Rule: Never Bypass Generated Types

```typescript
// ✅ GOOD - Use generated schema with proper options
import { fromJson, toJsonString } from '@bufbuild/protobuf';
import { ExecutionSchema } from '@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb';

const execution = fromJson(ExecutionSchema, apiPayload, {
  jsonOptions: { useProtoNames: true },
  ignoreUnknownFields: true,
});

// ❌ BAD - Unsafe cast bypasses all type checking
const execution = apiPayload as Execution;

// ❌ BAD - Hand-written interface duplicates proto type
interface Execution {
  executionId: string;
  status: string;
}
```

#### 5.2 JSON Casing Compatibility

Vrooli APIs use `snake_case` JSON. Generated proto types use `camelCase` property names. **Always use `useProtoNames: true`** for serialization/deserialization:

```typescript
// Parsing API response (snake_case JSON → proto type)
const execution = fromJson(ExecutionSchema, apiResponse, {
  jsonOptions: { useProtoNames: true },  // Handles snake_case input
  ignoreUnknownFields: true,             // Tolerates API evolution
});

// Sending to API (proto type → snake_case JSON)
const json = toJsonString(ExecutionSchema, execution, {
  jsonOptions: { useProtoNames: true },  // Outputs snake_case
});
```

**Go equivalent:**
```go
// Parsing
var execution Execution
opts := protojson.UnmarshalOptions{DiscardUnknown: true}
if err := opts.Unmarshal(data, &execution); err != nil {
    // handle error
}

// Serializing
opts := protojson.MarshalOptions{UseProtoNames: true}
data, err := opts.Marshal(&execution)
```

#### 5.3 Convergence Pattern: Type Consumption Decision Tree

```
How should I consume this proto type?
│
├─ Receiving data from API/WebSocket?
│   └─ fromJson(Schema, data, { jsonOptions: { useProtoNames: true } })
│
├─ Sending data to API?
│   └─ toJsonString(Schema, obj, { jsonOptions: { useProtoNames: true } })
│
├─ Need to check if optional field is set?
│   └─ Check with hasXxx() method or truthy check + presence
│
├─ Need to work with the type in UI components?
│   └─ Use the proto type directly, or create thin wrapper via create()
│
└─ Need to define component props?
    └─ Import and use the generated type, never duplicate
```

#### 5.4 Handling Optional Fields

Proto3 optional fields require explicit presence checks:

```typescript
// ✅ GOOD - Check presence before using
if (execution.timeoutMs !== undefined) {
  const timeout = execution.timeoutMs;
}

// ❌ BAD - Assumes undefined means zero
const timeout = execution.timeoutMs || 30000;  // Wrong if 0 is valid
```

**Go pattern:**
```go
if execution.TimeoutMs != nil {
    timeout := *execution.TimeoutMs
}
```

---

### 6. Runtime Validation at Boundaries

#### 6.1 The Boundary Rule

**Steer:** Validate at system boundaries, trust internally.

| Boundary | Validate? | Example |
|----------|-----------|---------|
| API ingress (external input) | ✅ Always | Incoming HTTP/gRPC requests |
| Inter-scenario calls | ✅ Recommended | Calls between scenarios |
| Internal service calls | Usually not | Within same scenario |
| UI before submit | ✅ Always | User input validation |
| UI after API response | Parsing validates | fromJson catches malformed |

#### 6.2 Protovalidate Runtime Enforcement

**TypeScript:**
```typescript
import { createValidator } from '@bufbuild/protovalidate';
import { CreateProfileRequestSchema } from '@vrooli/proto-types/agent-manager/v1/api/service_pb';

const validator = createValidator();
const request = fromJson(CreateProfileRequestSchema, body, opts);

const result = validator.validate(CreateProfileRequestSchema, request);
if (!result.valid) {
  throw new ValidationError(result.violations);
}
```

**Go:**
```go
import "github.com/bufbuild/protovalidate-go"

validator, _ := protovalidate.New()
if err := validator.Validate(request); err != nil {
    // Handle validation errors
}
```

#### 6.3 Layered Validation Strategy

```
┌─────────────────────────────────────────────────────────┐
│  UI Layer                                               │
│  • Zod/form validation for immediate feedback           │
│  • Mirrors proto constraints for UX                     │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│  API Boundary                                           │
│  • fromJson parsing (catches malformed JSON)            │
│  • Protovalidate (enforces schema constraints)          │
│  • Auth/authz middleware                                │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│  Domain Layer                                           │
│  • Business rule validation                             │
│  • Cross-field invariants                               │
│  • State machine guards                                 │
└─────────────────────────────────────────────────────────┘
```

---

### 7. Anti-Patterns to Detect and Eliminate

#### 7.1 Unsafe Type Assertions

```typescript
// ❌ ANTI-PATTERN: as-cast bypasses type safety
const execution = response.data as Execution;

// ❌ ANTI-PATTERN: any cast defeats the purpose
const execution: any = response.data;
execution.status;  // No type checking

// ✅ CORRECT: Use fromJson with schema
const execution = fromJson(ExecutionSchema, response.data, opts);
```

**Detection:** Search for `as Execution`, `as I[A-Z]`, `: any`, `as any` near API consumption.

#### 7.2 Hand-Written Duplicate Types

```typescript
// ❌ ANTI-PATTERN: Duplicating proto type
interface Execution {
  executionId: string;
  status: ExecutionStatus;
  entries: TimelineEntry[];
}

// ✅ CORRECT: Import from generated package
import type { Execution } from '@vrooli/proto-types/.../execution_pb';
```

**Detection:** Search for `interface.*{` in files that also import from `proto-types`.

#### 7.3 Ignoring Serialization Options

```typescript
// ❌ ANTI-PATTERN: Missing useProtoNames (casing mismatch)
const execution = fromJson(ExecutionSchema, data);

// ❌ ANTI-PATTERN: Parsing raw JSON without schema
const execution = JSON.parse(response.body);

// ✅ CORRECT: Proper options
const execution = fromJson(ExecutionSchema, data, {
  jsonOptions: { useProtoNames: true },
});
```

**Detection:** Search for `fromJson\(.*\)` without `useProtoNames`.

#### 7.4 Stringly-Typed Status Fields

```typescript
// ❌ ANTI-PATTERN: String comparison for status
if (execution.status === 'completed') { ... }

// ✅ CORRECT: Use generated enum
import { ExecutionStatus } from '@vrooli/proto-types/.../shared_pb';
if (execution.status === ExecutionStatus.COMPLETED) { ... }
```

#### 7.5 Skipping Validation at Boundaries

```typescript
// ❌ ANTI-PATTERN: Trust external input without validation
app.post('/api/profiles', (req, res) => {
  createProfile(req.body);  // What if body is malformed?
});

// ✅ CORRECT: Validate at boundary
app.post('/api/profiles', (req, res) => {
  const request = fromJson(CreateProfileRequestSchema, req.body, opts);
  const validation = validator.validate(request);
  if (!validation.valid) {
    return res.status(400).json(formatViolations(validation.violations));
  }
  createProfile(request);
});
```

---

### 8. Interoperability Audit

Before making changes, assess `{{TARGET}}`'s current interoperability posture.

#### 8.1 Audit Commands

```bash
# Find proto type imports in target scenario
rg "from '@vrooli/proto-types" --type ts -l scenarios/{{TARGET}}/

# Find unsafe type assertions near API calls
rg "as [A-Z][a-zA-Z]+\b" --type ts -C 2 scenarios/{{TARGET}}/ | rg -i "fetch|response|axios|api"

# Find hand-written interfaces that might duplicate protos
rg "interface\s+[A-Z][a-zA-Z]+\s*\{" --type ts scenarios/{{TARGET}}/

# Find fromJson without useProtoNames
rg "fromJson\([^)]+\)" --type ts scenarios/{{TARGET}}/ | rg -v "useProtoNames"

# Find string comparisons that should use enums
rg "=== ['\"]" --type ts scenarios/{{TARGET}}/ | rg -i "status|state|type|mode"

# Check if proto generation is in sync
cd packages/proto && make check
```

#### 8.2 Red Flags Checklist

- [ ] Files with `as SomeType` near fetch/axios/API calls
- [ ] Interfaces that duplicate proto message shapes
- [ ] `fromJson` calls missing `useProtoNames: true`
- [ ] String literals compared to status/type fields
- [ ] No protovalidate usage at API boundaries
- [ ] Proto gen/ directory out of sync with schemas/
- [ ] Schema files missing `@layer`, `@stability` annotations

#### 8.3 Document Findings

Record audit results in `scenarios/{{TARGET}}/docs/internal/INTEROP_AUDIT.md`:

```markdown
# {{TARGET}} Interoperability Audit

## Last Updated
[Date]

## Proto Adoption Status
- [ ] All API request/response types use generated protos
- [ ] All UI↔API communication uses fromJson/toJsonString
- [ ] Protovalidate enforced at API ingress
- [ ] No unsafe type assertions

## Issues Found
1. [File:line] - Issue description
2. ...

## Priority Fixes
1. [Highest impact] - Why
2. ...
```

---

### 9. Memory Management with Visited Tracker

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `interoperability`.

---

### 10. Documentation and Memory Loop

#### 10.1 At Session Start

Read existing interoperability documentation:
- `packages/proto/README.md` — Usage patterns
- `packages/proto/STYLE_GUIDE.md` — Schema conventions
- `scenarios/{{TARGET}}/docs/internal/INTEROP_AUDIT.md` — Prior audit findings for this scenario (if exists)

#### 10.2 At Session End

Update `scenarios/{{TARGET}}/docs/internal/INTEROP_AUDIT.md`:
- The code is the source of truth. Verify existing claims against actual code before extending.
- Correct any inaccuracies discovered
- Add new anti-pattern instances found
- Update priority fixes based on work completed
- Note areas not yet audited
- Create the `docs/internal/` directory if needed

---

### 11. Output Expectations

You may update in `scenarios/{{TARGET}}/`:
- Add or modify proto schemas following layer architecture
- Add protovalidate annotations for boundary enforcement
- Refactor consuming code to use proper fromJson/toJsonString patterns
- Add type imports from generated packages
- Remove unsafe type assertions and hand-written duplicate types

You must:
- Keep `{{TARGET}}` fully functional and non-regressed
- Run `make generate` after any proto schema changes
- Run `make lint` and `make breaking` after regenerating
- Use `useProtoNames: true` for all JSON serialization/deserialization
- Validate at system boundaries (API ingress, inter-scenario calls)
- Document breaking changes and migration paths

You must NOT:
- Use `as SomeType` to bypass generated type checking
- Define hand-written interfaces that duplicate proto types
- Edit files in `gen/` directly
- Skip protovalidate at API boundaries where external input enters
- Reuse reserved field numbers or enum values

**Avoid superficial changes that rename variables or restructure code without materially improving interoperability.**
