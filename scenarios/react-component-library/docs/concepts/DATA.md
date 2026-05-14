# Data — React Component Library

This document is the canonical storage map for the scenario.

## Storage Overview

React Component Library stores canonical component source in Git and
uses SQLite as an indexed registry and adoption ledger.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Component manifests | components | `library/components/<slug>/component.json` | Git-tracked file | Until component is removed | Manifest owns `libraryId`, display metadata, tags, latest, draft, and deprecated versions. |
| Component version source | components / versions | `library/components/<slug>/versions/<version>/*.tsx` plus indexed SQLite snapshot | Git-tracked file; SQLite stores content hash/content for stable reads | Released versions are immutable; drafts may change | Exactly one `.tsx` file per version folder in this phase. |
| Component registry | components | SQLite `components` | Manifest indexer | Rebuilt by `components index` | Soft-referenced by other domains through component id/library id. |
| Component versions | versions | SQLite `component_versions` | Manifest indexer | Rebuilt by `components index` | Stores status, source path, content, sha, and release timestamps. |
| Adoption records | adoptions | SQLite `adoption_records` | Apply/reapply service | Until deleted | Tracks scenario/path, adopted version, source sha, snapshot sha, and drift statuses. |
| Adopted component files | adoptions | Target scenario filesystem | Target scenario owns local edits after apply | Until scenario changes/deletes file | Files receive `@vrooliComponent*` provenance comments. |
| Dependency declarations | deps | SQLite | Latest indexed version header `@deps` | Rebuilt by index | Used to validate adoption against target scenario `package.json`. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `components` | components | `api/internal/components/schema.sql` | Registry list/get, apply lookup, preview |
| `component_versions` | versions/components | `api/internal/components/schema.sql`, mirrored in `api/internal/versions/schema.sql` | Version browsing, apply/reapply, diffs |
| `adoption_records` | adoptions | `api/internal/adoptions/schema.sql` | Apply/reapply/list/refresh |
| `dep_declarations` | deps | `api/internal/deps/schema.sql` | Dependency validation |

## Migration Policy

This scenario is still greenfield. Schema files are desired-state
bootstrap files using `CREATE TABLE IF NOT EXISTS`; existing local dev
SQLite data may be discarded while this model settles.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md)
- [`FLOWS.md`](FLOWS.md)
- [`../reference/configuration.md`](../reference/configuration.md)
