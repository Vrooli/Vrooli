# Quickstart — Personal Planner

Get this scenario running locally in under five minutes. The lifecycle
handles ports, environment variables, and dependencies — you should
not need to set anything by hand.

Personal Planner is an adaptive personal planning + focus app with a
calm, editorial **Observatory** UI. It works entirely as a manual tool:
**no AI provider or model is required** to plan, focus, and review. Its
one required scenario dependency is **scenario-authenticator** (it
supplies the authenticated subject each private workspace binds to);
**notification-hub** is an optional dependency used only when the
`notifications` domain delivers a notice. The lifecycle starts the
required dependency for you.

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

This starts the API, UI, and any declared dependencies (including
**scenario-authenticator**, which the lifecycle brings up so you get a
real, isolated workspace). The lifecycle allocates ports automatically
and exposes them through scenario commands such as `make status` and
`vrooli scenario port`. The API listens on a port in the `15000-19999`
range and the UI in `20000-24999`; readiness is reported at `/health`.

## 3 — Open

```bash
make open
```

Or check the URL directly:

```bash
vrooli scenario port personal-planner UI_PORT
```

You should see the example UI rendering live `/health` data and a
worked example feature pane backed by the local SQLite store.

## 4 — Talk to the API

Through the scenario CLI (preferred — uses the resolved port and token
automatically):

```bash
personal-planner status
personal-planner <domain> <command>   # e.g. list/create commands for your domain
```

Or directly via HTTP:

```bash
API_PORT=$(vrooli scenario port personal-planner API_PORT)
curl -s "http://localhost:${API_PORT}/health"
# Proto-typed calls hit /vrooli.personal_planner.v1.<domain>.<Service>/<Method>
```

<!-- EXAMPLE-DOMAIN:notes START -->
The shipped worked-example `notes` domain illustrates the full shape —
copy it, then remove it with `template-manager detemplate`:

```bash
personal-planner notes list
personal-planner notes create --title "First note" --body "Hello"
```

```bash
API_PORT=$(vrooli scenario port personal-planner API_PORT)
curl -s -X POST "http://localhost:${API_PORT}/vrooli.personal_planner.v1.notes.NotesService/ListNotes" \
  -H 'Content-Type: application/json' \
  -d '{}'
```
<!-- EXAMPLE-DOMAIN:notes END -->

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
CLI (`personal-planner --help`) — see
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
