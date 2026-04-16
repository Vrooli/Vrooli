# QuestDB Resource

Managed QuestDB time-series database for metrics, analytics, and event-ingestion workloads.

## Intent

- Resource ID: `questdb`
- Category: `database`
- Driver: `docker-service`
- Portability tier: `full`

## Use Cases

- Store time-series metrics for AI workloads, resources, and workflows.
- Ingest high-frequency operational events for monitoring and analytics.
- Provide a local analytical database for scenario observability and dashboards.

## Architecture

This resource is being aligned to the updated `docker-service` structure.

- `resource.json` is the declarative authority for lifecycle, runtime, ports, exports, health, and freshness metadata.
- `cli/` is the thin binary entrypoint and delegated command wiring surface.
- `cli/internal/` is the default home for QuestDB-specific Go logic when the manifest and shared control plane are not enough.
- `lib/` still contains retained shell behavior during the migration. That behavior should move into `cli/internal/...` over time rather than back into `cli/main.go`.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add QuestDB-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs resource-local operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/install`: install/bootstrap helpers unique to QuestDB
- `cli/internal/runtime`: runtime and config materialization helpers
- `cli/internal/status`: richer QuestDB status interpretation
- `cli/internal/health`: QuestDB-specific probe helpers
- `cli/internal/env`: environment export and derived-config helpers

## Usage

```bash
# Install or validate the resource contract
vrooli resource install questdb

# Check status through the shared control plane
resource-questdb status
```

Importable workflow artifacts live in `docs/`:

- `questdb-monitoring.n8n.json`
- `real-time-dashboard.node-red.json`

Connection defaults:

- HTTP: `http://localhost:9009`
- PostgreSQL wire: `localhost:8812`
- Influx line protocol: `localhost:9011`

## Notes

- Keep `cli/main.go` thin. Do not treat it as the implementation surface for SQL or table workflows.
- Keep runtime storage rooted in `${RESOURCE_*_DIR}` paths rather than repo-local mutable directories.
- Existing shell-heavy workflows in `lib/` are transitional. New logic should land in Go under `cli/internal/...`.
- Use [docs/OPERATIONS.md](/home/matthalloran8/Vrooli/resources/questdb/docs/OPERATIONS.md) as the architecture boundary for future migrations.
