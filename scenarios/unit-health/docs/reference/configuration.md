# Configuration — Unit Health

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
| `SQLITE_PATH` | `${SCENARIO_DATA_DIR}/unit-health.db` | Override SQLite file location. The default routes through `api-core/storage` and resolves to a writable per-scenario data directory. |
| `API_TOKEN` | unset | Shared bearer token for CLI ↔ API auth (only enforce in production deployments). |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `unit-health` the prefix is the scenario id upper-cased with
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
2. `${XDG_CONFIG_HOME}/vrooli/unit-health/config.json`
3. `~/.vrooli/config/unit-health/config.json`
4. `~/.config/vrooli/unit-health/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
unit-health configure api_base http://localhost:15001/api/v1
unit-health configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port unit-health API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
unit-health`").

## Test/CI configuration

| File | Owns |
|---|---|
| `.vrooli/testing.json` | Test categories — lint, unit, business checks (endpoints, CLI commands), Lighthouse, bundle size |
| `.vrooli/lighthouse.json` | Lighthouse pages, thresholds, Chrome flags |
| `.vrooli/endpoints.json` | API endpoint manifest (path, method, status codes, request/response shapes, CLI mapping) |
| `.github/workflows/test.yml` | CI gate — UI lint + test, Go vet + race + coverage, E2E binary smoke |

These files are read by tooling (`vrooli scenario test`, `test-genie`,
the doc viewer) — keep them in sync with the code they describe.

## Unit testing policy profile

Unit Health's policy-profile contract is a joined contract, not a
surface inventory. Three owners participate:

| Owner | Source | Responsibility |
|---|---|---|
| Unit Health | built-in resolver defaults | Canonical language/framework defaults, default coverage floors, unsupported-language behavior, and stable finding codes. |
| Template | `.vrooli/testing.json` `unit.policy_profile` | Template identity, required roles, policy classes, minimum thresholds, projection expectations, customization rules, and waiver metadata. |
| Code Facts | `DescribeCodeFacts` surfaces and parse units | Observed API, CLI, UI, and additional code surfaces present in the target scenario. |

`.vrooli/testing.json` must not enumerate the actual surfaces a scenario has.
Observed surfaces come from Code Facts, with Unit Health's filesystem fallback
used only as an explicitly degraded inventory when Code Facts is unavailable or
empty. The profile declares the policy that applies to those observed surfaces.

### Canonical shape

The canonical location is:

```json
{
  "unit": {
    "policy_profile": {
      "version": "1.0.0",
      "template": {
        "id": "react-vite",
        "scenario_class": "react-vite"
      },
      "required_roles": [
        { "role": "api", "policy_class": "go_service" },
        { "role": "cli", "policy_class": "go_cli" },
        { "role": "ui", "policy_class": "react_vite_ui" }
      ],
      "policy_classes": {
        "go_service": {},
        "go_cli": {},
        "react_vite_ui": {}
      },
      "customization": {
        "mode": "monotonic",
        "waivers": []
      }
    }
  }
}
```

`unit.policy_profile` is the canonical unit-infrastructure contract for
template-derived scenarios. Test Genie orchestration controls such as `phases`,
`presets`, `lint`, `business`, and `performance` remain top-level testing
configuration; they do not replace the unit policy profile.

### Resolver precedence

For each Code Facts observed surface, Unit Health resolves one effective policy
in this order:

1. Exact required role match. A surface whose kind/id/path corresponds to a
   required role uses that role's policy class and role-specific expectations.
2. Explicit profile match. Additive policy rules may target an extra role,
   path, framework class, or language when a scenario intentionally adds a
   surface beyond the template baseline.
3. Framework class match. Known framework evidence such as React/Vite maps to
   the matching Unit Health class when the profile does not name the role.
4. Language default. Supported languages without template policy use Unit
   Health defaults: Go uses `go test`, TypeScript/Vite uses Vitest, Python uses
   pytest, and Bash uses bats.
5. Unsupported or ungoverned fallback. Unknown languages or known languages
   with no acceptable framework receive a stable finding instead of being
   silently skipped.

Required roles are checked separately from observed surfaces. For a
react-vite-derived scenario, `api`, `cli`, and `ui` must be present unless a
valid waiver exists.

### Monotonic customization

Scenario-local policy may be equal to or stricter than the template baseline.
Examples of stricter changes include higher coverage floors, more required
coverage reporters, additional test utility roots, additional projection
checks, and extra additive surface policies.

Weakened policy is invalid unless it carries a formal waiver with:

- `reason`: why the weaker rule is acceptable now
- `owner`: accountable person or team
- `expires_at` or `revisit`: a concrete time or trigger
- `finding`: the Unit Health finding code/evidence the waiver addresses

Waivers do not change Code Facts discovery. They only explain why a specific
policy violation is temporarily accepted, and expired or malformed waivers are
reported as policy findings.

### Native projections

The profile is the desired policy. Native files prove whether the workspace
actually projects that policy:

| Surface | Projection examples |
|---|---|
| UI | `package.json` Vitest dependency and `test`/`test:coverage` scripts, `vite.config.ts` jsdom/setupFiles/V8 coverage/reporters/thresholds, `src/test-utils`, canonical render helper, production import ban. |
| API | `go test` coverage command compatibility, `internal/testutil` root, test helper production-import guard, injectable seam coverage. |
| CLI | `go test` coverage command compatibility, `cli/internal/testutil`, CLI production-import guard, smoke tests for app/help/version. |

Policy errors and projection drift are different failures. Invalid or weakened
declared policy is reported against `.vrooli/testing.json`; native config that
does not satisfy a valid policy is reported against the native file.

### Finding vocabulary

The hard cutover adds these stable finding classes before maturity mapping:

| Code | Class | Meaning |
|---|---|---|
| `UNIT_POLICY_PROFILE_INVALID` | schema | The profile is unreadable, unsupported, or violates the testing schema. |
| `UNIT_REQUIRED_ROLE_MISSING` | role | A required role such as `api`, `cli`, or `ui` was not observed by Code Facts. |
| `UNIT_SURFACE_UNGOVERNED` | governance | Code Facts observed a surface that has no explicit policy and no supported Unit Health default. |
| `UNIT_POLICY_WEAKENED` | policy | Scenario-local policy is weaker than the template baseline. |
| `UNIT_POLICY_WAIVER_INVALID` | governance | A waiver is malformed, expired, missing ownership, or does not reference evidence. |
| `UNIT_POLICY_PROJECTION_DRIFT` | projection | Native config or test infrastructure does not project the effective policy. |

Existing framework, coverage, architecture, quality, execution, and dependency
codes remain in use for concrete workspace failures. The new codes identify
contract ownership failures and should carry expected/observed/evidence fields
that explain whether the declared policy, discovered surface set, or native
projection is at fault.

### Non-goals

- `.vrooli/testing.json` is not a source of truth for actual surfaces.
- Unit Health does not generate native config files in this cutover.
- Test Genie does not reimplement policy resolution; it preserves config
  compatibility and delegates unit validation to Unit Health.
- L4/L5 advisory quality signals stay advisory unless a separate hardening pass
  proves a precise blocking rule.

## Cross-references

- [`QUICKSTART.md`](../QUICKSTART.md) — boot the scenario in 5 minutes
- [`api-endpoints.md`](api-endpoints.md) — endpoint reference
- [`cli-commands.md`](cli-commands.md) — CLI command reference
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — fixes for env/port/lifecycle issues
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — why these surfaces exist
