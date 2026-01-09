-- tidiness-manager schema
-- Stores tidiness issues from light scans, AI scans, and campaigns

-- issues table: central storage for all tidiness findings
CREATE TABLE IF NOT EXISTS issues (
    id SERIAL PRIMARY KEY,
    scenario VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    category VARCHAR(50) NOT NULL,  -- length, duplication, dead_code, complexity, style, lint, type
    severity VARCHAR(20) NOT NULL,  -- critical, high, medium, low, info
    title TEXT NOT NULL,
    description TEXT,
    line_number INTEGER,
    column_number INTEGER,
    agent_notes TEXT,
    remediation_steps TEXT,
    campaign_id INTEGER,
    session_id VARCHAR(100),
    resource_used VARCHAR(100),  -- resource-claude-code, resource-codes, make-lint, make-type
    status VARCHAR(20) DEFAULT 'open',  -- open, resolved, ignored
    resolution_notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT issues_scenario_file_idx UNIQUE (scenario, file_path, category, line_number, column_number, created_at)
);

-- campaigns table: tracks auto-tidiness campaign lifecycle
CREATE TABLE IF NOT EXISTS campaigns (
    id SERIAL PRIMARY KEY,
    scenario VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'created',  -- created, active, paused, completed, error
    max_sessions INTEGER DEFAULT 10,
    max_files_per_session INTEGER DEFAULT 5,
    current_session INTEGER DEFAULT 0,
    files_visited INTEGER DEFAULT 0,
    files_total INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    error_reason TEXT,
    visited_tracker_campaign_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- config table: centralized threshold configuration
CREATE TABLE IF NOT EXISTS config (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- scan_history table: audit trail for all scans
CREATE TABLE IF NOT EXISTS scan_history (
    id SERIAL PRIMARY KEY,
    scenario VARCHAR(255) NOT NULL,
    scan_type VARCHAR(50) NOT NULL,  -- light, smart, force
    resource_used VARCHAR(100),
    issues_found INTEGER DEFAULT 0,
    duration_seconds FLOAT,
    campaign_id INTEGER REFERENCES campaigns(id),
    session_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- file_metrics table: cached file line counts and visit stats
CREATE TABLE IF NOT EXISTS file_metrics (
    id SERIAL PRIMARY KEY,
    scenario VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    line_count INTEGER NOT NULL,
    visit_count INTEGER DEFAULT 0,
    last_visited TIMESTAMP,
    last_modified TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT file_metrics_unique UNIQUE (scenario, file_path)
);

-- indexes for performance
CREATE INDEX IF NOT EXISTS idx_issues_scenario ON issues(scenario);
CREATE INDEX IF NOT EXISTS idx_issues_category ON issues(category);
CREATE INDEX IF NOT EXISTS idx_issues_severity ON issues(severity);
CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status);

-- unique index for deduplication of lint/type issues (without created_at)
-- This enables ON CONFLICT upserts for repeated scans
CREATE UNIQUE INDEX IF NOT EXISTS idx_issues_dedup
    ON issues(scenario, file_path, category, COALESCE(line_number, 0), COALESCE(column_number, 0))
    WHERE status = 'open';
CREATE INDEX IF NOT EXISTS idx_campaigns_scenario ON campaigns(scenario);
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);
CREATE INDEX IF NOT EXISTS idx_file_metrics_scenario ON file_metrics(scenario);
CREATE INDEX IF NOT EXISTS idx_scan_history_scenario ON scan_history(scenario);

-- default configuration values
INSERT INTO config (key, value, description) VALUES
    ('long_file_threshold', '500', 'Line count threshold for flagging long files')
    ON CONFLICT (key) DO NOTHING;

INSERT INTO config (key, value, description) VALUES
    ('max_concurrent_campaigns', '3', 'Maximum number of concurrent auto-campaigns')
    ON CONFLICT (key) DO NOTHING;

INSERT INTO config (key, value, description) VALUES
    ('max_visit_count', '5', 'Maximum visits per file before exclusion (unless forced)')
    ON CONFLICT (key) DO NOTHING;

INSERT INTO config (key, value, description) VALUES
    ('max_files_per_batch', '10', 'Maximum files per AI batch')
    ON CONFLICT (key) DO NOTHING;
