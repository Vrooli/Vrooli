# Configuration — Secrets Manager

## Environment Variables

### Required At Runtime

The lifecycle supplies `API_PORT` and `UI_PORT`. The API exits when `API_PORT` is absent.

### Optional Overrides

- `SECRETS_MANAGER_SKIP_DB=true` skips database initialization for constrained development flows.
- `VROOLI_DESKTOP_MODE=true` selects the private desktop SQLite metadata store.
- `VROOLI_SCENARIO_DIR` enables lifecycle-declared receipt-signing configuration when present.

## Service Manifest (`.vrooli/service.json`)

The manifest declares Postgres as the required resource and Claude Code as
optional. Ordinary credentials use the canonical credential authority and do
not add a remote secret-service lifecycle dependency. This is the source of lifecycle port and
resource dependency metadata.

## Schema Bootstrap

Shared Postgres schema application is in `api/postgres_schema.go`. Desktop metadata initialization is in `api/desktop_storage.go`.

## CLI Config File

The CLI owns its API-base configuration through the `configure` command. Prefer lifecycle discovery over a hard-coded URL.

## API-Base Resolution Precedence

An explicit CLI `--api-base` overrides configured discovery. `--auto-start` asks the lifecycle to start the scenario when needed.

## Test/CI Configuration

Use `vrooli scenario test secrets-manager`. Test Genie owns server-side runs. UI coverage is configured in `ui/vite.config.ts`; API and CLI coverage commands are discovered by Unit Health.

## Cross-References

- [Integrations](../concepts/INTEGRATIONS.md)
- [Runbook](../operations/RUNBOOK.md)
