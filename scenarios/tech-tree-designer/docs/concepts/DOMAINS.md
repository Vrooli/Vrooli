# Domains - Tech Tree Designer

This document is the domain map for the regenerated Tech Tree Designer. The template notes domain has been removed; graph, planning, and ontology are implemented as product domains.

## Purpose Of This Document

Use this document to answer which bounded context owns each capability, data set, proto surface, CLI command, UI feature, and test seam.

## Domain Inventory

| Domain | Purpose | Owns Data | Surfaces | Source Paths | Status |
|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | No product data. | API, UI, cli-core `status`. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/tech-tree-designer/v1/health/` | Implemented scaffold surface. |
| graph | Build and query the scenario-centric interface graph. | Optional cache only. | Connect API, CLI, UI graph. | `api/internal/graph/`, `api/handlers/graph/`, `cli/domains/graph/`, `ui/src/features/graph/`, `packages/proto/schemas/tech-tree-designer/v1/graph/` | Implemented. |
| planning | Store planned scenarios as real planned proto files and validate/materialize them. | Planned scenarios and planned proto text. | Connect API, CLI, UI editor. | `api/internal/planning/`, `api/handlers/planning/`, `cli/domains/planning/`, `ui/src/features/planning/`, `packages/proto/schemas/tech-tree-designer/v1/planning/` | Implemented. |
| ontology | Own the top-down capability tree, fulfillment links, coverage analytics, focus ranking, and overlay graph projection. | Capabilities, capability edges, fulfillment links, explicit coverage exclusions. | Connect API, CLI, UI ontology. | `api/internal/ontology/`, `api/handlers/ontology/`, `cli/domains/ontology/`, `ui/src/features/ontology/`, `packages/proto/schemas/tech-tree-designer/v1/ontology/` | Implemented. |

## health

- Purpose: expose API/database readiness and prove the regenerated scaffold can run on SQLite.
- Owns: health endpoint metadata and UI health card.
- Does not own: product graph, planning, ontology, or persistence policy.
- Source paths: `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/tech-tree-designer/v1/health/`.

## graph

- Purpose: render the actual cross-scenario interface graph.
- Owns: `GraphSource` seam, SDA graph mapping, graph query algorithms, graph export, and generated-client CLI commands.
- Primary source: `scenario-dependency-analyzer` `DescribeInterfaceGraph`.
- Source paths: `api/internal/graph/`, `api/handlers/graph/`, `cli/domains/graph/`, `ui/src/features/graph/`, `packages/proto/schemas/tech-tree-designer/v1/graph/`.

## planning

- Purpose: let agents design future scenarios as proto contracts before implementation.
- Owns: planned scenario records, planned proto file text, validation findings, materialization governance, and planned graph overlays.
- Storage: `planned_scenario` and `planned_proto_file` in `api/internal/planning/schema.sql`.
- Validation: `ProtoValidator` compiles planned text with `bufbuild/protocompile` against an overlay plus live `packages/proto/schemas`.
- Materialization: writes validated planned text under `packages/proto/schemas/<slug>/` and runs `make generate` in `packages/proto`.
- CLI: `plan create`, `plan list`, `plan tree [path]`, `plan add`, `plan rm`, `plan validate`, `plan materialize`.
- Paths: `api/internal/planning/`, `api/handlers/planning/`, `cli/domains/planning/`, `ui/src/features/planning/`, `packages/proto/schemas/tech-tree-designer/v1/planning/`.

## ontology

- Purpose: keep a top-down capability ontology beside the bottom-up scenario graph.
- Owns: capability records, hierarchy/progression edges, fulfillment links, coverage/focus analytics, and overlay projection.
- Does not own: implementation graph topology or planned proto materialization.
- Storage: `capability`, `capability_edge`, `fulfillment`, and `coverage_exclusion` in `api/internal/ontology/schema.sql`.
- CLI: `ontology capabilities`, `ontology capability`, `ontology import`, `ontology fulfill`, `ontology coverage`, `ontology focus`, `ontology overlay`.
- Paths: `api/internal/ontology/`, `api/handlers/ontology/`, `cli/domains/ontology/`, `ui/src/features/ontology/`, `packages/proto/schemas/tech-tree-designer/v1/ontology/`.

## Non-Domains

- `api/internal/server/` - HTTP composition substrate.
- `api/internal/module/` - shared module and endpoint descriptor types.
- `api/internal/modules/` - thin boot/codegen registry.
- `api/internal/database/` - system schema and DB reachability helpers.
- `ui/src/components/` - shared UI primitives.
- `ui/src/test-utils/` - cross-feature test helpers.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`DATA.md`](DATA.md)
- [`INTEGRATIONS.md`](INTEGRATIONS.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
- [`../../requirements/index.json`](../../requirements/index.json)
