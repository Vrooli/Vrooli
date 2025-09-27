-- Migration 004: Add analytics fields to api_usage_logs
-- Adds tracking for requests, data volume, errors, and user identification

-- Add new columns to api_usage_logs
ALTER TABLE api_usage_logs ADD COLUMN IF NOT EXISTS user_id VARCHAR(255);
ALTER TABLE api_usage_logs ADD COLUMN IF NOT EXISTS requests INTEGER DEFAULT 0;
ALTER TABLE api_usage_logs ADD COLUMN IF NOT EXISTS data_mb FLOAT DEFAULT 0;
ALTER TABLE api_usage_logs ADD COLUMN IF NOT EXISTS errors INTEGER DEFAULT 0;
ALTER TABLE api_usage_logs ADD COLUMN IF NOT EXISTS timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Create indexes for analytics queries
CREATE INDEX IF NOT EXISTS idx_usage_logs_user_id ON api_usage_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_timestamp ON api_usage_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_usage_logs_api_timestamp ON api_usage_logs(api_id, timestamp);

-- Add version column to apis table for tracking API versions
ALTER TABLE apis ADD COLUMN IF NOT EXISTS version VARCHAR(50);

-- Create table for API version history
CREATE TABLE IF NOT EXISTS api_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    change_summary TEXT,
    breaking_changes BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_versions_api_id ON api_versions(api_id);
CREATE INDEX IF NOT EXISTS idx_api_versions_created ON api_versions(created_at);