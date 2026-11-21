-- quality domain schema (SQLite)
-- Collection quality metrics and per-collection statistics.
--
-- This domain owns these objects. Adding or removing the domain is a change
-- to this folder only; no central schema file declares them.
--
-- SQLite has no schema namespace, so tables are unqualified. Isolation comes
-- from the per-scenario database file resolved through api-core/storage.

CREATE TABLE IF NOT EXISTS quality_metrics (
    id TEXT PRIMARY KEY,
    collection_name TEXT NOT NULL,
    coherence_score REAL CHECK (coherence_score IS NULL OR (coherence_score >= 0 AND coherence_score <= 1)),
    freshness_score REAL CHECK (freshness_score IS NULL OR (freshness_score >= 0 AND freshness_score <= 1)),
    redundancy_score REAL CHECK (redundancy_score IS NULL OR (redundancy_score >= 0 AND redundancy_score <= 1)),
    coverage_score REAL CHECK (coverage_score IS NULL OR (coverage_score >= 0 AND coverage_score <= 1)),
    total_entries INTEGER DEFAULT 0,
    avg_quality REAL GENERATED ALWAYS AS (
        (coherence_score + freshness_score + redundancy_score + coverage_score) / 4
    ) STORED,
    measured_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS collection_stats (
    id TEXT PRIMARY KEY,
    collection_name TEXT UNIQUE NOT NULL,
    total_entries INTEGER DEFAULT 0,
    total_searches INTEGER DEFAULT 0,
    avg_search_score REAL,
    most_searched_terms TEXT,
    growth_rate REAL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quality_metrics_collection ON quality_metrics(collection_name);
CREATE INDEX IF NOT EXISTS idx_quality_metrics_measured_at ON quality_metrics(measured_at DESC);

-- Retention support. The pruner deletes by measured_at and the downsampler
-- groups by (collection_name, day); this composite covers both.
CREATE INDEX IF NOT EXISTS idx_quality_metrics_collection_measured
    ON quality_metrics(collection_name, measured_at DESC);

CREATE TRIGGER IF NOT EXISTS trg_quality_metrics_updated_at
AFTER UPDATE ON quality_metrics
FOR EACH ROW BEGIN
    UPDATE quality_metrics SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Latest sample per collection. Postgres expressed this with LEFT JOIN LATERAL,
-- which SQLite has no syntax for; the correlated subquery below is equivalent.
-- Dropped before creation so a definition change actually lands rather than
-- being silently skipped by IF NOT EXISTS.
DROP VIEW IF EXISTS dashboard_metrics;
CREATE VIEW dashboard_metrics AS
SELECT
    c.collection_name,
    c.total_entries,
    q.coherence_score,
    q.freshness_score,
    q.redundancy_score,
    q.coverage_score,
    q.avg_quality,
    c.total_searches,
    c.avg_search_score,
    q.measured_at
FROM collection_stats c
LEFT JOIN quality_metrics q ON q.id = (
    SELECT id FROM quality_metrics
    WHERE collection_name = c.collection_name
    ORDER BY measured_at DESC
    LIMIT 1
);
