# API Reference

## Base URL

`http://localhost:${API_PORT}/api/v1`

## Health

### GET /health
Returns API health status.

**Response**: `200 OK`
```json
{
  "status": "healthy",
  "version": "0.0.1",
  "uptime_seconds": 3600
}
```

## References

### GET /references
List all registered reference scenarios.

**Response**: `200 OK`
```json
{
  "references": [
    {
      "id": "uuid",
      "name": "reference-react-vite",
      "template": "react-vite",
      "path": "/home/user/Vrooli/scenarios/reference-react-vite",
      "created_at": "2026-03-11T00:00:00Z",
      "connection_count": 10,
      "configured_count": 8
    }
  ]
}
```

### POST /references
Register a new reference scenario.

**Request**:
```json
{
  "name": "reference-react-vite",
  "template": "react-vite"
}
```

**Response**: `201 Created`

### GET /references/{name}
Get details for a specific reference.

### DELETE /references/{name}
Remove a reference and all its connections.

## Skill Connections

### GET /connections?reference={name}
List all skill connections for a reference.

**Response**: `200 OK`
```json
{
  "connections": [
    {
      "id": "uuid",
      "reference_name": "reference-react-vite",
      "skill_id": "api-steer",
      "skill_version": 49,
      "skill_content_hash": "sha256...",
      "connected_at": "2026-03-11T00:00:00Z",
      "drift_detected": false,
      "structural_expectations_count": 5,
      "cli_assertions_count": 3
    }
  ]
}
```

### POST /connections
Create a new skill connection. Fetches current version/hash from prompt-manager.

**Request**:
```json
{
  "reference_name": "reference-react-vite",
  "skill_id": "api-steer"
}
```

### DELETE /connections/{id}
Remove a skill connection and all its expectations.

### POST /connections/{id}/refresh
Update the stored version/hash to current values from prompt-manager. Use after reviewing drift.

## Expectations

### GET /connections/{id}/expectations
List all expectations (structural + CLI tool) for a connection.

### POST /connections/{id}/expectations/structural
Add a structural expectation.

**Request**:
```json
{
  "type": "folder",
  "path": "api/handlers/projects/",
  "required": true,
  "description": "Projects domain module"
}
```

For snippets:
```json
{
  "type": "snippet",
  "path": "api/main.go",
  "snippet_content": "gracefulShutdown(",
  "snippet_location": "function_body",
  "required": true,
  "description": "Server implements graceful shutdown"
}
```

### POST /connections/{id}/expectations/cli-tool
Add a CLI tool assertion.

**Request**:
```json
{
  "command": "scenario-auditor audit reference-react-vite --json",
  "json_path": "$.total",
  "operator": "eq",
  "expected_value": 0,
  "timeout_seconds": 240,
  "description": "No auditor violations"
}
```

### DELETE /expectations/{id}
Remove a specific expectation.

## Validation

### POST /validate/{reference}
Run all validations for a reference scenario. Returns a comprehensive report.

**Response**: `200 OK`
```json
{
  "reference": "reference-react-vite",
  "run_at": "2026-03-11T15:30:00Z",
  "connections": [...],
  "overlaps": [...],
  "conflicts": [...],
  "unconfigured_skills": [...],
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

### GET /validate/{reference}/history
Get validation run history.

**Query params**: `?limit=10`

## Drift

### GET /drift/{reference}
Check drift status for all connections on a reference.

**Response**: `200 OK`
```json
{
  "reference": "reference-react-vite",
  "connections": [
    {
      "skill_id": "api-steer",
      "drift_detected": false,
      "stored_version": 49,
      "current_version": 49
    },
    {
      "skill_id": "cli-steer",
      "drift_detected": true,
      "stored_version": 12,
      "current_version": 15,
      "versions_behind": 3
    }
  ]
}
```

## Baselines [P1]

### POST /baselines/{reference}
Run all tooling baselines for a reference.

**Query params**: `?tool=auditor|test-genie|completeness` (optional, run specific tool only)

### GET /baselines/{reference}/history
Get baseline run history.

### PUT /baselines/{reference}/config
Configure baseline expectations.

## Coverage Map [P1]

### GET /coverage/{reference}
Get the file/folder coverage map showing which skills have expectations for each area.

## Maturity Scores [P1]

### GET /maturity/{reference}
Get maturity scores for all connected skills.

## Error Format

All errors follow a consistent shape:

```json
{
  "error": {
    "code": "REFERENCE_NOT_FOUND",
    "message": "Reference scenario 'foo' is not registered",
    "details": {
      "reference": "foo"
    },
    "request_id": "uuid"
  }
}
```
