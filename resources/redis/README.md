# Redis Resource

Managed Redis cache and event-bus runtime for local scenario workflows.

## Intent

- Resource ID: `redis`
- Category: `storage`
- Driver: `docker-service`
- Portability tier: `full`

## Use Cases

- Cache hot data for scenario APIs and worker flows.
- Coordinate queues, transient state, and event-driven workflows.
- Provide fast ephemeral storage for automation steps that do not belong in PostgreSQL.

## Architecture

This resource is being aligned to the updated `docker-service` structure.

- `resource.json` is the declarative authority for lifecycle, runtime, health, exports, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Redis-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Redis-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to Redis
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/status`: richer Redis status interpretation
- `cli/internal/health`: Redis-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install using the declarative contract
vrooli resource install redis

# Check status through the shared control plane
resource-redis status

# Connect directly with redis-cli if needed
redis-cli -p 6380
```

Connection defaults:

- Host: `localhost`
- Port: `6380`
- URL: `redis://localhost:6380`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for Redis behavior.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/redis/docs/OPERATIONS.md) as the architecture boundary for future migrations.
