# API Reference

Complete REST API endpoint documentation for Scenario-to-Cloud.

> [CODE: api/main.go `setupRoutes()`] — canonical route registration

## Base URL

All endpoints are prefixed with `/api/v1`.

## Health

### GET /health

Check API health status. Also available at `/api/v1/health`.

**Response:**
```json
{
  "status": "ok",
  "service": "scenario-to-cloud",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Scenarios

> [CODE: api/scenarios.go]

### GET /scenarios

List all locally available scenarios with their service.json metadata.

**Response:**
```json
{
  "scenarios": [
    {
      "id": "my-scenario",
      "name": "My Scenario",
      "description": "...",
      "services": ["api", "ui"],
      "ports": { "api": 8080, "ui": 3000 }
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /scenarios/{id}/ports

Get port mappings for a specific scenario.

**Response:**
```json
{
  "scenario_id": "my-scenario",
  "ports": { "api": 8080, "ui": 3000 },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /scenarios/{id}/dependencies

Get resource and scenario dependencies for a specific scenario.

**Response:**
```json
{
  "scenario_id": "my-scenario",
  "resources": ["postgres", "redis"],
  "scenarios": [],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Validation

### POST /validate/reachability

Validate that a VPS host is reachable via SSH.

**Request Body:**
```json
{
  "host": "192.168.1.100",
  "port": 22,
  "user": "root",
  "key_path": "~/.ssh/id_rsa"
}
```

**Response:**
```json
{
  "ok": true,
  "message": "Host reachable",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Manifest

> [CODE: api/manifest/]

### POST /manifest/validate

Validate a deployment manifest.

**Request Body:**
```json
{
  "scenario": { "id": "my-scenario" },
  "target": { "vps": { "host": "example.com" } },
  "edge": { "domain": "app.example.com" }
}
```

**Response:**
```json
{
  "valid": true,
  "issues": [],
  "manifest": { "..." },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Bundles

> [CODE: api/bundle/handlers.go]

### POST /bundle/build

Build a deployment bundle.

**Request Body:** Same as `/manifest/validate`

**Response:**
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

### GET /bundles

List all locally cached bundles.

**Response:**
```json
{
  "bundles": [
    {
      "filename": "vrooli-bundle-my-scenario-20240115.tar.gz",
      "scenario_id": "my-scenario",
      "sha256": "abc123...",
      "size_bytes": 15000000,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /bundles/stats

Get aggregate bundle statistics.

**Response:**
```json
{
  "stats": {
    "total_bundles": 5,
    "total_size_bytes": 75000000,
    "total_size_kb": 73242,
    "by_scenario": {
      "my-scenario": { "count": 3, "total_size_bytes": 45000000 }
    }
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### DELETE /bundles/{sha256}

Delete a specific local bundle by its SHA256 hash.

**Response:**
```json
{
  "ok": true,
  "freed_bytes": 15000000,
  "message": "Bundle deleted",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /bundles/cleanup

Clean up old bundles locally and optionally on VPS.

**Request Body:**
```json
{
  "scenario_id": "optional-filter",
  "keep_latest": 3,
  "clean_vps": false,
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "~/.ssh/id_rsa",
  "workdir": "~/Vrooli"
}
```

**Response:**
```json
{
  "ok": true,
  "local_deleted": ["..."],
  "local_freed_bytes": 30000000,
  "vps_deleted": 0,
  "vps_freed_bytes": 0,
  "vps_error": "",
  "message": "Cleaned up 2 bundles",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /bundles/vps/list

List bundles present on a remote VPS.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "~/.ssh/id_rsa",
  "workdir": "~/Vrooli"
}
```

**Response:**
```json
{
  "ok": true,
  "bundles": [
    {
      "filename": "vrooli-bundle-my-scenario-20240115.tar.gz",
      "scenario_id": "my-scenario",
      "sha256": "abc123...",
      "size_bytes": 15000000,
      "mod_time": "2024-01-15T10:30:00Z"
    }
  ],
  "total_size_bytes": 15000000,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /bundles/vps/delete

Delete a specific bundle from a remote VPS.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "~/.ssh/id_rsa",
  "workdir": "~/Vrooli",
  "filename": "vrooli-bundle-my-scenario-20240115.tar.gz"
}
```

**Response:**
```json
{
  "ok": true,
  "freed_bytes": 15000000,
  "message": "VPS bundle deleted",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Preflight

> [CODE: api/vps/preflight/handlers.go]

### POST /preflight

Run preflight checks against a VPS.

**Request Body:**
```json
{
  "manifest": { "..." }
}
```

**Response:**
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
  "issues": [],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /preflight/fix/ports

Stop services occupying required ports on the VPS.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "~/.ssh/id_rsa",
  "ports": [80, 443],
  "pids": [1234],
  "services": ["nginx", "apache2"],
  "prefer_service_stop": true
}
```

**Response:**
```json
{
  "ok": true,
  "stopped": ["service nginx", "pid 1234"],
  "failed": [],
  "message": "Stopped 2 port occupiers",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /preflight/fix/firewall

Open required firewall ports on the VPS.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "~/.ssh/id_rsa",
  "ports": [80, 443]
}
```

**Response:**
```json
{
  "ok": true,
  "message": "Firewall ports opened",
  "ports": [80, 443],
  "status": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /preflight/fix/stop-processes

Stop running scenario processes on the VPS.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "~/.ssh/id_rsa",
  "workdir": "/root/Vrooli",
  "scenario_id": "optional"
}
```

**Response:**
```json
{
  "ok": true,
  "action": "stop_scenario",
  "message": "Stopped scenario processes",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /preflight/disk/usage

Get disk usage information from the VPS.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "~/.ssh/id_rsa"
}
```

**Response:**
```json
{
  "ok": true,
  "free_space": "10 GB",
  "free_bytes": 10737418240,
  "total_space": "50 GB",
  "total_bytes": 53687091200,
  "used_percent": 80,
  "largest_dirs": [
    { "path": "/var/log", "size": "2.5 GB", "bytes": 2684354560 }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /preflight/disk/cleanup

Free disk space on the VPS by running cleanup actions.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "~/.ssh/id_rsa",
  "actions": ["apt_clean", "journal_vacuum", "docker_prune", "tmp_clean"]
}
```

**Response:**
```json
{
  "ok": true,
  "space_freed": "1.2 GB",
  "space_freed_kb": 1258291,
  "message": "Cleaned up 1.2 GB",
  "actions_run": ["apt_clean", "journal_vacuum"],
  "actions_failed": [],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Secrets

> [CODE: api/secrets/]

### GET /secrets/{scenario}

Get local secrets defined for a scenario.

**Response:**
```json
{
  "secrets": { "KEY": "value" },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## VPS Plan/Apply Operations

> [CODE: api/handlers_vps_management.go]

These endpoints expose a plan/apply pattern for VPS operations.

### POST /vps/setup/plan

Generate a VPS setup execution plan.

**Request Body:** Deployment manifest JSON.

**Response:**
```json
{
  "plan": [
    { "id": "build", "title": "Build Bundle", "description": "..." }
  ],
  "issues": [],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /vps/setup/apply

Execute VPS setup (transfer bundle, run setup scripts).

**Request Body:** Deployment manifest JSON.

**Response:** `VPSSetupResult` with step results, errors, and timing.

### POST /vps/deploy/plan

Generate a deployment execution plan.

**Request Body:** Deployment manifest JSON.

**Response:** Same shape as `/vps/setup/plan`.

### POST /vps/deploy/apply

Execute scenario deployment (start resources, services, Caddy).

**Request Body:** Deployment manifest JSON.

**Response:** `VPSDeployResult` with step results, errors, and timing.

### POST /vps/inspect/plan

Generate an inspection execution plan.

**Request Body:** Deployment manifest JSON.

**Response:** Same shape as `/vps/setup/plan`.

### POST /vps/inspect/apply

Execute VPS inspection (check processes, logs, health).

**Request Body:** Deployment manifest JSON.

**Response:** `VPSInspectResult` with scenario status, logs, and resource state.

---

## Deployments

> [CODE: api/handlers_deployment.go, api/deployment/orchestrator.go]

### GET /deployments

List all deployments.

**Query Parameters:**
- `status` — Filter by deployment status
- `scenario_id` — Filter by scenario ID

**Response:**
```json
{
  "deployments": [
    {
      "id": "uuid",
      "name": "my-scenario @ example.com",
      "scenario_id": "my-scenario",
      "status": "deployed",
      "error_message": "",
      "progress_step": "",
      "progress_percent": 100,
      "created_at": "2024-01-15T10:30:00Z",
      "last_deployed_at": "2024-01-15T10:35:00Z",
      "domain": "app.example.com",
      "host": "192.168.1.100"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments

Create (or update) a deployment.

**Request Body:**
```json
{
  "manifest": { "..." },
  "name": "Optional custom name",
  "bundle_path": "optional",
  "bundle_sha256": "optional",
  "bundle_size_bytes": 0
}
```

**Response:**
```json
{
  "deployment": { "..." },
  "created": true,
  "updated": false,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}

Get full deployment details.

**Response:**
```json
{
  "deployment": {
    "id": "uuid",
    "name": "my-scenario @ example.com",
    "status": "deployed",
    "manifest": { "..." },
    "setup_result": { "..." },
    "deploy_result": { "..." },
    "created_at": "2024-01-15T10:30:00Z",
    "last_deployed_at": "2024-01-15T10:35:00Z"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### DELETE /deployments/{id}

Delete a deployment record.

**Query Parameters:**
- `stop=true` — Also stop the scenario on VPS
- `cleanup=true` — Also cleanup bundle files

**Response:**
```json
{
  "deleted": true,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/execute

Execute the full deployment pipeline (preflight + setup + deploy).

**Request Body** (optional):
```json
{
  "run_preflight": true,
  "force_bundle_build": false,
  "provided_secrets": { "KEY": "value" }
}
```

**Response:**
```json
{
  "deployment": { "..." },
  "run_id": "uuid",
  "message": "Deployment started",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/progress

**SSE (Server-Sent Events) stream** of deployment progress.

Each event is a JSON object:
```json
{
  "step": "extract",
  "status": "running",
  "message": "Extracting bundle...",
  "percent": 45,
  "error_category": "",
  "retryable": false,
  "hint": ""
}
```

> Connect with `EventSource`. The stream closes when the deployment completes or fails. See [architecture.md](../concepts/architecture.md#sse-progress-streaming) for details.

### POST /deployments/{id}/inspect

Inspect deployment status on the VPS.

**Response:**
```json
{
  "result": {
    "ok": true,
    "scenario_status": { "..." },
    "scenario_logs": "...",
    "timestamp": "2024-01-15T10:30:00Z"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/stop

Stop the deployment on VPS.

**Response:**
```json
{
  "success": true,
  "error": "",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/start

Start a previously stopped deployment.

**Request Body** (optional):
```json
{
  "provided_secrets": { "KEY": "value" }
}
```

**Response:**
```json
{
  "deployment_id": "uuid",
  "run_id": "uuid",
  "message": "Deployment starting",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Live State Inspection

> [CODE: api/handlers_live_state.go, api/vps/live_state.go]

### GET /deployments/{id}/live-state

Get comprehensive live state from the VPS (runs ~15 SSH commands in parallel).

**Response:**
```json
{
  "result": { "..." },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

The result includes processes, ports, disk, memory, CPU, uptime, scenario status, resource status, Caddy config, and more. See [architecture.md](../concepts/architecture.md#live-state-parallel-inspection) for the full collection.

### GET /deployments/{id}/files

List files on the remote VPS.

**Query Parameters:**
- `path` — Directory path (defaults to deployment workdir)

**Response:**
```json
{
  "ok": true,
  "path": "/root/Vrooli",
  "entries": [
    {
      "name": "scripts",
      "size": "4096",
      "permissions": "drwxr-xr-x",
      "owner": "root",
      "modified": "2024-01-15"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/files/content

Read a file's content from the VPS.

**Query Parameters:**
- `path` (required) — File path on the VPS

**Response:**
```json
{
  "ok": true,
  "path": "/root/Vrooli/scenarios/my-scenario/.env",
  "size_bytes": 256,
  "content": "KEY=value\n...",
  "truncated": false,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/drift

Detect drift between expected and actual state on the VPS.

**Response:**
```json
{
  "result": { "..." },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/health

Get deployment health status.

**Response:**
```json
{
  "ok": true,
  "checks": { "..." },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Live State Actions

> [CODE: api/handlers_live_state.go]

### POST /deployments/{id}/actions/kill

Kill a process on the VPS by PID.

**Request Body:**
```json
{
  "pid": 12345,
  "signal": "TERM"
}
```

**Response:**
```json
{
  "ok": true,
  "pid": 12345,
  "signal": "TERM",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/actions/restart

Restart a scenario or resource on the VPS.

**Request Body:**
```json
{
  "type": "scenario",
  "id": "my-scenario"
}
```

**Response:**
```json
{
  "ok": true,
  "type": "scenario",
  "id": "my-scenario",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/actions/process

Generic process control (start/stop/restart/setup).

**Request Body:**
```json
{
  "action": "restart",
  "type": "resource",
  "id": "postgres"
}
```

**Response:**
```json
{
  "ok": true,
  "action": "restart",
  "type": "resource",
  "id": "postgres",
  "message": "Restarted",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/actions/vps

VPS-level management actions (reboot, stop Vrooli, cleanup).

**Request Body:**
```json
{
  "action": "reboot",
  "cleanup_level": 3,
  "confirmation": "REBOOT"
}
```

**Response:**
```json
{
  "ok": true,
  "action": "reboot",
  "message": "VPS rebooting",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## History & Logs

> [CODE: api/handlers_history.go]

### GET /deployments/{id}/history

Get deployment event history.

**Response:**
```json
{
  "ok": true,
  "history": [
    {
      "type": "deploy_started",
      "timestamp": "2024-01-15T10:30:00Z",
      "message": "Deployment started",
      "details": "...",
      "success": true,
      "step_name": "setup"
    }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/history

Add a custom history event.

**Request Body:**
```json
{
  "type": "custom_event",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Manual intervention",
  "details": "...",
  "success": true
}
```

**Response:**
```json
{
  "ok": true,
  "event": { "..." },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/logs

Get deployment logs with filtering.

**Query Parameters:**
- `source` (default: `all`) — Filter by log source
- `level` (default: `all`) — Filter by level: `INFO`, `WARN`, `ERROR`, `DEBUG`
- `tail` (default: `200`, max: `2000`) — Number of lines
- `search` — Text search filter

**Response:**
```json
{
  "ok": true,
  "logs": [
    {
      "timestamp": "2024-01-15T10:30:00Z",
      "source": "scenario",
      "level": "INFO",
      "message": "Server started on port 8080"
    }
  ],
  "total": 500,
  "filtered": 200,
  "sources": ["scenario", "caddy", "system"]
}
```

---

## VPS Secrets Management

> [CODE: api/secrets/handlers_management.go]

Post-deployment CRUD for secrets stored on the VPS.

### GET /deployments/{id}/secrets

List all secrets on the VPS (values masked by default).

**Response:**
```json
{
  "secrets": [
    { "key": "DATABASE_URL", "masked": true, "source": "provisioned" }
  ],
  "metadata": {
    "environment": "production",
    "last_updated": "2024-01-15T10:30:00Z",
    "scenario_id": "my-scenario"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/secrets/{key}

Get a specific secret.

**Query Parameters:**
- `reveal=true` — Show the actual value (otherwise masked)

**Response:**
```json
{
  "secret": {
    "key": "DATABASE_URL",
    "value": "postgres://...",
    "masked": false,
    "source": "provisioned"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/secrets

Create a new secret on the VPS.

**Request Body:**
```json
{
  "key": "MY_API_KEY",
  "value": "secret-value",
  "restart_scenario": false
}
```

**Response:**
```json
{
  "ok": true,
  "key": "MY_API_KEY",
  "action": "created",
  "message": "Secret created",
  "scenario_restart": false,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### PUT /deployments/{id}/secrets/{key}

Update an existing secret.

**Request Body:**
```json
{
  "value": "new-secret-value",
  "restart_scenario": false
}
```

**Response:**
```json
{
  "ok": true,
  "key": "MY_API_KEY",
  "action": "updated",
  "message": "Secret updated",
  "scenario_restart": false,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### DELETE /deployments/{id}/secrets/{key}

Delete a secret from the VPS.

**Request Body:**
```json
{
  "confirmation": "DELETE",
  "restart_scenario": false
}
```

**Response:**
```json
{
  "ok": true,
  "key": "MY_API_KEY",
  "action": "deleted",
  "message": "Secret deleted",
  "scenario_restart": false,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/expected-secrets

Get secrets defined in the scenario's service.json that are expected to exist on the VPS.

**Response:**
```json
{
  "secrets": [
    { "key": "DATABASE_URL", "required": true, "description": "..." }
  ],
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Terminal

### GET /deployments/{id}/terminal

**WebSocket** endpoint — opens an interactive SSH terminal session to the VPS.

Connect via WebSocket. Sends/receives raw terminal data. The server proxies stdin/stdout/stderr over SSH.

---

## Edge/TLS Management

> [CODE: api/handlers_edge.go]

### GET /deployments/{id}/edge/dns-check

Check if DNS records point to the deployment's VPS.

**Response:**
```json
{
  "ok": true,
  "vps_host": "192.168.1.100",
  "vps_ips": ["192.168.1.100"],
  "domains": [
    {
      "domain": "app.example.com",
      "role": "apex",
      "ok": true,
      "domain_ips": ["192.168.1.100"],
      "points_to_vps": true,
      "proxied": false,
      "message": "DNS configured correctly",
      "hint": "",
      "hint_data": {
        "type": "A",
        "name": "app.example.com",
        "value": "192.168.1.100",
        "ttl": 3600
      }
    }
  ],
  "message": "All DNS records OK",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/edge/dns-records

Get raw DNS records for the deployment's domains.

**Response:**
```json
{
  "ok": true,
  "domains": [
    {
      "domain": "app.example.com",
      "records": {
        "a": ["192.168.1.100"],
        "aaaa": [],
        "cname": "",
        "mx": [],
        "txt": []
      }
    }
  ],
  "message": "",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/edge/caddy

Control the Caddy web server on the VPS.

**Request Body:**
```json
{
  "action": "restart"
}
```

Actions: `start`, `stop`, `restart`, `reload`.

**Response:**
```json
{
  "ok": true,
  "action": "restart",
  "message": "Caddy restarted",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### GET /deployments/{id}/edge/tls

Get TLS certificate information for the deployment's domain.

**Response:**
```json
{
  "ok": true,
  "domain": "app.example.com",
  "valid": true,
  "validation": "valid",
  "issuer": "Let's Encrypt",
  "subject": "app.example.com",
  "not_before": "2024-01-01T00:00:00Z",
  "not_after": "2024-04-01T00:00:00Z",
  "days_remaining": 76,
  "serial_number": "...",
  "sans": ["app.example.com"],
  "alpn": {
    "supported": true,
    "protocols": ["h2", "http/1.1"]
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /deployments/{id}/edge/tls/renew

Force TLS certificate renewal on the VPS.

**Response:**
```json
{
  "ok": true,
  "domain": "app.example.com",
  "message": "Certificate renewal initiated",
  "output": "...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## SSH Key Management

> [CODE: api/ssh/handlers.go, api/ssh/keys.go, api/ssh/keys_generate.go, api/ssh/keys_copy.go]

### GET /ssh/keys

List SSH keys available on the local machine.

**Response:**
```json
{
  "ok": true,
  "status": "success",
  "keys": [
    {
      "name": "id_ed25519",
      "type": "ed25519",
      "bits": 256,
      "fingerprint": "SHA256:...",
      "comment": "user@host",
      "path": "/home/user/.ssh/id_ed25519",
      "public_path": "/home/user/.ssh/id_ed25519.pub",
      "has_passphrase": false,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "ssh_dir": "/home/user/.ssh",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### DELETE /ssh/keys

Delete an SSH key pair from the local machine.

**Request Body:**
```json
{
  "key_path": "/home/user/.ssh/id_ed25519"
}
```

**Response:**
```json
{
  "ok": true,
  "status": "success",
  "message": "Key pair deleted",
  "private_deleted": true,
  "public_deleted": true,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /ssh/keys/generate

Generate a new SSH key pair.

**Request Body:**
```json
{
  "type": "ed25519",
  "bits": 4096,
  "comment": "deploy@example.com",
  "filename": "deploy_key",
  "password": ""
}
```

**Response:**
```json
{
  "ok": true,
  "status": "success",
  "key": { "..." },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /ssh/keys/public

Get the public key content for a given private key path.

**Request Body:**
```json
{
  "key_path": "/home/user/.ssh/id_ed25519"
}
```

**Response:**
```json
{
  "ok": true,
  "status": "success",
  "public_key": "ssh-ed25519 AAAA... user@host",
  "fingerprint": "SHA256:...",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### POST /ssh/test

Test SSH connectivity to a remote host.

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "/home/user/.ssh/id_ed25519"
}
```

**Response:**
```json
{
  "ok": true,
  "status": "success",
  "message": "Connection successful",
  "hint": "",
  "server_info": "SSH-2.0-OpenSSH_9.0",
  "fingerprint": "SHA256:...",
  "latency_ms": 45,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

Possible `status` values: `success`, `auth_failed`, `timeout`, `host_unreachable`, `host_key_changed`, `ipv6_unavailable`, `key_error`, `error`. See [configuration.md](configuration.md#error-categories) for details.

### POST /ssh/copy-key

Copy an SSH public key to a remote server's `authorized_keys` (uses password authentication).

**Request Body:**
```json
{
  "host": "example.com",
  "port": 22,
  "user": "root",
  "key_path": "/home/user/.ssh/id_ed25519",
  "password": "server-password"
}
```

**Response:**
```json
{
  "ok": true,
  "status": "success",
  "message": "Key copied successfully",
  "hint": "",
  "key_copied": true,
  "already_exists": false,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Investigation (Legacy)

> [CODE: api/investigation/] — Legacy endpoints, kept for backward compatibility. Prefer the unified [Tasks](#tasks) endpoints.

### POST /deployments/{id}/investigate

Start an investigation using the agent-manager.

**Request Body:**
```json
{
  "focus": { "harness": true, "subject": true },
  "note": "Deployment failing after deploy step",
  "effort": "inspect"
}
```

**Response:**
```json
{
  "investigation": { "..." }
}
```

### GET /deployments/{id}/investigations

List investigations for a deployment.

**Response:**
```json
{
  "investigations": [{ "..." }]
}
```

### GET /deployments/{id}/investigations/{invId}

Get a specific investigation.

**Response:**
```json
{
  "investigation": { "..." }
}
```

### POST /deployments/{id}/investigations/{invId}/stop

Stop a running investigation.

**Response:**
```json
{
  "success": true,
  "message": "Investigation stopped"
}
```

### POST /deployments/{id}/investigations/{invId}/apply-fixes

Apply fixes recommended by an investigation.

**Response:**
```json
{
  "success": true,
  "applied": ["..."]
}
```

### GET /agent-manager/status

Check if the agent-manager integration is available.

**Response:**
```json
{
  "available": true,
  "profile": "scenario-to-cloud-investigator",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Tasks (Unified)

> [CODE: api/handlers_tasks.go, api/tasks/]

Unified task endpoints that replace the legacy investigation API.

### POST /deployments/{id}/tasks

Create a new task (investigate or fix).

**Request Body:**
```json
{
  "task_type": "investigate",
  "focus": { "harness": true, "subject": true },
  "note": "Deployment failing after deploy step",
  "effort": "inspect",
  "permissions": {
    "immediate": false,
    "permanent": false,
    "prevention": false
  },
  "include_contexts": [],
  "source_investigation_id": "",
  "max_iterations": 5
}
```

**Response:**
```json
{
  "task": { "..." }
}
```

### GET /deployments/{id}/tasks

List tasks for a deployment.

**Query Parameters:**
- `limit` (default: `10`) — Maximum number of tasks

**Response:**
```json
{
  "tasks": [{ "..." }]
}
```

### GET /deployments/{id}/tasks/{taskId}

Get a specific task.

**Response:**
```json
{
  "task": { "..." }
}
```

### POST /deployments/{id}/tasks/{taskId}/stop

Stop a running task.

**Response:**
```json
{
  "success": true,
  "message": "Task stopped"
}
```

---

## Tool Discovery Protocol

> [CODE: api/toolhandlers/, api/toolregistry/]

### GET /tools

List all available tools with their schemas.

**Response:**
```json
{
  "tools": [
    {
      "name": "execute_deployment",
      "description": "Execute a deployment pipeline",
      "input_schema": { "..." },
      "category": "deployment"
    }
  ],
  "scenario": {
    "name": "scenario-to-cloud",
    "version": "1.0.0",
    "description": "..."
  }
}
```

### GET /tools/{name}

Get a specific tool's schema and metadata.

**Response:**
```json
{
  "tool": {
    "name": "execute_deployment",
    "description": "...",
    "input_schema": { "..." },
    "category": "deployment"
  }
}
```

---

## Tool Execution Protocol

> [CODE: api/toolexecution/]

### POST /tools/execute

Execute a tool by name with arguments.

**Request Body:**
```json
{
  "name": "execute_deployment",
  "arguments": {
    "deployment": "uuid",
    "run_preflight": true
  }
}
```

**Response:**
```json
{
  "result": { "..." },
  "is_error": false
}
```

Available tools: `list_deployments`, `create_deployment`, `execute_deployment`, `stop_deployment`, `start_deployment`, `check_deployment_status`, `inspect_deployment`, `get_deployment_logs`, `get_live_state`, `validate_manifest`, `run_preflight`.

---

## Documentation

### GET /docs/manifest

Get the documentation navigation manifest.

**Response:**
```json
{
  "version": "1.0.0",
  "title": "Scenario-to-Cloud Documentation",
  "defaultDocument": "QUICKSTART.md",
  "sections": [{ "..." }]
}
```

### GET /docs/content

Get a document's content.

**Query Parameters:**
- `path` — Document path (e.g., `QUICKSTART.md`)

**Response:**
```json
{
  "path": "QUICKSTART.md",
  "content": "# Quick Start Guide\n..."
}
```

---

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
- `400` — Bad Request (invalid input)
- `404` — Not Found
- `422` — Unprocessable Entity (validation failed)
- `500` — Internal Server Error

SSH-related endpoints include structured error metadata. See [configuration.md](configuration.md#error-categories) for the full error category taxonomy.
