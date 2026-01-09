# manifest - Bundle Manifest Schema

The `manifest` package defines the data structures for parsing and validating `bundle.json` manifest files.

## Overview

Every desktop bundle includes a `bundle.json` manifest that describes the application, its services, secrets, port requirements, and health checks. This package provides the Go types and loading functions for that schema.

## Key Types

| Type | Purpose |
|------|---------|
| `Manifest` | Root structure containing all bundle configuration |
| `App` | Application metadata (name, version) |
| `IPC` | Control API configuration (host, port, auth) |
| `Telemetry` | Telemetry file path and upload URL |
| `PortConfig` | Default port range and reserved ports |
| `Secret` | Secret definition with prompts and targets |
| `Service` | Service definition with binaries, health checks, etc. |
| `Binary` | Platform-specific executable configuration |
| `Health` | Health check configuration (HTTP, TCP, command, log_match) |
| `Readiness` | Readiness check configuration |
| `Migration` | Schema migration definition |
| `Asset` | Asset verification rules (size, checksum) |

## Usage

```go
// Load manifest from file
m, err := manifest.LoadManifest("bundle.json")
if err != nil {
    log.Fatal(err)
}

// Access configuration
fmt.Println("App:", m.App.Name, m.App.Version)
for _, svc := range m.Services {
    fmt.Printf("Service: %s (type: %s)\n", svc.ID, svc.Type)
}

// Resolve platform-specific binary
bin, ok := m.ResolveBinary(svc)
if !ok {
    log.Fatal("no binary for current platform")
}

// Resolve paths relative to bundle
cmdPath := manifest.ResolvePath(bundlePath, bin.Path)
```

## Helper Functions

| Function | Purpose |
|----------|---------|
| `LoadManifest(path)` | Load and validate a manifest file |
| `ResolveBinary(svc)` | Get the binary for the current OS/arch |
| `ResolvePath(base, rel)` | Join paths safely |
| `CurrentPlatform()` | Get current platform key (e.g., "linux-x64") |

## Schema Version

Current schema version: `0.1`

See the main `README.md` for a complete manifest example.

## Validation

The loader performs basic validation:
- Required fields are present
- Schema version is supported
- Service IDs are unique
- Referenced secrets exist

## Dependencies

- **Depends on**: Standard library (`encoding/json`, `os`, `runtime`)
- **Depended on by**: `bundleruntime`, `ports`, `secrets`, `health`, `assets`
