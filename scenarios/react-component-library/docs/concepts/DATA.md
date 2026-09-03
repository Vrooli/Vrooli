# Data — React Component Library

This document is the canonical storage map for the scenario.

## Storage Overview

React Component Library stores canonical component source in Git and
uses SQLite as an indexed registry and adoption ledger.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Asset manifests | components | `library/components/<slug>/component.json` or `library/hooks/<slug>/component.json` | Git-tracked file | Until asset is removed | Manifest owns `libraryId`, asset kind, display metadata, tags, version pointers, and pinned library-asset dependencies. |
| Component version source | components / versions | Materialized versions in `library/components/<slug>/versions/<version>/`; every version's identity and file mirror in SQLite | Git-tracked files are the warm projection; `component_versions` and `component_version_files` retain the complete cold bytes and hashes | Released versions are immutable; drafts and transient materializations may change placement, never released bytes | `presence=materialized` means files are in the working tree; `presence=evicted` means the manifest retains the version identity and the SQLite file mirror is the readable source. A version is one entry `.tsx`/`.ts` plus optional same-folder companions. |
| Component registry | components | SQLite `components` | Manifest indexer | Rebuilt by `components index` | Soft-referenced by other domains through component id/library id. Includes the indexed adoption `slot` and latest-version `category` facet. |
| Component versions | components (indexed source); versions (content-save recorder) | SQLite `component_versions`, `component_version_files` | Manifest indexer for identity and materialized rows; the file mirror is retained for evicted rows | Version rows and file mirrors remain durable across reindex; materialized rows are refreshed, evicted rows are preserved | `component_versions` mirrors the entry for legacy reads; `component_version_files` is the authoritative per-file path/content/hash set and carries a per-file `slot` so multi-file versions place each file independently. `presence` is reconciled from the reachability graph. `created_at` is first-git-seen time and `released_at` is stamped when the manifest latest pointer resolves to the version. |
| Reviewed template divergences | adoptions | `api/internal/adoptions/testdata/reviewed-template-divergences.json` | Human review (Git-tracked) | Until the divergence is resolved | Dated allowlist of explicitly-accepted mismatches in template-owned or ejected source (path, library id, pinned version, catalog version, status, reason, report pointer). The parity test fails for any un-listed behind/deprecated source or stale entry, so the allowlist is an audit trail, not a suppression. |
| Version story contracts | components | `library/<kind>/<slug>/versions/<version>/story.json` plus typed SQLite story projections | Git-tracked story file | Rebuilt by `components index` | One schemaVersioned contract owns public args/input validation, named baselines, explicit environment fixtures, interactions, and expectations. It is declarative data only; the preview harness resolves only the documented allowlisted `$` vocabulary. |
| Preview-session story edits | preview host / harness | Component-editor React state and one registered iframe's memory | Validated public-path edits over the selected story | Cleared by Reset, reload, navigation, or unmount | Generated Args controls update only declared paths after complete effective-args validation. Environment controls select declared fixtures. Neither is written to story.json, SQLite, localStorage, source, or an RPC. |
| Component headers | components | SQLite `component_headers` | Latest version source header | Rebuilt by `components index` | Stores non-structural header metadata. Identity, version, category, and deps are stored in typed columns/tables instead; conflicting structural header hints emit `header_disagreement` findings instead of owning projection facts. |
| Component design affinities | components | SQLite `component_design_affinities` | `component.json` `designStyles[]` | Rebuilt by `components index` | Component-scoped, reconciled against `templates/design/*/metadata.json`, and carries optional rationale text for search, adoption workflows, and UI. Unknown style IDs emit non-fatal staleness findings so catalog drift is visible without dropping the component. |
| Adoption records | adoptions | SQLite `adoption_records`, `adoption_files` | Link/eject service | Until deleted | Each direct parent records `mode=linked` or `mode=ejected`, the pinned asset/version, and obligation state. Linked rows point at the governed package; only reasoned ejections own materialized source files. Each `adoption_files` row records authoritative origin, so effective usage does not depend on filename inference. |
| Assisted workflows | workflows | SQLite `assisted_workflows` | Workflows service | Until deleted by retention policy | RCL-owned linkage from asset/source/target/idempotency key to Agent Manager task/run, status, event sequence, summary/error, and timestamps. |
| Adopted component files | adoptions | Target scenario filesystem | Target scenario owns local edits after eject | Until scenario changes/deletes file | Only ejected files receive `@vrooliComponent*` provenance comments; linked source remains in the governed package. |
| Dependency declarations | deps | SQLite `component_dep_declarations` | Version source header `@deps` | Rebuilt by index | Stores one row per component version, dependency name, range, and kind (`runtime`, `peer`, `dev`) for adoption validation. |
| Adoption suggestions | adoptions | Derived at request time | Inventory surfaces + component registry + validation verdicts + adoption ledger | Not persisted | Suggestions carry their inventory path and human-readable reason strings; existing adoption pairs are excluded. |
| Catalog gate evidence | catalogcoverage | SQLite `catalog_gate_evidence` | Gate runner result for a resolved asset version | Append-only, newest three repeats retained per asset/target/gate/version | Version-scoped verdict history; the current coverage projection selects the latest version. |
| Component test reports | componenttests | SQLite `component_test_reports` | Component test runner | Five newest payloads per component/version plus first pass and first failure, bounded by a 256 MiB total payload ceiling | Evicted payloads remain represented by the version rollup counters; the byte ceiling is enforced after every write and at startup. |
| Component version test rollups | componenttests | SQLite `component_version_test_rollup` | Component test report writer | Until the version is retired | Lossless pass/fail/blocked totals keyed by stable library id and version. |
| Version ledger | versionledger | SQLite `version_ledger` | Replayable projection of indexed versions, gate evidence, test rollups, and adoption facts | Durable, including evicted and retired versions | Stable `(library_id, version)` analytical row; source content is never copied into this projection. `versions export-archive` is the portable backup of the identity and file mirror tables. |

