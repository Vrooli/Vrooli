# Configuration — Scenario Authenticator

> **Current configuration reference.** `API_PORT`, `UI_PORT`, `SQLITE_PATH`,
> `REDIS_URL`, and the persisted signing-key settings are wired through the
> lifecycle and API composition. Values marked planned or deferred below are
> intentionally not runtime knobs yet (for example managed DB, automated key
> rotation, and federation). **Secrets are always referenced by name, never
> inlined — only hashes and signed material are ever stored at rest.**

How this scenario is configured — env vars consumed by the binaries, the
`.vrooli/service.json` manifest, and the per-user CLI config file.

The lifecycle (`vrooli scenario start`, `make start`) sets every required
variable automatically. You only need this reference when running a
binary by hand or when a scenario adds a new variable.

## Environment variables

### Required at runtime (set by the lifecycle)

| Variable | Range / format | Purpose |
|---|---|---|
| `API_PORT` | `15000-19999` | Port for the Go API server |
| `UI_PORT` | `20000-24999` | Port for the production UI server (`ui/server.js`) |
| `REDIS_URL` | `redis://host:port[/db]` | **Required.** Redis backs sessions, token-family revocation, OAuth CSRF state, and cross-replica rate limiting. Treat it as a hard dependency — session-revocation correctness and distributed rate-limit accuracy depend on it (PRD Operational Risks). |

If the scenario adds WebSocket channels on the existing API or UI server,
do not add another `ports` entry. Declare an additional port only when
the scenario starts a separate listener process.

The canonical bands all sit below 32768 so Linux never hands out the
ports as outbound source ports. See the project-level
[port-allocation reference](../../../../docs/reference/port-allocation.md)
for the full policy.

### Storage seam (SQLite default → managed DB at scale)

All persistence routes through the `api-core/storage` seam — **never
shared Postgres** (removing the shared-database blast radius is a core
reason for the rewrite). SQLite is the local and default substrate; the
seam keeps the database swappable to a managed server DB for cloud scale
without touching domain code. The hot path (token verification) is
stateless and never touches SQLite.

| Variable | Default | Purpose |
|---|---|---|
| `SQLITE_PATH` | `${SCENARIO_DATA_DIR}/scenario-authenticator.db` | SQLite file location. The default routes through `api-core/storage` and resolves to a writable per-scenario data directory. |
| `DATABASE_DSN` | unset | *(planned, P2)* Managed-DB DSN for cloud/HA deployments. When set, the storage seam backs onto the managed DB instead of SQLite (OT-P2-006). Referenced by secret name in production, never inlined. |

SQLite is single-writer — that is the authenticator's own write-throughput
ceiling, mitigated because verification is stateless and the seam allows
a managed DB when throughput demands it. Schema changes are always
**additive migrations, never database recreation**.

### Signing key (RS256 — load-or-generate, persisted)

The RS256 signing keypair is the crux of the token contract. It is
**persisted to the storage root** and loaded-or-generated on boot (the
carried-over pattern): if the key files exist they are loaded; otherwise
a keypair is generated and written once. This must be deliberate — a key
that regenerates on every restart invalidates every live token (PRD
Operational Risks).

| Variable | Default | Purpose |
|---|---|---|
| `SIGNING_KEY_DIR` | `${SCENARIO_DATA_DIR}/keys` | Directory in the storage root holding `private.pem` / `public.pem`. Generated on first boot if absent. The public half is what `/.well-known/jwks.json` publishes; the private half never leaves the process. |
| `JWT_KID` | derived from the key | *(planned)* Key id surfaced in JWKS + token header; enables overlapping `kid`s during rotation (OT-P2-005). |

Only signed material (the keypair) and credential **hashes** live at
rest — never plaintext secrets. The algorithm is locked to exactly RS256;
`none` and HS-family confusion are rejected at the verification layer.

### Token TTLs and default-realm settings

