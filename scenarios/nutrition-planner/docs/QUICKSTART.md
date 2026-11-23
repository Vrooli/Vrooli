# Quickstart — Nutrition Planner

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
vrooli scenario port nutrition-planner UI_PORT
```

> **Known issue (2026-09-22):** the UI opens, but every page currently shows an
> error because the local runtime has no authentication profile and every
> workspace request is rejected as unauthenticated (blocker B1). See
> [`guides/troubleshooting.md`](guides/troubleshooting.md) → "Every page shows
> an error". Until that is fixed, only `/health` and the app shell render.

When the app works, the index route is **Today** in the Nooch redesign; the
five destinations are Today, Week, Meals, Groceries, and Kitchen, with Settings
in the header (see [`concepts/EXPERIENCE.md`](concepts/EXPERIENCE.md)).

## 4 — Talk to the API

Through the scenario CLI (preferred — uses the resolved port and token
automatically):

```bash
nutrition-planner status
nutrition-planner workspace list
nutrition-planner recipe list --workspace-id "<workspace-id>"
```

The CLI currently exposes only `status`, `workspace list`, `recipe list`, and
`recipe create`; the workspace and recipe commands fail with an
unauthenticated error until blocker B1 is fixed.

Or directly via HTTP:

```bash
API_PORT=$(vrooli scenario port nutrition-planner API_PORT)
curl -s "http://localhost:${API_PORT}/health"
# Proto-typed calls hit /vrooli.nutrition_planner.v1.<domain>.<Service>/<Method>
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
CLI (`nutrition-planner --help`) — see
[`reference/cli-commands.md`](reference/cli-commands.md).

## Troubleshooting

If anything misbehaves on first boot, check
[`guides/troubleshooting.md`](guides/troubleshooting.md). The most
common first-time issues are:

- Every request rejected as unauthenticated — the missing authentication
  profile (blocker B1); see the troubleshooting guide
- A previous scenario instance still holding ports — `make restart`
- Stale build artifacts after editing source — `make setup` rebuilds
- Missing `vrooli` CLI — run workspace-root `make setup`

## Next steps

- Read [the accepted specification](reference/product-specification.md),
  [GOAL](internal/goal/GOAL.md), [QUEUE](internal/goal/QUEUE.md) and the admitted
  [epoch](internal/goal/epochs/) before delivery work. [ACCEPTANCE](internal/goal/ACCEPTANCE.md)
  owns closure obligations. Historical redesign files preserve earlier evidence.
- Read [`START-HERE.md`](START-HERE.md) for the template initialization gates.
- Read [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md) for the
  mental model: three surfaces, proto bridge, layered API, where to
  add code.
- Read [`internal/TESTING.md`](internal/TESTING.md) before writing
  your first non-trivial test.
- Append a one-line entry to [`internal/PROGRESS.md`](internal/PROGRESS.md)
  whenever you land work, so future agents can replay the lifecycle.
