# api-core

Shared Go utilities for Vrooli scenario APIs. Provides automatic startup checks including staleness detection, auto-rebuild, and lifecycle management verification.

## Quick Start

Add to your scenario's `api/go.mod`:

```go
require github.com/vrooli/api-core v0.0.0

replace github.com/vrooli/api-core => ../../../packages/api-core
```

Add preflight checks to your `main()`:

```go
package main

import "github.com/vrooli/api-core/preflight"

func main() {
    // Must be first - before any initialization
    if preflight.Run(preflight.Config{
        ScenarioName: "my-scenario",
    }) {
        return // Process was re-exec'd after rebuild
    }

    // Safe to initialize server, DB connections, etc.
    server := NewServer()
    server.Start()
}
```

## What Preflight Does

The `preflight` package combines two critical startup checks:

1. **Staleness Detection** - Detects if source files changed since compilation and auto-rebuilds
2. **Lifecycle Guard** - Verifies the API is running under the Vrooli lifecycle system

### Flow Diagram

```
vrooli scenario start my-scenario
         │
         ├─► Sets VROOLI_LIFECYCLE_MANAGED=true
         ├─► Sets API_PORT, VROOLI_ROOT, etc.
         │
         └─► Executes: ./my-scenario-api
                  │
                  ▼
            preflight.Run()
                  │
         ┌───────┴───────┐
         │               │
    Binary stale?    Binary fresh?
         │               │
         ▼               ▼
      Rebuild      Lifecycle guard
      Re-exec      (check env var)
         │               │
         ▼               ▼
    New process     Continue to
    starts fresh    initialization
```

## Features

### Auto-Rebuild on Source Changes

When source files are modified after the binary was compiled:

```
api-core: binary is stale (source file modified: pkg/handlers/scores.go)
api-core: rebuilding binary...
api-core: rebuild successful, restarting...
```

### Shared Package Detection

Changes to local dependencies via `replace` directives are detected:

```go
// go.mod
replace github.com/vrooli/api-core => ../../../packages/api-core
```

```
api-core: binary is stale (dependency modified: ../../../packages/api-core (checker.go))
```

### Lifecycle Guard

Prevents direct execution without the lifecycle system:

```
$ ./my-scenario-api
This binary must be run through the Vrooli lifecycle system.

Instead, use:
   vrooli scenario start my-scenario
```

### Graceful Fallbacks

- **No `go` in PATH**: Logs warning, continues with existing binary
- **Build fails**: Logs error, continues with existing binary
- **Rebuild loop**: Detected and prevented (60-second cooldown)

## Configuration

```go
preflight.Run(preflight.Config{
    // Required: scenario name for error messages
    ScenarioName: "my-scenario",

    // Optional: disable staleness check (e.g., in production)
    DisableStaleness: os.Getenv("PRODUCTION") == "true",

    // Optional: detect staleness without rebuilding
    SkipRebuild: false,

    // Optional: disable lifecycle guard (for testing)
    DisableLifecycleGuard: false,

    // Optional: custom logger
    Logger: log.Printf,
})
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VROOLI_LIFECYCLE_MANAGED` | Set by lifecycle system; required for API to start |
| `VROOLI_API_SKIP_STALE_CHECK` | Set to `"true"` to disable staleness checking |

## Package Structure

```
packages/api-core/
├── preflight/              # High-level API (recommended)
│   ├── preflight.go
│   └── preflight_test.go
├── staleness/              # Low-level staleness detection
│   ├── checker.go
│   ├── checker_test.go
│   ├── gomod.go
│   ├── gomod_test.go
│   ├── timestamps.go
│   └── timestamps_test.go
├── docs/                   # Documentation with flow diagrams
│   ├── preflight.md
│   └── staleness.md
├── go.mod
└── README.md
```

## Documentation

- [Preflight Checks](docs/preflight.md) - Detailed preflight documentation with flow diagrams
- [Staleness Detection](docs/staleness.md) - Low-level staleness checker details

## Testing

```bash
cd packages/api-core
go test ./...
```

## Migration from Separate Checks

If your scenario has separate staleness and lifecycle checks:

### Before

```go
func main() {
    checker := staleness.NewChecker(staleness.CheckerConfig{})
    if checker.CheckAndMaybeRebuild() {
        return
    }

    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, "Must run via lifecycle\n")
        os.Exit(1)
    }
    // ...
}
```

### After

```go
func main() {
    if preflight.Run(preflight.Config{
        ScenarioName: "my-scenario",
    }) {
        return
    }
    // ...
}
```

## Currently Using api-core

- `scenario-completeness-scoring` - Reference implementation
