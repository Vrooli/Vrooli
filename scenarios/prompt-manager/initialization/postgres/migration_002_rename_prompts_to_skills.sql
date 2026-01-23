-- Migration: Rename prompts to skills, avatars to members
-- Date: 2026-01-23
-- Description: Renames prompt_metrics table to skill_metrics and updates column names

-- Rename table prompt_metrics to skill_metrics
ALTER TABLE IF EXISTS prompt_metrics RENAME TO skill_metrics;

-- Rename column prompt_id to skill_id in skill_metrics
ALTER TABLE IF EXISTS skill_metrics RENAME COLUMN prompt_id TO skill_id;

-- Rename column prompt_id to skill_id in test_results
ALTER TABLE IF EXISTS test_results RENAME COLUMN prompt_id TO skill_id;

-- Drop old indexes and create new ones with updated names
DROP INDEX IF EXISTS idx_prompt_metrics_prompt_id;
DROP INDEX IF EXISTS idx_prompt_metrics_usage;
DROP INDEX IF EXISTS idx_prompt_metrics_last_used;
DROP INDEX IF EXISTS idx_test_results_prompt;

CREATE INDEX IF NOT EXISTS idx_skill_metrics_skill_id ON skill_metrics(skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_metrics_usage ON skill_metrics(usage_count DESC);
CREATE INDEX IF NOT EXISTS idx_skill_metrics_last_used ON skill_metrics(last_used DESC);
CREATE INDEX IF NOT EXISTS idx_test_results_skill ON test_results(skill_id);

-- Drop old trigger and function, recreate with new names
DROP TRIGGER IF EXISTS trigger_update_prompt_metrics_timestamp ON skill_metrics;
DROP FUNCTION IF EXISTS update_prompt_metrics_timestamp();

-- Create new function with updated name
CREATE OR REPLACE FUNCTION update_skill_metrics_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create new trigger
CREATE TRIGGER trigger_update_skill_metrics_timestamp
    BEFORE UPDATE ON skill_metrics
    FOR EACH ROW EXECUTE FUNCTION update_skill_metrics_timestamp();
