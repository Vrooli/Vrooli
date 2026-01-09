# env - Environment Variable Templating

The `env` package renders environment variable templates with runtime values like paths, ports, and system information.

## Overview

Services need environment variables configured with runtime-specific values: data directories, allocated ports, bundle paths, etc. This package provides a template engine that expands placeholder variables in manifest-defined environment values.

## Key Types

| Type | Purpose |
|------|---------|
| `Renderer` | Template expansion engine |

## Template Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `${data}` | App data directory | `/home/user/.config/myapp` |
| `${bundle}` | Bundle root path | `/opt/myapp` |
| `${svc.port}` | Allocated port for service | `${api.http}` → `47001` |
| `${env.VAR}` | System environment variable | `${env.HOME}` → `/home/user` |

## Usage

```go
// Create renderer
renderer := env.NewRenderer(appData, bundlePath, portAllocator, envReader)

// Render a single value
path := renderer.RenderValue("${data}/logs")
// → "/home/user/.config/myapp/logs"

// Render command arguments
args := renderer.RenderArgs([]string{"--port", "${api.http}", "--data", "${data}"})
// → ["--port", "47001", "--data", "/home/user/.config/myapp"]

// Render full environment map for a service
envMap, err := renderer.RenderEnvMap(service, binary)
// Merges: OS env → service.env → binary.env (with template expansion)
```

## Environment Precedence

When building the environment for a service:

1. **OS Environment**: Inherited from parent process
2. **Service Env**: From `service.env` in manifest
3. **Binary Env**: From `binary.env` in manifest (overrides service)

Later sources override earlier ones for the same key.

## Manifest Example

```json
{
  "services": [{
    "id": "api",
    "env": {
      "DATA_DIR": "${data}/api",
      "BUNDLE_PATH": "${bundle}",
      "DB_PORT": "${db.postgres}",
      "LOG_LEVEL": "info"
    }
  }]
}
```

## Special Handling

- Empty template variables (`${}`) are passed through unchanged
- Unknown service/port combinations return empty string
- System env vars that don't exist return empty string

## Dependencies

- **Depends on**: `infra` (EnvReader), `ports` (Allocator)
- **Depended on by**: `bundleruntime`
