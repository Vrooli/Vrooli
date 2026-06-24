# Getting Started — First-Run Walkthrough

A five-minute path from a clean checkout to a running Ecosystem Manager with a
task moving through the queue. For conceptual background read
[`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) first; for command
detail see [`../reference/cli-commands.md`](../reference/cli-commands.md).

## Prerequisites

- The Vrooli CLI is installed (`vrooli help` works).
- You are at the repo root or inside `scenarios/ecosystem-manager/`.
- `agent-manager` is available (Ecosystem Manager delegates agent execution to it).

## 1. Start the scenario

Always go through the lifecycle system — never run the binary directly.

```bash
cd scenarios/ecosystem-manager
make start          # wrapper for `vrooli scenario start ecosystem-manager`
make status         # confirm api + ui are healthy
```

`make start` builds the Go API, installs and builds the UI, allocates ports, and
registers the processes with the lifecycle manager. The API serves `/health`; the
UI is served on its configured `UI_PORT`.

## 2. Open the UI

`make status` prints the allocated `UI_PORT`. Open `http://localhost:<UI_PORT>`.
You should see the kanban board (Pending / In Progress / Review / Completed /
Failed) and the steering and execution-history panels.

## 3. Create your first task

From the CLI:

```bash
ecosystem-manager --help                 # discover the command surface
ecosystem-manager tasks list             # should be empty on first run
```

Or use the UI's "New Task" modal to create a resource/scenario × generate/improve
task. It appears in **Pending** immediately and the queue processor picks it up
when a slot is free.

## 4. Watch it process

```bash
make logs                                # tail the scenario logs
ecosystem-manager queue status           # processor state + slot usage
```

The board updates live over WebSocket as the task moves Pending → In Progress →
Completed/Failed.

## 5. Tune behavior via settings

Slots, cooldown, theme, and agent options live in settings (persisted to
`config/settings.json`). Adjust them in the UI's settings modal or via the CLI
`settings` commands. Slot/cooldown changes take effect on the running processor.

## Stopping

```bash
make stop            # vrooli scenario stop ecosystem-manager
```

## Next steps

- [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md) — the autosteer improvement loop.
- [`../reference/api-endpoints.md`](../reference/api-endpoints.md) — the API surface (and its REST→Connect migration; see [`../internal/COHERENCE-NOTES.md`](../internal/COHERENCE-NOTES.md)).
- [`troubleshooting.md`](troubleshooting.md) — common first-run problems.
