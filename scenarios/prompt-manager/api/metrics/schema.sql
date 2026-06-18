CREATE TABLE IF NOT EXISTS skill_metrics (
    id TEXT PRIMARY KEY,
    skill_id TEXT NOT NULL UNIQUE,
    usage_count INTEGER NOT NULL DEFAULT 0,
    last_used TEXT,
    effectiveness_rating INTEGER CHECK (effectiveness_rating IS NULL OR (effectiveness_rating >= 1 AND effectiveness_rating <= 5)),
    notes TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_skill_metrics_skill_id ON skill_metrics(skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_metrics_usage ON skill_metrics(usage_count);
CREATE INDEX IF NOT EXISTS idx_skill_metrics_last_used ON skill_metrics(last_used);

