# scripts/resources — legacy resource shell frameworks

This directory holds **legacy bash** that a handful of not-yet-ported resources still source. It is not the resource system's documentation and not an entrypoint.

- Resource documentation: **[/docs/resources/README.md](../../docs/resources/README.md)**
- Resource management: `vrooli resource <install|start|stop|status> <name>`, or the per-resource CLI (`resource-postgres`, `resource-qdrant`, …)
- Resource contract: `.vrooli/schemas/resource.schema.json`, enforced by `internal/resources/validate.go`
- Retirement plan for this directory: **[../README.md](../README.md)**

## What lives here

| Path | Consumer |
|---|---|
| `lib/` | resource `lib/*.sh` in `claude-code`, `codex`, `home-assistant`, `k6`, `minio`, `postgres`, `qdrant` |
| `common.sh`, `common/config-manager.js` | `claude-code`, `minio`, `postgres` (service.json read/write) |
| `populate/` | `lifecycle.setup` step in ~29 scenario `service.json` files |
| `tests/lib/` | `test/integration-test.sh` in `claude-code`, `minio`, `postgres`, `qdrant` |
| `port_registry.{sh,json}` | legacy fallback only — `resource.json` `ports` takes precedence |

Do not add new files here. New resource behavior belongs in the resource's Go CLI (`resources/<name>/cli/`).
