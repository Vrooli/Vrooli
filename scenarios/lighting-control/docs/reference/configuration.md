# Configuration — Lighting Control

How this scenario is configured — env vars consumed by the binaries,
the `.vrooli/service.json` manifest, and the per-user CLI config file.

The lifecycle (`vrooli scenario start`, `make start`) sets every
required variable automatically. You only need this reference when
configuring lifecycle settings or supported deployment overrides.

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
(`path:docs/reference/port-allocation.md`) for the full policy.

### Optional overrides

| Variable | Default | Purpose |
|---|---|---|
| _(none)_ | — | The SQLite file location is **not** configurable through the environment. It is resolved from the scenario id by `api-core/storage`, so no inherited variable can point one scenario at another's database. Set `VROOLI_STORAGE_ROOT` to relocate the whole storage tree for a test run. |
| `API_TOKEN` | unset | Optional deployment bearer credential; ordinary lighting writes require a verified project-agent identity or an existing standing capability; privileged setup retains separate checks. |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `lighting-control` the prefix is the scenario id upper-cased with
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
| `components` | portable process builds, argv, environment, port ownership, readiness, peers, and storage |
| `lifecycle.health` | `/health` endpoint, startup grace period, periodic checks |
| `lifecycle.setup` | exceptional ordered setup work expressed as native argv steps |
| `lifecycle.develop` | exceptional ordered development work expressed as native argv steps |
| `lifecycle.stop` | how to shut down cleanly |
| `environment` | static env vars set for every lifecycle step |
| `dependencies.resources` | shared local resources (postgres, redis, qdrant, …) |

Testing is not a lifecycle phase. `.vrooli/testing.json` declares suites,
and `vrooli scenario test <name>` delegates the run to Test Genie.

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
2. `${XDG_CONFIG_HOME}/vrooli/lighting-control/config.json`
3. `~/.vrooli/config/lighting-control/config.json`
4. `~/.config/vrooli/lighting-control/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
lighting-control configure api_base http://localhost:15001/api/v1
lighting-control configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port lighting-control API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
lighting-control`").

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
| `ui` | `react_vite_ui` | Vitest through pnpm, jsdom setup, V8 coverage, 85% coverage target with a committed floor gated by Unit Health (no native Vitest thresholds), `ui/src/test-utils/renderWithProviders.tsx`. |

Scenario customizations are monotonic: they may add surfaces, add stricter
checks, or raise coverage targets. They may not weaken the template baseline unless
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

## Browser sign-in (shipped)

Light Desk uses the installation's existing `scenario-authenticator` realm.
Configure that provider explicitly through its supported setup workflow; Light
Desk never starts it or creates accounts. The scenario manifest declares the
`lighting-control` resource with audience `scenario-authenticator:lighting-control`. The
account must already have `lighting-control:write` for lighting changes;
signing in proves identity and does not grant capabilities.

Open Settings → Account and enter the installation account's email and
password. If the issuer requests MFA, enter the authenticator or recovery code
before a session can be established. Status is derived from the API's verified,
unexpired principal, not from the sign-in response. Access-token expiry returns
the account panel to signed out; Renew session rotates the existing refresh
cookie through the issuer. Sign out calls the issuer's canonical Logout operation
and expires both local cookies. That operation currently revokes all issuer
sessions for the account when the access token is valid; an already expired or
invalid access token only clears this browser's cookies. Provider outage is
reported explicitly. Sign out still clears this browser’s cookies and reports
that server revocation was not confirmed; retry revocation through the identity
provider when it is available.

The browser calls only its same-origin `/api/session` status and
`/api/session/login`, `/api/session/refresh`, `/api/session/logout` endpoints.
The server uses typed AccountsService calls through shared discovery. Tokens
stay in canonical `vrooli_access_token` and `vrooli_refresh_token` HttpOnly,
SameSite=Lax cookies; browser responses never include token values. Use HTTPS
for remote installations and set the provider's `VROOLI_AUTH_COOKIE_SECURE=true`
when TLS terminates at a proxy. The API preserves issuer Secure cookie flags
and enforces Secure on direct TLS or a validated same-origin HTTPS browser
Origin through the loopback UI proxy. Proxy deployments must preserve the original
browser authority through the shared UI proxy, which replaces
`X-Forwarded-Host` from the incoming Host; only a loopback peer's forwarded
host is accepted for the session Origin check. Other reverse-proxy topologies
need an explicitly trusted ingress configuration and are not implied qualified.
Cross-site session mutations fail before calling the provider.

Account bootstrap, password recovery, capability changes and realm administration
remain the authenticator's responsibility; this UI exposes none of those
management operations. No desktop personal-local mode, external account linking,
or offline identity fallback is implemented here.

## Lighting authorization

Ordinary lighting control, enrollment, colors, Off and presets do **not** require agent approval, an operation grant or lighting capability scopes. They accept the current verified, unexpired Agent Manager project-agent identity through the canonical request middleware. A claimed source label, anonymous request, invalid token, expired identity or foreign issuer cannot establish that identity. Identity and runtime binding verification remain in the shared middleware; no borrowed credential or direct SDK route is used.

Human and service callers retain the existing verified `lighting-control:write` capability policy. Public metadata reads remain available for inspecting current controllers and desired state. Host setup, installation, enable/disable and device-permission changes retain their existing capability checks and native OS permissions; ordinary lighting access does not broaden those paths.

Scenario Authenticator tokens use audience `scenario-authenticator:lighting-control`. Signing in proves identity and does not grant human or service capabilities. The CLI uses its existing ordinary Agent Manager identity transport; no lighting-specific consent or operation proof is needed.
