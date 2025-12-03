# Test Genie API Reference

> **Status**: Placeholder - To be populated with complete API documentation

## Overview

Test Genie provides a REST API for programmatic access to test suite management, execution, and analysis. The API is designed to be agent-friendly for AI-powered automation.

## Base URL

```
http://localhost:{API_PORT}/api/v1
```

Get the API port dynamically:
```bash
API_PORT=$(vrooli scenario port test-genie API_PORT)
```

## Authentication

Currently, the API does not require authentication for local access. Future versions may add token-based authentication.

## Core Endpoints

### Health Check

```http
GET /health
```

Returns infrastructure readiness plus operational telemetry.

**Response:**
```json
{
  "status": "healthy",
  "operations": {
    "queue": {
      "depth": 3,
      "pending": 2,
      "failed": 1,
      "oldestAge": "5m"
    },
    "lastExecution": {
      "id": "...",
      "status": "completed",
      "completedAt": "..."
    }
  }
}
```

### Suite Requests

#### Create Suite Request
```http
POST /api/v1/suite-requests
Content-Type: application/json

{
  "scenarioName": "my-scenario",
  "requestedTypes": ["unit", "integration"],
  "coverageTarget": 90,
  "priority": "normal",
  "notes": "Optional notes"
}
```

#### List Suite Requests
```http
GET /api/v1/suite-requests
GET /api/v1/suite-requests?scenario=my-scenario
```

#### Get Suite Request
```http
GET /api/v1/suite-requests/{id}
```

### Test Execution

#### Execute Suite (Async)
```http
POST /api/v1/test-suite/{suite_id}/execute
Content-Type: application/json

{
  "execution_type": "full",
  "environment": "local",
  "timeout_seconds": 600
}
```

#### Execute Suite (Sync)
```http
POST /api/v1/test-suite/{suite_id}/execute-sync
Content-Type: application/json

{
  "execution_type": "full",
  "environment": "local",
  "timeout_seconds": 600
}
```

Returns complete results in a single request. See [Sync Execution Guide](../guides/sync-execution.md).

### Execution History

#### List Executions
```http
GET /api/v1/executions
GET /api/v1/executions?scenario=my-scenario&limit=10
```

#### Get Execution Details
```http
GET /api/v1/executions/{id}
```

### Phase Catalog

#### List Phases
```http
GET /api/v1/phases
```

Returns the Go-native phase catalog with descriptions and optionality flags.

### Scenario Catalog

#### List Scenarios
```http
GET /api/v1/scenarios
```

Returns all tracked scenarios with their test status.

## Response Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request (invalid parameters) |
| 404 | Not Found |
| 500 | Server Error |

## Error Format

```json
{
  "error": "Error message",
  "details": "Additional context"
}
```

## Rate Limiting

No rate limiting is currently enforced for local deployments.

## See Also

- [Sync Execution Guide](../guides/sync-execution.md) - Detailed sync endpoint usage
- [Sync Execution Cheatsheet](sync-execution-cheatsheet.md) - Quick reference
- [CLI Commands](cli-commands.md) - CLI equivalents
