CREATE TABLE IF NOT EXISTS conversation_search_documents (
    document_id TEXT PRIMARY KEY,
    source_run_id TEXT NOT NULL,
    source_event_id TEXT NOT NULL,
    source_message_id TEXT NOT NULL,
    chunk_index INTEGER NOT NULL CHECK (chunk_index >= 0),
    chunk_total INTEGER NOT NULL CHECK (chunk_total > 0 AND chunk_index < chunk_total),
    start_byte INTEGER NOT NULL CHECK (start_byte >= 0),
    end_byte INTEGER NOT NULL CHECK (end_byte >= start_byte),
    event_sequence INTEGER NOT NULL CHECK (event_sequence >= 0),
    role TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    content TEXT NOT NULL,
    content_class INTEGER NOT NULL,
    source_hash TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    recipe_version TEXT NOT NULL,
    harness TEXT NOT NULL DEFAULT '',
    source_session_id TEXT NOT NULL DEFAULT '',
    provider_origin TEXT NOT NULL DEFAULT '',
    importer TEXT NOT NULL DEFAULT '',
    project_scope TEXT NOT NULL DEFAULT '',
    cwd_scope TEXT NOT NULL DEFAULT '',
    runner TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    profile TEXT NOT NULL DEFAULT '',
    run_status TEXT NOT NULL DEFAULT '',
    run_label TEXT NOT NULL DEFAULT '',
    tags_json TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(tags_json) AND json_type(tags_json) = 'array'),
    workloads_json TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(workloads_json) AND json_type(workloads_json) = 'array'),
    evidence_ref TEXT NOT NULL DEFAULT '',
    visible INTEGER NOT NULL DEFAULT 1 CHECK (visible IN (0, 1)),
    indexed_at TEXT NOT NULL,
    UNIQUE(source_run_id, source_event_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_conversation_search_source
    ON conversation_search_documents(source_run_id, event_sequence, chunk_index);
CREATE INDEX IF NOT EXISTS idx_conversation_search_visibility_time
    ON conversation_search_documents(visible, occurred_at, document_id);
CREATE INDEX IF NOT EXISTS idx_conversation_search_role_time
    ON conversation_search_documents(role, occurred_at, document_id) WHERE visible = 1;
CREATE INDEX IF NOT EXISTS idx_conversation_search_harness_time
    ON conversation_search_documents(harness, occurred_at, document_id) WHERE visible = 1;
CREATE INDEX IF NOT EXISTS idx_conversation_search_project_time
    ON conversation_search_documents(project_scope, occurred_at, document_id) WHERE visible = 1;
CREATE INDEX IF NOT EXISTS idx_conversation_search_model_time
    ON conversation_search_documents(model, occurred_at, document_id) WHERE visible = 1;
CREATE INDEX IF NOT EXISTS idx_conversation_search_profile_time
    ON conversation_search_documents(profile, occurred_at, document_id) WHERE visible = 1;
CREATE INDEX IF NOT EXISTS idx_conversation_search_status_time
    ON conversation_search_documents(run_status, occurred_at, document_id) WHERE visible = 1;
CREATE INDEX IF NOT EXISTS idx_conversation_search_content_hash
    ON conversation_search_documents(content_hash, document_id) WHERE visible = 1;

-- Full rebuilds are written here first. Promotion copies one validated
-- generation into the serving table in a single transaction, so readers see
-- either the previous complete projection or the next complete projection.
CREATE TABLE IF NOT EXISTS conversation_search_generation_documents AS
SELECT '' AS generation_id, d.* FROM conversation_search_documents d WHERE 0;
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_search_generation_document
    ON conversation_search_generation_documents(generation_id, document_id);
CREATE INDEX IF NOT EXISTS idx_conversation_search_generation_source
    ON conversation_search_generation_documents(generation_id, source_run_id, source_event_id);

CREATE VIRTUAL TABLE IF NOT EXISTS conversation_search_fts USING fts5(
    document_id UNINDEXED,
    content,
    tokenize = 'unicode61 remove_diacritics 2'
);

CREATE TRIGGER IF NOT EXISTS conversation_search_documents_ai
AFTER INSERT ON conversation_search_documents
WHEN new.visible = 1
BEGIN
    INSERT INTO conversation_search_fts(rowid, document_id, content)
    VALUES (new.rowid, new.document_id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS conversation_search_documents_ad
AFTER DELETE ON conversation_search_documents
WHEN old.visible = 1
BEGIN
    DELETE FROM conversation_search_fts WHERE rowid = old.rowid;
END;

CREATE TRIGGER IF NOT EXISTS conversation_search_documents_au
AFTER UPDATE ON conversation_search_documents
BEGIN
    DELETE FROM conversation_search_fts WHERE rowid = old.rowid;
    INSERT INTO conversation_search_fts(rowid, document_id, content)
    SELECT new.rowid, new.document_id, new.content WHERE new.visible = 1;
END;

CREATE TABLE IF NOT EXISTS conversation_search_checkpoints (
    source_name TEXT PRIMARY KEY,
    source_cursor TEXT NOT NULL DEFAULT '',
    source_fingerprint TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    last_error_code TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS conversation_search_generations (
    generation_id TEXT PRIMARY KEY,
    state TEXT NOT NULL CHECK (state IN ('building', 'ready', 'active', 'retired', 'failed', 'cancelled')),
    recipe_version TEXT NOT NULL,
    source_checkpoint TEXT NOT NULL DEFAULT '',
    planned_documents INTEGER NOT NULL DEFAULT 0 CHECK (planned_documents >= 0),
    processed_documents INTEGER NOT NULL DEFAULT 0 CHECK (processed_documents >= 0),
    failed_documents INTEGER NOT NULL DEFAULT 0 CHECK (failed_documents >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_search_one_active_generation
    ON conversation_search_generations(state) WHERE state = 'active';
CREATE INDEX IF NOT EXISTS idx_conversation_search_generation_updated
    ON conversation_search_generations(updated_at, generation_id);

CREATE TABLE IF NOT EXISTS conversation_search_changes (
    sequence INTEGER PRIMARY KEY AUTOINCREMENT,
    operation TEXT NOT NULL CHECK (operation IN ('upsert_run', 'delete_event', 'delete_run', 'repair')),
    source_run_id TEXT NOT NULL DEFAULT '',
    source_event_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    processed_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_conversation_search_changes_pending
    ON conversation_search_changes(sequence) WHERE processed_at IS NULL;

CREATE TABLE IF NOT EXISTS conversation_search_deleted_sources (
    document_id TEXT PRIMARY KEY,
    source_run_id TEXT NOT NULL,
    source_event_id TEXT NOT NULL,
    deleted_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_conversation_search_deleted_source_event
    ON conversation_search_deleted_sources(source_run_id, source_event_id);

-- Privacy-safe request and outcome telemetry. Query text, snippets, regex
-- patterns, message content, and raw source paths have no columns by design.
CREATE TABLE IF NOT EXISTS conversation_search_telemetry (
    request_id TEXT PRIMARY KEY,
    session_hash TEXT NOT NULL DEFAULT '',
    mode TEXT NOT NULL,
    sort_order TEXT NOT NULL,
    filter_families_json TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(filter_families_json) AND json_type(filter_families_json) = 'array'),
    duration_ms INTEGER NOT NULL CHECK (duration_ms >= 0),
    candidate_count INTEGER NOT NULL DEFAULT 0 CHECK (candidate_count >= 0),
    result_count INTEGER NOT NULL DEFAULT 0 CHECK (result_count >= 0),
    result_stable_hit_ids_json TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(result_stable_hit_ids_json) AND json_type(result_stable_hit_ids_json) = 'array'),
    weak_only INTEGER NOT NULL DEFAULT 0 CHECK (weak_only IN (0, 1)),
    degradation_reasons_json TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(degradation_reasons_json) AND json_type(degradation_reasons_json) = 'array'),
    freshness_band TEXT NOT NULL DEFAULT 'unknown',
    error_category TEXT NOT NULL DEFAULT '',
    lexical_contributed INTEGER NOT NULL DEFAULT 0 CHECK (lexical_contributed IN (0, 1)),
    semantic_contributed INTEGER NOT NULL DEFAULT 0 CHECK (semantic_contributed IN (0, 1)),
    reformulated INTEGER NOT NULL DEFAULT 0 CHECK (reformulated IN (0, 1)),
    selected_rank INTEGER CHECK (selected_rank IS NULL OR selected_rank BETWEEN 1 AND 100),
    selected_stable_hit_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_conversation_search_telemetry_created
    ON conversation_search_telemetry(created_at, request_id);
CREATE INDEX IF NOT EXISTS idx_conversation_search_telemetry_session
    ON conversation_search_telemetry(session_hash, created_at) WHERE session_hash <> '';
