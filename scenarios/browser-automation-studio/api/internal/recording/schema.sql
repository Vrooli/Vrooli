CREATE TABLE IF NOT EXISTS recording_sessions (
    id TEXT PRIMARY KEY,
    profile_id TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    viewport_width INTEGER,
    viewport_height INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recording_sessions_profile ON recording_sessions(profile_id);
CREATE INDEX IF NOT EXISTS idx_recording_sessions_status ON recording_sessions(status);
CREATE INDEX IF NOT EXISTS idx_recording_sessions_created_at ON recording_sessions(created_at DESC);

CREATE TABLE IF NOT EXISTS recording_actions (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES recording_sessions(id) ON DELETE CASCADE,
    page_id TEXT NOT NULL,
    sequence_num INTEGER NOT NULL,
    action_type TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    duration_ms INTEGER,
    selector TEXT,
    element_meta TEXT,
    bounding_box TEXT,
    payload TEXT,
    url TEXT,
    page_title TEXT,
    confidence REAL DEFAULT 1.0,
    source TEXT DEFAULT 'auto',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(session_id, sequence_num)
);

CREATE INDEX IF NOT EXISTS idx_recording_actions_session ON recording_actions(session_id);
CREATE INDEX IF NOT EXISTS idx_recording_actions_page ON recording_actions(page_id);
CREATE INDEX IF NOT EXISTS idx_recording_actions_type ON recording_actions(action_type);
CREATE INDEX IF NOT EXISTS idx_recording_actions_timestamp ON recording_actions(timestamp);
