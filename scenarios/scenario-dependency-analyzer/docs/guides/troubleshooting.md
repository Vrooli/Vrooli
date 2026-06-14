# Troubleshooting

## Lifecycle and ports

Use lifecycle commands to discover active ports. Avoid direct binary execution.

## API and CLI

If CLI commands fail, verify the API is healthy and the CLI was reinstalled after Go changes.

## Build and dependencies

Use pnpm for the UI. A single lockfile, `pnpm-lock.yaml`, is expected.

## Tests

Use `vrooli scenario test scenario-dependency-analyzer` for full validation and focused Go/UI commands while iterating.

## Storage

Check `SQLITE_PATH` and the API logs when persistence fails.

## Proto codegen

Run proto lint/codegen workflows after editing files under `packages/proto/schemas`.

## When to add a new entry here

Add entries for repeated operational failures or phase failures that future agents are likely to hit.

## Cross-references

- `../operations/RUNBOOK.md`
- `../reference/configuration.md`
