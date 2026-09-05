# Configuration — Device Control

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
(`path:docs/reference/port-allocation.md`) for the full policy.

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
For `device-control` the prefix is the scenario id upper-cased with
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
2. `${XDG_CONFIG_HOME}/vrooli/device-control/config.json`
3. `~/.vrooli/config/device-control/config.json`
4. `~/.config/vrooli/device-control/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
device-control configure api_base http://localhost:15001/api/v1
device-control configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port device-control API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
device-control`").

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


## Desktop accessibility binding

The protected owner bootstrap may set paired `helper.accessibility_socket` and
`helper.accessibility_bus_id` values for the explicitly selected local desktop
accessibility bus. The generated helper config retains its signing-key pin.
During a managed restart, owner provisioning can reconcile only these two
fields; destination/session/authority changes remain refused. Reconciliation
requires the helper lock to be free. The sidecar waits for generated config to
match the protected owner bootstrap before acquiring its lock, avoiding startup
with an old binding. Preserve both configured owner/helper environment paths
on managed lifecycle restarts. Do not delete generated key-pin witnesses.

A replaced accessibility daemon requires an explicit new bus-ID binding. A
mismatch at the same path is refused. Removing both optional binding fields
through the protected bootstrap and managed restart returns to plain X11.


### Native application discovery and semantic resolution

Use `device-control desktop applications --socket <owner.sock> --request <request.json>`
with the `session` returned by desktop open. The catalog returns opaque
application IDs, a revision and expiry. Observe with `session`, `application_id`
and `application_revision`, plus `--output <new.png>`. Do not combine application
selection with `process_id`. Choose a window from the semantic observation using
its opaque `window_id`.

`device-control desktop resolve --socket <owner.sock> --request <request.json>`
accepts `session` and `selector` containing `observation_revision`, `window_id`,
`name` and optional `editable_only`. Names match exactly. The result explicitly
reports absent, unique or ambiguous and returns matching `element_ids`. Only a
unique result identifies a single field; ambiguous candidates require further
selection. A stale observation or changed tree returns an error. Obtain a fresh
observation before resolving again. References confer no authority and remain
bound to the original lease. Stop the lease when work ends.


### Admitted desktop flows

`device-control desktop run-flow --socket <owner.sock> --request <flow.json> --json`
uses the current session and application catalog references. The request contains
`session`, `application_id`, `application_revision`, a unique `run_id`, and
`flow` with `transport: "desktop"`. Each `desktop-text` step has an `id`, exact
field name in `target`, and arguments `window`, `text`, and optional Unicode
character `position`. All steps are validated before execution; max32 steps
and30 seconds per run. Each step resolves its window/field afresh.

Repeat the identical request and run ID to retrieve the existing record without
re-execution. Changing the request under that ID refuses. A `claimed` record
after interruption is not permission to restart; an `incomplete` record requires
state inspection. Do not automatically retry with a new run ID. `passed` means
all expected native commands have applied receipts. Call desktop stop to end
the lease.

A `desktop-text-assert` step checks exact current field text without inserting
it. It accepts `window` and `text` (including an empty expected value), with no
`position`. Promotion requires a complete passing run ending in this assertion.

The following commands use the same explicit `--socket` and `--request` flags:

- `promote-flow`: supply the current `session`, original `source_session`,
  `source_run_id`, and comparison `context_key`. Omit `id` and use
  `expected_version: 0` for a new revision family. Repairs supply the saved `id`
  and exact `expected_version`; they must preserve outcome assertions and policy.
- `get-saved-flow`: supply current `session`, saved `id`, exact positive
  `version`, and matching `context_key`.
- `run-saved-flow`: supply those same fields, a new `run_id`, and current
  `application_id` and `application_revision`. Execution uses the immutable
  saved procedure and existing control lease. Duplicate requests return the
  prior run record without repeating steps.

