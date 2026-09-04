# Troubleshooting — Compute Manager

Symptoms with a stable diagnosis and a repeatable fix. Most of this guide
covers issues that surface across any scenario built from the
`react-vite` template; the last section covers the ones specific to
Compute Manager. Known defects, deliberate debt and blockers waiting on
another scenario belong in
[`internal/PROBLEMS.md`](../internal/PROBLEMS.md), not here. Incident
procedures, where the response is a sequence of decisions rather than one
fix, belong in
[`operations/RUNBOOK.md`](../operations/RUNBOOK.md).

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

If a process is genuinely orphaned, let the control plane reconcile its owned state:

```bash
vrooli scenario status compute-manager
make stop
vrooli scenario start compute-manager --clean-stale
```

**Don't** use `make stop && make start` on autopilot — `make restart`
is the canonical command and gives the lifecycle a chance to clean
state in order.

### Vrooli CLI command not found

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
vrooli scenario port compute-manager UI_PORT
```

Then open the localhost UI URL shown by the port command directly. If the URL works
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
compute-manager status --auto-start

# Or override the API base for this invocation
compute-manager status --api-base "http://localhost:$(vrooli scenario port compute-manager API_PORT)/api/v1"
```

### CLI behaves like an old version after editing source

cli-core auto-rebuilds the binary when sources change, but only
before commands marked `NeedsAPI: true`. If your edit landed in a
non-API command, force a rebuild:

```bash
make setup   # rebuilds CLI from sources
```

### `compute-manager configure` doesn't persist

