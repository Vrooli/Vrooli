# API Reference

Complete REST API endpoint documentation for Scenario-to-Cloud.

## Base URL

All endpoints are prefixed with `/api/v1`.

## Health

### GET /health

Check API health status.

**Response**:
```json
{
  "status": "ok",
  "service": "scenario-to-cloud",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Manifest

### POST /manifest/validate

Validate a deployment manifest.

**Request Body**:
```json
{
  "scenario": { "id": "my-scenario" },
  "target": { "vps": { "host": "example.com" } },
  "edge": { "domain": "app.example.com" }
}
```

**Response**:
```json
{
  "valid": true,
  "issues": [],
  "manifest": { ... },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Plan

### POST /plan

Generate a deployment plan.

**Request Body**: Same as `/manifest/validate`

**Response**:
```json
{
  "plan": [
    { "id": "build", "title": "Build Bundle", "description": "..." },
    { "id": "transfer", "title": "Transfer to VPS", "description": "..." }
  ],
  "issues": [],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Bundle

### POST /bundle/build

Build a deployment bundle.

**Request Body**: Same as `/manifest/validate`

**Response**:
```json
{
  "artifact": {
    "path": "/path/to/bundle.tar.gz",
    "sha256": "abc123...",
    "size_bytes": 15000000
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Preflight

### POST /preflight

Run preflight checks against VPS.

**Request Body**: Same as `/manifest/validate`

**Response**:
```json
{
  "ok": true,
  "checks": [
    {
      "id": "ssh_connectivity",
      "title": "SSH Connectivity",
      "status": "pass",
      "details": "Connected successfully"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Deployments

### GET /deployments

List all deployments.

**Response**:
```json
{
  "deployments": [
    {
      "id": "uuid",
      "name": "my-scenario @ example.com",
      "scenario_id": "my-scenario",
      "status": "deployed",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments

Create a new deployment.

**Request Body**:
```json
{
  "manifest": { ... },
  "name": "Optional custom name"
}
```

**Response**:
```json
{
  "deployment": { ... },
  "created": true,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}

Get deployment details.

**Response**:
```json
{
  "deployment": {
    "id": "uuid",
    "name": "my-scenario @ example.com",
    "status": "deployed",
    "manifest": { ... },
    "setup_result": { ... },
    "deploy_result": { ... },
    "created_at": "2024-01-15T10:30:00Z",
    "last_deployed_at": "2024-01-15T10:35:00Z"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### DELETE /deployments/{id}

Delete a deployment.

**Query Parameters**:
- `stop=true` - Also stop the scenario on VPS

**Response**:
```json
{
  "deleted": true,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/execute

Execute deployment (setup + deploy).

**Response**:
```json
{
  "deployment": { ... },
  "success": true,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/inspect

Inspect deployment status on VPS.

**Response**:
```json
{
  "result": {
    "ok": true,
    "scenario_status": { ... },
    "scenario_logs": "...",
    "timestamp": "2024-01-15T10:30:00Z"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/stop

Stop deployment on VPS.

**Response**:
```json
{
  "success": true,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Documentation

### GET /docs/manifest

Get the documentation manifest.

**Response**:
```json
{
  "version": "1.0.0",
  "title": "Scenario-to-Cloud Documentation",
  "defaultDocument": "QUICKSTART.md",
  "sections": [ ... ]
}
```

### GET /docs/content

Get document content.

**Query Parameters**:
- `path` - Document path (e.g., `QUICKSTART.md`)

**Response**:
```json
{
  "path": "QUICKSTART.md",
  "content": "# Quick Start Guide\n..."
}
```

## Error Responses

All endpoints return errors in this format:

```json
{
  "error": {
    "code": "error_code",
    "message": "Human-readable message",
    "hint": "Optional suggestion"
  }
}
```

Common status codes:
- `400` - Bad Request (invalid input)
- `404` - Not Found
- `422` - Unprocessable Entity (validation failed)
- `500` - Internal Server Error
