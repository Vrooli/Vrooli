-- The catalog replaced conversation_search_documents on 2026-09-14. Its
-- explicit INTEGER PRIMARY KEY is the FTS content rowid: VACUUM may renumber
-- the implicit rowid of a table without one, which would silently point the
-- lexical index at the wrong documents. The owner loop of the indexer copies
-- and then drops the legacy objects (projection_upgrade.go).
CREATE TABLE IF NOT EXISTS conversation_search_catalog (
    id INTEGER PRIMARY KEY,
    document_id TEXT NOT NULL UNIQUE,
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

-- Relevance search is driven by FTS and joins the catalog by id, so the
-- catalog needs only the source lookup (context windows, per-run replacement)
-- and one time index for browsing without a lexical query. The per-filter time
-- indexes this replaced cost about 125 MB each; live telemetry on 2026-09-14
-- showed about 6,500 of 6,600 searches were relevance ranked.
CREATE INDEX IF NOT EXISTS idx_conversation_search_catalog_source
    ON conversation_search_catalog(source_run_id, event_sequence, chunk_index);
CREATE INDEX IF NOT EXISTS idx_conversation_search_catalog_visible_time
    ON conversation_search_catalog(occurred_at, document_id) WHERE visible = 1;

-- Generations are staged here first and published into the catalog in bounded
-- batches. Staged rows are deleted once their generation settles (active,
-- retired, failed or cancelled); only building and ready generations keep them.
-- The column list is explicit so the staging copy never carries catalog ids.
CREATE TABLE IF NOT EXISTS conversation_search_generation_documents AS
SELECT '' AS generation_id, document_id, source_run_id, source_event_id, source_message_id,
    chunk_index, chunk_total, start_byte, end_byte, event_sequence, role, occurred_at, content,
    content_class, source_hash, content_hash, recipe_version, harness, source_session_id,
    provider_origin, importer, project_scope, cwd_scope, runner, model, profile, run_status,
    run_label, tags_json, workloads_json, evidence_ref, visible, indexed_at
FROM conversation_search_catalog d WHERE 0;
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_search_generation_document
    ON conversation_search_generation_documents(generation_id, document_id);
CREATE INDEX IF NOT EXISTS idx_conversation_search_generation_source
    ON conversation_search_generation_documents(generation_id, source_run_id, source_event_id);

-- External content: the index stores tokens only and reads text from the
-- catalog, which removed a second full copy of every transcript chunk (0.88 GB
-- on 2026-09-14). The index mirrors every catalog row and queries filter
-- visibility, so the FTS5 integrity-check and rebuild commands stay valid. An
-- external-content delete must restate the old values exactly.
CREATE VIRTUAL TABLE IF NOT EXISTS conversation_search_catalog_fts USING fts5(
    document_id UNINDEXED,
    content,
    content = 'conversation_search_catalog',
    content_rowid = 'id',
    tokenize = 'unicode61 remove_diacritics 2'
);

CREATE TRIGGER IF NOT EXISTS conversation_search_catalog_ai
AFTER INSERT ON conversation_search_catalog
BEGIN
    INSERT INTO conversation_search_catalog_fts(rowid, document_id, content)
    VALUES (new.id, new.document_id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS conversation_search_catalog_ad
AFTER DELETE ON conversation_search_catalog
BEGIN
    INSERT INTO conversation_search_catalog_fts(conversation_search_catalog_fts, rowid, document_id, content)
    VALUES ('delete', old.id, old.document_id, old.content);
END;

-- Only a change to indexed text or identity touches the index; visibility and
-- metadata updates leave it alone.
CREATE TRIGGER IF NOT EXISTS conversation_search_catalog_au
AFTER UPDATE OF id, document_id, content ON conversation_search_catalog
BEGIN
    INSERT INTO conversation_search_catalog_fts(conversation_search_catalog_fts, rowid, document_id, content)
    VALUES ('delete', old.id, old.document_id, old.content);
    INSERT INTO conversation_search_catalog_fts(rowid, document_id, content)
    VALUES (new.id, new.document_id, new.content);
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

-- Qdrant is an external derived store, but its generation ownership remains
-- durable in the conversation-search domain. Unknown physical collections
-- are intentionally absent here and therefore quarantined by the owner API.
CREATE TABLE IF NOT EXISTS conversation_search_qdrant_generations (
    generation_id TEXT PRIMARY KEY,
    owner TEXT NOT NULL,
    namespace TEXT NOT NULL,
    alias_name TEXT NOT NULL,
    collection_name TEXT NOT NULL UNIQUE,
    content_identity TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL CHECK (state IN ('building', 'candidate', 'active', 'retired', 'protected', 'expired', 'failed', 'quarantined', 'deleted')),
    lease_id TEXT NOT NULL DEFAULT '',
    lease_holder TEXT NOT NULL DEFAULT '',
    lease_expires_at TEXT,
    points INTEGER NOT NULL DEFAULT 0 CHECK (points >= 0),
    bytes INTEGER NOT NULL DEFAULT 0 CHECK (bytes >= 0),
    cleanup_outcome TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_conversation_search_qdrant_lifecycle
    ON conversation_search_qdrant_generations(namespace, alias_name, state, created_at);

CREATE TABLE IF NOT EXISTS conversation_search_qdrant_cleanup_receipts (
    idempotency_key TEXT PRIMARY KEY,
    receipt_json TEXT NOT NULL CHECK (json_valid(receipt_json)),
    created_at TEXT NOT NULL
);

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

-- External owners retain canonical records locally but may revoke derived
-- search visibility. This identity-keyed tombstone survives index repair.
CREATE TABLE IF NOT EXISTS conversation_search_external_tombstones (
    source_harness TEXT NOT NULL,
    source_session_id TEXT NOT NULL,
    tombstoned_at TEXT NOT NULL,
    PRIMARY KEY(source_harness, source_session_id)
);

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
