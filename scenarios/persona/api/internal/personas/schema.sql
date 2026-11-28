CREATE TABLE IF NOT EXISTS personas (
  id TEXT PRIMARY KEY NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('personal', 'business')),
  legal_subject_id TEXT NOT NULL,
  legal_subject_name TEXT NOT NULL,
  legal_basis_type TEXT NOT NULL,
  display_name TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('active', 'archived')),
  created_at TEXT NOT NULL,
  archived_at TEXT
);

CREATE TABLE IF NOT EXISTS persona_identifiers (
  persona_id TEXT NOT NULL REFERENCES personas(id),
  identifier_type TEXT NOT NULL,
  identifier_value TEXT NOT NULL,
  PRIMARY KEY (persona_id, identifier_type),
  UNIQUE (persona_id, identifier_type, identifier_value)
);

CREATE INDEX IF NOT EXISTS personas_status_created_idx ON personas(status, created_at DESC);
