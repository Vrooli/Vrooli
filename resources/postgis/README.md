# PostGIS Resource

Managed PostGIS spatial database runtime for geospatial and location-aware workflows.

## Intent

- Resource ID: `postgis`
- Category: `database`
- Driver: `compose-service`
- Portability tier: `full`

## Use Cases

- Run geospatial queries and storage locally with PostgreSQL-compatible tooling.
- Support location-aware scenarios that need routing, proximity, or map data.
- Provide a reusable spatial database for analytics, operations, and GIS workflows.

## Architecture

This resource is being aligned to the updated `compose-service` structure.

- `resource.json` is the declarative authority for lifecycle, compose orchestration, ports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for PostGIS-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json` and `compose.yaml`
2. rely on the shared `vrooli resource ...` control plane
3. add PostGIS-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/compose`: compose-specific runtime graph helpers
- `cli/internal/topology`: service dependency and readiness semantics
- `cli/internal/runtime`: runtime shaping helpers
- `cli/internal/health`: PostGIS-specific readiness helpers
- `cli/internal/env`: environment export helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install postgis

# Check status through the shared control plane
resource-postgis status
```

Connection defaults:

- Host: `localhost`
- Port: `5434`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for GIS, import, or analysis workflows.
- Keep runtime state rooted in `${RESOURCE_*_DIR}` paths and compose-managed mounts rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/postgis/docs/OPERATIONS.md) as the architecture boundary for future migrations.
