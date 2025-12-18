# api-core

Shared Go utilities for Vrooli scenario APIs. Currently provides automatic staleness detection and hot-rebuild capabilities.

## Features

### Staleness Checker

The `staleness` package enables Go API binaries to detect when their source files have changed and automatically rebuild/restart themselves.

**How it works:**
1. On startup, the API compares its binary's modification time against source files
2. If any `.go` files, `go.mod`, or `go.sum` are newer than the binary, it triggers a rebuild
3. After successful rebuild, the process re-execs itself with the new binary
4. Loop detection prevents infinite rebuild cycles

## Installation

Add to your scenario's `api/go.mod`:

```go
require github.com/vrooli/api-core v0.0.0

replace github.com/vrooli/api-core => ../../../packages/api-core
```

## Usage

Add the staleness check at the very start of your `main()` function, before any other initialization:

```go
package main

import (
    "github.com/vrooli/api-core/staleness"
)

func main() {
    // Staleness check - auto-rebuild if source files have changed
    // Must be first, before any initialization
    checker := staleness.NewChecker(staleness.CheckerConfig{})
    if checker.CheckAndMaybeRebuild() {
        return // Process was re-exec'd with new binary
    }

    // Rest of your main() ...
}
```

## Configuration

The checker accepts a `CheckerConfig` struct:

```go
type CheckerConfig struct {
    // APIDir is the directory containing the Go source files.
    // Default: directory containing the binary
    APIDir string

    // BinaryPath is the path to the compiled binary.
    // Default: os.Executable()
    BinaryPath string

    // Logger is a function for logging messages.
    // Default: log.Printf
    Logger func(format string, args ...interface{})

    // Disabled completely disables staleness checking.
    // Default: false
    Disabled bool

    // SkipRebuild detects staleness but doesn't rebuild (useful for testing).
    // Default: false
    SkipRebuild bool
}
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VROOLI_API_SKIP_STALE_CHECK` | Set to `true` to disable staleness checking |

## What Gets Checked

The checker monitors:

1. **Source files** (`*.go`) in the API directory and subdirectories
2. **go.mod** - dependency changes
3. **go.sum** - dependency version changes
4. **Local replace directives** - changes in local packages referenced via `replace` in go.mod

### Skipped Directories

The following directories are automatically skipped during file scanning:
- `.git`
- `vendor`
- `node_modules`
- `dist`
- `build`
- `__pycache__`
- `.cache`
- `testdata`

## Graceful Fallbacks

The checker handles edge cases gracefully:

- **No `go` in PATH**: Logs warning, continues with existing binary
- **Build fails**: Logs error, continues with existing binary
- **No go.mod**: Skips checking (not a Go module)
- **Loop detected**: Logs warning, continues with existing binary (prevents infinite rebuilds)

## Example Output

When a rebuild is triggered:
```
[staleness] stale binary detected: source file modified: /path/to/api/handlers/scores.go
[staleness] rebuilding with: go build -o /path/to/api/api .
[staleness] rebuild successful, re-executing...
```

## Testing

Run the test suite:

```bash
cd packages/api-core
go test ./...
```

## Architecture

```
packages/api-core/
├── go.mod
├── staleness/
│   ├── checker.go          # Main StalenessChecker
│   ├── checker_test.go
│   ├── gomod.go            # Parse replace directives
│   ├── gomod_test.go
│   ├── timestamps.go       # File timestamp utilities
│   └── timestamps_test.go
└── README.md
```
