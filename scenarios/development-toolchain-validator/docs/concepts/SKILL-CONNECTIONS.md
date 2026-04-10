# Skill Connections

## Overview

A **skill connection** is the core entity in DTV. It maps a prompt-manager steer skill to a reference scenario, creating a testable relationship between "what the skill says" and "what the reference looks like."

## Connection Lifecycle

```
1. CONNECT (no config)
   │  Skill is connected to reference with version pinned.
   │  Status: unconfigured — signals that this skill needs
   │  structured expectations defined.
   │
   ▼
2. CONFIGURE (add expectations)
   │  Structural expectations and/or CLI tool assertions
   │  are added to describe what the skill expects.
   │  Status: configured — can be validated.
   │
   ▼
3. VALIDATE (run checks)
   │  Expectations are evaluated against the reference.
   │  Results: pass/fail per expectation.
   │
   ▼
4. MAINTAIN (handle drift)
      When the skill's content changes in prompt-manager,
      drift is detected. Expectations may need updating.
```

## Version Pinning

When a skill is connected, DTV stores:
- **Skill version number**: From prompt-manager's version history (integer, e.g., 49)
- **Content hash**: SHA256 of the skill content at connection time

This enables drift detection: if either the version or hash changes in prompt-manager, DTV flags the connection as potentially stale.

### How Versioning Works in prompt-manager

prompt-manager tracks skill versions at three levels:
1. **`revision` field** in `skill.json`: Lightweight metadata counter
2. **Content hash** from `/api/v1/skills/sync`: SHA256 of full response for change detection
3. **Version numbers** from `/api/v1/skills/{id}/versions`: Sequential integers (1, 2, 3...) for point-in-time content recovery

DTV uses the **version number** (most granular) and **content hash** (most efficient for bulk checking).

## Unconfigured Connections

A connection with no structural expectations and no CLI tool assertions is **unconfigured**. This is a valid state — it means:

1. The skill is known to be applicable to this reference
2. Nobody has yet defined what the skill expects structurally
3. The skill may be too prose-heavy or vague for programmatic description

Unconfigured connections are surfaced in reports as the lowest maturity level. The meta optimization team should treat these as work items: either structure the skill's guidance enough to define expectations, or determine that the skill is inherently unsuitable for programmatic validation.

## Structural Expectations

Three types of structural expectations can be defined:

### Folder Expectations
```json
{
  "type": "folder",
  "path": "api/handlers/projects/",
  "required": true,
  "description": "API handlers organized by domain module (projects)"
}
```

The `path` field supports simple patterns. For the reference scenario `reference-react-vite`, DTV checks that this folder exists at `scenarios/reference-react-vite/api/handlers/projects/`.

### File Expectations
```json
{
  "type": "file",
  "path": "api/handlers/**/*_test.go",
  "required": true,
  "description": "Every handler file has a co-located test file"
}
```

The `path` field supports glob patterns. DTV uses standard glob matching to verify files exist.

### Snippet Expectations
```json
{
  "type": "snippet",
  "path": "api/main.go",
  "snippet_content": "gracefulShutdown(",
  "snippet_location": "function_body",
  "required": true,
  "description": "API server implements graceful shutdown"
}
```

Snippet expectations verify that specific content appears in a specific file. The `snippet_location` field provides context for where to look (e.g., "function_body", "import_block", "file_top", or null for anywhere in file).

## CLI Tool Assertions

CLI tool assertions run read-only commands and validate their JSON output.

### Assertion Structure
```json
{
  "command": "scenario-auditor audit reference-react-vite --json",
  "json_path": "$.total",
  "operator": "eq",
  "expected_value": 0,
  "description": "No auditor violations on reference"
}
```

### Important Properties

1. **Commands must be read-only**: They check state but do not modify the reference. Never configure a command that writes files, starts processes, or changes state.

2. **Commands must support `--json`**: DTV parses the output as JSON and evaluates JSONPath expressions against it. Commands without structured output cannot be used.

3. **Timeouts**: Commands have a configurable timeout (default: 60 seconds). Long-running commands like `test-genie execute` may need higher timeouts.

4. **Idempotency**: Running the same assertion twice should produce the same result (barring external state changes).

### JSONPath Expressions

DTV uses JSONPath notation to extract values from JSON output:

| Expression | Meaning |
|------------|---------|
| `$.score` | Top-level "score" field |
| `$.breakdown.quality.score` | Nested field access |
| `$.phases[0].status` | Array index access |
| `$.phaseSummary.failed` | Nested summary field |
| `$.bySeverity.critical` | Map/object field access |

### Operator Reference

| Operator | Value Type | Example |
|----------|-----------|---------|
| `eq` | any | `{"path": "$.success", "op": "eq", "value": true}` |
| `neq` | any | `{"path": "$.status", "op": "neq", "value": "failed"}` |
| `gt` | number | `{"path": "$.score", "op": "gt", "value": 80}` |
| `gte` | number | `{"path": "$.score", "op": "gte", "value": 96}` |
| `lt` | number | `{"path": "$.violations", "op": "lt", "value": 5}` |
| `lte` | number | `{"path": "$.penalty", "op": "lte", "value": 2}` |
| `exists` | (none) | `{"path": "$.breakdown.quality", "op": "exists"}` |
| `contains` | string | `{"path": "$.classification", "op": "contains", "value": "ready"}` |
| `matches` | regex | `{"path": "$.version", "op": "matches", "value": "^\\d+\\.\\d+"}` |
| `between` | [min,max] | `{"path": "$.rate", "op": "between", "value": [0, 1]}` |

## Overlap and Conflict Detection

### Overlaps

When multiple skill connections have structural expectations targeting the same path, DTV reports an **overlap**. Overlaps are informational — they mean multiple skills care about the same area.

Example overlap:
- `api-steer` expects folder `api/handlers/`
- `unit-testing-architecture-steer` expects files `api/handlers/*_test.go`

This is a healthy overlap — both skills legitimately care about the handler directory.

### Conflicts

A **conflict** occurs when overlapping expectations are mutually exclusive. DTV detects these by analyzing whether all expectations for the same path can be satisfied simultaneously.

Example conflict:
- `api-steer` expects `api/routes.go` (single file for all routes)
- `screaming-architecture-audit` expects `api/handlers/{domain}/routes.go` (per-domain route files)

Both cannot be true — this is a genuine cross-steer conflict that must be resolved by updating one of the skills.
