# Runbook

## Purpose Of This Document

Provide operator procedures for SDA.

## Start / Stop / Status

Use `make start`, `make stop`, `make logs`, and `make status` from the scenario directory, or the corresponding `vrooli scenario` lifecycle commands.

## Common Incidents

Missing actual graph evidence usually means `proto-health` or `code-facts` is unhealthy or stale.

## Backup / Restore

Back up the configured SQLite path if historical analysis data matters. Most graph state can be regenerated.

## Maintenance Tasks

Regenerate proto artifacts after schema changes and keep UI dependencies locked with pnpm.

## Escalation

Use scenario logs and test-genie phase artifacts to isolate failing domains.

## Cross-References

- `OBSERVABILITY.md`
- `../guides/troubleshooting.md`
