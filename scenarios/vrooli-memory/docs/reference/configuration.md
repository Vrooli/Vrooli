# Configuration — Vrooli Memory

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
ports as outbound source ports. See the project-level port allocation reference
(`path:../../docs/reference/port-allocation.md`) for the full policy.

### Optional overrides

| Variable | Default | Purpose |
|---|---|---|
| `SQLITE_PATH` | `${SCENARIO_DATA_DIR}/vrooli-memory.db` | Override SQLite file location. The default routes through `api-core/storage` and resolves to a writable per-scenario data directory. |
| `API_TOKEN` | unset | Shared bearer token for CLI ↔ API auth (only enforce in production deployments). |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |
| `VROOLI_MEMORY_FRONTIER_TARGET` | `16` | Target number of compaction-eligible, unpinned frontier nodes before compaction considers a reduction. This is a compaction-pressure control, not a prompt-size limit. The full recall frontier also contains non-episode roots, which are deliberately not compacted. The value is calibrated from the initial 412-entry, single-harness import: it retains a 16-node working episode frontier while requiring only cohesive episode clusters to collapse. Must be a positive integer. Recalibrate after the first multi-harness import. |
| `VROOLI_MEMORY_WAKE_BUDGET` | `96` | Maximum wake-context line budget for non-pinned frontier content in the `agent-memory` scope. Pinned content is always retained and sets `overflow=true` when it alone exceeds this value. Must be a positive integer. |
| `VROOLI_MEMORY_MAX_ENTRY_LINES` | `2` | Maximum lines contributed by one ambient memory in the `agent-memory` scope. |
| `facet_policies.resident_budget` | seeded per facet (`standing-rule=8`, `episode=12`, other resident facets=4) | Data-driven wake and standing-rule curation ceilings. Operators govern the value through the facet policy row; pin requests above the standing-rule ceiling create a trade-off proposal. |
| `pins.review_at` | 90 days after pin/reconfirmation | Operator-selected quarterly review deadline for a pin. Expiry lapses the pin without deleting the journal entry; reconfirmation updates the deadline without creating another journal row. |
| `VROOLI_MEMORY_MAINTENANCE_INTERVAL` | `6h` | In-process import-then-projection interval. A restart runs one immediate tick; `0` disables scheduled work. Each runtime operation is bounded at two minutes, and each run records start/end times and per-runtime outcomes. |

## Scope registry

`agent-memory` is created at startup from the three `VROOLI_MEMORY_*` defaults
above. Other ledgers are durable rows in the `scopes` registry and keep their
own frontier target, wake line budget, maximum entry excerpt, facet vocabulary,
retention policies, and residency budgets. Environment variables never rewrite
named scopes.

Create a scope atomically through the CLI, supplying its vocabulary as JSON:

```bash
vrooli-memory scopes create marketing \
  --label "Marketing ledger" --frontier-target 8 --wake-budget 32 \
  --max-entry-lines 2 \
  --facets-json '[{"id":"campaign","label":"Campaign","guidance":"Campaign decisions","retention_policy":"retain","resident_budget":8}]'
vrooli-memory scopes list
```

Creation rejects residency totals that cannot fit within
`wake_budget / max_entry_lines`, naming the required and available values.
Each named scope also receives a derived `vrooli-memory.scope.<id>` search-hub
provider whose recall request carries that scope explicitly.

## Compaction calibration

The compaction scorer is `mean pairwise cosine cohesion × slots freed`.
It is deliberately pressure-driven: it ranks available clusters rather than
requiring a fixed cohesion cutoff. On the initial imported corpus, the 28
episode topic-vector candidates produced 378 pairwise comparisons with cosine
similarity `0.5752` minimum, `0.7343` mean, and `0.9391` maximum. That spread
supports ranking rather than a magic threshold: isolated material remains on
the frontier until stronger clusters have freed the required slots. The first
multi-harness import is the revisit trigger because this calibration is drawn
only from Claude Code material.

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `vrooli-memory` the prefix is the scenario id upper-cased with
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
each domain's tables and is embedded into the binary through Go's embed directive
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
for the per-seam table including each domain's `<domain>.Schema` and
`database.SystemSchema`.

## CLI config file

The scenario CLI persists per-user configuration to a JSON file.
Resolution order (first match wins):

1. `${<PREFIX>_CONFIG_DIR}/config.json` (the scenario-prefixed env var; see "Scenario-prefixed CLI variables" above)
2. `${XDG_CONFIG_HOME}/vrooli/vrooli-memory/config.json`
3. `~/.vrooli/config/vrooli-memory/config.json`
4. `~/.config/vrooli/vrooli-memory/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
vrooli-memory configure api_base http://localhost:15001/api/v1
vrooli-memory configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port vrooli-memory API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
vrooli-memory`").

## Test/CI configuration

| File | Owns |
|---|---|
| `.vrooli/testing.json` | Test categories — lint, unit, business checks (endpoints, CLI commands), Lighthouse, bundle size |
| `.vrooli/lighthouse.json` | Lighthouse pages, thresholds, Chrome flags |
| `.vrooli/endpoints.json` | API endpoint manifest (path, method, status codes, request/response shapes, CLI mapping) |
| `.github/workflows/test.yml` | CI gate — UI lint + test, Go vet + race + coverage, E2E binary smoke |

These files are read by tooling (`vrooli scenario test`, `test-genie`,
the doc viewer) — keep them in sync with the code they describe.

### Unit testing policy profile

The `unit.policy_profile` block in `.vrooli/testing.json` declares the
template's unit-test policy. It is not a list of surfaces. Code Facts discovers
the actual `api`, `cli`, `ui`, and any additional code surfaces; Unit Health
joins those observed surfaces to the profile and reports drift.

The React/Vite template requires three roles:

| Role | Policy class | Baseline |
|---|---|---|
| `api` | `go_service` | Go `go test`, 75% total coverage, `api/internal/testutil`, production-import guardrail. |
| `cli` | `go_cli` | Go `go test`, 75% total coverage, `cli/internal/testutil`, app smoke test, production-import guardrail. |
| `ui` | `react_vite_ui` | Vitest through pnpm, jsdom setup, V8 coverage, 85% coverage thresholds, `ui/src/test-utils/renderWithProviders.tsx`. |

Scenario customizations are monotonic: they may add surfaces, add stricter
checks, or raise thresholds. They may not weaken the template baseline unless
the policy includes a waiver with an owner, reason, expiry or revisit trigger,
and the Unit Health finding evidence it addresses.

`unit.policy_profile` is the only unit-infrastructure contract emitted by this
template. Test orchestration knobs such as phase timeouts and presets stay in
their own top-level blocks; unit surfaces are discovered by Code Facts and
governed by the policy profile.

## Cross-references

- [`QUICKSTART.md`](../QUICKSTART.md) — boot the scenario in 5 minutes
- [`api-endpoints.md`](api-endpoints.md) — endpoint reference
- [`cli-commands.md`](cli-commands.md) — CLI command reference
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for env/port/lifecycle issues
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — why these surfaces exist
