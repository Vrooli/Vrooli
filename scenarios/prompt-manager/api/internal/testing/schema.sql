CREATE TABLE IF NOT EXISTS test_results (
    id TEXT PRIMARY KEY,
    skill_id TEXT NOT NULL,
    model TEXT NOT NULL DEFAULT 'ollama/chat.small',
    input_variables TEXT,
    response TEXT,
    response_time REAL,
    token_count INTEGER,
    rating INTEGER CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5)),
    notes TEXT,
    tested_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_test_results_skill ON test_results(skill_id);
CREATE INDEX IF NOT EXISTS idx_test_results_tested_at ON test_results(tested_at);
