# Quick Start

Use this runbook to start ecosystem-manager and verify the core loop.

## Start

```bash
cd scenarios/ecosystem-manager
make start
```

## Verify

1. Open UI at `http://localhost:30500`.
2. Check API health: `curl -s http://localhost:30500/health`.
3. List queue state: `ecosystem-manager queue`.

## Core Actions

1. Create a task from the UI or CLI.
2. Confirm it appears in queue APIs.
3. Watch execution and logs in the dashboard.

## Stop

```bash
cd scenarios/ecosystem-manager
make stop
```

[CODE: api/main.go]
[CODE: api/pkg/server/server.go]
[CODE: cli/main.go]
