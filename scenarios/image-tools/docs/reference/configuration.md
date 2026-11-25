# Configuration — Image Tools

How this scenario is configured — env vars consumed by the binaries,
the `.vrooli/service.json` manifest, and the per-user CLI config file.

The lifecycle (`vrooli scenario start`, `make start`) sets every
required variable automatically. You only need this reference when
running a binary by hand or when a scenario adds a new variable.

## Environment variables

### Required at runtime (set by the lifecycle)

See the [canonical port allocation reference](../../../../docs/reference/port-allocation.md) for API and UI ranges and the full listener policy.

If the scenario adds WebSocket channels on the existing API or UI server, do
not add another `ports` entry. Declare an additional port only when the
scenario starts a separate listener process.



### Optional overrides

| Variable | Default | Purpose |
|---|---|---|
| _(none)_ | — | The SQLite file location is **not** configurable through the environment. It is resolved from the scenario's own identity by `api-core/storage`, so no inherited variable can point one scenario at another's database. To relocate storage for a test run, set `VROOLI_STORAGE_ROOT`, which redirects the whole class tree and stays scenario-agnostic. |
| `API_TOKEN` | unset | Shared bearer token for CLI ↔ API auth (only enforce in production deployments). |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |
| `IMAGE_TOOLS_CPU_WORKERS` | `4` | Maximum simultaneous local job admission/execution slots, including AI and analysis. Valid range: `1`-`32`. CPU/RAM/GPU capacity and exclusive warm-worker use can impose lower concurrency. Raising this ceiling never bypasses admission. |
| `IMAGE_TOOLS_INSTALL_MB_PER_SECOND` | `15` | Assumed model-download throughput for install ETA calculation. Valid range: `1`-`10000` MB/s. Tune to measured host/network throughput; it affects ETA reporting only, not download behavior. |
| `IMAGE_TOOLS_DISK_FLOOR` | the host minimum (`120GiB`) | Free space every ordinary disk job must leave, raised above the host minimum. A value at or below the minimum is ignored and reported, never applied. See [Disk floor](#disk-floor). |
| `IMAGE_TOOLS_MODEL_AUTO_ACQUIRE` | `on` | Lets an AI or analysis job download the selected model's weights through the governed installer and shared disk admission when they are missing. Only enabled, rights-permitted catalog models with a non-manual source qualify; custom, imported and manual entries always need `image-tools models install`. `on/off`, `true/false`, `1/0`, `yes/no`. A caller can opt out per request (`--no-auto-acquire`, `AIParams.auto_acquire=false`). |
| `IMAGE_TOOLS_MODEL_AUTO_OFFLOAD` | `on` | Lets the periodic and pre-acquisition passes remove idle model weights. Off keeps status and dry-runs (`image-tools models storage`, `models offload`) and operator-requested `models offload --apply`. Same boolean syntax. |
| `IMAGE_TOOLS_MODEL_OFFLOAD_RESERVE` | `24GiB` | Headroom above the 120 GiB shared disk floor that offload restores when free space falls short (plus the bytes a pending acquisition needs). Binary unit required (`B`, `KiB`, `MiB`, `GiB`, `TiB`). |
| `IMAGE_TOOLS_MODEL_OFFLOAD_AMPLE` | `96GiB` | Headroom at or above which no model idles out. Must exceed the reserve. |
| `IMAGE_TOOLS_MODEL_OFFLOAD_MIN_IDLE` | `24h` | Idle threshold at (or below) the reserve. Go duration syntax. |
| `IMAGE_TOOLS_MODEL_OFFLOAD_MAX_IDLE` | `720h` | Idle threshold just below ample headroom; between reserve and ample the threshold shrinks geometrically from this toward the minimum. |
| `IMAGE_TOOLS_MODEL_OFFLOAD_INTERVAL` | `30m` | Time between periodic offload passes. The first pass runs 20 minutes after start so `image-tools models storage` can be reviewed after a restart. |
| `IMAGE_TOOLS_MODEL_STAGING_GRACE` | `2h` | How long an acquisition staging directory (`staging/models/.acquiring-*`, `staging/adapters/.acquiring-*` in the data root; earlier releases staged beside the weights and those leftovers are swept too) with no owning install in this process must go without any write before it is abandoned. Abandoned staging is reclaimed about 2 minutes after boot, on every periodic pass and before a pre-acquisition offload (only while auto-offload is on; otherwise it is only reported by `image-tools models storage`). An install of the same repository revision within the grace period resumes the partial snapshot instead of restarting. Go duration syntax. |
| `IMAGE_TOOLS_CLOUD_FALLBACK` | `on` | Lets `edit_instruct`, `image_to_image` and `text_to_image` use the cloud without the caller asking: on a host with no usable GPU instead of a weak CPU model, and as a wait-budget re-route when local capacity is not free in time. `on/off`, `true/false`, `1/0`, `yes/no`. A caller's `--fallback-policy local_only` always wins; `--byok` or `--fallback-policy cloud_allowed` permits the cloud even when this is off. |
| `IMAGE_TOOLS_CLOUD_WAIT_BUDGET` | `10m` | How long a job may wait for local admission (queue plus capacity) before it is re-routed to the cloud. `0` disables the wait trigger. Go duration syntax. Per request: `--max-wait`. |
| `IMAGE_TOOLS_CLOUD_MAX_JOB_USD` | `0.25` | Per-job cloud spend cap in USD, compared with a per-role cost estimate before the call. Per request: `--max-cost`. |
| `IMAGE_TOOLS_CLOUD_MAX_DAILY_USD` | `3` | Rolling 24-hour cloud spend cap in USD, from recorded `cost_usd` plus the estimates of cloud jobs in flight. Not settable per request. |
| `IMAGE_TOOLS_CLOUD_COOLDOWN` | `30m` | How long automatic cloud routing pauses after a cloud route is refused before any billing (no credits, bad credentials, provider or gateway unreachable, rate limited). The pause is read from the job table, so it survives restarts and covers every waiting job. `0` disables it. A caller who demands the cloud (`--byok` or a named `--role`) is never held back by it. |
| `IMAGE_TOOLS_CLOUD_COST_ESTIMATES` | built in | Per-role cost ceilings used before a call, `role=usd,role=usd` (for example `image.edit.identity=0.14,image.edit.default=0.05`). Built in: identity edit 0.14, default edit 0.05, quality generate 0.14, default generate 0.05, fast generate 0.03, any other role 0.15. A malformed cloud routing value stops the service at boot. |
| `IMAGE_TOOLS_DEPLOYMENT_TIER` | `local` | Selects the Responsible-Use control preset. Anything other than `public`/`prod`/`production`/`monetized`/`saas` resolves to `local`. The tier chooses defaults only; it never enables or disables a capability. |
| `IMAGE_TOOLS_REQUIRE_CONSENT_NONLOCAL` | on (every tier) | Require and audit a consent affirmation when a high-weight operation's resolved route leaves this machine (cloud provider, BYOK, remote node). Local runs are never affected. `1/true/yes/on` or `0/false/no/off`; other values are ignored. |
| `IMAGE_TOOLS_REQUIRE_CONSENT` | preset (off local, on public) | Require and audit a consent affirmation for high-weight (identity/body/clothing/appearance-altering) operations. `1/true/yes/on` or `0/false/no/off`; other values are ignored. |
| `IMAGE_TOOLS_NSFW_SCAN` | preset (off local, on public) | Force content scanning of generated output regardless of the per-request flag. Same boolean syntax. |
| `IMAGE_TOOLS_REQUIRE_PROVENANCE` | preset (off local, on public) | Mark outputs with provenance/attribution metadata. Same boolean syntax. Turning it on never blocks an edit. |
| `IMAGE_TOOLS_RATE_LIMIT_PER_MIN` | preset (`0` local, `60` public) | AI-submit rate cap. `0` = unlimited. Malformed values leave the preset intact. |
| `IMAGE_TOOLS_MINOR_SAFETY_CMD` | unset (bundled reviewer) | Replaces the **bundled** minor-safety reviewer for the one non-configurable boundary (sexual content involving minors). With it unset, the bundled reviewer (the configured classifier ensemble + YuNet faces + FairFace age, CPU-only) runs on every tier and every produced image, and a reviewer that cannot run fails closed. A custom value is whitespace-separated program + args, executed without a shell; it receives one JSON object on stdin (including the resolved `profile`) and must return `{"action":"allow|flag|refuse","reason_code":"...","minor":bool,"sexual":bool,"detail":string}`; the older `{"minor":bool,"sexual":bool,"detail":string}` is still accepted. A reviewer error or timeout fails closed. See [SECURITY.md](../internal/SECURITY.md#reviewer-contract). |
| `IMAGE_TOOLS_SAFETY_PROFILE` | unset (embedded `profiles.json`) | Path to a declarative reviewer profile set (same schema as `api/internal/safety/profiles.json`). It names the classifiers, thresholds, strict thresholds, face/age models, second-opinion policy, refusal action and mode. Malformed JSON or an unsafe profile stops the service at boot rather than silently changing the boundary. |
| `IMAGE_TOOLS_SAFETY_PROFILE_NAME` | tier name (`local` / `public`) | Selects a profile by name from `IMAGE_TOOLS_SAFETY_PROFILE` (or the embedded set). |
| `IMAGE_TOOLS_SAFETY_MODE` | `enforce` | `enforce` refuses as configured; `shadow` records a would-refuse verdict and marks the output sensitive without deleting or failing. Shadow is for local experimentation and should not be used on a deployment that serves other people. |
| `IMAGE_TOOLS_SAFETY_ON_REFUSAL` | `drop_variation` local, `fail_job` public | What a refusal does to a multi-variation job: `drop_variation` withholds only the refused variation and returns the others; `fail_job` withholds every variation and fails the job. A withheld variation is quarantined when `quarantine_hours > 0`, otherwise deleted; it is never published under either setting. |
| `IMAGE_TOOLS_SAFETY_QUARANTINE_HOURS` | profile `quarantine_hours` (`168` local, `0` public) | How long a refused output is retained in the private admin-review quarantine. `0` deletes it immediately. Expired entries are purged hourly. Local deployments are ungated; a public deployment also needs an admin token. |
| `IMAGE_TOOLS_SAFETY_QUARANTINE_TOKEN` | unset | Public/cloud admin token for the quarantine list and image route. Local deployments do not need it (the local operator is the admin). A public deployment that sets a positive retention without a token logs the misconfiguration and falls back to immediate deletion, so quarantined bytes are never stored without a review path. |
| `IMAGE_TOOLS_OLLAMA_URL` | `http://127.0.0.1:11434` | Loopback URL for the face-independent age second opinion when the AI Gateway's `vision.default` route cannot execute. The call is always local-only; no hosted URL is accepted by the reviewer path. |
| `IMAGE_TOOLS_AGE_JUDGE_MODEL` | `qwen3-vl:2b` | Ollama model used by that loopback age question. It fits beside the resident models on this 16 GB card and returns the JSON answer; the 4B primary needs about 7.4 GB and was not reliable as a cold load here. |
| `IMAGE_TOOLS_AGE_JUDGE_FALLBACK_MODEL` | `llava:7b` | Second loopback model when the primary spends its answer on prose or fails; it returns JSON reliably but reads some adult bodies younger, so an uncertain reading still refuses. |

The harm-protection overrides win in **both** directions: a personal install
may opt into provenance marking, and an operator running their own public
instance may turn a control off. The minor-safety control is the exception: it is
required on the public tier and cannot be overridden off. `image-tools safety policy` / `SafetyService.GetPolicy`
reports the resolved result, so the UI cannot claim a protection that is off.

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Disk floor

One module, `api/internal/diskpolicy`, decides how much free disk a write must
leave. Installs, job output, link imports and the offload planner all read it,
so no file carries its own number. `image-tools models containment` prints the
value with its reasons.

- **Floor = the host minimum.** The shared capacity ledger holds every ordinary
  disk job to at least 120 GiB of free space afterwards and refuses to lower it
  outside tests, so image-tools requests exactly that and no private, larger
  figure. A job is refused only when the host's own policy refuses it. Raise it
  with `IMAGE_TOOLS_DISK_FLOOR` (for example `200GiB`); the operator value must
  exceed the minimum to apply. On a 1.8 TB disk the floor is 6.5% of the disk.
- **Small writes use the small-write class.** A write of at most 64 MiB in total
  (a link import) is declared to the ledger as a small write and held to its
  8 GiB floor, so a 5 MB import is never refused for the space a 40 GB download
  reserved.
- **Admission is sized by real bytes.** A model install reserves the artifact
  plus `max(10%, 64 MiB)` at the model root, tool scratch of
  `max(128 MiB, min(artifact/16, 2 GiB))` and a Hub cache allowance of
  `min(artifact, 512 MiB)`. If the download passes the plan, the lease is
  extended to the observed bytes plus 25% (at least 1 GiB) before the next write,
  never to a multiple of them.
- **A busy denial waits.** The ledger admits when free space is at least every
  other active reservation, plus this job's peak, plus the floor. When raw free
  space would have been enough and only other owners' in-flight reservations (a
  large install, another scenario's job) cause the denial, a job is held and
  admitted again instead of failing. When the disk really is too full above the
  floor it fails with the numbers.

### CPU and memory declared by support jobs

Every job that is not model inference holds a RAM and CPU claim on the host
ledger while it runs. The default is one core and 256 MiB. A job that is
network- and disk-bound declares what it uses instead:
`api/internal/capacity/support_profiles.json` has one row per operation, with a
reason. Model and adapter installs declare a quarter of a core, so a host with
every core claimed no longer holds a download in the queue; disk space for the
download is admitted separately (see above). Add a row when a new operation's
real need is clearly below the default; a test fails if a row names an operation
no job runs under. The scheduler ages waiting jobs (one rank per 30 seconds
queued) and backs refusals off to at most 16 seconds, so an old job is retried
at a steady rate. The shared ledger itself grants or denies on the spot and has
no queue of waiters, so a stream of small claims by other owners can still keep
one large claim waiting.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `image-tools` the prefix is the scenario id upper-cased with
hyphens replaced by underscores (so `my-scenario` → `MY_SCENARIO`).
The following are recognised, in precedence order (first-found wins);
substitute your scenario's prefix for `<PREFIX>`:

The standard variable names and order come from `cliapp.StandardScenarioEnv`; see the [CLI Core scenario environment contract](../../../../packages/cli-core/README.md#scenario-cli-environment). Keep scenario-specific additions in this reference.

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
for the per-seam table including `jobs.Schema`, `models.Schema`, and
`database.SystemSchema`.

## CLI config file

The scenario CLI persists per-user configuration to a JSON file.
Resolution order (first match wins):

1. `${<PREFIX>_CONFIG_DIR}/config.json` (the scenario-prefixed env var; see "Scenario-prefixed CLI variables" above)
2. `${XDG_CONFIG_HOME}/vrooli/image-tools/config.json`
3. `~/.vrooli/config/image-tools/config.json`
4. `~/.config/vrooli/image-tools/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
image-tools configure api_base http://localhost:15001/api/v1
image-tools configure token <token>
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port image-tools API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
image-tools`").

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
