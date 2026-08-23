# Quickstart — Scenario to Plugin

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
vrooli scenario port scenario-to-plugin UI_PORT
```

Today you will see the scaffold UI rendering live `/health` data and a
worked example feature pane backed by the local SQLite store. The product
surfaces — the readiness board, the gate ladder, and the evidence pages
described in [`../experience/index.json`](../experience/index.json) — are
authored as design intent and not yet built.

## 4 — Talk to the API

Through the scenario CLI (preferred — uses the resolved port and token
automatically):

```bash
scenario-to-plugin status
```

The product command groups — `readiness`, `package`, `check`, `attest`,
`rehearse`, `publish`, `revoke` — are specified in
[`reference/cli-commands.md`](reference/cli-commands.md) and are **not yet
implemented**. That document is marked `draft` for exactly that reason.

Or directly via HTTP:

```bash
API_PORT=$(vrooli scenario port scenario-to-plugin API_PORT)
curl -s "http://localhost:${API_PORT}/health"
# Proto-typed calls hit /vrooli.scenario_to_plugin.v1.<domain>.<Service>/<Method>
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
CLI (`scenario-to-plugin --help`) — see
[`reference/cli-commands.md`](reference/cli-commands.md).

## Troubleshooting

If anything misbehaves on first boot, check
[`guides/troubleshooting.md`](guides/troubleshooting.md). The most
common first-time issues are:

- A previous scenario instance still holding ports — `make restart`
- Stale build artifacts after editing source — `make setup` rebuilds
- Missing `vrooli` CLI — run workspace-root `make setup`

## Next steps

**This scenario is pre-implementation.** The product contract is complete
and validates clean; no domain code exists. Read in this order:

1. [`START-HERE.md`](START-HERE.md) — the initialization gates. Gates 1
   (charter), 2 (requirements), 3 (domain map), 4 (dependencies), 5
   (design language), 5a (experience), and 5b (business/ops stubs) are
   **closed**. Gates 0, 6, and 7 are open.
2. [`concepts/DOMAINS.md`](concepts/DOMAINS.md) — the six pipeline domains
   and, critically, the build order. The domain chain is acyclic and each
   stage reads only the one before it, so the map *is* the schedule.
3. [`../PRD.md`](../PRD.md) and [`../requirements/`](../requirements/) —
   what this scenario promises and how each promise will be proven.
4. [`internal/TESTING.md`](internal/TESTING.md) §"Validation strategy for
   this scenario" — read before the first test. This product's value is
   refusal, so the primary test for every gate is a deliberately broken
   fixture that must fail.
5. [`internal/DECISIONS.md`](internal/DECISIONS.md) — thirteen settled
   choices. Read before re-litigating one.

The first implementation slice is the `declaration` domain: it is the only
domain that reads nothing downstream, and every other domain needs it.
Follow `START-HERE.md` Gate 6 — start in proto, then API, transport, CLI,
and UI, finishing one domain before starting the next.

Append a one-line entry to [`internal/PROGRESS.md`](internal/PROGRESS.md)
whenever you land work, so future agents can replay the lifecycle.
