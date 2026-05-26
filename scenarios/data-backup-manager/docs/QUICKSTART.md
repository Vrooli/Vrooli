# Quickstart — Data Backup Manager

Get this scenario running locally in under five minutes. The lifecycle
handles ports, environment variables, and dependencies — you should
not need to set anything by hand.

Data Backup Manager gives Vrooli engine-backed backup and verified
restore: owning scenarios self-register the runtime state they own
(a **Target**), operators point it at one or more encrypted
**Destinations**, a **Plan** binds targets to destinations on a
schedule, each **Run** snapshots them, and a **Restore** (with a
verify mode) proves they come back. The commands in steps 4 and below
that name destinations, plans, runs, and restores describe the
**intended CLI surface** and are pending implementation; the lifecycle
commands (`make setup`/`start`/`status`/`test`) work today.

## Prerequisites

- **Vrooli CLI** installed and on `PATH` (run `vrooli help` to confirm)
- **Go** matching the versions declared in `api/go.mod` and `cli/go.mod`
- **Node 20+ and pnpm 9+** for the UI bundle

If `vrooli` is not on your `PATH`, run `make setup` from the workspace
root (one level above this directory) once.

## 1 — Setup

From this scenario's directory:

```bash
make setup
```

This runs the scenario's setup lifecycle: dependencies are prepared,
the API/CLI/UI are built as needed, and the scenario CLI is installed.
Keep the exact lifecycle steps in `.vrooli/service.json`; this guide is
only the user-facing path.

Run this once after generation, and again whenever dependencies change.

## 2 — Start

```bash
make start
```

This starts the API, UI, and any declared resources. The lifecycle
allocates ports automatically and exposes them through scenario
commands such as `make status` and `vrooli scenario port`.

## 3 — Open

```bash
make open
```

Or check the URL directly:

```bash
vrooli scenario port data-backup-manager UI_PORT
```

You should see the UI rendering live `/health` data, with destination
usage, plans, and run history as the operational centerpiece once those
features are built.

## 4 — Talk to the API

The scenario CLI is preferred (it resolves the port and token
automatically). The `status` command works today; the rest below is the
**planned operator and self-registration surface** (see
[`reference/cli-commands.md`](reference/cli-commands.md)):

```bash
# Works today
data-backup-manager status

# Planned operator surface — register a destination, plan, run, restore
data-backup-manager destinations create --name nightly-local \
  --backend filesystem --path /var/backups/nightly
data-backup-manager plans create --name nightly \
  --target prompt-manager/store-teams --destination nightly-local \
  --schedule "0 3 * * *" --keep-daily 7
data-backup-manager runs start --plan nightly
data-backup-manager runs list
data-backup-manager restores verify --target prompt-manager/store-teams \
  --destination nightly-local
```

A scenario self-registers the state it owns (the planned
self-registration call, run at its own lifecycle):

```bash
data-backup-manager targets register --owner prompt-manager \
  --name store-teams --kind filesystem --locator store/teams
```

The `/health` endpoint is reachable directly for probes:

```bash
API_PORT=$(vrooli scenario port data-backup-manager API_PORT)
curl -s "http://localhost:${API_PORT}/health"
```

Proto-typed operations use generated Connect-RPC clients rather than
hand-built JSON requests; see
[`reference/api-endpoints.md`](reference/api-endpoints.md).

## 5 — Run the tests

```bash
make test
```

This runs the scenario test lifecycle. The current phase list and
coverage expectations live in `.vrooli/testing.json`,
`.github/workflows/test.yml`, and [`internal/TESTING.md`](internal/TESTING.md).

## Common follow-up commands

| Command | What it does |
|---|---|
| `make logs` | Tail API + UI logs (or `vrooli scenario logs`) |
| `make status` | Show running surfaces and their ports |
| `make stop` | Shut everything down cleanly |
| `make restart` | `stop` then `start` (preferred over manually restarting individual surfaces) |

For scenario-specific commands beyond the lifecycle, use the scenario
CLI (`data-backup-manager --help`) — see
[`reference/cli-commands.md`](reference/cli-commands.md).

## Troubleshooting

If anything misbehaves on first boot, check
[`guides/troubleshooting.md`](guides/troubleshooting.md). The most
common first-time issues are:

- A previous scenario instance still holding ports — `make restart`
- Stale build artifacts after editing source — `make setup` rebuilds
- Missing `vrooli` CLI — run workspace-root `make setup`

## Next steps

- Read [`START-HERE.md`](START-HERE.md) before implementing product
  behavior. It owns the first-session workflow after generation.
- Read [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md) for the
  mental model: three surfaces, proto bridge, layered API, where to
  add code.
- Read [`internal/TESTING.md`](internal/TESTING.md) before writing
  your first non-trivial test.
- Update `PRD.md` with your operational targets, then add requirement
  modules under `requirements/`.
- Append a one-line entry to [`internal/PROGRESS.md`](internal/PROGRESS.md)
  whenever you land work, so future agents can replay the lifecycle.
