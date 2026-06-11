# Configuration — CLI Health

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
| `SQLITE_PATH` | `${SCENARIO_DATA_DIR}/cli-health.db` | Override SQLite file location. The default routes through `api-core/storage` and resolves to a writable per-scenario data directory. |
| `API_TOKEN` | unset | Shared bearer token for CLI ↔ API auth (only enforce in production deployments). |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `cli-health` the prefix is the scenario id upper-cased with
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
| `lifecycle.test` | which test command to invoke |
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
2. `${XDG_CONFIG_HOME}/vrooli/cli-health/config.json`
3. `~/.vrooli/config/cli-health/config.json`
4. `~/.config/vrooli/cli-health/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
cli-health configure api_base http://localhost:15001/api/v1
cli-health configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port cli-health API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
cli-health`").

## Test/CI configuration

| File | Owns |
|---|---|
| `.vrooli/testing.json` | Test categories — lint, unit, business checks (endpoints, CLI commands), Lighthouse, bundle size |
| `.vrooli/lighthouse.json` | Lighthouse pages, thresholds, Chrome flags |
| `.vrooli/endpoints.json` | API endpoint manifest (path, method, status codes, request/response shapes, CLI mapping) |
| `.github/workflows/test.yml` | CI gate — UI lint + test, Go vet + race + coverage, E2E binary smoke |

These files are read by tooling (`vrooli scenario test`, `test-genie`,
the doc viewer) — keep them in sync with the code they describe.

## Search tuning (`.vrooli/search.json`)

The command-search subsystem is **not** tuned through env vars. Its tuning factors
live in `.vrooli/search.json` — the scenario-owned SSOT for the search descriptor,
the tuning block, and the golden test corpus. The old `CLI_HEALTH_RERANK_*` /
`CLI_HEALTH_EMBED_TASK_PREFIX` env reads and the `NewDenseEngine(...)` code literal
are gone; the schema + per-knob dashboard live in the engine package
[`packages/ai-go/search/docs/reference/search-json.md`](../../../../packages/ai-go/search/docs/reference/search-json.md).

| What | Value | Notes |
|---|---|---|
| Provider id | `cli-health.commands` | the cross-scenario CLI-command corpus. |
| Engine | `dense` | terse 1-line commands embed well; no sparse leg needed. |
| `embed_task_prefix` | `true` | nomic `search_query:`/`search_document:` — +0.20 recall@5 on this corpus. |
| `rerank_enabled` + `rerank_blend` | `true` + `true` | the cross-encoder collapses gibberish to ~0 (junk rejection); RRF blend keeps canonical hits from being buried. |
| Recall gate | `recallGateK = 5`, `recallGateTarget = 0.8` (test constants) | the per-build REQ-P0-004 gate grades `expect_ids` within top-K; K and the bar are gate policy in `recall_test.go`, not corpus fields. |

This is the measured-best `aisearch.CommandCorpusTuning()` preset (recall@5
`0.50 → 0.70`). At boot, `main.go` reads the `tuning` block and wires the engine
with `NewServiceForTuning(...)` (engine shape chosen by **data**, not a code
literal), then self-registers the whole file with `search-hub` via
`searchregister.Register` (idempotent upsert; the returned control token is cached
for the secured control plane). search-hub's `evals sweep` is the loop that
refines these values and writes them back — see the
[search-hub tuning recipe](../../../../scenarios/search-hub/docs/reference/configuration.md#search-tuning-control-surface).

**Control plane (token-gated).** cli-health implements the shared
`SearchControlService` (`handlers/searchcontrol`): `Reindex`/`ReindexStatus`/
`ReindexCancel` + `WriteConfig` (the RPC search-hub calls to persist a swept
tuning back into this file). It is gated solely by the **registration-minted
control token** — search-hub is its only holder, so only search-hub's sweep can
drive these verbs; there is **no env flag**. A scenario that does not want to be
auto-tuned at all simply omits `reindex_endpoint`/`config_endpoint` from its
`search.json` (tunability is declared in the SSOT, not toggled by the
environment). Likewise the query-time override channel (`handlers/search`) is
token-gated only: a public request carries no token and gets ordinary search.

**In-process index-time apply.** A *query-time* tuning change (rerank / blend /
floor) takes effect immediately on the next request. An *index-time* recipe change
(`embed_task_prefix`, an in-dimension `embed_model` swap) now applies **live**:
`WriteConfig` calls `ApplyTuning`, which rebuilds the engine for the new tuning and
re-embeds the corpus with the new recipe **without a restart** (the engine is held
behind an RWMutex and swapped atomically; the sync loop resolves the current
reconciler each tick). A *structural* change (`engine` dense↔hybrid) flips the
collection's sparse-vector layout — the schema guard surfaces that as an error
without dropping data, so that one arm still needs a manual collection rebuild /
restart. See [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).

## Cross-references

- [`QUICKSTART.md`](../QUICKSTART.md) — boot the scenario in 5 minutes
- [`api-endpoints.md`](api-endpoints.md) — endpoint reference
- [`cli-commands.md`](cli-commands.md) — CLI command reference
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for env/port/lifecycle issues
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — why these surfaces exist
