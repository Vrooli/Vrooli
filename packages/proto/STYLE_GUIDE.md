# Protocol Buffer Style Guide

This guide establishes documentation and naming standards for Vrooli proto schemas.
Following these standards ensures consistency, improves agent/tool parseability, and
makes schemas self-documenting.

## Table of Contents

- [File Structure](#file-structure)
- [Documentation Annotations](#documentation-annotations)
- [Naming Conventions](#naming-conventions)
- [Field Documentation](#field-documentation)
- [Enum Documentation](#enum-documentation)
- [Deprecation Patterns](#deprecation-patterns)
- [Layer Architecture](#layer-architecture)

---

## File Structure

### File Header Template

Every proto file MUST start with a header block:

```protobuf
syntax = "proto3";

package scenario_name.v1;

import "...";

option go_package = "...";

// =============================================================================
// [DOMAIN NAME IN CAPS]
// =============================================================================
//
// @layer [0-5]
// @domain [domain-name]
// @imports [comma-separated list of imported domains, or "none"]
// @stability [stable|beta|experimental]
//
// [1-3 sentence description of what this file contains and why]
//
// USAGE CONTEXTS:
//   - [Context 1]: [Description]
//   - [Context 2]: [Description]
//
// =============================================================================
```

### @stability Annotation

Every file MUST include a `@stability` annotation in the header to indicate API maturity:

| Value | Meaning | Compatibility Guarantee |
|-------|---------|------------------------|
| `stable` | Production-ready, widely used | Breaking changes require deprecation cycle |
| `beta` | Feature-complete, may have rough edges | Breaking changes with notice |
| `experimental` | Early development, subject to change | No compatibility guarantee |

```protobuf
// @stability stable
// Core types used throughout the system

// @stability beta
// New feature being validated in production

// @stability experimental
// Prototype types, expect changes
```

### @version Annotation (Optional)

For tracking schema evolution, you may add `@version` annotations to document when types were added or changed:

```protobuf
// @version 1.0 - Initial release
// @version 1.1 - Added metadata_typed field
// @version 1.2 - Deprecated old_field, added new_field
message PricingTier {
  ...
}
```

### Section Separators

Use consistent separators to group related types:

```protobuf
// =============================================================================
// SECTION NAME
// =============================================================================
```

---

## Documentation Annotations

Use `@` annotations for machine-parseable metadata. These MUST appear at the
start of a line within a comment block.

## Validation (Protovalidate)

Use Protovalidate rules as the canonical, machine-readable source of truth for validation constraints.

- Prefer `buf/validate/validate.proto` rules (e.g., required, bounds, formats, CEL) over documenting constraints purely in comments.
- Keep comments for semantics and context (why the field exists, units, API behavior, defaults), even when Protovalidate rules exist.
- Treat `@constraint` as a documentation aid only; when a constraint is enforceable, encode it with Protovalidate.

### Standard Annotations

| Annotation | Purpose | Example |
|------------|---------|---------|
| `@layer` | Import hierarchy level (0-5) | `@layer 2` |
| `@domain` | Domain category | `@domain actions` |
| `@imports` | Dependencies | `@imports base, domain` |
| `@usage` | Where this type is used | `@usage Execution.status` |
| `@see` | Related types/docs | `@see WorkflowNodeV2` |
| `@deprecated` | Deprecation notice | `@deprecated Use TimelineEntry instead` |
| `@constraint` | Validation rules | `@constraint Must be unique` |
| `@default` | Default value | `@default 30000` |
| `@unit` | Measurement unit | `@unit milliseconds` |
| `@format` | Data format | `@format uuid` |
| `@example` | Example value | `@example "https://example.com"` |

### Message-Level Documentation

```protobuf
// Brief one-line description of the message.
//
// Extended description explaining purpose, behavior, and context.
// Can span multiple lines.
//
// @usage Where/how this type is used
// @see Related types
message ExampleMessage {
  ...
}
```

### Field-Level Documentation

```protobuf
message Example {
  // Brief description of the field.
  // Additional context if needed.
  // @format uuid
  // @constraint Must be non-empty
  string id = 1;

  // Description with unit information.
  // @unit milliseconds
  // @default 30000
  optional int32 timeout_ms = 2;
}
```

---

## Naming Conventions

### Package Names

- Use `snake_case` for package names
- Include version suffix: `scenario_name.v1`
- Match directory structure

### Message Names

- Use `PascalCase`
- Be descriptive but concise
- Suffix with purpose when helpful: `*Request`, `*Response`, `*Config`, `*Params`

### Field Names

- Use `snake_case`
- Use consistent suffixes for common patterns:

| Suffix | Meaning | Example |
|--------|---------|---------|
| `_id` | Identifier field | `workflow_id`, `project_id` |
| `_at` | Timestamp field | `created_at`, `started_at` |
| `_ms` | Duration in milliseconds | `timeout_ms`, `delay_ms` |
| `_count` | Count/quantity | `workflow_count`, `retry_count` |
| `_url` | URL string | `storage_url`, `callback_url` |
| `_path` | Filesystem path | `folder_path`, `workflow_path` |

### Enum Names

- Use `SCREAMING_SNAKE_CASE`
- Prefix all values with enum name: `ENUM_NAME_VALUE`
- First value MUST be `ENUM_NAME_UNSPECIFIED = 0`

```protobuf
enum ExampleStatus {
  EXAMPLE_STATUS_UNSPECIFIED = 0;
  EXAMPLE_STATUS_ACTIVE = 1;
  EXAMPLE_STATUS_INACTIVE = 2;
}
```

### Timestamp Fields

Use these standard suffixes:

| Suffix | When to Use |
|--------|-------------|
| `created_at` | Resource creation time (immutable) |
| `updated_at` | Last modification time |
| `started_at` | Operation/execution start time |
| `completed_at` | Operation/execution end time |
| `timestamp` | Generic event occurrence time |

### UUID Fields

Fields containing UUIDs MUST include the `@format uuid` annotation for machine parseability:

```protobuf
// Unique identifier for the workflow.
// @format uuid
string workflow_id = 1;
```

For common ID field patterns, use this standard format:

```protobuf
// Unique workflow identifier.
// @format uuid
string workflow_id = 1;

// Project this workflow belongs to.
// @format uuid
string project_id = 2;

// Node ID from the workflow definition.
// @format uuid
// @see WorkflowNodeV2.id
optional string node_id = 3;
```

**IMPORTANT**: Do NOT use the older `(UUID format)` inline style. Always use `@format uuid` annotation:

```protobuf
// CORRECT - machine parseable
// Unique execution identifier.
// @format uuid
string execution_id = 1;

// DEPRECATED - avoid this style
// Unique execution identifier (UUID format).
string execution_id = 1;
```

---

## Field Documentation

### Required Documentation

Every field MUST have a comment that includes:

1. Brief description of what the field contains
2. Any constraints or validation rules
3. Default value if optional and non-obvious
4. Unit of measurement for numeric fields

### Optional vs Required

- Use `optional` when distinguishing "unset" from zero/empty matters
- Document the semantic difference in comments

```protobuf
// Maximum timeout. If unset, uses system default (30000ms).
// @unit milliseconds
// @default 30000
optional int32 timeout_ms = 1;

// User ID who created this resource. Always populated.
string created_by = 2;
```

### Map Fields

Document both key and value meanings:

```protobuf
// Variables injected into workflow execution.
// Key: Variable name (e.g., "username")
// Value: Variable value (supports string interpolation)
map<string, string> variables = 1;
```

### Repeated Fields

Document ordering semantics:

```protobuf
// Captured timeline entries, ordered by sequence_num ascending.
repeated TimelineEntry entries = 1;
```

---

## Enum Documentation

### Enum-Level Documentation

```protobuf
// Brief description of what this enum represents.
//
// Extended context about usage and behavior.
// Include state machine diagrams for lifecycle enums.
//
// @usage Where this enum is used
enum ExampleStatus {
  ...
}
```

### Value Documentation

Every enum value MUST have a comment explaining:

1. What this value represents
2. When it occurs / conditions
3. What actions are valid in this state (for lifecycle enums)

```protobuf
enum ExecutionStatus {
  // Default/unknown state. Should never appear in valid data.
  // Indicates missing or corrupted status field.
  EXECUTION_STATUS_UNSPECIFIED = 0;

  // Execution is queued and waiting for an available executor.
  // Valid transitions: → RUNNING, → CANCELLED
  EXECUTION_STATUS_PENDING = 1;

  // Execution is actively running.
  // Valid transitions: → COMPLETED, → FAILED, → CANCELLED
  EXECUTION_STATUS_RUNNING = 2;

  // Terminal state: Execution finished successfully.
  EXECUTION_STATUS_COMPLETED = 3;
}
```

### State Machine Documentation

For lifecycle enums, include ASCII state diagram:

```protobuf
// State machine:
//   PENDING → RUNNING → COMPLETED|FAILED|CANCELLED
//                  ↓
//              RETRYING → RUNNING
```

---

## Deprecation Patterns

### Deprecating Messages

```protobuf
// DEPRECATED: Use NewMessage instead.
//
// @deprecated Use NewMessage from new_file.proto
// @see NewMessage
message OldMessage {
  option deprecated = true;

  // Delegate to new type for migration period.
  NewMessage delegate = 1;
}
```

### Deprecating Fields

```protobuf
message Example {
  // DEPRECATED: Use new_field instead.
  // @deprecated Replaced by new_field in v1.2
  string old_field = 1 [deprecated = true];

  // Replacement for old_field with improved semantics.
  string new_field = 2;
}
```

### Reserved Fields

When removing fields entirely, you MUST document what was removed and why:

```protobuf
message Example {
  // Reserved fields from migrations:
  // - 3: old_field_name (removed in v1.2, replaced by new_field)
  // - 5: another_field (removed in v1.3, no longer needed)
  reserved 3, 5;
  reserved "old_field_name", "another_field";
}
```

**Required format for reserved field comments:**

```protobuf
// Reserved: field [N] was '[field_name]' ([reason for removal])
reserved N;

// Or for multiple fields:
// Reserved fields from migrations:
// - [N]: [field_name] ([reason])
// - [M]: [field_name] ([reason])
reserved N, M;
```

**Common removal reasons:**
- `removed in vX.Y, replaced by [new_field]` - field was superseded
- `removed in vX.Y, migrated to [new_type]` - data moved to different type
- `removed in vX.Y, no longer needed` - feature removed
- `removed in vX.Y, use [alternative] instead` - different approach taken

**Examples from codebase:**

```protobuf
// GOOD - clear explanation
// Reserved: field 5 was timeout_seconds (migrated to timeout_ms for consistency)
reserved 5;

// GOOD - multiple fields with context
// Reserved fields from migrations:
// - 6: trigger_data (migrated to trigger_metadata)
// - 7: source (migrated to trigger_type)
// - 12: result_data (migrated to result)
reserved 6, 7, 12;

// BAD - no explanation
reserved 5;  // Why was this removed?
```

### Deprecation File Organization

Place deprecated types in `_deprecated/` subdirectory:

```
domain/
├── current_types.proto
└── _deprecated/
    └── old_types.proto   # Re-exports for backwards compat
```

---

## Layer Architecture

### Layer Definitions

| Layer | Directory | Purpose | Can Import |
|-------|-----------|---------|------------|
| 0 | `base/` | Fundamental types, enums | External only |
| 1 | `domain/` | Domain primitives | Layer 0 |
| 2 | `actions/`, `workflows/` | Action/workflow definitions | Layers 0-1 |
| 3 | `timeline/`, `recording/` | Execution history | Layers 0-2 |
| 4 | `execution/` | Runtime state | Layers 0-3 |
| 5 | `api/`, `projects/` | Service definitions | Layers 0-4 |

### Import Rules

1. Types MUST only import from lower layers
2. Circular imports are FORBIDDEN
3. Intra-layer imports ARE allowed (e.g., `domain/telemetry.proto` can import `domain/selectors.proto`)
4. Document layer in file header with `@layer` annotation
5. Document imports with `@imports` annotation using domain names (not file paths)

### @imports Annotation Format

The `@imports` annotation lists **domain names** that are imported, not file paths:

```protobuf
// CORRECT - list domain names
// @imports base, domain

// INCORRECT - don't use file paths or subdirectories
// @imports base/shared, domain/selectors
// @imports timeline/entry
```

For intra-layer imports within the same domain, you may omit them or note them explicitly:

```protobuf
// Option 1: Omit intra-domain imports (preferred for simplicity)
// @imports base

// Option 2: Note intra-domain imports explicitly
// @imports base (also: domain/selectors within same layer)
```

### Adding New Types

1. Identify the appropriate layer based on dependencies
2. Place in the correct directory matching that layer
3. Add `@layer`, `@domain`, `@imports`, and `@stability` annotations to file header
4. Run `make lint` to verify no circular dependencies
5. Update domain README if adding new domain

---

## Validation Checklist

Before committing proto changes:

- [ ] File header includes `@layer`, `@domain`, `@imports`, `@stability` annotations
- [ ] `@imports` uses domain names (not file paths)
- [ ] All messages have description comments
- [ ] All fields have description comments
- [ ] UUID fields have `@format uuid` annotation (not inline "(UUID format)")
- [ ] Timestamp fields use standard `_at` suffixes
- [ ] Duration fields include unit (prefer `_ms` suffix)
- [ ] All enum values have description comments
- [ ] Deprecated items have `@deprecated` annotation AND `option deprecated = true`
- [ ] Reserved fields have comment explaining what was removed and why
- [ ] `make lint` passes
- [ ] `make generate` produces no unexpected changes
