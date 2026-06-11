# Configuration — Search Hub

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
| `SQLITE_PATH` | `${SCENARIO_DATA_DIR}/search-hub.db` | Override SQLite file location. The default routes through `api-core/storage` and resolves to a writable per-scenario data directory. |
| `API_TOKEN` | unset | Shared bearer token for CLI ↔ API auth (only enforce in production deployments). |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |
| `SEARCH_HUB_QUERY_TIMEOUT` | `25s` | Total router budget for listing providers, fan-out, optional external escalation, and reranking. Supported range: `1s`-`29s`. Keep this below the CLI HTTP timeout so degraded responses can return before clients give up. |
| `SEARCH_HUB_RERANK_TIMEOUT` | `5s` | Maximum reranker call duration. Supported range: `100ms`-`20s`; the router also clips it to the remaining query budget minus a response cushion. |
| `SEARCH_HUB_RERANK_BREAKER_FAILURES` | `3` | Consecutive reranker failures/timeouts before the router opens the reranker circuit and skips rerank. Supported range: `1`-`20`. |
| `SEARCH_HUB_RERANK_BREAKER_COOLDOWN` | `60s` | How long an open reranker circuit stays open before one half-open probe is allowed. Supported range: `1s`-`10m`. |
| `SEARCH_HUB_AUTO_ROUTE_EXTERNAL` | `false` | Allow automatic/classifier routing to reach external providers when a query is confidently web-shaped, and allow fallback escalation when project results are weak. Truthy values: `1`, `true`, `yes`, `on`. |

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Reranker degradation behavior

Unified reranking remains enabled when the reranker is healthy and there are at
least two candidates. The router skips rerank for zero or one candidate because
there is no ordering value to add.

When the reranker times out, exits, or returns invalid output, Search Hub keeps
the by-provider groups, marks the query response degraded, and adds a
`routing_explanation` line when `--explain` is used. After repeated reranker
failures, the circuit breaker opens and later allows a single half-open probe;
a successful probe closes the circuit.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `search-hub` the prefix is the scenario id upper-cased with
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
2. `${XDG_CONFIG_HOME}/vrooli/search-hub/config.json`
3. `~/.vrooli/config/search-hub/config.json`
4. `~/.config/vrooli/search-hub/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
search-hub configure api_base http://localhost:15001/api/v1
search-hub configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port search-hub API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
search-hub`").

## Test/CI configuration

| File | Owns |
|---|---|
| `.vrooli/testing.json` | Test categories — lint, unit, business checks (endpoints, CLI commands), Lighthouse, bundle size |
| `.vrooli/lighthouse.json` | Lighthouse pages, thresholds, Chrome flags |
| `.vrooli/endpoints.json` | API endpoint manifest (path, method, status codes, request/response shapes, CLI mapping) |
| `.github/workflows/test.yml` | CI gate — UI lint + test, Go vet + race + coverage, E2E binary smoke |

These files are read by tooling (`vrooli scenario test`, `test-genie`,
the doc viewer) — keep them in sync with the code they describe.

## Search tuning control surface

Beyond routing and eval *tracking*, search-hub owns the **search-tuning
authority**: it sweeps a provider's tuning factors against that provider's golden
corpus and writes the winner back, and it grows + grades the corpus itself. This
is the operator recipe; the factor schema (`tuning` block of `.vrooli/search.json`)
and the per-knob dashboard live in the engine package
[`packages/ai-go/search/docs/reference/search-json.md`](../../../../packages/ai-go/search/docs/reference/search-json.md).

### The closed loop

```
search.json (scenario SSOT: descriptor + tuning + tests)
   │  self-register at boot ─────────────────────────────────► search-hub registry
   │                                                              (mints + stores a control TOKEN)
   ▼
evals generate ──► grow the golden corpus (query inversion + hard negatives, de-duped, marked "generated")
   ▼
evals sweep ──► run the suite under candidate tunings, clear the overfit guards, pick a winner
   ▼  --apply (token-gated WriteConfig RPC)
search.json `tuning` rewritten by the scenario  (+ reindex iff an index-time factor moved)
```

### `evals sweep` — the two-tier optimizer

```bash
# Preview a ranked arm table + recommendation (no write):
search-hub evals sweep <suite_id>
# Cheap path only (no reindex), and write the winner back:
search-hub evals sweep <suite_id> --query-time-only --apply
```

The provider is derived from the suite (e.g. `cli-health.commands.primary` →
`cli-health.commands`); there is no separate `--provider` argument.

