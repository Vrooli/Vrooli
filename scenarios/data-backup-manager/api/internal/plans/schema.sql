-- Plans tables — owned by internal/plans/. Embedded by schema.go and applied
-- via database.EnsureSchemas at boot through the modules.AllSchemas registry.
-- Times are RFC3339Nano strings matching the wire format. CREATE ... IF NOT
-- EXISTS so re-runs are no-ops.

CREATE TABLE IF NOT EXISTS plans (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  schedule    TEXT NOT NULL DEFAULT '',
  keep_latest INTEGER NOT NULL DEFAULT 0,
  enabled     INTEGER NOT NULL DEFAULT 1,
  protection_tier TEXT NOT NULL DEFAULT 'full_primary',
  recovery_drill_schedule TEXT NOT NULL DEFAULT '',
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plans_name ON plans(name);

-- plan_targets is the many-to-many membership table linking plans to targets.
-- A target may appear in multiple plans.
CREATE TABLE IF NOT EXISTS plan_targets (
  plan_id   TEXT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
  target_id TEXT NOT NULL,
  PRIMARY KEY (plan_id, target_id)
);

CREATE INDEX IF NOT EXISTS idx_plan_targets_plan ON plan_targets(plan_id);

-- plan_destinations links plans to destination ids.
CREATE TABLE IF NOT EXISTS plan_destinations (
  plan_id        TEXT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
  destination_id TEXT NOT NULL,
  PRIMARY KEY (plan_id, destination_id)
);

CREATE INDEX IF NOT EXISTS idx_plan_destinations_plan ON plan_destinations(plan_id);
