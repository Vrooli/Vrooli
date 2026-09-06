# Sketch storage implementation evidence

Measured 2026-09-05 while executing the visual design studio successor. This record covers the sketch domain; other scenario storage domains were not reassessed.

## Ownership and persistence

`api/internal/sketch/Repository` owns authored page reads, expected-hash writes, revision history, and recovery. `Store` implements it with scenario-owned `experience/pages/*.json` as the mutable authority. Immutable original-byte snapshots live under `experience/designs/<page>/revisions/`. Publication receipts distinguish applied content revisions from prepared snapshots; they are not an operation-attempt ledger.

Native file locks serialize writers across processes. A recoverable apply manifest precedes page replacement. Recovery either completes that recorded replacement or reports an intervening-edit conflict. The shared `api-core/storage.WriteFileAtomicInRoot` helper anchors IO to an open experience directory. Individual files are replaced atomically; the multi-file operation uses recovery rather than claiming cross-file atomicity.

`handlers/sketch` resolves repositories per request. Live authored pages remain in the workspace. Test-mode requests use the active file lease's `config/workspace` fixture tree. Missing or expired leases are refused. Inventory scanners use the same selected workspace. No database migration or secondary mutable page store was introduced.

## Evidence

- `api/internal/sketch/store_test.go`: source preservation, nested extensions, exact conflict hashes, cross-process writers, source identity precision, ambiguous JSON rejection, immutable history, and interruption recovery.
- `api/handlers/sketch/connect_handler_test.go`: typed Connect conflicts, restoration metadata, history/recovery, and lease isolation including expiration.
- `packages/api-core/storage/fs_test.go`: rooted atomic replacement and escape rejection.
- Targeted domain and handler tests pass with the race detector. API composition-root compilation and CLI domain registration tests pass.
- Lifecycle operation `startop-8c29029b02a10e4fd3cf8060de63f2c6` completed healthy. Live history/get commands returned matching hashes for Switchboard Conversations.

## Outstanding work

The full storage maturity gate has not been run. Accepted-design references, canonical Experience Manager schema validation, candidate composition revisions, and scenario-wide multi-page application remain part of the unfinished successor plan. Current snapshots provide safe editing history; they do not certify design acceptance or UI behavior.
