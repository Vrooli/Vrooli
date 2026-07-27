CREATE TABLE IF NOT EXISTS category (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  reserved INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  retired_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS category_active_idx ON category(retired_at, name);

CREATE TABLE IF NOT EXISTS taxonomy (
  id TEXT PRIMARY KEY,
  category_id TEXT NOT NULL REFERENCES category(id),
  label TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  UNIQUE(category_id, label)
);

CREATE TABLE IF NOT EXISTS classification (
  id TEXT PRIMARY KEY,
  signal_id TEXT NOT NULL REFERENCES signal(id),
  proposed_category_id TEXT NOT NULL REFERENCES category(id),
  proposed_confidence REAL NOT NULL,
  model TEXT NOT NULL DEFAULT '',
  confirmed_category_id TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  CHECK (proposed_confidence >= 0 AND proposed_confidence <= 1)
);
CREATE INDEX IF NOT EXISTS classification_signal_created_idx ON classification(signal_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS classification_queue (
  signal_id TEXT PRIMARY KEY REFERENCES signal(id),
  reason TEXT NOT NULL,
  enqueued_at TEXT NOT NULL
);
