# Configuration — Web Search

How this scenario is configured — env vars consumed by the binaries,
the `.vrooli/service.json` manifest, and the per-user CLI config file.

The lifecycle (`vrooli scenario start`, `make start`) sets every
required variable automatically. You only need this reference when
running a binary by hand or when a scenario adds a new variable.

> **Status (2026-06-10):** all domains are implemented; the resource
> wiring below is live (`SEARXNG_URL`, `OLLAMA_URL`, `BROWSERLESS_URL`,
> and the aisearch wiring are read in `api/main.go`), as are the
> platform-level variables (`API_PORT`, `UI_PORT`, `SQLITE_PATH`) and
> the tuning levers.

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
| `SQLITE_PATH` | `${SCENARIO_DATA_DIR}/web-search.db` | Override SQLite file location. The default routes through `api-core/storage` and resolves to a writable per-scenario data directory. |
| `API_TOKEN` | unset | Shared bearer token for CLI ↔ API auth (only enforce in production deployments). |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Resource wiring

web-search talks to several local resources and to search-hub. The
lifecycle injects each resource's endpoint from its
`.vrooli/service.json` dependency declaration:

| Variable | Used by | Default / source | Purpose |
|---|---|---|---|
| `SEARXNG_URL` | livesearch (L0/L1), research (L2 source) | local SearXNG resource endpoint | Live web search JSON API. If unreachable, `web-search.live` degrades to unavailable; learnings unaffected. |
| `QDRANT_URL` | findings | local Qdrant resource endpoint | Semantic index (collection `web-search-findings`) via aisearch-go. If unreachable, recall falls back to text matching. |
| `OLLAMA_URL` | findings (embeddings), livesearch/research (synthesis, distillation) | local Ollama resource endpoint | `nomic-embed-text` embeddings + small chat model. If unreachable, no embeddings/synthesis; raw hits still returned. |
| `RERANKER_URL` | findings (ranking) | local reranker resource endpoint | TEI cross-encoder (bge-reranker-v2-m3). Falls back to raw dense order. |
| browser-automation-studio discovery | research (L2 browser escalation) | scenario discovery (`discovery.ResolveScenarioURLDefault`) | Browser leg of the L2 fetch stack: `CaptureService.Capture` (Connect-RPC, `inline_dom=true`) for pages the HTTP leg cannot read. If unreachable, L2 degrades to HTTP-only fetch; L0/L1 unaffected. |
| search-hub registration target | federation | lifecycle / search-hub discovery | Self-registers `web-search.live` + `web-search.learnings` from `.vrooli/search.json`; control token gates mutating verbs (reindex/config/query-overrides). |

### Tuning levers (boot-time)

The scenario's control surface: six env levers parsed once at startup
by `api/tuning.go`. An unset, malformed, or out-of-range value falls
back to the compiled default (boot is never blocked by a bad knob);
the defaults remain the SSOT in their owning packages. Changes require
a restart. Rationale for the surface (and for the knobs deliberately
NOT exposed, like the 1-minute governor window) is recorded in
[`DECISIONS.md`](../internal/DECISIONS.md).

