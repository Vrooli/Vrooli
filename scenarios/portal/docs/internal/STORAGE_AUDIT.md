# Portal Storage Architecture Audit

## Last Updated

2026-09-07

## Current Pattern

- [x] Domain-owned schemas and repository interfaces.
- [x] Runtime filesystem paths use `api-core/storage`.
- [x] Retention state is declared under the state storage class.

## Migration Strategy

- Greenfield/dev-only assumption for this change; no production-user evidence
  was available in the execution context.
- Schema changes remain declarative. No scenario-local migration runner was added.

## Architecture Status

- [x] Context-capture schema is owned by `internal/contextcapture`.
- [x] Business logic uses repository interfaces.
- [x] Production context-capture requests use `*database.RoutedDB`.
- [x] Test leases use isolated pools without retaining a package-level `*sql.DB` handle.

## Engine and Filesystem Status

- SQLite remains behind the repository seam.
- Context blobs use the resolver-selected Portal data root.
- Retention enforcement receipts use `retention/enforcement-receipt.json` under
  the resolver-selected Portal state root.

## Evidence

`vrooli scenario test portal storage --wait --json` passed on run
`20260907-185947-6613476c` with storage maturity `L3` and no findings.

## Remaining Review

The five-platform native certification remains tracked by the active Portal
Everywhere plan. This audit does not claim live platform evidence.