| Flag | Effect |
|---|---|
| `<suite_id>` (positional) | the golden corpus to optimize against (the provider's `tests`, registered as a suite). |
| `--query-time-only` | skip the expensive index-time tier (no reindex-per-arm). |
| `--apply` | gate the **write-back**; default previews the ranked table + recommendation and changes nothing. |
| `--limit <n>` | per-case fetch depth (0 = a sensible default). |

- **Incumbent** — the provider's current tuning is always evaluated as the baseline.
- **Query-time tier** — `rerank_enabled` × `rerank_blend` (and other query-time
  factors) explored **full-factorial** via per-request overrides. No reindex; cheap.
- **Index-time tier** — `engine`, `embed_model`, `embed_task_prefix` explored by
  **coordinate-ascent** (one factor at a time) to bound reindex cost: each value
  is `config-push → reindex → poll terminal → run suite`. Interactions across
  index-time factors are *not* explored — that limitation is reported in the
  result note, never silently capped. Skipped entirely under `--query-time-only`.
- Every arm is one stored, immutable, tagged `EvalRun` with a complete
  `ConfigSnapshot` (engine, embed recipe, rerank policy, floor regime) so it is
  self-describing and reproducible.

### The four overfit guards (all mandatory; a winner clears every one)

The sweep's contract is *"find a **robust** improvement or report none."* It will
**refuse to promote a within-noise win**:

1. **Significance** — a paired bootstrap CI of the per-case recall margin
   (winner − incumbent) must exclude 0.
2. **Held-out validation** — the winner, selected on the tuning fold, must hold on
   an independent held-out fold. Machine-`generated` cases are *always* held out
   of the tuning fold (so a tuning is never selected on cases a machine wrote).
3. **Multi-objective constraints** — recall is maximized **subject to** a
   gibberish/negatives ceiling and a p95-latency budget; an arm that leaks junk or
   blows the budget is infeasible regardless of recall.
4. **Complexity / incumbent tie-break** — among significant, feasible arms, prefer
   the simpler config (dense over hybrid, rerank-off over on) and the incumbent;
   switch only past the noise band guard #1 defines.

On a cleared winner with `--apply`, the sweep calls the provider's token-gated
`SearchControlService.WriteConfig` to persist the new `tuning` into the scenario's
own `search.json`; otherwise it reports "no significant improvement" and changes
nothing.

### `evals generate` — grow + grade the corpus

Optimizing against a hand-curated dozen-query corpus is trivially overfittable, so
search-hub can grow the corpus from the provider's own index:

```bash
# Preview proposals (no write):
search-hub evals generate <suite_id> --count 20 --negatives
# Append the proposals to the provider's tests (human/agent reviews before merge):
search-hub evals generate <suite_id> --count 20 --negatives --apply
```

It stratified-samples the index, inverts each sampled item to a natural-language
query that should retrieve it (a positive case), optionally proposes hard
negatives, and de-dupes against the existing corpus and each other. Every proposal
is marked `tags:["generated", <stratum>]` — the marker the sweep holds out (guard
#2). **Generation augments the curated golden core; it never replaces it.**

**Adequacy warnings** (warn-level, *never* gating — a malformed suite is rejected
by `Validate`, but a thin one is only flagged) surface on `evals show`, `evals
run`, and `evals generate`:

| Code | Fires when |
|---|---|
| `too_few_cases` | fewer than 12 positive cases (too small for a trustworthy held-out split). |
| `no_negatives` | no junk-rejection cases — the gibberish constraint can't be checked. |
| `thin_difficulty` | every case is one difficulty band (over-reports recall). |
| `duplicate_query` | two cases share a query. |
| `coverage_gap` | a live-index stratum (type/group/origin bucket) has no case (only when a live sample is supplied). |

### Two documented limitations (not bugs — see [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md))

- **The sampler enumerates only what its probes surface.** The unified search
  contract has no list-all RPC, so `evals generate`'s sampler discovers items via
  probe queries and reports the true count (no silent cap).
- **De-dup is lexical (token-Jaccard), not embedding-semantic.** search-hub holds
  no embedder of its own; `JaccardDeduper` is a stand-in behind the `Deduper` seam
  and a cosine-over-embeddings deduper drops in unchanged when an embedder reaches
  search-hub.

### Security — the control token

Sweep + reindex + config-write run over a **token-gated** control plane
(`search-hub.v1.control.SearchControlService`). search-hub mints an opaque control
token at the provider's first `RegisterProvider`, stores it server-side, and
presents it on every reindex / config-write / per-request override. Public search
(no overrides) needs no token. A provider that declares neither `reindex_endpoint`
nor `config_endpoint` is routable but not sweep-tunable.

## Cross-references

- [`QUICKSTART.md`](../QUICKSTART.md) — boot the scenario in 5 minutes
- [`api-endpoints.md`](api-endpoints.md) — endpoint reference
- [`cli-commands.md`](cli-commands.md) — CLI command reference
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for env/port/lifecycle issues
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — why these surfaces exist
