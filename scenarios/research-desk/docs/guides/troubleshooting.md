# Troubleshooting — Research Desk

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
vrooli scenario status research-desk
make stop
vrooli scenario start research-desk --clean-stale
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
vrooli scenario port research-desk UI_PORT
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
research-desk status --auto-start

# Or override the API base for this invocation
research-desk status --api-base "http://localhost:$(vrooli scenario port research-desk API_PORT)/api/v1"
```

### CLI behaves like an old version after editing source

cli-core auto-rebuilds the binary when sources change, but only
before commands marked `NeedsAPI: true`. If your edit landed in a
non-API command, force a rebuild:

```bash
make setup   # rebuilds CLI from sources
```

### `research-desk configure` doesn't persist

Check which config-file path resolved (precedence in
[`../reference/configuration.md`](../reference/configuration.md#cli-config-file)).
Most commonly `~/.vrooli/config/research-desk/config.json`. The
parent directory is created on first write — if that fails, your
home directory is read-only or `XDG_CONFIG_HOME` is set to an
unwritable path.

## Build and dependencies

### `go build` fails with `cgo` errors

The build uses `CGO_ENABLED=0`. Codex `goals_1.sqlite` files are read
through `modernc.org/sqlite` (pure Go). If
your build fails with cgo errors, a recently added dependency wants C:

```bash
cd api && go list -deps -f '{{`{{.ImportPath}}: {{.CgoFiles}}`}}' ./... | grep -v ': $'
```

Replace the offending dependency with a pure-Go alternative.

### `pnpm install` fails or installs the wrong tree

Run normal setup from the scenario directory:

```bash
make setup
```

For dependency additions or changes, use
`scenario-dependency-analyzer deps install` through
package governance (`path:docs/package-governance.md`). Do not run a raw
package manager.

The UI is a standalone pnpm project, isolated from the repo-root
`packages/*` workspace. The template's `ui/pnpm-workspace.yaml` is a
workspace boundary: it stops pnpm from discovering the parent workspace
and keeps the UI's lockfile and overrides in scope. Keep this file in place.

If setup reports "Scope: N workspace projects" and includes the repo-root
workspace, check `ui/pnpm-workspace.yaml`. Restore a missing boundary file
from the scenario's template, then rerun `make setup` from the scenario
directory. Workspace isolation does not replace dependency governance.

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

The E2E harness boots the actual binary over an empty research home in a temporary
folder (`RESEARCH_DESK_RESEARCH_DIR`, `RESEARCH_DESK_STATE_DIR`, `RESEARCH_DESK_CODEX_DIR`)
and waits for `/health`. The desk has no database to bootstrap; a hang is the binary not
starting. Read the boot log the harness prints.

### UI unit phase fails with `COVERAGE_BELOW_COMMITTED`

The tests passed but a coverage metric fell below the floor this scenario
committed to in `.vrooli/testing.json` (`coverage.committed`). The finding
names each metric with its measured and committed values. Add or restore
tests for the code whose coverage dropped; do not lower the commitment. When
coverage rises, ratchet the floor with
`unit-health fix apply research-desk --rules COVERAGE_COMMITMENT_BEHIND`. See
[coverage target and committed floor](/scenarios/unit-health/docs/reference/configuration.md#coverage-target-and-committed-floor).

### Coverage gate fails (`API coverage 71.4% < 75%`)

Coverage dropped below the floor. The fix is to add tests to the
file the report names — never to lower the threshold. Floors live in
[Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md#coverage-thresholds).

## Research home

### Every commission reads `unknown`, or `now` fails with "no research home"

The API reads `~/.vrooli/research` of the user running it. Check that the folder
exists for that user; `GET /health` reports `research-home` as degraded with the
path it tried. Nothing is written there by reading.

### Mission checks or ceilings read `unknown`

They come from `scenarios/research-desk/skills/research-coordination/`
(`scripts/mission_check.py`, `templates/limits.json`) run with `python3`. The
reason beside `unknown` names the missing file or the script's error; run the
script by hand with the same arguments to see its output.

## Proto codegen

### "type not found" after editing a `.proto`

You haven't regenerated. From the workspace root:

```bash
make generate
```

The generator runs entirely on local plugins (no BSR network calls) and
writes to language-specific output paths: Go under
`packages/proto/gen/go/research-desk/v1/`, TypeScript under
`packages/proto/gen/typescript/research-desk/v1/`, and Python under
`packages/proto/gen/python/research_desk/v1/`.

### Codegen ran but Go imports still fail

The generated package paths follow `packages/proto/gen/go/research-desk/v1/<domain>/`.
After a rename, run `scenario-dependency-analyzer deps reconcile --scenario research-desk --apply`
so both modules (`api/` and `cli/`) record what the new import paths need.

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
