# Start Here

Fall Foliage Explorer is a lifecycle-managed scenario for tracking foliage status, predicting peak dates, accepting user reports, planning trips, and exporting route data. Its product contract is [DOC: ../PRD.md#operational-targets].

## Documentation Map

- Runtime orientation: [DOC: docs/concepts/ARCHITECTURE.md]
- API contract: [DOC: docs/reference/api-endpoints.md]
- CLI contract: [DOC: docs/reference/cli-commands.md]
- Configuration and ports: [DOC: docs/reference/configuration.md]
- Operations: [DOC: docs/operations/RUNBOOK.md]
- Requirement registry: [DOC: requirements/README.md]
- Known problems: [DOC: docs/internal/PROBLEMS.md]

## Code Map

- [CODE: api/main.go] - Go HTTP API, database reads/writes, prediction fallback, and server wiring.
- [CODE: ui/src/app.js] - Browser application, app-monitor iframe bridge, API calls, map, tabs, reports, trips, and exports.
- [CODE: ui/src/styles.css] - Responsive autumn-themed UI styling.
- [CODE: cli/app.go] - Scenario CLI root command and metadata.
- [CODE: cli/domains/foliage/register.go] - CLI foliage status, prediction, and weather commands.
- [CODE: cli/domains/reports/register.go] - CLI report list and submit commands.
- [CODE: cli/domains/trips/register.go] - CLI trip list and save commands.
- [CODE: initialization/postgres/schema.sql] - PostgreSQL tables and seed data.

## Safe Change Rules

Use `make start`, `make test`, `make logs`, and `make stop` from the scenario root, or the equivalent `vrooli scenario ...` commands. Do not start `api/fall-foliage-explorer-api` or `ui/server.js` directly because direct execution bypasses lifecycle port allocation, logging, and health tracking.
