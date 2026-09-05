# Plan Manager Storage Architecture Audit

## Last Updated

2026-09-02

## Current Pattern

- Per-domain, embedded `internal/<domain>/schema.sql` providers registered by `internal/modules`.
- SQLite through `api-core/database` and the variant-aware `api-core/storage` resolver.
- Database path is resolved by the variant-aware `api-core/storage` resolver, so it differs by
  profile and the difference matters:
  - production / installed: `~/.vrooli/data/vrooli/plan-manager/plan-manager.db`
  - development from a source checkout: `scenarios/plan-manager/data/plan-manager.db`
    (git-ignored via `.gitignore` `data/`; 75 scenarios do the same)
  On a machine that has run both, the two diverge and **only the resolved one is live**. Read the
  path from the running process (`lsof`) rather than assuming, and note that a `retention` budget
  binds whichever path the resolver returns for the running profile — so on a dev host it does not
  bound the production copy, and vice versa.
- Startup runs a small, domain-owned migration before schema drift verification when a compatible additive SQLite change is needed. It is idempotent, runs before listeners open, and never rewrites evidence during reads.

## Architecture Status

- Each SQL table is owned by one domain; `internal/database/system.sql` remains empty for SQLite.
- Repositories are domain-local and handlers do not issue SQL directly. `storage-manager`'s three `DIRECT_SQL_IN_HANDLERS` findings are false positives on endpoint-descriptor strings, not executable SQL.
- SQLite is isolated by the scenario namespace and uses WAL, foreign keys, a busy timeout, and a single connection to avoid nested-query deadlocks.

## Retention

- `.vrooli/service.json` declares budgets for the three operational tables that grow without an
  upper bound: `log_entries` (365d / 256MiB), `validation_operations` (180d / 128MiB) and
  `candidate_revisions` (30d / 64MiB, pruned on its own `expires_at`). They are enforced by the
  shared `api-core/retention` engine, wired in `api/main.go`, all using the builtin pruner.
- Deliberately unbounded, each for a different reason:
  - `plans` — a plan is the durable product of this scenario, not regenerable state.
  - `authoring_sessions` — the builtin pruner has no row predicate, so an age rule alone cannot
    distinguish a finished session from an open one and would delete in-progress authoring.
    Bounding it needs a custom pruner keyed on `finalized`.
  - the rendered mirror tree at `~/.vrooli/plans` — the repo contract declares that runtime-home
    entry `protected: true` with `cleanup: "never"`. Adding retention there would violate it.
- The mirror index (`~/.vrooli/plans/_index.json`) is swept by `OSMirrorStore.PruneIndex`, which a
  non-dry-run `plans reconcile --repair-mirrors` invokes. It drops records whose rendered file is
  gone and collapses duplicate ids; records for orphaned mirrors whose file still exists are kept,
  because that is the state an operator may want to recover from.

## Migration Hygiene

- `EnsureSchemas` applies only idempotent desired-state schemas and detects SQLite column drift at boot.
- `validation.EnsureMigrations` adds missing terminal-result receipt columns before `EnsureSchemas` performs its SQLite drift check. Old result rows receive safe zero values and cannot satisfy an execution gate; they require fresh validation.
- Validation-operation payloads are forward-only schema V2. Older command-only payloads are rejected with an actionable migration error rather than silently rewritten on read.

## Cross-References

- `storage-manager validate scenario plan-manager`
- `packages/api-core/database/schemas.go`
- `docs/concepts/DATA.md`
