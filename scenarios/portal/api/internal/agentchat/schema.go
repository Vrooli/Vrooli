package agentchat

import "context"

// Schema owns durable agent admission and exact remote-run associations.
func Schema() string {
	return `
CREATE TABLE IF NOT EXISTS agent_chat_runs (
 id TEXT PRIMARY KEY,
 chat_id TEXT NOT NULL,
 message_id TEXT NOT NULL,
 task_id TEXT NOT NULL DEFAULT '',
 run_id TEXT UNIQUE,
 brief_id TEXT NOT NULL DEFAULT '',
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

// EnsureBriefIDColumn upgrades databases created before brief attribution was
// added. The declarative schema remains the greenfield source of truth; this
// narrow additive check is the compatibility path for an existing Portal DB.
func EnsureBriefIDColumn(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(agent_chat_runs)")
	if err != nil {
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == "brief_id" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if found {
		return nil
	}
	_, err = db.ExecContext(ctx, "ALTER TABLE agent_chat_runs ADD COLUMN brief_id TEXT NOT NULL DEFAULT ''")
	return err
}
