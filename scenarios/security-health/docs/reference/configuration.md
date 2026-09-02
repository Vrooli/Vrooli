# Configuration — Security Health

How this scenario is configured — env vars consumed by the binaries,
the `.vrooli/service.json` manifest, and the per-user CLI config file.

The lifecycle (`vrooli scenario start`, `make start`) sets every
required variable automatically. You only need this reference when
running a binary by hand or when a scenario adds a new variable.

## Environment variables

### Required at runtime (set by the lifecycle)

| Variable | Range / format | Purpose |
|---|---|---|
| `API_PORT` | `15000-19999` | Port for the Go API server |
| `UI_PORT` | `20000-24999` | Port for the production UI server (`ui/server.js`) |

If the scenario adds WebSocket channels on the existing API or UI server, do
not add another `ports` entry. Declare an additional port only when the
scenario starts a separate listener process.

The canonical bands all sit below 32768 so Linux never hands out the
ports as outbound source ports. See the project-level
[port-allocation reference](../../../../docs/reference/port-allocation.md)
for the full policy.

### Optional overrides

| Variable | Default | Purpose |
|---|---|---|
| _(none)_ | — | The SQLite file location is **not** configurable through the environment. It is resolved from the scenario's own identity by `api-core/storage`, so no inherited variable can point one scenario at another's database. To relocate storage for a test run, set `VROOLI_STORAGE_ROOT`, which redirects the whole class tree and stays scenario-agnostic. |
| `API_TOKEN` | unset | Shared bearer token for CLI ↔ API auth (only enforce in production deployments). |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |
| `SECURITY_HEALTH_RECONCILE_INTERVAL` | `5m` | Base cadence of the background fleet reconcile loop (Go duration, e.g. `5m`, `10m`). A per-tick jitter of up to `interval/4` is added so a fleet of self-monitoring scenarios doesn't burst on an aligned boundary. Invalid/non-positive values fall back to the default. |
| `SECURITY_HEALTH_RECONCILE_MAX_INTERVAL` | `1h` | Ceiling for exponential backoff after unchanged fleet reconciles. Each unchanged pass doubles the effective interval; any scanner work resets it to the base interval. Jitter remains applied. |
| `SECURITY_HEALTH_SCAN_CONCURRENCY` | `4` | Max scenarios scanned in parallel during a fleet reconcile. Bounds peak CPU from the ~110 per-scenario `osv-scanner` runs; raise it to shorten a large changed-scenario pass at the cost of higher peak CPU. Minimum `1`; invalid values fall back to the default. |
| `SECURITY_HEALTH_SCANNER_CAPACITY` | `4` | Shared weighted capacity for validation scanner subprocesses. Valid range is `3-32`; invalid values fall back to `4`. Higher values admit more expensive scanner work concurrently across requests and increase peak CPU/memory. |
| `SECURITY_HEALTH_SCANNER_MAX_PROCS` | one quarter of host CPUs, floor `2` | Sets child `GOMAXPROCS` for `gosec` and `govulncheck`. Values are clamped to `1..runtime.NumCPU()`; invalid values use the default. |

