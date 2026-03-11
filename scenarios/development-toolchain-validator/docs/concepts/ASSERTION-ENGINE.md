# Assertion Engine

## Overview

The assertion engine is the core validation component of DTV. It evaluates two types of assertions against reference scenarios:

1. **Structural assertions**: Filesystem checks (folders, files, snippets)
2. **CLI tool assertions**: Subprocess execution with JSON output evaluation

Both types produce pass/fail results with detailed failure messages.

## Structural Assertion Engine

### Folder Checks

For each folder expectation:
1. Resolve the path relative to the reference scenario root
2. Check if the directory exists using `os.Stat`
3. If `required=true` and missing → FAIL with path details
4. If `required=false` and missing → SKIP (informational)

### File Glob Checks

For each file expectation:
1. Resolve the glob pattern relative to the reference scenario root
2. Execute glob matching using `filepath.Glob` or `doublestar` for `**` patterns
3. If `required=true` and no matches → FAIL with pattern details
4. Count matches for the report

### Snippet Checks

For each snippet expectation:
1. Read the target file
2. Search for `snippet_content` in the file
3. If `snippet_location` is specified:
   - `"function_body"`: Search within function/method bodies
   - `"import_block"`: Search within import declarations
   - `"file_top"`: Search in the first N lines
   - `null`/`"anywhere"`: Search entire file
4. If `required=true` and not found → FAIL with context

### Error Handling

- File read errors (permission, not found) → FAIL with error details
- Glob pattern errors → FAIL with pattern details
- All errors include the expectation description for debugging

## CLI Tool Assertion Engine

### Execution Flow

```
1. Parse command string
2. Resolve scenario name in command (replace {reference} with actual path)
3. Execute via os/exec with:
   - Working directory: Vrooli project root
   - Timeout: configurable (default 60s, some tools need more)
   - Environment: inherit current environment
4. Capture stdout and stderr
5. Parse stdout as JSON
6. Evaluate JSONPath expression against parsed JSON
7. Apply operator with expected value
8. Return pass/fail with actual vs expected values
```

### JSONPath Evaluation

DTV uses a JSONPath library to extract values from JSON output. The extraction handles:

- **Simple paths**: `$.score` → extracts the "score" field from root object
- **Nested paths**: `$.breakdown.quality.score` → traverses nested objects
- **Array access**: `$.phases[0].status` → extracts from array by index
- **Missing paths**: Returns a "path not found" error (which `exists` operator checks for)

### Operator Implementation

```go
// Conceptual implementation
func evaluate(actual interface{}, op string, expected interface{}) (bool, error) {
    switch op {
    case "eq":
        return reflect.DeepEqual(actual, expected), nil
    case "neq":
        return !reflect.DeepEqual(actual, expected), nil
    case "gt":
        return toFloat64(actual) > toFloat64(expected), nil
    case "gte":
        return toFloat64(actual) >= toFloat64(expected), nil
    case "lt":
        return toFloat64(actual) < toFloat64(expected), nil
    case "lte":
        return toFloat64(actual) <= toFloat64(expected), nil
    case "exists":
        return actual != nil, nil // path was found
    case "contains":
        return strings.Contains(toString(actual), toString(expected)), nil
    case "matches":
        return regexp.MatchString(toString(expected), toString(actual))
    case "between":
        bounds := expected.([]interface{})
        val := toFloat64(actual)
        return val >= toFloat64(bounds[0]) && val <= toFloat64(bounds[1]), nil
    }
}
```

### Timeout Handling

Different tools have different execution times:

| Tool | Typical Duration | Recommended Timeout |
|------|-----------------|-------------------|
| `scenario-auditor audit --json` | 10-60s | 240s |
| `test-genie execute --json` | 2-15min | 900s |
| `scenario-completeness-scoring score --json` | 5-30s | 120s |
| Custom validation commands | varies | 60s (default) |

Timeouts are configurable per assertion. When a timeout occurs, the assertion FAILs with a timeout error rather than hanging.

### Safety Constraints

1. **Read-only commands only**: DTV should never execute commands that modify files, start/stop services, or change state.
2. **No shell injection**: Commands are parsed and executed via `os/exec`, not through a shell. Command strings are split into arguments safely.
3. **Resource limits**: Subprocess output is capped to prevent memory issues from unexpectedly large output.
4. **No credential exposure**: CLI tool assertions should not contain secrets. If a tool requires authentication, it should use environment variables or config files.

## Report Generation

After all assertions run, the engine generates a comprehensive report:

```json
{
  "reference": "reference-react-vite",
  "run_at": "2026-03-11T15:30:00Z",
  "connections": [
    {
      "skill_id": "api-steer",
      "skill_version": 49,
      "drift_detected": false,
      "structural": {
        "pass": 5,
        "fail": 0,
        "skip": 1,
        "results": [...]
      },
      "cli_tools": {
        "pass": 3,
        "fail": 0,
        "results": [...]
      }
    }
  ],
  "overlaps": [
    {
      "path": "api/handlers/",
      "skills": ["api-steer", "unit-testing-architecture-steer"],
      "expectations": [...]
    }
  ],
  "conflicts": [],
  "unconfigured_skills": ["polish", "security"],
  "summary": {
    "total_connections": 10,
    "configured": 8,
    "unconfigured": 2,
    "all_passing": true,
    "structural_pass": 40,
    "structural_fail": 0,
    "cli_pass": 15,
    "cli_fail": 0,
    "overlaps": 3,
    "conflicts": 0,
    "drifted": 1
  }
}
```
