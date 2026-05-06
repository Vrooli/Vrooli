# Test Genie API Reference

Complete reference for the Test Genie REST API. All endpoints are designed for both human and agent consumption.

## Base URL

```
http://localhost:{API_PORT}/api/v1
```

Get the API port dynamically:
```bash
API_PORT=$(vrooli scenario port test-genie API_PORT)
curl http://localhost:${API_PORT}/health
```

## Authentication

Currently, the API does not require authentication for local access. Future versions may add token-based authentication for remote deployments.

## Response Format

All responses are JSON. Successful responses include the requested data directly. Error responses follow this format:

```json
{
  "error": "Error message",
  "details": "Additional context (optional)"
}
```

---

## Health & Status

### GET /health

Returns infrastructure readiness plus operational telemetry.

**Response:**
```json
{
  "status": "healthy",
  "service": "Test Genie API",
  "version": "1.0.0",
  "readiness": true,
  "timestamp": "2025-01-15T10:30:00Z",
  "dependencies": {
    "database": "connected"
  },
  "operations": {
    "queue": {
      "total": 15,
      "queued": 2,
      "delegated": 1,
      "running": 1,
      "completed": 10,
      "failed": 1,
      "pending": 3,
      "timestamp": "2025-01-15T10:30:00Z",
      "oldestQueuedAt": "2025-01-15T10:25:00Z",
      "oldestQueuedAgeSeconds": 300
    },
    "lastExecution": {
      "executionId": "550e8400-e29b-41d4-a716-446655440000",
      "scenario": "my-scenario",
      "success": true,
      "completedAt": "2025-01-15T10:28:00Z",
      "startedAt": "2025-01-15T10:25:00Z",
      "phaseSummary": {
        "total": 11,
        "passed": 11,
        "failed": 0,
        "durationSeconds": 188,
        "observationCount": 24
      },
      "preset": "comprehensive"
    }
  }
}
```

**Status Values:**
| Value | Description |
|-------|-------------|
| `healthy` | All dependencies operational |
| `unhealthy` | One or more dependencies failing |

**Use Cases:**
- Health checks in load balancers
- Monitoring dashboards
- Agent readiness checks before test execution

---

## Suite Requests

Suite requests represent queued test generation jobs that may be delegated to downstream AI agents.

### POST /api/v1/suite-requests

Create a new test suite generation request.

**Request Body:**
```json
{
  "scenarioName": "my-scenario",
  "requestedTypes": ["unit", "integration"],
  "coverageTarget": 90,
  "priority": "normal",
  "notes": "Focus on API handler coverage"
}
```

**Parameters:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `scenarioName` | string | Yes | - | Target scenario name |
| `requestedTypes` | string[] | No | `["unit", "integration"]` | Test types to generate |
| `coverageTarget` | int | No | `95` | Target coverage percentage (1-100) |
| `priority` | string | No | `"normal"` | Queue priority |
| `notes` | string | No | `""` | Additional context for generation |

**Allowed Values:**

| Field | Values |
|-------|--------|
| `requestedTypes` | `unit`, `integration`, `performance`, `vault`, `regression` |
| `priority` | `low`, `normal`, `high`, `urgent` |

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "scenarioName": "my-scenario",
  "requestedTypes": ["unit", "integration"],
  "coverageTarget": 90,
  "priority": "normal",
  "status": "queued",
  "notes": "Focus on API handler coverage",
  "createdAt": "2025-01-15T10:30:00Z",
  "updatedAt": "2025-01-15T10:30:00Z",
  "estimatedQueueTimeSeconds": 150
}
```

**Errors:**
| Code | Cause |
|------|-------|
| 400 | Invalid JSON, missing scenarioName, invalid type/priority |

---

### GET /api/v1/suite-requests

List recent suite requests.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 50 | Maximum results (max 50) |

**Response:**
```json
{
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "scenarioName": "my-scenario",
      "requestedTypes": ["unit", "integration"],
      "coverageTarget": 90,
      "priority": "normal",
      "status": "completed",
      "createdAt": "2025-01-15T10:30:00Z",
      "updatedAt": "2025-01-15T10:35:00Z"
    }
  ],
  "count": 1
}
```

**Status Values:**
| Status | Description |
|--------|-------------|
| `queued` | Waiting in queue |
| `delegated` | Assigned to downstream agent |
| `running` | Currently executing |
| `completed` | Successfully finished |
| `failed` | Execution failed |

---

### GET /api/v1/suite-requests/{id}

Get a specific suite request by ID.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Suite request identifier |

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "scenarioName": "my-scenario",
  "requestedTypes": ["unit", "integration"],
  "coverageTarget": 90,
  "priority": "normal",
  "status": "running",
  "notes": "Focus on API handler coverage",
  "delegationIssueId": "12345",
  "createdAt": "2025-01-15T10:30:00Z",
  "updatedAt": "2025-01-15T10:32:00Z",
  "estimatedQueueTimeSeconds": 150
}
```

