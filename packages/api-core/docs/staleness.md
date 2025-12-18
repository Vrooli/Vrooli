# Staleness Detection

The `staleness` package provides timestamp-based detection for stale Go API binaries. When source files are modified after the binary was compiled, the staleness checker can automatically rebuild and restart the API.

> **Note**: For most use cases, prefer using the higher-level `preflight` package which combines staleness detection with lifecycle management.

## How It Works

The staleness checker compares the binary's modification time against:

1. **Source files** (`*.go`) in the API directory and subdirectories
2. **go.mod** - dependency manifest changes
3. **go.sum** - dependency version lockfile changes
4. **Local replace directives** - changes in local packages referenced via `replace` in go.mod

### Timestamp-Based Detection

```
Binary mtime: 2024-01-15 10:00:00
                    │
                    ▼
┌─────────────────────────────────────┐
│  File Walk: api/ and dependencies   │
├─────────────────────────────────────┤
│                                     │
│  handlers/scores.go  10:05:00  ◄── NEWER! Stale detected
│  handlers/health.go  09:30:00      (older, OK)
│  main.go             09:45:00      (older, OK)
│  go.mod              09:00:00      (older, OK)
│                                     │
└─────────────────────────────────────┘
```

### Why Timestamps vs Fingerprints?

| Aspect | Timestamps | Fingerprints (SHA256) |
|--------|------------|----------------------|
| Speed | ~10ms | ~100ms-1s |
| Accuracy | Good (rare false positives) | Perfect |
| False positives | `touch`, git checkout | None |
| Scale | Excellent for many scenarios | Slower at scale |

For Vrooli's scale (50+ agents, thousands of scenarios), timestamps provide the right balance of speed and accuracy.

## Direct Usage

```go
package main

import "github.com/vrooli/api-core/staleness"

func main() {
    checker := staleness.NewChecker(staleness.CheckerConfig{})
    if checker.CheckAndMaybeRebuild() {
        return // Process was re-exec'd
    }

    // Continue with initialization...
}
```

## Rebuild Flow

```
CheckAndMaybeRebuild()
         │
         ▼
┌─────────────────┐
│ Resolve paths   │
│ - Binary path   │
│ - API directory │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Get binary      │
│ modification    │
│ time            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Check staleness │
│ - *.go files    │
│ - go.mod        │
│ - go.sum        │
│ - replace deps  │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
  Stale     Fresh
    │         │
    ▼         ▼
┌─────────┐  Return
│ Detect  │  false
│ loop?   │
└────┬────┘
     │
┌────┴────┐
│         │
Loop    OK
│         │
▼         ▼
Log &   ┌─────────────┐
Return  │ go build    │
false   │ -o binary . │
        └──────┬──────┘
               │
          ┌────┴────┐
          │         │
        Fail     Success
          │         │
          ▼         ▼
        Log &   ┌─────────────┐
        Return  │ syscall.Exec│
        false   │ (re-exec)   │
                └──────┬──────┘
                       │
                       ▼
                 New process
                 (same PID)
```

## Replace Directive Detection

The staleness checker parses `go.mod` to find local dependencies:

```go
// go.mod
module my-scenario/api

replace github.com/vrooli/api-core => ../../../packages/api-core
replace github.com/vrooli/proto => ../../../packages/proto
```

For each local dependency, the following are checked:
- `*.go` source files
- `go.mod` (dependency changes)
- `go.sum` (dependency version updates)

Both `packages/api-core` and `packages/proto` are checked for modifications.

### Parsing Logic

```
go.mod content:
┌─────────────────────────────────────────────────────────┐
│ module my-scenario/api                                  │
│                                                         │
│ replace github.com/vrooli/api-core => ../../../pkg/ac   │  ◄── Extracted
│ replace github.com/vrooli/proto => ../../../pkg/proto   │  ◄── Extracted
│ replace example.com/foo v1.0.0 => v1.1.0                │  ◄── Ignored (version, not path)
│                                                         │
│ replace (                                               │
│     github.com/bar => ../bar                            │  ◄── Extracted (block syntax)
│ )                                                       │
└─────────────────────────────────────────────────────────┘
```

## Loop Prevention

To prevent infinite rebuild loops (e.g., if the build always produces a different binary), the checker uses an environment variable:

```
First rebuild:
┌──────────────────────────────────────┐
│ Set API_CORE_REBUILD_TIMESTAMP=now   │
│ syscall.Exec(binary, args, env)      │
└──────────────────────────────────────┘
                │
                ▼
Second run (if still stale):
┌──────────────────────────────────────┐
│ Check API_CORE_REBUILD_TIMESTAMP     │
│ Within 60 seconds? → Skip rebuild    │
└──────────────────────────────────────┘
```

## Skipped Directories

The following directories are automatically skipped during file walks:

- `.git`
- `.vscode`
- `.idea`
- `vendor`
- `node_modules`
- `dist`
- `build`
- `tmp`
- `data`
- `testdata`
- `coverage`

## Configuration

```go
type CheckerConfig struct {
    // APIDir is the directory containing the API source code.
    // Default: directory containing the binary (os.Executable())
    APIDir string

    // BinaryPath is the path to the API binary being checked.
    // Default: os.Executable()
    BinaryPath string

    // Logger for output messages.
    // Default: fmt.Fprintf(os.Stderr, ...)
    Logger func(format string, args ...interface{})

    // Disabled completely skips all staleness checking.
    Disabled bool

    // SkipRebuild only logs staleness without attempting rebuild.
    SkipRebuild bool

    // CommandRunner overrides exec.Cmd.Run() for testing.
    CommandRunner func(cmd *exec.Cmd) error

    // Reexec overrides syscall.Exec for testing.
    Reexec func(binary string, args []string, env []string) error

    // LookPath overrides exec.LookPath for testing.
    LookPath func(file string) (string, error)
}
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VROOLI_API_SKIP_STALE_CHECK` | Set to `"true"` to disable staleness checking |
| `API_CORE_REBUILD_TIMESTAMP` | Internal: Unix timestamp of last rebuild (loop detection) |

## Example Output

### Stale Binary Detected

```
api-core: binary is stale (source file modified: pkg/handlers/scores.go)
api-core: rebuilding binary...
api-core: rebuild successful, restarting...
```

### Stale Dependency (Source)

```
api-core: binary is stale (dependency modified: ../../../packages/api-core (checker.go))
api-core: rebuilding binary...
api-core: rebuild successful, restarting...
```

### Stale Dependency (go.sum)

```
api-core: binary is stale (dependency go.sum modified: ../../../packages/api-core)
api-core: rebuilding binary...
api-core: rebuild successful, restarting...
```

### Loop Detection

```
api-core: rebuild loop detected, skipping auto-rebuild
  Staleness reason: source file modified: main.go
```

### No Go in PATH

```
api-core: 'go' not found in PATH, cannot auto-rebuild
```
