# Troubleshooting — Scenario Authenticator

Identity-Provider-specific issues first, then the common
`react-vite`-template issues that surface in any scenario.
Scenario-specific tech debt belongs in
[`internal/PROBLEMS.md`](../internal/PROBLEMS.md), not here.

> **Status: documentation-first orientation.** The auth-specific section
> below describes the **target** failure modes and fixes from
> [`../../PRD.md`](../../PRD.md); the auth domains are not implemented yet,
> so these are the intended diagnoses, not reproduced bugs. The
> template-issues sections are live today.

## Identity Provider issues

### Redis unavailable — sessions and rate limiting degraded

Redis is a **required** resource (not optional). When it is down:

- Token issuance and verification keep working — verification is stateless
  against JWKS and never touches Redis.
- **Session revocation / "revoke all" cannot be honored**, and
  cross-replica rate-limit accuracy degrades.

The system must **fail safe, not open**: `/health` should report
unhealthy and stale sessions must not be silently accepted. Recover:

```bash
make status     # confirm the Redis resource state
make restart     # bring Redis + API back under the lifecycle
API_PORT=$(vrooli scenario port scenario-authenticator API_PORT)
curl -s "http://localhost:${API_PORT}/health"   # confirm Redis reachable again
```

After recovery, re-revoke any sessions that should have been killed during
the outage. See [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md).

### Tokens invalidate on restart — signing key not persisting

If every Relying Party suddenly rejects previously-valid tokens after a
restart, the **signing keypair regenerated** instead of loading. The
load-or-generate pattern mints a *fresh* key when the persisted PEM pair
is missing/unreadable, silently invalidating all live tokens.

Check the storage-root keypair:

```bash
# The keypair lives in the storage root alongside the SQLite DB:
echo "${SQLITE_PATH:-${SCENARIO_DATA_DIR}/scenario-authenticator.db}"
#   expect: ${storage-root}/private.pem  and  public.pem  to PERSIST across restarts
```