**Errors:**
| Code | Cause |
|------|-------|
| 400 | Invalid UUID format |
| 404 | Suite request not found |

---

## Test Execution

### POST /api/v1/executions/plan

Resolve the actual phase plan and timing guidance for a request without running any tests.

**Request Body:**
```json
{
  "scenarioName": "my-scenario",
  "preset": "comprehensive",
  "phases": ["structure", "unit", "integration"],
  "skip": ["performance"],
  "failFast": false
}
```

**Parameters:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `scenarioName` | string | Yes | - | Target scenario |
| `preset` | string | No | `""` | Preset to expand before skip filters |
| `phases` | string[] | No | all enabled phases | Explicit phase list; overrides `preset` |
| `skip` | string[] | No | `[]` | Requested phase exclusions |
| `failFast` | bool | No | `false` | Included for parity with the execute request surface |
| `suiteRequestId` | UUID | No | - | Accepted but ignored by the planner |
| `uiUrl` / `apiUrl` / `browserlessUrl` | string | No | auto-detected when possible | Runtime overrides for phases that depend on running services |
| `scenarioPath` | string | No | resolved from scenario name | Absolute physical scenario directory to read and write |
| `logicalRepoRoot` | string | No | none | Absolute repo root for repo-relative validation |
| `logicalScenarioRelPath` | string | No | none | Scenario directory relative to `logicalRepoRoot` |

Provide `logicalRepoRoot` and `logicalScenarioRelPath` together when the
physical `scenarioPath` is a temporary copy that should be validated as if it
lived under a real repo. Omit them when `scenarioPath` is the authoritative
location.

**Response:**
```json
{
  "scenarioName": "my-scenario",
  "presetUsed": "comprehensive",
  "phases": [
    {
      "name": "structure",
      "description": "Validates scenario layout, manifests, and JSON health before deeper checks run.",
      "optional": false,
      "estimatedDurationSeconds": 4,
      "timeoutSeconds": 900,
      "estimateSource": "scenario_history",
      "estimateConfidence": "high",
      "estimateSampleSize": 12
    },
    {
      "name": "playbooks",
      "description": "Executes Vrooli Ascension workflows declared under bas/.",
      "optional": true,
      "estimatedDurationSeconds": 900,
      "timeoutSeconds": 900,
      "estimateSource": "timeout_fallback",
      "estimateConfidence": "low",
      "estimateSampleSize": 0
    }
  ],
  "summary": {
    "phaseCount": 11,
    "estimatedDurationSeconds": 246,
    "timeoutSeconds": 6240
  },
  "warnings": [
    "Phase 'playbooks' is globally disabled and was skipped by default."
  ]
}
```

**Estimate Source Values:**
| Value | Meaning |
|-------|---------|
| `scenario_history` | Derived from recent runs of the same scenario and phase |
| `blended_history` | Weighted blend of scenario history and global phase history |
| `global_history` | Derived from recent runs of the phase across all scenarios |
| `timeout_fallback` | No useful runtime history was available; uses the timeout budget |

**Estimate Confidence Values:**
| Value | Meaning |
|-------|---------|
| `high` | Strong scenario-specific evidence |
| `medium` | Useful but still somewhat sparse history |
| `low` | Weak or no history; treat as rough guidance |

**Notes:**
- Estimates are phase-level medians from recent history, not timeout budgets.
- `timeoutSeconds` always reflects the configured runtime budget after scenario overrides are applied.
- The planner uses the same scenario-aware phase selection logic as actual execution.

**Errors:**
| Code | Cause |
|------|-------|
| 400 | Missing scenarioName, invalid preset/phase, malformed UUID |
| 500 | Planning service unavailable |

---

### POST /api/v1/executions

Execute a test suite for a scenario. This is the primary endpoint for running tests.

**Request Body:**
```json
{
  "scenarioName": "my-scenario",
  "suiteRequestId": "550e8400-e29b-41d4-a716-446655440000",
  "preset": "comprehensive",
  "phases": ["structure", "dependencies", "unit"],
  "skip": ["performance"],
  "failFast": false
}
```

**Parameters:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `scenarioName` | string | Yes | - | Target scenario |
| `suiteRequestId` | UUID | No | - | Link to queued request |
| `preset` | string | No | `""` | Preset configuration |
| `phases` | string[] | No | all | Phases to run |
| `skip` | string[] | No | `[]` | Phases to skip |
| `failFast` | bool | No | `false` | Stop on first failure |

