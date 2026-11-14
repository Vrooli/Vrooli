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

## Durable runs and typed evidence (Connect-RPC)

`RunsService` is the canonical programmatic surface for lifecycle-managed suite
runs. Its procedure paths are generated from
`packages/proto/schemas/test-genie/v1/runs/runs.proto`; callers should use the
generated client or the `test-genie runs` CLI instead of constructing REST
polling loops.

| RPC group | Methods | Contract |
| --- | --- | --- |
| Lifecycle | `StartRun`, `FollowRun`, `WaitRun`, `AbortRun`, `GetRunStatus` | The server owns execution. A client disconnect only detaches. Terminal `WaitRun` projects the same persisted snapshot as `GetRun`. |
| History/retention | `ListRuns`, `GetRun`, `DeleteRun`, `PinRun`, `UnpinRun`, `FindRun` | Run IDs are durable; owner-scoped pins are idempotent and protect retained evidence. |
| Comparison | `CompareRuns`, `CompareRunVisuals` | Joins immutable phase keys from each run's captured descriptor snapshot and returns typed comparability reasons. Visual deltas are advisory. |
| Evidence | `ListRunArtifacts`, `GetRunArtifact`, `GetRunFindings` | Returns path-free typed metadata and opaque run-scoped IDs. Artifact bytes stream only through the validated opaque HTTP route. |
| Compatibility | `GetPhaseArtifact`, `ListRunVideos`, `ListRunVisuals` | Retained for older callers; new consumers use the typed artifact catalog and never filter by producer phase. The legacy relative-path artifact HTTP route has been removed. |

New runs persist three coordinated durable records: the canonical terminal
snapshot, the planning-time descriptor snapshot, and the runtime evidence
catalog. Each carries schema/digest metadata. Missing legacy fields, digest
failure, corrupt snapshots, or absent bytes produce explicit degraded/errors;
they are never converted into an empty successful run.

The opaque byte route is:

```text
GET /scenarios/{scenario}/runs/{runId}/artifacts/{artifactId}
```

The ID is valid only for its owning run. The handler rejects traversal,
cross-run reuse, and symlink escape and applies content-type, no-sniff, and
active-content sandbox headers.

---

## Remediation jobs

Remediation starts from immutable execution evidence; it never accepts a
free-form agent prompt or runtime-policy controls.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/scenarios/{name}/remediation/plans/{executionId}` | Build stable-ID findings, requirement evidence, and cohesive bundles from a completed execution. |
| `POST` | `/api/v1/scenarios/{name}/remediation/jobs` | Create and launch the one allowed active remediation job for the scenario. |
| `GET` | `/api/v1/scenarios/{name}/remediation/jobs` | List durable job history and verification deltas. |
| `GET` | `/api/v1/scenarios/{name}/remediation/jobs/{id}` | Read a job. |
| `POST` | `/api/v1/scenarios/{name}/remediation/jobs/{id}/cancel` | Cancel an active job. |
| `POST` | `/api/v1/scenarios/{name}/remediation/jobs/{id}/agent-status` | Reconcile Agent Manager's terminal status without resolving findings. |
| `POST` | `/api/v1/scenarios/{name}/remediation/jobs/{id}/verify` | Start a server-owned verification rerun and persist its stable-ID delta. |

Create payload:

```json
{
  "sourceExecutionId": "550e8400-e29b-41d4-a716-446655440000",
  "findingIds": ["afid:contracts:missing-version"],
  "requirementIds": ["REQ-001"],
  "roleRef": "code.default",
  "additionalContext": "Keep the public API compatible."
}
```

`prompts`, sandbox/tool/network settings, and provider runtime parameters are
rejected. Agent Manager remains the policy owner.

## Test Execution

### POST /api/v1/executions/plan

Resolve the actual phase plan and timing guidance for a request without running any tests.

**Request Body:**
```json
{
  "scenarioName": "my-scenario",
  "target": "package:api-core",
  "preset": "comprehensive",
  "phases": ["structure", "unit", "integration"],
  "skip": ["performance"],
  "failFast": false
}
```

**Parameters:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `scenarioName` | string | Yes* | - | Legacy bare scenario slug or display alias |
| `target` | string | Yes* | - | First-class target expression (`kind:id`); takes precedence when present |
| `preset` | string | No | `""` | Preset to expand before skip filters |
| `phases` | string[] | No | all enabled phases | Explicit phase list; overrides `preset` |
| `skip` | string[] | No | `[]` | Requested phase exclusions |
| `failFast` | bool | No | `false` | Included for parity with the execute request surface |
| `uiUrl` / `apiUrl` | string | No | auto-detected when possible | Runtime overrides for phases that depend on running services |
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
      "name": "workflow",
      "description": "Delegates BAS workflow validation and safe execution to workflow-health.",
      "optional": false,
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
    "Phase 'workflow' is globally disabled and was skipped by default."
  ]
}
```

**Estimate Source Values:**
| Value | Meaning |
|-------|---------|
| `scenario_history` | Exact same-scenario full-run history, or conservative same-scenario phase history when the summary says `additive_phase_history` |
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
- A summary with `estimateMode: "comparable_full_run"` is a P90 exact-shape
  full-run estimate and already includes startup/orchestration. A summary with
  `estimateMode: "additive_phase_history"` is a lower-confidence P90 phase sum
  plus `orchestrationOverheadSeconds`; it is not comparable-run evidence.
