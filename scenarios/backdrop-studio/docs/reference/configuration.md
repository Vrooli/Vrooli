# Configuration — Backdrop Studio

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
(`path:../../docs/reference/port-allocation.md`) for the full policy.

### Optional overrides

| Variable | Default | Purpose |
|---|---|---|
| `SQLITE_PATH` | `${SCENARIO_DATA_DIR}/backdrop-studio.db` | Override SQLite file location. The default routes through `api-core/storage` and resolves to a writable per-scenario data directory. |
| `API_TOKEN` | unset | Shared bearer token for CLI ↔ API auth (only enforce in production deployments). |
| `UI_BASE_URL` | (resolved by `@vrooli/api-base`) | External UI URL when the scenario is iframe-embedded. |
| `BRAND_MANAGER_URL` | (service discovery) | Pin the Brand Manager address. Normally unset — `--brand` resolves brand-manager through the same discovery every other cross-scenario call uses. |
| `BACKDROP_STUDIO_API_SOURCE` | (walk up from the working directory) | Pin the `api/` source tree the build fingerprint is computed from. |

## Ink slots and the brand palette

A style's treatment parameters name inks by **slot**, not by hex. Slot names are
exactly the token names Brand Manager's `BrandsService.GetTokens` emits, so
there is no mapping table between the two vocabularies and nothing that can
drift out of sync.

| Slot | Brand Manager token | Typical use in a treatment |
|---|---|---|
| `$brand.primary` | `colors.primary` | the dark ink of a two-ink ramp; a scrim colour |
| `$brand.secondary` | `colors.secondary` | a third ink in a tritone ramp |
| `$brand.accent` | `colors.accent` | a spot ink, usually one small high-chroma area |
| `$brand.background` | `colors.background` | the light ink — the "paper" a screen prints on |
| `$brand.surface` | `colors.surface` | a raised panel behind a framed inset |
| `$brand.text` | `colors.text` | overlay copy colour the legibility gate measures |
| `$brand.error` | `colors.error` | reserved; no seeded style uses it |

The enforcement is in code, not in this table: `catalog.BrandSlots` is the one
list, `validateStyle` refuses a style naming a slot outside it, and
`TestEverySeededSlotHasADeclaredDefault` fails the build if a seeded style
references a slot it cannot bind.

### Resolution order

A slot resolves against the **effective palette**, which is built once per
render:

1. the style's own declared `inks` — its art-directed defaults;
2. overlaid by the bound brand's tokens, when a brand is bound.

This is what lets the same style render on a cold install with nothing
configured *and* render differently per brand. It also means the two are not
alternatives: a brand that defines only `primary` overrides only that slot and
leaves the style's other inks intact.

### Unresolved slots fail closed

A slot that neither the style nor the brand defines is a
`FailedPrecondition` naming the slot, the operation and the field. It is never
written onto the wire as a literal.

That fall-through is not hypothetical: it is what shipped, and image-tools
answered `422 invalid color "$brand.primary"` for ten of the sixteen styles seeded at the time
while every unit test passed — because the tests all bound a brand, which is the
one thing a CLI caller never did.

### Binding a brand from the CLI

```bash
# Render from the style's own ink defaults — no brand needed.
backdrop-studio render submit --style cyanotype-arcade --seed 7

# Bind a Brand Manager brand; its tokens override the style's defaults.
backdrop-studio render submit --style cyanotype-arcade --seed 7 --brand <brand-id>

# Bind slots directly, for a caller with no Brand Manager.
backdrop-studio render submit --style cyanotype-arcade --seed 7 \
  --brand-token primary=#1B3FBF --brand-token background=#F5EFDC
```

`--brand-token` accepts the slot with or without the `$brand.` prefix. Explicit
tokens win over `--brand`, because a caller who states an ink means it.

## Catalog seed versions and upgrades

Seed content is versioned data under `api/internal/catalog/seed/`, embedded into
the binary with `go:embed`. On every start the store applies each version it has
not applied yet, upserting rows by id.

- A row whose `origin` is `operator` is **never** overwritten and never deleted.
- A row whose `origin` is `seed` is upgraded in place.
- A style dropped from a later seed version stays on installs that already have
  it, because a released backdrop may reference it.

Check what an install is running:

```bash
curl -s "http://localhost:${API_PORT}/api/v1/build" | jq
```

`seed_version` is what the binary ships; `applied_seed_version` is what the
database has. They differ only between an upgrade and the next restart.

## The perceptual quality gate

Every candidate is scored against the source it was made from, after the
treatment chain and before it becomes a job record. A candidate that fails is
refused with `FailedPrecondition` naming the metric and the value; a candidate
that passes carries its scores on `quality_json`, visible in `render get`.

The gate exists because `engraved-colonnade` shipped illegible moire past a
fully green test suite. The legibility check measures overlay text contrast,
which high-contrast noise passes easily — contrast is not legibility.

### The metrics

