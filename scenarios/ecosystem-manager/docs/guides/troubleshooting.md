# Troubleshooting — Ecosystem Manager

Common failures when running the Ecosystem Manager control plane and
how to resolve them. Scenario-internal sharp edges and open questions
live in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md); this guide
covers operational symptoms you hit while running the scenario.

## Lifecycle and ports

### "Port already in use" / dashboard won't load

A previous instance may still hold the port. The UI is fixed at
`21110` and the dashboard is served at `http://localhost:30500`; the
API port is lifecycle-allocated.

```bash
vrooli scenario status ecosystem-manager
make restart
# if that doesn't recover:
make stop
make start
```

Never start the scenario by running the binary directly
(`./api/ecosystem-manager-api`); that bypasses the lifecycle's process
naming, port allocation, and health checks, and causes detection
issues.

### `vrooli: command not found`

The Vrooli CLI isn't on your `PATH`. Run `make setup` from the
workspace root, then open a new shell.

## API and CLI

### `/health` reports unhealthy

The health endpoint reports both API and database status. The database
is an embedded SQLite file, so an unhealthy `/health` usually means the
storage data root is not writable (see
[Storage](#storage-sqlite--filesystem)). Check logs:

```bash
curl -s http://localhost:30500/health
make logs
```

### CLI: "API not available"

The API isn't running or the CLI is pointed at the wrong base. Start
the scenario (`make start`) and confirm the dashboard responds at
`http://localhost:30500`. Re-run after `make setup` if the CLI binary
is stale.

## Build and dependencies

### `make setup` fails building the API

`make setup` builds the Go API and installs UI/CLI dependencies. A build
failure is usually a Go toolchain mismatch — confirm your Go version
matches `api/go.mod` and `cli/go.mod`. There is no separate database
step: the embedded SQLite file is created and its schemas applied at API
boot via `database.EnsureSchemas`.

### UI build / lint failures in tests

`vrooli scenario test ecosystem-manager` treats eslint warnings as
failures. Drive UI lint to zero before expecting a green run.

## Tests

### Scenario tests fail

Run the full lifecycle and read the failing phase:

```bash
vrooli scenario test ecosystem-manager
```

Distinguish "mine vs pre-existing": use a
`git-control-tower baseline snapshot` / `diff` to isolate whether a
failure is from your change. Never `git stash` to do this — concurrent
agents share the tree.

## Storage (SQLite + filesystem)

Ecosystem Manager keeps active state in an embedded SQLite file and
definitions on the filesystem:

- **SQLite** — `<data-root>/vrooli/<namespace>/ecosystem-manager.db`
  (resolved through `api/pkg/storagepaths`) holds task/auto-steer
  execution state and history.
- **Filesystem** — git-tracked `profiles/` (auto-steer profile
  definitions) in the scenario tree, plus the runtime task queue YAML
  under `<data-root>/vrooli/<namespace>/queue/<status>/`.

### Storage data root not writable

If the storage data root cannot be created or written, the SQLite file
cannot open, `/health` reports unhealthy, and state does not persist
across restarts. Confirm the data root exists and is writable, then
restart the scenario.

### Auto-steer state missing for a task

Profile *definitions* live on disk; active *state* lives in SQLite.
If state is missing:

1. Verify the task actually has a profile attached.
2. Check `/api/auto-steer/execution/{taskId}`.
3. Trigger a manual seek initialization from the UI (or
   `/api/auto-steer/execution/seek`).

## Auto-steer and the queue

### Queue stalls — tasks never execute

The queue dispatches work through `agent-manager`. If `agent-manager`
is not running, tasks sit unexecuted. Confirm it is up:

```bash
vrooli scenario status agent-manager
```

### Stuck agent process

A task can wedge on a hung agent process. Terminate it:

```bash
curl -s -X POST http://localhost:30500/api/queue/processes/terminate
```

### Rate-limit backoff

If the queue has backed off on rate limits and isn't picking up work,
clear the backoff:

```bash
curl -s -X POST http://localhost:30500/api/queue/reset-rate-limit
```

### Queue paused / not processing

Check the **Settings** active toggle in the UI. To resume processing:

```bash
curl -s -X POST http://localhost:30500/api/queue/start
```

### PRD metric confusion

Ecosystem Manager's PRD completion metric is read locally from the target
scenario's `PRD.md`; it does not require `scenario-completeness-scoring`
to be running. Use SCS when you need the separate cached maturity/
freshness/completeness view:

```bash
scenario-completeness-scoring score get <scenario>
```

### Auto-steer not advancing

If a steered task never advances phases, the usual cause is a **stop
condition referencing a metric that was never collected** — evaluating
an uncollected metric errors. Compare the profile's `stop_conditions`
against the metrics actually collected for the task (in
`/api/auto-steer/execution/{taskId}`), and either collect the metric or
fix the condition.

## When to add a new entry here

Add an entry when a failure is **operational and reproducible while
running the scenario** (a port/lifecycle issue, a missing dependency, a
queue/auto-steer stall, a storage symptom). Keep each entry to the
symptom, the cause, and the fix command.

Do **not** add entries here for: internal design sharp edges or open
questions (those go in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md)),
or one-off failures you can't reproduce. If a failure is a genuine
defect outside your scope, file it via `report-bug`.

## Cross-references

- [`../START-HERE.md`](../START-HERE.md) — orientation
- [`../QUICKSTART.md`](../QUICKSTART.md) — getting it running
- [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md) — the auto-steer controller
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system mental model
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — internal known issues
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — operational procedures
