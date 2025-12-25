# api-core

Shared Go utilities for Vrooli scenario APIs. Provides:

- **Preflight checks** - Staleness detection, auto-rebuild, lifecycle management
- **Database connections** - Auto-configured from environment with retry and backoff
- **Scenario discovery** - Runtime port resolution for inter-scenario communication
- **Retry utilities** - Exponential backoff with jitter for reliable connections

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

## Database Connections

The `database` package provides database connections with automatic configuration from environment variables and retry with exponential backoff.

### Basic Usage

```go
import (
    "context"
    "log"

    "github.com/vrooli/api-core/database"
)

func main() {
    // Preflight checks first...

    // Connect to database - reads POSTGRES_* from environment automatically
    db, err := database.Connect(context.Background(), database.Config{
        Driver: "postgres",
    })
    if err != nil {
        log.Fatalf("Database connection failed: %v", err)
    }
    defer db.Close()

    // Use db...
}
```

### How It Works

For known drivers (`postgres`, `sqlite3`), connection parameters are automatically read from environment variables set by the Vrooli lifecycle system:

| Variable | Description |
|----------|-------------|
| `POSTGRES_HOST` | Database host (required) |
| `POSTGRES_PORT` | Database port (required) |
| `POSTGRES_USER` | Username (required) |
| `POSTGRES_PASSWORD` | Password (optional for local dev) |
| `POSTGRES_DB` | Database name (required) |
| `POSTGRES_SSLMODE` | SSL mode (default: `disable`) |
| `POSTGRES_URL` | Complete URL (used if set, overrides components) |
| `POSTGRES_POOL_SIZE` | Connection pool size (default: 25) |

### Retry Behavior

Connections are retried automatically with exponential backoff and jitter:

- **10 attempts** by default
- **500ms** base delay, growing exponentially
- **30s** maximum delay
- **25% jitter** to prevent thundering herd

```go
// Custom retry configuration
db, err := database.Connect(ctx, database.Config{
    Driver: "postgres",
    Retry: &retry.Config{
        MaxAttempts: 5,
        BaseDelay:   time.Second,
        MaxDelay:    time.Minute,
    },
})
```

### Custom Pool Settings

```go
db, err := database.Connect(ctx, database.Config{
    Driver:          "postgres",
    MaxOpenConns:    50,
    MaxIdleConns:    10,
    ConnMaxLifetime: 10 * time.Minute,
})
```

### Explicit DSN (Bypass Environment)

```go
db, err := database.Connect(ctx, database.Config{
    Driver: "postgres",
    DSN:    "postgres://user:pass@host:5432/db?sslmode=require",
})
```

### SQLite Support

```go
// Reads SQLITE_PATH or SQLITE_DB from environment
db, err := database.Connect(ctx, database.Config{
    Driver: "sqlite3",
})

// Or explicit path
db, err := database.Connect(ctx, database.Config{
    Driver: "sqlite3",
    DSN:    "/data/app.db",
})
```

### Testing

```go
func TestDatabaseLogic(t *testing.T) {
    cfg := database.Config{
        Driver: "postgres",
        EnvGetter: func(key string) string {
            // Return test values
            env := map[string]string{
                "POSTGRES_HOST": "testhost",
                "POSTGRES_PORT": "5432",
                // ...
            }
            return env[key]
        },
        Opener: func(driver, dsn string) (*sql.DB, error) {
            // Return mock DB
            return sqlmock.New()
        },
        Retry: &retry.Config{
            MaxAttempts: 1,
            Sleeper:     func(d time.Duration) {}, // No-op for fast tests
        },
    }

    db, err := database.Connect(context.Background(), cfg)
    // ...
}
```

## Retry Utilities

The `retry` package provides generic retry logic with exponential backoff and jitter. Used internally by `database.Connect()` but available for any retry needs.

### Basic Usage

```go
import (
    "context"
    "github.com/vrooli/api-core/retry"
)

err := retry.Do(ctx, retry.DefaultConfig(), func(attempt int) error {
    return someOperation()
})
```

### Custom Configuration

```go
cfg := retry.Config{
    MaxAttempts:    5,           // Stop after 5 attempts
    BaseDelay:      time.Second, // Start with 1s delay
    MaxDelay:       time.Minute, // Cap at 1 minute
    JitterFraction: 0.25,        // Add up to 25% random jitter
    OnRetry: func(attempt int, err error, delay time.Duration) {
        log.Printf("Attempt %d failed: %v (retrying in %v)", attempt, err, delay)
    },
}

err := retry.Do(ctx, cfg, func(attempt int) error {
    return connectToService()
})
```