Saved revisions retain authenticated actor, device, and surface scope. Current
owner admission remains required; a saved procedure grants no control authority.
To use the same account-owned revision as Portal, add `--access-token-file`
with a private regular file containing the account access token. On Unix, remove
all group and other permission bits (for example, mode0600). This selects the
account service on the explicit owner socket; an invalid or denied token never
falls back to local Unix authority. Without this flag, commands use the local
principal and its separate library. Tokens are read from the file, not arguments.
Portal forwards these operations through its authenticated desktop session
service. The context key is an explicit comparison label, not automatic proof
of application compatibility across machines.


### Desktop cleanup readback

After an uncertain Stop reply, use the same typed session request file to read
owner evidence without repeating Stop:

```sh
device-control desktop read-cleanup --socket /absolute/path/desktop-owner.sock --request session.json --json
```

For account-authorized sessions, also pass `--access-token-file` with a private
file containing a current token for the original actor. The command uses the
account namespace and does not fall back to local authority. `released: false`
means cleanup is still unconfirmed; a missing receipt or failed lookup also remains
unknown. Only `released: true` for the exact session confirms cleanup. Reading does
not restore input permission, acquire a replacement session, or send Stop.

To discover session references after a client restart, use a bounded admission
history page. The request file may contain `{}` for the default 50 entries, or
`{"page_size": 25, "page_token": "<previous next_page_token>"}`. The maximum page
size is 100. Continue until `next_page_token` is empty; cursors represent traversal
positions, not a snapshot of concurrent admissions.

```sh
device-control desktop list-admissions --socket /absolute/path/desktop-owner.sock --request page.json --json
```

Use the same account token option for account-owned sessions. The owner selects
the actor from authenticated context and limits results to its configured surface.
Entries include expired sessions and admissions whose helper Open reply was lost.
Each entry exposes only a session reference, expiry, and original control mode;
it contains no credential and does not authorize renewed control. Read cleanup
for each discovered session before treating it as released. Missing evidence
remains unresolved, even when the recorded expiry is in the past.

For new Open requests, include a unique canonical UUID in `request_id` and retain
the exact surface, control mode, and TTL request. If the reply is uncertain, use
that same request file with admission reconciliation:

```sh
device-control desktop reconcile-open --socket /absolute/path/desktop-owner.sock --request open-request.json --json
```

This operation cancels admission if the request has not reached the helper. The
`not_admitted` result prevents a later arrival of that request from executing.
A `forwarding` result returns the exact session reference; it remains unresolved
until exact cleanup evidence is available. Reconciliation does not repeat Open or
Stop and cannot relabel a forwarded request as never admitted. Use the original
account's current private token file for account-owned requests. Changing the
request payload while reusing its ID is rejected.


If a historical lease was never admitted by the helper, cleanup readback can
confirm absence only after an owner-signed permanent revocation. The helper
persists that revocation under the same exclusion boundary as Open and input,
so a delayed request or stale grant publication cannot revive it. An existing
lease or failed cleanup remains pending until the helper releases held input.
Ordinary historical credentials cannot manufacture this absence proof, and a
revocation credential cannot authorize Open or input.

Desktop activation context uses the current observation lease. The helper retains
one ephemeral context per destination and returns an opaque reference with display
geometry, pointer coordinates, and a maximum 30-second lifetime bounded by the
lease. A new capture replaces the reference. Native window and process IDs stay
inside the helper; the reference grants no focus or input authority. Owner capture
and read operations recheck the exact session and actor on every request.

Use `device-control desktop capture-activation --socket /absolute/owner.sock
--request session.json --json` with an `OwnerStopRequest` JSON object containing the
exact `session`. Use `desktop read-activation` with the same session and
`context_id` to read the returned reference without capturing again. Both commands
accept `--access-token-file` for explicit account authority and never fall back to
local authority. Capture replaces ephemeral context; it does not inject input.

`CaptureCompanionActivation` is a local-owner-only operation. Its request contains
the native companion window identity, never a process ID. The owner obtains the
process ID from Linux Unix peer credentials and the helper checks XRes window
ownership before and after capture. Account-forwarded requests are refused.
