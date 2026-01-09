# health - Health and Readiness Monitoring

The `health` package provides health checking and readiness monitoring for managed services.

## Overview

Services need time to start up and may become unhealthy during operation. This package implements various health check strategies and readiness gates to ensure services are fully operational before dependent services start or traffic is routed.

## Key Types

| Type | Purpose |
|------|---------|
| `Checker` | Interface for health monitoring |
| `Monitor` | Production implementation of `Checker` |
| `Status` | Service status with ready flag and message |
| `MonitorConfig` | Configuration for creating a Monitor |

## Checker Interface

```go
type Checker interface {
    // WaitForReadiness blocks until service is ready or context cancels
    WaitForReadiness(ctx context.Context, serviceID string) error

    // CheckOnce performs a single health check
    CheckOnce(ctx context.Context, serviceID string) bool

    // WaitForDependencies waits for all service dependencies to be ready
    WaitForDependencies(ctx context.Context, svc *manifest.Service) error
}
```

## Health Check Types

| Type | Description | Configuration |
|------|-------------|---------------|
| `http` | HTTP GET, expect 2xx response | `path`, `port_name` |
| `tcp` | TCP connection attempt | `port_name` |
| `command` | Execute command, expect exit 0 | `command` |
| `log_match` | Regex match in log file | `path` (regex pattern) |

## Readiness Check Types

| Type | Description | Configuration |
|------|-------------|---------------|
| `health_success` | Poll health check until success | `timeout_ms` |
| `port_open` | Wait for TCP port to accept | `port_name`, `timeout_ms` |
| `log_match` | Wait for pattern in logs | `pattern`, `timeout_ms` |
| `dependencies_ready` | Wait for dependent services | (none) |

## Usage

```go
// Create monitor
monitor := health.NewMonitor(health.MonitorConfig{
    Manifest:     manifest,
    Ports:        portAllocator,
    Dialer:       dialer,
    CmdRunner:    cmdRunner,
    FS:           fs,
    Clock:        clock,
    AppData:      appData,
    StatusGetter: supervisor.getStatus,
})

// Wait for service readiness
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := monitor.WaitForReadiness(ctx, "api"); err != nil {
    log.Printf("service not ready: %v", err)
}

// Wait for dependencies before starting dependent service
if err := monitor.WaitForDependencies(ctx, &webService); err != nil {
    log.Printf("dependencies not ready: %v", err)
}
```

## Retry Configuration

Health checks are retried with configurable:
- `interval_ms`: Time between checks (default: 1000ms)
- `timeout_ms`: Single check timeout (default: 5000ms)
- `retries`: Number of failures before giving up (default: 3)

## Manifest Example

```json
{
  "health": {
    "type": "http",
    "path": "/health",
    "port_name": "http",
    "interval_ms": 1000,
    "timeout_ms": 5000,
    "retries": 3
  },
  "readiness": {
    "type": "health_success",
    "timeout_ms": 30000
  }
}
```

## Dependencies

- **Depends on**: `manifest`, `infra` (Clock, NetworkDialer, CommandRunner, FileSystem), `ports`
- **Depended on by**: `bundleruntime`
