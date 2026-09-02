CREATE TABLE IF NOT EXISTS library_programs (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  version INTEGER NOT NULL,
  source TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  origin TEXT NOT NULL,
  created_at TEXT NOT NULL,
  source_program_id TEXT NOT NULL DEFAULT '',
  promoted_by TEXT NOT NULL DEFAULT '',
  promotion_reason TEXT NOT NULL DEFAULT '',
  called_binding_ids TEXT NOT NULL DEFAULT '[]',
  tier TEXT NOT NULL DEFAULT 'promoted',
  declared_inputs TEXT NOT NULL DEFAULT '[]',
  declared_outputs TEXT NOT NULL DEFAULT '[]',
  coverage TEXT NOT NULL DEFAULT '',
  validated_at TEXT NOT NULL DEFAULT '',
  UNIQUE(name, version)
);

CREATE TABLE IF NOT EXISTS library_current (
  name TEXT PRIMARY KEY,
  version INTEGER NOT NULL,
  FOREIGN KEY(name, version) REFERENCES library_programs(name, version)
);

CREATE INDEX IF NOT EXISTS idx_library_programs_name ON library_programs(name, version);
