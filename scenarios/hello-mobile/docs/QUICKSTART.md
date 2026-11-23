# Quickstart — Hello Mobile

Get this scenario running locally in under five minutes. The lifecycle
handles ports, environment variables, and dependencies — you should
not need to set anything by hand.

If the scenario is already running, go directly to the operation you need.

## Prerequisites

- **Vrooli CLI** installed and on `PATH`
- **Go** matching the versions declared in `api/go.mod` and `cli/go.mod`
- **Node 20+ and pnpm 9+** for the UI bundle

If the Vrooli CLI is missing, run `make setup` from the Vrooli repository root.

## 1 — Setup

From this scenario's directory:

```bash
make setup
```

The setup plan is declared in [`.vrooli/service.json`](../.vrooli/service.json). See
[The Declared Scenario](../../../docs/architecture/the-declared-scenario.md) for
the shared lifecycle contract.

Use setup for a fresh scaffold or when the lifecycle or dependency gateway
reports that setup is needed.

## 2 — Start

```bash
make start
```

See [`.vrooli/service.json`](../.vrooli/service.json) for component and port
declarations, and [The Declared Scenario](../../../docs/architecture/the-declared-scenario.md) for
how the lifecycle uses them.

## 3 — Open

```bash
make open
```

Or check the URL directly:

```bash
vrooli scenario port hello-mobile UI_PORT
```

You should see the example UI rendering live `/health` data and a
worked example feature pane backed by the local SQLite store.

## 4 — Talk to the API

Through the scenario CLI (preferred — uses the resolved port and token
automatically):

```bash
hello-mobile status
hello-mobile <domain> <command>   # e.g. list/create commands for your domain
```

Or directly via HTTP:

```bash
API_PORT=$(vrooli scenario port hello-mobile API_PORT)
curl -s "http://localhost:${API_PORT}/health"
# Proto-typed calls hit /vrooli.hello_mobile.v1.<domain>.<Service>/<Method>
```

<!-- EXAMPLE-DOMAIN:notes START -->
The shipped worked-example `notes` domain illustrates the full shape —
copy it, then remove it with `template-manager detemplate`:

```bash
hello-mobile notes list
hello-mobile notes create --title "First note" --body "Hello"
```

```bash
API_PORT=$(vrooli scenario port hello-mobile API_PORT)
curl -s -X POST "http://localhost:${API_PORT}/vrooli.hello_mobile.v1.notes.NotesService/ListNotes" \
  -H 'Content-Type: application/json' \
  -d '{}'
```
<!-- EXAMPLE-DOMAIN:notes END -->

## 5 — Run the tests

Run the full suite for scaffold validation or when your change warrants it.
For a focused change, select the relevant phases with
`vrooli scenario test <name> --phases <relevant-phases>`.

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
CLI (`hello-mobile --help`) — see
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
