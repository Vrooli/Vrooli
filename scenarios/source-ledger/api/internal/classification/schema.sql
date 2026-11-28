CREATE TABLE IF NOT EXISTS classification_progress (
  category_id TEXT NOT NULL,
  definition_version TEXT NOT NULL,
  definition_spec TEXT NOT NULL,
  history_position INTEGER NOT NULL DEFAULT 0,
  history_through INTEGER NOT NULL,
  live_position INTEGER NOT NULL,
  PRIMARY KEY (category_id, definition_version)
) WITHOUT ROWID;

CREATE TABLE IF NOT EXISTS classification_retries (
  category_id TEXT NOT NULL,
  definition_version TEXT NOT NULL,
  position INTEGER NOT NULL,
  attempts INTEGER NOT NULL,
  next_attempt INTEGER NOT NULL,
  reason TEXT NOT NULL,
  PRIMARY KEY (category_id, definition_version, position)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS idx_classification_retries_due
ON classification_retries(category_id, definition_version, next_attempt, position);

CREATE TABLE IF NOT EXISTS classification_matches (
  sequence INTEGER PRIMARY KEY,
  category_id TEXT NOT NULL,
  definition_version TEXT NOT NULL,
  scope TEXT NOT NULL,
  entry_id TEXT NOT NULL,
  decision_method TEXT NOT NULL,
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  UNIQUE (category_id, definition_version, scope, entry_id)
);
CREATE INDEX IF NOT EXISTS idx_classification_matches_notices
ON classification_matches(scope, category_id, definition_version, sequence);
