# Configuration — Tunnel Manager

How this scenario is configured — source-controlled defaults, durable
runtime state, operator secrets, lifecycle environment overrides, the
`.vrooli/service.json` manifest, and the per-user CLI config file.

The lifecycle (`vrooli scenario start`, `make start`) sets required
process variables automatically. Operators should configure Tunnel
Manager through the Settings UI or scenario CLI unless they are
deliberately supplying lifecycle/runtime overrides.

## Final configuration policy

Tunnel Manager uses four distinct configuration homes:

| Category | Storage | Examples | Operator surface |
|---|---|---|---|
| Source-controlled defaults | domain constants and non-secret scenario docs/manifests | first boot mode `local`, scheduler default intervals, fixed UI port policy | code review / manifest review |
| Product runtime state | SQLite domain tables | `tunnel_config`, routes, leases, metrics, probe history, recovery events | API/CLI/UI domain actions |
| Operator secrets | Vrooli credential authority (native secure store or encrypted fallback) | Cloudflare account id, tunnel id, API token | Settings UI, `config credentials-*` CLI |
| Environment overrides | lifecycle/operator process env | scheduler toggles, authz token | lifecycle only; never a credential source |

Secrets never live in repo `.vrooli`, SQLite, UI localStorage, logs, or
docs. Environment variables are supported for lifecycle and deployment
overrides, but never supply Cloudflare credentials.

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
| `SQLITE_PATH` | `${SCENARIO_DATA_DIR}/tunnel-manager.db` | Override SQLite file location. The default routes through `api-core/storage` and resolves to a writable per-scenario data directory. |
| `API_TOKEN` | unset | Shared bearer token for CLI ↔ API auth. Also used as the fallback privileged-mutation token when `TUNNEL_MANAGER_AUTHZ_ENFORCED=1` and `TUNNEL_MANAGER_OPERATOR_TOKEN` is unset. |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Tunnel Manager configuration

These settings drive the implemented `config` / `tunnel` domains.
Fixed-port values (`UI_PORT`, `API_PORT` range) are declared in
`.vrooli/service.json`.

| Variable / setting | Default | Purpose |
|---|---|---|
| `UI_PORT` | **21240 (fixed)** | Tunnel Manager's own UI port — fixed because it enforces the fixed-UI-port contract on others. Pinned in `service.json` `ports.ui`. |
| `API_PORT` | `15000-19999` (range) | Go API server port (lifecycle-allocated). |
| Tunnel mode | `local` on first boot | `remote` = programmatic ingress via Cloudflare API (hot-reload); `local` = generate/maintain `~/.cloudflared/config.yml` (restart on change). Persisted in the `tunnel_config` row; switch via `tunnel-manager config mode`. **Switching mode is pure — it never writes ingress; run `config sync` afterward to apply.** |
| Local config path | `~/.cloudflared/config.yml` | Source of ingress in local mode, generated from the manifest. |
| Prometheus endpoint | `${CLOUDFLARED_METRICS_URL}` from the managed resource | cloudflared metrics endpoint scraped by the `tunnel` domain (HA connections, request errors, RTT, active streams). Standalone fallback: `http://127.0.0.1:20241`. |
| `TUNNEL_MANAGER_EXPOSURE_RECONCILE_INTERVAL` | `5m` | Periodic exposure scheduler cadence. The scheduler runs once at boot, then on this interval, reconciling CORE routes and reaping expired leases. Go duration syntax (`30s`, `5m`, `1h`) is accepted. |
| `TUNNEL_MANAGER_EXPOSURE_SCHEDULER_DISABLED` | unset | Set to `1`, `true`, or `yes` to disable the background exposure scheduler for controlled maintenance/tests. Manual `tunnel-manager exposure reconcile` remains available. |
| `TUNNEL_MANAGER_PROBE_INTERVAL` | `1m` | Periodic probe scheduler cadence. The scheduler runs once at boot, then on this interval, probing enabled routes and persisting internal/external reachability results. |
| `TUNNEL_MANAGER_PROBE_SCHEDULER_DISABLED` | unset | Set to `1`, `true`, or `yes` to disable background probes. Manual `tunnel-manager probes run` remains available. |
| `TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED` | unset / disabled | Set to `1`, `true`, or `yes` to enable background recovery evaluation. Recovery acts through the Vrooli-managed cloudflared resource; manual `tunnel-manager recovery run` remains available. |
| `TUNNEL_MANAGER_RECOVERY_EVALUATE_INTERVAL` | `1m` | Recovery evaluation cadence when `TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED` is set. The scheduler runs once at boot, then on this interval, and delegates thresholds/backoff/circuit-breaker behavior to the recovery service. |
| `TUNNEL_READY_URL` | `${CLOUDFLARED_METRICS_URL}/ready` | cloudflared readiness endpoint checked by the recovery engine before and after restart attempts. |
| `TUNNEL_MANAGER_AUTHZ_ENFORCED` | unset / disabled | Set to `1`, `true`, or `yes` to require an operator token for privileged mutation RPCs. Default is local/operator-open for lifecycle-managed local use. |
| `TUNNEL_MANAGER_OPERATOR_TOKEN` | unset | Preferred privileged-mutation token when authz enforcement is enabled. Falls back to `API_TOKEN` if unset. |
| Default domain | `itsagitime.com` | Per-route `domain` is a **manifest field**, not a constant; this is the default. `public_url` = `https://<subdomain>.<domain>`. |
| Lease default TTL | ≈ 1 week | Default lifetime of a LEASED exposure; extendable/revocable, auto-reaped on expiry. |
| Core set | `packages/api-core/coreset` | SSOT for CORE-tier scenarios that are always exposed and never auto-expired. |

