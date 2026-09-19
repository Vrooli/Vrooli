# Prepared one-shot: retire legacy Qdrant collections (namespace adoption cleanup)

Status: **HISTORICAL — DO NOT APPLY.** This retired preparation note predates the
completed workflow cutover and is retained only as migration evidence behind a
`data-backup-manager` snapshot and a Qdrant collection snapshot. This note is the
plan + runbook only.

Owner surface: `scenarios/swarm-manager/api/internal/aisearch/` (vector index).
Related isolation work (Phase 1a): the hardcoded Qdrant collection namespace at
`internal/aisearch/vectorstore.go` was replaced with the variant-aware
`storage.Collection`/`defaultCollection()` helpers, clearing
`STORAGE_NAMESPACE_HARDCODED`. This note handles the *data* side that the code
change leaves behind.

## Why there is anything to migrate

`internal/aisearch/env.go` already composes collection names through the
variant-aware helper `storage.Collection(domain)` (root `swarm-manager` live,
`swarm-manager_<variant>` shadow, joined to the domain with `_`). So the **live
code already reads and writes the underscore collections**:
`swarm-manager_backlog`, `swarm-manager_initiatives`, `swarm-manager_records`.

Before that adoption the code used hyphen-joined names (`swarm-manager-backlog`,
`swarm-manager-initiatives`). Those pre-adoption collections are still resident in
Qdrant but are **no longer written to** — they are orphans. The convergent
reconciler (`aisearch.Reconciler`) already repopulated the underscore collections
from the stores of record, which is why they carry *more* points than the stale
hyphenated ones.

Qdrant here is a **derived index**, not a system of record. The sources of truth
are the on-disk backlog/initiative/record stores (filesystem + the SQLite event
log). Any collection can be dropped and rebuilt by the reconciler; no vector data
is authoritative.

## Inventory (captured 2026-07-14, live Qdrant `http://localhost:6333`)

| Collection | Points | Vector params | Status |
|---|---|---|---|
| `swarm-manager_backlog` | 600 | size 768, Cosine | **LIVE** (variant-aware, current) |
| `swarm-manager_initiatives` | 68 | size 768, Cosine | **LIVE** (variant-aware, current) |
| `swarm-manager_records` | 1458 | size 768, Cosine | **LIVE** (variant-aware, current) |
| `swarm-manager-backlog` | 483 | size 768, Cosine | LEGACY orphan (pre-adoption, stale) |
| `swarm-manager-initiatives` | 60 | size 768, Cosine | LEGACY orphan (pre-adoption, stale) |
| `swarm-manager` (bare slug) | — | — | ABSENT (old `DefaultCollectionName` fallback never materialized) |

Notes:
- There is **no** legacy `swarm-manager-records` collection: the records domain was
  added after helper adoption, so it only ever had the underscore name.
- The bare-slug `swarm-manager` collection does not exist. The removed
  `DefaultCollectionName = "swarm-manager"` constant was an empty-collection
  fallback that production never reached (every caller passes a resolved name), so
  the Phase 1a code change that replaced it with `defaultCollection()` has **zero**
  live-data impact and creates no `swarm-manager_default` collection.

## Prepared migration (apply in Phase 8, not now)

This is a **delete-orphans** cleanup, not a reindex — no data is copied because the
underscore collections are already current and the reconciler owns them.

Preconditions:
1. Confirm live code still targets the underscore names (grep `storage.Collection`
   adoption in `env.go`; unchanged since this note was written).
2. `data-backup-manager safety backup-now --scenario swarm-manager` — snapshot the
   filesystem + SQLite stores of record (the rebuild source) as the outer safety net.
3. Qdrant-level safety net for the collections being deleted (they are the only
   copies of their vectors, even if rebuildable): take a Qdrant snapshot of each
   before deletion.
   ```sh
   curl -s -X POST http://localhost:6333/collections/swarm-manager-backlog/snapshots
   curl -s -X POST http://localhost:6333/collections/swarm-manager-initiatives/snapshots
   ```

One-shot script (throwaway; lives in `/tmp/swarm-manager/`, never committed — per
storage-steer §5 greenfield one-shot pattern):

```sh
# /tmp/swarm-manager/migrate-drop-legacy-qdrant-collections.sh
set -euo pipefail
QDRANT="${QDRANT_URL:-http://localhost:6333}"

for c in swarm-manager-backlog swarm-manager-initiatives; do
  # Safety snapshot first (idempotent — a second snapshot is harmless).
  curl -sf -X POST "$QDRANT/collections/$c/snapshots" >/dev/null || true
  # Delete the orphaned legacy collection.
  curl -sf -X DELETE "$QDRANT/collections/$c" >/dev/null
  echo "dropped legacy collection: $c"
done
```

Post-conditions / verification:
- `GET /collections` no longer lists `swarm-manager-backlog` or
  `swarm-manager-initiatives`.
- The underscore collections are untouched and still current; if in any doubt,
  trigger a reconcile (`POST /api/v1/search/ai/reconcile`) and confirm the
  underscore collections converge against the on-disk stores.

Rollback:
- The vectors are rebuildable from the stores of record: recreate via the
  reconciler, or restore the pre-delete Qdrant snapshot with
  `PUT /collections/<name>/snapshots/recover`.

## Explicitly out of scope for this note

- No shadow/variant collection creation. Shadow collections
  (`swarm-manager_shadow_*`) are created on demand by a Baseline-Modes shadow
  instance through the same helper; nothing to pre-create.
- No SQLite/event-log migration. The routed-isolation work (Phase 1a) applies the
  event-log + evidence schema via `database.EnsureSchemas` at boot; no column-level
  migration is pending there.
