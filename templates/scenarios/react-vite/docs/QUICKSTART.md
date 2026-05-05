# Quickstart — {{SCENARIO_DISPLAY_NAME}}

Get this scenario running locally in under five minutes. The lifecycle
handles ports, environment variables, and dependencies — you should
not need to set anything by hand.

## Prerequisites

- **Vrooli CLI** installed and on `PATH` (run `vrooli help` to confirm)
- **Go 1.22+** for the API and CLI binaries
- **Node 20+ and pnpm 9+** for the UI bundle

If `vrooli` is not on your `PATH`, run `make setup` from the workspace
root (one level above this directory) once.

## 1 — Setup

From this scenario's directory:

```bash
make setup
```

This runs `vrooli scenario setup` which:

- installs UI dependencies (`pnpm install --ignore-workspace`)
- runs `go mod tidy` for `api/` and `cli/`
- builds the API binary and the production UI bundle
- regenerates proto types if `proto/` has changed
- installs the scenario CLI to `~/.vrooli/bin/`

Run this once after generation, and again whenever dependencies change.

## 2 — Start

```bash
make start
```

This starts the API, UI, and any declared resources. The lifecycle
allocates ports automatically (`API_PORT` in `15000-19999`,
`UI_PORT` in `20000-24999`) and exposes them through the scenario's
CLI.

## 3 — Open

```bash
make open
```

Or check the URL directly:

```bash
vrooli scenario port {{SCENARIO_ID}} UI_PORT
```

You should see the example UI rendering live `/health` data and a
notes pane backed by the local SQLite store.

## 4 — Talk to the API

Through the scenario CLI (preferred — uses the resolved port and token
automatically):

```bash
{{SCENARIO_ID}} status
{{SCENARIO_ID}} notes list
{{SCENARIO_ID}} notes create --title "First note" --body "Hello"
```

Or directly via HTTP:

```bash
API_PORT=$(vrooli scenario port {{SCENARIO_ID}} API_PORT)
curl -s "http://localhost:${API_PORT}/health"
curl -s -X POST "http://localhost:${API_PORT}/vrooli.{{SCENARIO_ID_SNAKE}}.v1.notes.NotesService/ListNotes" \
  -H 'Content-Type: application/json' \
  -d '{}'
```

## 5 — Run the tests

```bash
make test
```

This runs the full local gate: UI lint + type-check + Vitest with
coverage; Go vet + race-detector tests + coverage; the API E2E binary
smoke; and the structure checks declared in `.vrooli/testing.json`.
Coverage floors are 85% for UI and 75% for Go (see
[`internal/TESTING.md`](internal/TESTING.md)).

## Common follow-up commands

| Command | What it does |
|---|---|
| `make logs` | Tail API + UI logs (or `vrooli scenario logs`) |
| `make status` | Show running surfaces and their ports |
| `make stop` | Shut everything down cleanly |
| `make restart` | `stop` then `start` (preferred over manually restarting individual surfaces) |

For scenario-specific commands beyond the lifecycle, use the scenario
CLI (`{{SCENARIO_ID}} --help`) — see
[`reference/cli-commands.md`](reference/cli-commands.md).

## Troubleshooting

If anything misbehaves on first boot, check
[`guides/troubleshooting.md`](guides/troubleshooting.md). The most
common first-time issues are:

- A previous scenario instance still holding ports — `make restart`
- Stale build artifacts after editing source — `make setup` rebuilds
- Missing `vrooli` CLI — run workspace-root `make setup`

## Next steps

- Read [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md) for the
  mental model: three surfaces, proto bridge, layered API, where to
  add code.
- Read [`internal/TESTING.md`](internal/TESTING.md) before writing
  your first non-trivial test.
- Update `PRD.md` with your operational targets, then add requirement
  modules under `requirements/`.
- Append a one-line entry to [`internal/PROGRESS.md`](internal/PROGRESS.md)
  whenever you land work, so future agents can replay the lifecycle.