The `cloudflared` daemon is supervised by the Vrooli managed-service resource
and its endpoint is exported to Tunnel Manager. Tunnel Manager does not
install or own it; it remains the single authoritative owner of cloudflared
*restarts* via the `recovery` domain.

Cloudflare credentials resolve through the config-domain credential authority,
not ad hoc env reads, plaintext files, or legacy aliases. The authority
reports a stable identity/field reference and the active provider to Settings
and CLI status. Runtime environment variables are not a credential fallback;
the canonical setup path is the Settings UI or the scenario CLI backed by the
shared authority.

`tunnel-manager config get` returns a browser-safe readiness model:
current/desired mode, whether remote credentials are present,
canonical missing fields, credential source (`credential-authority` or
`missing`), a non-secret credential reference such as
`vrooli/tunnel-manager:cloudflare-api-token`, local config
path, sync readiness, and an operator-readable mode reason. The
Cloudflare API token value is never persisted in SQLite or returned to
UI/CLI clients.

Operators can provision the credential authority without editing environment
variables:

```bash
tunnel-manager config credentials-status
printf '%s' <token> | tunnel-manager config credentials-set --account-id <id> --tunnel-id <id> --api-token-stdin
tunnel-manager config credentials-clear --field api_token
```

The Settings UI exposes the same setup flow with source badges,
missing-field guidance, and the next recommended action. Token input is
write-only: after save, clients only receive
field presence, source, writability, and non-secret references.

`tunnel-manager config sync --dry-run true` is safe in all modes. In
remote mode without credentials it reports `setup_required` plus missing
canonical fields rather than attempting a Cloudflare API call. Applying
remote sync or switching to remote still requires the complete
Cloudflare credential set.

#### Additive reconcile, drift & ownership

`config sync` is **additive by default**: it publishes the desired
manifest (scenario + external routes) merged onto whatever ingress is
currently live, and never removes an entry it does not own. Pre-existing
ingress TM did not author survives every sync. Removal is always
explicit:

- `tunnel-manager config sync --prune true` removes only **orphaned**
  entries — hostnames TM previously created (recorded `MANAGED` in the
  `ingress_ownership` ledger) whose routes are now gone. It never removes
  unmanaged drift.
- `tunnel-manager drift prune <hostname>` removes one named hostname from
  live ingress and the ledger.

`tunnel-manager drift` (alias `drift list`) reconciles the desired
manifest, the live tunnel, and the ownership ledger into a classified
view. Each hostname lands in exactly one state: `managed` (scenario route,
live), `missing` (desired, not yet live), `external` (external route or
adopted-external, live), `orphaned` (we made it, route gone), `ignored`
(acknowledged external, never touched), or `unmanaged` (live drift needing
a decision). Per-entry decisions:

