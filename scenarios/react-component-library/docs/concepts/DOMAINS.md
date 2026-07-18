# Domains — React Component Library

This document maps the product capability boundaries for the scenario.

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Source Paths |
|---|---|---|---|---|---|
| components | Index Git-tracked component and hook manifests, versions, dependency pins, catalog metrics, and catalog conformance automation. | catalog | `components`, `component_asset_dependencies`, manifest files, version source paths | API, CLI, UI | `api/internal/components/`, `api/handlers/components/`, `cli/domains/components/`, `ui/src/features/`, `ui/scripts/catalog-conformance.mjs`, `ui/scripts/preview-e2e.mjs` |
| inventory | Scan a scenario's UI inventory to supply adoption recommendations. | analyzer | Derived scan results only; no durable product data | API | `api/handlers/inventory/`, `api/internal/uimanifest/` |
| versions | Browse and diff indexed component release artifacts. | catalog | `component_versions` release snapshots | API, CLI, UI | `api/internal/versions/`, `api/handlers/versions/`, `cli/domains/versions/`, `ui/src/features/versions/` |
| adoptions | Copy a dependency closure into target scenarios and track per-asset provenance/drift. | workflow | `adoption_records`, target scenario files | API, CLI, UI | `api/internal/adoptions/`, `api/handlers/adoptions/`, `cli/domains/adoptions/`, `ui/src/features/adoptions/` |
| deps | Validate component `@deps` declarations against target scenarios. | validator | Dependency declaration rows | API, UI support | `api/internal/deps/`, `api/handlers/deps/`, `ui/src/api/deps.ts` |
| preview | Bundle component source for iframe preview. | workbench | No durable product data | API, UI | `api/internal/preview/`, `api/handlers/preview/`, `ui/src/features/components/` |
| themes | Resolve built-in and scenario-derived theme tokens. | resolver | Built-in theme rows | API, UI | `api/internal/themes/`, `api/handlers/themes/`, `ui/src/features/components/` |
| workflows | Persist and observe assisted extraction/adoption tasks; Agent Manager is execution evidence only. | workflow | `assisted_workflows` | API, CLI, UI | `api/internal/workflows/`, `api/handlers/workflows/`, `cli/domains/workflows/`, `ui/src/components/ActiveWorkMenu.tsx` |
| health | Report runtime readiness. | health | No product data | API, CLI, UI | `api/handlers/health/`, `ui/src/features/health/` |

## Domain Notes

- `components` owns manifest parsing and indexing. Source headers are
  validation hints; `component.json` is authoritative for manifest
  fields and the latest-version `@category` hint is promoted into a
  typed registry column.
- `adoptions` owns filesystem writes to peer scenarios. Apply copies
  full source plus provenance; reapply requires confirmation when local
  edits are detected.
- `versions` now represents version-folder release artifacts, not
  save-history snapshots.
- An adoption resolves a deterministic, deduplicated, version-pinned closure;
  each asset receives its own provenance record and no overwrite is inferred.

## Non-Domains

- `api/internal/server/`, `api/internal/module/`, and
  `api/internal/modules/` are infrastructure.
- `ui/src/components/` contains shared UI primitives, not product
  component-library source.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`DATA.md`](DATA.md)
- [`FLOWS.md`](FLOWS.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
