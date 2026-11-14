# Configuration: Compute Manager

How this scenario is configured: the environment variables its binaries
consume, the credential authority that holds provider tokens, the
[`.vrooli/service.json`](../../.vrooli/service.json) manifest, and the
per-user CLI config file.

The lifecycle (`vrooli scenario start`, `make start`) sets every required
variable automatically. You only need this reference when running a
binary by hand, or when adding a new setting.

## Read this first: implementation status

**None of the Compute Manager settings is read by code yet.** This
scenario was generated from the `react-vite` template. The only
configuration that is consumed at runtime is the template's own: two
ports, the optional CLI overrides cli-core derives from the scenario
name, and the CLI config file.

The manifest, however, is no longer empty.
[`.vrooli/service.json`](../../.vrooli/service.json) declares four
scenario dependencies, one credential descriptor for the Hetzner Cloud
API token, a `storage` entry for the scenario-owned SQLite database, and
a five-key `environment` block. **Declared is not read.** No business
suite URL, bridge URL, sweep interval or billable unit reaches any Go
code today, because no domain exists to read it.

Every table below is labelled. Rows marked **exists today** are consumed
at runtime. Rows marked **declared** are in the manifest with no reader
behind them. Rows marked **planned** are a design commitment that is in
neither place yet, and adding one means adding it to the manifest and the
reader that consumes it in the same change.

## Configuration homes

Compute Manager keeps four kinds of configuration apart on purpose,
because mixing them is how a provider token ends up in a log line.

| Category | Home | Examples | Status |
|---|---|---|---|
| Lifecycle process settings | process environment set by the lifecycle | `API_PORT`, `UI_PORT` | Exists today |
| Product runtime state | scenario-owned SQLite | instance intents, instances, provider receipts, reservations, usage, reconciliation findings and enrollment queue | Declared: `storage.entries.data` in the manifest; schema is loaded from `api/internal/store/schema.sql` |
| Operator secrets | Vrooli credential authority | provider API tokens | Declared: one descriptor in the manifest. Nothing resolves it yet |
| Operator preferences | per-user CLI config file | `api_base`, `token` | Exists today |

A provider API token belongs in exactly one of those, the credential
authority. It is never a process environment variable, never a command
line argument, never a row in this scenario's database, and never a value
returned to a UI or CLI client.

## Environment variables

### Required at runtime, set by the lifecycle (exists today)

| Variable | Range / format | Purpose |
|---|---|---|
| `API_PORT` | `15000-19999` | Port for the Go API server |
| `UI_PORT` | `20000-24999` | Port for the production UI server (`ui/server.js`) |

If the scenario adds WebSocket channels on the existing API or UI server,
do not add another `ports` entry. Declare an additional port only when the
scenario starts a separate listener process.

The canonical bands all sit below 32768 so Linux never hands out the
ports as outbound source ports. See the project-level port allocation
reference (`path:../../docs/reference/port-allocation.md`) for the full policy.

### Optional overrides (exists today)

| Variable | Default | Purpose |
|---|---|---|
| _(none for storage)_ | not applicable | The SQLite file location is **not** configurable through the environment. It is resolved from the scenario id by `api-core/storage`, so no inherited variable can point one scenario at another's database. Set `VROOLI_STORAGE_ROOT` to relocate the whole storage tree for a test run |
| `API_TOKEN` | unset | Shared bearer token for CLI to API auth. Only enforce it in production deployments |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded |

The browser UI does not read `API_PORT` directly. It resolves API calls
through the UI origin, and `ui/server.js` proxies `/api/*` plus the
scenario's Connect RPC namespace to the API process using the
lifecycle-provided `API_PORT`.

### Scenario-prefixed CLI variables (exists today)

`cli-core` derives a standard set of env vars from the scenario name. For
`compute-manager` the prefix is `COMPUTE_MANAGER`, which is the scenario
id upper-cased with hyphens replaced by underscores. The following are
recognised, in precedence order with first found winning:

| Purpose | Variables |
|---|---|
| API base URL | `COMPUTE_MANAGER_API_BASE`, `COMPUTE_MANAGER_API_URL`, `VROOLI_API_BASE` |
| API port | `COMPUTE_MANAGER_API_PORT` |
| API token | `COMPUTE_MANAGER_API_TOKEN`, `VROOLI_API_TOKEN` |
| Config dir | `COMPUTE_MANAGER_CONFIG_DIR`, `VROOLI_CLI_CONFIG_DIR` |
| HTTP timeout | `COMPUTE_MANAGER_HTTP_TIMEOUT`, `VROOLI_HTTP_TIMEOUT` |

