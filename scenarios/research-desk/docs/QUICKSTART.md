# Quickstart — Research Desk

Get this scenario running locally in under five minutes. The lifecycle
handles ports, environment variables, and dependencies — you should
not need to set anything by hand.

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
vrooli scenario port research-desk UI_PORT
```

You should see Now: what research is moving, what needs you and the desk's routes, read
from `~/.vrooli/research` (the desk keeps no database).

## 4 — Talk to the API

Through the scenario CLI (preferred — uses the resolved port and token
automatically):

```bash
research-desk status
research-desk now
```

`now` shows which research is moving. Run `research-desk` without arguments for
every command: what is moving, build/launch/control, and steer/answer/review.

Or directly via HTTP:

```bash
API_PORT=$(vrooli scenario port research-desk API_PORT)
curl -s "http://localhost:${API_PORT}/health"
# Proto-typed calls hit /vrooli.research_desk.v1.<domain>.<Service>/<Method>
```

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
CLI (`research-desk --help`) — see
[`reference/cli-commands.md`](reference/cli-commands.md).

## Troubleshooting

If anything misbehaves on first boot, check
[`guides/troubleshooting.md`](guides/troubleshooting.md). The most
common first-time issues are:

- A previous scenario instance still holding ports — `make restart`
- Stale build artifacts after editing source — `make setup` rebuilds
- Missing `vrooli` CLI — run workspace-root `make setup`

## Next steps

- Read [`../README.md`](../README.md) and then
  [`concepts/SYSTEM.md`](concepts/SYSTEM.md), the one design document.
- Read [`internal/TESTING.md`](internal/TESTING.md) before writing
  your first non-trivial test.
- Append a one-line entry to [`internal/PROGRESS.md`](internal/PROGRESS.md)
  whenever you land work, so future agents can replay the lifecycle.
