# Configuration: git-control-tower

The scenario reads its configuration from environment variables and
the Vrooli lifecycle's per-scenario service configuration.

## Lifecycle-managed values

| Variable | Set by | Used for |
| --- | --- | --- |
| `API_PORT` | Vrooli lifecycle | API HTTP listen port (auto-assigned). |
| `UI_PORT`  | Vrooli lifecycle | UI HTTP listen port (auto-assigned). |
| `VROOLI_BUILD_MODE` | User shell when invoking `vrooli scenario restart` | Set to `profile` to produce the perf-build channel; see [docs/perf/2026-05-03-after-fixes.md](../perf/2026-05-03-after-fixes.md). |

## Scenario-specific environment

The Go API reads (search [CODE: api/main.go]):

- Standard `*_PORT` resolution from cli-core conventions.
- Optional database / cache configuration through `db.go` (SQLite path
  resolution; uses lifecycle-provisioned default when unset).
- Optional Ollama / OpenRouter endpoints for AI commit message
  suggestions ([REQ: OT-P1-002]).
- Optional Redis URL for diff/status caching.

## CLI configuration

The CLI follows the cli-core pattern:

- Auto-discovers `API_PORT` via `cliutil.DetectPortFromVrooli()`.
- Honors a project-local `~/.config/git-control-tower/config.json` for
  persistent flags (none active by default).

## Baseline snapshot & visual diff behavior

`baseline snapshot` **starts** one comprehensive, server-durable test-genie run
and returns immediately with the run id + ETA — it does not block for the run to
finish. The pin + manifest write happen server-side when the run completes
(durable across client disconnect). Follow the run with the streaming verb
`test-genie runs follow <scenario> <run-id>`; once it completes, the baseline is
queryable via `baseline show`/`baseline diff`. The CLI's short start ceiling is
`snapshotStartCeiling` (2m) — it bounds only the fast start call, not the run.

The **visuals** surface of `baseline diff` is **advisory**. test-genie delegates
pixel comparison through `CompareRunVisuals` to ui-health, where the
`UI_HEALTH_VISUAL_*` levers live; git-control-tower renders the per-page deltas
as the neutral **`changed`** tier ("changed — review before/after", with the
change magnitude). A visual difference is never a
failure here and never affects the diff exit code (regression → 1,
not-comparable → 2, everything else incl. `changed` → 0). A clearly-broken render
is caught earlier — it fails its phase at smoke time and shows on the test
surface, not the visuals surface.

## Where to extend

Add new env vars in [CODE: api/routes.go] (for API-level) or the
relevant cli-core derived `StandardScenarioEnv` block. Document each
new var here; cross-link from the file's first reader site with
`// DOC: docs/reference/configuration.md#<name>`.
