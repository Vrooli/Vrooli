# CLI Commands

## Global flags (provided by cli-core)

Use the standard Vrooli CLI flags for output, configuration, and help where available.

## Built-in commands (auto-provided by `cli-core`)

Health, status, and help behavior follows shared CLI infrastructure.

## Scenario commands — `notes` (CRUD reference)

SDA commands include `analyze`, `scan`, `dag`, `graph actual`, `drift`, `health`, `deps approved`, `deployment`, `optimize`, and scenario catalog operations. Full command examples remain in `../cli.md`.

## Output contracts

Commands that support automation provide `--json` output. `health <scenario> --json` is the dependency-health producer contract; in this slice it includes Code Facts-backed surfaces, readiness checks, runtime checks for required resources/scenarios, approved dependency governance, pnpm release-age policy validation, a thin security-health dependency-index availability check, and graph drift. Drift severities are `INFO` for declared-only and `WARNING` for actual undeclared scenario use.

Approved dependency governance is exposed through generated Connect-backed commands: `deps approved list`, `deps approved search`, `deps approved explain`, `deps approved validate`, `deps approved findings`, `deps approved usage`, `deps approved upsert`, `deps approved approve`, `deps approved deny`, `deps approved remediate`, and `deps approved deny-vulnerable`. `deps approved validate --all --json` returns a first-class fleet rollup with per-scenario results and dependency usage groups. `deps approved findings --json` filters fleet findings, and `deps approved usage <ecosystem>/<package> --json` shows every scenario/surface using one dependency. Mutation commands dry-run by default and require `--apply` to write `.vrooli/dependencies/approved-dependencies.json`. Security-derived denial commands consume Security Health vulnerability evidence and do not run scanners inside SDA. Approved dependency records are not an exhaustive allowlist; unrecorded packages produce review guidance in advisory mode and fail in strict mode.

## Adding a new command

Register the command in the relevant CLI domain, add API coverage when needed, update generated docs, and add scenario tests for machine-readable output.

## Cross-references

- `../cli.md`
- `api-endpoints.md`