### Delay Calculation

```
delay = min(baseDelay * 2^attempt, maxDelay) + random(0, delay * jitterFraction)
```

Example with defaults (500ms base, 30s max, 25% jitter):
- Attempt 0: 500ms + 0-125ms jitter
- Attempt 1: 1s + 0-250ms jitter
- Attempt 2: 2s + 0-500ms jitter
- Attempt 3: 4s + 0-1s jitter
- ...
- Attempt 6+: 30s + 0-7.5s jitter (capped)

### Testing

```go
func TestRetryLogic(t *testing.T) {
    var delays []time.Duration

    cfg := retry.Config{
        MaxAttempts:    3,
        BaseDelay:      100 * time.Millisecond,
        JitterFraction: 0, // Disable jitter for predictable tests
        Sleeper:        func(d time.Duration) { delays = append(delays, d) },
    }

    attempts := 0
    retry.Do(ctx, cfg, func(attempt int) error {
        attempts++
        if attempts < 3 {
            return errors.New("not ready")
        }
        return nil
    })

    // Verify: 2 sleeps (100ms, 200ms), succeeded on 3rd attempt
}
```

## Scenario Discovery

`api-core/discovery` resolves scenario ports at runtime by invoking the Vrooli CLI
every call (no caching). This avoids hard-coded or stale ports when scenarios
restart on new allocations.

```go
import (
    "context"
    "log"

    "github.com/vrooli/api-core/discovery"
)

func callOtherScenario() {
    resolver := discovery.NewResolver(discovery.ResolverConfig{})
    url, err := resolver.ResolveScenarioURLDefault(context.Background(), "other-scenario")
    if err != nil {
        log.Printf("resolve failed: %v", err)
        return
    }
    // url -> http://localhost:<port>
}
```

### Testing with Static URLs

For unit tests using `httptest.Server`, use `NewStaticResolver` to bypass CLI discovery:

```go
func TestMyClient(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    resolver := discovery.NewStaticResolver(server.URL)
    client := NewClient(resolver, server.Client())

    // Test your client...
}
```

This replaces the need for complex mocking of the `CommandRunner`:

```go
// Before: ~15 lines of URL parsing and CommandRunner mocking
resolver := discovery.NewResolver(discovery.ResolverConfig{
    Host:   host,
    Scheme: parsed.Scheme,
    CommandRunner: func(ctx context.Context, name string, args ...string) ([]byte, error) {
        return []byte(port), nil
    },
})

// After: 1 line
resolver := discovery.NewStaticResolver(server.URL)
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

Changes to local dependencies via `replace` directives are detected. For each local dependency, the following are monitored:
- Source files (`*.go`)
- `go.mod` changes
- `go.sum` changes (dependency version updates)

```go
// go.mod
replace github.com/vrooli/api-core => ../../../packages/api-core
```

```
api-core: binary is stale (dependency modified: ../../../packages/api-core (checker.go))
api-core: binary is stale (dependency go.sum modified: ../../../packages/api-core)
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

| Variable | Set By | Description |
|----------|--------|-------------|
| `VROOLI_LIFECYCLE_MANAGED` | Lifecycle system | Required for API to start; set to `"true"` |
| `VROOLI_API_SKIP_STALE_CHECK` | User/Environment | Set to `"true"` to disable staleness checking |
| `API_CORE_REBUILD_TIMESTAMP` | api-core (internal) | Unix timestamp of last rebuild; used for loop detection |

## Package Structure

```
packages/api-core/
├── database/               # Database connections with retry
│   ├── connect.go
│   └── connect_test.go
├── discovery/              # Scenario port resolution
│   ├── resolve.go
│   └── resolve_test.go
├── preflight/              # Startup checks (recommended entry point)
│   ├── preflight.go
│   └── preflight_test.go
├── retry/                  # Exponential backoff with jitter
│   ├── backoff.go
│   └── backoff_test.go
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

**Preflight:**
- `scenario-completeness-scoring` - Reference implementation

**Database:**
- New package - scenarios can migrate from manual backoff implementations

**Discovery:**
- `browser-automation-studio` - Discovers `test-genie`
- `git-control-tower` - Discovers `workspace-sandbox`
- `system-monitor` - Discovers `agent-manager`

**Retry:**
- Used internally by `database.Connect()`
- Available for HTTP clients, external APIs, etc.
