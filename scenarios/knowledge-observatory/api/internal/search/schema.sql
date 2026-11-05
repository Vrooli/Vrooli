-- search domain schema (SQLite)
-- Search history for analytics.
--
-- This domain owns these objects. Adding or removing the domain is a change
-- to this folder only; no central schema file declares them.

CREATE TABLE IF NOT EXISTS search_history (
    id TEXT PRIMARY KEY,
    query TEXT NOT NULL,
    collection TEXT,
    result_count INTEGER DEFAULT 0,
    avg_score REAL,
    response_time_ms INTEGER,
    user_session TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Postgres indexed this column with GIN over to_tsvector('english', query).
-- No code queries search_history by text — rows are written for analytics and
-- read by created_at — so the plain index below covers the real access pattern.
-- If text search over history is ever added, reintroduce it here as an FTS5
-- virtual table rather than assuming the GIN index was load-bearing.
CREATE INDEX IF NOT EXISTS idx_search_history_query ON search_history(query);
CREATE INDEX IF NOT EXISTS idx_search_history_created ON search_history(created_at DESC);
