# Bundle Endpoints

Endpoints for validating, assembling, and exporting bundle manifests.

## POST /bundles/validate

Validate a desktop bundle manifest against schema v0.1.

**Request:**

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/validate" \
  -H "Content-Type: application/json" \
  -d @bundle.json
```

**Request Body:** Full bundle manifest (see [manifest schema](../examples/manifests/))

**Response (valid):**

```json
{
  "status": "valid",
  "schema": "desktop.v0.1"
}
```

**Response (invalid):**

```json
{
  "error": "bundle failed validation",
  "details": "schema_version is required"
}
```

**Validation Checks:**
- `schema_version` must be "v0.1"
- `target` must be "desktop"
- `app.name` and `app.version` required
- `ipc.mode` must be "loopback-http"
- At least one service required
- All secrets must have valid `class` values
- All services must have valid `type` values

---

## POST /bundles/merge-secrets

Merge secret plans from secrets-manager into a bundle manifest.

**Request:**

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/merge-secrets" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario": "picker-wheel",
    "tier": "tier-2-desktop",
    "manifest": { ... }
  }'
```

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `scenario` | string | Yes | Scenario name |
| `tier` | string | No | Target tier (default: "tier-2-desktop") |
| `manifest` | object | Yes | Base manifest to merge into |

**Response:** Updated manifest with secrets section populated:

```json
{
  "schema_version": "v0.1",
  "target": "desktop",
  "app": { "name": "picker-wheel", "version": "1.0.0" },
  "secrets": [
    {
      "id": "jwt_secret",
      "class": "per_install_generated",
      "generator": { "type": "random", "length": 32, "charset": "alnum" },
      "target": { "type": "env", "name": "JWT_SECRET" }
    },
    {
      "id": "session_secret",
      "class": "user_prompt",
      "prompt": { "label": "Session Secret", "description": "Enter a secret..." },
      "target": { "type": "env", "name": "SESSION_SECRET" }
    }
  ],
  "services": [ ... ]
}
```

**Notes:**
- Fetches secret plans from secrets-manager
- Filters out infrastructure secrets
- Validates manifest before and after merge

---

## POST /bundles/assemble

Assemble a complete bundle manifest for a scenario.

**Request:**

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/assemble" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario": "picker-wheel",
    "tier": "tier-2-desktop",
    "include_secrets": true
  }'
```

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `scenario` | string | Yes | Scenario name |
| `tier` | string | No | Target tier (default: "tier-2-desktop") |
| `include_secrets` | boolean | No | Include secrets section (default: true) |

**Response:**

```json
{
  "status": "assembled",
  "schema": "desktop.v0.1",
  "manifest": {
    "schema_version": "v0.1",
    "target": "desktop",
    "app": {
      "name": "picker-wheel",
      "version": "1.2.3",
      "description": "Picker wheel application"
    },
    "ipc": {
      "mode": "loopback-http",
      "host": "127.0.0.1",
      "port": 47710,
      "auth_token_path": "runtime/auth-token"
    },
    "telemetry": {
      "file": "telemetry/deployment-telemetry.jsonl"
    },
    "ports": {
      "default_range": { "min": 47000, "max": 47999 },
      "reserved": []
    },
    "swaps": [],
    "secrets": [ ... ],
    "services": [ ... ]
  }
}
```

**Process:**
1. Fetches skeleton bundle from scenario-dependency-analyzer
2. Applies profile swaps (if any)
3. Merges secrets from secrets-manager (if `include_secrets: true`)
4. Validates final manifest against schema

---

## POST /bundles/export

Export a production-ready bundle manifest with checksum.

**Request:**

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/bundles/export" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario": "picker-wheel",
    "tier": "tier-2-desktop",
    "include_secrets": true
  }'
```

**Request Body:** Same as `/bundles/assemble`

**Response:**

```json
{
  "status": "exported",
  "schema": "desktop.v0.1",
  "scenario": "picker-wheel",
  "tier": "tier-2-desktop",
  "manifest": { ... },
  "checksum": "sha256:a1b2c3d4e5f6...",
  "generated_at": "2025-01-15T12:00:00Z"
}
```

**Differences from /bundles/assemble:**
- Includes SHA256 checksum of manifest content
- Includes generation timestamp
- Designed for production artifact creation

---

## Bundle Manifest Schema (v0.1)

### Root Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `schema_version` | string | Yes | Must be "v0.1" |
| `target` | string | Yes | Must be "desktop" |
| `app` | object | Yes | Application metadata |
| `ipc` | object | Yes | Inter-process communication config |
| `telemetry` | object | No | Telemetry configuration |
| `ports` | object | No | Port allocation rules |
| `swaps` | array | No | Dependency swaps applied |
| `secrets` | array | No | Secret definitions |
| `services` | array | Yes | Service definitions |

### App Object

```json
{
  "name": "picker-wheel",
  "version": "1.2.3",
  "description": "Optional description"
}
```

### IPC Object

```json
{
  "mode": "loopback-http",
  "host": "127.0.0.1",
  "port": 47710,
  "auth_token_path": "runtime/auth-token"
}
```

### Secret Object

```json
{
  "id": "jwt_secret",
  "class": "per_install_generated|user_prompt|remote_fetch",
  "description": "JWT signing key",
  "format": "^[A-Za-z0-9]{32,}$",
  "generator": { "type": "random", "length": 32, "charset": "alnum" },
  "prompt": { "label": "...", "description": "..." },
  "target": { "type": "env|file", "name": "JWT_SECRET" }
}
```

### Service Object

```json
{
  "id": "api",
  "type": "api-binary|ui-bundle|resource",
  "description": "API server",
  "binaries": {
    "darwin-arm64": { "path": "services/api/macos-arm64/api" },
    "win-x64": { "path": "services\\api\\windows-x64\\api.exe" }
  },
  "env": { "PORT": "${api.http}" },
  "secrets": ["jwt_secret"],
  "data_dirs": ["data/app"],
  "log_dir": "logs/api",
  "ports": {
    "requested": [{ "name": "http", "range": { "min": 47040, "max": 47060 } }]
  },
  "health": {
    "type": "http|tcp|command",
    "path": "/healthz",
    "port_name": "http",
    "interval_ms": 500,
    "timeout_ms": 2000,
    "retries": 5
  },
  "readiness": {
    "type": "dependencies_ready|health_success|port_open",
    "timeout_ms": 30000
  },
  "dependencies": ["sqlite"],
  "migrations": [
    { "version": "2025-01-01", "command": ["./api", "migrate"], "run_on": "upgrade" }
  ],
  "assets": [
    { "path": "data/seed.sql", "sha256": "...", "size_bytes": 2048 }
  ],
  "gpu": { "requirement": "optional|required" },
  "critical": true
}
```

---

## Related

- [Example Manifests](../examples/manifests/) - Complete bundle.json examples
- [Desktop Workflow](../workflows/desktop-deployment.md) - How bundles are used
- [CLI Reference](../cli/deployment-commands.md) - CLI commands for packaging