- Failed and timed-out elapsed runs remain timing evidence, and stale samples
  are conservatively penalized. Descriptor/configuration or phase-set changes
  intentionally invalidate comparable-run confidence.
- `timeoutSeconds` always reflects the configured runtime budget after scenario overrides are applied.
- The planner uses the same scenario-aware phase selection logic as actual execution.

**Errors:**
| Code | Cause |
|------|-------|
| 400 | Missing scenarioName, invalid preset/phase, malformed UUID |
| 500 | Planning service unavailable |

---

### POST /api/v1/executions

Execute a test suite for a validation target. This is the primary endpoint for running tests.

**Request Body:**
```json
{
  "scenarioName": "my-scenario",
  "target": "package:api-core",
  "preset": "comprehensive",
  "phases": ["structure", "dependencies", "unit"],
  "skip": ["performance"],
  "failFast": false
}
```

**Parameters:**

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `scenarioName` | string | Yes* | - | Legacy bare scenario slug or display alias |
| `target` | string | Yes* | - | First-class target expression (`kind:id`); takes precedence when present |
| `preset` | string | No | `""` | Preset configuration |
| `phases` | string[] | No | all | Phases to run |
| `skip` | string[] | No | `[]` | Phases to skip |
| `failFast` | bool | No | `false` | Stop on first failure |

**Available Phases:**

The live catalog is descriptor-backed. Use `GET /api/v1/phases` or
`test-genie phases list` for the exact phase set and provider metadata. Common
provider-backed phases include `structure`, `contracts`, `ui-health`, `api`,
`architecture`, `dependencies`, `quality`, `docs`, `unit`, `storage`,
`workflow`, `business`, `performance`, `tidiness`, `security`, `measures`,
`proto`, `branding`, and `search`.

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
  "plannedPhases": ["structure", "contracts", "ui-health", "api", "architecture", "dependencies", "quality", "docs", "performance", "unit", "storage", "workflow", "business", "tidiness", "security", "measures", "proto", "branding", "search"],
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
| 400 | Missing target/scenarioName, invalid preset/phase |
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
      "plannedPhases": ["structure", "contracts", "ui-health", "api", "architecture", "dependencies", "quality", "docs", "performance", "unit", "storage", "workflow", "business", "tidiness", "security", "measures", "proto", "branding", "search"],
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

Returns the provider-backed phase catalog with descriptions and configuration.

**Response (abbreviated):**
```json
{
  "items": [
    {
      "name": "structure",
      "optional": false,
      "description": "Validates scenario layout, manifests, and JSON health before deeper checks run.",
      "source": "validation-provider",
      "defaultTimeoutSeconds": 900
    },
    {
      "name": "api",
      "optional": false,
      "description": "Delegates API readiness validation to api-health through ScenarioValidationService.",
      "source": "validation-provider",
      "defaultTimeoutSeconds": 120
    },
    {
      "name": "workflow",
      "optional": false,
      "description": "Delegates BAS workflow validation and safe execution to workflow-health.",
      "source": "validation-provider",
      "defaultTimeoutSeconds": 900
    }
  ],
  "count": 19
}
```

The full catalog is descriptor-backed. Use `GET /api/v1/phases` for the current
list; current in-repo descriptors include `structure`, `contracts`, `ui-health`,
`api`, `architecture`, `dependencies`, `quality`, `docs`, `performance`, `unit`,
`storage`, `workflow`, `business`, `tidiness`, `security`, `measures`, `proto`,
`branding`, and `search`.

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
| `invalid remediation selector` | An unknown source execution, finding, or requirement was selected | Reload the execution plan and select an offered stable ID |
| `an active remediation job already exists` | The scenario already has work in progress | Observe, cancel, or verify the existing job first |
| `execution service unavailable` | Service not running | Check test-genie status |

---

## Rate Limiting

No rate limiting is currently enforced for local deployments.

---

## Example Workflows

### Execute, Remediate, and Verify

```bash
# 1. Preview and execute a scenario suite
curl -X POST http://localhost:8080/api/v1/executions/plan \
  -H "Content-Type: application/json" \
  -d '{
    "scenarioName": "my-scenario",
    "preset": "comprehensive"
  }'

# 2. Use its completed execution UUID to create a remediation job.
curl -X POST http://localhost:8080/api/v1/executions \
  -H "Content-Type: application/json" \
  -d '{
    "scenarioName": "my-scenario",
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
# "contracts"
# "ui-health"
# "api"
# "architecture"
# "dependencies"
# "quality"
# "docs"
# "unit"
# "workflow"
# "business"
# "performance"
```

---

## See Also

- [CLI Commands](cli-commands.md) - CLI equivalents for all API operations
- [Execution Configuration](configuration.md) - Timeouts, planning, and estimate behavior
- [Server-Owned Execution Guide](../guides/sync-execution.md) - Durable execution and re-attach protocol
- [Presets Reference](presets.md) - Preset definitions
