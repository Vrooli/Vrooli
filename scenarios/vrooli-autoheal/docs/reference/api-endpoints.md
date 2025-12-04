# API Reference

Complete REST API documentation for vrooli-autoheal.

## Base URL

```
http://localhost:{API_PORT}
```

Get the API port with:
```bash
vrooli scenario port vrooli-autoheal API_PORT
```

## Authentication

Currently no authentication required. All endpoints are accessible without credentials.

## Endpoints

### Health & Lifecycle

#### GET /health

Lifecycle health check for the Vrooli scenario system.

**Response:**
```json
{
  "status": "healthy",
  "service": "vrooli-autoheal-api",
  "version": "1.0.0",
  "readiness": true,
  "dependencies": {
    "database": "connected"
  }
}
```

**Status Codes:**
| Code | Meaning |
|------|---------|
| 200 | Service healthy |
| 503 | Service unhealthy |

---

### Platform

#### GET /api/v1/platform

Get detected platform and capabilities.

**Response:**
```json
{
  "platform": "linux",
  "supportsRdp": false,
  "supportsSystemd": true,
  "supportsLaunchd": false,
  "supportsWindowsServices": false,
  "isHeadlessServer": true,
  "hasDocker": true,
  "isWsl": false,
  "supportsCloudflared": true
}
```

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| platform | string | "linux", "windows", "macos", or "other" |
| supportsRdp | boolean | Remote desktop available |
| supportsSystemd | boolean | Linux systemd available |
| supportsLaunchd | boolean | macOS launchd available |
| supportsWindowsServices | boolean | Windows SCM available |
| isHeadlessServer | boolean | No display manager detected |
| hasDocker | boolean | Docker daemon available |
| isWsl | boolean | Running in WSL |
| supportsCloudflared | boolean | Cloudflare tunnel available |

---

### Status

#### GET /api/v1/status

Get current health status summary with all check results.

