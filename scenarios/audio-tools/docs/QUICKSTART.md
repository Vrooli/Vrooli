# Quickstart — Audio Tools

Start the existing Audio Tools development surfaces through the lifecycle.
Setup time depends on builds and selected resources; starting the UI is not a
voice-latency or hosted-delivery qualification. A usable local, BYOK or owned
route needs its own capabilities and configuration; owned delivery is currently
an implementation target, not a working fallback.

## Prerequisites

- **Vrooli CLI** installed and on `PATH`
- **Go** matching the versions declared in `api/go.mod` and `cli/go.mod`
- **Node 20+ and pnpm 9+** for the UI bundle

If `vrooli` is not on your `PATH`, run `make setup` from the workspace
root (`../..` from the scenario directory) once.

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

This starts the API/UI and manages dependencies according to their declared
startup policy; optional resources can remain unavailable. Components and ports follow [`.vrooli/service.json`](../.vrooli/service.json); see
[The Declared Scenario](../../../docs/architecture/the-declared-scenario.md) for the shared
lifecycle contract.

## 3 — Open

```bash
make open
```

Or check the URL directly:

```bash
vrooli scenario port audio-tools UI_PORT
```

You should see the Dictation Studio and provider status surfaces using the
local SQLite store.

## 4 — Talk to the API

Through the scenario CLI (preferred — uses the resolved port and token
automatically):

```bash
audio-tools status
audio-tools diagnostics run
audio-tools health show
```

Or directly via HTTP:

```bash
API_PORT=$(vrooli scenario port audio-tools API_PORT)
curl -s "http://localhost:${API_PORT}/health"
```

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
CLI (`audio-tools --help`) — see
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
- Read the current `portable-voice-v2` PRD and its requirements before changes.
  Regenerate authorized product-contract changes through `prd-authoring` and
  Business Health with a preview; preserve stable IDs and existing evidence.
- Append a one-line entry to [`internal/PROGRESS.md`](internal/PROGRESS.md)
  whenever you land work, so future agents can replay the lifecycle.
