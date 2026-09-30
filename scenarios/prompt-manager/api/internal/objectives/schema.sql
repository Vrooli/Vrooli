CREATE TABLE IF NOT EXISTS objectives (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    class TEXT NOT NULL,
    evidence_source TEXT NOT NULL DEFAULT '',
    has_evidence INTEGER NOT NULL DEFAULT 0,
    gap_marker TEXT NOT NULL DEFAULT '',
    global_order INTEGER NOT NULL,
    meaning_revision TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_objectives_global_order ON objectives(global_order);

CREATE TABLE IF NOT EXISTS objective_attachments (
    objective_id TEXT NOT NULL,
    team_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT '',
    coverage TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    priority INTEGER NOT NULL,
    acknowledged_revision TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (objective_id, team_id)
);

CREATE INDEX IF NOT EXISTS idx_objective_attachments_team ON objective_attachments(team_id, priority);

CREATE TABLE IF NOT EXISTS team_attachment_revisions (
    team_id TEXT PRIMARY KEY,
    revision TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS objective_relations (
    from_objective_id TEXT NOT NULL,
    to_objective_id TEXT NOT NULL,
    PRIMARY KEY (from_objective_id, to_objective_id)
);

-- Migration receipts and recoverable source snapshots. A migration is keyed by
-- the digest of its operator source files, so replaying the same import is a
-- no-op and a partial (interrupted) import converges on retry. The snapshot
-- keeps the pre-import declaration bytes in the routed store so recovery never
-- depends on the working tree.
CREATE TABLE IF NOT EXISTS objective_import_snapshots (
    source_digest TEXT PRIMARY KEY,
    payload TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS objective_import_receipts (
    source_digest TEXT PRIMARY KEY,
    payload TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