Scanner weights and evidence freshness are correctness policy, not operator
preferences, so they are intentionally not configurable. Static scanners use
weights `1-2`; advisory scanners use weights `2-3`. Advisory identities include
a stable per-scenario UTC-hour epoch, while static evidence remains valid until
content, tool, or normalization policy changes. See
[`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) for input boundaries,
metrics, failure behavior, and safe invalidation.

The per-scenario `osv-scanner` result is content-cached (no flag — on by default): a
reconcile re-scans a scenario only when its resolved-version lockfiles
(`go.mod`/`go.sum`/`pnpm-lock.yaml`/`package-lock.json`/`yarn.lock`/`npm-shrinkwrap.json`),
the installed `osv-scanner` version, or the stable per-scenario advisory epoch changes — so steady-state
reconciles run near-zero scanner subprocesses while every scenario still re-scans at most
once per day to pick up newly-published vulnerabilities. The cache key folds in all of those
inputs, so it is always correctness-preserving: a real change forces a re-scan and the 25-hour
epoch bounds staleness. Scans run **online** (osv-scanner resolves the live OSV
database); offline mode was rejected because osv-scanner loads its full ~200 MB+ per-ecosystem
database into memory on every invocation (see `docs/perf/2026-06-24-reconcile-scan-incremental.md`).
When a scanner exposes a database identity, that identity is preferred over the
epoch: govulncheck reports its `DB updated` timestamp, while osv-scanner 1.9.2
did not report one in the 2026-08-28 measurement and therefore uses the
per-scenario fallback.
The reconcile loop backs off exponentially after unchanged passes, up to
`SECURITY_HEALTH_RECONCILE_MAX_INTERVAL`, and resets immediately when scanner
work is required. The reconcile loop and the on-demand `Reindex` RPC share an overlap lock so they can never
scan the fleet concurrently. Cache effectiveness is reported on the dependencies `Status`
reconcile outcome (`scans_run=… scans_skipped_cache=…`).

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `security-health` the prefix is the scenario id upper-cased with
hyphens replaced by underscores (so `my-scenario` → `MY_SCENARIO`).
The following are recognised, in precedence order (first-found wins);
substitute your scenario's prefix for `<PREFIX>`:

| Purpose | Variables |
|---|---|
| API base URL | `<PREFIX>_API_BASE`, `<PREFIX>_API_URL`, `VROOLI_API_BASE` |
| API port | `<PREFIX>_API_PORT` |
| API token | `<PREFIX>_API_TOKEN`, `VROOLI_API_TOKEN` |
| Config dir | `<PREFIX>_CONFIG_DIR`, `VROOLI_CLI_CONFIG_DIR` |
| HTTP timeout | `<PREFIX>_HTTP_TIMEOUT`, `VROOLI_HTTP_TIMEOUT` |

> **Do not** set the un-prefixed `API_PORT` for a CLI invocation —
> when CLIs run inside web-console terminals it leaks across scenarios.
> Use the scenario-prefixed form or the `--api-base` flag.

## Service manifest (`.vrooli/service.json`)

Single source of truth for everything the lifecycle needs to know.

| Section | Owns |
|---|---|
| `service` | name, display name, description, version, category, maintainers, repository URL |
| `ports` | port-name → env-var + range mapping (lifecycle allocates from these) |
| `cli` | command name, install scripts (per OS), invoke shape, freshness inputs |
| `lifecycle.health` | `/health` endpoint, startup grace period, periodic checks |
| `lifecycle.setup` | build steps + idempotency conditions (binary present, UI bundle fresh) |
| `lifecycle.develop` | how to start the running scenario |
| `lifecycle.stop` | how to shut down cleanly |
| `environment` | static env vars set for every lifecycle step |
| `dependencies.resources` | shared local resources (postgres, redis, qdrant, …) |

The template ships with `dependencies.resources: {}` — SQLite is
in-process, so no resource is required. Scenarios add resources here
when they need shared infrastructure.

## Schema bootstrap

Schema is owned per-domain. `api/internal/<dom>/schema.sql` declares
each domain's tables and is embedded into the binary via `go:embed`
from `api/internal/<dom>/schema.go::Schema()`. Cross-cutting
infrastructure (postgres extensions, custom types, cross-domain views)
lives in `api/internal/database/system.sql` — empty by default in
SQLite scenarios.

The shared registry at `api/internal/modules/registry.go::AllSchemas()`
collects them in order (system first, then domains alphabetical), and
`apidb.EnsureSchemas(ctx, db, modules.AllSchemas()...)` from
`api-core/database` applies them at startup. The path is idempotent —
all DDL uses `CREATE TABLE IF NOT EXISTS` / `ALTER TABLE … ADD COLUMN
IF NOT EXISTS`, so re-runs on every boot are no-ops.

Adding a column lands in the same diff as the Go struct field, the
repository scan, and the proto wire shape — single location, single
edit. Drops/renames in production data need the brownfield
versioned-migration helpers (`Migrate` / `MigrationProvider` in
`api-core/database`, deferred until the first scenario hits the pain).

See [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#domain-owned-schema)
for the design rationale and [`../internal/SEAMS.md`](../internal/SEAMS.md)
for the per-seam table including `notes.Schema` and
`database.SystemSchema`.

## CLI config file

The scenario CLI persists per-user configuration to a JSON file.
Resolution order (first match wins):

1. `${<PREFIX>_CONFIG_DIR}/config.json` (the scenario-prefixed env var; see "Scenario-prefixed CLI variables" above)
2. `${XDG_CONFIG_HOME}/vrooli/security-health/config.json`
3. `~/.vrooli/config/security-health/config.json`
4. `~/.config/vrooli/security-health/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
security-health configure api_base http://localhost:15001/api/v1
security-health configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port security-health API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
security-health`").

## Test/CI configuration

| File | Owns |
|---|---|
| `.vrooli/testing.json` | Test categories — lint, unit, business checks (endpoints, CLI commands), Lighthouse, bundle size |
| `.vrooli/lighthouse.json` | Lighthouse pages, thresholds, Chrome flags |
| `.vrooli/endpoints.json` | API endpoint manifest (path, method, status codes, request/response shapes, CLI mapping) |
| `.github/workflows/test.yml` | CI gate — UI lint + test, Go vet + race + coverage, E2E binary smoke |

These files are read by tooling (`vrooli scenario test`, `test-genie`,
the doc viewer) — keep them in sync with the code they describe.

## Cross-references

- [`QUICKSTART.md`](../QUICKSTART.md) — boot the scenario in 5 minutes
- [`api-endpoints.md`](api-endpoints.md) — endpoint reference
- [`cli-commands.md`](cli-commands.md) — CLI command reference
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for env/port/lifecycle issues
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — why these surfaces exist
