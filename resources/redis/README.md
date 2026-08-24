# Redis Resource

Managed Redis cache and event-bus runtime for local scenario workflows.

## Intent

- Resource ID: `redis`
- Category: `storage`
- Driver: `managed-service`
- Portability tier: `Linux native; Windows amd64 build-verified; macOS unsupported`

## Use Cases

- Cache hot data for scenario APIs and worker flows.
- Coordinate queues, transient state, and event-driven workflows.
- Provide fast ephemeral storage for automation steps that do not belong in PostgreSQL.

## Architecture

This resource uses the managed-service structure. Linux bytes are extracted
from a digest-pinned official OCI image without a container runtime. Windows
amd64 uses the checksum-pinned Redis 8.10.0 MSYS2 archive from the Redis Windows
release channel; Windows ARM and macOS remain explicit unsupported targets.

- `resource.json` is the declarative authority for lifecycle, runtime, health, exports, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Redis-specific Go logic when the manifest and shared control plane are not enough.
- The supported operator surface is the Go CLI and shared control plane; no
  resource-local shell runtime is required.

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

# Back up and restore a key prefix (best effort; quiesce writers for consistency)
resource-redis dump --prefix app: --output /safe/app.redis.json
resource-redis restore --prefix app: --input /safe/app.redis.json

# Connect directly with redis-cli if needed
redis-cli -p 6380
```

Connection defaults:

- Host: `localhost`
- Port: `6380`
- URL: `redis://localhost:6380`

On Windows amd64, `vrooli resource install redis` stages the checksum-pinned
archive and launches `redis-server.exe` from its governed artifact root. A real
Windows host smoke run is still required before promoting the target beyond
build-verified.

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for Redis behavior.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Resource-specific behavior belongs in `cli/internal/...`; runtime shape,
  ports, volumes, and health remain declarative in `resource.json`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/redis/docs/OPERATIONS.md) as the architecture boundary for future migrations.
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
