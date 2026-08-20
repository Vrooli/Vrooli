# Troubleshooting — Performance Health

Common issues that surface across any scenario built from the
`react-vite` template. Scenario-specific issues belong in
[`internal/PROBLEMS.md`](../internal/PROBLEMS.md), not here.

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
vrooli scenario status performance-health
make stop
vrooli scenario start performance-health --clean-stale
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
vrooli scenario port performance-health UI_PORT
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
performance-health status --auto-start

# Or override the API base for this invocation
performance-health status --api-base "http://localhost:$(vrooli scenario port performance-health API_PORT)/api/v1"
```

### CLI behaves like an old version after editing source

cli-core auto-rebuilds the binary when sources change, but only
before commands marked `NeedsAPI: true`. If your edit landed in a
non-API command, force a rebuild:

```bash
make setup   # rebuilds CLI from sources
```

### `performance-health configure` doesn't persist

Check which config-file path resolved (precedence in
[`../reference/configuration.md`](../reference/configuration.md#cli-config-file)).
Most commonly `~/.vrooli/config/performance-health/config.json`. The
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
`${XDG_DATA_HOME:-~/.local/share}/vrooli/performance-health/`.

### Coverage gate fails (`API coverage 71.4% < 75%`)

Coverage dropped below the floor. The fix is to add tests to the
file the report names — never to lower the threshold. Floors live in
[`../internal/TESTING.md`](../internal/TESTING.md#coverage-thresholds).

## Storage

### `SQLITE_PATH` resolves to an unwritable directory

The default route is `${SCENARIO_DATA_DIR}/performance-health.db` via
`api-core/storage`. If your filesystem is unusual (read-only home,
strict sandboxing), override:

```bash
export SQLITE_PATH=/tmp/performance-health.db
make start
```

The schema is embedded and idempotent, so a fresh path is always
safe.

### "database is locked"

SQLite single-writer behaviour. If two processes (e.g., a stale API
plus a new one) hold the file, find and kill the older one:

```bash
fuser "$(echo "${SQLITE_PATH:-${SCENARIO_DATA_DIR}/performance-health.db}")"
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
`packages/proto/gen/go/performance-health/v1/`, TypeScript under
`packages/proto/gen/typescript/js/performance-health/v1/`, and Python under
`packages/proto/gen/python/performance_health/v1/`.

### Codegen ran but Go imports still fail

The generated package paths follow `packages/proto/gen/go/performance-health/v1/<domain>/`.
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

- [`../QUICKSTART.md`](../QUICKSTART.md) — first-touch setup
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and config precedence
- [`../reference/cli-commands.md`](../reference/cli-commands.md) — CLI command reference
- [`../internal/TESTING.md`](../internal/TESTING.md) — test patterns and coverage gates
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — scenario-specific issues
