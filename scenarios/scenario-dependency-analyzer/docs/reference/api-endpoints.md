# API Endpoints

## System

The API exposes health, scenario catalog, declared graph, actual interface graph, drift, deployment readiness, and generated Connect RPC surfaces. Legacy REST details remain in `../api.md`.

## Notes (CRUD reference)

SDA is analysis-oriented rather than CRUD-heavy. Mutating endpoints are limited to scan/apply and generated bundle or deployment artifacts.

## Selected dependency removal

`POST /api/v1/optimize-scenarios` requires one target `scenario` and nonempty
`only_scenarios`. It intersects those exact dependency names with recommendation
`type`; resource removals and other scenario dependencies are excluded. `apply`
defaults to false. The distinct route makes old servers reject selected requests
instead of ignoring the selector. CLI: `optimize <scenario> --only-scenarios a,b`.

## Adding a new endpoint

Add the handler, tests, generated proto/Connect contract when the surface is programmatic, `.vrooli/endpoints.json` metadata, and this reference entry.

## Cross-references

- `../../.vrooli/endpoints.json`
- `../api.md`
- `configuration.md`

## Dependency validation operation budget

The native dependency-health and shared scenario-validation Connect handlers
use the dependency phase `timeout` from `.vrooli/test-genie.json`, captured at
server startup. Their work receives that bounded request context, including
earlier caller cancellation. The shared server operation-budget wrapper clears
the ordinary response write deadline for these mounted services so a completed
validation or timeout can return its receipt. Other API endpoints keep the
ordinary server timeout. A missing, malformed or non-positive descriptor budget
leaves these services explicitly unavailable.

[CODE: scenarios/scenario-dependency-analyzer/api/internal/dependencyhealth/connect.go#loadValidationBudget]
[CODE: packages/api-core/server/operation_budget.go#WithOperationBudget]
