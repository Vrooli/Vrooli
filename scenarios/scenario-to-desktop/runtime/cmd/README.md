# cmd - Command-Line Entry Points

The `cmd` directory contains the main entry points for the runtime executables.

## Overview

The runtime package is a library. This directory provides executable wrappers that can be built and distributed with desktop bundles.

## Executables

### runtime

The runtime daemon that supervises services.

```bash
# Start with manifest
runtime --manifest bundle.json

# Override app data directory
runtime --manifest bundle.json --app-data /custom/path

# Dry-run mode (no actual service launches)
runtime --manifest bundle.json --dry-run
```

**Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--manifest` | Path to bundle.json | Required |
| `--app-data` | App data directory | User config dir |
| `--dry-run` | Skip service launches | false |

**Behavior:**
1. Loads and validates manifest
2. Initializes supervisor with configuration
3. Starts control API server
4. Waits for shutdown signal (SIGINT/SIGTERM)
5. Gracefully stops all services

### runtimectl

CLI tool for interacting with a running runtime.

```bash
# Check runtime health
runtimectl health

# Check service readiness
runtimectl ready

# Get allocated ports
runtimectl ports

# Get specific port
runtimectl port --service api --port-name http

# View service logs
runtimectl logs --service api --lines 100

# Trigger shutdown
runtimectl shutdown
```

**Commands:**
| Command | Description |
|---------|-------------|
| `health` | Check runtime is running |
| `ready` | Check all services ready |
| `ports` | List allocated ports |
| `port` | Get specific service port |
| `logs` | Tail service logs |
| `shutdown` | Initiate graceful shutdown |

**Global Flags:**
| Flag | Description | Default |
|------|-------------|---------|
| `--url` | Control API URL | `http://127.0.0.1:47710` |
| `--token-file` | Auth token file | Auto-detect |

## Building

```bash
cd runtime

# Build both executables
go build -o runtime ./cmd/runtime
go build -o runtimectl ./cmd/runtimectl

# Cross-compile for distribution
GOOS=linux GOARCH=amd64 go build -o runtime-linux ./cmd/runtime
GOOS=darwin GOARCH=amd64 go build -o runtime-darwin ./cmd/runtime
GOOS=windows GOARCH=amd64 go build -o runtime.exe ./cmd/runtime
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Manifest load failed |
| 4 | Startup failed |

## Dependencies

- **Depends on**: `bundleruntime`, `manifest`
