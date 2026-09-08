package agentchat

// Schema owns durable agent admission and exact remote-run associations.
func Schema() string {
	return `
CREATE TABLE IF NOT EXISTS agent_chat_runs (
 id TEXT PRIMARY KEY,
 chat_id TEXT NOT NULL,
 message_id TEXT NOT NULL,
 task_id TEXT NOT NULL DEFAULT '',
 run_id TEXT UNIQUE,
 UNIQUE(chat_id, message_id)
);
-- Admission ownership survives deletion of the source conversation. A missing
-- row identifies a legacy admission, never an account inferred from chat state.
CREATE TABLE IF NOT EXISTS agent_chat_run_owners (
 admission_id TEXT PRIMARY KEY REFERENCES agent_chat_runs(id) ON DELETE CASCADE,
 owner TEXT NOT NULL CHECK(length(owner)>0 AND length(owner)<=256)
);
CREATE INDEX IF NOT EXISTS idx_agent_chat_run_owners ON agent_chat_run_owners(owner,admission_id);
`
}
