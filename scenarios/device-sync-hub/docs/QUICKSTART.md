# Quickstart — Device Sync Hub

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
vrooli scenario port device-sync-hub UI_PORT
```

## 4 — Set up the hub's first device (you are the owner)

A hub has exactly one **owner** (a [scenario-authenticator](../../scenario-authenticator)
account) and a trust group of **devices**. On a fresh install the very first
device is set up by the owner — there is no pairing code yet because nothing
has been paired.

### From the browser

1. Open the UI. The first-run screen offers two paths: **Set up this hub**
   (you're the owner) or **Join an existing hub**.
2. Choose **Set up this hub**, then **Sign in** with your owner account — or
   **Create account** if you're new (registration signs you in immediately).
   Then click **Make this my first device**. That claims the hub for you
   (first-owner-wins) and trusts this browser directly — no pairing code needed.
3. You're now in the app. To add more devices, open **Devices → Add a device**
   to issue a pairing code (or let another device request approval).

> Sign-in and registration post to the hub's own API (same origin); the hub
> forwards to scenario-authenticator, whose URL it resolves automatically by
> name via `api-core/discovery`. There is nothing to configure — no
> `AUTH_SERVICE_URL`, no port — and the browser never calls scenario-authenticator
> directly.

### From the CLI

```bash
# 1. Create the owner account (or `auth login` if you already have one).
#    Either stores the returned owner token in the CLI config.
device-sync-hub auth register --email <you@example.com> --password <password>
device-sync-hub auth whoami            # confirm the signed-in owner

# 2. Claim the hub + trust THIS machine as the first device.
device-sync-hub devices setup --name "Workstation"
#    prints a one-time device token — export it for transfer commands:
export DEVICE_SYNC_HUB_DEVICE_TOKEN=<token-from-output>

# 3. Add more devices: issue a code the new device redeems.
device-sync-hub devices pair --name "Phone"
device-sync-hub devices list
```

`auth register`/`auth login` call the hub's own `IdentityService`, which forwards
to scenario-authenticator (resolved by name via `api-core/discovery` — no
`AUTH_SERVICE_URL` or port to set). Every owner-authed `devices` command then
rides the stored owner token automatically; the hub verifies it locally against
scenario-authenticator's published RS256 key.

## 4b — Health check

```bash
API_PORT=$(vrooli scenario port device-sync-hub API_PORT)
curl -s "http://localhost:${API_PORT}/health"
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
CLI (`device-sync-hub --help`) — see
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