Check which config-file path resolved (precedence in
[`../reference/configuration.md`](../reference/configuration.md#cli-config-file)).
Most commonly `~/.vrooli/config/compute-manager/config.json`. The
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
cd api && go list -deps -f '{{.ImportPath}}: {{.CgoFiles}}' ./... | grep -v ': \[\]$'
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
schema bootstrap fails (corrupt SQLite file, unwritable data
directory), `/health` never returns ready and the test times out.
Wipe the test data dir and retry. The default lives under
`${XDG_DATA_HOME:-~/.local/share}/vrooli/compute-manager/`.

### Coverage gate fails (`API coverage 71.4% < 75%`)

Coverage dropped below the floor. The fix is to add tests to the
file the report names — never to lower the threshold. Floors live in
[`../internal/TESTING.md`](../internal/TESTING.md#coverage-thresholds).

## Storage

### The resolved database directory is not writable

The default route is `${SCENARIO_DATA_DIR}/compute-manager.db` via
`api-core/storage`. If your filesystem is unusual (read-only home,
strict sandboxing), override:

```bash
export VROOLI_STORAGE_ROOT=/tmp/vrooli-storage
make start
```

The schema is embedded and idempotent, so a fresh path is always
safe.

### "database is locked"

SQLite single-writer behaviour. If two processes (e.g., a stale API
plus a new one) hold the file, find and kill the older one:

```bash
fuser "${SCENARIO_DATA_DIR}/compute-manager.db"
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
`packages/proto/gen/go/compute-manager/v1/`, TypeScript under
`packages/proto/gen/typescript/compute-manager/v1/`, and Python under
`packages/proto/gen/python/compute_manager/v1/`.

### Codegen ran but Go imports still fail

The generated package paths follow `packages/proto/gen/go/compute-manager/v1/<domain>/`.
After a rename, run `go mod tidy` from the affected module (`api/`
and `cli/`) so the import paths resolve.

## Compute Manager specifics

> **Status: none of these can happen yet.** There is no provider adapter,
> no reservation path, no reconciler and no enrollment path, so no
> symptom below has ever been observed. Each entry records the intended
> diagnosis for a failure the design predicts, so the first person to hit
> it does not start from nothing. Treat every command shown as an
> intended command surface, not an available one, and check Part 1 of
> [`../reference/cli-commands.md`](../reference/cli-commands.md) for what
> actually dispatches.

### A provider call times out, and you do not know whether a machine exists

This is the expensive failure in this scenario, and it is why intent is
written before any provider is called. A timeout is not a failure with a
known outcome; it is an unknown outcome, and the machine may well be
running and billing.

Do not retry with a fresh request. A retry that carries a new
idempotency key is how one lost response becomes two machines. The intent
holds the original key precisely so a retry can reuse it.

```bash
compute-manager instance list --json
```

Match each `provider-only` finding to a stuck intent by its idempotency
key. A match means the provider created the machine and the response
never arrived. Recover it with
[Quarantine An Unaccounted Instance](../operations/RUNBOOK.md#quarantine-an-unaccounted-instance),
whose ownership branch is exactly this case. A rate limit and a plain
outage are the same procedure with a safer starting point; all three are
covered by
[Respond To A Provider Outage, Rate Limit Or Timeout](../operations/RUNBOOK.md#respond-to-a-provider-outage-rate-limit-or-timeout).

Distinguishing the three matters because only the timeout leaves an
unknown outcome. A provider that refuses cleanly created nothing.

### A capacity request is refused for insufficient credit

This is a correct refusal, not an incident. Credit is reserved
server-side before any provider is called, so a request that would run
past available credit is refused before it can cost anything.

```bash
compute-manager instance list --json
```

The refusal names which ceiling was reached and what would raise it. Read
that first: a tenant ceiling and an empty wallet are different problems
with different owners, and the ceiling is computed from this scenario's
own meter rather than from a provider spend alert.

Escalate only when credit is demonstrably present and the request is
still refused. Two causes look identical from the outside and must not
be confused:

- **The business suite is unreachable.** Provisioning fails closed by
  design, so an unreachable business suite refuses every request. That is
  deliberate. Do not add a bypass: a machine that boots unmetered is cost
  that grows hourly and cannot be recovered afterwards. Check
  `landing-page-business-suite` `/health` first.
- **A server error is being reported as out-of-credit.** The upstream
  reference client discards the response body on a non-2xx, which makes a
  genuine refusal and a server fault indistinguishable. This scenario
  must not inherit that, and if a refusal carries no named ceiling, this
  is the first thing to suspect.

### The reconciler is stale, and an empty findings list means nothing

A reconciler that has stopped looks exactly like a fleet with nothing
wrong. That is the failure this scenario's observability is built around,
so an empty list is only trustworthy when you have also read the last
sweep time.

```bash
compute-manager instance list --json
```

If the on-demand sweep succeeds but the background loop's last success is
old, the loop is the problem and not the sweep. Read `make logs` for the
sweep and establish which of the three it is: the loop is not running,
the loop runs and every pass fails, or the loop runs and its query
returns nothing it should have returned.

Do not restart the scenario before reading the backlog. A restart resets
the evidence, and the size of the backlog is what says how long the
reconciler has been dead.

The expiry sweeper has the same failure shape and its own procedure,
[Respond To An Expiry Sweep Failure](../operations/RUNBOOK.md#respond-to-an-expiry-sweep-failure).

### Instances are created but never become trusted nodes

Enrollment delegates entirely to `vrooli-bridge`, whose owner-gated
onboarding-key procedure supplies the public key used in first boot.

```bash
compute-manager status --json               # scenario health and dependency readiness
compute-manager instance get "<instance-id>" --json
vrooli-bridge machines list --json
```

The scenario contains no SSH implementation and no password-bearing
enrollment path. If key retrieval or onboarding fails, use the durable queue
and retry path rather than introducing a credential shortcut.

Un-enrolled is a degradation and not an outage. The instance is still
created, still metered and still expiring, and enrollment queues and
retries. If the bridge is reachable and the key is retrievable and
enrollment still does not complete, that is a real defect rather than the
known blocker; record it in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) with the instance
identifier and the bridge response.

## When to add a new entry here

Add to this guide if:

- The issue can occur in **any** scenario from this template, or is a
  recurring Compute Manager symptom with a stable diagnosis
- The root cause is non-obvious from the error message
- The fix is a stable, repeatable command

Add to [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) if:

- The issue is specific to your scenario
- It's tech debt or a known workaround pending a real fix
- It needs scenario-specific context to act on

## Cross-references

- [`../QUICKSTART.md`](../QUICKSTART.md) — first-touch setup
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and config precedence
- [`../reference/cli-commands.md`](../reference/cli-commands.md) — CLI command reference
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md): operator procedures and incident responses
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md): the signals these symptoms show up in
- [`../internal/TESTING.md`](../internal/TESTING.md) — test patterns and coverage gates
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — scenario-specific issues
