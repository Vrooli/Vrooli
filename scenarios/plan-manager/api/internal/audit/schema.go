package audit

const schemaSQL = `
CREATE TABLE IF NOT EXISTS plan_audit_facts (
  event_id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL,
  task_id TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  plan_id TEXT NOT NULL,
  content_digest TEXT NOT NULL DEFAULT '',
  occurred_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_plan_audit_facts_run ON plan_audit_facts(run_id, occurred_at);
`

func Schema() string { return schemaSQL }
