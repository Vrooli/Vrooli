CREATE TABLE IF NOT EXISTS persona_handoffs (
  id TEXT PRIMARY KEY NOT NULL,
  persona_id TEXT NOT NULL REFERENCES personas(id),
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  human_action TEXT NOT NULL,
  checkpoint_json TEXT NOT NULL,
  state TEXT NOT NULL CHECK (state IN ('open', 'delivered', 'awaiting_human', 'completed', 'expired', 'cancelled', 'resumed')),
  opened_by_run_id TEXT NOT NULL,
  authorising_human TEXT NOT NULL,
  deadline TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  relay_state TEXT NOT NULL DEFAULT 'queued'
);

CREATE INDEX IF NOT EXISTS persona_handoffs_persona_state_idx ON persona_handoffs(persona_id, state, updated_at DESC);
