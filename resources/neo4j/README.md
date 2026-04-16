# Neo4j Resource

Managed Neo4j graph database runtime for local graph and relationship-heavy workflows.

## Intent

- Resource ID: `neo4j`
- Category: `database`
- Driver: `docker-service`
- Portability tier: `full`

## Use Cases

- Build knowledge graphs, relationship maps, and dependency models.
- Run graph queries and traversal-heavy workloads locally.
- Support scenario workflows that benefit from graph-native storage instead of relational tables.

## Architecture

This resource is being aligned to the updated `docker-service` structure.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for Neo4j-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add Neo4j-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to Neo4j
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/status`: richer Neo4j status interpretation
- `cli/internal/health`: Neo4j-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install neo4j

# Check status through the shared control plane
resource-neo4j status
```

Connection defaults:

- HTTP: `http://localhost:7474`
- Bolt: `localhost:7687`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for graph query or import workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/neo4j/docs/OPERATIONS.md) as the architecture boundary for future migrations.