> **Do not** set the un-prefixed `API_PORT` for a CLI invocation. When
> CLIs run inside web-console terminals it leaks across scenarios. Use
> the scenario-prefixed form or the `--api-base` flag.

### Provider credentials (declared), and never an environment variable

A provider API token is the highest-value secret this scenario touches.
Anyone holding it can create billable machines. It is intended to resolve
through the Vrooli credential authority at call time, and the descriptor
is declared in `.vrooli/service.json` under `credentials.descriptors`.
One descriptor is declared today, for the first adapter, Hetzner Cloud.
No code resolves it, because no provider adapter exists.

This table mirrors the shipped descriptor field for field:

| Field | Declared value | Why |
|---|---|---|
| `logical_id` | `vrooli/compute-manager` | Backend-neutral identity, durable across hosts, platforms and deployment tiers. A descriptor never names Vault, a file, or any storage-backend-shaped path. The identity is the scenario, so each adapter's token is a `field` under it rather than a namespace of its own |
| `field` | `hetzner-api-token` | Which value of that identity this descriptor addresses. Naming the provider here is what keeps a second adapter from colliding with the first |
| `label` | `Hetzner Cloud API token` | Shown in onboarding and admin surfaces |
| `description` | Authorises instance creation and destruction at Hetzner Cloud. Resolved through credential-authority-go at call time and deliberately NOT injected into the process environment, so it cannot be read from the process table or inherited by a child process | States what it unlocks and how it is reached, which are the two parts that matter here |
| `required` | `false` | **Deliberate, and revisited when the first adapter lands.** The schema defines `required` as whether the declaring component can do its job without the value. Today no component reads it: the scenario boots, serves inventory, findings and settings, and the Settings surface has a declared `empty` state for exactly the no-provider case. Marking it required now would report a permanently unsatisfied credential against a scenario that has nothing to satisfy it for. Flipping it to `true` belongs in the same change that adds the first provider adapter |
| `provisioning` | **omitted; the schema default `operator` applies** | A person supplies it. There is nothing to derive and nothing to generate, so the default is already correct and stating it adds nothing |
| `obtain_url` | `https://console.hetzner.cloud/` | Where an operator generates it |
| `env` | **omitted deliberately** | See below |

**The `env` key is omitted on purpose, and that omission is the point.**
The schema permits `env` only when the consumer is a process Vrooli does
not author, such as a database container or a third-party CLI, which can
receive a value no other way. This scenario's provider adapters are
Vrooli-authored Go code, so they resolve through the credential-authority
binding for their language. That keeps the value out of the process
environment, where it would be readable at `/proc/<pid>/environ` and
inherited by every subprocess the API spawns.

The same rule applies to the command line. A token passed as an argument
is visible in `ps` output to every user on the host, so no planned CLI
verb accepts a provider token as a flag or positional. If a token needs
to be supplied through the CLI, it is read from standard input into the
credential authority, never echoed and never stored by this scenario.

A second adapter adds a second descriptor with its own `field` under the
same `logical_id`. It does not add an environment variable either.

### Compute Manager settings

None of these is read by any code today. Five are declared in the
manifest's `environment` block, which sets them for every lifecycle step;
two are not, and that omission is deliberate. Naming follows the
ecosystem convention of the scenario prefix plus a setting name, and Go
duration syntax (`30s`, `5m`, `1h`) is accepted wherever an interval is
expected.

