# Seams - Tech Tree Designer

## Wire contracts live in proto, not seams

Proto schemas under `packages/proto/schemas/tech-tree-designer/` define wire contracts. This document records substitutable boundaries inside the implementation.

## Workflow transitions are not seams

Future planned-proto editor workflows may use formal transition models. Those models are not service seams unless production code swaps implementations through an interface.

## Current seams

| Seam | Interface | Production Wiring | Test Fake | Why It Exists |
|---|---|---|---|---|
| health dependency probes | api-core health builder pingers | `handlers/health.Module` receives routed DB handle | handler tests inject test DB / health builder setup | Keeps runtime readiness observable without product tables. |
| database routing | `*database.RoutedDB` | lifecycle and test-mode middleware choose primary/test pool | `internal/testutil/db` SQLite helpers | Lets test-genie route test data without direct DB sharing. |
| GraphSource | `api/internal/graph.GraphSource.Graph(ctx, SourceRequest)` returning `TechTreeGraph` | `ProtoHealthSource` backed by `proto-health` `DescribeScenariosProtos`; `graph.Service` consumes it for Describe/Neighborhood/Path/Ancestors/Export | fake `ProtoSurfaceClient` fixtures and fake `GraphSource` service tests | Keeps graph logic independent of upstream provider shape and leaves room for the later SDA source. |
| PlannedGraphSource | `api/internal/graph.PlannedGraphSource.PlannedGraph(ctx)` | planning service returns planned nodes and import-derived edges | fake planned graph source in graph service tests | Lets graph merge planned nodes without importing planning storage. |
| planning Repository | `api/internal/planning.Repository` | SQLiteRepository over `planned_scenario` and `planned_proto_file` | small package-local fakes in service tests | Keeps planning service independent of SQLite query mechanics. |
| ProtoValidator | `api/internal/planning.ProtoValidator` | `CompilerValidator` using `bufbuild/protocompile` with planned-file overlay plus live schemas | deterministic fake validator when service-only tests need it | Validates planned proto text without shelling out to `buf` at runtime. |
| Materializer | `api/internal/planning.Materializer` | `FilesystemMaterializer` writes under `packages/proto/schemas/<slug>/` and runs `make generate` | command-injected/temp-dir materializer tests | Makes the outside-scenario write path explicit and testable. |
| ontology Repository | `api/internal/ontology.Repository` | SQLiteRepository over `capability`, `capability_edge`, `fulfillment`, and `coverage_exclusion` | package-local SQLite repository tests | Keeps ontology CRUD and coverage reads independent of SQLite query mechanics. |
| ontology ScenarioSource | `api/internal/ontology.ScenarioSource` | graph service `Describe` result including live and planned nodes | fake scenario source in ontology service tests | Lets coverage and overlay analytics read implementation state without owning graph topology. |

## Adding a new seam

Add a seam only when production code genuinely swaps behavior behind an interface or external system boundary. Keep fakes close to the owning domain.

## Architecture Alignment Notes

The old Gin/Postgres code is intentionally gone. Do not recreate compatibility seams for deleted handlers, migrations, or heuristic scenario catalog code.

## UI-side seams

Current UI seams are test utilities only. Future graph/planning UI should keep API clients in `ui/src/api/` and feature mocks under the owning feature folder.

## What is NOT a seam

- `api/internal/modules` registry entries.
- Generated proto types.
- Static capability kind labels.
- D3 rendering helpers unless they hide an external dependency.

## API contract manifest

`.vrooli/endpoints.json` is generated from API descriptors and CLI seed data. Regenerate it; do not hand-edit it.

## Cross-references

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
- [`PROBLEMS.md`](PROBLEMS.md)
