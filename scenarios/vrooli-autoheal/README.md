# Vrooli Autoheal

Self-healing supervisor that bootstraps, monitors, and auto-repairs Vrooli infrastructure across platforms.

## Overview

vrooli-autoheal replaces the legacy bash-based autoheal cronjob with a cross-platform solution featuring:

- **CLI commands** (`vrooli-autoheal tick/loop/status`) for health management
- **Go API** for health status and configuration
- **React dashboard** for visualization and monitoring
- **OS-level watchdog** (systemd/launchd/Windows service) to keep autoheal running

## Quick Start

```bash
# Build and install
vrooli scenario run vrooli-autoheal --setup

# Start the scenario through the managed lifecycle
make start

# Check health status
vrooli-autoheal status

# Run a single health cycle
vrooli-autoheal tick

# Run continuous health monitoring
vrooli-autoheal loop --interval-seconds=60
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `vrooli-autoheal tick` | Single health cycle (bootstrap + checks + watchdog verify) |
| `vrooli-autoheal loop` | Continuous monitoring with configurable interval |
| `vrooli-autoheal status` | Show last-known health summary |

## Architecture

```
literal:vrooli-autoheal/
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
- Resource health for every resource in `vrooli supervision-set`
- Scenario health for every scenario in `vrooli supervision-set`
- OS watchdog verification

### P1 (Infrastructure)
- Network connectivity and DNS resolution
- Disk space and swap usage
- Docker daemon health
- Cloudflared tunnel connectivity
- RDP/xrdp/TermService health

## Configuration

Health checks and supervised resources/scenarios are governed via:

1. **Canonical authority**: `vrooli supervision-set --json` computes the operator-declared dependency closure and attribution chains.
2. **Additive overrides**: `~/.vrooli-autoheal/config.json` may add monitoring targets, but cannot remove or disable canonical members.
3. **Last-known-good behavior**: a source outage retains the most recent non-empty set and raises `vrooli-supervision-set-source` as degraded; startup fails closed if no good set has ever loaded.
4. **SQLite** stores bounded health history and action logs, never supervision membership.

Canonical members take precedence when an additive override names the same
check. `must_start` members are critical; `try_start` members are warning-level.
Use `vrooli supervision-set --json` to inspect the effective member and its
attribution chain before changing autoheal configuration.

All scheduled and manual recovery actions cross the same heal interlock. A
successful start-like action blocks a dangerous action from another check
against the same target for the 30-second safety window. This guard is separate
from orphan classification because independently scheduled checks can race
even when both classify the process correctly.

Supervised-member availability is persisted as continuous outage intervals.
Query both total unavailable seconds and distinct outage count with:

```bash
vrooli-autoheal measure outages --member-id resource-qdrant --window-hours 24
```

After three consecutive real recovery failures by default, autoheal suspends
that check, raises a durable incident, and continues observing without retrying
forever. Inspect with `vrooli-autoheal check suspended`; after repairing the
cause, resume only that check with `vrooli-autoheal check resume <check-id>`.

The project-level model and precedence table are documented in
[`docs/concepts/ARCHITECTURE.md`](../../docs/concepts/ARCHITECTURE.md#supervision-authority).

## Testing

```bash
# Full test suite
make test

# Phased tests
vrooli scenario test vrooli-autoheal

# Quick developer loop
vrooli scenario test vrooli-autoheal
```

## Documentation

- [PRD.md](./PRD.md) - Operational targets and product requirements
- [docs/internal/PROGRESS.md](./docs/internal/PROGRESS.md) - Development progress log
- [docs/internal/PROBLEMS.md](./docs/internal/PROBLEMS.md) - Known issues and deferred work
- [docs/internal/RESEARCH.md](./docs/internal/RESEARCH.md) - Background research and related scenarios
- [requirements/README.md](./requirements/README.md) - Technical requirements registry

## Related Scenarios

- **system-monitor** - Metrics and anomaly detection (complements autoheal)
