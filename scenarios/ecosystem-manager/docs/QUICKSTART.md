# Quickstart — Ecosystem Manager

Get the Ecosystem Manager control plane running locally and create your
first task. The lifecycle handles ports, environment, and process
management — you should not need to wire anything by hand.

Ecosystem Manager is an internal Vrooli control plane: it runs
autonomous agent loops (auto-steer) to generate and improve scenarios
and resources. Before going deep, read
[`concepts/CONTROL-MODEL.md`](concepts/CONTROL-MODEL.md) and
[`START-HERE.md`](START-HERE.md).

## Prerequisites

- **Vrooli CLI** on `PATH` (run `vrooli help` to confirm)
- **Go** matching `api/go.mod` and `cli/go.mod`
- **Node 20+ and pnpm 9+** for the UI bundle
- **Postgres** running as `vrooli-postgres-main` (provides the
  `vrooli_ecosystem_manager` database)
- The scenarios **`agent-manager`** (executes tasks) and
  **`scenario-completeness-scoring`** (supplies the PRD/completeness
  metric) running

If `vrooli` is not on your `PATH`, run `make setup` from the workspace
root once.

## 1 — Setup

From this scenario's directory:

```bash
make setup
```

This builds the API, installs UI/CLI dependencies, and creates the
Postgres database `vrooli_ecosystem_manager`. Run it once after
checkout, and again whenever dependencies change.

## 2 — Start

```bash
make start
# or: vrooli scenario start ecosystem-manager
```

The lifecycle starts the API and UI and allocates the API port
automatically (UI fixed at `21110`; the dashboard is served at
`http://localhost:30500`). Never run `./api/ecosystem-manager-api`
directly — that bypasses process naming, port allocation, and health
checks.

Confirm it is up:

```bash
make status
curl -s http://localhost:30500/health
```

## 3 — Open

Open the dashboard (kanban board) in your browser:

```
http://localhost:30500
```

You should see the task board and live queue state.

## 4 — Create your first task

A task targets a scenario or resource, runs as a `generator` or
`improver` operation, and may attach an auto-steer profile.

Via the CLI (preferred):

```bash
# Generate a new scenario, steered by the "balanced" profile
ecosystem-manager task add scenario my-app --steer-profile balanced

# Improve an existing scenario toward a higher bar
ecosystem-manager task improve scenario my-app --steer-profile production-ready

# Inspect
ecosystem-manager task list --status pending --type scenario
ecosystem-manager task show <task-id>
```

`task add` creates a generator task; `task improve` creates an improver
task (`add` / `improve` are the canonical task-create verbs). You can
also create tasks directly on the kanban board in the UI.

Once a task is created, the queue picks it up and dispatches it through
`agent-manager`; if an auto-steer profile is attached, the controller
advances it through profile phases until a stop condition is met. Watch
progress with `ecosystem-manager queue` or in the dashboard.

## 5 — Run the tests

```bash
make test
# or: vrooli scenario test ecosystem-manager
```

This runs the full scenario test lifecycle (Go API/CLI tests plus UI
checks). Eslint warnings count as failures, so drive UI lint to zero.

## Common follow-up commands

| Command | What it does |
|---|---|
| `make logs` | Tail API + UI logs |
| `make status` | Show running surfaces and ports |
| `make restart` | `stop` then `start` (preferred over manual restarts) |
| `make stop` | Shut everything down cleanly |

If something misbehaves, see
[`guides/troubleshooting.md`](guides/troubleshooting.md).
