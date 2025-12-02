# ports - Dynamic Port Allocation

The `ports` package manages dynamic port allocation for services, ensuring each service gets an available port within its configured range.

## Overview

Desktop bundles often need multiple services running simultaneously. Rather than hardcoding ports (which may conflict with other applications), the runtime dynamically allocates ports at startup. This package handles port selection, conflict detection, and port mapping.

## Key Types

| Type | Purpose |
|------|---------|
| `Allocator` | Interface for port allocation |
| `Manager` | Production implementation of `Allocator` |
| `Range` | Port range configuration (`Min`, `Max`) |

## Interface

```go
type Allocator interface {
    // Allocate assigns ports to all services
    Allocate() error

    // Resolve returns the allocated port for a service
    Resolve(serviceID, portName string) (int, bool)

    // Map returns all allocated ports
    Map() map[string]map[string]int
}
```

## Usage

```go
// Create manager from manifest
manager := ports.NewManager(manifest, dialer)

// Allocate all ports
if err := manager.Allocate(); err != nil {
    log.Fatal("port allocation failed:", err)
}

// Get allocated port for a service
port, ok := manager.Resolve("api", "http")
if !ok {
    log.Fatal("port not allocated")
}

// Get full port map for templates
portMap := manager.Map()
// portMap["api"]["http"] = 47001
```

## Allocation Strategy

1. For each service, iterate through requested ports
2. For each port, find an available port in its configured range
3. Test availability by attempting to dial the port
4. If available (connection refused), allocate it
5. If range exhausted, return error

## Default Range

When no range is specified, ports are allocated from `47000-48000` (configurable in manifest).

## Reserved Ports

The manifest can specify reserved ports (e.g., the control API port) that won't be allocated to services.

## Template Variables

Allocated ports are available in environment templates:

```json
{
  "env": {
    "API_PORT": "${api.http}"
  }
}
```

## Dependencies

- **Depends on**: `manifest`, `infra` (NetworkDialer)
- **Depended on by**: `bundleruntime`, `health`, `env`
