CREATE TABLE IF NOT EXISTS persona_grants (
  id TEXT PRIMARY KEY NOT NULL,
  persona_id TEXT NOT NULL REFERENCES personas(id),
  human_subject TEXT NOT NULL,
  level TEXT NOT NULL CHECK (level IN ('act', 'propose')),
  source TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (persona_id, human_subject)
);

CREATE INDEX IF NOT EXISTS persona_grants_persona_idx ON persona_grants(persona_id, human_subject);
