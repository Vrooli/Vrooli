# Data — React Component Library

This document is the canonical storage map for the scenario.

## Storage Overview

React Component Library stores canonical component source in Git and
uses SQLite as an indexed registry and adoption ledger.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Component manifests | components | `library/components/<slug>/component.json` | Git-tracked file | Until component is removed | Manifest owns `libraryId`, display metadata, `slot`, tags, design-style affinities, affinity reasons, latest, draft, and deprecated versions. |
| Component version source | components / versions | `library/components/<slug>/versions/<version>/*.tsx` plus indexed SQLite snapshot | Git-tracked file; SQLite stores content hash/content for stable reads | Released versions are immutable; drafts may change | Exactly one `.tsx` file per version folder in this phase. |
| Component registry | components | SQLite `components` | Manifest indexer | Rebuilt by `components index` | Soft-referenced by other domains through component id/library id. Includes the indexed adoption `slot` and latest-version `category` facet. |
| Component versions | versions | SQLite `component_versions` | Manifest indexer | Rebuilt by `components index` | Stores status, source path, content, sha, and release timestamps. |
| Component headers | components | SQLite `component_headers` | Latest version source header | Rebuilt by `components index` | Stores non-structural header metadata. Identity, version, category, and deps are stored in typed columns/tables instead; conflicting structural header hints emit `header_disagreement` findings instead of owning projection facts. |
| Component design affinities | components | SQLite `component_design_affinities` | `component.json` `designStyles[]` | Rebuilt by `components index` | Component-scoped, reconciled against `templates/design/*/metadata.json`, and carries optional rationale text for search, adoption workflows, and UI. Unknown style IDs emit non-fatal staleness findings so catalog drift is visible without dropping the component. |
| Adoption records | adoptions | SQLite `adoption_records` | Apply/reapply service | Until deleted | Tracks scenario/path, adopted version, source sha, snapshot sha, and drift statuses. |
| Adopted component files | adoptions | Target scenario filesystem | Target scenario owns local edits after apply | Until scenario changes/deletes file | Files receive `@vrooliComponent*` provenance comments. |
| Dependency declarations | deps | SQLite `component_dep_declarations` | Version source header `@deps` | Rebuilt by index | Stores one row per component version, dependency name, range, and kind (`runtime`, `peer`, `dev`) for adoption validation. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `components` | components | `api/internal/components/schema.sql` | Registry list/get, apply lookup, preview |
| `component_versions` | versions/components | `api/internal/components/schema.sql`, mirrored in `api/internal/versions/schema.sql` | Version browsing, apply/reapply, diffs |
| `component_headers` | components | `api/internal/components/schema.sql` | Component get for non-structural latest-version metadata |
| `component_design_affinities` | components | `api/internal/components/schema.sql` | Component get/list, style/affinity search, UI badges |
| `adoption_records` | adoptions | `api/internal/adoptions/schema.sql` | Apply/reapply/list/refresh |
| `component_dep_declarations` | deps | `api/internal/deps/schema.sql` | Dependency validation and dep-aware preview import maps |

## Migration Policy

This scenario is greenfield, but local development data is still
preserved during normal hardening. Schema files are desired-state
bootstrap files using `CREATE TABLE IF NOT EXISTS`; when a pre-existing
SQLite table needs a new column, index, or primary-key shape, apply a
one-shot `/tmp/react-component-library/migrate-*.sql` migration with the
scenario stopped. Do not recreate the database just to satisfy schema
drift.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md)
- [`FLOWS.md`](FLOWS.md)
- [`../reference/configuration.md`](../reference/configuration.md)
