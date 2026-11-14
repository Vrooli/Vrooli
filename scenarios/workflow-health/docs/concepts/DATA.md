# Data — Workflow Health

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

Workflow Health uses embedded SQLite through `modernc.org/sqlite` for
scenario-local catalog snapshots, validation runs, fix previews, and run
metadata. Browser Automation Studio remains the source of browser execution
artifact files; workflow-health stores references and summaries.

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

Each domain owns its persisted data. The `health` domain owns no product
data. The catalog domain owns normalized facts derived from target scenario
files; source workflow JSON remains in the target scenario. Execution owns
workflow-health run state and artifact pointers, not BAS artifact bytes.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Catalog snapshot | catalog | SQLite | `api/internal/workflows/schema.sql` | Replaced on rescan; retained with validation/run history until cleanup policy deletes old runs. | Derived from target scenario `bas/` files. |
| Asset facts | catalog | SQLite | `api/internal/workflows/schema.sql` | Same lifecycle as catalog snapshot. | Stable IDs, paths, roles, metadata, refs, and dependency edges. |
| Validation run | validation | SQLite | `api/internal/validation/schema.sql` | Retained for operator history and Test Genie artifact references. | Shared response remains authoritative. |
| Fix preview | remediation | SQLite | `api/internal/fixes/schema.sql` | Expires after apply, supersede, or retention cleanup. | Stores deterministic diff plans, not behavioral edits. |
| Workflow run | execution | In-memory result + artifact JSON now; SQLite planned | `api/internal/execution` | Retained with artifact references; cleaned by run retention policy. | Includes safety verdict and BAS execution IDs. |
| Artifact reference | execution | Target scenario `coverage/workflow-health/runs/**` now; SQLite + BAS output directory planned | `api/internal/artifacts` | Metadata retained with run; bytes follow BAS artifact retention. | Timeline/latest summaries now; screenshots/video/console/network/page-error references planned. |
| Search leaf metadata | search | In-memory response now; Search Hub provider cache after registration | `api/internal/search`, `.vrooli/search.json` | Rebuilt from latest catalog and run signals. | Publishes typed leaves through `WorkflowSearchService.SearchWorkflows`, `workflow-health workflows search`, and Search Hub self-registration. |

## Workflow Catalog Model

The first implemented catalog surface is an in-memory scanner in
`api/internal/workflows`. It treats source workflow JSON as immutable input
and emits deterministic `ScenarioWorkflowCatalog` records for later
validation, execution, search, and UI domains.

| Shape | Purpose | Source |
|---|---|---|
| `ScenarioWorkflowCatalog` | Scenario-level snapshot with registry metadata, assets, seeds, dependency edges, and stale registry paths. | Target scenario `bas/` tree. |
| `WorkflowAsset` | Shared normalized facts for BAS cases, flows, actions, seeds, and registry-only entries. | `bas/cases`, `bas/flows`, `bas/actions`, `bas/seeds`, `bas/registry.json`. |
| `WorkflowCase` | Validation evidence asset; these are the only assets that feed the workflow validation phase by default. | `bas/cases/**/*.json`. |
| `WorkflowFlow` | Agent-discoverable user journey candidate exposed as `workflow.flow` leaves with safety metadata. | `bas/flows/**/*.json`. |
| `WorkflowAction` | Reusable dependency fragment; hidden from default action search unless explicitly requested. | `bas/actions/**/*.json`. |
| `SeedContract` | Deterministic state setup entrypoint facts. | `bas/seeds/**`. |
| `RequirementLink` | Requirement traceability from `requirements/**` validation refs, with workflow metadata as a legacy fallback. | `requirements/index.json`, requirement modules, workflow metadata labels. |
| `SelectorRef` | Selector references used by workflow nodes, including `@selector/...` tokens and direct selector fields. | BAS node JSON. |
| `RouteRef` | Scenario route references used by navigate nodes. | Current `action.navigate.scenario_path` and legacy `data.scenarioPath` shapes. |
| `SafetyProfile` | Static safety summary derived from `execution_mode`, reset labels, and confirmation metadata. | Workflow metadata. |
| `DependencyEdge` | Subflow/fixture graph between workflows. | Current `action.subflow.workflow_path`, `action.subflow.workflowPath`, legacy `data.workflowPath`, and `@fixture/<slug>` nodes. |

Stable asset IDs use `<scenario>:<relative-path>`, for example
`workflow-health:bas/flows/perf-example-scroll.json`. This keeps findings,
search leaves, and run artifacts stable across machines while preserving the
canonical source path.

## Execution And Artifact Model

`api/internal/execution` now owns the first executable slice. It composes the
static validation engine with a narrow BAS client seam and always runs safety
preflight before any BAS call. Static-only requests return the selected assets
without touching BAS. Observer cases can execute immediately. Mutating cases
and flows refuse before BAS unless the caller provides explicit mutating
confirmation and a routed-isolation proof; this keeps destructive workflows
fail-closed while routed lease installation is still being moved from Test
Genie.

The service writes per-asset summaries through `api/internal/artifacts`.
Current artifact roots live in the target scenario, not workflow-health:

```text
<target-scenario>/coverage/workflow-health/runs/<run-id>/<asset-id>/
  latest.json
  timeline.json
```

`latest.json` records the workflow asset path, BAS execution ID, terminal
status, success flag, timing, and timeline counts. `timeline.json` stores the
BAS proto timeline JSON when available. Later phases can add screenshots,
video, trace, HAR, console/network/page-error projections, and SQLite indexes
without changing the safety gate.