**Available Phases:**
1. `structure` - Validates scenario layout, manifests, and JSON health
2. `standards` - Runs scenario-auditor standards enforcement
3. `dependencies` - Confirms required commands, runtimes, and declared resources
4. `lint` - Runs type checking and linting
5. `docs` - Validates markdown, mermaid, and links
6. `smoke` - Performs fast UI / runtime handshake validation
7. `unit` - Runs Go, Node, Python, and shell unit suites as applicable
8. `integration` - Exercises CLI/Bats suites and scenario-local integrations
9. `playbooks` - Executes BAS browser automation workflows
10. `business` - Audits requirement coverage and operational targets
11. `performance` - Builds binaries and checks duration/performance budgets

Use `POST /api/v1/executions/plan` to see the actual selected phases, runtime estimate, and timeout budget before executing.

**Response:**
```json
{
  "executionId": "660e8400-e29b-41d4-a716-446655440001",
  "scenarioName": "my-scenario",
  "startedAt": "2025-01-15T10:30:00Z",
  "completedAt": "2025-01-15T10:35:00Z",
  "success": true,
  "preset": "comprehensive",
  "requestedPreset": "comprehensive",
  "requestedSkipPhases": ["performance"],
  "plannedPhases": ["structure", "standards", "dependencies", "lint", "docs", "smoke", "unit", "integration", "playbooks", "business"],
  "failFast": false,
  "phases": [
    {
      "name": "structure",
      "status": "passed",
      "durationSeconds": 5,
      "logPath": "scenarios/my-scenario/coverage/logs/latest/structure.log",
      "observations": [
        { "text": "All required files present" },
        { "prefix": "SUCCESS", "text": "JSON manifests valid" }
      ]
    },
    {
      "name": "unit",
      "status": "passed",
      "durationSeconds": 45,
      "logPath": "scenarios/my-scenario/coverage/logs/latest/unit.log",
      "observations": [
        { "text": "32 tests passed" },
        { "text": "Coverage: 87%" }
      ]
    }
  ],
  "phaseSummary": {
    "total": 10,
    "passed": 10,
    "failed": 0,
    "durationSeconds": 300,
    "observationCount": 14
  }
}
```

**Phase Status Values:**
| Status | Description |
|--------|-------------|
| `passed` | Phase completed successfully |
| `failed` | Phase failed |
| `failed` with `classification=timeout` | Phase exceeded its timeout budget |

**Errors:**
| Code | Cause |
|------|-------|
| 400 | Missing scenarioName, invalid preset/phase |
| 404 | Suite request not found (if suiteRequestId provided) |
| 500 | Execution service unavailable |

---

### GET /api/v1/executions

List execution history.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `scenario` | string | - | Filter by scenario name |
| `limit` | int | 100 | Maximum results |
| `offset` | int | 0 | Pagination offset |

**Response:**
```json
{
  "items": [
    {
      "executionId": "660e8400-e29b-41d4-a716-446655440001",
      "scenarioName": "my-scenario",
      "success": true,
      "startedAt": "2025-01-15T10:30:00Z",
      "completedAt": "2025-01-15T10:35:00Z",
      "preset": "comprehensive",
      "plannedPhases": ["structure", "standards", "dependencies", "lint", "docs", "smoke", "unit", "integration", "playbooks", "business"],
      "failFast": false
    }
  ],
  "count": 1
}
```

---

### GET /api/v1/executions/{id}

Get detailed execution results.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | UUID | Execution identifier |

**Response:**
Same as POST /api/v1/executions response.

**Errors:**
| Code | Cause |
|------|-------|
| 400 | Invalid UUID format |
| 404 | Execution not found |

---

## Scenarios

### GET /api/v1/scenarios

List all tracked scenarios with their test status.

**Response:**
```json
{
  "items": [
    {
      "name": "my-scenario",
      "path": "/home/user/Vrooli/scenarios/my-scenario",
      "hasTests": true,
      "testTypes": ["unit", "integration"],
      "lastExecuted": "2025-01-15T10:35:00Z",
      "lastStatus": "passed"
    },
    {
      "name": "another-scenario",
      "path": "/home/user/Vrooli/scenarios/another-scenario",
      "hasTests": false,
      "testTypes": [],
      "lastExecuted": null,
      "lastStatus": null
    }
  ],
  "count": 2
}
```

---

### GET /api/v1/scenarios/{name}

Get detailed information about a specific scenario.

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Scenario name |

**Response:**
```json
{
  "name": "my-scenario",
  "path": "/home/user/Vrooli/scenarios/my-scenario",
  "hasTests": true,
  "testTypes": ["unit", "integration", "e2e"],
  "capabilities": {
    "hasGoAPI": true,
    "hasNodeUI": true,
    "hasCLI": true,
    "hasBASWorkflows": true
  },
  "lastExecuted": "2025-01-15T10:35:00Z",
  "lastStatus": "passed",
  "requirementsCoverage": {
    "total": 25,
    "complete": 20,
    "inProgress": 3,
    "pending": 2
  }
}
```

