CREATE TABLE IF NOT EXISTS persona_document_bindings (
  id TEXT PRIMARY KEY NOT NULL,
  persona_id TEXT NOT NULL REFERENCES personas(id),
  document_id TEXT NOT NULL,
  document_kind TEXT NOT NULL,
  valid_until TEXT,
  created_at TEXT NOT NULL,
  UNIQUE (persona_id, document_id)
);

CREATE INDEX IF NOT EXISTS persona_document_bindings_persona_idx ON persona_document_bindings(persona_id, created_at DESC);
