# Data — React Component Library

This document is the canonical storage map for the scenario.

## Storage Overview

React Component Library stores canonical component source in Git and
uses SQLite as an indexed registry and adoption ledger.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Asset manifests | components | `library/components/<slug>/component.json` or `library/hooks/<slug>/component.json` | Git-tracked file | Until asset is removed | Manifest owns `libraryId`, asset kind, display metadata, tags, version pointers, and pinned library-asset dependencies. |
| Component version source | components / versions | `library/components/<slug>/versions/<version>/` plus indexed SQLite snapshot | Git-tracked file; SQLite stores content hash/content for stable reads | Released versions are immutable; drafts may change | A version is one entry `.tsx` plus optional same-folder `.ts`/`.tsx` companions. `component.json.entry` selects the entry when more than one `.tsx` exists; subdirectories are rejected. |
| Component registry | components | SQLite `components` | Manifest indexer | Rebuilt by `components index` | Soft-referenced by other domains through component id/library id. Includes the indexed adoption `slot` and latest-version `category` facet. |
| Component versions | versions | SQLite `component_versions`, `component_version_files` | Manifest indexer | Rebuilt by `components index` | `component_versions` mirrors the entry for legacy reads; `component_version_files` is the authoritative per-file path/content/hash set and carries a per-file `slot` so multi-file versions place each file independently. The table is single-owner: the former mirrored `internal/versions` schema was deleted so `internal/components/schema.sql` is the sole DDL. |
| Reviewed template divergences | adoptions | `api/internal/adoptions/testdata/reviewed-template-divergences.json` | Human review (Git-tracked) | Until the divergence is resolved | Dated allowlist of explicitly-accepted mismatches between the react-vite template's vendored copies and catalog latest (path, library id, vendored version, catalog version, status, reason, report pointer). The parity test fails for any un-listed behind/deprecated copy and for stale entries, so the allowlist is an audit trail, not a suppression. |
| Component examples | components | `library/components/<slug>/versions/<version>/examples.json` plus SQLite `component_examples` | Git-tracked examples file | Rebuilt by `components index` | Data-only named states for preview. Each example has a stable `name`, display label, JSON `props`, optional declared `controls`, optional `setup`, and optional `expect[]` assertions. Controls only describe text, number, boolean, and select editing; they cannot execute code. Non-serializable props use `$` vocabulary values such as `{ "$text": "Save" }`; the preview harness resolves them at render time. |
| Preview-session props overrides | preview host / harness | Component-editor React state and one registered iframe's memory | Volatile user input over the indexed example's `props` | Cleared by Reset, reload, navigation, or unmount | A valid JSON object is shallow-merged over the selected example's indexed props and resolved through the same `$` vocabulary. It is never written to `examples.json`, SQLite, localStorage, source, or an RPC. `setup` remains indexed data only; it has no runtime semantics in this workspace. |
| Component headers | components | SQLite `component_headers` | Latest version source header | Rebuilt by `components index` | Stores non-structural header metadata. Identity, version, category, and deps are stored in typed columns/tables instead; conflicting structural header hints emit `header_disagreement` findings instead of owning projection facts. |
| Component design affinities | components | SQLite `component_design_affinities` | `component.json` `designStyles[]` | Rebuilt by `components index` | Component-scoped, reconciled against `templates/design/*/metadata.json`, and carries optional rationale text for search, adoption workflows, and UI. Unknown style IDs emit non-fatal staleness findings so catalog drift is visible without dropping the component. |
| Adoption records | adoptions | SQLite `adoption_records`, `adoption_files` | Apply/reapply service | Until deleted | One direct parent record owns a materialized closure. Each `adoption_files` row records its originating asset/library/version as authoritative provenance, enabling direct and effective usage projections without filename inference; old rows with no deterministic origin remain explicitly unknown. |
| Assisted workflows | workflows | SQLite `assisted_workflows` | Workflows service | Until deleted by retention policy | RCL-owned linkage from asset/source/target/idempotency key to Agent Manager task/run, status, event sequence, summary/error, and timestamps. |
| Adopted component files | adoptions | Target scenario filesystem | Target scenario owns local edits after apply | Until scenario changes/deletes file | Files receive `@vrooliComponent*` provenance comments. |
| Dependency declarations | deps | SQLite `component_dep_declarations` | Version source header `@deps` | Rebuilt by index | Stores one row per component version, dependency name, range, and kind (`runtime`, `peer`, `dev`) for adoption validation. |
| Adoption suggestions | adoptions | Derived at request time | Inventory surfaces + component registry + validation verdicts + adoption ledger | Not persisted | Suggestions carry their inventory path and human-readable reason strings; existing adoption pairs are excluded. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `components` | components | `api/internal/components/schema.sql` | Registry list/get, apply lookup, preview |
| `component_versions` | components | `api/internal/components/schema.sql` (single-owner; the mirrored `api/internal/versions` schema was removed) | Version browsing, apply/reapply, diffs |
| `component_version_files` | components | `api/internal/components/schema.sql` | File tabs, file-scoped content RPCs, per-file slot placement, preview/adoption units |
| `reviewed-template-divergences.json` | adoptions | `api/internal/adoptions/testdata/` | Template origin-parity test (allowlisted divergences + calibration) |
| `component_examples` | components | `api/internal/components/schema.sql` | Example browsing, preview harness inputs, component spec reconciliation |
| `component_headers` | components | `api/internal/components/schema.sql` | Component get for non-structural latest-version metadata |
| `component_design_affinities` | components | `api/internal/components/schema.sql` | Component get/list, style/affinity search, UI badges |
| `component_asset_dependencies` | components | `api/internal/components/schema.sql` | Deterministic preview/adoption closure resolution |
| `adoption_records` | adoptions | `api/internal/adoptions/schema.sql` | Apply/reapply/list/refresh |
| `adoption_files` | adoptions | `api/internal/adoptions/schema.sql` | Per-file adoption paths, snapshots, and drift detail |
| `assisted_workflows` | workflows | `api/internal/workflows/schema.sql` | Assisted-work header, CLI, and Agent Manager status reconciliation |
| `component_dep_declarations` | deps | `api/internal/deps/schema.sql` | Dependency validation and dep-aware preview import maps |

## Migration Policy

This scenario is greenfield, but local development data is still
preserved during normal hardening. Schema files are desired-state
bootstrap files using `CREATE TABLE IF NOT EXISTS`; when a pre-existing
SQLite table needs a new column, index, or primary-key shape, apply a
one-shot `/tmp/react-component-library/migrate-*.sql` migration with the
scenario stopped. Do not recreate the database just to satisfy schema
drift.

## Preview-session boundary

The editor treats the indexed example as the immutable baseline for every
specimen. A Try props experiment is intentionally not a second example and is
not an edit to the component: the host sends a data-only object to the matching
iframe, the harness shallow-merges it over indexed `props`, and Reset restores
the indexed object. The existing `setup` field is visible in the index contract
but is not evaluated by the harness; no consumer may infer arbitrary code or
setup-program execution from its presence.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md)
- [`FLOWS.md`](FLOWS.md)
- [`../reference/configuration.md`](../reference/configuration.md)
