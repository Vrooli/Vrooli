# Quickstart — Scenario Authenticator

Get the fleet's Identity Provider (IdP) running locally in under five
minutes. The lifecycle handles ports, environment variables, and
dependencies — you should not need to set anything by hand.

> **Status: documentation-first orientation.** This describes the
> **target** first-run experience from [`../PRD.md`](../PRD.md). Setup,
> start, open, and test (steps 1–3, 5) work today on the scaffold. The
> IdP-specific steps (create the default realm, first user, JWKS verify
> — step 4) depend on the `realms`/`identity`/`tokens` domains, which are
> **not implemented yet**; those commands are marked clearly below.

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
vrooli scenario port scenario-authenticator UI_PORT
```

The UI hosts three audiences once built (PRD §UX): the **admin console**
(realms, users, roles, sessions, audit), **end-user self-service**, and
the **hosted login/consent** screens an RP redirects to. Today the
scaffold UI renders live `/health` data and the worked example pane.

## 4 — Confirm health + JWKS, then bootstrap

The IdP's identity files live in the storage root:

```bash
# SQLite identity store (realms, users, credential hashes, token families, audit):
echo "${SQLITE_PATH:-${SCENARIO_DATA_DIR}/scenario-authenticator.db}"
# Signing keypair (load-or-generate, RS256) — BACK THIS UP; losing it invalidates all tokens:
#   ${storage-root}/private.pem  and  public.pem
```

Confirm the API is healthy and the public verification key is being
served (the JWKS endpoint is how Relying Parties verify tokens locally):

```bash
API_PORT=$(vrooli scenario port scenario-authenticator API_PORT)
curl -s "http://localhost:${API_PORT}/health"
curl -s "http://localhost:${API_PORT}/.well-known/jwks.json"   # active public key (target)
```

Then create the **default realm** (which issues `aud`-scoped tokens) and
the first admin user. These commands depend on the **unbuilt P0
`realms`/`identity` domains** — they are the target shape, not working
yet:

```bash
# TARGET — depends on unbuilt domains (realms, identity):
scenario-authenticator realms ensure-default                  # idempotent default realm
scenario-authenticator users create --realm default \
  --email admin@example.com --role admin                      # first admin (Argon2id-hashed)
```

Proto-typed calls hit
`/vrooli.scenario_authenticator.v1.<domain>.<Service>/<Method>` (Connect);
the only REST endpoints are non-RPC web standards (JWKS, OAuth callbacks).

## How a Relying Party points at it

This scenario is consumed **API-to-API** — never via cross-origin browser
calls. An adopting scenario (a Relying Party / RP) integrates like this
(the device-sync-hub reference pattern; full contract in
[`concepts/INTEGRATIONS.md`](concepts/INTEGRATIONS.md)):

1. **Resolve by slug**, not a hardcoded URL/port: resolve
   `scenario-authenticator` at runtime via `api-core/discovery`.
2. **Fetch JWKS once and cache it** from `/.well-known/jwks.json`.
3. **Verify the token locally** on every request — RS256-locked (reject
   `none`/HS confusion), offline, **no per-request callback**.
4. **Check `aud` against the verifying realm** and trust the claims
   (`sub`/`user_id`, `roles`, `scopes`); the RP owns its own
   authorization.
5. For interactive sign-in, the RP's API **forwards same-origin** to the
   authenticator and relays the token back — the browser never calls the
   authenticator directly.

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
CLI (`scenario-authenticator --help`) — see
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
- Read [`concepts/DOMAINS.md`](concepts/DOMAINS.md) for the target domain
  map (realms, identity, tokens, sessions, authorization, audit, mfa,
  federation, apikeys) and which P0 domains to build first.
- Read [`concepts/INTEGRATIONS.md`](concepts/INTEGRATIONS.md) for the full
  Relying-Party contract, and `../PRD.md` Appendix A–C (IdP↔RP split,
  realm primitive, carried-over crypto invariants).
- Read [`concepts/ARCHITECTURE.md`](concepts/ARCHITECTURE.md) for the
  mental model: three surfaces, proto bridge, layered API, where to
  add code.
- Read [`internal/TESTING.md`](internal/TESTING.md) before writing
  your first non-trivial test.
- For operations, see [`operations/RUNBOOK.md`](operations/RUNBOOK.md)
  (key rotation, session revocation, backup) and
  [`operations/DEPLOYMENT.md`](operations/DEPLOYMENT.md) (the two
  deployment shapes).
- Append a one-line entry to [`internal/PROGRESS.md`](internal/PROGRESS.md)
  whenever you land work, so future agents can replay the lifecycle.
