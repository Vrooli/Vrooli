-- Visual validation records for deployment quality gates
CREATE TABLE IF NOT EXISTS visual_validations (
    id VARCHAR(255) PRIMARY KEY,
    profile_id VARCHAR(255) NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    deployment_id VARCHAR(255),
    smoke_test_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    video_path TEXT,
    video_size_bytes BIGINT,
    video_duration_ms BIGINT,
    platform VARCHAR(50),
    review_decision VARCHAR(50),
    reviewed_by VARCHAR(255),
    review_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_visual_validations_profile ON visual_validations(profile_id);
CREATE INDEX IF NOT EXISTS idx_visual_validations_status ON visual_validations(status);
