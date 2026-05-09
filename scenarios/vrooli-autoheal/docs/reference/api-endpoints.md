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
| limit | int | 20 | Maximum results (max 200) |
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

Get per-check trend data over a time window.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| hours | int | 24 | Time window in hours (max 168) |

**Response:**
```json
{
  "infra-network": {
    "total": 1440,
    "ok": 1438,
    "warning": 2,
    "critical": 0,
    "uptime": 99.86
  },
  ...
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

Get uptime statistics over a time window.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| hours | int | 24 | Time window in hours (max 168) |

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

#### GET /api/v1/uptime/history

Get time-bucketed uptime data for charting.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| hours | int | 24 | Time window in hours (max 168) |
| buckets | int | 24 | Number of time buckets (max 100) |

**Response:**
```json
{
  "windowHours": 24,
  "bucketCount": 24,
  "buckets": [
    {
      "start": "2024-01-14T10:00:00Z",
      "end": "2024-01-14T11:00:00Z",
      "uptimePercent": 100.0,
      "checkCount": 60
    },
    ...
  ]
}
```

---

#### GET /api/v1/incidents

Get durable operator-facing incidents.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| status | string |  | Filter by `open`, `acknowledged`, `resolved`, or `ignored` |
| severity | string |  | Filter by `info`, `warning`, or `critical` |
| type | string |  | Filter by incident type |
| limit | int | 50 | Maximum incidents (max 200) |

**Response:**
```json
{
  "incidents": [
    {
      "id": "inc_3be099c9313dee0b819991a7",
      "fingerprint": "incfp_3be099c9313dee0b819991a7",
      "type": "host_integrity",
      "severity": "critical",
      "status": "open",
      "title": "Host integrity issue detected",
      "summary": "Runtime/device stack mismatch",
      "detectedAt": "2026-05-08T15:45:57Z",
      "lastSeenAt": "2026-05-08T16:03:13Z",
      "eventCount": 1,
      "observationCount": 2
    },
    ...
  ]
}
```

#### GET /api/v1/incidents/{incidentId}/remediations

List structured remediation candidates for one incident. Candidates are descriptive only; autoheal does not execute privileged mutations.

**Response:**
```json
{
  "incidentId": "inc_3be099c9313dee0b819991a7",
  "remediations": [
    {
      "id": "ubuntu-nvidia-kernel-module-mismatch",
      "title": "Install matching NVIDIA kernel module package for the running kernel",
      "applicability": "applicable",
      "requiresOperator": true,
      "requiresPrivilege": true,
      "riskLevel": "moderate",
      "templateId": "ubuntu-nvidia-kernel-module-mismatch",
      "simulation": "apt-get -s install <expected-package>",
      "artifactPolicy": "generate_only_under_user_state",
      "postChecks": ["nvidia-smi", "vrooli-autoheal incidents latest --json"]
    }
  ],
  "total": 1
}
```

`applicability` may be `applicable`, `not_applicable`, `unsupported`, `blocked`, or `needs_corroboration`. Artifact generation is refused unless the candidate is `applicable`.

#### POST /api/v1/incidents/{incidentId}/remediations/{remediationId}/generate

Generate an operator-reviewable remediation artifact under resolver-backed user state using `api-core/storage`. The endpoint writes files such as `remediation.sh`, `metadata.json`, `README.md`, and `post-checks.json`; it never runs the generated script.

Generated artifacts are incident-specific and host-specific. They belong under the storage-resolved state directory for `vrooli-autoheal`, not under the checked-in `scenarios/vrooli-autoheal` source tree. Source code may contain reusable remediation generator/template logic, but the operator-run script produced from that logic is stored only as a state artifact. The exact root can vary by OS, profile, and `VROOLI_STATE_ROOT`; consumers should use the returned `artifact.path`.

**Response:**
```json
{
  "incidentId": "inc_3be099c9313dee0b819991a7",
  "artifact": {
    "id": "ubuntu-nvidia-kernel-module-mismatch-artifact",
    "remediationId": "ubuntu-nvidia-kernel-module-mismatch",
    "path": "/home/user/.local/state/vrooli/vrooli-autoheal/incidents/inc_3be099c9313dee0b819991a7/remediation/ubuntu-nvidia-kernel-module-mismatch",
    "generatedAt": "2026-05-08T16:10:00Z"
  },
  "files": {
    "remediation.sh": ".../remediation.sh",
    "metadata.json": ".../metadata.json",
    "README.md": ".../README.md",
    "post-checks.json": ".../post-checks.json"
  }
}
```

#### POST /api/v1/incidents/{incidentId}/remediations/{remediationId}/outcome

Record an operator-reported remediation outcome on the incident.

**Request:**
```json
{
  "status": "verified",
  "note": "post-checks are healthy"
}
```

Supported statuses are `generated`, `operator_ran`, `verified`, `failed`, and `abandoned`.

#### GET /api/v1/transitions

Get derived health-check status transitions for timeline and trends views.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| hours | int | 24 | Time window in hours (max 168) |
| limit | int | 50 | Maximum transitions (max 200) |

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
  "template": "[Unit]\nDescription=Vrooli Autoheal...",
  "instructions": "1. Save the template to /etc/systemd/system/...",
  "oneLiner": "curl -s .../api/v1/watchdog/template | jq ..."
}
```

