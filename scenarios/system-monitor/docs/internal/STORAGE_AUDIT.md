# System Monitor Storage Architecture Audit

## Last Updated

2026-08-21

## Current Pattern

- SQLite is the durable runtime store.
- Metric observations are owned by the metrics repository and grouped by an
  explicit cycle ID.
- The schema is declarative and idempotent. This scenario is greenfield; it
  does not ship a migration runner or a permanent compatibility layer.
- In-memory storage is an explicit non-production test mode only.

## Architecture Status

- Repository interfaces hide the storage engine from services.
- SQLite and memory repositories implement the same cycle contract.
- Runtime paths resolve through `api-core/storage`.
- SQLite is routed through `database.RoutedDB` for test isolation.

## Issues Found

1. The existing personal development database predates the cycle schema. The
   greenfield contract intentionally does not preserve it through runtime
   migrations. If that local history is needed, use a one-shot operator-owned
   export/rebuild action while the scenario is stopped; do not add that action
   to the shipped API.
2. The schema remains centralized in the SQLite repository rather than a
   per-domain embedded `schema.sql` package. A future storage refactor can move
   the metrics tables next to their repository without changing services.

## Engine Status

- SQLite via `modernc.org/sqlite` (CGO-free).
- No Redis or Qdrant state.
- Filesystem placement uses `api-core/storage`.

## Cross-References

- `storage-manager validate scenario system-monitor`
- `packages/api-core/database/schemas.go`
- `scenarios/system-monitor/api/internal/repository/interfaces.go`
