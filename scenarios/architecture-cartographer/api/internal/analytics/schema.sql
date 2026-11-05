-- Analytics tables — owned by internal/analytics/. Embedded by
-- schema.go and applied via database.EnsureSchemas at boot. Append-only:
-- no UPDATE / DELETE on the events table; corrections insert new rows
-- with corrects_event_id set.

CREATE TABLE IF NOT EXISTS analytics_events (
  id                  TEXT PRIMARY KEY,
  kind                TEXT NOT NULL,
  scenario            TEXT NOT NULL,
  domain              TEXT NOT NULL DEFAULT '',
  conflict_id         TEXT NOT NULL DEFAULT '',
  chunk_id            TEXT NOT NULL DEFAULT '',
  plan_id             TEXT NOT NULL DEFAULT '',
  run_id              TEXT NOT NULL DEFAULT '',
  corrects_event_id   TEXT NOT NULL DEFAULT '',
  payload             BLOB,
  actor               TEXT NOT NULL DEFAULT '',
  recorded_at         TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_analytics_events_scenario_time
  ON analytics_events(scenario, recorded_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_analytics_events_kind
  ON analytics_events(kind, recorded_at DESC);

CREATE TABLE IF NOT EXISTS analytics_placements (
  id           TEXT PRIMARY KEY,
  scenario     TEXT NOT NULL,
  chunk_id     TEXT NOT NULL,
  chunk_path   TEXT NOT NULL DEFAULT '',
  verdict_json BLOB,
  outcome      TEXT NOT NULL DEFAULT '',
  auto_acted   INTEGER NOT NULL DEFAULT 0,
  recorded_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_analytics_placements_scenario_time
  ON analytics_placements(scenario, recorded_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS analytics_overrides (
  id               TEXT PRIMARY KEY,
  scenario         TEXT NOT NULL,
  chunk_id         TEXT NOT NULL,
  verdict_domain   TEXT NOT NULL DEFAULT '',
  chosen_domain    TEXT NOT NULL DEFAULT '',
  note             TEXT NOT NULL DEFAULT '',
  verdict_event_id TEXT NOT NULL DEFAULT '',
  recorded_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_analytics_overrides_scenario_time
  ON analytics_overrides(scenario, recorded_at DESC, id DESC);
