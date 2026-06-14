# Domains - Tech Tree Designer

This document is the domain map for the regenerated Tech Tree Designer. Phase 1 removes the template notes domain; future implementation phases add the product domains below.

## Purpose Of This Document

Use this document to answer which bounded context owns each capability, data set, proto surface, CLI command, UI feature, and test seam.

## Domain Inventory

| Domain | Purpose | Owns Data | Surfaces | Source Paths | Status |
|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | No product data. | API, UI, cli-core `status`. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/tech-tree-designer/v1/health/` | Implemented scaffold surface. |
| graph | Build and query the scenario-centric interface graph. | Optional cache only. | Connect API, CLI, UI graph. | `api/internal/graph/`, `api/handlers/graph/`, `cli/domains/graph/`, `packages/proto/schemas/tech-tree-designer/v1/graph/` | Implemented API/CLI; UI pending. |
| planning | Store future scenarios as real planned proto files and validate/materialize them. | Planned scenarios and planned proto text. | Connect API, CLI, UI editor. | `api/internal/planning/`, `api/handlers/planning/`, `cli/domains/planning/`, `ui/src/features/planning/` | Planned Phase 4. |
| roadmap | Attach sectors, tiers, and milestones as metadata overlays. | Sector, tier, milestone, overlay metadata. | Connect API, CLI, UI roadmap. | `api/internal/roadmap/`, `api/handlers/roadmap/`, `cli/domains/roadmap/`, `ui/src/features/roadmap/` | Planned Phase 5. |

## health

- Purpose: expose API/database readiness and prove the regenerated scaffold can run on SQLite.
- Owns: health endpoint metadata and UI health card.
- Does not own: product graph, planning, roadmap, or persistence policy.
- Source paths: `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/tech-tree-designer/v1/health/`.

## graph

- Purpose: render the actual cross-scenario interface graph.
- Owns: `GraphSource` seam, proto-health mapping, graph query algorithms, graph export, and generated-client CLI commands.
- Primary source: `proto-health` `DescribeScenariosProtos`.
- Future source: `scenario-dependency-analyzer` `DescribeInterfaceGraph`.
- Source paths: `api/internal/graph/`, `api/handlers/graph/`, `cli/domains/graph/`, `packages/proto/schemas/tech-tree-designer/v1/graph/`.
- Planned UI path: `ui/src/features/graph/`.

## planning

- Purpose: let agents design future scenarios as proto contracts before implementation.
- Owns: planned scenario records, planned proto file text, validation findings, materialization governance.
- Planned storage: `planned_scenario` and `planned_proto_file`.
- Planned paths: `api/internal/planning/`, `api/handlers/planning/`, `cli/domains/planning/`, `ui/src/features/planning/`, `packages/proto/schemas/tech-tree-designer/v1/planning/`.

## roadmap

- Purpose: layer sector, tier, and milestone metadata over live and planned scenario nodes.
- Owns: overlay metadata and progress rollups from node stability.
- Does not own: graph topology or scenario fulfillment heuristics.
- Planned paths: `api/internal/roadmap/`, `api/handlers/roadmap/`, `cli/domains/roadmap/`, `ui/src/features/roadmap/`, `packages/proto/schemas/tech-tree-designer/v1/roadmap/`.

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
