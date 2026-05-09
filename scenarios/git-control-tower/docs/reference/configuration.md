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

## Where to extend

Add new env vars in [CODE: api/routes.go] (for API-level) or the
relevant cli-core derived `StandardScenarioEnv` block. Document each
new var here; cross-link from the file's first reader site with
`// DOC: docs/reference/configuration.md#<name>`.
