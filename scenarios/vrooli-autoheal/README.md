# Vrooli Autoheal

Self-healing supervisor that bootstraps, monitors, and auto-repairs Vrooli infrastructure across platforms.

## Overview

vrooli-autoheal replaces the legacy bash-based autoheal cronjob with a cross-platform solution featuring:

- **CLI commands** (`vrooli autoheal tick/loop/status`) for health management
- **Go API** for health status and configuration
- **React dashboard** for visualization and monitoring
- **OS-level watchdog** (systemd/launchd/Windows service) to keep autoheal running

## Quick Start

```bash
# Build and install
vrooli scenario run vrooli-autoheal --setup

# Start the scenario
make start   # or: vrooli scenario run vrooli-autoheal --dev

# Check health status
vrooli autoheal status

# Run a single health cycle
vrooli autoheal tick

# Run continuous health monitoring
vrooli autoheal loop --interval-seconds=60
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `vrooli autoheal tick` | Single health cycle (bootstrap + checks + watchdog verify) |
| `vrooli autoheal loop` | Continuous monitoring with configurable interval |
| `vrooli autoheal status` | Show last-known health summary |

## Architecture

```
vrooli-autoheal/
├── api/           # Go API server (health registry, status endpoints)
├── cli/           # CLI wrapper (vrooli-autoheal binary)
├── ui/            # React dashboard (Vite + TypeScript)
├── .vrooli/       # Lifecycle configuration
├── requirements/  # Technical requirements (11 modules)
└── test/          # Phased test suite
```

### Health Check Registry

All health checks implement a common interface and are registered at startup:

```go
type HealthCheck interface {
    ID() string
    Description() string
    IntervalSeconds() int
    Platforms() []Platform  // nil = all platforms
    Run(ctx HealthCheckContext) HealthResult
}
```

### Platform Detection

The system detects the current platform and capabilities:

- **Platforms**: Linux, Windows, macOS, other
- **Capabilities**: supportsRdp, supportsSystemd, supportsLaunchd, hasDocker, isHeadlessServer

## Health Checks

### P0 (Core)
- Resource health (postgres, redis, qdrant, ollama, etc.)
- Scenario health (configurable list of critical scenarios)
- OS watchdog verification

### P1 (Infrastructure)
- Network connectivity and DNS resolution
- Disk space and swap usage
- Docker daemon health
- Cloudflared tunnel connectivity
- RDP/xrdp/TermService health

## Configuration

Health checks and monitored resources/scenarios are configured via:

1. **Environment variables** (e.g., `VROOLI_AUTOHEAL_RESOURCES`, `VROOLI_AUTOHEAL_SCENARIOS`)
2. **PostgreSQL** (runtime configuration)
3. **Default values** for common checks

## Testing

```bash
# Full test suite
make test

# Phased tests
cd test && ./run-tests.sh

# Quick developer loop
cd test && ./run-tests.sh quick
```

## Documentation

- [PRD.md](./PRD.md) - Operational targets and product requirements
- [docs/PROGRESS.md](./docs/PROGRESS.md) - Development progress log
- [docs/PROBLEMS.md](./docs/PROBLEMS.md) - Known issues and deferred work
- [docs/RESEARCH.md](./docs/RESEARCH.md) - Background research and related scenarios
- [requirements/](./requirements/) - Technical requirements registry

## Related Scenarios

- **maintenance-orchestrator** - Controls activation of maintenance scenarios
- **system-monitor** - Metrics and anomaly detection (complements autoheal)
