# API Preflight Checks

The `preflight` package provides startup checks that must run before any API initialization. It combines staleness detection with lifecycle management verification into a single, easy-to-use function.

## Quick Start

```go
package main

import "github.com/vrooli/api-core/preflight"

func main() {
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

Preflight runs two checks in a specific order:

1. **Staleness Check** - Detects if source files changed since compilation
2. **Lifecycle Guard** - Verifies the API is running under the lifecycle system

### Why Order Matters

The staleness check must run first because:
- It may rebuild and re-exec the process
- Re-exec preserves all environment variables
- `VROOLI_LIFECYCLE_MANAGED` remains set after re-exec

If the order were reversed, the lifecycle guard would fail before staleness could check/rebuild.

## Flow Diagrams

### Normal Startup (Fresh Binary)

```
vrooli scenario start my-scenario
         │
         ├─► Sets VROOLI_LIFECYCLE_MANAGED=true
         ├─► Sets API_PORT=8120
         ├─► Sets VROOLI_ROOT=/home/user/Vrooli
         │
         └─► Executes: ./my-scenario-api
                  │
                  ▼
            preflight.Run()
                  │
                  ▼
         ┌───────────────────┐
         │  Staleness Check  │
         │  Binary fresh? ✓  │
         └─────────┬─────────┘
                   │
                   ▼
         ┌───────────────────┐
         │  Lifecycle Guard  │
         │  Env var set? ✓   │
         └─────────┬─────────┘
                   │
                   ▼
            Returns false
                   │
                   ▼
         Server initialization
                   │
                   ▼
            API running
```

### Stale Binary (Auto-Rebuild)

```
vrooli scenario start my-scenario
         │
         └─► Executes: ./my-scenario-api (stale)
                  │
                  ▼
            preflight.Run()
                  │
                  ▼
         ┌───────────────────┐
         │  Staleness Check  │
         │  Binary stale!    │
         └─────────┬─────────┘
                   │
                   ▼
         ┌───────────────────┐
         │  go build -o ...  │
         │  Rebuild binary   │
         └─────────┬─────────┘
                   │
                   ▼
         ┌───────────────────┐
         │  syscall.Exec()   │
         │  Replace process  │
         │  (preserves env)  │
         └─────────┬─────────┘
                   │
                   ▼
            preflight.Run()     ◄── New process, same PID
                   │
                   ▼
         ┌───────────────────┐
         │  Staleness Check  │
         │  Binary fresh? ✓  │
         └─────────┬─────────┘
                   │
                   ▼
         ┌───────────────────┐
         │  Lifecycle Guard  │
         │  Env var set? ✓   │
         └─────────┬─────────┘
                   │
                   ▼
            Returns false
                   │
                   ▼
            API running (new code)
```

### Direct Execution (Blocked)

```
./my-scenario-api          ◄── Direct execution (no lifecycle)
         │
         ▼
   preflight.Run()
         │
         ▼
┌───────────────────┐
│  Staleness Check  │
│  (runs normally)  │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│  Lifecycle Guard  │
│  Env var set? ✗   │
└─────────┬─────────┘
          │
          ▼
   Print error message:
   "Must run via vrooli scenario start my-scenario"
          │
          ▼
     os.Exit(1)
```

### Staleness Detection Details

```
Binary modified: 2024-01-15 10:00:00
         │
         ▼
┌─────────────────────────────────────────────┐
│           Files Checked for Staleness        │
├─────────────────────────────────────────────┤
│                                             │
│  api/*.go                    ─► Modified?   │
│  api/pkg/**/*.go             ─► Modified?   │
│  api/go.mod                  ─► Modified?   │
│  api/go.sum                  ─► Modified?   │
│                                             │
│  Replace directives:                        │
│  ../../../packages/api-core  ─► Modified?   │
│  ../../../packages/proto     ─► Modified?   │
│                                             │
└─────────────────────────────────────────────┘
         │
         ▼
   Any file newer than binary?
         │
    ┌────┴────┐
    │         │
   Yes        No
    │         │
    ▼         ▼
 Rebuild   Continue
```

## Configuration Reference

```go
type Config struct {
    // ScenarioName is used in error messages to guide users.
    // Example: "my-scenario" produces "vrooli scenario start my-scenario"
    ScenarioName string

    // DisableStaleness skips the staleness check entirely.
    // Use in production where rebuilds aren't desired.
    DisableStaleness bool

    // SkipRebuild detects staleness but doesn't rebuild.
    // Useful for debugging staleness detection.
    SkipRebuild bool

    // DisableLifecycleGuard skips the lifecycle management check.
    // Use for testing outside the lifecycle system.
    DisableLifecycleGuard bool

    // Logger overrides the default stderr logger.
    Logger func(format string, args ...interface{})

    // StalenessConfig provides additional staleness checker configuration.
    // Most scenarios can leave this nil for defaults.
    StalenessConfig *staleness.CheckerConfig
}
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VROOLI_LIFECYCLE_MANAGED` | Set to `"true"` by the lifecycle system |
| `VROOLI_API_SKIP_STALE_CHECK` | Set to `"true"` to disable staleness checking |
| `API_CORE_REBUILD_TIMESTAMP` | Internal: prevents infinite rebuild loops |

## Best Practices

### DO: Call preflight first in main()

```go
func main() {
    // ✅ First thing in main()
    if preflight.Run(preflight.Config{ScenarioName: "my-scenario"}) {
        return
    }

    // Now safe to initialize
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    server := NewServer()
}
```

### DON'T: Initialize anything before preflight

```go
func main() {
    // ❌ BAD: This runs before preflight
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    db := openDatabase()

    if preflight.Run(...) {
        return // db connection leaked!
    }
}
```

### DO: Always provide ScenarioName

```go
// ✅ Good error message
preflight.Run(preflight.Config{
    ScenarioName: "my-scenario",  // "vrooli scenario start my-scenario"
})

// ❌ Generic error message
preflight.Run(preflight.Config{})  // "vrooli scenario start <scenario-name>"
```

### DO: Disable staleness in production (optional)

```go
preflight.Run(preflight.Config{
    ScenarioName:     "my-scenario",
    DisableStaleness: os.Getenv("PRODUCTION") == "true",
})
```

## Graceful Error Handling

Preflight handles errors gracefully:

| Situation | Behavior |
|-----------|----------|
| No `go` in PATH | Logs warning, continues with existing binary |
| Build fails | Logs error, continues with existing binary |
| No go.mod | Skips staleness check (not a Go module) |
| Rebuild loop detected | Logs warning, continues (prevents infinite rebuilds) |
| Lifecycle guard fails | Prints helpful message, exits with code 1 |

## Migration from Separate Checks

If your scenario currently has separate staleness and lifecycle checks:

### Before

```go
func main() {
    // Staleness check
    checker := staleness.NewChecker(staleness.CheckerConfig{})
    if checker.CheckAndMaybeRebuild() {
        return
    }

    // Lifecycle guard
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, "Must run via lifecycle\n")
        os.Exit(1)
    }

    // ... rest of main
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

    // ... rest of main
}
```
