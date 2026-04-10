# Writing Assertions

## Overview

Assertions are the declarative expectations you define for each skill-reference connection. They bridge the gap between a steer skill's prose guidance and programmatic validation.

Two types of assertions exist:
1. **Structural**: Check the reference scenario's filesystem
2. **CLI Tool**: Run commands and validate JSON output

## Structural Assertions

### Folder Assertions

Check that a directory exists in the reference scenario.

```bash
development-toolchain-validator expectations add structural \
  --connection <skill>:<reference> \
  --type folder \
  --path "api/handlers/projects/" \
  --required true \
  --description "Projects domain module exists"
```

**When to use**: When a skill mandates a specific directory structure (e.g., api-steer's "organize by domain").

### File Assertions

Check that files matching a pattern exist.

```bash
# Exact file
development-toolchain-validator expectations add structural \
  --connection <skill>:<reference> \
  --type file \
  --path "api/main.go" \
  --required true \
  --description "API entry point exists"

# Glob pattern
development-toolchain-validator expectations add structural \
  --connection <skill>:<reference> \
  --type file \
  --path "api/handlers/**/*_test.go" \
  --required true \
  --description "Handler test files co-located with handlers"
```

**When to use**: When a skill requires specific files or file patterns (e.g., unit-testing-architecture-steer's "co-locate test files").

### Snippet Assertions

Check that specific content appears in a file.

```bash
development-toolchain-validator expectations add structural \
  --connection <skill>:<reference> \
  --type snippet \
  --path "api/main.go" \
  --snippet-content "gracefulShutdown(" \
  --snippet-location "function_body" \
  --required true \
  --description "Server implements graceful shutdown"
```

**When to use**: When a skill requires specific patterns in code (e.g., api-steer's "graceful shutdown with connection draining").

**snippet-location values**:
- `"function_body"` — within any function/method
- `"import_block"` — within import declarations
- `"file_top"` — first 20 lines of file
- `"anywhere"` (default) — anywhere in file

### Required vs Optional

- `--required true` (default): Missing = FAIL
- `--required false`: Missing = SKIP (informational). Useful for checking that something does NOT exist, or for optional patterns.

## CLI Tool Assertions

### Basic Structure

```bash
development-toolchain-validator expectations add cli-tool \
  --connection <skill>:<reference> \
  --command "<command with --json>" \
  --path "<JSONPath expression>" \
  --op <operator> \
  --value <expected value> \
  --description "What this checks"
```

### Common Patterns

#### scenario-auditor: Zero Violations

```bash
development-toolchain-validator expectations add cli-tool \
  --connection api-steer:reference-react-vite \
  --command "scenario-auditor audit reference-react-vite --json" \
  --path "$.total" --op eq --value 0 \
  --description "No auditor violations"
```

#### test-genie: All Phases Pass

```bash
development-toolchain-validator expectations add cli-tool \
  --connection unit-testing-architecture-steer:reference-react-vite \
  --command "test-genie execute reference-react-vite --preset comprehensive --json" \
  --path "$.success" --op eq --value true \
  --timeout 900 \
  --description "All test-genie phases pass"
```

#### scenario-completeness-scoring: Score Threshold

```bash
development-toolchain-validator expectations add cli-tool \
  --connection documentation-health:reference-react-vite \
  --command "scenario-completeness-scoring score reference-react-vite --json" \
  --path "$.score" --op gte --value 96 \
  --description "Completeness score is Production Ready"
```

#### Custom Validation Commands

Any CLI tool that produces JSON output can be used:

```bash
# Check that docs/manifest.json has all required sections
development-toolchain-validator expectations add cli-tool \
  --connection documentation-health:reference-react-vite \
  --command "knowledge-observatory docs audit reference-react-vite --json" \
  --path "$.missing_count" --op eq --value 0 \
  --description "No missing documentation files"
```

### Operator Reference

| Operator | Value Type | Checks |
|----------|-----------|--------|
| `eq` | any | Exact equality |
| `neq` | any | Not equal |
| `gt` | number | Greater than |
| `gte` | number | Greater than or equal |
| `lt` | number | Less than |
| `lte` | number | Less than or equal |
| `exists` | (none) | Path exists in JSON |
| `contains` | string | String contains substring |
| `matches` | regex | Regex match on string |
| `between` | [min,max] | Inclusive numeric range |

### Timeout Configuration

Some tools take longer to execute. Set per-assertion timeouts:

```bash
--timeout 900  # 15 minutes for test-genie
--timeout 240  # 4 minutes for scenario-auditor
--timeout 120  # 2 minutes for completeness scoring
```

Default timeout is 60 seconds.

## Guidelines for Good Assertions

### Do

- **Assert observable properties**: Things you can see in the filesystem or tool output.
- **Use stable paths**: Avoid paths that change with every build (timestamps, random IDs).
- **Add descriptions**: Every assertion should explain why it exists, not just what it checks.
- **Start simple**: Begin with folder structure and tool output, add snippet checks later.
- **Match the skill's language**: If the skill says "organize by domain", assert for domain folders.

### Don't

- **Don't assert implementation details**: Check structure, not specific variable names or line numbers.
- **Don't use mutable commands**: Every command must be read-only and idempotent.
- **Don't over-specify**: If a skill says "handlers should be short", you can't easily assert that. Focus on structural patterns that are checkable.
- **Don't duplicate tool checks**: If scenario-auditor already has a rule for something, use a CLI assertion against auditor output rather than reinventing the check.

### Recognizing Unassertable Skills

Some steer skill guidance cannot be expressed as assertions. Examples:
- "Handlers should orchestrate, not implement" (semantic — requires code understanding)
- "Error messages should be human-readable" (subjective)
- "APIs should feel professional and boring" (aesthetic judgment)

This is expected. The skill maturity score reflects this: skills with assertable guidance score higher. Over time, CLI tools may be built that encode these semantic checks programmatically.
