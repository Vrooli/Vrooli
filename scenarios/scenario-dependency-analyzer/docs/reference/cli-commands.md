# CLI Commands

## Global flags (provided by cli-core)

Use the standard Vrooli CLI flags for output, configuration, and help where available.

## Built-in commands (auto-provided by `cli-core`)

Health, status, and help behavior follows shared CLI infrastructure.

## Scenario commands — `notes` (CRUD reference)

SDA commands include `analyze`, `scan`, `dag`, `graph actual`, `drift`, `health`, `freshness`, `deps approved`, `deployment`, `optimize`, and scenario catalog operations. Full command examples remain in `../cli.md`.

## Output contracts

Commands that support automation provide `--json` output. `health <scenario> --json` is the dependency-health producer contract; in this slice it includes Code Facts-backed surfaces, readiness checks, runtime checks for required resources/scenarios, approved dependency governance, pnpm release-age policy validation, a thin security-health dependency-index availability check, and graph drift. Drift severities are `INFO` for declared-only and `WARNING` for actual undeclared scenario use.

Dependency freshness is owned by SDA, not root `vrooli hygiene`. Root hygiene is
an aggregate registry that may present dependency findings, but it must delegate
freshness logic to SDA rather than embedding package topology or shared-package
rules. Per-scenario freshness remains part of `health <scenario>` and uses
Code Facts-discovered dependency surfaces, falling back to conventional
`api`/`cli`/`ui` roots only when Code Facts is unavailable.

Fleet and touched-surface freshness is exposed as
`scenario-dependency-analyzer freshness [--touched|--all] [--concurrency <n>] [--json]`. The
contract is:
- select changed files from git state;
- map touched files to discovered in-repo package/module roots and impacted
  scenario dependency surfaces;
- fan out root `go.mod`/`go.sum` changes to all impacted Go surfaces;
- validate each impacted surface with the same readiness findings emitted by
  `health <scenario>`;
- run surface checks with bounded concurrency while preserving deterministic
  report ordering;
- return machine-readable scenario, surface, status, diff paths, summary counts,
  and next-action fields suitable for root hygiene rendering.

The retired root shared-drift parity floor was: touched selection,
no-touched clean/skip behavior, stale `go.mod`/`go.sum` detection through
`go mod tidy -diff`, optional Go build checks as a separate signal, JSON output
that identifies affected scenarios and module paths, and next-action/fix
commands for deterministic Go tidy repair. SDA freshness supersedes that floor
by covering every discovered Go surface and by routing missing local replace
repair through `scenario-dependency-analyzer deps reconcile`.
Reports classify deterministic tidy drift as automatic and missing local
replace reconciliation as guided preview/apply work so root hygiene can show
users and agents every relevant recovery command instead of only the first fix.

Freshness findings map to the existing maturity capabilities. Go `go.mod`
presence, missing local replace targets, tidy drift, optional build freshness,
Node `package.json`, lockfile, single-lockfile, and local install readiness all
belong to `package_readiness`. Discovery or classification failures belong to
`surface_inventory`. New emitted rule IDs must be added to
`.vrooli/maturity.json` before tests can pass.

Autofix is preview/apply only. Deterministic repairs may be implemented for
specific Go surfaces, starting with local replace reconciliation and `go mod
tidy` for a known module root. Unsupported ecosystems, ambiguous package-root
mapping, optional build failures, and third-party dependency governance changes
remain advisory until SDA has a typed fixer that can prove the write is safe.

Approved dependency governance is exposed through generated Connect-backed commands: `deps approved list`, `deps approved search`, `deps approved explain`, `deps approved validate`, `deps approved triage`, `deps approved findings`, `deps approved usage`, `deps approved propose-records`, `deps approved upsert-batch`, `deps approved security-gaps`, `deps approved approve-observed`, `deps approved widen-range`, `deps approved upsert`, `deps approved approve`, `deps approved deny`, `deps approved remediate`, and `deps approved deny-vulnerable`. `deps approved validate --all --json` returns a first-class fleet rollup with per-scenario results, dependency usage groups, and normalized signal categories such as `direct_runtime`, `direct_dev`, `indirect`, and `lockfile_transitive`. `deps approved triage` groups fleet findings into operator decisions; use `--section security|seeding|ranges|hotspots|expired`, `--limit`, or `--json` for typed output. Its next commands include typed recipes such as `approve-observed <ecosystem>/<package> --from-findings` and `widen-range <ecosystem>/<package> --to-major-line`, both dry-run by default. `deps approved security-gaps` consumes Security Health vulnerability evidence and shows vulnerable package/version/advisory exposures that are not yet covered by SDA denied governance records. `deps approved propose-records --json` turns high-frequency unrecorded direct dependencies into honest draft records, and `deps approved upsert-batch --file proposals.json --dry-run --json` previews atomic registry changes before `--apply`. `deps approved findings --json` filters raw fleet findings, and `deps approved usage <ecosystem>/<package> --json` shows every scenario/surface using one dependency. Mutation commands dry-run by default and require `--apply` to write `.vrooli/dependencies/approved-dependencies.json`; `approve`, `deny`, and recipe commands accept range policy controls so broad decisions are explicit. Security-derived denial commands consume Security Health vulnerability evidence, write `security_denied` range policy records, and do not run scanners inside SDA. Approved dependency records are not an exhaustive allowlist; unrecorded packages produce review guidance in advisory mode and fail in strict mode.

## Adding a new command

Register the command in the relevant CLI domain, add API coverage when needed, update generated docs, and add scenario tests for machine-readable output.

## Cross-references

- `../cli.md`
- `api-endpoints.md`
