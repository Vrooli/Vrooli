-- App Monitor Database Seed Data
-- 
-- This file intentionally contains no sample/mock data.
-- All app data comes from the real-time orchestrator via 'vrooli scenario status --json'
-- App logs and metrics are collected from actual running processes.
--
-- The database tables exist to support historical data storage and
-- advanced features, but initial data comes from live system state.

-- Create additional indexes if they don't exist
CREATE INDEX IF NOT EXISTS idx_apps_status ON apps(status);
CREATE INDEX IF NOT EXISTS idx_app_status_timestamp_desc ON app_status(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_app_logs_level ON app_logs(level);