## Search Leaf Model

`api/internal/search` turns the catalog into deterministic typed leaves:

| Leaf Type | Source | Default Visibility | Purpose |
|---|---|---|---|
| `workflow.flow` | `bas/flows/**/*.json` | yes | Agent/operator runnable journey candidates. Observer flows can be marked runnable; mutating flows carry confirmation and isolation guardrails. |
| `workflow.test` | `bas/cases/**/*.json` | yes | Validation evidence for prove, validate, check, and requirements queries. These are not default agent actions. |
| `workflow.fragment` | `bas/actions/**/*.json` | only with `--type workflow.fragment` or `--include-fragments` | Reusable dependency fragments for explain/compose views. |

Search ranking is deterministic. Run/do/execute-style queries prefer
`workflow.flow` results and give safe observer flows a small boost.
Validate/prove/test-style queries prefer `workflow.test` results and
requirement-linked assets. Exact text matches still outrank generic safety
boosts, so a specific mutating intent returns the matching mutating flow with
guardrails instead of hiding it behind unrelated observer flows.

## Validation And Fix Model

`api/internal/validation` is currently an in-memory static analysis layer over
the catalog model. It returns a `Report` containing the scanned catalog and
stable `Finding` records. Every finding code is declared in
`.vrooli/maturity.json`, which is loaded through `maturity-go` before shared
`MaturityAssessment` construction.

The first rule set covers static workflow readiness:

| Rule | Purpose | Fix Posture |
|---|---|---|
| `workflow.surface_absent` | No BAS workflow surface exists. | Manual. |
| `workflow.registry_missing` | Cases exist without `bas/registry.json`. | Auto: rebuild registry from cases. |
| `workflow.registry_stale` | Registry references missing cases. | Auto: rebuild registry from cases. |
| `workflow.parse_error` | Workflow JSON cannot parse. | Manual. |
| `workflow.metadata_incomplete` | Workflow lacks name or description. | Auto: fill deterministic stubs. |
| `workflow.requirement_unlinked` | Validation case has no requirement link. | Manual traceability decision. |
| `workflow.selector_unregistered` | Selector bypasses `@selector/` registry indirection. | Manual UI contract decision. |
| `workflow.subflow_unresolved` | Subflow or fixture dependency does not resolve. | Manual workflow repair. |
| `workflow.execution_mode_invalid` | `execution_mode` is not `observer`, `mutating`, or `destructive`. | Auto: normalize to `observer`. |
| `workflow.observer_content_unsafe` | An observer-labeled workflow, including its resolved subflows, contains a non-read-only action. | Auto: relabel to `mutating`. |
| `workflow.reset_legacy` | Legacy `reset=database` is present. | Auto: normalize to `full`. |
| `workflow.mutating_safety_missing` | Mutating workflow lacks confirmation or routed isolation metadata. | Manual safety decision. |
| `workflow.seed_missing` | Full-reset mutating workflow lacks seed/fixture setup. | Manual data safety decision. |
| `workflow.execution_refused` | Runtime execution was refused by fail-closed workflow safety policy. | Manual operator proof or request change. |
| `workflow.execution_failed` | BAS validation or execution did not complete successfully. | Manual workflow/runtime repair. |

Fix application composes rule-by-rule so multiple mechanical edits to the same
workflow JSON do not overwrite each other.

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| workflow catalogs | catalog | `api/internal/workflows/schema.sql` | scanner, validation, search, UI inventory |
| validation runs | validation | `api/internal/validation/schema.sql` | provider handler, findings UI, Test Genie native detail |
| workflow runs | execution | `api/internal/execution/schema.sql` | execution service, artifact UI, provider execution detail |
| workflow fixes | remediation | `api/internal/fixes/schema.sql` | fix preview/apply API, CLI, UI |
| workflow search leaves | search | `api/internal/search` and `packages/proto/schemas/workflow-health/v1/workflows/workflows.proto` | search provider, workflows search CLI |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

## Migrations And Compatibility

The generated template uses idempotent schema bootstrap. Domain schema
files should use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| `workflow-health workflows search "<query>" --json` | JSON / `SearchWorkflowsResponse` | search | implemented |
| `workflow-health validate scenario <name> --json` | scenario-validation response JSON | validation | implemented |
| `workflow-health fix preview <name> --json` | JSON diff plan | remediation | implemented |
| Search Hub workflow leaves | provider descriptors from `.vrooli/search.json` | search | implemented for workflow-health's own BAS catalog |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Catalog snapshots | New scan supersedes old snapshot or retention cleanup runs. | Keep enough history to explain latest validation/run artifacts. | Exact retention period is deferred until persistence schema is implemented. |
| Validation and workflow runs | Explicit cleanup, scenario data reset, or retention cleanup. | Keep recent operator evidence and Test Genie artifact refs. | Cleanup command is deferred. |
| BAS artifact bytes | BAS artifact retention policy. | Workflow Health stores pointers only. | Cross-scenario artifact retention contract needs implementation-phase confirmation. |
| Fix previews | Apply, supersede, cancel, or retention cleanup. | Short-lived; previews must not outlive the catalog version they were created from. | Preview expiration enforcement is deferred. |

## Privacy Notes

Generated template data is local development data. If a scenario stores
personal, regulated, customer, financial, or sensitive business data,
update this document and [`../internal/SECURITY.md`](../internal/SECURITY.md)
before implementation expands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