**Errors:**
| Code | Cause |
|------|-------|
| 404 | Scenario not found |

---

### POST /api/v1/scenarios/{name}/tests

Run tests for a specific scenario directly (alternative to /api/v1/executions).

**Path Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `name` | string | Scenario name |

**Request Body (optional):**
```json
{
  "type": "unit"
}
```

**Parameters:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `type` | string | all | Test type to run (unit, integration, etc.) |

**Response:**
```json
{
  "status": "completed",
  "command": {
    "type": "unit",
    "path": "/home/user/Vrooli/scenarios/my-scenario/coverage/phases/test-unit.sh"
  },
  "type": "unit",
  "logPath": "/tmp/test-genie/logs/my-scenario-unit.log"
}
```

**Errors:**
| Code | Cause |
|------|-------|
| 400 | Invalid test type |
| 404 | Scenario not found |

---

## Phase Catalog

### GET /api/v1/phases

Returns the Go-native phase catalog with descriptions and configuration.

**Response (abbreviated):**
```json
{
  "items": [
    {
      "name": "structure",
      "optional": false,
      "description": "Validates scenario layout, manifests, and JSON health before deeper checks run.",
      "source": "native",
      "defaultTimeoutSeconds": 900
    },
    {
      "name": "standards",
      "optional": false,
      "description": "Runs scenario-auditor standards enforcement against the scenario workspace.",
      "source": "native",
      "defaultTimeoutSeconds": 60
    },
    {
      "name": "playbooks",
      "optional": true,
      "description": "Executes Vrooli Ascension workflows declared under bas/.",
      "source": "native",
      "defaultTimeoutSeconds": 900
    }
  ],
  "count": 11
}
```

The full catalog currently includes `structure`, `standards`, `dependencies`, `lint`, `docs`, `smoke`, `unit`, `integration`, `playbooks`, `business`, and `performance`.

---

## Response Codes Summary

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created (POST requests) |
| 400 | Bad Request (invalid parameters, validation error) |
| 404 | Not Found (resource doesn't exist) |
| 500 | Server Error (internal failure) |

---

## Error Response Format

All errors follow this structure:

```json
{
  "error": "Human-readable error message",
  "details": "Additional context (optional)"
}
```

**Common Errors:**

| Error Message | Cause | Solution |
|--------------|-------|----------|
| `scenarioName is required` | Missing required field | Add scenarioName to request |
| `requested type 'X' is not supported` | Invalid test type | Use: unit, integration, performance, vault, regression |
| `priority 'X' is not supported` | Invalid priority | Use: low, normal, high, urgent |
| `suite request id must be a valid UUID` | Invalid ID format | Provide valid UUID |
| `execution service unavailable` | Service not running | Check test-genie status |

---

## Rate Limiting

No rate limiting is currently enforced for local deployments.

---

## Example Workflows

### Generate and Execute Tests

```bash
# 1. Create suite request
curl -X POST http://localhost:8080/api/v1/suite-requests \
  -H "Content-Type: application/json" \
  -d '{"scenarioName": "my-scenario", "requestedTypes": ["unit", "integration"]}'

# Response includes ID: 550e8400-e29b-41d4-a716-446655440000

# 2. Preview the actual plan and timing guidance
curl -X POST http://localhost:8080/api/v1/executions/plan \
  -H "Content-Type: application/json" \
  -d '{
    "scenarioName": "my-scenario",
    "suiteRequestId": "550e8400-e29b-41d4-a716-446655440000",
    "preset": "comprehensive"
  }'

# 3. Execute the suite
curl -X POST http://localhost:8080/api/v1/executions \
  -H "Content-Type: application/json" \
  -d '{
    "scenarioName": "my-scenario",
    "suiteRequestId": "550e8400-e29b-41d4-a716-446655440000",
    "preset": "comprehensive"
  }'

# 4. Check execution status
curl http://localhost:8080/api/v1/executions/660e8400-e29b-41d4-a716-446655440001
```

### Quick Health Check

```bash
curl -s http://localhost:8080/health | jq '.status'
# "healthy"
```

### List Available Phases

```bash
curl -s http://localhost:8080/api/v1/phases | jq '.items[].name'
# "structure"
# "standards"
# "dependencies"
# "lint"
# "docs"
# "smoke"
# "unit"
# "integration"
# "playbooks"
# "business"
# "performance"
```

---

## See Also

- [CLI Commands](cli-commands.md) - CLI equivalents for all API operations
- [Execution Configuration](configuration.md) - Timeouts, planning, and estimate behavior
- [Sync Execution Guide](../guides/sync-execution.md) - Detailed sync endpoint usage
- [Sync Execution Cheatsheet](sync-execution-cheatsheet.md) - Quick reference
- [Presets Reference](presets.md) - Preset definitions
