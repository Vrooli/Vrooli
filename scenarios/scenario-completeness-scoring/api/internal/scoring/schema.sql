CREATE TABLE IF NOT EXISTS score_snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scenario TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT 'utility',
  digest TEXT NOT NULL,
  composite INTEGER NOT NULL,
  classification TEXT NOT NULL,
  working_rung TEXT NOT NULL DEFAULT '',
  breakdown_json TEXT NOT NULL DEFAULT '{}',
  importance REAL,
  source TEXT NOT NULL DEFAULT 'sweeper',
  created_at TEXT NOT NULL,
  UNIQUE (scenario, digest)
);

CREATE INDEX IF NOT EXISTS idx_score_snapshots_scenario_created_at
  ON score_snapshots (scenario, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_score_snapshots_scenario_digest
  ON score_snapshots (scenario, digest);

CREATE INDEX IF NOT EXISTS idx_score_snapshots_list_composite
  ON score_snapshots (composite DESC, scenario ASC);

CREATE INDEX IF NOT EXISTS idx_score_snapshots_list_created_at
  ON score_snapshots (created_at DESC, scenario ASC);