| Variable | Manifest value | Status | Purpose |
|---|---|---|---|
| `COMPUTE_MANAGER_LPBS_BASE_URL` | `""` | Declared | Base URL of `landing-page-business-suite`, which holds the reservation and settlement surface. The empty value means unset: the base URL is resolved from the lifecycle, and a non-empty value overrides that, which is how a test run points at a fake business suite. **This dependency fails closed.** If it cannot be reached, provisioning refuses rather than degrades, because a machine that boots unmetered is cost that grows hourly and cannot be recovered afterwards. Existing instances continue and settle when it returns |
| `COMPUTE_MANAGER_BRIDGE_BASE_URL` | `""` | Declared | Base URL of `vrooli-bridge`, used for the onboarding public key, the Machine record and onboarding start. The empty value means unset and resolves from the lifecycle, exactly as above. **This dependency degrades.** If it cannot be reached, the instance is still created, metered and expiring; enrollment queues and retries, and the instance is visible and flagged as not enrolled. Blocking capacity on the trust plane would make capacity unavailable for a reason unrelated to capacity |
| `COMPUTE_MANAGER_RECONCILE_INTERVAL` | `15m` | Declared | Cadence of the bidirectional reconciliation sweep. The loop runs once at boot, then on this interval, comparing provider inventory against local records in both directions. The sweep reports findings and mutates no instance, so a shorter interval costs provider API calls and nothing else |
| `COMPUTE_MANAGER_EXPIRY_INTERVAL` | `1m` | Declared | Cadence of the expiry sweep, which destroys instances past their lifetime. Short on purpose: this loop is the first of the two expiry enforcement points, and every minute of lateness is billable |
| `COMPUTE_MANAGER_MINIMUM_BILLABLE_UNIT` | `1h` | Declared | The smallest span this scenario charges for. Most providers round a partial hour up to a full hour, so a five-minute instance costs a full hour of provider spend. Charging below the unit we are billed at loses money on every short workload |
| `COMPUTE_MANAGER_RECONCILE_SCHEDULER_DISABLED` | not in the manifest | Planned | Set to `1`, `true` or `yes` to stop the background reconciler for controlled maintenance. Manual `compute-manager reconcile run` stays available. Leaving it disabled lets divergence accumulate silently, so it is a maintenance switch and not a setting |
| `COMPUTE_MANAGER_EXPIRY_SCHEDULER_DISABLED` | not in the manifest | Planned | Set to `1`, `true` or `yes` to stop the background sweeper. The second enforcement point, the timer rendered into the instance's own first-boot configuration, is unaffected and still drains the fleet. That is exactly why it exists |

**Why the two `_SCHEDULER_DISABLED` switches are absent from the
manifest, and should stay absent.** The `environment` block sets a value
for every lifecycle step, so a key listed there is always present in the
process environment. A maintenance switch has to be distinguishable from
its own default: an operator sets it for one run and unsets it
afterwards. Declaring it in the manifest would make "not disabled" a
value the manifest asserts on every start, and would hand the reader an
empty string to interpret rather than an absent key. These two are read
from the environment if present and are otherwise unset, which is the
shape a maintenance switch needs.

Two notes on the minimum billable unit, because it is easy to conflate
two different numbers:

- The **provider's** minimum billable unit and rounding behaviour are
  billing facts each adapter declares as data, alongside whether a
  stopped instance bills and whether inbound traffic counts against the
  transfer allowance. Those are adapter constants read through
  `provider describe`, not configuration, because they are properties of
  the provider and not choices an operator makes.
- The **scenario's** minimum billable unit above is the floor we apply.
  It should never sit below the provider's, and warm pooling is the other
  mitigation for hourly rounding, deferred until measured churn shows the
  rounding dominating the bill.

Provider spend alerts are not configuration for this scenario at all.
They send mail and stop nothing. The real ceiling is the one computed
from our own meter.

## Service manifest (`.vrooli/service.json`)

Single source of truth for everything the lifecycle needs to know. The
live file is [`.vrooli/service.json`](../../.vrooli/service.json).

