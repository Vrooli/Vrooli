# Troubleshooting — Secrets Manager

## Lifecycle And Ports

If the scenario is not healthy, run `make status` and `make logs`. Start it with `make start`; do not run a service binary directly.

## API And CLI

Run `secrets-manager health` or `secrets-manager status`. Use `--auto-start` when appropriate. If a specific running instance is intended, use its lifecycle-discovered API base instead of assuming a port.

## Build And Dependencies

Use the approved scenario dependency workflow for dependency changes. Run UI commands from `ui/` and Go tests from `api/` or `cli/`.

## Tests

Run `vrooli scenario test secrets-manager`. Test Genie owns long runs; retrieve the recorded run evidence after it reaches a terminal state. A UI coverage threshold failure means coverage must be raised rather than the threshold weakened.

## Storage

Desktop mode uses a private SQLite database. Shared mode uses routed Postgres metadata. A locked SQLite database indicates concurrent access or an unclosed cursor; do not delete the database as a first response.

## Vault

If preflight reports missing Secret Service tooling, use `vrooli host install secret-tool` with an authorized interactive session. If bundle staging rejects an artifact, obtain the required detached checksum signature.

## Cross-References

- [Runbook](../operations/RUNBOOK.md)
- [Configuration](../reference/configuration.md)
