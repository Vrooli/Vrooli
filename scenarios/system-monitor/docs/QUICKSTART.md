# Quick Start

## Prerequisites

- Go 1.22+
- Node.js 20+
- Vrooli lifecycle system installed (`vrooli` CLI available)

## Start

```bash
cd scenarios/system-monitor && make start
```

This starts both the API server and UI dashboard via the Vrooli lifecycle system.

## Verify

```bash
system-monitor health
```

Expected output: `System monitor is healthy` with uptime and status details.

## Open the Dashboard

```bash
system-monitor dashboard
```

Or navigate to `http://localhost:${UI_PORT}` in your browser (default port assigned by lifecycle).

## Common CLI Commands

```bash
system-monitor metrics          # Current CPU/memory/TCP metrics
system-monitor status           # HEALTHY / WARNING / CRITICAL / OFFLINE
system-monitor alerts           # Active threshold alerts
system-monitor investigate      # Latest investigation results
system-monitor report daily     # Generate a daily report
system-monitor watch            # Live monitoring with ASCII bars (2s refresh)
```

## Stop

```bash
cd scenarios/system-monitor && make stop
```

## Next Steps

- [Architecture Overview](concepts/ARCHITECTURE.md)
- [API Endpoints Reference](reference/api-endpoints.md)
- [CLI Commands Reference](reference/cli-commands.md)
- [Configuration Reference](reference/configuration.md)
- [Troubleshooting](guides/troubleshooting.md)