| Variable | Default (planned) | Purpose |
|---|---|---|
| `ACCESS_TOKEN_TTL` | `15m` | Lifetime of issued RS256 access tokens (short-lived; paired with rotating refresh). |
| `REFRESH_TOKEN_TTL` | `720h` (30d) | Lifetime of a refresh-token family before re-auth is required. |
| `DEFAULT_REALM` | `default` | Slug of the single default realm that exists from day one and issues `aud`-scoped tokens (OT-P0-008). |
| `DEFAULT_REALM_DISPLAY_NAME` | `Default` | Human label for the default realm. |
| `TOKEN_ISSUER` | `scenario-authenticator` | The locked `iss` claim — do not change; the carried-over claims contract depends on it. |

These are bootstrap defaults for the default realm. Once realms become a
true tenant boundary (P1), per-realm overrides are stored **on the realm
record** (below), not in environment variables.

### Per-realm policy knobs (stored on the realm, P1)

At P1, realm CRUD moves token TTLs, password rules, redirect URIs,
enabled methods, and branding out of environment defaults and onto the
**realm record** in SQLite, so each tenant configures independently. The
environment values above seed the default realm; the realm record is the
runtime source of truth thereafter.

| Realm field | Owns |
|---|---|
| `policy.password_rules` | Min length, complexity, history (faithful validation reasons relayed to the client). |
| `policy.lockout` | Failed-attempt thresholds + lockout window for brute-force defense (OT-P0-006). |
| `policy.enabled_methods` | Which credentials/methods are allowed (password, TOTP, passkey, social) and whether MFA is required. |
| `policy.token_ttls` | Per-realm access/refresh TTLs (override the env defaults). |
| `redirect_uris` | Allow-listed OAuth/OIDC redirect targets; a non-listed `redirect_uri` is rejected at `StartOAuth`. |
| `branding` | Logo + color overrides rendered on hosted login/consent at P1. |

### Rate-limit / lockout thresholds

Rate limiting is Redis-authoritative so the protected budget is consistent
across replicas and fails closed when Redis is unavailable. Defaults are
conservative; per-realm lockout policy (above) refines them.

| Variable | Default (planned) | Purpose |
|---|---|---|
| `RATE_LIMIT_AUTH_PER_MINUTE` | `10` | Per-IP auth-request ceiling before throttling. |
| `LOCKOUT_MAX_ATTEMPTS` | `5` | Failed sign-ins before account lockout. |
| `LOCKOUT_WINDOW` | `15m` | Lockout duration. |

### Argon2id cost parameters

Passwords are hashed with **Argon2id** at a documented cost (carried-over
invariant — only hashes are stored, never plaintext). The cost is
configurable so it can be tuned to host hardware without a code change.

| Variable | Default (planned) | Purpose |
|---|---|---|
| `ARGON2_MEMORY_KIB` | `65536` (64 MiB) | Memory cost. |
| `ARGON2_ITERATIONS` | `3` | Time cost (passes). |
| `ARGON2_PARALLELISM` | `2` | Lanes / threads. |
| `ARGON2_SALT_LENGTH` | `16` | Salt bytes. |
| `ARGON2_KEY_LENGTH` | `32` | Derived hash bytes. |

### OAuth provider credentials (referenced by secret name)

Social federation (P1) needs an upstream client id + secret per provider.
**Secrets are referenced by name, never written into config or
committed.** The lifecycle injects them from the secret store; the table
below names the variables the federation domain reads.

| Variable | Purpose |
|---|---|
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | Google OAuth2/OIDC client credentials. |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | GitHub OAuth credentials. |
| `MICROSOFT_CLIENT_ID` / `MICROSOFT_CLIENT_SECRET` | Microsoft OAuth2/OIDC credentials. |
| `OAUTH_CALLBACK_BASE_URL` | Public base URL the provider redirects back to (`…/api/v1/auth/{provider}/callback`). |

A provider with no configured credentials is simply absent from
`ListProviders` — the scenario boots cleanly without any social provider.
Client *secrets* are held in the secret store and never logged, echoed,
or persisted to SQLite; only the linked-identity records (no secret) and
short-lived CSRF state (Redis, TTL-bounded) are stored.

### Optional overrides

| Variable | Default | Purpose |
|---|---|---|
| `API_TOKEN` | unset | Shared bearer token for CLI ↔ API auth (only enforce in production deployments). |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |

