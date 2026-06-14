# CLI Commands

## Global flags (provided by cli-core)

Use the standard Vrooli CLI flags for output, configuration, and help where available.

## Built-in commands (auto-provided by `cli-core`)

Health, status, and help behavior follows shared CLI infrastructure.

## Scenario commands — `notes` (CRUD reference)

SDA commands include `analyze`, `scan`, `dag`, `graph actual`, `drift`, `deployment`, `optimize`, and scenario catalog operations. Full command examples remain in `../cli.md`.

## Output contracts

Commands that support automation provide `--json` output. Drift severities are `INFO` for declared-only and `WARNING` for actual undeclared scenario use.

## Adding a new command

Register the command in the relevant CLI domain, add API coverage when needed, update generated docs, and add scenario tests for machine-readable output.

## Cross-references

- `../cli.md`
- `api-endpoints.md`
