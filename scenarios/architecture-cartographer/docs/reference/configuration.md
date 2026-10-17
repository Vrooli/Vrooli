# Configuration — Architecture Cartographer

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

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `architecture-cartographer` the prefix is the scenario id upper-cased with
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

### Architecture control surface (tunable levers)

These are **cartographer-global** levers — they tune cartographer's own
derivation and advisory heuristics. There is **no per-scenario
configuration**: cartographer derives any target scenario's domain map and
scores its boundaries with zero per-scenario setup. Every lever has a default
that reproduces the day-one behavior; an out-of-range or unparseable value is
clamped (or reverts to its default) with a logged diagnostic — a misconfigured
lever never fails startup. Resolved by `internal/config.Load`.

| Variable | Default | Range | Purpose |
|---|---|---|---|
| `CARTOGRAPHER_GOD_DOMAIN_FANOUT` | `0.6` | (0, 1] | Efferent fan-out fraction at/above which a non-exempt domain earns a `god_domain` coupling smell. The smell also requires at least two outgoing domain dependencies in a graph with at least three peer domains, so tiny two-domain graphs do not look like hubs. |
| `CARTOGRAPHER_INSTABILITY_WARN_BAND` | `0.7` | [0, 1] | Instability (I = Ce/(Ce+Ca)) at/above which a depended-upon domain earns an `unstable_dependency` smell. |
| `CARTOGRAPHER_AUTO_PLACE_MIN` | `0.85` | [0, 1] | Aggregator tier threshold: verdicts at/above this value are `auto_place`. Must be ≥ suggest-min (raised if not). |
| `CARTOGRAPHER_SUGGEST_MIN` | `0.55` | [0, 1] | Aggregator tier threshold: verdicts at/above this value (but below auto-place) are `suggest`. |
| `CARTOGRAPHER_TIE_DELTA` | `0.10` | [0, 1] | Minimum gap between the top and runner-up domain before a verdict is treated as a tie (→ `conflict`). |
| `CARTOGRAPHER_QUORUM_HIGH` | `0.45` | [0, 1] | Minimum signal participation confidence required before a high-direction verdict may become `auto_place`. Must be ≥ quorum-low (raised if not). |
| `CARTOGRAPHER_QUORUM_LOW` | `0.30` | [0, 1] | Minimum signal participation confidence required before a high-direction verdict may become `suggest`. |
| `CARTOGRAPHER_ARCHETYPE_EXEMPTIONS` | `composition-root,infrastructure` | CSV | Domain archetypes exempt from the `god_domain` smell (composition roots legitimately wire many domains). |
| `CARTOGRAPHER_NON_DOMAIN_FOLDERS` | (empty) | CSV | Extra `api/internal/` folder names treated as infrastructure (extends the built-in set: server, module, modules, database, testutil, middleware, clock, git, httpc, httpx). |
| `CARTOGRAPHER_LADDER_ORDER` | `domains_doc,api_folders,cli_groups` | CSV of source names | Domain-extraction trust order, highest first. Valid sources: `api_manifest` (reserved), `domains_doc`, `api_folders`, `cli_groups`. The UI rung is always advisory and appended regardless. |
| `CARTOGRAPHER_BANNED_VOCAB` | `common,helpers,manager,misc,stuff,utils` | CSV | Generic domain/package names the `naming` detector flags because they hide product intent. |
| `CARTOGRAPHER_LAYERING_STRICT` | `true` | boolean | When true, blocker-eligible `layering` violations (domain→transport and substrate→product imports) emit blocker severity; when false, they emit error severity for rollout. |
| `CARTOGRAPHER_VALIDATE_CONCURRENCY` | `1` | [1, CPU count] | Process-wide cap for `audit.run`, `audit.run-all` workers, and scenario-validation RPCs. Raise only when the host has headroom; queued requests respect client cancellation. |
| `CARTOGRAPHER_SIGNAL_WORKERS` | `min(4, CPU count)` | [1, 8] | Per-request worker cap for batched signal scoring. Higher values can reduce one validation's latency but multiply CPU use when several validations overlap. |
| `CARTOGRAPHER_PPROF_ENABLED` | `false` | boolean | Mounts dev-only stdlib pprof routes at `/debug/pprof/*` on the API listener. Keep disabled except during local profiling of high CPU or memory incidents. |

**Deliberately NOT exposed.** Cartographer intentionally has no lever for:

- **Declared allowed-dependencies / forbidden edges.** Architectural health
  is recovered from the import graph via cycle detection and the coupling
  heuristics above, not from a hand-maintained dependency allow-list. A
  specific acyclic forbidden dependency is therefore undetectable by design;
  this trade-off buys zero-maintenance, zero-config operation. (It may return
  later as an optional overlay.)
- **File or function size thresholds.** File length, symbol count, and similar
  tidiness signals belong to `tidiness-manager`, not to cartographer's
  structural architecture surface.
- **Per-scenario thresholds or signal weights.** These levers are global to
  the cartographer instance, never declared inside a target scenario.
- **Stable-kernel detection bounds.** The shared-substrate (stable-kernel)
  signature is derived, not tuned.

Durable, sanctioned deviations from the heuristics live as in-repo
`// arch:allow` markers next to the code they excuse — see
[`domains-contract.md`](domains-contract.md) and the suppression-marker
grammar — not as configuration here.

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
2. `${XDG_CONFIG_HOME}/vrooli/architecture-cartographer/config.json`
3. `~/.vrooli/config/architecture-cartographer/config.json`
4. `~/.config/vrooli/architecture-cartographer/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
architecture-cartographer configure api_base http://localhost:15001/api/v1
architecture-cartographer configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port architecture-cartographer API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
architecture-cartographer`").

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