| Section | Owns | Status here |
|---|---|---|
| `service` | name, display name, description, version, type, category, tags, `class`, `capabilities` | Populated. `class` is `external-integration`, because the scenario's reach is outbound HTTPS to provider APIs. `capabilities` lists the five it claims: `compute-provisioning`, `compute-inventory`, `compute-metering`, `instance-reconciliation`, `instance-expiry`. Those are claims about intended scope, not evidence that any of them runs |
| `generation` | template id and version, design kit, content hashes | Populated |
| `ports` | port name to env var and range mapping | Populated with `api` and `ui` |
| `cli` | command name, adapter, invoke shape, freshness inputs | Populated |
| `components` | portable process builds, argv, cwd, port ownership | Populated with `api` and `ui` |
| `lifecycle.health` | `/health` endpoint, startup grace period, periodic checks | Populated, 15s startup grace |
| `lifecycle.setup` / `develop` / `stop` | exceptional ordered work as native argv steps | Absent, template defaults apply |
| `environment` | static env vars set for every lifecycle step | Populated with five keys: `COMPUTE_MANAGER_LPBS_BASE_URL` and `COMPUTE_MANAGER_BRIDGE_BASE_URL` (both empty, meaning resolve from the lifecycle), `COMPUTE_MANAGER_RECONCILE_INTERVAL` `15m`, `COMPUTE_MANAGER_EXPIRY_INTERVAL` `1m`, `COMPUTE_MANAGER_MINIMUM_BILLABLE_UNIT` `1h`. Set for every lifecycle step and read by nothing yet |
| `dependencies.resources` | shared local resources such as postgres or redis | **`{}` and expected to stay empty.** This scenario runs no local service. Its state is SQLite and its reach is outbound HTTPS |
| `dependencies.scenarios` | scenario dependencies and their degradation behaviour | All four declared, each with a `description` and a `degraded_behavior`: `landing-page-business-suite` required, `must_start`, fail-closed; `vrooli-bridge` required, `try_start`, degrading; `treasury` optional, `ignore`, gating the agent purchase path only; `offer-desk` optional, `ignore`, catalog only with no runtime path. `api/internal/capabilities/registry.go` mirrors all four |
| `credentials.descriptors` | credential identities this scenario needs | One descriptor, `vrooli/compute-manager` field `hetzner-api-token`, shaped as in the table above. A second adapter adds a second `field` under the same `logical_id` |
| `storage` | scenario-owned storage entries, their rung, class and budget | Populated with one entry, `data`: rung `owned`, kind `dir`, class `data`, `regenerable: false`, budget ceiling `5GiB`. The rationale is that an instance the database forgets is an instance that keeps billing, so none of it may be treated as regenerable. No table has been created inside it yet |
| `maturity` | the scenario's declared maturity rung | `greenfield`. Consistent with every implementation-status banner in these docs |

Testing is not a lifecycle phase. `.vrooli/testing.json` declares suites,
and `vrooli scenario test compute-manager` delegates the run to Test
Genie.

## Schema bootstrap

Schema is owned per domain. `api/internal/<domain>/schema.sql` declares
each domain's tables and is embedded into the binary through Go's embed
directive from `api/internal/<domain>/schema.go::Schema()`. Cross-cutting
infrastructure such as extensions, custom types and cross-domain views
lives in `api/internal/database/system.sql`, which is empty in SQLite
scenarios like this one.

The shared registry at `api/internal/modules/registry.go::AllSchemas()`
collects them in a stable order, system first and then domains
alphabetically, and `apidb.EnsureSchemas(ctx, db, modules.AllSchemas()...)`
from `api-core/database` applies them at startup. The path is idempotent,
because all DDL uses `CREATE TABLE IF NOT EXISTS` and `ALTER TABLE ...
ADD COLUMN IF NOT EXISTS`, so re-runs on every boot are no-ops.

Registered today, in `api/internal/modules/registry.go::AllSchemas()`:
the system schema and the health schema, and nothing else. The `notes`
example schema went with the example domain when `template-manager
detemplate compute-manager` ran. The planned domain tables are
`instance_intents`,
`instances`, `provider_receipts`, `reservations` and
`reconcile_findings`, each owned by its domain and registered with two
lines in the registry when that domain is built.

No provider token and no customer payment data is ever written to this
database. The `reservations` table holds reservation identifiers so
settlement can find them, and nothing more; wallets, entitlements and
invoices belong to `landing-page-business-suite`.

Adding a column lands in the same diff as the Go struct field, the
repository scan, and the proto wire shape, so it is a single location and
a single edit. Drops and renames in production data need the brownfield
versioned-migration helpers (`Migrate` and `MigrationProvider` in
`api-core/database`), deferred until the first scenario hits the pain.

See [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) for the
design rationale and [`../internal/SEAMS.md`](../internal/SEAMS.md) for
the per-seam table.

## CLI config file

The scenario CLI persists per-user configuration to a JSON file.
Resolution order, first match wins:

