-- This table is append-only by contract. Corrections are new rows.
CREATE TABLE IF NOT EXISTS persona_journal (
  id TEXT PRIMARY KEY NOT NULL,
  persona_id TEXT NOT NULL,
  actor TEXT NOT NULL,
  verb TEXT NOT NULL,
  run_id TEXT NOT NULL,
  authorising_human TEXT NOT NULL,
  at TEXT NOT NULL,
  outcome TEXT NOT NULL,
  constraint_name TEXT NOT NULL,
  details_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS persona_journal_persona_at_idx ON persona_journal(persona_id, at DESC, id DESC);