| Metric | Measures | Catches |
|---|---|---|
| `subject_survival` | Absolute correlation between the source's and the result's 64x36 low-frequency lightness fields | A treatment that erased the composition it was screening |
| `tonal_occupancy` | The p2–p98 lightness span the result occupies | A dither or quantiser that collapsed the frame into one flat ink |
| `frequency_modulation` | Standard deviation of per-block ink coverage across a 16x9 grid, in L\* units | Texture that is equally busy everywhere — noise wearing the costume of a screen |
| `reserved_quiet` | Texture inside the style's reserved regions relative to the whole frame | A treatment that concentrated its detail exactly where the headline goes |

`subject_survival` takes the *absolute* correlation on purpose: a duotone
changes every pixel's colour and no pixel's structure, and an inverted print
preserves the composition just as faithfully. Both must read as survival or
every legitimate treatment fails.

`reserved_quiet` skips regions marked `decorative` — a decorative region is not
a place text has to be readable, so measuring it would reject candidates for a
problem that does not exist.

### Thresholds

Bars are per treatment *family*, because nine screens that fail the same way
need one number rather than nine. A chain spanning families takes the strictest
bar on each metric: its output has to be good enough for the most demanding
thing in it.

| Family | Treatments | survival ≥ | occupancy ≥ | modulation ≥ | quiet ≤ |
|---|---|---|---|---|---|
| `screen` | halftone, line_screen, stipple, engraving, ascii_mosaic, dither_ordered, dither_diffusion | 0.60 | 0.40 | 0.030 | 1.60 |
| `tonal` | duotone, posterize, curve, grain, scrim | 0.80 | 0.40 | 0.030 | 1.60 |
| `optical` | bloom, defocus, motion_blur, aberration, displacement, pixel_sort | 0.70 | 0.35 | 0.030 | 1.60 |

A style overrides any of these by setting `quality` on its catalog record. A
positive value replaces the family default; a **negative** value disables that
one metric, which is how a deliberately extreme style opts out of one bar
without opting out of all of them. A style that opts out of every metric has no
gate at all, so a reason belongs beside it in the seed data.

### How the numbers were chosen

Every bar is derived from measurement, not taste. The whole catalog was rendered
at seed 7 and scored; the observed floors were:

| Metric | Worst observed | Style | Bar |
|---|---|---|---|
| `subject_survival` | 0.8585 | `ascii-field` | 0.60 |
| `tonal_occupancy` | 0.6142 | `cyanotype-arcade` | 0.40 |
| `frequency_modulation` | 0.0517 | `ascii-field` | 0.030 |
| `reserved_quiet` | 1.2735 (ceiling) | `riso-horizon` | 1.60 |

The headroom is deliberately wide. These metrics separate *working* from
*destroyed* by a wide margin — a treatment that erases its subject scores below
0.2 on survival and near zero on modulation — so a bar placed just under the
observed floor would police art direction rather than catch defects. The gate's
job is to make "unusable" impossible to ship, not to enforce a house style.

**Tuning rule.** If a style needs a threshold outside these ranges twice, the
check is wrong, not the style: record it in `docs/internal/PROBLEMS.md` rather
than widening the range silently.

### What it does not prove

Automated metrics prove a treatment did not destroy its subject. They do not
prove an image is beautiful. This gate is necessary and not sufficient, and the
human verdict on a catalog change remains required.

The corpus at `docs/evidence/perceptual/corpus.json` records where every style
sits relative to its bar, so a change that quietly moves a style *toward* the
floor is visible before it falls through. The integration lane fails when a
metric moves more than 0.05 from its recorded value — renders are deterministic,
so movement is a change in the code or the catalog, never noise.

## Build fingerprint

The same endpoint reports `fingerprint`, a content hash over the API's Go
sources, embedded seed data and SQL schema. `/health` carries the first twelve
characters as semver build metadata (`1.0.0+82eb6be45d76`).

It exists because two audits in two days drew false conclusions from a stale
binary — one recorded a working feature as missing. The integration lane
computes the same hash from the working tree and refuses to render anything on a
mismatch.

The browser UI does not read `API_PORT` directly. It resolves API calls through
the UI origin, and `ui/server.js` proxies `/api/*` plus the scenario's Connect
RPC namespace to the API process using the lifecycle-provided `API_PORT`.

### Scenario-prefixed CLI variables

`cli-core` derives a standard set of env vars from the scenario name.
For `backdrop-studio` the prefix is the scenario id upper-cased with
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
2. `${XDG_CONFIG_HOME}/vrooli/backdrop-studio/config.json`
3. `~/.vrooli/config/backdrop-studio/config.json`
4. `~/.config/vrooli/backdrop-studio/config.json`

File shape:

```json
{
  "api_base": "http://localhost:15001/api/v1",
  "token": "optional-auth-token"
}
```

Set values via the CLI rather than editing the file directly:

```bash
backdrop-studio configure api_base http://localhost:15001/api/v1
backdrop-studio configure token "<token>"
```

## API-base resolution precedence

When the CLI calls the API, the base URL is resolved in this order
(first match wins):

1. `--api-base <url>` flag
2. Scenario-prefixed env vars (above)
3. CLI config file (`api_base` field)
4. Vrooli lifecycle port detection (`vrooli scenario port backdrop-studio API_PORT`)
5. Compile-time default (only set if explicitly configured in `app.go`)

If none of these resolve, the command exits with an actionable error
("API not available — try `--auto-start` or `vrooli scenario start
backdrop-studio`").

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