| Variable | Default | Range | Effect when raised |
|---|---|---|---|
| `WEB_SEARCH_HIGH_CONFIDENCE_THRESHOLD` | `0.75` (`research.HighConfidenceThreshold`) | `(0, 1]` | L3 reconcile becomes more conservative: more contradictions are FLAGGED as disputes instead of SUPERSEDING existing findings. |
| `WEB_SEARCH_DECAY_HALF_LIFE` | `4320h` = 180d (`findings.DecayHalfLife`) | Go duration > 0 | Findings stay trusted longer; the GC supersede min-age (2× half-life) stretches with it. |
| `WEB_SEARCH_MAX_GATHER_FINDINGS` | `20` (`research.MaxGatherFindings`) | int ≥ 1 | The bounded GATHER sweep reads more nearby findings per L3 cycle — more reconcile context, more read cost. |
| `WEB_SEARCH_L3_MAX_LOOPS` | `10` (`research.DefaultMaxResearchLoops`) | int ≥ 1 | The L3 task contract allows more search→read→gap→re-search loops before the agent must converge — deeper research, longer/costlier runs. Prompt-level budget; the hard run timeout is agent-manager's. |
| `WEB_SEARCH_GOVERNOR_CAPACITY` | `60` (`livesearch.DefaultGovernorCapacity`) | int ≥ 1 | More live SearXNG calls allowed per rolling minute before the service degrades to "rate-limited". |
| `WEB_SEARCH_CACHE_TTL` | `5m` (`livesearch.DefaultCacheTTL`) | Go duration > 0 | Identical live queries are served from cache longer — fresher-is-better vs. budget-is-scarce tradeoff. |
| `WEB_SEARCH_GC_INTERVAL` | unset = background GC off | Go duration > 0 | Enables the periodic store-consistency GC loop at this cadence (`findings gc` CLI works either way). |
| `WEB_SEARCH_SYNC_INTERVAL` | aisearch-go default (5m) | Go duration > 0 | Periodic findings-index reconcile cadence. Since the capture-kick landed, every successful findings write also kicks an immediate (2s-debounced) reconcile, so this interval is purely the **repair** cadence for drift (e.g. qdrant restored from backup) — do not shorten it to chase freshness. |
| `WEB_SEARCH_FETCH_TIMEOUT` | `15s` (`fetch.DefaultHTTPTimeout`) | Go duration > 0 | One HTTP-leg L2 page fetch may take longer before being abandoned — helps slow origins, slows the whole L2 pass. |
| `WEB_SEARCH_FETCH_MAX_BYTES` | `2097152` = 2 MiB (`fetch.DefaultMaxBodyBytes`) | int ≥ 1 | More of each fetched page body is read before extraction — longer articles survive intact, more memory/IO per page. |
| `WEB_SEARCH_BROWSER_ESCALATION` | on | `off`/`false`/`0`/`no`/`disabled` to disable | Lever is the OFF switch: when disabled, the L2 fetch stack is HTTP-only and JS-shell pages contribute thin or no text (no browser-automation-studio call is ever made). |
| `WEB_SEARCH_MIN_READABLE_CHARS` | `200` (`fetch.DefaultMinReadableChars`) | int ≥ 1 | Raises the JS-shell heuristic: HTTP-leg results with fewer extracted characters escalate to the browser leg — more escalations, better coverage of borderline pages, more 2–10s browser fetches. |
| `WEB_SEARCH_SYNTH_EXCERPT_CHARS` | `6000` (`research.DefaultExcerptChars`) | int ≥ 1 | More of each fetched page reaches the synthesis model per document — richer grounding, larger prompts (watch the model's context window). |
| `WEB_SEARCH_SYNTH_RELEVANT_EXCERPTS` | on | `off`/`false`/`0`/`no`/`disabled` to disable | Lever is the OFF switch: when disabled, L2 reverts to positional first-N-chars truncation instead of relevance-selected (chunk+embed) excerpts. Relevance mode self-degrades to positional when the embedder is unreachable, so OFF is only for debugging/measurement (the `TestL2AnswerQualityEval` harness uses it for the A/B baseline). |

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `web-search` the prefix is the scenario id upper-cased with
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

Unlike the bare template (`dependencies.resources: {}`), web-search
declares real resources here: `searxng`, `ollama`, `qdrant`, and
`reranker` (enabled, `try_start`). Scenario dependencies
(`browser-automation-studio` for the L2 browser-escalation leg,
`search-hub`, and `agent-manager`) are declared alongside. SQLite remains in-process. See
[`INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) for the full dependency
contract and failure behavior.

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
2. `${XDG_CONFIG_HOME}/vrooli/web-search/config.json`
3. `~/.vrooli/config/web-search/config.json`
4. `~/.config/vrooli/web-search/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
web-search configure api_base http://localhost:15001/api/v1
web-search configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port web-search API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
web-search`").

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