**Response:**
```json
{
  "status": "ok",
  "platform": {
    "platform": "linux",
    "hasDocker": true,
    ...
  },
  "summary": {
    "total": 7,
    "ok": 6,
    "warning": 1,
    "critical": 0
  },
  "checks": [
    {
      "checkId": "infra-network",
      "status": "ok",
      "message": "Network connectivity OK",
      "timestamp": "2024-01-15T10:30:00Z",
      "duration": 15
    },
    ...
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| status | string | Overall status: "ok", "warning", "critical" |
| platform | object | Platform capabilities |
| summary | object | Count of checks by status |
| checks | array | Individual check results |
| timestamp | string | ISO 8601 timestamp |

---

### Health Checks

#### POST /api/v1/tick

Run a health check cycle.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| force | boolean | false | Ignore interval restrictions |

**Request:**
```bash
curl -X POST "http://localhost:PORT/api/v1/tick?force=true"
```

**Response:**
```json
{
  "success": true,
  "results": [
    {
      "checkId": "infra-network",
      "status": "ok",
      "message": "Network connectivity OK",
      "timestamp": "2024-01-15T10:30:00Z",
      "duration": 15
    },
    ...
  ],
  "summary": {
    "total": 7,
    "ok": 7,
    "warning": 0,
    "critical": 0
  }
}
```

---

#### GET /api/v1/checks

List all registered health checks with metadata.

**Response:**
```json
[
  {
    "id": "infra-network",
    "title": "Network Connectivity",
    "description": "TCP connectivity to 8.8.8.8:53",
    "intervalSeconds": 60,
    "category": "infrastructure",
    "importance": "critical",
    "platforms": null
  },
  ...
]
```

**Fields:**
| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique check identifier |
| title | string | Human-readable title |
| description | string | What the check does |
| intervalSeconds | int | Run frequency in seconds |
| category | string | Check category |
| importance | string | "critical", "high", "medium", "low" |
| platforms | array | Platforms to run on (null = all) |

---

#### GET /api/v1/checks/{checkId}

Get the latest result for a specific check.

**Response:**
```json
{
  "checkId": "infra-network",
  "status": "ok",
  "message": "Network connectivity OK",
  "timestamp": "2024-01-15T10:30:00Z",
  "duration": 15,
  "details": {
    "target": "8.8.8.8:53"
  }
}
```

**Status Codes:**
| Code | Meaning |
|------|---------|
| 200 | Check found |
| 404 | Check not found |

---

#### GET /api/v1/checks/{checkId}/history

Get historical results for a specific check.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 100 | Maximum results |
| since | string | 24h ago | ISO 8601 timestamp |

**Response:**
```json
{
  "checkId": "infra-network",
  "results": [
    {
      "status": "ok",
      "message": "Network connectivity OK",
      "timestamp": "2024-01-15T10:30:00Z",
      "duration": 15
    },
    ...
  ]
}
```

---

#### GET /api/v1/checks/trends

Get trend data for all checks over time.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| period | string | "24h" | "24h", "7d", or "30d" |

**Response:**
```json
{
  "period": "24h",
  "checks": {
    "infra-network": {
      "total": 1440,
      "ok": 1438,
      "warning": 2,
      "critical": 0,
      "uptime": 99.86
    },
    ...
  }
}
```

---

### Timeline & History

#### GET /api/v1/timeline

Get recent health events for the timeline.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 50 | Maximum events |
| since | string | 24h ago | ISO 8601 timestamp |

**Response:**
```json
{
  "events": [
    {
      "id": "evt-123",
      "type": "status_change",
      "checkId": "infra-docker",
      "from": "ok",
      "to": "critical",
      "message": "Docker daemon not responding",
      "timestamp": "2024-01-15T10:25:00Z"
    },
    {
      "id": "evt-124",
      "type": "autoheal_action",
      "checkId": "infra-docker",
      "action": "restart",
      "success": true,
      "timestamp": "2024-01-15T10:25:30Z"
    },
    ...
  ]
}
```

---

#### GET /api/v1/uptime

Get uptime statistics.

**Response:**
```json
{
  "periods": {
    "24h": 99.86,
    "7d": 99.42,
    "30d": 99.21
  },
  "currentStreak": {
    "status": "ok",
    "since": "2024-01-14T08:00:00Z",
    "duration": "26h30m"
  }
}
```

---

#### GET /api/v1/incidents

Get incident history.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 20 | Maximum incidents |
| status | string | all | "open", "resolved", or "all" |

**Response:**
```json
{
  "incidents": [
    {
      "id": "inc-456",
      "checkId": "infra-docker",
      "status": "resolved",
      "startedAt": "2024-01-15T10:25:00Z",
      "resolvedAt": "2024-01-15T10:26:00Z",
      "duration": 60,
      "autoHealed": true
    },
    ...
  ]
}
```

---

### Watchdog

#### GET /api/v1/watchdog

Get watchdog status.

**Response:**
```json
{
  "installed": true,
  "running": true,
  "type": "systemd",
  "serviceName": "vrooli-autoheal",
  "lastChecked": "2024-01-15T10:30:00Z"
}
```

---

#### GET /api/v1/watchdog/template

Get the watchdog configuration template for the current platform.

**Response:**
```json
{
  "platform": "linux",
  "type": "systemd",
  "template": "[Unit]\nDescription=Vrooli Autoheal...",
  "installPath": "/etc/systemd/system/vrooli-autoheal.service",
  "instructions": "sudo systemctl enable vrooli-autoheal"
}
```

---

### Documentation

#### GET /api/v1/docs/manifest

Get the documentation manifest for navigation.

**Response:**
```json
{
  "version": "1.0.0",
  "title": "Autoheal Documentation",
  "defaultDocument": "QUICKSTART.md",
  "sections": [
    {
      "id": "getting-started",
      "title": "Getting Started",
      "documents": [
        { "path": "QUICKSTART.md", "title": "Quick Start" },
        ...
      ]
    },
    ...
  ]
}
```

---

#### GET /api/v1/docs/content

Get the content of a specific documentation file.

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| path | string | yes | Relative path to doc file |

**Request:**
```bash
curl "http://localhost:PORT/api/v1/docs/content?path=QUICKSTART.md"
```

**Response:**
```json
{
  "path": "QUICKSTART.md",
  "content": "# Quick Start\n\nGet vrooli-autoheal running..."
}
```

**Status Codes:**
| Code | Meaning |
|------|---------|
| 200 | Document found |
| 400 | Invalid path |
| 404 | Document not found |

## Error Responses

All errors follow this format:

```json
{
  "error": "Error message here",
  "code": "ERROR_CODE",
  "details": {}
}
```

Common error codes:
| Code | HTTP Status | Description |
|------|-------------|-------------|
| NOT_FOUND | 404 | Resource not found |
| INVALID_REQUEST | 400 | Malformed request |
| INTERNAL_ERROR | 500 | Server error |
