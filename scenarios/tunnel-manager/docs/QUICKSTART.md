# Quickstart — Tunnel Manager

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
vrooli scenario port tunnel-manager UI_PORT
```

You should see the Tunnel Manager dashboard. The first screen summarizes
tunnel health, exposure state, recovery state, and setup readiness.

## 4 — Talk to the API

Through the scenario CLI (preferred — uses the resolved port and token
automatically):

```bash
tunnel-manager status
tunnel-manager <domain> <command>   # e.g. list/create commands for your domain
```

### Tunnel Manager commands

> The commands below are the implemented operator CLI groups. See
> [`reference/cli-commands.md`](reference/cli-commands.md) for the full
> reference and [`../PRD.md`](../PRD.md) for scope.

```bash
tunnel-manager tunnel status           # tunnel health overview
tunnel-manager routes list             # list the exposure manifest (SSOT)
tunnel-manager exposure expose <scenario>  # request leased exposure (default TTL ~1 week)
tunnel-manager exposure leases         # list active leases
tunnel-manager exposure extend <id>    # extend a lease TTL
tunnel-manager exposure revoke <id>    # revoke a lease early
tunnel-manager probes run              # run internal + external liveness probes
tunnel-manager audit run               # port-compliance findings
tunnel-manager recovery state          # inspect recovery state
tunnel-manager recovery run            # manually trigger recovery
tunnel-manager config credentials-status
tunnel-manager config credentials-set --account-id <id> --tunnel-id <id> --api-token <token>
tunnel-manager config credentials-clear --field all
tunnel-manager config sync --dry-run true  # preview ingress changes safely
tunnel-manager config sync             # reconcile Cloudflare ingress with the manifest
tunnel-manager config mode --target remote  # switch ingress mode
```

Every command supports proto-typed `--json` output.

For remote mode, use Settings or the `config credentials-*` CLI commands
to save Cloudflare credentials through the Vrooli credential authority.
`CLOUDFLARE_*` environment variables are not accepted as a credential
source.

Or directly via HTTP:

```bash
API_PORT=$(vrooli scenario port tunnel-manager API_PORT)
curl -s "http://localhost:${API_PORT}/health"
# Proto-typed calls hit /vrooli.tunnel_manager.v1.<domain>.<Service>/<Method>
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
CLI (`tunnel-manager --help`) — see
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
