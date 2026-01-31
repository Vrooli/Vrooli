# Assumptions and Invariants

This document captures key assumptions, invariants, and environment dependencies for the scenario-to-desktop API.

## Environment Variables

### Required Variables

| Variable | Purpose | Default Fallback | Used In |
|----------|---------|------------------|---------|
| None required | All variables have fallback behavior | - | - |

### Optional Variables

| Variable | Purpose | Default Fallback | Used In |
|----------|---------|------------------|---------|
| `VROOLI_ROOT` | Vrooli repository root directory | `$HOME/Vrooli` → git root detection → current directory | `shared/path/vrooli.go`, `generation/analyzer.go`, `tasks/service.go`, `signing/file_repository.go` |
| `API_PORT` | API server listening port | `15020` | `main.go` |
| `PORT` | Alternative to API_PORT | `15020` | `main.go` |
| `ALLOWED_ORIGIN` | CORS allowed origin | `http://localhost:${UI_PORT}` | `main.go` |
| `UI_PORT` | UI port for CORS configuration | `15021` | `main.go` |
| `AGENT_MANAGER_ENABLED` | Enable agent-manager integration | `true` | `main.go` |
| `DISPLAY` | X11 display (Linux GUI detection) | None (headless mode) | `smoketest/runner.go` |
| `XDG_CONFIG_HOME` | Linux config directory | `$HOME/.config` | `system/info.go` |

### Port Validation

The API port (`API_PORT` or `PORT`) is validated to be within the range **1024-65535**. Values outside this range will cause a startup error.

## VROOLI_ROOT Detection Strategy

The canonical detection function is `shared/path.DetectVrooliRoot()`. It uses the following priority order:

1. **Environment variable** (`VROOLI_ROOT`) - highest priority, used in production
2. **Default home directory** (`$HOME/Vrooli`) - if directory exists
3. **Git root detection** - walks up from current directory looking for `.vrooli` marker
4. **Relative path fallback** - assumes running from `api/` directory

### Usage Consistency

All code should use `shared/path.DetectVrooliRoot()` instead of inline detection logic. This ensures consistent behavior across:

- `generation/analyzer.go` - Scenario analysis
- `tasks/service.go` - Agent execution working directory
- `signing/file_repository.go` - Scenario locator

## Shared Utilities

### Environment Reader (`shared/env`)

The `shared/env.Reader` interface provides testable environment variable access:

```go
type Reader interface {
    GetEnv(key string) string
    LookupEnv(key string) (string, bool)
}
```

**Production implementation**: `env.NewOSReader()`

**Used by**:
- `signing/config.go`
- `signing/validation/prerequisites.go`
- `distribution/interfaces.go`

## Adapter Store Behavior

### Nil Store Handling

The following adapters gracefully handle nil stores with logging:

- `generationRecordStoreAdapter.Upsert()` - Logs warning, returns nil
- `scenarioRecordStoreAdapter.List()` - Logs warning, returns nil

This behavior is intentional for optional record storage but may indicate configuration errors if seen in production.

## Security Assumptions

### Path Handling

- All file paths use `filepath.Clean()` for normalization
- No centralized path traversal validation exists (potential improvement area)
- Certificate files are read with permissions validated by the OS

### Credentials

- Certificate passwords read from environment variables (never stored in config files)
- API keys for distribution targets read at runtime from environment
- No credential caching beyond the request lifecycle

## Build Assumptions

### Scenario Structure

Expected scenario directory structure:

```
scenarios/<name>/
├── .vrooli/
│   └── service.json      # Scenario metadata
├── api/                   # Optional API server
├── ui/
│   ├── package.json      # UI metadata
│   └── dist/             # Built UI assets (required for desktop generation)
│       └── index.html
└── signing.json          # Optional signing configuration
```

### UI Build Requirement

Desktop app generation requires a built UI (`ui/dist/index.html` must exist). The `generation/analyzer.go` checks for this and returns an appropriate error if missing.

## Pipeline Assumptions

### Stage Execution

- Stages execute sequentially in defined order
- Failed stages block subsequent stages
- Stage artifacts are stored in-memory during pipeline execution

### Artifact Paths

- Build artifacts stored relative to scenario output path
- Default output path: `<scenario>/desktop-app/`
- Staging path: `<output>/staging/`

## Testing Assumptions

### Interface Mocking

All external dependencies are abstracted behind interfaces for testing:

- `FileSystem` - File operations
- `CommandRunner` - External command execution
- `EnvironmentReader` - Environment variables
- `TimeProvider` - Time operations

This enables deterministic unit tests without filesystem or network access.