The browser UI does not read `API_PORT` directly. It resolves API calls
through the UI origin (same-origin only — never cross-origin to the
authenticator), and `ui/server.js` proxies `/api/*` plus the scenario's
Connect RPC namespace to the API process using the lifecycle-provided
`API_PORT`.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name. For
`scenario-authenticator` the prefix is the scenario id upper-cased with
hyphens replaced by underscores (so `my-scenario` → `MY_SCENARIO`). The
following are recognised, in precedence order (first-found wins);
substitute your scenario's prefix for `<PREFIX>`:

| Purpose | Variables |
|---|---|
| API base URL | `<PREFIX>_API_BASE`, `<PREFIX>_API_URL`, `VROOLI_API_BASE` |
| API port | `<PREFIX>_API_PORT` |
| API token | `<PREFIX>_API_TOKEN`, `VROOLI_API_TOKEN` |
| Config dir | `<PREFIX>_CONFIG_DIR`, `VROOLI_CLI_CONFIG_DIR` |
| HTTP timeout | `<PREFIX>_HTTP_TIMEOUT`, `VROOLI_HTTP_TIMEOUT` |

> **Do not** set the un-prefixed `API_PORT` for a CLI invocation — when
> CLIs run inside web-console terminals it leaks across scenarios. Use the
> scenario-prefixed form or the `--api-base` flag.

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

Unlike most scaffolded scenarios, scenario-authenticator declares
**`redis`** under `dependencies.resources` (it is a required dependency —
see `REDIS_URL` above). SQLite is in-process via the storage seam, so no
database resource is declared.

## Schema bootstrap

Schema is owned per-domain. `api/internal/<dom>/schema.sql` declares each
domain's tables and is embedded into the binary via `go:embed` from
`api/internal/<dom>/schema.go::Schema()`. Cross-cutting infrastructure
(custom types, cross-domain views) lives in
`api/internal/database/system.sql` — empty by default in SQLite scenarios.

The shared registry at `api/internal/modules/registry.go::AllSchemas()`
collects them in order (system first, then domains alphabetical), and
`apidb.EnsureSchemas(ctx, db, modules.AllSchemas()...)` from
`api-core/database` applies them at startup. The path is idempotent — all
DDL uses `CREATE TABLE IF NOT EXISTS` / `ALTER TABLE … ADD COLUMN IF NOT
EXISTS`, so re-runs on every boot are no-ops.

Adding a column lands in the same diff as the Go struct field, the
repository scan, and the proto wire shape — single location, single edit.
Drops/renames in production identity data need the brownfield
versioned-migration helpers (`Migrate` / `MigrationProvider` in
`api-core/database`); for an auth store, schema changes are **always
additive migrations, never database recreation** — user/credential rows
are never disposable.

See [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#domain-owned-schema)
for the design rationale and [`../internal/SEAMS.md`](../internal/SEAMS.md)
for the per-seam table including each domain's `<domain>.Schema` and
`database.SystemSchema`.

## CLI config file

The scenario CLI persists per-user configuration to a JSON file.
Resolution order (first match wins):

1. `${<PREFIX>_CONFIG_DIR}/config.json` (the scenario-prefixed env var; see "Scenario-prefixed CLI variables" above)
2. `${XDG_CONFIG_HOME}/vrooli/scenario-authenticator/config.json`
3. `~/.vrooli/config/scenario-authenticator/config.json`
4. `~/.config/vrooli/scenario-authenticator/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
scenario-authenticator configure api_base http://localhost:15001/api/v1
scenario-authenticator configure token <token>
```

The `token` here is a CLI ↔ API bearer for operator convenience — it is
not a long-lived user credential. Never put signing keys, OAuth secrets,
or passwords in this file; those are referenced from the secret store.

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order (first
match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port scenario-authenticator API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
scenario-authenticator`").

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

- [`api-endpoints.md`](api-endpoints.md) — endpoint reference (signing key, JWKS, realm policy in action)
- [`cli-commands.md`](cli-commands.md) — CLI command reference
- [`ui-manifest.md`](ui-manifest.md) — UI surface + manifest contract
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — which domain owns each config knob
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — RS256/JWKS/Argon2id posture and secret handling
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — why these surfaces exist
