# CLI Commands

## Global flags (provided by cli-core)

Use the standard Vrooli CLI flags for output, configuration, and help where available.

## Built-in commands (auto-provided by `cli-core`)

Health, status, and help behavior follows shared CLI infrastructure.

## Scenario commands — `notes` (CRUD reference)

SDA commands include `analyze`, `scan`, `dag`, `graph actual`, `drift`, `health`, `deps approved`, `deployment`, `optimize`, and scenario catalog operations. Full command examples remain in `../cli.md`.

## Output contracts

Commands that support automation provide `--json` output. `health <scenario> --json` is the dependency-health producer contract; in this slice it includes Code Facts-backed surfaces, readiness checks, runtime checks for required resources/scenarios, approved dependency governance, pnpm release-age policy validation, a thin security-health dependency-index availability check, and graph drift. Drift severities are `INFO` for declared-only and `WARNING` for actual undeclared scenario use.

Approved dependency governance is exposed through generated Connect-backed commands: `deps approved list`, `deps approved search`, `deps approved explain`, and `deps approved validate`. Approved dependency records are not an exhaustive allowlist; unrecorded packages produce review guidance rather than hand-roll pressure.

## Adding a new command

Register the command in the relevant CLI domain, add API coverage when needed, update generated docs, and add scenario tests for machine-readable output.

## Cross-references

- `../cli.md`
- `api-endpoints.md`