```bash
tunnel-manager drift list
tunnel-manager drift adopt api.itsagitime.com --scenario web-console   # → managed scenario route
tunnel-manager drift adopt api.itsagitime.com --target http://127.0.0.1:9000  # → external route
tunnel-manager drift ignore legacy.itsagitime.com --note "operator dashboard"  # → never push/prune
tunnel-manager drift prune stale.itsagitime.com                          # → remove this one entry
```

The ownership ledger (`ingress_ownership`) is keyed on the full hostname
and is the authoritative answer to "managed vs. external vs. ignored";
absence of a record means a live hostname is treated as `unmanaged`
drift (the safe default — surfaced, never silently removed).

#### External routes

Routes carry a `source` (`scenario` or `external`). External routes point
at an arbitrary local `service_target` and skip the scenario/fixed-UI-port
rule, so non-scenario services can be exposed through the same governed
plane:

```bash
tunnel-manager routes create --external --subdomain api --target http://127.0.0.1:9000 [--domain itsagitime.com]
```

External routes reconcile as `external` (EXTERNAL_OK) once live and are
full-CRUD across CLI and UI alongside scenario routes.

### DNS automation (CNAME on expose/revoke)

TM now manages the **DNS records** an exposed hostname needs to be publicly
resolvable, not just the tunnel ingress rule. When the credential token carries
`Zone:Read` + `Zone:DNS:Edit`, a remote-mode reconcile (expose, `config sync`,
`routes create`) ensures a proxied CNAME `<sub>.<apex> → <tunnel-id>.cfargotunnel.com`
for every managed hostname; revoke / `sync --prune` removes the CNAMEs TM
created. This reverses the previous non-goal — without it a freshly-exposed
hostname returned **NXDOMAIN** even though ingress was live.

Ownership is ledgered (`dns_ownership` table, mirroring the ingress ledger): TM
only ever deletes records **it created** — a CNAME set out-of-band is left
untouched (`EnsureRecord` never clobbers an existing record). DNS automation is
remote-mode only; local (`config.yml`) mode manages its own resolver. TM still
never touches Cloudflare **Access**.

Run `config credentials-status --verify` to confirm the token has the DNS scope
before relying on automation; a present-but-unscoped token surfaces as
`insufficient_scope` rather than producing a dead URL.

Privileged mutation RPCs are local/operator-open by default and can be
fail-closed with `TUNNEL_MANAGER_AUTHZ_ENFORCED=1`. When enabled, the
API requires `Authorization: Bearer <token>` or
`X-Vrooli-Operator-Token: <token>` for config sync/mode changes, drift
adopt/ignore/prune, route create/update/delete, exposure
expose/extend/revoke/reconcile, and manual recovery. `GetDrift` is a read
RPC and stays open. The token must match `TUNNEL_MANAGER_OPERATOR_TOKEN`,
falling back to `API_TOKEN`. Read RPCs remain available so operators
and monitors can inspect state.

The operator term "CLAP" is not a configuration key in this scenario. If
it appears in future notes, resolve it before wiring product behavior:
it may refer to Cloudflare API credentials, Cloudflare Access, or a
separate Vrooli credential provider.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `tunnel-manager` the prefix is the scenario id upper-cased with
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
in-process, so no resource is required. Tunnel Manager's only planned
optional resource is **`redis`** (UI pub/sub for real-time updates; falls
back to HTTP polling when absent); it is declared here only if enabled.

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
for the per-seam table including each domain's `<domain>.Schema` and
`database.SystemSchema`.

## CLI config file

The scenario CLI persists per-user configuration to a JSON file.
Resolution order (first match wins):

1. `${<PREFIX>_CONFIG_DIR}/config.json` (the scenario-prefixed env var; see "Scenario-prefixed CLI variables" above)
2. `${XDG_CONFIG_HOME}/vrooli/tunnel-manager/config.json`
3. `~/.vrooli/config/tunnel-manager/config.json`
4. `~/.config/vrooli/tunnel-manager/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
tunnel-manager configure api_base http://localhost:15001/api/v1
tunnel-manager configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port tunnel-manager API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
tunnel-manager`").

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
