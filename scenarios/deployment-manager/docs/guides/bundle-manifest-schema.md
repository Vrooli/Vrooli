# Bundle Manifest Schema Reference

> **Complete reference for the desktop bundle manifest (`bundle.json`) schema v0.1.**
>
> This document defines all fields, validation rules, and usage patterns for the manifest that drives bundled desktop deployments.

## Overview

The bundle manifest (`bundle.json`) is the central configuration file that the runtime supervisor uses to orchestrate a bundled desktop application. It defines:

- Application metadata
- Service definitions with platform-specific binaries
- Port allocation rules
- Health and readiness checks
- Secret injection configuration
- Database migrations
- Telemetry settings

## Schema Version

**Current version**: `v0.1`

The schema version determines validation rules and supported features. Always specify the version explicitly.

```json
{
  "schema_version": "v0.1",
  "target": "desktop"
}
```

---

## Root Structure

```json
{
  "schema_version": "v0.1",
  "target": "desktop",
  "app": { ... },
  "ipc": { ... },
  "telemetry": { ... },
  "ports": { ... },
  "swaps": [ ... ],
  "secrets": [ ... ],
  "services": [ ... ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `schema_version` | string | **Yes** | Must be `"v0.1"` |
| `target` | string | **Yes** | Must be `"desktop"` for tier 2 |
| `app` | object | **Yes** | Application metadata |
| `ipc` | object | **Yes** | Inter-process communication configuration |
| `telemetry` | object | **Yes** | Telemetry collection settings |
| `ports` | object | No | Port allocation rules |
| `swaps` | array | No | Dependency swaps applied |
| `secrets` | array | No | Secret definitions |
| `services` | array | **Yes** | Service definitions (at least one required) |

---

## App Object

Application metadata displayed to users and used for installers.

```json
{
  "app": {
    "name": "picker-wheel",
    "version": "1.2.3",
    "description": "Offline picker wheel with bundled API and SQLite store"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | **Yes** | Application identifier (alphanumeric, hyphens) |
| `version` | string | **Yes** | Semantic version (e.g., `"1.2.3"`) |
| `description` | string | No | Human-readable description |

**Validation rules:**
- `name` must not be empty
- `version` must not be empty
- `name` is used for directory names, so avoid special characters

---

## IPC Object

Configuration for communication between Electron and the runtime supervisor.

```json
{
  "ipc": {
    "mode": "loopback-http",
    "host": "127.0.0.1",
    "port": 47710,
    "auth_token_path": "runtime/auth-token"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mode` | string | **Yes** | Must be `"loopback-http"` |
| `host` | string | **Yes** | Bind address (typically `"127.0.0.1"`) |
| `port` | integer | **Yes** | Control API port (must be > 0) |
| `auth_token_path` | string | No | Relative path to auth token file |

**Validation rules:**
- `mode` must be `"loopback-http"` (only supported mode in v0.1)
- `host` must not be empty
- `port` must be greater than 0

**Security note:** The runtime control API uses Bearer token authentication. The token is generated on first run and stored at `auth_token_path`.

---

## Telemetry Object

Configuration for deployment telemetry collection.

```json
{
  "telemetry": {
    "file": "telemetry/deployment-telemetry.jsonl",
    "upload_url": "https://api.example.com/v1/telemetry/upload"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file` | string | **Yes** | Relative path for JSONL telemetry file |
| `upload_url` | string | No | URL for automatic telemetry upload |

**Telemetry events recorded:**
- `app_start` - Application launched
- `ready` - All services ready
- `shutdown` - Clean shutdown
- `dependency_unreachable` - Service connection failed
- `migration_failed` - Database migration error
- `secrets_missing` - Required secrets not provided
- `update_check_started`, `update_available`, `update_downloaded`, `update_error`

---

## Ports Object

Port allocation configuration for services.

```json
{
  "ports": {
    "default_range": {
      "min": 47000,
      "max": 47999
    },
    "reserved": [47100, 47200]
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `default_range` | object | No | Default port range for allocation |
| `default_range.min` | integer | **Yes** (if range specified) | Minimum port number |
| `default_range.max` | integer | **Yes** (if range specified) | Maximum port number |
| `reserved` | array[integer] | No | Ports to never allocate |

**Port allocation behavior:**
- Services request ports within their own range (see Service.ports)
- If a service doesn't specify a range, `default_range` is used
- `reserved` ports are never allocated
- Ports are allocated dynamically at runtime and can be referenced via `${service_id.port_name}`

---

## Swaps Array

Documents dependency swaps applied to make the scenario bundle-safe.

```json
{
  "swaps": [
    {
      "original": "postgres",
      "replacement": "sqlite",
      "reason": "Bundle-safe storage",
      "limitations": "Single-user file store; no replication"
    }
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `original` | string | **Yes** | Resource/scenario that was replaced |
| `replacement` | string | **Yes** | Bundle-safe alternative |
| `reason` | string | No | Why the swap was applied |
| `limitations` | string | No | Functionality differences to note |

**Common swaps for desktop:**

| Original | Replacement | Impact |
|----------|-------------|--------|
| `postgres` | `sqlite` | Single-user, file-based, no network required |
| `redis` | `in-process` | Memory-only, no persistence |
| `ollama` | packaged models | Offline AI, larger bundle size |
| `browserless` | `playwright-driver` | Bundled Chromium |

---

## Secrets Array

Secret definitions with injection configuration.

```json
{
  "secrets": [
    {
      "id": "jwt_secret",
      "class": "per_install_generated",
      "description": "JWT signing key",
      "format": "^[A-Za-z0-9]{32,}$",
      "generator": {
        "type": "random",
        "length": 32,
        "charset": "alnum"
      },
      "target": {
        "type": "env",
        "name": "JWT_SECRET"
      }
    },
    {
      "id": "session_secret",
      "class": "user_prompt",
      "description": "Secret for signing cookies and sessions",
      "format": "^[A-Za-z0-9]{32,}$",
      "prompt": {
        "label": "Session secret",
        "description": "Enter a 32+ character random string for signing cookies."
      },
      "target": {
        "type": "env",
        "name": "SESSION_SECRET"
      }
    }
  ]
}
```

### Secret Object Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | **Yes** | Unique identifier for the secret |
| `class` | string | **Yes** | Secret classification (see below) |
| `description` | string | No | Human-readable description |
| `format` | string | No | Regex pattern for validation |
| `required` | boolean | No | Whether the secret is required (default: true) |
| `prompt` | object | **Conditional** | Required for `user_prompt` class |
| `generator` | object | **Conditional** | Required for `per_install_generated` class |
| `target` | object | **Yes** | How the secret is injected |

### Secret Classes

| Class | Description | Bundle Behavior |
|-------|-------------|-----------------|
| `per_install_generated` | Auto-generated on first run | Included with generator config |
| `user_prompt` | User provides during first-run wizard | Included with prompt config |
| `remote_fetch` | Fetched from external vault | Reference only |
| `infrastructure` | **Never bundled** | Excluded from manifest |

### Prompt Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `label` | string | **Yes** | Short label for UI |
| `description` | string | No | Help text shown to user |

### Generator Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | **Yes** | Generator type: `"random"`, `"uuid"` |
| `length` | integer | Conditional | Required for `random` type |
| `charset` | string | No | Character set: `"alnum"`, `"hex"`, `"base64"` |

### Target Object

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | **Yes** | Injection type: `"env"` or `"file"` |
| `name` | string | **Yes** | Environment variable name or file path |

---

## Services Array

Service definitions - at least one required.

```json
{
  "services": [
    {
      "id": "api",
      "type": "api-binary",
      "description": "Picker Wheel API with embedded SQLite",
      "binaries": { ... },
      "env": { ... },
      "secrets": ["jwt_secret", "session_secret"],
      "data_dirs": ["data/picker-wheel"],
      "log_dir": "logs/api",
      "ports": { ... },
      "health": { ... },
      "readiness": { ... },
      "dependencies": ["sqlite"],
      "migrations": [ ... ],
      "assets": [ ... ],
      "gpu": { ... },
      "critical": true
    }
  ]
}
```

### Service Object Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | **Yes** | Unique identifier for the service |
| `type` | string | **Yes** | Service type (see below) |
| `description` | string | No | Human-readable description |
| `binaries` | object | **Yes** | Platform-specific binary configurations |
| `env` | object | No | Environment variables (supports templates) |
| `secrets` | array[string] | No | Secret IDs to inject |
| `data_dirs` | array[string] | No | Directories for persistent data |
| `log_dir` | string | No | Directory for log output |
| `ports` | object | No | Port allocation configuration |
| `health` | object | **Yes** | Health check configuration |
| `readiness` | object | **Yes** | Readiness check configuration |
| `dependencies` | array[string] | No | Service IDs that must be ready first |
| `migrations` | array | No | Database migration commands |
| `assets` | array | No | Bundled assets with checksums |
| `gpu` | object | No | GPU requirements |
| `critical` | boolean | No | If false, deployment continues if service fails |

### Service Types

| Type | Description |
|------|-------------|
| `ui-bundle` | Serves bundled UI assets over HTTP |
| `api-binary` | Go/Rust/compiled API binary |
| `resource` | Supporting service (database, cache, etc.) |

### Binaries Object

Map of platform keys to binary configurations.

```json
{
  "binaries": {
    "darwin-arm64": {
      "path": "services/api/macos-arm64/picker-wheel-api",
      "args": ["--port", "${api.http}"],
      "env": { "EXTRA_VAR": "value" },
      "cwd": "services/api"
    },
    "darwin-x64": { "path": "services/api/macos-x64/picker-wheel-api" },
    "linux-x64": { "path": "services/api/linux-x64/picker-wheel-api" },
    "win-x64": { "path": "services\\api\\windows-x64\\picker-wheel-api.exe" }
  }
}
```

**Platform keys:**
- `darwin-arm64` - macOS Apple Silicon
- `darwin-x64` (alias: `mac-x64`) - macOS Intel
- `linux-x64` - Linux x86_64
- `linux-arm64` - Linux ARM64
- `win-x64` (alias: `windows-x64`) - Windows x86_64

**Binary object fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | **Yes** | Relative path to binary |
| `args` | array[string] | No | Command-line arguments |
| `env` | object | No | Additional environment variables |
| `cwd` | string | No | Working directory (relative to bundle) |

### Environment Variable Templates

Environment values can reference allocated ports and data directories:

| Template | Description | Example |
|----------|-------------|---------|
| `${service_id.port_name}` | Allocated port for a service | `${api.http}` |
| `${data}` | Data directory root | `${data}/picker-wheel.db` |

```json
{
  "env": {
    "HTTP_PORT": "${api.http}",
    "DATABASE_URL": "file:${data}/picker-wheel.db",
    "LOG_LEVEL": "info"
  }
}
```

### Ports Object (Service-Level)

```json
{
  "ports": {
    "requested": [
      {
        "name": "http",
        "range": { "min": 47040, "max": 47060 },
        "requires_socket": false
      }
    ]
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `requested` | array | **Yes** | List of port requests |
| `requested[].name` | string | **Yes** | Port name for templating |
| `requested[].range` | object | **Yes** | Port range for allocation |
| `requested[].requires_socket` | boolean | No | If true, creates Unix socket |

### Health Check Object

```json
{
  "health": {
    "type": "http",
    "path": "/healthz",
    "port_name": "http",
    "interval_ms": 500,
    "timeout_ms": 2000,
    "retries": 5
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | **Yes** | Check type: `"http"`, `"tcp"`, `"command"` |
| `path` | string | Conditional | HTTP path (required for `http` type) |
| `port_name` | string | Conditional | Port to check (required for `http`/`tcp`) |
| `command` | array[string] | Conditional | Command to run (required for `command` type) |
| `interval_ms` | integer | No | Check interval in milliseconds |
| `timeout_ms` | integer | No | Timeout per check |
| `retries` | integer | No | Failures before unhealthy |

**Health check types:**
- `http` - HTTP GET to `path` on `port_name`, expects 2xx response
- `tcp` - TCP connection to `port_name`
- `command` - Execute command, expects exit code 0

### Readiness Check Object

```json
{
  "readiness": {
    "type": "health_success",
    "timeout_ms": 30000
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | **Yes** | Check type (see below) |
| `port_name` | string | Conditional | Port name (for `port_open` type) |
| `pattern` | string | No | Log pattern to match (for `log_match` type) |
| `timeout_ms` | integer | No | Maximum wait time |

**Readiness types:**
- `dependencies_ready` - Wait for all `dependencies` to be ready
- `health_success` - Wait for health check to pass
- `port_open` - Wait for port to accept connections
- `log_match` - Wait for pattern in logs (for services with slow startup)

### Migration Object

```json
{
  "migrations": [
    {
      "version": "2025-01-01T00:00:00Z",
      "command": ["./picker-wheel-api", "migrate", "--dsn", "file:${data}/picker-wheel.db"],
      "env": { "MIGRATION_MODE": "up" },
      "run_on": "upgrade"
    }
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | **Yes** | Migration version identifier |
| `command` | array[string] | **Yes** | Migration command with arguments |
| `env` | object | No | Environment variables for migration |
| `run_on` | string | **Yes** | When to run: `"install"`, `"upgrade"`, `"always"` |

### Asset Object

```json
{
  "assets": [
    {
      "path": "data/picker-wheel/seed.sql",
      "sha256": "abc123...",
      "size_bytes": 2048
    }
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | **Yes** | Relative path to asset |
| `sha256` | string | **Yes** | SHA256 checksum for verification |
| `size_bytes` | integer | No | File size in bytes |

### GPU Object

```json
{
  "gpu": {
    "requirement": "optional"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `requirement` | string | **Yes** | `"optional"`, `"required"`, `"optional_with_cpu_fallback"` |

---

## Build Configuration

When binaries don't exist for a platform, the packager can automatically compile them if a `build` configuration is provided.

### Build Object

```json
{
  "services": [{
    "id": "api",
    "binaries": { ... },
    "build": {
      "type": "go",
      "source_dir": "api",
      "entry_point": "./cmd/api",
      "output_pattern": "bin/{{platform}}/my-api{{ext}}",
      "args": ["-ldflags", "-s -w"],
      "env": {
        "CGO_ENABLED": "0"
      }
    }
  }]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | **Yes** | Build system: `"go"`, `"rust"`, `"npm"`, `"custom"` |
| `source_dir` | string | **Yes** | Path to source code relative to manifest |
| `entry_point` | string | No | Entry point (default: `.` for Go, `Cargo.toml` for Rust) |
| `output_pattern` | string | No | Output path pattern with placeholders |
| `args` | array[string] | No | Additional build arguments |
| `env` | object | No | Environment variables for build |

### Build Types

| Type | Tool | Notes |
|------|------|-------|
| `go` | `go build` | Sets `CGO_ENABLED=0`, `GOOS`, `GOARCH` automatically |
| `rust` | `cargo build --release` | Maps to Rust target triples |
| `npm` | `npm install && npm run build` | Requires bundler that outputs platform binaries |
| `custom` | User-specified | First `args` element is the command |

### Build Placeholders

| Placeholder | Example Values | Description |
|-------------|----------------|-------------|
| `{{platform}}` | `linux-x64`, `darwin-arm64`, `win-x64` | Combined OS-arch key |
| `{{goos}}` | `linux`, `darwin`, `windows` | Go OS identifier |
| `{{goarch}}` | `amd64`, `arm64` | Go architecture identifier |
| `{{ext}}` | `` (empty) or `.exe` | Platform-specific extension |
| `{{output}}` | `/path/to/binary` | Full output path |

### Go Build Example

```json
{
  "build": {
    "type": "go",
    "source_dir": "api",
    "entry_point": "./cmd/api",
    "output_pattern": "bin/{{platform}}/my-api{{ext}}",
    "args": ["-ldflags", "-s -w"],
    "env": { "CGO_ENABLED": "0" }
  }
}
```

### Rust Build Example

```json
{
  "build": {
    "type": "rust",
    "source_dir": "worker",
    "entry_point": "worker",
    "output_pattern": "target/release/worker{{ext}}"
  }
}
```

Rust target mapping:
- `linux-x64` → `x86_64-unknown-linux-gnu`
- `linux-arm64` → `aarch64-unknown-linux-gnu`
- `darwin-x64` → `x86_64-apple-darwin`
- `darwin-arm64` → `aarch64-apple-darwin`
- `win-x64` → `x86_64-pc-windows-msvc`

### Custom Build Example

```json
{
  "build": {
    "type": "custom",
    "source_dir": "tools",
    "output_pattern": "bin/{{platform}}/tool{{ext}}",
    "args": ["make", "build", "GOOS={{goos}}", "GOARCH={{goarch}}", "OUTPUT={{output}}"]
  }
}
```

---

## Code Signing (Future)

> **Note**: Code signing configuration is planned but not yet implemented in v0.1. See [ROADMAP.md](../ROADMAP.md).

When implemented, code signing will be configured as:

```json
{
  "code_signing": {
    "enabled": true,
    "windows": {
      "certificate_file": "./certs/windows.pfx",
      "certificate_password_env": "WIN_CERT_PASSWORD",
      "timestamp_server": "http://timestamp.digicert.com"
    },
    "macos": {
      "identity": "Developer ID Application: Your Name (TEAMID)",
      "team_id": "TEAMID",
      "hardened_runtime": true,
      "notarize": true,
      "apple_id_env": "APPLE_ID",
      "apple_id_password_env": "APPLE_APP_PASSWORD"
    },
    "linux": {
      "gpg_key_id": "YOUR_GPG_KEY",
      "gpg_key_passphrase_env": "GPG_PASSPHRASE"
    }
  }
}
```

**Current workaround**: Configure signing directly in `package.json`:

```json
{
  "build": {
    "win": {
      "certificateFile": "./cert.pfx",
      "certificatePassword": "${WIN_CSC_KEY_PASSWORD}",
      "signAndEditExecutable": true
    },
    "mac": {
      "hardenedRuntime": true,
      "gatekeeperAssess": true,
      "entitlements": "build/entitlements.mac.plist"
    }
  }
}
```

**Platform requirements:**

| Platform | Requirement | Cost |
|----------|-------------|------|
| Windows | Authenticode certificate | $200-400/year |
| macOS | Apple Developer account | $99/year |
| Linux | GPG key (optional) | Free |

---

## Complete Example

See [desktop-happy.json](../examples/manifests/desktop-happy.json) for a complete working example.

---

## Validation Rules Summary

A manifest is **valid** when:

1. `schema_version` equals `"v0.1"`
2. `target` equals `"desktop"`
3. `app.name` and `app.version` are non-empty
4. `ipc.mode` equals `"loopback-http"`
5. `ipc.host` and `ipc.port` are specified
6. At least one service is defined
7. Each service has:
   - Non-empty `id`
   - At least one binary for a supported platform
   - `health.type` and `readiness.type` specified
8. All secrets have valid `class` values
9. All `per_install_generated` secrets have `generator` config
10. All `user_prompt` secrets have `prompt` config

---

## API Endpoints

| Endpoint | Purpose |
|----------|---------|
| `POST /api/v1/bundles/validate` | Validate a manifest |
| `POST /api/v1/bundles/assemble` | Assemble manifest from scenario |
| `POST /api/v1/bundles/export` | Export manifest with checksum |
| `POST /api/v1/bundles/merge-secrets` | Merge secret plans into manifest |

See [API Reference - Bundles](../api/bundles.md) for details.

---

## Related Documentation

- [Example Manifests](../examples/manifests/) - Working examples
- [Secrets Management](secrets-management.md) - Secret classification and handling
- [Desktop Workflow](../workflows/desktop-deployment.md) - End-to-end deployment guide
- [Fitness Scoring](fitness-scoring.md) - How compatibility is calculated