1. `${COMPUTE_MANAGER_CONFIG_DIR}/config.json`
2. `${XDG_CONFIG_HOME}/vrooli/compute-manager/config.json`
3. `~/.vrooli/config/compute-manager/config.json`
4. `~/.config/vrooli/compute-manager/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values through the CLI rather than editing the file directly:

```bash
compute-manager configure api_base http://localhost:15001/api/v1
compute-manager configure token "<token>"
```

The `token` field is the CLI to API bearer token for this scenario's own
API. **It is not a provider credential**, and no provider token may be
written here. This file sits in a user's home directory in plain text; a
credential that can create billable machines does not belong in it.

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order, first
match wins:

1. `--api-base <url>` flag
2. Scenario-prefixed env vars, listed above
3. CLI config file, the `api_base` field
4. Vrooli lifecycle port detection
   (`vrooli scenario port compute-manager API_PORT`)
5. Compile-time default, only if explicitly configured in `cli/app.go`

If none of these resolve, the command exits with an actionable error
naming the address it tried and suggesting `--auto-start` or `vrooli
scenario start compute-manager`.

The declared dependency base URLs follow the same shape, resolved from
the lifecycle first and overridable by the scenario-prefixed variable, so a
test run can point the scenario at a fake business suite or a bridge fake
without touching code.

## Test/CI configuration

| File | Owns |
|---|---|
| `.vrooli/testing.json` | Test categories: lint, unit, structure, business checks, performance |
| `.vrooli/lighthouse.json` | Lighthouse pages, thresholds, Chrome flags |
| `.vrooli/endpoints.json` | API endpoint manifest: path, method, status codes, request and response shapes, CLI mapping |
| `.vrooli/orientation.json` | Scenario orientation state |
| `.github/workflows/test.yml` | CI gate: UI lint and test, Go vet, race and coverage, end-to-end binary smoke |

These files are read by tooling (`vrooli scenario test`, `test-genie`,
the doc viewer), so keep them in sync with the code they describe.

Two test-side settings matter more than usual for this scenario, and
neither exists yet:

- **A fake provider is the default test target.** The reserve, provision
  and settle spine, intent before action, bidirectional reconciliation
  and double-enforced expiry are the four failure modes, and none of them
  needs a real API key. The Hetzner adapter is wired only once those
  hold, so no suite should ever require a provider credential to pass.
- **No test may read a real provider token.** A suite that needs one is a
  suite that would fail in CI and pass only on the author's machine.
  Contract tests against a real provider, when they are eventually
  written, belong behind an explicit opt-in and never in the default
  gate.

### Unit testing policy profile

The `unit.policy_profile` block in `.vrooli/testing.json` declares the
template's unit-test policy. It is not a list of surfaces. Code Facts
discovers the actual `api`, `cli` and `ui` code surfaces; Unit Health
joins those observed surfaces to the profile and reports drift.

The React and Vite template requires three roles:

| Role | Policy class | Baseline |
|---|---|---|
| `api` | `go_service` | Go `go test`, 75% total coverage, `api/internal/testutil`, production-import guardrail |
| `cli` | `go_cli` | Go `go test`, 75% total coverage, `cli/internal/testutil`, app smoke test, production-import guardrail |
| `ui` | `react_vite_ui` | Vitest through pnpm, jsdom setup, V8 coverage, 85% coverage thresholds, `ui/src/test-utils/renderWithProviders.tsx` |

Scenario customizations are monotonic. They may add surfaces, add
stricter checks, or raise thresholds. They may not weaken the template
baseline unless the policy includes a waiver with an owner, a reason, an
expiry or revisit trigger, and the Unit Health finding evidence it
addresses.

`unit.policy_profile` is the only unit-infrastructure contract emitted by
this template. Test orchestration knobs such as phase timeouts and
presets stay in their own top-level blocks; unit surfaces are discovered
by Code Facts and governed by the policy profile.

## Cross-references

- [`../QUICKSTART.md`](../QUICKSTART.md): boot the scenario in five minutes
- [`api-endpoints.md`](api-endpoints.md): endpoint reference, including what is planned
- [`cli-commands.md`](cli-commands.md): CLI command reference
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md): what each dependency does when it is unavailable, and the provider choice behind the credential descriptor
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md): the domains that own the planned tables
- [`../internal/SECURITY.md`](../internal/SECURITY.md): credential handling rules
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md): fixes for environment, port and lifecycle issues
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md): why these surfaces exist
