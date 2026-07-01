-- plans + plan_edges — owned by internal/plans/. The structured-plan SSOT.
--
-- A plan's first-class queryable columns (id/slug/title/status/content_hash/
-- timestamps) sit alongside a `document` JSON column that carries the rest of
-- the structured record — purpose/scope/constraints/non_goals/definition_of_done
-- plus the ordered phases[], references[] and the regression anchor. Phases and
-- references are always loaded with their plan and are never queried across
-- plans, so they persist within the plan document rather than as separate
-- tables (the ownership contract in docs/concepts/DATA.md holds either way; this
-- shape avoids the SQLite pool=1 nested-query deadlock and keeps round-trips
-- deterministic). The supersession/dependency graph DOES query across plans, so
-- plan_edges is a first-class table.
--
-- Embedded by schema.go and applied idempotently via database.EnsureSchemas at
-- boot through the modules.AllSchemas registry. Timestamps are RFC3339Nano. Use
-- CREATE ... IF NOT EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS plans (
  id           TEXT PRIMARY KEY,
  slug         TEXT NOT NULL,
  title        TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT 'draft',
  content_hash TEXT NOT NULL DEFAULT '',
  document     TEXT NOT NULL DEFAULT '{}',
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plans_slug_lookup ON plans(slug);
CREATE INDEX IF NOT EXISTS idx_plans_status ON plans(status);

-- plan_edges — the supersession/dependency graph. A row (from, to, kind) means
-- `from` supersedes/depends-on `to`. Queried across plans by GetGraph.
CREATE TABLE IF NOT EXISTS plan_edges (
  from_plan_id TEXT NOT NULL,
  to_plan_id   TEXT NOT NULL,
  kind         TEXT NOT NULL DEFAULT 'supersedes',
  PRIMARY KEY (from_plan_id, to_plan_id, kind)
);

CREATE INDEX IF NOT EXISTS idx_plan_edges_from ON plan_edges(from_plan_id);
CREATE INDEX IF NOT EXISTS idx_plan_edges_to ON plan_edges(to_plan_id);