---

#### POST /api/v1/watchdog/install

Install the OS-level watchdog service. Timeout: 2 minutes.

**Request Body (optional):**
```json
{
  "serviceName": "vrooli-autoheal",
  "enableLinger": true
}
```

**Response:**
```json
{
  "success": true,
  "message": "Watchdog service installed and started"
}
```

---

#### POST /api/v1/watchdog/uninstall

Remove the OS-level watchdog service.

**Response:**
```json
{
  "success": true,
  "message": "Watchdog service removed"
}
```

---

#### GET /api/v1/watchdog/status

Get detailed watchdog service status from the OS service manager.

---

#### POST /api/v1/watchdog/linger

Enable systemd linger for the current user (Linux only). Allows user services to run without an active login session.

---

### Configuration

#### GET /api/v1/config

Get the current autoheal configuration.

---

#### PUT /api/v1/config

Replace the entire configuration. Body must be a valid config object.

---

#### POST /api/v1/config/validate

Validate a configuration object without applying it.

---

#### GET /api/v1/config/schema

Get the JSON schema for the configuration format.

---

#### GET /api/v1/config/export

Export the current configuration for backup or transfer.

---

#### POST /api/v1/config/import

Import a previously exported configuration.

---

#### GET /api/v1/config/defaults

Get the default configuration values for all checks.

---

#### GET /api/v1/config/global

Get global configuration (tick interval, auto-heal policy, etc.).

---

#### GET /api/v1/config/ui

Get UI-specific configuration.

---

#### PUT /api/v1/config/checks/bulk

Bulk update check configurations (enable/disable multiple checks at once).

---

#### GET /api/v1/config/checks/{checkId}

Get configuration for a specific check.

---

#### PUT /api/v1/config/checks/{checkId}/enabled

Enable or disable a specific check.

**Request Body:**
```json
{ "enabled": true }
```

---

#### PUT /api/v1/config/checks/{checkId}/autoheal

Enable or disable auto-heal for a specific check.

**Request Body:**
```json
{ "enabled": true }
```

---

### Monitoring Configuration

#### GET /api/v1/config/monitoring

Get the monitoring configuration (which scenarios and resources are monitored).

---

#### PUT /api/v1/config/monitoring

Update the monitoring configuration.

---

#### POST /api/v1/config/monitoring/scenarios

Add a scenario to monitoring.

---

#### DELETE /api/v1/config/monitoring/scenarios/{name}

Remove a scenario from monitoring.

---

#### PUT /api/v1/config/monitoring/scenarios/{name}/critical

Set whether a scenario is critical (affects severity of failures).

**Request Body:**
```json
{ "critical": true }
```

---

#### POST /api/v1/config/monitoring/resources

Add a resource to monitoring.

---

#### DELETE /api/v1/config/monitoring/resources/{name}

Remove a resource from monitoring.

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

---

### System Event Timeline

#### GET /api/v1/system-events

Returns normalized host-level events used for forensics and change correlation. This is separate from `/api/v1/timeline`, which remains the health-check result timeline.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| since | string | none | RFC3339 time or duration such as `72h`, `7d`, `30d` |
| until | string | none | RFC3339 end time |
| limit | int | 100 | Maximum events, capped at 500 |
| category | string | none | Comma-separated categories such as `kernel,driver,crash` |
| severity | string | none | Comma-separated `info`, `warning`, `critical` |
| source | string | none | Comma-separated sources such as `dpkg-log,journalctl` |
| platform | string | none | Platform filter |
| bootId | string | none | Boot ID filter |
| correlate | boolean | false | Include deterministic temporal correlation hints |

**Response:**
```json
{
  "events": [
    {
      "id": 1,
      "fingerprint": "abc123",
      "occurredAt": "2026-05-08T12:57:16Z",
      "source": "dpkg-log",
      "platform": "linux",
      "category": "driver",
      "severity": "info",
      "title": "Package upgrade: nvidia-driver-580-open",
      "summary": "upgrade nvidia-driver-580-open 580.126 -> 580.142"
    }
  ],
  "count": 1,
  "sources": [
    { "source": "dpkg-log", "platform": "linux", "status": "ok" }
  ],
  "correlations": []
}
```

#### POST /api/v1/system-events/refresh

Runs bounded system-event ingestion immediately.

**Response:**
```json
{
  "ingested": 12,
  "deduped": 40,
  "durationMs": 238,
  "sources": [
    { "source": "journalctl", "platform": "linux", "status": "ok" }
  ]
}
```

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
