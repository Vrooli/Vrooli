# Quickstart — Plan Manager

Get this scenario running locally in under five minutes. The lifecycle
handles ports, environment variables, and dependencies — you should
not need to set anything by hand.

To inspect an existing plan, use `plan-manager plans get <plan>` or
`plan-manager plans render <plan>`. Use the
[CLI reference](reference/cli-commands.md) for other plan operations.

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
vrooli scenario port plan-manager UI_PORT
```

You should see the example UI rendering live `/health` data and a
worked example feature pane backed by the local SQLite store.

## 4 — Talk to the API

Through the scenario CLI (preferred — uses the resolved port and token
automatically):

```bash
plan-manager status
plan-manager "<domain>" "<command>"   # e.g. list/create commands for your domain
```

Or directly via HTTP:

```bash
API_PORT=$(vrooli scenario port plan-manager API_PORT)
curl -s "http://localhost:${API_PORT}/health"
# Proto-typed calls hit /vrooli.plan_manager.v1.<domain>.<Service>/<Method>
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
CLI (`plan-manager --help`) — see
[`reference/cli-commands.md`](reference/cli-commands.md).

## Troubleshooting

If anything misbehaves on first boot, check
[`guides/troubleshooting.md`](guides/troubleshooting.md). The most
common first-time issues are:

- A previous scenario instance still holding ports — `make restart`
- Stale build artifacts after editing source — `make setup` rebuilds
- Missing `vrooli` CLI — run workspace-root `make setup`

## Next steps

### Coordinate multiple plans

Create a family only when the initiative contains independently executable
child plans. Add members and typed claims, propose the graph, inspect every
edge reason, and record an explicit review before using the frontier:

```bash
plan-manager families create --slug validation-cutover --outcome "Move validation ownership to Test Genie" --max-parallel 2 --json
plan-manager families member-put <family-id> --revision <revision> --member-json '{"planId":"<plan-id>","role":"MEMBER_ROLE_PRIMARY","state":"MEMBER_STATE_PENDING"}' --json
plan-manager families graph-propose <family-id> --revision <revision> --json
plan-manager families frontier <family-id> --json
plan-manager families graph-review <family-id> --revision <revision> --graph-revision <graph-revision> --decision approved --reviewer <actor> --rationale "Claims and dependencies verified" --json
```

An unreviewed, cyclic, or stale graph is never launchable. Run
`plan-manager families render <family-id>` for the accessible table projection.
Use
`prompt-manager skill read plan-family-orchestration` for the decision rules
that distinguish a real family from phases of one plan, and for the
revision-safe review and frontier loop. Plan Manager owns the family state;
do not keep a parallel launch-order ledger in notes or prompts.


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
