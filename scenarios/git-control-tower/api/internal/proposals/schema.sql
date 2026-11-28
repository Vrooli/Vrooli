CREATE TABLE IF NOT EXISTS gct_proposal_anchors (
			id TEXT PRIMARY KEY, repository_id TEXT NOT NULL, effort_ref TEXT NOT NULL DEFAULT '',
			epoch TEXT NOT NULL DEFAULT '', record_json TEXT NOT NULL, created_at TEXT NOT NULL);

CREATE INDEX IF NOT EXISTS idx_gct_proposal_anchors_work ON gct_proposal_anchors(repository_id, effort_ref, epoch, created_at);

CREATE TABLE IF NOT EXISTS gct_proposals (
			id TEXT PRIMARY KEY, repository_id TEXT NOT NULL, state TEXT NOT NULL, revision INTEGER NOT NULL,
			effort_ref TEXT NOT NULL DEFAULT '', epoch TEXT NOT NULL DEFAULT '', record_json TEXT NOT NULL,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL);

CREATE INDEX IF NOT EXISTS idx_gct_proposals_queue ON gct_proposals(repository_id, state, created_at);

CREATE INDEX IF NOT EXISTS idx_gct_proposals_work ON gct_proposals(repository_id, effort_ref, epoch, state);

CREATE TABLE IF NOT EXISTS gct_proposal_digests (
			day TEXT PRIMARY KEY, proposal_count INTEGER NOT NULL, sent_at TEXT NOT NULL);

CREATE TABLE IF NOT EXISTS gct_proposal_revisions (
			proposal_id TEXT NOT NULL, revision INTEGER NOT NULL, record_json TEXT NOT NULL,
			PRIMARY KEY (proposal_id, revision));

CREATE TABLE IF NOT EXISTS gct_proposal_blobs (blob_id TEXT PRIMARY KEY, content BLOB NOT NULL);
