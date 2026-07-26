-- Immutable workflow catalog. Definitions are scenario-owned desired state;
-- Agent Manager owns only revisions and the active pointer.
CREATE TABLE IF NOT EXISTS workflow_revisions (
    id TEXT PRIMARY KEY,
    owner TEXT NOT NULL,
    workflow_key TEXT NOT NULL,
    semantic_version TEXT NOT NULL,
    digest TEXT NOT NULL UNIQUE,
    definition_json TEXT NOT NULL,
    source_path TEXT NOT NULL,
    source_hash TEXT NOT NULL,
    source_updated_at TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_revisions_owner_key
    ON workflow_revisions(owner, workflow_key, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_workflow_revisions_active
    ON workflow_revisions(owner, workflow_key) WHERE active = 1;

CREATE TABLE IF NOT EXISTS workflow_executions (
    id TEXT PRIMARY KEY,
    owner TEXT NOT NULL,
    workflow_key TEXT NOT NULL,
    definition_digest TEXT NOT NULL REFERENCES workflow_revisions(digest),
    status TEXT NOT NULL,
    current_node_id TEXT NOT NULL,
    input_json TEXT NOT NULL,
    output_json TEXT,
    terminal_reason_json TEXT,
    budget_usage_json TEXT NOT NULL,
    edge_traversals_json TEXT NOT NULL,
    version INTEGER NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    parent_execution_id TEXT,
    parent_attempt_id TEXT,
    depth INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    ended_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_recoverable ON workflow_executions(status, updated_at);

CREATE TABLE IF NOT EXISTS workflow_node_attempts (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    ordinal INTEGER NOT NULL,
    strategy TEXT NOT NULL,
    status TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    input_snapshot_json TEXT NOT NULL,
    prompt_snapshot TEXT NOT NULL,
    experiment_id TEXT,
    variant_id TEXT,
    prompt_hash TEXT,
    run_id TEXT,
    conversation_id TEXT,
    source_attempt_id TEXT,
    child_execution_id TEXT,
    error_code TEXT,
    raw_output TEXT,
    validation_error TEXT,
    version INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    completed_at TEXT,
    UNIQUE(execution_id, node_id, ordinal)
);
CREATE INDEX IF NOT EXISTS idx_workflow_attempts_execution ON workflow_node_attempts(execution_id, node_id, ordinal);

CREATE TABLE IF NOT EXISTS workflow_journal (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,
    kind TEXT NOT NULL,
    node_id TEXT,
    attempt_id TEXT,
    payload_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(execution_id, sequence)
);
CREATE INDEX IF NOT EXISTS idx_workflow_journal_order ON workflow_journal(execution_id, sequence);