If the storage root is ephemeral (e.g. a tmpfs or a wiped data dir), point
it at durable storage and **restore the keypair from backup** so issued
tokens verify again. The fix is never to "just regenerate" — that locks in
the invalidation. Back the keypair up with the DB; see
[`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md#signing-key-backup--rotation).

### `aud` mismatch — RP verifying against the wrong realm

Symptom: tokens verify cryptographically (signature is valid) but the RP
rejects them, or accepts tokens it shouldn't. Each realm issues
`aud`-scoped tokens; verification must reject a token whose `aud` does not
match the verifying realm (OT-P0-008). Causes and fixes:

- The RP is verifying against the wrong realm's expected `aud` — align the
  RP's expected audience with the realm the user authenticated to.
- A single default-realm deployment still enforces `aud` — a token minted
  for realm A is correctly rejected by realm B. This is **not** a bug; it
  is cross-tenant isolation working as designed.

### `api-core/discovery` can't resolve the authenticator

Symptom: an RP fails closed (treats requests as unauthenticated) because
it cannot resolve `scenario-authenticator` by slug. Almost always the
scenario is not started:

```bash
make status                                   # is scenario-authenticator running?
vrooli scenario start scenario-authenticator   # start it if not
```

RPs must resolve **by slug** via `api-core/discovery` — never a hardcoded
URL/port. If the slug resolves but the RP still can't reach it, check the
RP's discovery client, not a baked-in address.

### RP token verification failing — stale JWKS cache

Symptom: verification fails after a signing-key rotation, with a "no
matching `kid`" or signature error. The RP cached an old JWKS that lacks
the new key. The correct RP behavior is **refetch-on-miss**: when a token
references a `kid` the cache doesn't have, refetch
`/.well-known/jwks.json` once and retry before rejecting. Fixes:

- Confirm the authenticator is serving the new key:
  `curl -s "http://localhost:${API_PORT}/.well-known/jwks.json"`.
- Ensure rotation kept the old `kid` published until the access-token TTL
  elapsed (overlapping `kid`s) — see
  [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md).
- If the RP caches forever with no refetch-on-miss, that is an RP bug, not
  an authenticator bug.

### Cross-origin browser call attempted (forbidden)

Symptom: a browser tries to call the authenticator directly and is
blocked (CORS / refused). **This is by design.** There are no cross-origin
browser calls anywhere in the model. The browser talks only to its own
scenario's API; if that API must reach the authenticator, it **forwards
same-origin** and relays the token back (the device-sync-hub
`internal/identity.Forwarder` pattern). Fix the integration to use the
same-origin forwarder — do not add CORS to permit a direct browser call.
See [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md).

## Lifecycle and ports

### "Port already in use" or "address already in use"

A previous scenario instance still holds the port. The lifecycle
allocates from declared ranges (`API_PORT 15000-19999`,
`UI_PORT 20000-24999`), so collisions usually mean a previous run
did not stop cleanly.

```bash
make restart
# or, if that doesn't recover:
make stop
make start
```

If a process is genuinely orphaned, find and kill it:

```bash
vrooli scenario status scenario-authenticator
# Then either:
make stop
# Or, as last resort:
pkill -f 'scenario-authenticator-api'
pkill -f 'node server.js'
```

**Don't** use `make stop && make start` on autopilot — `make restart`
is the canonical command and gives the lifecycle a chance to clean
state in order.

### `vrooli: command not found`

The Vrooli CLI isn't on your `PATH`. Run the workspace-root setup:

```bash
cd ../../..   # to the Vrooli repo root
make setup
```

Then re-source your shell or open a new terminal.

### Scenario won't open in browser

Confirm the UI server is running:

```bash
make status
vrooli scenario port scenario-authenticator UI_PORT
```

Then open `http://localhost:<UI_PORT>` directly. If the URL works
but `make open` doesn't, the issue is your default-browser handler,
not the scenario.

## API and CLI

### CLI: "API not available at http://localhost:..."

The API isn't running, or it's running on a different port than the
CLI is looking at. Resolution order is documented in
[`../reference/configuration.md`](../reference/configuration.md#api-base-resolution-precedence).

Quick fixes:

```bash
# Start the API if it's not running
make start

# Or let the CLI auto-start it
scenario-authenticator status --auto-start

# Or override the API base for this invocation
scenario-authenticator status --api-base "http://localhost:$(vrooli scenario port scenario-authenticator API_PORT)/api/v1"
```

### CLI behaves like an old version after editing source

cli-core auto-rebuilds the binary when sources change, but only
before commands marked `NeedsAPI: true`. If your edit landed in a
non-API command, force a rebuild:

```bash
make setup   # rebuilds CLI from sources
```

### `scenario-authenticator configure` doesn't persist

Check which config-file path resolved (precedence in
[`../reference/configuration.md`](../reference/configuration.md#cli-config-file)).
Most commonly `~/.vrooli/config/scenario-authenticator/config.json`. The
parent directory is created on first write — if that fails, your
home directory is read-only or `XDG_CONFIG_HOME` is set to an
unwritable path.

### API returns `400 invalid_request` on multipart upload

Likely cause: the upload request is not valid `multipart/form-data` or
is missing the `file` part. Proto-typed operations use Connect-RPC and
will surface Connect codes such as `invalid_argument`; `invalid_request`
is reserved for REST exceptions like file upload.

## Build and dependencies

### `go build` fails with `cgo` errors

The template requires `CGO_ENABLED=0`. SQLite is via `modernc.org/sqlite`
(pure-Go), and the CI gate proves no C dependency has snuck in. If
your build fails with cgo errors, a recently added dependency wants C:

```bash
cd api && go list -deps -f '{{`{{.ImportPath}}: {{.CgoFiles}}`}}' ./... | grep -v ': $'
```

Replace the offending dependency with a pure-Go alternative.

### `pnpm install` fails or installs the wrong tree

The UI is a standalone pnpm project, isolated from the repo-root
`packages/*` workspace. Since template 1.1.0, `ui/pnpm-workspace.yaml`
is a **workspace boundary**: pnpm stops its upward workspace search
there, so a plain `pnpm install` from `ui/` is always scoped to the UI.
Do not delete that file — without it, a plain install walks up to the
repo root, joins the root workspace, ignores this project's lockfile
and overrides, and regenerates stray root artifacts.

```bash
cd ui
pnpm install                      # safe with the boundary file present
pnpm install --ignore-workspace   # equivalent; used by lifecycle commands
```

`make setup` does this automatically. If the install ever shows
"Scope: N workspace projects", the boundary file is missing — restore
it, delete `ui/node_modules`, and re-run.

### UI build is slow (5–10 minutes)

`vite build` processes 4400+ modules in the production bundle. This
is expected. Use `pnpm dev` (via `make start`) for the fast iteration
loop and reserve `pnpm build` for verification.

### `pnpm strings:check` fails in CI

The codegen is out of sync with `en.json`. Regenerate locally:

```bash
cd ui
pnpm strings:gen
git add src/consts/strings.generated.ts
```

Commit `en.json`, the locale files, **and** `strings.generated.ts`
together — never one without the others.

## Tests

### Vitest test fails with `t('app.title')` returning the literal key

That's by design. The test runner sets i18next to `cimode` so
component tests are copy-independent. Assert against the typed
`strings.x.y` registry, not real translations. If your test needs
real-locale behaviour, opt back in with `await setLocale("en")` in
its own `beforeEach` (see `App.test.tsx` for an example).

### Go test fails with `dial tcp: connection refused` on `httpx.NewLiveServer`

The harness binds to `127.0.0.1:0` and lets the OS assign a port. If
this fails, you have a system-level limit (`ulimit -n`, IPv4 disabled).
Increase open-file limits or check that loopback is reachable.

### API E2E test (`go test -tags=e2e`) hangs

The E2E harness boots the actual binary and waits for `/health`. If
schema bootstrap fails (corrupt SQLite file, unwritable
`SQLITE_PATH`), `/health` never returns ready and the test times out.
Wipe the test data dir and retry. The default lives under
`${XDG_DATA_HOME:-~/.local/share}/vrooli/scenario-authenticator/`.

### Coverage gate fails (`API coverage 71.4% < 75%`)

Coverage dropped below the floor. The fix is to add tests to the
file the report names — never to lower the threshold. Floors live in
[`../internal/TESTING.md`](../internal/TESTING.md#coverage-thresholds).

## Storage

### `SQLITE_PATH` resolves to an unwritable directory

The default route is `${SCENARIO_DATA_DIR}/scenario-authenticator.db` via
`api-core/storage`. If your filesystem is unusual (read-only home,
strict sandboxing), override:

```bash
export SQLITE_PATH=/tmp/scenario-authenticator.db
make start
```

The schema is embedded and idempotent, so a fresh path is always
safe.

### "database is locked"

SQLite single-writer behaviour. If two processes (e.g., a stale API
plus a new one) hold the file, find and kill the older one:

```bash
fuser "$(echo "${SQLITE_PATH:-${SCENARIO_DATA_DIR}/scenario-authenticator.db}")"
```

`make stop` followed by `make start` is usually sufficient.

## Proto codegen

### "type not found" after editing a `.proto`

You haven't regenerated. From the workspace root:

```bash
make generate
```

The generator runs entirely on local plugins (no BSR network calls) and
writes to language-specific output paths: Go under
`packages/proto/gen/go/scenario-authenticator/v1/`, TypeScript under
`packages/proto/gen/typescript/js/scenario-authenticator/v1/`, and Python under
`packages/proto/gen/python/scenario_authenticator/v1/`.

### Codegen ran but Go imports still fail

The generated package paths follow `packages/proto/gen/go/scenario-authenticator/v1/<domain>/`.
After a rename, run `go mod tidy` from the affected module (`api/`
and `cli/`) so the import paths resolve.

## When to add a new entry here

Add to this guide if:

- The issue can occur in **any** scenario from this template
- The root cause is non-obvious from the error message
- The fix is a stable, repeatable command

Add to [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) if:

- The issue is specific to your scenario
- It's tech debt or a known workaround pending a real fix
- It needs scenario-specific context to act on

## Cross-references

- [`../QUICKSTART.md`](../QUICKSTART.md) — first-touch setup + RP integration
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — key rotation, session revocation, Redis recovery, backup
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — the Relying-Party contract (slug resolve, JWKS verify, same-origin forward)
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and config precedence
- [`../reference/cli-commands.md`](../reference/cli-commands.md) — CLI command reference
- [`../internal/TESTING.md`](../internal/TESTING.md) — test patterns and coverage gates
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — scenario-specific issues
