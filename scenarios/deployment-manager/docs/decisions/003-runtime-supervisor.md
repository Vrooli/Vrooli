# ADR-003: Go Runtime Supervisor

## Status

Accepted

## Context

Desktop bundles need to manage multiple processes:
- UI server (serves static files)
- API binary (business logic)
- Resource binaries (SQLite server, etc.)

These processes have dependencies (API waits for database), need health monitoring, require port allocation, and must handle graceful shutdown.

Early approaches tried:
1. **Electron managing processes directly**: Worked but mixed concerns. Electron's main process became bloated with orchestration logic.
2. **Shell scripts**: Not portable to Windows. Error handling was fragile.
3. **Node.js subprocess manager**: Required Node.js runtime in bundle. Added ~100MB to bundle size.

## Decision

Build a **Go runtime supervisor** - a single static binary that:

1. Reads the bundle manifest (`bundle.json`)
2. Allocates ports from configured ranges
3. Starts services in dependency order
4. Monitors health and readiness
5. Handles secret injection (env vars and files)
6. Runs migrations on first start and upgrades
7. Records telemetry events
8. Exposes a control API for Electron to query status
9. Handles graceful shutdown

### Architecture

```
┌─────────────────────────────────────────────────┐
│                   Electron                       │
│  ┌─────────────┐                                │
│  │   main.ts   │──spawns──▶ runtime binary      │
│  └─────────────┘            │                   │
│        │                    │                   │
│        │ IPC                │ Control API       │
│        ▼                    ▼                   │
│  ┌─────────────┐    ┌──────────────────┐       │
│  │  renderer   │    │ Runtime Supervisor│       │
│  │  (UI)       │    │  - Port allocation│       │
│  └─────────────┘    │  - Service launch │       │
│                     │  - Health monitor │       │
│                     │  - Secret inject  │       │
│                     │  - Migration run  │       │
│                     │  - Telemetry      │       │
│                     └──────────────────┘       │
│                            │                    │
│              ┌─────────────┼─────────────┐     │
│              ▼             ▼             ▼     │
│         ┌───────┐    ┌───────┐    ┌───────┐   │
│         │  API  │    │  UI   │    │SQLite │   │
│         │ binary│    │server │    │server │   │
│         └───────┘    └───────┘    └───────┘   │
└─────────────────────────────────────────────────┘
```

### Control API Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /healthz` | Supervisor health |
| `GET /readyz` | All services ready |
| `GET /ports` | Allocated port map |
| `GET /logs/tail` | Recent log entries |
| `POST /shutdown` | Graceful shutdown |
| `POST /secrets` | Submit prompted secrets |
| `GET /validate` | Pre-flight validation |

## Consequences

### Positive

- **Zero runtime dependencies**: Static Go binary runs anywhere (no Node.js, Python, etc.)
- **Small footprint**: ~10MB binary vs. ~100MB for Node.js runtime
- **Cross-platform**: Single codebase compiles to Windows, macOS, Linux (x64, arm64)
- **Fast startup**: Go binaries start in milliseconds
- **Reliable process management**: Go's exec package is more robust than shell scripts
- **Testable**: Go's testing framework makes supervisor logic easy to unit test

### Negative

- **Language boundary**: Electron (TypeScript) ↔ Runtime (Go) requires IPC/HTTP
- **Build complexity**: Must cross-compile Go for each target platform
- **Go expertise required**: Contributors need to know both TypeScript and Go

### Neutral

- Runtime binary is bundled alongside Electron app
- Control API uses Bearer token for security (generated on start)
- Telemetry written to local JSONL file (uploaded by Electron if configured)

## Alternatives Considered

### Node.js Supervisor

Use a Node.js process (pm2, forever, or custom) to manage services.

**Rejected because**:
- Adds ~100MB Node.js runtime to every bundle
- Platform-specific Node.js binaries needed
- Memory overhead for Node.js event loop

### Electron Main Process as Supervisor

Handle all orchestration in Electron's main process.

**Rejected because**:
- Mixes UI concerns with orchestration concerns
- Electron main process should focus on windowing
- Harder to test without Electron

### Rust Supervisor

Use Rust instead of Go for better performance and smaller binaries.

**Rejected because**:
- Smaller contributor pool familiar with Rust
- Go's tooling (cross-compilation, testing) is more mature
- Performance difference negligible for this use case

### systemd/launchd/Windows Services

Use OS-native service managers.

**Rejected because**:
- Different APIs per platform
- Requires elevated permissions on some systems
- Harder to bundle and distribute

## References

- [Runtime Source Code](/scenarios/scenario-to-desktop/runtime/)
- [Bundle Packager](/scenarios/scenario-to-desktop/api/bundle_packager.go)
- [Bundled Runtime Plan](/docs/plans/bundled-desktop-runtime-plan.md)
