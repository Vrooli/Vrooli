# Quick Start

Get vrooli-autoheal running and monitoring your infrastructure in minutes.

## What is Autoheal?

Vrooli Autoheal is a self-healing infrastructure supervisor that:

- **Monitors** critical services (Docker, databases, tunnels, network)
- **Detects** problems before they become outages
- **Recovers** automatically by restarting failed services
- **Survives** reboots via OS-level watchdog integration

## Prerequisites

- Vrooli CLI installed (`vrooli --version`)
- PostgreSQL resource available

## Starting Autoheal

```bash
# Start the scenario
vrooli scenario start vrooli-autoheal

# Check status
vrooli scenario status vrooli-autoheal
```

Once started, the web dashboard is available at the UI port shown in the status output.

## Understanding the Dashboard

The dashboard shows:

1. **Status Badge** - Overall system health (OK/Warning/Critical)
2. **Summary Cards** - Count of healthy, warning, and critical checks
3. **Health Checks** - Individual check results grouped by severity
4. **Platform Info** - Detected OS and capabilities
5. **Events Timeline** - Recent health events and actions

## CLI Commands

```bash
# Run a single health check cycle
vrooli-autoheal tick

# Run with force (ignore intervals)
vrooli-autoheal tick --force

# Show current status
vrooli-autoheal status

# Show status as JSON
vrooli-autoheal status --json

# List registered checks
vrooli-autoheal checks

# Show platform capabilities
vrooli-autoheal platform
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Lifecycle health check |
| `/api/v1/status` | GET | Current health summary |
| `/api/v1/tick` | POST | Run health check cycle |
| `/api/v1/platform` | GET | Platform capabilities |
| `/api/v1/checks` | GET | List registered checks |

## Next Steps

- Read the [Architecture](concepts/architecture.md) to understand how it works
- Check the [Check Catalog](reference/check-catalog.md) to see all available checks
- Learn how to [Add Custom Checks](guides/adding-checks.md)
