# Integrations — Workflow Health

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, catalog, validation, execution, search, remediation | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable; static validation can still explain missing persistence if implemented. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| Browser Automation Studio | scenario | yes for execution | execution, artifacts, validation | BAS workflow execute by file, schema/lint contracts, artifact output directories | Static validation/search still run; execution returns skipped or failed findings depending on request strictness. |
| Test Genie | scenario | yes for phase orchestration | validation provider consumer | `scenario-validation/v1` delegated provider contract | Provider-contract tests fail; workflow phase migration is blocked. |
| Search Hub | scenario | yes for federated discovery | search | `.vrooli/search.json` self-registration for typed leaves `workflow.flow`, `workflow.test`, `workflow.fragment` backed by `WorkflowSearchService.SearchWorkflows` | Local workflow search still works if Search Hub is unavailable; federated AI/action discovery degrades until registration succeeds. |
| Storage Health | scenario | yes for mutating execution proof | execution safety | routed isolation findings and test-pool proof | Mutating workflow execution is refused. |
| Business Health | scenario | yes for contract validation | PRD and requirements | PRD/requirements validation and deterministic fixes | Contract drift is reported before implementation proceeds. |

## Vrooli Resources

Workflow Health should not add shared resources until a domain needs them.
The first implementation uses lifecycle-managed SQLite and scenario calls.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| Routed test database | planned dependency evidence | Required to permit destructive workflow execution. | Execution phase implements mutating fixtures. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| browser-automation-studio | required for execution | Executes workflow JSON and owns browser artifacts. | CLI/API workflow execution by file and artifact directory contract. |
| test-genie | required for orchestration | Delegates the canonical workflow phase to workflow-health. | `ScenarioValidationService` provider contract. |
| search-hub | required for federated discovery | Federates typed workflow leaves. | Provider leaf types and query routing; workflow-health owns the source search response. |
| storage-health | required for mutation safety | Proves routed test isolation before destructive workflow execution. | Finding/maturity signal consumed by workflow-health safety policy. |
| business-health | required for contract validation | Validates PRD and requirements registry. | `vrooli scenario requirements validate workflow-health --json`. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | Generated scenario has no third-party dependency. | Add when PRD/requirements require external APIs, webhooks, auth, payments, or data feeds. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| BAS unavailable | connection error or provider health failure | Static validation/search can continue; execution returns explicit skipped/failed evidence according to request options. | execution fake tests and live observer check |
| Missing routed isolation | storage-health finding or absent safety proof | Mutating workflow execution fails before BAS call with blocker finding. | safety policy tests |
| Search Hub unavailable | provider registration/query failure | `workflow-health workflows search` remains usable locally; federated `search-hub query --type workflow.flow` is unavailable until workflow-health re-registers. | search registration descriptor test and live query smoke |
| Test Genie provider contract drift | provider-contract check failure | Workflow phase migration is blocked. | provider-contract check |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
