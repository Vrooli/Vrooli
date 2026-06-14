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

## Planned seams

| Seam | Interface | Planned Production Wiring | Test Fake | Why It Exists |
|---|---|---|---|---|
| ProtoValidator | validate planned `.proto` file tree | protocompile overlay resolver over planned files + real schemas | deterministic fake validator | Lets planning service tests cover validation outcomes without invoking compiler in every case. |
| Materializer | write validated proto files + run generation | filesystem writer under `packages/proto/schemas/<slug>/` | temp-dir writer / dry-run | Makes the outside-scenario write path explicit and testable. |

## Adding a new seam

Add a seam only when production code genuinely swaps behavior behind an interface or external system boundary. Keep fakes close to the owning domain.

## Architecture Alignment Notes

The old Gin/Postgres code is intentionally gone. Do not recreate compatibility seams for deleted handlers, migrations, or heuristic scenario catalog code.

## UI-side seams

Current UI seams are test utilities only. Future graph/planning UI should keep API clients in `ui/src/api/` and feature mocks under the owning feature folder.

## What is NOT a seam

- `api/internal/modules` registry entries.
- Generated proto types.
- Static roadmap tier constants.
- D3 rendering helpers unless they hide an external dependency.

## API contract manifest

`.vrooli/endpoints.json` is generated from API descriptors and CLI seed data. Regenerate it; do not hand-edit it.

## Cross-references

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
- [`PROBLEMS.md`](PROBLEMS.md)