Released authored source (`*.ts`, `*.tsx`, stories, and experience contracts)
is immutable and hash-protected. `dependencies.json` is a derived lock: it
stores exact resolutions for major-line imports and is regenerated when the
active dependency changes, so it is intentionally excluded from the release
hash ledger.

## Cold version tier

Version identity and version bytes have separate placement. The reachability
predicate used by `versions reap` is the authority: latest and
draft versions, adopted or dependency-pinned versions, and versions named by
adoption files or source imports stay materialized. Other non-retired versions
may be marked `evicted`; their manifest identity, SQLite row, complete file
mirror, and per-file SHA-256 hashes remain available to `versions show`, diff,
preview, and package materialization.

`versions reconcile-presence` is idempotent and is also invoked after catalog
indexing and adoption lifecycle changes. Materialization verifies the complete
file set before an atomic directory rename. A missing or mismatched mirror is
an error, never an invitation to treat the version as empty. `versions doctor`
checks these invariants, and `versions export-archive`/`import-archive` provide
the portable recovery boundary for the host-local SQLite store.

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `components` | components | `api/internal/components/schema.sql` | Registry list/get, apply lookup, preview |
| `component_versions` | components | `api/internal/components/schema.sql` (single-owner; the mirrored `api/internal/versions` schema was removed) | Version browsing, apply/reapply, diffs; `created_at` survives idempotent reindex upserts and `released_at` records the latest transition |
| `component_version_files` | components | `api/internal/components/schema.sql` | File tabs, file-scoped content RPCs, per-file slot placement, preview/adoption units |
| `reviewed-template-divergences.json` | adoptions | `api/internal/adoptions/testdata/` | Template origin-parity test (allowlisted divergences + calibration) |
| `component_stories` and typed child projections | components | `api/internal/components/schema.sql` | Story browsing, generated controls, preview harness, runner, and component spec reconciliation |
| `component_headers` | components | `api/internal/components/schema.sql` | Component get for non-structural latest-version metadata |
| `component_design_affinities` | components | `api/internal/components/schema.sql` | Component get/list, style/affinity search, UI badges |
| `component_asset_dependencies` | components | `api/internal/components/schema.sql` | Deterministic preview/adoption closure resolution |
| `adoption_records` | adoptions | `api/internal/adoptions/schema.sql` | Apply/reapply/list/refresh |
| `adoption_files` | adoptions | `api/internal/adoptions/schema.sql` | Per-file adoption paths, snapshots, and drift detail |
| `assisted_workflows` | workflows | `api/internal/workflows/schema.sql` | Assisted-work header, CLI, and Agent Manager status reconciliation |
| `component_dep_declarations` | deps | `api/internal/deps/schema.sql` | Dependency validation and dep-aware preview import maps |
| `catalog_gate_evidence` | catalogcoverage | `api/internal/catalogcoverage/evidence.go` plus one-shot migration | Versioned gate history and catalog coverage projection |
| `component_test_reports` | componenttests | `api/internal/components/schema.sql` | Tests tab and report detail; bounded payload retention |
| `component_version_test_rollup` | componenttests | `api/internal/components/schema.sql` | Version ledger test totals and pass-rate measures |
| `version_ledger` | versionledger | `api/internal/versionledger/schema.go` | Progression data, retirement candidates, and measures |

## Migration Policy

This scenario is greenfield, but local development data is still
preserved during normal hardening. Schema files are desired-state
bootstrap files using `CREATE TABLE IF NOT EXISTS`; when a pre-existing
SQLite table needs a new column, index, or primary-key shape, apply a
one-shot `/tmp/react-component-library/migrate-*.sql` migration with the
scenario stopped. Do not recreate the database just to satisfy schema
drift.

## Preview-session boundary

The editor treats a selected named story as its immutable baseline. Generated
Args controls edit only declared public paths, and every partial edit is
validated before the matching iframe receives the complete effective args.
Reset restores the selected story; no source file, SQLite row, localStorage
entry, or write RPC is changed. Environment controls select only declared
server-owned fixtures. Internal state is reached through real interactions or
the component's normal controlled/default API; arbitrary setup, imports, and
hook mutation have no runtime model. See [STORY-CONTRACT.md](STORY-CONTRACT.md).

## Cross-References

- [`DOMAINS.md`](DOMAINS.md)
- [`FLOWS.md`](FLOWS.md)
- [`../reference/configuration.md`](../reference/configuration.md)
