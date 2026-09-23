# Quickstart — Nutrition Planner

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

- Read [`internal/REDESIGN_PLAN.md`](internal/REDESIGN_PLAN.md) and
  [`internal/REDESIGN_GOAL.md`](internal/REDESIGN_GOAL.md) before changing
  product behavior; they drive the current v2.0 redesign.
- Read [`START-HERE.md`](START-HERE.md) for the template initialization gates.
- Read [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md) for the
  mental model: three surfaces, proto bridge, layered API, where to
  add code.
- Read [`internal/TESTING.md`](internal/TESTING.md) before writing
  your first non-trivial test.
- Append a one-line entry to [`internal/PROGRESS.md`](internal/PROGRESS.md)
  whenever you land work, so future agents can replay the lifecycle.
